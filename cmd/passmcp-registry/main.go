// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Command passmcp-registry builds the signed public scorecard of the remote
// servers listed in the MCP Registry.
//
//	passmcp-registry run     check the listed servers and write the site
//	passmcp-registry verify  check every published record offline
//	passmcp-registry report  build the yearly report from the records
//	passmcp-registry version print the version
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"satellion.com/passmcp-reporting/attestation"

	"satellion.com/passmcp-registry/internal/annual"
	"satellion.com/passmcp-registry/internal/checker"
	"satellion.com/passmcp-registry/internal/disclosure"
	"satellion.com/passmcp-registry/internal/job"
	"satellion.com/passmcp-registry/internal/limiter"
	"satellion.com/passmcp-registry/internal/policy"
	"satellion.com/passmcp-registry/internal/registry"
	"satellion.com/passmcp-registry/internal/site"
)

// Version is set at build time with -ldflags.
var Version = "dev"

// DefaultRegistry is the official MCP Registry.
const DefaultRegistry = "https://registry.modelcontextprotocol.io"

const usage = `usage: passmcp-registry <command> [flags]

commands:
  run      check the listed servers and write the site
  verify   check every published record offline
  report   build the yearly report from the records
  version  print the version
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "run":
		err = cmdRun(ctx, args[1:], stdout, stderr)
	case "verify":
		err = cmdVerify(args[1:], stdout, stderr)
	case "report":
		err = cmdReport(args[1:], stdout, stderr)
	case "version", "--version":
		_, _ = fmt.Fprintln(stdout, "passmcp-registry", Version)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "passmcp-registry: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	var uerr usageError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &uerr):
		_, _ = fmt.Fprintln(stderr, "passmcp-registry:", err)
		return 2
	default:
		_, _ = fmt.Fprintln(stderr, "passmcp-registry:", err)
		return 1
	}
}

type usageError struct{ error }

func parse(fs *flag.FlagSet, args []string, stderr io.Writer) error {
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return usageError{err}
	}
	if fs.NArg() > 0 {
		return usageError{fmt.Errorf("unexpected argument %q", fs.Arg(0))}
	}
	return nil
}

func userAgent() string {
	return "passmcp-registry/" + Version + " (+https://github.com/sebastienrousseau/passmcp-registry)"
}

func cmdRun(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	base := fs.String("registry", DefaultRegistry, "the registry's base URL")
	bin := fs.String("passmcp", "passmcp", "path of the pinned passmcp binary")
	siteDir := fs.String("site", "site", "the published site directory")
	priv := fs.String("private", "private", "the private disclosure queue directory; never published")
	optOut := fs.String("opt-out", "opt-out.txt", "the opt-out list")
	workers := fs.Int("workers", 4, "checks in flight at once")
	perHost := fs.Duration("per-host", 10*time.Second, "minimum time between two checks starting on one host")
	rps := fs.Float64("rps", 1, "passmcp's request rate within one check")
	timeout := fs.Duration("timeout", 20*time.Second, "passmcp's per-call timeout")
	if err := parse(fs, args, stderr); err != nil {
		return err
	}
	if abs, _ := filepath.Abs(*priv); isUnder(abs, *siteDir) {
		return usageError{errors.New("the private queue must not be inside the site directory")}
	}
	opt, err := policy.LoadOptOut(*optOut)
	if err != nil {
		return err
	}
	sc := &checker.Passmcp{Bin: *bin, UserAgent: userAgent(), RPS: *rps, Timeout: *timeout}
	ver, err := sc.Version(ctx)
	if err != nil {
		return err
	}
	cfg := job.Config{
		Registry: &registry.Client{Base: *base, HTTP: &http.Client{Timeout: time.Minute}, UserAgent: userAgent()},
		Checker:  sc, PassmcpVersion: ver, OptOut: opt, Site: &site.Site{Dir: *siteDir},
		Queue: &disclosure.Queue{Dir: *priv}, Limiter: limiter.New(*perHost), Workers: *workers,
		Now: time.Now, Log: stderr,
	}
	sum, err := job.Run(ctx, cfg)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "listed %d, checked %d, published %d, withheld %d, unreachable %d, opted out %d, skipped %d, pruned %d\n",
		sum.Listed, sum.Checked, sum.Published, sum.Withheld, sum.Unreachable, sum.OptedOut, sum.Skipped, len(sum.Pruned))
	return nil
}

func isUnder(abs, dir string) bool {
	d, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(d, abs)
	return err == nil && rel != ".." && (len(rel) < 3 || rel[:3] != "../")
}

func cmdVerify(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	siteDir := fs.String("site", "site", "the published site directory")
	bundles := fs.Bool("require-bundles", false, "fail when a record has no sigstore bundle beside its statement")
	if err := parse(fs, args, stderr); err != nil {
		return err
	}
	recs, err := (&site.Site{Dir: *siteDir}).Records()
	if err != nil {
		return err
	}
	bad := 0
	for _, r := range recs {
		if err := verifyRecord(*siteDir, r, *bundles); err != nil {
			bad++
			_, _ = fmt.Fprintf(stderr, "%s@%s: %v\n", r.Name, r.Version, err)
		}
	}
	if bad > 0 {
		return fmt.Errorf("%d of %d records do not verify", bad, len(recs))
	}
	_, _ = fmt.Fprintf(stdout, "verify: %d records, every statement matches its digest and verifies\n", len(recs))
	return nil
}

func verifyRecord(siteDir string, r site.Record, requireBundle bool) error {
	b, err := os.ReadFile(filepath.Join(siteDir, filepath.FromSlash(r.Attestation))) // #nosec G304 -- under the site
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != r.StatementSHA256 {
		return errors.New("the statement's digest is not the one the record names")
	}
	st, err := attestation.Parse(b)
	if err != nil {
		return err
	}
	if !st.Covers("http", r.URL) {
		return fmt.Errorf("the statement is not about %s", r.URL)
	}
	if requireBundle {
		if _, err := os.Stat(filepath.Join(siteDir, filepath.FromSlash(r.Bundle))); err != nil {
			return errors.New("no sigstore bundle beside the statement")
		}
	}
	return nil
}

func cmdReport(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	siteDir := fs.String("site", "site", "the published site directory")
	year := fs.Int("year", time.Now().UTC().Year()-1, "the year to report on")
	out := fs.String("out", "report", "where to write the report")
	if err := parse(fs, args, stderr); err != nil {
		return err
	}
	rep, err := annual.Build(*siteDir, *year, *out)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "report %d: %d records, %d rejected, written to %s\n", rep.Year, rep.Records, len(rep.Rejected), *out)
	return nil
}
