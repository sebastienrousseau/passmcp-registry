// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"bytes"
	"net/url"
	"strings"
	"testing"
)

// FuzzListingPage feeds arbitrary bytes to the page decoder and files every
// entry, as List does with a page the registry sent. Whatever the registry
// answers, nothing panics, and every endpoint that reaches the check list
// is one the job may contact: a named server, an HTTP transport, and a
// plain http(s) URL with a host, no template and no credentials. Every
// endpoint set aside says why.
func FuzzListingPage(f *testing.F) {
	for _, seed := range []string{
		page1, page2, "", "null", "{}", `{"servers":null}`, `{"servers":[1,"x",null]}`,
		`{"servers":[{"name":"a/b","remotes":[{"type":"sse","url":"http://h/"}]}]}`,
		`{"servers":[{"server":null,"_meta":{"io.modelcontextprotocol.registry/official":7}}]}`,
		`{"servers":[{"name":"a/b","remotes":[{"type":"sse","url":"http://%zz"}]}]}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		p, err := readPage(bytes.NewReader(data))
		if err != nil {
			return
		}
		var out Listing
		for _, raw := range p.Servers {
			addEntry(&out, raw)
		}
		for _, r := range out.Remotes {
			checkable(t, r)
		}
		for _, s := range out.Skipped {
			if s.Name == "" || s.Reason == "" {
				t.Fatalf("an endpoint set aside without a name or a reason: %+v", s)
			}
		}
	})
}

// checkable fails unless r is an endpoint unusable would let through.
func checkable(t *testing.T, r Remote) {
	t.Helper()
	if r.Name == "" {
		t.Fatalf("an unnamed server was listed: %+v", r)
	}
	if r.Type != "streamable-http" && r.Type != "sse" {
		t.Fatalf("a %q transport was listed: %+v", r.Type, r)
	}
	if strings.ContainsAny(r.URL, "{}") {
		t.Fatalf("a URL template was listed: %+v", r)
	}
	u, err := url.Parse(r.URL)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		t.Fatalf("an endpoint the job must not contact was listed: %+v", r)
	}
}
