// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package disclosure keeps the private queue of results withheld from the
// scorecard, each with a drafted notice for the server's owner.
//
// The queue is written to a private directory, never to the site, and this
// package sends nothing: a person reads each draft and sends it under
// DISCLOSURE.md. The ninety days run from the first time a finding was
// seen, so running the job again does not restart the clock.
package disclosure

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"satellion.com/passmcp-registry/internal/policy"
	"satellion.com/passmcp-registry/internal/site"
)

// Window is how long a withheld result stays private before it may be
// published, unless the owner fixes it sooner.
const Window = 90 * 24 * time.Hour

// Entry is one withheld result.
type Entry struct {
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	URL            string          `json:"url"`
	FirstFound     time.Time       `json:"first_found"`
	LastSeen       time.Time       `json:"last_seen"`
	DiscloseAfter  time.Time       `json:"disclose_after"`
	MethodVersion  string          `json:"method_version"`
	Reasons        []policy.Reason `json:"reasons"`
	NoticeDraft    string          `json:"notice_draft"`
	NoticeSent     bool            `json:"notice_sent"`
	NoticeSentNote string          `json:"notice_sent_note,omitempty"`
}

// Queue is the private directory the entries live in.
type Queue struct {
	Dir string
}

// Add records a withheld result and writes its notice draft. An entry that
// already exists keeps its first date and its sent status.
func (q *Queue) Add(name, version, url, method string, reasons []policy.Reason, now time.Time) (Entry, error) {
	if q.Dir == "" {
		return Entry{}, errors.New("disclosure: no private directory configured")
	}
	if err := os.MkdirAll(q.Dir, 0o700); err != nil {
		return Entry{}, err
	}
	base := site.Slug(name) + "@" + site.Slug(version)
	path := filepath.Join(q.Dir, base+".json")
	e := Entry{Name: name, Version: version, URL: url, FirstFound: now, MethodVersion: method}
	if b, err := os.ReadFile(path); err == nil { // #nosec G304 -- under the private directory
		var prev Entry
		if json.Unmarshal(b, &prev) == nil && !prev.FirstFound.IsZero() {
			e.FirstFound, e.NoticeSent, e.NoticeSentNote = prev.FirstFound, prev.NoticeSent, prev.NoticeSentNote
		}
	}
	e.LastSeen = now
	e.DiscloseAfter = e.FirstFound.Add(Window)
	e.Reasons = reasons
	e.NoticeDraft = base + ".md"
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return e, err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return e, err
	}
	return e, os.WriteFile(filepath.Join(q.Dir, e.NoticeDraft), []byte(Notice(e)), 0o600)
}

// Notice drafts the message to the owner. It says what was seen and how,
// what happens next and by when, and how to check the fix; it contains
// nothing the owner could not reproduce with the published method.
func Notice(e Entry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Subject: Security finding in %s, from the passmcp-registry scorecard\n\n", e.Name)
	fmt.Fprintf(&b, "Hello,\n\nThe passmcp-registry scorecard checks the remote MCP servers listed in the official MCP Registry, read-only and without credentials. On %s it checked %s (%s, version %s) and found:\n\n",
		e.FirstFound.UTC().Format("2 January 2006"), e.Name, e.URL, e.Version)
	for _, r := range e.Reasons {
		fmt.Fprintf(&b, "- %s\n  %s\n", r.Rule, r.Detail)
	}
	fmt.Fprintf(&b, "\nThis result is withheld from the public scorecard. It will stay withheld until %s, or until a later run no longer finds it, whichever comes first, under the policy in DISCLOSURE.md: https://github.com/sebastienrousseau/passmcp-registry/blob/main/DISCLOSURE.md\n\n",
		e.DiscloseAfter.UTC().Format("2 January 2006"))
	fmt.Fprintf(&b, "You can reproduce the check yourself, with the same flags the scorecard uses (method %s):\n\n    passmcp check %s --phases net,discovery,handshake,protocol,catalog --auth none\n\n", e.MethodVersion, e.URL)
	b.WriteString("If you would rather your server were not checked at all, the opt-out process is in OPT-OUT.md.\n\nThank you,\npassmcp-registry maintainers\n")
	return b.String()
}
