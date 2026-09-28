// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package limiter spaces out the checks the job makes against one host.
//
// passmcp already throttles the requests inside one check (--rps). This is
// the other half: a hosting provider that serves many listed servers from
// one hostname sees the checks against them start no closer together than
// the interval, however many workers the job runs.
package limiter

import (
	"context"
	"sync"
	"time"
)

// PerHost hands out start times, one host at a time.
type PerHost struct {
	Interval time.Duration

	mu   sync.Mutex
	next map[string]time.Time
	now  func() time.Time
}

// New returns a limiter that starts at most one check per host per
// interval.
func New(interval time.Duration) *PerHost {
	return &PerHost{Interval: interval, next: map[string]time.Time{}, now: time.Now}
}

// Wait blocks until a check against host may start, or ctx ends.
func (l *PerHost) Wait(ctx context.Context, host string) error {
	l.mu.Lock()
	now := l.now()
	at := l.next[host]
	if at.Before(now) {
		at = now
	}
	l.next[host] = at.Add(l.Interval)
	l.mu.Unlock()

	d := at.Sub(now)
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
