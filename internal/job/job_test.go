// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package job

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"satellion.com/passmcp-reporting/attestation"

	"satellion.com/passmcp-registry/internal/checker"
	"satellion.com/passmcp-registry/internal/disclosure"
	"satellion.com/passmcp-registry/internal/limiter"
	"satellion.com/passmcp-registry/internal/policy"
	"satellion.com/passmcp-registry/internal/registry"
	"satellion.com/passmcp-registry/internal/site"
)

// statement builds a statement passmcp-reporting accepts, about endpoint.
func statement(t *testing.T, endpoint string) []byte {
	t.Helper()
	target := attestation.Target{Transport: "http", Endpoint: endpoint}
	s := &attestation.Statement{
		Type:          attestation.StatementType,
		Subject:       []attestation.Subject{attestation.SubjectFor(target)},
		PredicateType: attestation.PredicateType,
		Predicate: attestation.Evaluation{
			SubjectKind:   attestation.SubjectKindDescriptor,
			Target:        target,
			JudgedAgainst: attestation.Basis{Rubric: "1", CheckInventory: "120"},
			Instrument:    attestation.Instrument{Name: "passmcp", Version: "0.0.8", SchemaVersion: 1},
			RanAt:         time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC),
			Took:          "1s",
			Verdicts:      []attestation.Verdict{{ID: "net.tcp", Phase: "net", Status: "pass"}},
			Counts:        attestation.Counts{Pass: 1},
			Score:         &attestation.Score{Total: 88, Grade: "B", Assessed: 1, Of: 1},
		},
	}
	b, err := s.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := attestation.Parse(b); err != nil {
		t.Fatalf("fixture statement does not verify: %v", err)
	}
	return b
}

const cleanReport = `{"auth":{"mode":"none"},"score":{"total":88,"grade":"B"},"counts":{"pass":1},
 "catalog":{"tools":[{"name":"search","read_only":true}]},"phases":[{"findings":[{"id":"net.tcp","status":"pass"}]}]}`

const openReport = `{"auth":{"mode":"none"},"score":{"total":40,"grade":"F"},
 "catalog":{"tools":[{"name":"delete_everything","read_only":false}]},"phases":[]}`

type fakeLister struct {
	l   registry.Listing
	err error
}

func (f fakeLister) List(context.Context) (registry.Listing, error) { return f.l, f.err }

type fakeChecker struct {
	t       *testing.T
	mu      sync.Mutex
	called  []string
	reports map[string]string
	fail    map[string]error
	wrong   map[string]string // endpoint -> the endpoint its statement claims
}

func (f *fakeChecker) Check(_ context.Context, endpoint string) (checker.Result, error) {
	f.mu.Lock()
	f.called = append(f.called, endpoint)
	f.mu.Unlock()
	if err := f.fail[endpoint]; err != nil {
		return checker.Result{}, err
	}
	about := endpoint
	if w, ok := f.wrong[endpoint]; ok {
		about = w
	}
	rep := f.reports[endpoint]
	if rep == "" {
		rep = cleanReport
	}
	return checker.Result{Report: []byte(rep), Statement: statement(f.t, about), Exit: 0}, nil
}

type env struct {
	cfg  Config
	chk  *fakeChecker
	site *site.Site
	priv string
	log  *bytes.Buffer
}

func newEnv(t *testing.T, remotes []registry.Remote, optOut string) *env {
	t.Helper()
	dir := t.TempDir()
	optPath := filepath.Join(dir, "opt-out.txt")
	if err := os.WriteFile(optPath, []byte(optOut), 0o600); err != nil {
		t.Fatal(err)
	}
	opt, err := policy.LoadOptOut(optPath)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{chk: &fakeChecker{t: t, reports: map[string]string{}, fail: map[string]error{}, wrong: map[string]string{}},
		site: &site.Site{Dir: filepath.Join(dir, "site")}, priv: filepath.Join(dir, "private"), log: &bytes.Buffer{}}
	e.cfg = Config{
		Registry: fakeLister{l: registry.Listing{Remotes: remotes,
			Skipped: []registry.Skipped{{Name: "io.github.t/templ", URL: "https://{x}.t/mcp", Reason: "the URL is a template the user fills in"}}}},
		Checker: e.chk, PassmcpVersion: "0.0.8", OptOut: opt, Site: e.site,
		Queue: &disclosure.Queue{Dir: e.priv}, Limiter: limiter.New(0), Workers: 3,
		Now: func() time.Time { return time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC) }, Log: e.log,
	}
	return e
}

func remote(name, url string) registry.Remote {
	return registry.Remote{Name: name, Version: "1.0.0", Type: "streamable-http", URL: url}
}

// AC: REG-01, REG-05
func TestTheRunContactsOnlyTheListedServers(t *testing.T) {
	e := newEnv(t, []registry.Remote{remote("io.github.a/one", "https://one.example/mcp"), remote("io.github.b/two", "https://two.example/mcp")}, "")
	sum, err := Run(context.Background(), e.cfg)
	if err != nil {
		t.Fatal(err)
	}
	got := append([]string(nil), e.chk.called...)
	sort.Strings(got)
	if strings.Join(got, " ") != "https://one.example/mcp https://two.example/mcp" {
		t.Fatalf("checked %v; want exactly the listed endpoints", got)
	}
	if sum.Listed != 2 || sum.Checked != 2 || sum.Published != 2 || sum.Skipped != 1 {
		t.Fatalf("summary %+v", sum)
	}
}

// AC: REG-02
func TestAPublishedRecordCarriesAStatementTheVerifierAccepts(t *testing.T) {
	e := newEnv(t, []registry.Remote{remote("io.github.a/one", "https://one.example/mcp")}, "")
	if _, err := Run(context.Background(), e.cfg); err != nil {
		t.Fatal(err)
	}
	recs, err := e.site.Records()
	if err != nil || len(recs) != 1 {
		t.Fatalf("records %v %v", recs, err)
	}
	r := recs[0]
	if r.MethodVersion != MethodVersion || r.PassmcpVersion != "0.0.8" || r.CheckedAt.IsZero() || r.Score != 88 || r.Grade != "B" {
		t.Fatalf("record %+v", r)
	}
	b, err := os.ReadFile(filepath.Join(e.site.Dir, r.Attestation))
	if err != nil {
		t.Fatal(err)
	}
	st, err := attestation.Parse(b)
	if err != nil || !st.Covers("http", "https://one.example/mcp") {
		t.Fatalf("published statement: %v", err)
	}
	sign, _ := os.ReadFile(filepath.Join(e.site.Dir, site.ToSignFile))
	if strings.TrimSpace(string(sign)) != r.Attestation {
		t.Fatalf("the signing list is %q", sign)
	}
}

// AC: REG-03
func TestAnOptedOutServerIsNeitherCheckedNorPublished(t *testing.T) {
	e := newEnv(t, []registry.Remote{remote("io.github.a/one", "https://one.example/mcp"), remote("io.github.b/two", "https://two.example/mcp")}, "")
	if _, err := Run(context.Background(), e.cfg); err != nil {
		t.Fatal(err)
	}
	// The owner of two opts out; the next run removes its record and never
	// contacts it.
	optPath := filepath.Join(t.TempDir(), "opt-out.txt")
	_ = os.WriteFile(optPath, []byte("io.github.b/two\n"), 0o600)
	opt, _ := policy.LoadOptOut(optPath)
	e.cfg.OptOut = opt
	e.chk.called = nil
	sum, err := Run(context.Background(), e.cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range e.chk.called {
		if strings.Contains(c, "two.example") {
			t.Fatal("an opted-out server was contacted")
		}
	}
	if sum.OptedOut != 1 || strings.Join(sum.Pruned, ",") != "io.github.b/two" {
		t.Fatalf("summary %+v", sum)
	}
	index, _ := os.ReadFile(filepath.Join(e.site.Dir, "index.json"))
	if strings.Contains(string(index), "io.github.b/two") {
		t.Fatal("the opted-out server is still in the index")
	}
}

// AC: REG-04
func TestAVulnerableResultIsWithheldAndQueuedNeverPublished(t *testing.T) {
	e := newEnv(t, []registry.Remote{remote("io.github.a/open", "https://open.example/mcp")}, "")
	// An earlier, clean record of the same version exists.
	if _, err := Run(context.Background(), e.cfg); err != nil {
		t.Fatal(err)
	}
	e.chk.reports["https://open.example/mcp"] = openReport
	sum, err := Run(context.Background(), e.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Withheld != 1 || sum.Published != 0 {
		t.Fatalf("summary %+v", sum)
	}
	index, _ := os.ReadFile(filepath.Join(e.site.Dir, "index.json"))
	if strings.Contains(string(index), "io.github.a/open") || strings.Contains(string(index), "open.example") {
		t.Fatalf("a withheld server appears in the public index:\n%s", index)
	}
	if _, err := os.Stat(filepath.Join(e.site.Dir, site.RecordDir("io.github.a/open", "1.0.0"))); !os.IsNotExist(err) {
		t.Fatal("the earlier public record was not removed")
	}
	entry, err := os.ReadFile(filepath.Join(e.priv, "io.github.a~open@1.0.0.json"))
	if err != nil || !strings.Contains(string(entry), policy.RuleOpenMutatingTools) {
		t.Fatalf("queue entry %s, err %v", entry, err)
	}
	if strings.HasPrefix(e.priv, e.site.Dir) {
		t.Fatal("the private queue must not live under the site")
	}
}

func TestUnreachableAndUnverifiableResultsAreNeverPassing(t *testing.T) {
	e := newEnv(t, []registry.Remote{
		remote("io.github.a/down", "https://down.example/mcp"),
		remote("io.github.a/liar", "https://liar.example/mcp"),
		remote("io.github.a/garbled", "https://garbled.example/mcp"),
		{Name: "io.github.a/badurl", Version: "1", Type: "sse", URL: "%zz"},
	}, "")
	e.chk.fail["https://down.example/mcp"] = errors.New("passmcp exited 1: connection refused")
	e.chk.wrong["https://liar.example/mcp"] = "https://elsewhere.example/mcp"
	e.chk.reports["https://garbled.example/mcp"] = "{"
	sum, err := Run(context.Background(), e.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Published != 0 || sum.Unreachable != 4 {
		t.Fatalf("summary %+v", sum)
	}
	index, _ := os.ReadFile(filepath.Join(e.site.Dir, "index.json"))
	for _, want := range []string{"connection refused", "not about the listed endpoint", "not a passmcp report"} {
		if !strings.Contains(string(index), want) {
			t.Errorf("index lacks %q", want)
		}
	}
}

func TestOneRemotePerServerVersionPreferringStreamableHTTP(t *testing.T) {
	e := newEnv(t, []registry.Remote{
		{Name: "io.github.a/one", Version: "1", Type: "sse", URL: "https://one.example/sse"},
		{Name: "io.github.a/one", Version: "1", Type: "streamable-http", URL: "https://one.example/mcp"},
	}, "")
	if _, err := Run(context.Background(), e.cfg); err != nil {
		t.Fatal(err)
	}
	if strings.Join(e.chk.called, " ") != "https://one.example/mcp" {
		t.Fatalf("checked %v", e.chk.called)
	}
}

func TestARegistryFailureStopsTheRun(t *testing.T) {
	e := newEnv(t, nil, "")
	e.cfg.Registry = fakeLister{err: errors.New("registry: down")}
	if _, err := Run(context.Background(), e.cfg); err == nil {
		t.Fatal("want the registry's error")
	}
}

func TestACancelledRunRecordsNothingAsPassing(t *testing.T) {
	e := newEnv(t, []registry.Remote{remote("io.github.a/one", "https://one.example/mcp")}, "")
	e.cfg.Limiter = limiter.New(time.Hour)
	_ = e.cfg.Limiter.Wait(context.Background(), "one.example")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sum, err := Run(ctx, e.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Published != 0 || sum.Unreachable != 1 {
		t.Fatalf("summary %+v", sum)
	}
}
