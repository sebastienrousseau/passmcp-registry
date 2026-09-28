// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !windows

package checker

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakePassmcp is a shell script standing in for passmcp: it records its
// arguments and PASSMCP_CONFIG, and answers each subcommand from files.
// FAKE_ATTEST_EXIT and FAKE_VERSION_EXIT make those subcommands fail.
func fakePassmcp(t *testing.T, checkExit int, report string) (bin, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "argv.log")
	if err := os.WriteFile(filepath.Join(dir, "report.json"), []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
printf '%s\n' "$*" >> "` + log + `"
printf 'PASSMCP_CONFIG=%s\n' "$(cat "$PASSMCP_CONFIG" 2>/dev/null)" >> "` + log + `"
case "$1" in
  version) echo "passmcp 0.0.8"; exit "${FAKE_VERSION_EXIT:-0}" ;;
  check) cat "` + filepath.Join(dir, "report.json") + `"; echo "check noise" >&2; exit ` + strconv.Itoa(checkExit) + ` ;;
  attest) cat >/dev/null; echo '{"_type":"statement"}'; exit "${FAKE_ATTEST_EXIT:-0}" ;;
esac
exit 9
`
	bin = filepath.Join(dir, "passmcp")
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return bin, log
}

// AC: REG-01
func TestTheCheckRunsOnlyNonInvokingPhasesUnauthenticated(t *testing.T) {
	bin, log := fakePassmcp(t, 2, `{"score":{"total":70}}`)
	s := &Passmcp{Bin: bin, UserAgent: "passmcp-registry/0.0.1 (+https://example)", RPS: 1, Timeout: 20 * time.Second}
	res, err := s.Check(context.Background(), "https://srv.example/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if res.Exit != 2 || !strings.Contains(string(res.Statement), "statement") {
		t.Fatalf("result %+v", res)
	}
	b, _ := os.ReadFile(log)
	argv := string(b)
	if !strings.Contains(argv, "check https://srv.example/mcp --phases net,discovery,handshake,protocol,catalog --auth none") {
		t.Fatalf("argv:\n%s", argv)
	}
	for _, banned := range []string{"execution", "performance", "resilience", "--allow-mutations", "--allow-destructive", ",auth", "--token", "--client-secret", "--capture-bodies"} {
		if strings.Contains(argv, banned) {
			t.Errorf("passmcp was given %q:\n%s", banned, argv)
		}
	}
	if !strings.Contains(argv, "User-Agent: passmcp-registry/0.0.1") {
		t.Error("the published User-Agent was not passed")
	}
	if !strings.Contains(argv, "PASSMCP_CONFIG={}") {
		t.Error("passmcp did not run with an empty configuration")
	}
}

func TestAPassmcpFailureIsAnErrorWithItsOutputBounded(t *testing.T) {
	bin, _ := fakePassmcp(t, 1, "")
	s := &Passmcp{Bin: bin, RPS: 1, Timeout: time.Second}
	if _, err := s.Check(context.Background(), "https://down.example/mcp"); err == nil || !strings.Contains(err.Error(), "exited 1") {
		t.Fatalf("err %v", err)
	}
	if _, err := (&Passmcp{Bin: filepath.Join(t.TempDir(), "absent")}).Check(context.Background(), "https://x/mcp"); err == nil {
		t.Fatal("want an error for a missing binary")
	}
	if got := tail([]byte(strings.Repeat("a", 1000))); len(got) > maxStderr+len("…") {
		t.Fatal("stderr not bounded")
	}
}

func TestVersionReadsTheBinary(t *testing.T) {
	bin, _ := fakePassmcp(t, 0, "{}")
	v, err := (&Passmcp{Bin: bin}).Version(context.Background())
	if err != nil || v != "0.0.8" {
		t.Fatalf("version %q, err %v", v, err)
	}
}

func TestAttestAndVersionFailuresAreErrors(t *testing.T) {
	bin, _ := fakePassmcp(t, 0, `{"score":{"total":90}}`)
	s := &Passmcp{Bin: bin, RPS: 1, Timeout: time.Second}

	t.Setenv("FAKE_ATTEST_EXIT", "3")
	res, err := s.Check(context.Background(), "https://srv.example/mcp")
	if err == nil || !strings.Contains(err.Error(), "attest exited 3") {
		t.Fatalf("a failed attestation is an error: %v", err)
	}
	if len(res.Report) == 0 || res.Statement != nil {
		t.Errorf("the report is kept and no statement is claimed: %+v", res)
	}
	t.Setenv("FAKE_ATTEST_EXIT", "0")

	t.Setenv("FAKE_VERSION_EXIT", "4")
	if _, err := s.Version(context.Background()); err == nil || !strings.Contains(err.Error(), "version exited 4") {
		t.Errorf("a failing version is an error: %v", err)
	}
	if _, err := (&Passmcp{Bin: filepath.Join(t.TempDir(), "absent")}).Version(context.Background()); err == nil {
		t.Error("a missing binary has no version")
	}
}

func TestNoTemporaryDirectoryMeansNoCheck(t *testing.T) {
	bin, log := fakePassmcp(t, 0, "{}")
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "absent"))
	if _, err := (&Passmcp{Bin: bin}).Check(context.Background(), "https://srv.example/mcp"); err == nil {
		t.Fatal("without the empty configuration, passmcp must not run")
	}
	if b, _ := os.ReadFile(log); len(b) != 0 {
		t.Errorf("passmcp ran without its empty configuration:\n%s", b)
	}
}
