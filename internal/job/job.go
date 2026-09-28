// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package job is one scorecard run: list the registry, check what it lists,
// publish what may be published, and queue what must go to its owner first.
package job

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"sync"
	"time"

	"satellion.com/passmcp-reporting/attestation"

	"satellion.com/passmcp-registry/internal/checker"
	"satellion.com/passmcp-registry/internal/disclosure"
	"satellion.com/passmcp-registry/internal/limiter"
	"satellion.com/passmcp-registry/internal/policy"
	"satellion.com/passmcp-registry/internal/registry"
	"satellion.com/passmcp-registry/internal/site"
)

// MethodVersion names the scorecard's method: the phases checked, the
// withholding rules and the record format. It changes when any of them does.
const MethodVersion = "1"

// Lister lists a registry.
type Lister interface {
	List(ctx context.Context) (registry.Listing, error)
}

// Config is everything one run needs.
type Config struct {
	Registry       Lister
	Checker        checker.Checker
	PassmcpVersion string
	OptOut         *policy.OptOut
	Site           *site.Site
	Queue          *disclosure.Queue
	Limiter        *limiter.PerHost
	Workers        int
	Now            func() time.Time
	Log            io.Writer
}

// Summary is what one run did.
type Summary struct {
	Listed, Checked, Published, Withheld, Unreachable, OptedOut, Skipped int
	Pruned                                                               []string
}

type outcome struct {
	target registry.Remote
	at     time.Time
	res    checker.Result
	err    error
}

// Run performs one scorecard run.
func Run(ctx context.Context, cfg Config) (Summary, error) {
	var sum Summary
	listing, err := cfg.Registry.List(ctx)
	if err != nil {
		return sum, err
	}
	sum.Listed = len(listing.Remotes)
	sum.Skipped = len(listing.Skipped)
	targets := selectTargets(listing.Remotes, cfg.OptOut, &sum)

	outcomes := check(ctx, cfg, targets)
	var unreachable []site.Unreachable
	for _, o := range outcomes {
		u, err := record(cfg, o, &sum)
		if err != nil {
			return sum, err
		}
		if u != nil {
			unreachable = append(unreachable, *u)
		}
	}
	pruned, err := cfg.Site.Prune(cfg.OptOut.Excludes)
	if err != nil {
		return sum, err
	}
	sum.Pruned = pruned
	ix := site.Index{MethodVersion: MethodVersion, PassmcpVersion: cfg.PassmcpVersion, GeneratedAt: cfg.Now().UTC(),
		Unreachable: unreachable, Skipped: skipped(listing.Skipped, cfg.OptOut)}
	return sum, cfg.Site.WriteIndex(ix)
}

// selectTargets drops opted-out servers and keeps one remote per server
// version, preferring Streamable HTTP, so a record is about one endpoint.
func selectTargets(remotes []registry.Remote, opt *policy.OptOut, sum *Summary) []registry.Remote {
	chosen := map[string]registry.Remote{}
	var order []string
	for _, r := range remotes {
		if opt.Excludes(r.Name) {
			sum.OptedOut++
			continue
		}
		key := r.Name + "\x00" + r.Version
		prev, seen := chosen[key]
		if !seen {
			order = append(order, key)
		}
		if !seen || (prev.Type != "streamable-http" && r.Type == "streamable-http") {
			chosen[key] = r
		}
	}
	out := make([]registry.Remote, 0, len(order))
	for _, k := range order {
		out = append(out, chosen[k])
	}
	return out
}

// check runs the checks on a small worker pool, spaced per host.
func check(ctx context.Context, cfg Config, targets []registry.Remote) []outcome {
	workers := cfg.Workers
	if workers < 1 {
		workers = 1
	}
	out := make([]outcome, len(targets))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				out[i] = checkOne(ctx, cfg, targets[i])
			}
		}()
	}
	for i := range targets {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}

func checkOne(ctx context.Context, cfg Config, t registry.Remote) outcome {
	o := outcome{target: t}
	u, err := url.Parse(t.URL)
	if err != nil {
		o.err, o.at = err, cfg.Now()
		return o
	}
	if err := cfg.Limiter.Wait(ctx, u.Hostname()); err != nil {
		o.err, o.at = err, cfg.Now()
		return o
	}
	o.at = cfg.Now()
	o.res, o.err = cfg.Checker.Check(ctx, t.URL)
	return o
}

// record files one outcome: published, withheld, or unreachable.
func record(cfg Config, o outcome, sum *Summary) (*site.Unreachable, error) {
	t := o.target
	rec, reasons, problem := assess(cfg, o, sum)
	switch {
	case problem != "":
		sum.Unreachable++
		_, _ = fmt.Fprintf(cfg.Log, "unreachable: %s %s: %s\n", t.Name, t.URL, problem)
		return &site.Unreachable{Name: t.Name, Version: t.Version, URL: t.URL, CheckedAt: o.at.UTC(), Error: problem}, nil
	case len(reasons) > 0:
		sum.Withheld++
		if _, err := cfg.Queue.Add(t.Name, t.Version, t.URL, MethodVersion, reasons, o.at.UTC()); err != nil {
			return nil, err
		}
		_, _ = fmt.Fprintf(cfg.Log, "withheld: %s: queued for its owner\n", t.Name)
		return nil, cfg.Site.Remove(t.Name, t.Version)
	default:
		sum.Published++
		return nil, cfg.Site.Publish(rec, o.res.Statement)
	}
}

// assess decides what an outcome is. A non-empty problem means the server
// could not be assessed: nothing about it may be published or withheld.
func assess(cfg Config, o outcome, sum *Summary) (rec site.Record, reasons []policy.Reason, problem string) {
	if o.err != nil {
		return rec, nil, o.err.Error()
	}
	sum.Checked++
	stmt, err := attestation.Parse(o.res.Statement)
	if err != nil {
		return rec, nil, "the statement passmcp produced does not verify: " + err.Error()
	}
	if !stmt.Covers("http", o.target.URL) {
		return rec, nil, "the statement is not about the listed endpoint"
	}
	if reasons, err = policy.Withhold(o.res.Report); err != nil {
		return rec, nil, err.Error()
	}
	if len(reasons) > 0 {
		return rec, reasons, ""
	}
	if rec, err = newRecord(cfg, o); err != nil {
		return rec, nil, err.Error()
	}
	return rec, nil, ""
}

func newRecord(cfg Config, o outcome) (site.Record, error) {
	var r struct {
		Score struct {
			Total float64 `json:"total"`
			Grade string  `json:"grade"`
		} `json:"score"`
		Counts json.RawMessage `json:"counts"`
	}
	if err := json.Unmarshal(o.res.Report, &r); err != nil {
		return site.Record{}, fmt.Errorf("not a passmcp report: %w", err)
	}
	t := o.target
	return site.Record{Name: t.Name, Version: t.Version, URL: t.URL, Transport: t.Type, CheckedAt: o.at.UTC(),
		MethodVersion: MethodVersion, PassmcpVersion: cfg.PassmcpVersion, Score: r.Score.Total, Grade: r.Score.Grade,
		Counts: r.Counts}, nil
}

func skipped(in []registry.Skipped, opt *policy.OptOut) []site.Skipped {
	out := []site.Skipped{}
	for _, s := range in {
		if opt.Excludes(s.Name) {
			continue
		}
		out = append(out, site.Skipped{Name: s.Name, URL: s.URL, Reason: s.Reason})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name+out[i].URL < out[j].Name+out[j].URL })
	return out
}
