// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package disclosure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-registry/internal/policy"
)

// AC: REG-04
func TestAWithheldResultIsQueuedWithANoticeAndANinetyDayClock(t *testing.T) {
	q := &Queue{Dir: filepath.Join(t.TempDir(), "private")}
	first := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	reasons := []policy.Reason{{Rule: policy.RuleOrigin, Detail: "protocol.origin: accepted https://evil.example"}}
	e, err := q.Add("io.github.a/one", "1.0.0", "https://one.example/mcp", "1", reasons, first)
	if err != nil {
		t.Fatal(err)
	}
	if !e.DiscloseAfter.Equal(first.Add(90 * 24 * time.Hour)) {
		t.Fatalf("disclose after %v", e.DiscloseAfter)
	}
	notice, err := os.ReadFile(filepath.Join(q.Dir, e.NoticeDraft))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"io.github.a/one", "https://one.example/mcp", "withheld", "26 December 2026", "DISCLOSURE.md", "--auth none", policy.RuleOrigin} {
		if !strings.Contains(string(notice), want) {
			t.Errorf("notice lacks %q:\n%s", want, notice)
		}
	}

	// A later run finds it again: the clock keeps its first date, and a
	// notice already marked sent stays marked.
	path := filepath.Join(q.Dir, "io.github.a~one@1.0.0.json")
	var saved Entry
	b, _ := os.ReadFile(path)
	_ = json.Unmarshal(b, &saved)
	saved.NoticeSent, saved.NoticeSentNote = true, "sent by email 28 Sep"
	b, _ = json.Marshal(saved)
	_ = os.WriteFile(path, b, 0o600)
	again, err := q.Add("io.github.a/one", "1.0.0", "https://one.example/mcp", "1", reasons, first.Add(7*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !again.FirstFound.Equal(first) || !again.NoticeSent || again.NoticeSentNote == "" {
		t.Fatalf("re-run reset the entry: %+v", again)
	}
}

func TestTheQueueNeedsAPrivateDirectory(t *testing.T) {
	if _, err := (&Queue{}).Add("a", "1", "u", "1", nil, time.Now()); err == nil {
		t.Fatal("want an error")
	}
	file := filepath.Join(t.TempDir(), "f")
	_ = os.WriteFile(file, nil, 0o600)
	if _, err := (&Queue{Dir: file}).Add("a", "1", "u", "1", nil, time.Now()); err == nil {
		t.Fatal("a file is not a directory")
	}
}
