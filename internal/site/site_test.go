// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func rec(name, version string) Record {
	return Record{Name: name, Version: version, URL: "https://x.example/mcp", Transport: "streamable-http",
		CheckedAt: time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC), MethodVersion: "1", PassmcpVersion: "0.0.8", Score: 90, Grade: "A"}
}

func TestPublishWritesTheRecordAndItsStatement(t *testing.T) {
	s := &Site{Dir: t.TempDir()}
	if err := s.Publish(rec("io.github.a/one", "1.0.0"), []byte(`{"stmt":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteIndex(Index{MethodVersion: "1", PassmcpVersion: "0.0.8"}); err != nil {
		t.Fatal(err)
	}
	ix := readIndex(t, s.Dir)
	if ix.Licence != "CC-BY-4.0" || len(ix.Records) != 1 {
		t.Fatalf("index %+v", ix)
	}
	r := ix.Records[0]
	if r.Attestation != "servers/io.github.a~one/1.0.0/attestation.json" || r.Bundle != "servers/io.github.a~one/1.0.0/attestation.sigstore.json" {
		t.Fatalf("paths %q %q", r.Attestation, r.Bundle)
	}
	if r.StatementSHA256 != "8f5cd1fbcbd2c93bc389c9a0e2ea6a95ad1d9b5c0bde4c0e7a4bcc0e0e3b4b39" && len(r.StatementSHA256) != 64 {
		t.Fatalf("digest %q", r.StatementSHA256)
	}
	sign, _ := os.ReadFile(filepath.Join(s.Dir, ToSignFile))
	if strings.TrimSpace(string(sign)) != r.Attestation {
		t.Fatalf("to-sign %q", sign)
	}
}

func readIndex(t *testing.T, dir string) Index {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ix Index
	if err := json.Unmarshal(b, &ix); err != nil {
		t.Fatal(err)
	}
	return ix
}

// AC: REG-03
func TestPruneRemovesEveryRecordOfAnOptedOutServer(t *testing.T) {
	s := &Site{Dir: t.TempDir()}
	for _, r := range []Record{rec("io.github.a/one", "1.0.0"), rec("io.github.a/one", "1.1.0"), rec("io.github.b/two", "0.1.0")} {
		if err := s.Publish(r, []byte("{}")); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := s.Prune(func(n string) bool { return n == "io.github.a/one" })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(removed, ",") != "io.github.a/one" {
		t.Fatalf("removed %v", removed)
	}
	recs, _ := s.Records()
	if len(recs) != 1 || recs[0].Name != "io.github.b/two" {
		t.Fatalf("left %+v", recs)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "servers", "io.github.a~one")); !os.IsNotExist(err) {
		t.Fatal("the opted-out server's directory survived")
	}
}

func TestRemoveAndEmptySite(t *testing.T) {
	s := &Site{Dir: t.TempDir()}
	recs, err := s.Records()
	if err != nil || len(recs) != 0 {
		t.Fatalf("empty site: %v %v", recs, err)
	}
	_ = s.Publish(rec("x/y", "1"), []byte("{}"))
	if err := s.Remove("x/y", "1"); err != nil {
		t.Fatal(err)
	}
	if recs, _ := s.Records(); len(recs) != 0 {
		t.Fatal("record survived Remove")
	}
	if err := s.WriteIndex(Index{}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(s.Dir, "index.json"))
	if !strings.Contains(string(b), `"unreachable": []`) || !strings.Contains(string(b), `"skipped": []`) {
		t.Fatalf("empty lists must be present:\n%s", b)
	}
}

func TestRecordsRejectsACorruptRecord(t *testing.T) {
	s := &Site{Dir: t.TempDir()}
	dir := filepath.Join(s.Dir, "servers", "a", "1")
	_ = os.MkdirAll(dir, 0o750)
	_ = os.WriteFile(filepath.Join(dir, RecordFile), []byte("{"), 0o600)
	if _, err := s.Records(); err == nil {
		t.Fatal("want an error for a corrupt record")
	}
	if _, err := s.Prune(func(string) bool { return true }); err == nil {
		t.Fatal("prune must surface the error")
	}
	if err := s.WriteIndex(Index{}); err == nil {
		t.Fatal("index must surface the error")
	}
}

func TestSlugKeepsPathsSafe(t *testing.T) {
	for in, want := range map[string]string{
		"io.github.a/one": "io.github.a~one", "": "-", "a b@c": "a-b-c", "...": "-", "..": "-",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
	// A slug is one path segment: no separator survives, and it is never
	// "." or "..", so a hostile name cannot climb out of the site.
	for _, in := range []string{"../../etc", "a/../../b", `..\..\x`, "./."} {
		got := Slug(in)
		if strings.ContainsAny(got, `/\`) || got == "." || got == ".." {
			t.Errorf("Slug(%q) = %q is not a safe segment", in, got)
		}
	}
}

func TestRecordsRefusesARecordThatLeadsOutsideTheSite(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, RecordFile), []byte(`{"name":"planted"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Site{Dir: t.TempDir()}
	dir := filepath.Join(s.Dir, serversDir, "x", "1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, RecordFile), filepath.Join(dir, RecordFile)); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	recs, err := s.Records()
	if err == nil || len(recs) != 0 {
		t.Fatalf("records %+v, err %v: a symlink out of the site was followed", recs, err)
	}
}
