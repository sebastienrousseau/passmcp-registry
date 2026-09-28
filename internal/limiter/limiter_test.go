// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package limiter

import (
	"context"
	"sync"
	"testing"
	"time"
)

// AC: REG-05
func TestChecksAgainstOneHostAreSpacedByTheInterval(t *testing.T) {
	l := New(40 * time.Millisecond)
	var mu sync.Mutex
	var starts []time.Time
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := l.Wait(context.Background(), "host.example"); err != nil {
				t.Error(err)
			}
			mu.Lock()
			starts = append(starts, time.Now())
			mu.Unlock()
		}()
	}
	wg.Wait()
	first, last := starts[0], starts[0]
	for _, s := range starts {
		if s.Before(first) {
			first = s
		}
		if s.After(last) {
			last = s
		}
	}
	if gap := last.Sub(first); gap < 75*time.Millisecond {
		t.Fatalf("three checks on one host spread over %v; want at least two intervals", gap)
	}
}

func TestDifferentHostsDoNotWaitForEachOther(t *testing.T) {
	l := New(time.Hour)
	start := time.Now()
	for _, h := range []string{"a.example", "b.example", "c.example"} {
		if err := l.Wait(context.Background(), h); err != nil {
			t.Fatal(err)
		}
	}
	if time.Since(start) > time.Second {
		t.Fatal("distinct hosts were serialised")
	}
}

func TestWaitGivesUpWhenTheContextEnds(t *testing.T) {
	l := New(time.Hour)
	_ = l.Wait(context.Background(), "h")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := l.Wait(ctx, "h"); err == nil {
		t.Fatal("want the context's error")
	}
	if err := l.Wait(ctx, "fresh"); err == nil {
		t.Fatal("want the context's error even with no wait")
	}
}
