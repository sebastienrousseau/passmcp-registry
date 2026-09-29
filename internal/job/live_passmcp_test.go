// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

//go:build integration

// This test runs the real passmcp binary named by PASSMCP_BIN against fake MCP
// servers on 127.0.0.1. It needs no network beyond loopback: nothing here
// contacts a real registry or a real server.
//
//	PASSMCP_BIN=$(go env GOPATH)/bin/passmcp go test -tags integration ./internal/job/

package job

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"satellion.com/passmcp-registry/internal/checker"
	"satellion.com/passmcp-registry/internal/disclosure"
	"satellion.com/passmcp-registry/internal/limiter"
	"satellion.com/passmcp-registry/internal/policy"
	"satellion.com/passmcp-registry/internal/registry"
	"satellion.com/passmcp-registry/internal/site"
)

// mcpServer is a small, well-behaved MCP server that counts its requests
// and records the User-Agent and every JSON-RPC method it was sent.
type mcpServer struct {
	mu      sync.Mutex
	hits    int
	agents  map[string]int
	methods map[string]int
	called  []string // tool names in every tools/call
	*httptest.Server
}

func newMCP() *mcpServer {
	m := &mcpServer{agents: map[string]int{}, methods: map[string]int{}}
	m.Server = httptest.NewServer(http.HandlerFunc(m.serve))
	return m
}

func (m *mcpServer) serve(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.hits++
	m.agents[r.Header.Get("User-Agent")]++
	m.mu.Unlock()
	if r.Method != http.MethodPost {
		http.Error(w, "no", http.StatusMethodNotAllowed)
		return
	}
	// A well-behaved server refuses a browser Origin that is not its own.
	if o := r.Header.Get("Origin"); o != "" && !strings.Contains(o, r.Host) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	_ = json.Unmarshal(body, &req)
	m.mu.Lock()
	m.methods[req.Method]++
	if req.Method == "tools/call" {
		m.called = append(m.called, req.Params.Name)
	}
	m.mu.Unlock()
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var result string
	switch req.Method {
	case "initialize":
		result = `{"protocolVersion":"2025-11-25","capabilities":{"tools":{}},"serverInfo":{"name":"fake","version":"1.0.0"}}`
	case "tools/list":
		result = `{"tools":[{"name":"search","description":"Search the public catalogue by keyword.","annotations":{"readOnlyHint":true},"inputSchema":{"type":"object","properties":{"q":{"type":"string","description":"keyword"}},"required":["q"]}}]}`
	case "ping":
		result = `{}`
	default:
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"method not found"}}`, req.ID)
		return
	}
	fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
}

// AC: REG-05, REG-01
func TestTheRealPassmcpContactsOnlyListedServersAndCallsNoTool(t *testing.T) {
	bin := os.Getenv("PASSMCP_BIN")
	if bin == "" {
		t.Skip("PASSMCP_BIN names the passmcp binary to run")
	}
	a, b, unlisted := newMCP(), newMCP(), newMCP()
	defer a.Close()
	defer b.Close()
	defer unlisted.Close()

	dir := t.TempDir()
	opt, _ := policy.LoadOptOut(filepath.Join(dir, "none"))
	sc := &checker.Passmcp{Bin: bin, UserAgent: "passmcp-registry/test (+https://example.invalid)", RPS: 0, Timeout: 10 * time.Second}
	ver, err := sc.Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Registry: fakeLister{l: registry.Listing{Remotes: []registry.Remote{
			{Name: "io.github.t/a", Version: "1", Type: "streamable-http", URL: a.URL + "/mcp"},
			{Name: "io.github.t/b", Version: "1", Type: "streamable-http", URL: b.URL + "/mcp"},
		}}},
		Checker: sc, PassmcpVersion: ver, OptOut: opt, Site: &site.Site{Dir: filepath.Join(dir, "site")},
		Queue: &disclosure.Queue{Dir: filepath.Join(dir, "private")}, Limiter: limiter.New(0), Workers: 2,
		Now: time.Now, Log: io.Discard,
	}
	sum, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Published != 2 {
		entries, _ := filepath.Glob(filepath.Join(dir, "private", "*.json"))
		for _, e := range entries {
			b, _ := os.ReadFile(e)
			t.Log(string(b))
		}
		t.Fatalf("summary %+v", sum)
	}
	if unlisted.hits != 0 {
		t.Fatalf("the unlisted server received %d requests", unlisted.hits)
	}
	for name, srv := range map[string]*mcpServer{"a": a, "b": b} {
		checkListedServer(t, name, srv)
	}
}

// checkListedServer fails unless a listed server was contacted with the
// published User-Agent and had none of its tools invoked.
func checkListedServer(t *testing.T, name string, srv *mcpServer) {
	t.Helper()
	if srv.hits == 0 {
		t.Errorf("listed server %s was not contacted", name)
	}
	// protocol.unknown_tool sends tools/call for a name the server does
	// not list, to see it refused; that invokes nothing. A call naming a
	// listed tool would be a real invocation.
	for _, called := range srv.called {
		if called == "search" {
			t.Errorf("server %s had its tool %q invoked", name, called)
		}
	}
	t.Logf("server %s: %d requests, tools/call names %v (none listed), user agents %v", name, srv.hits, srv.called, srv.agents)
	var ua []string
	for k := range srv.agents {
		ua = append(ua, k)
	}
	if !strings.Contains(strings.Join(ua, "|"), "passmcp-registry/test") {
		t.Errorf("server %s never saw the published User-Agent: %v", name, ua)
	}
}
