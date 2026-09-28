// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC: REG-03
func TestOptOutMatchesNamesAndNamespaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opt-out.txt")
	body := "# owners who asked\n\nio.github.alice/one\nio.github.bob/*\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	o, err := LoadOptOut(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{
		"io.github.alice/one": true, "io.github.alice/two": false,
		"io.github.bob/anything": true, "io.github.bobby/x": false,
	} {
		if o.Excludes(name) != want {
			t.Errorf("Excludes(%q) = %v", name, !want)
		}
	}
	missing, err := LoadOptOut(filepath.Join(t.TempDir(), "absent"))
	if err != nil || missing.Excludes("io.github.alice/one") {
		t.Fatal("a missing list must be empty, not an error")
	}
	if _, err := LoadOptOut(t.TempDir()); err == nil {
		t.Fatal("a directory is not a list")
	}
}

// AC: REG-04
func TestWithholdNamesEveryRuleAResultTrips(t *testing.T) {
	report := `{"auth":{"mode":"none","reached":true},
	 "catalog":{"tools":[{"name":"delete_repo","read_only":false},{"name":"list","read_only":true}]},
	 "phases":[{"findings":[
	   {"id":"protocol.origin","status":"fail","detail":"accepted Origin: https://evil.example"},
	   {"id":"catalog.text.hidden","status":"fail","detail":"zero-width text in delete_repo"},
	   {"id":"discovery.as.https","status":"fail","detail":"http://auth.example"},
	   {"id":"net.tls.cert","status":"fail","detail":"expired"},
	   {"id":"catalog.text.comments","status":"pass"}]}]}`
	got, err := Withhold([]byte(report))
	if err != nil {
		t.Fatal(err)
	}
	var rules []string
	for _, r := range got {
		rules = append(rules, r.Rule)
	}
	want := []string{RuleOpenMutatingTools, RuleOrigin, RuleToolPoisoning, RulePlainAuthServer}
	if strings.Join(rules, "|") != strings.Join(want, "|") {
		t.Fatalf("rules %v, want %v", rules, want)
	}
	if got[0].Detail != "delete_repo" {
		t.Errorf("W1 detail %q", got[0].Detail)
	}
}

func TestWithholdPublishesAnOrdinaryResult(t *testing.T) {
	for _, r := range []string{
		`{"auth":{"mode":"none"},"catalog":{"tools":[{"name":"read","read_only":true}]},"phases":[{"findings":[{"id":"net.tls.cert","status":"fail"}]}]}`,
		`{"auth":{"mode":"none"},"catalog":{"tools":[]},"phases":[]}`,
		// A server that refused the unauthenticated tools/list lists nothing.
		`{"auth":{"mode":"bearer"},"catalog":{"tools":[{"name":"w","read_only":false}]},"phases":[]}`,
	} {
		got, err := Withhold([]byte(r))
		if err != nil || len(got) != 0 {
			t.Fatalf("%s: withheld %v (err %v)", r, got, err)
		}
	}
	if _, err := Withhold([]byte("{")); err == nil {
		t.Fatal("want an error for a broken report")
	}
}

func TestWithholdBoundsTheServersText(t *testing.T) {
	long := strings.Repeat("x", 1000)
	got, _ := Withhold([]byte(`{"phases":[{"findings":[{"id":"protocol.origin","status":"fail","detail":"` + long + `"}]}]}`))
	if len(got) != 1 || len(got[0].Detail) > maxDetail+len("…") {
		t.Fatalf("detail not bounded: %d", len(got[0].Detail))
	}
}
