// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package checker runs passmcp against one listed endpoint and returns its
// report and the in-toto statement made from it.
//
// It runs the passmcp program rather than linking passmcp's engine, so every
// safety property passmcp has holds here too, and the scorecard's method is
// exactly a pinned passmcp release plus the flags below.
package checker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Phases are the only phases the scorecard runs: connectivity, discovery,
// handshake, protocol conformance and the catalogue. None of them invokes
// a tool. Execution, performance and resilience call tools or hold
// sessions open, and the auth phase presents a made-up credential; none of
// them runs against someone else's server without the owner's say-so.
var Phases = []string{"net", "discovery", "handshake", "protocol", "catalog"}

// Result is one check's output.
type Result struct {
	Report    []byte // passmcp's JSON report
	Statement []byte // the in-toto statement passmcp attest made from it
	Exit      int    // 0 all clear, 2 a failing check
}

// Checker checks one endpoint.
type Checker interface {
	Check(ctx context.Context, endpoint string) (Result, error)
}

// Passmcp runs the passmcp program.
type Passmcp struct {
	Bin       string        // path of the pinned passmcp binary
	UserAgent string        // sent on every request passmcp's client makes
	RPS       float64       // passmcp's own request throttle within a check
	Timeout   time.Duration // per-call timeout inside passmcp
}

// maxStderr bounds the passmcp error text kept for an unreachable server.
const maxStderr = 400

// Args is the passmcp check command line for an endpoint, exposed so the
// method page and the tests state the same thing.
func (s *Passmcp) Args(endpoint string) []string {
	args := []string{"check", endpoint,
		"--phases", strings.Join(Phases, ","),
		"--auth", "none",
		"--output", "json",
		"--no-color",
		"--log-level", "error",
		"--rps", fmt.Sprint(s.RPS),
		"--timeout", s.Timeout.String(),
	}
	if s.UserAgent != "" {
		args = append(args, "--header", "User-Agent: "+s.UserAgent)
	}
	return args
}

// Check runs passmcp check, then passmcp attest on its report.
func (s *Passmcp) Check(ctx context.Context, endpoint string) (Result, error) {
	cfg, cleanup, err := emptyConfig()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	report, stderr, code, err := s.run(ctx, cfg, nil, s.Args(endpoint)...)
	if err != nil {
		return Result{}, err
	}
	// 0 is all clear and 2 a failing check; both carry a report. Anything
	// else is passmcp saying it could not produce one.
	if (code != 0 && code != 2) || len(bytes.TrimSpace(report)) == 0 {
		return Result{Exit: code}, fmt.Errorf("passmcp exited %d: %s", code, tail(stderr))
	}
	stmt, stderr, acode, err := s.run(ctx, cfg, report, "attest", "-")
	if err != nil {
		return Result{}, err
	}
	if acode != 0 {
		return Result{Report: report, Exit: code}, fmt.Errorf("passmcp attest exited %d: %s", acode, tail(stderr))
	}
	return Result{Report: report, Statement: stmt, Exit: code}, nil
}

// Version is what the pinned binary reports as its version.
func (s *Passmcp) Version(ctx context.Context) (string, error) {
	out, stderr, code, err := s.run(ctx, "", nil, "version")
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("passmcp version exited %d: %s", code, tail(stderr))
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "passmcp")), nil
}

func (s *Passmcp) run(ctx context.Context, cfg string, stdin []byte, args ...string) (stdout, stderr []byte, code int, err error) {
	cmd := exec.CommandContext(ctx, s.Bin, args...) // #nosec G204 -- the operator pins the binary; arguments are ours
	// passmcp reads the operator's own configuration by default. Pointing it
	// at an empty file means no profile anyone wrote for their own use can
	// add a credential or switch on mutations here.
	cmd.Env = append(os.Environ(), "PASSMCP_CONFIG="+cfg, "NO_COLOR=1")
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	runErr := cmd.Run()
	var exit *exec.ExitError
	switch {
	case runErr == nil:
		return out.Bytes(), errb.Bytes(), 0, nil
	case errors.As(runErr, &exit):
		return out.Bytes(), errb.Bytes(), exit.ExitCode(), nil
	default:
		return nil, nil, -1, fmt.Errorf("running passmcp: %w", runErr)
	}
}

func emptyConfig() (string, func(), error) {
	dir, err := os.MkdirTemp("", "passmcp-registry-config-")
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	return path, func() { _ = os.RemoveAll(dir) }, nil
}

func tail(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > maxStderr {
		s = "…" + s[len(s)-maxStderr:]
	}
	return s
}
