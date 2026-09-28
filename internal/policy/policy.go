// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package policy holds the two decisions made about a server before
// anything about it is published: whether its owner opted out, and whether
// its result exposes a vulnerability that has to go to the owner first.
package policy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// OptOut is the set of registry names whose owners asked not to be checked
// or published. OPT-OUT.md is the process; the list is the data.
type OptOut struct {
	exact    map[string]bool
	prefixes []string
}

// LoadOptOut reads the opt-out list: one registry name per line, or a
// namespace ending in "/*" for every server under it. Blank lines and lines
// starting with # are ignored. A missing file is an empty list.
func LoadOptOut(path string) (*OptOut, error) {
	o := &OptOut{exact: map[string]bool{}}
	f, err := os.Open(path) // #nosec G304 -- the operator names the list
	if os.IsNotExist(err) {
		return o, nil
	}
	if err != nil {
		return nil, fmt.Errorf("opt-out list: %w", err)
	}
	defer func() { _ = f.Close() }()
	return o, o.read(f)
}

func (o *OptOut) read(r io.Reader) error {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if ns, ok := strings.CutSuffix(line, "/*"); ok {
			o.prefixes = append(o.prefixes, ns+"/")
			continue
		}
		o.exact[line] = true
	}
	return sc.Err()
}

// Excludes reports whether a server name is opted out.
func (o *OptOut) Excludes(name string) bool {
	if o.exact[name] {
		return true
	}
	for _, p := range o.prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// Reason is one rule a result tripped, and what showed it.
type Reason struct {
	Rule   string `json:"rule"`
	Detail string `json:"detail"`
}

// The rules, in DISCLOSURE.md's words. Each names something an attacker
// could use today and the owner can fix, which is what makes publishing it
// before the owner hears about it harmful.
const (
	RuleOpenMutatingTools = "W1: tools callable with no credentials, not all declared read-only"
	RuleOrigin            = "W2: requests from a foreign Origin are accepted (DNS rebinding)"
	RuleToolPoisoning     = "W3: tool text carries hidden or injected instructions"
	RulePlainAuthServer   = "W4: the authorization server is reached over plain HTTP"
)

type report struct {
	Auth struct {
		Mode    string `json:"mode"`
		Reached bool   `json:"reached"`
	} `json:"auth"`
	Phases []struct {
		Findings []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Detail string `json:"detail"`
		} `json:"findings"`
	} `json:"phases"`
	Catalog struct {
		Tools []struct {
			Name     string `json:"name"`
			ReadOnly bool   `json:"read_only"`
		} `json:"tools"`
	} `json:"catalog"`
}

// maxDetail bounds what a reason quotes from the server's own text.
const maxDetail = 300

// Withhold returns every rule a passmcp JSON report trips. An empty result
// means the record may be published.
func Withhold(reportJSON []byte) ([]Reason, error) {
	var r report
	if err := json.Unmarshal(reportJSON, &r); err != nil {
		return nil, fmt.Errorf("withhold: not a passmcp report: %w", err)
	}
	var out []Reason
	if r.Auth.Mode == "none" {
		if names := notReadOnly(r); len(names) > 0 {
			out = append(out, Reason{RuleOpenMutatingTools, bound(strings.Join(names, ", "))})
		}
	}
	for _, p := range r.Phases {
		for _, f := range p.Findings {
			if f.Status != "fail" {
				continue
			}
			if rule := ruleFor(f.ID); rule != "" {
				out = append(out, Reason{rule, bound(f.ID + ": " + f.Detail)})
			}
		}
	}
	return out, nil
}

func notReadOnly(r report) []string {
	var names []string
	for _, t := range r.Catalog.Tools {
		if !t.ReadOnly {
			names = append(names, t.Name)
		}
	}
	sort.Strings(names)
	return names
}

func ruleFor(id string) string {
	switch {
	case id == "protocol.origin":
		return RuleOrigin
	case strings.HasPrefix(id, "catalog.text."):
		return RuleToolPoisoning
	case id == "discovery.as.https":
		return RulePlainAuthServer
	}
	return ""
}

func bound(s string) string {
	if len(s) <= maxDetail {
		return s
	}
	return s[:maxDetail] + "…"
}
