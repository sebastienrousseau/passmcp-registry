// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Two pages: the first in the wrapped shape with nextCursor, the second in
// the flat shape with next_cursor, as the official registry has served both.
const page1 = `{"servers":[
 {"server":{"name":"io.github.a/one","version":"1.2.0","remotes":[{"type":"streamable-http","url":"https://one.example/mcp"}]},
  "_meta":{"io.modelcontextprotocol.registry/official":{"status":"active","isLatest":true}}},
 {"server":{"name":"io.github.a/one","version":"1.1.0","remotes":[{"type":"streamable-http","url":"https://one.example/mcp"}]},
  "_meta":{"io.modelcontextprotocol.registry/official":{"status":"active","isLatest":false}}},
 {"server":{"name":"io.github.b/gone","version":"0.1.0","remotes":[{"type":"sse","url":"https://gone.example/sse"}]},
  "_meta":{"io.modelcontextprotocol.registry/official":{"status":"deleted","isLatest":true}}},
 {"server":{"name":"io.github.c/local","version":"2.0.0","packages":[{"registryType":"npm","identifier":"c"}]}}
],"metadata":{"nextCursor":"p2","count":4}}`

const page2 = `{"servers":[
 {"name":"io.github.d/two","version":"0.3.0","remotes":[
   {"transport_type":"sse","url":"https://two.example/sse"},
   {"type":"streamable-http","url":"https://{tenant}.two.example/mcp"},
   {"type":"streamable-http","url":"https://two.example/keyed","headers":[{"name":"X-API-Key","isRequired":true}]},
   {"type":"stdio","url":""},
   {"type":"streamable-http","url":"ftp://two.example/x"},
   {"type":"streamable-http","url":"https://user:pw@two.example/mcp"}
 ]}
],"metadata":{"next_cursor":""}}`

func fakeRegistry(t *testing.T, seenUA *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/servers" {
			http.NotFound(w, r)
			return
		}
		if seenUA != nil {
			*seenUA = r.Header.Get("User-Agent")
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			fmt.Fprint(w, page1)
		case "p2":
			fmt.Fprint(w, page2)
		default:
			http.Error(w, "bad cursor", http.StatusBadRequest)
		}
	}))
}

// AC: REG-01
func TestListEnumeratesRemoteServersOfTheLatestActiveVersions(t *testing.T) {
	var ua string
	srv := fakeRegistry(t, &ua)
	defer srv.Close()
	c := &Client{Base: srv.URL, HTTP: srv.Client(), UserAgent: "passmcp-registry/test", PageSize: 2}
	got, err := c.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var urls []string
	for _, r := range got.Remotes {
		urls = append(urls, r.Name+"@"+r.Version+" "+r.Type+" "+r.URL)
	}
	want := []string{
		"io.github.a/one@1.2.0 streamable-http https://one.example/mcp",
		"io.github.d/two@0.3.0 sse https://two.example/sse",
	}
	if strings.Join(urls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("remotes:\n%s\nwant:\n%s", strings.Join(urls, "\n"), strings.Join(want, "\n"))
	}
	reasons := map[string]string{}
	for _, s := range got.Skipped {
		reasons[s.URL] = s.Reason
	}
	for url, want := range map[string]string{
		"https://{tenant}.two.example/mcp": "template",
		"https://two.example/keyed":        "requires headers the user supplies: X-API-Key",
		"":                                 "not a remote HTTP transport: stdio",
		"ftp://two.example/x":              "not an http(s) URL",
		"https://user:pw@two.example/mcp":  "credentials",
	} {
		if !strings.Contains(reasons[url], want) {
			t.Errorf("skip reason for %q = %q, want it to mention %q", url, reasons[url], want)
		}
	}
	if ua != "passmcp-registry/test" {
		t.Errorf("User-Agent = %q", ua)
	}
}

func TestListReportsRegistryErrors(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"status":   func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) },
		"not json": func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "<html>") },
		"endless": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{"servers":[],"metadata":{"nextCursor":"c%s"}}`, r.URL.Query().Get("cursor"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			if _, err := (&Client{Base: srv.URL}).List(context.Background()); err == nil {
				t.Fatal("want an error")
			}
		})
	}
	if _, err := (&Client{Base: "http://127.0.0.1:1"}).List(context.Background()); err == nil {
		t.Fatal("want a connection error")
	}
	if _, err := (&Client{Base: "::bad"}).List(context.Background()); err == nil {
		t.Fatal("want a URL error")
	}
}

func TestListStopsWhenTheCursorRepeats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"servers":[{"name":"x/y","remotes":[{"type":"sse","url":"http://127.0.0.1:9/sse"}]}],"metadata":{"nextCursor":"same"}}`)
	}))
	defer srv.Close()
	got, err := (&Client{Base: srv.URL}).List(context.Background())
	if err != nil || len(got.Remotes) != 2 {
		t.Fatalf("got %d remotes, err %v; want the page read twice then stop", len(got.Remotes), err)
	}
}

func TestAddEntryIgnoresMalformedEntries(t *testing.T) {
	var l Listing
	addEntry(&l, []byte(`"not an object"`))
	addEntry(&l, []byte(`{"server":{"name":""}}`))
	addEntry(&l, []byte(`{"name":"x/y","status":"deprecated","remotes":[{"type":"sse","url":"https://x/sse"}]}`))
	if len(l.Remotes)+len(l.Skipped) != 0 {
		t.Fatalf("got %+v", l)
	}
}
