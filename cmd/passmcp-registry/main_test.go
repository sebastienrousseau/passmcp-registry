// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !windows

package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-reporting/attestation"
)

func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func statementFor(t *testing.T, endpoint string) []byte {
	t.Helper()
	target := attestation.Target{Transport: "http", Endpoint: endpoint}
	s := &attestation.Statement{Type: attestation.StatementType, Subject: []attestation.Subject{attestation.SubjectFor(target)},
		PredicateType: attestation.PredicateType, Predicate: attestation.Evaluation{
			SubjectKind: attestation.SubjectKindDescriptor, Target: target,
			JudgedAgainst: attestation.Basis{Rubric: "1", CheckInventory: "120"},
			Instrument:    attestation.Instrument{Name: "passmcp", Version: "0.0.8", SchemaVersion: 1},
			RanAt:         time.Now().UTC(), Took: "1s",
			Verdicts: []attestation.Verdict{{ID: "net.tcp", Phase: "net", Status: "pass"}}, Counts: attestation.Counts{Pass: 1}}}
	b, err := s.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// world is a fake registry listing two servers, and a passmcp stand-in that
// answers for exactly those.
func world(t *testing.T) (registryURL, passmcpBin, dir string) {
	t.Helper()
	dir = t.TempDir()
	urls := []string{"https://one.example/mcp", "https://two.example/mcp"}
	reg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"servers":[
		 {"server":{"name":"io.github.a/one","version":"1","remotes":[{"type":"streamable-http","url":%q}]}},
		 {"server":{"name":"io.github.b/two","version":"2","remotes":[{"type":"streamable-http","url":%q}]}}],
		 "metadata":{}}`, urls[0], urls[1])
	}))
	t.Cleanup(reg.Close)
	for _, u := range urls {
		if err := os.WriteFile(filepath.Join(dir, "stmt-"+safeName(u)), statementFor(t, u), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	report := `{"auth":{"mode":"none"},"score":{"total":91,"grade":"A"},"counts":{"pass":1},"catalog":{"tools":[]},"phases":[]}`
	_ = os.WriteFile(filepath.Join(dir, "report.json"), []byte(report), 0o600)
	script := fmt.Sprintf(`#!/bin/sh
case "$1" in
  version) echo "passmcp 0.0.8" ;;
  check) printf '%%s' "$2" > %[1]q/last; cat %[1]q/report.json; exit 0 ;;
  attest) cat >/dev/null; cat %[1]q/stmt-"$(tr -c 'a-zA-Z0-9' '_' < %[1]q/last)" ;;
  *) exit 9 ;;
esac
`, dir)
	passmcpBin = filepath.Join(dir, "passmcp")
	if err := os.WriteFile(passmcpBin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return reg.URL, passmcpBin, dir
}

func runCLI(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(context.Background(), args, &out, &errb)
	return code, out.String(), errb.String()
}

// AC: REG-02
func TestRunThenVerifyEndToEnd(t *testing.T) {
	reg, bin, dir := world(t)
	siteDir := filepath.Join(dir, "site")
	code, out, errOut := runCLI("run", "--registry", reg, "--passmcp", bin, "--site", siteDir,
		"--private", filepath.Join(dir, "private"), "--opt-out", filepath.Join(dir, "none.txt"), "--per-host", "0s",
		// The stand-in remembers the last URL in one file; real passmcp keeps
		// no state between runs, and the job's own tests cover concurrency.
		"--workers", "1")
	if code != 0 {
		t.Fatalf("run exited %d: %s %s", code, out, errOut)
	}
	if !strings.Contains(out, "published 2") {
		t.Fatalf("summary: %s", out)
	}
	checkVerifyAndReport(t, siteDir, filepath.Join(dir, "report"))
}

// checkVerifyAndReport verifies a freshly run, unsigned site, then tampers
// with a statement and checks that verify and report both catch it.
func checkVerifyAndReport(t *testing.T, siteDir, reportDir string) {
	t.Helper()
	if code, out, errOut := runCLI("verify", "--site", siteDir); code != 0 || !strings.Contains(out, "2 records") {
		t.Fatalf("verify: %d %s %s", code, out, errOut)
	}
	// Before the signing step, a bundle is missing, and --require-bundles
	// says so.
	if code, _, errOut := runCLI("verify", "--site", siteDir, "--require-bundles"); code != 1 || !strings.Contains(errOut, "sigstore bundle") {
		t.Fatalf("require-bundles: %d %s", code, errOut)
	}
	// An edited statement is caught.
	matches, _ := filepath.Glob(filepath.Join(siteDir, "servers", "*", "*", "attestation.json"))
	_ = os.WriteFile(matches[0], []byte("{}"), 0o600)
	if code, _, errOut := runCLI("verify", "--site", siteDir); code != 1 || !strings.Contains(errOut, "digest") {
		t.Fatalf("tampered: %d %s", code, errOut)
	}
	year := fmt.Sprint(time.Now().UTC().Year())
	if code, out, _ := runCLI("report", "--site", siteDir, "--year", year, "--out", reportDir); code != 0 || !strings.Contains(out, "1 rejected") {
		t.Fatalf("report: %d %s", code, out)
	}
}

func TestUsageAndErrors(t *testing.T) {
	for _, c := range []struct {
		args []string
		code int
		want string
	}{
		{nil, 2, "usage"},
		{[]string{"nope"}, 2, "unknown command"},
		{[]string{"verify", "--bogus"}, 2, "flag provided but not defined"},
		{[]string{"verify", "extra"}, 2, "unexpected argument"},
		{[]string{"run", "--site", "s", "--private", "s/private"}, 2, "must not be inside the site"},
		{[]string{"run", "--passmcp", "/nonexistent/passmcp", "--private", "/tmp/p-x", "--site", "/tmp/s-x"}, 1, "running passmcp"},
		{[]string{"report", "--site", "/nonexistent/site", "--out", "/dev/null/x"}, 1, ""},
	} {
		code, out, errOut := runCLI(c.args...)
		if code != c.code || !strings.Contains(out+errOut, c.want) {
			t.Errorf("%v: exit %d (want %d), output %q", c.args, code, c.code, out+errOut)
		}
	}
	if code, out, _ := runCLI("version"); code != 0 || !strings.Contains(out, "passmcp-registry") {
		t.Fatalf("version: %d %s", code, out)
	}
	if !strings.Contains(userAgent(), "passmcp-registry/") {
		t.Fatal("the user agent does not name the project")
	}
}
