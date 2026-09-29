// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package policy

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzWithhold feeds arbitrary bytes to Withhold as a passmcp report. A
// report the server shaped cannot make it panic; an error comes with no
// reasons; every reason names one of the four DISCLOSURE.md rules and
// quotes at most maxDetail bytes of the server's text; and the decision is
// the same on a second look, since the disclosure clock depends on it.
func FuzzWithhold(f *testing.F) {
	for _, seed := range []string{
		"", "null", "{}", "[]",
		`{"auth":{"mode":"none"},"catalog":{"tools":[{"name":"rm","read_only":false},{"name":"ls","read_only":true}]}}`,
		`{"auth":{"mode":"oauth"},"catalog":{"tools":[{"name":"rm"}]}}`,
		`{"phases":[{"findings":[{"id":"protocol.origin","status":"fail","detail":"accepted"}]}]}`,
		`{"phases":[{"findings":[{"id":"catalog.text.hidden","status":"fail","detail":"` + strings.Repeat("x", 400) + `"}]}]}`,
		`{"phases":[{"findings":[{"id":"discovery.as.https","status":"pass"}]},null]}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := Withhold(data)
		if err != nil {
			if got != nil {
				t.Fatalf("an error came with reasons: %v", got)
			}
			return
		}
		for _, r := range got {
			checkReason(t, r)
		}
		again, _ := Withhold(data)
		if !reflect.DeepEqual(got, again) {
			t.Fatalf("the same report was judged twice differently: %v, then %v", got, again)
		}
	})
}

// checkReason fails unless r names a known rule and its detail is bounded.
func checkReason(t *testing.T, r Reason) {
	t.Helper()
	switch r.Rule {
	case RuleOpenMutatingTools, RuleOrigin, RuleToolPoisoning, RulePlainAuthServer:
	default:
		t.Fatalf("an unknown rule: %q", r.Rule)
	}
	if len(r.Detail) > maxDetail+utf8.RuneLen('…') {
		t.Fatalf("a detail of %d bytes escaped the bound", len(r.Detail))
	}
}
