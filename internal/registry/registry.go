// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package registry lists the remote MCP servers published in an MCP
// Registry, through the registry's v0 servers API.
//
// It reads two response shapes, because the official registry has served
// both: entries that are the server document itself, with its registry
// metadata under "_meta", and entries that wrap it as {"server": …,
// "_meta": …}. Pagination follows metadata.nextCursor or
// metadata.next_cursor, whichever the registry sends.
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// officialMeta is the key the official registry files its own metadata
// under.
const officialMeta = "io.modelcontextprotocol.registry/official"

// maxPages bounds a listing, so a registry that never stops paging cannot
// keep the job running.
const maxPages = 1000

// maxBody bounds one page of the listing.
const maxBody = 16 << 20

// Remote is one HTTP endpoint a server declares.
type Remote struct {
	Name    string // the server's registry name, e.g. io.github.owner/server
	Version string // the server version the listing names
	Type    string // streamable-http or sse
	URL     string
}

// Skipped is a listed endpoint the job does not check, and why.
type Skipped struct {
	Name, Version, URL, Reason string
}

// Listing is what a registry lists: the endpoints to check and the
// endpoints set aside.
type Listing struct {
	Remotes []Remote
	Skipped []Skipped
}

// Client reads a registry.
type Client struct {
	Base      string // e.g. https://registry.modelcontextprotocol.io
	HTTP      *http.Client
	UserAgent string
	PageSize  int
}

type page struct {
	Servers  []json.RawMessage `json:"servers"`
	Metadata struct {
		NextCursor  string `json:"nextCursor"`
		NextCursor2 string `json:"next_cursor"`
	} `json:"metadata"`
}

type serverDoc struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Remotes []struct {
		Type          string `json:"type"`
		TransportType string `json:"transport_type"`
		URL           string `json:"url"`
		Headers       []struct {
			Name       string `json:"name"`
			IsRequired bool   `json:"isRequired"`
		} `json:"headers"`
	} `json:"remotes"`
	Meta map[string]json.RawMessage `json:"_meta"`
}

type wrapped struct {
	Server *serverDoc                 `json:"server"`
	Meta   map[string]json.RawMessage `json:"_meta"`
}

type official struct {
	Status   string `json:"status"`
	IsLatest *bool  `json:"isLatest"`
}

// List pages through the registry and returns every remote endpoint of the
// latest version of every active server.
func (c *Client) List(ctx context.Context) (Listing, error) {
	var out Listing
	cursor := ""
	for i := 0; i < maxPages; i++ {
		p, err := c.fetch(ctx, cursor)
		if err != nil {
			return out, err
		}
		for _, raw := range p.Servers {
			addEntry(&out, raw)
		}
		next := p.Metadata.NextCursor
		if next == "" {
			next = p.Metadata.NextCursor2
		}
		if next == "" || next == cursor {
			return out, nil
		}
		cursor = next
	}
	return out, fmt.Errorf("registry: more than %d pages; refusing to continue", maxPages)
}

func (c *Client) fetch(ctx context.Context, cursor string) (page, error) {
	var p page
	u, err := url.Parse(strings.TrimRight(c.Base, "/") + "/v0/servers")
	if err != nil {
		return p, fmt.Errorf("registry: base URL: %w", err)
	}
	q := u.Query()
	size := c.PageSize
	if size <= 0 {
		size = 100
	}
	q.Set("limit", fmt.Sprint(size))
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return p, err
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return p, fmt.Errorf("registry: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return p, fmt.Errorf("registry: %s answered %d", u.Redacted(), resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return p, fmt.Errorf("registry: %w", err)
	}
	if len(body) > maxBody {
		return p, errors.New("registry: a page exceeded 16 MiB")
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return p, fmt.Errorf("registry: not a v0 servers page: %w", err)
	}
	return p, nil
}

// addEntry reads one listing entry in either shape and files its remotes.
func addEntry(out *Listing, raw json.RawMessage) {
	var w wrapped
	if err := json.Unmarshal(raw, &w); err != nil {
		return
	}
	doc := w.Server
	meta := w.Meta
	if doc == nil {
		var d serverDoc
		if err := json.Unmarshal(raw, &d); err != nil {
			return
		}
		doc, meta = &d, d.Meta
	}
	if doc.Name == "" || !current(doc, meta) {
		return
	}
	for _, r := range doc.Remotes {
		typ := r.Type
		if typ == "" {
			typ = r.TransportType
		}
		if reason := unusable(typ, r.URL, requiredHeaders(r.Headers)); reason != "" {
			out.Skipped = append(out.Skipped, Skipped{doc.Name, doc.Version, r.URL, reason})
			continue
		}
		out.Remotes = append(out.Remotes, Remote{Name: doc.Name, Version: doc.Version, Type: typ, URL: r.URL})
	}
}

// current reports whether an entry is the latest version of an active
// server. Older versions and deleted or deprecated servers are not
// checked: the scorecard is about what an agent would connect to today.
func current(doc *serverDoc, meta map[string]json.RawMessage) bool {
	status := doc.Status
	var off official
	if raw, ok := meta[officialMeta]; ok {
		_ = json.Unmarshal(raw, &off)
		if off.Status != "" {
			status = off.Status
		}
		if off.IsLatest != nil && !*off.IsLatest {
			return false
		}
	}
	return status == "" || status == "active"
}

func requiredHeaders(hs []struct {
	Name       string `json:"name"`
	IsRequired bool   `json:"isRequired"`
}) []string {
	var names []string
	for _, h := range hs {
		if h.IsRequired {
			names = append(names, h.Name)
		}
	}
	return names
}

// unusable is why an endpoint cannot be checked unauthenticated and
// unconfigured, or "" when it can.
func unusable(typ, raw string, required []string) string {
	if typ != "streamable-http" && typ != "sse" {
		return "not a remote HTTP transport: " + typ
	}
	if strings.ContainsAny(raw, "{}") {
		return "the URL is a template the user fills in"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "not an http(s) URL"
	}
	if u.User != nil {
		return "the URL carries credentials"
	}
	if len(required) > 0 {
		return "requires headers the user supplies: " + strings.Join(required, ", ")
	}
	return ""
}
