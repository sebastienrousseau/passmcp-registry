// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

package annual

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"satellion.com/passmcp-reporting/attestation"

	"satellion.com/passmcp-registry/internal/site"
)

func stmt(t *testing.T, endpoint string, verdicts []attestation.Verdict) []byte {
	t.Helper()
	target := attestation.Target{Transport: "http", Endpoint: endpoint}
	var c attestation.Counts
	for _, v := range verdicts {
		switch v.Status {
		case "pass":
			c.Pass++
		case "fail":
			c.Fail++
		}
	}
	s := &attestation.Statement{Type: attestation.StatementType, Subject: []attestation.Subject{attestation.SubjectFor(target)},
		PredicateType: attestation.PredicateType, Predicate: attestation.Evaluation{
			SubjectKind: attestation.SubjectKindDescriptor, Target: target,
			JudgedAgainst: attestation.Basis{Rubric: "1", CheckInventory: "120"},
			Instrument:    attestation.Instrument{Name: "passmcp", Version: "0.0.8", SchemaVersion: 1},
			RanAt:         time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Took: "1s", Verdicts: verdicts, Counts: c}}
	b, err := s.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func publish(t *testing.T, s *site.Site, name, version, url string, at time.Time, score float64, grade string, verdicts []attestation.Verdict) {
	t.Helper()
	r := site.Record{Name: name, Version: version, URL: url, Transport: "streamable-http", CheckedAt: at,
		MethodVersion: "1", PassmcpVersion: "0.0.8", Score: score, Grade: grade}
	if err := s.Publish(r, stmt(t, url, verdicts)); err != nil {
		t.Fatal(err)
	}
}

var (
	pass = attestation.Verdict{ID: "net.tcp", Phase: "net", Status: "pass"}
	fail = attestation.Verdict{ID: "protocol.ping", Phase: "protocol", Status: "fail", Severity: "minor"}
)

// AC: REG-06
func TestEveryFigureIsReproducibleFromTheSignedRecords(t *testing.T) {
	s := &site.Site{Dir: t.TempDir()}
	jan := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	jun := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	publish(t, s, "io.github.a/one", "1.0.0", "https://one.example/mcp", jan, 60, "D", []attestation.Verdict{pass, fail})
	publish(t, s, "io.github.a/one", "1.1.0", "https://one.example/mcp", jun, 90, "A", []attestation.Verdict{pass})
	publish(t, s, "io.github.b/two", "2.0.0", "https://two.example/mcp", jun, 70, "C", []attestation.Verdict{pass, fail})
	publish(t, s, "io.github.c/old", "1", "https://old.example/mcp", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), 10, "F", []attestation.Verdict{fail})
	// A record whose statement was edited after publication is rejected.
	publish(t, s, "io.github.d/tampered", "1", "https://tampered.example/mcp", jun, 99, "A", []attestation.Verdict{pass})
	_ = os.WriteFile(filepath.Join(s.Dir, site.RecordDir("io.github.d/tampered", "1"), site.StatementFile), []byte(`{"edited":true}`), 0o600)

	out := filepath.Join(t.TempDir(), "report")
	rep, err := Build(s.Dir, 2026, out)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Records != 3 || len(rep.Rejected) != 1 || !strings.Contains(rep.Rejected[0], "tampered") {
		t.Fatalf("records %d, rejected %v", rep.Records, rep.Rejected)
	}

	// Recompute from data.csv alone and compare with figures.json.
	rows := readCSV(t, filepath.Join(out, "data.csv"))
	latest := map[string][]string{}
	for _, row := range rows {
		if prev, ok := latest[row[0]]; !ok || row[3] > prev[3] {
			latest[row[0]] = row
		}
	}
	var scores []float64
	grades := map[string]int{}
	for _, row := range latest {
		sc, _ := strconv.ParseFloat(row[6], 64)
		scores = append(scores, sc)
		grades[row[7]]++
	}
	fig := map[string]Figure{}
	b, _ := os.ReadFile(filepath.Join(out, "figures.json"))
	var onDisk Report
	if err := json.Unmarshal(b, &onDisk); err != nil {
		t.Fatal(err)
	}
	for _, f := range onDisk.Figures {
		fig[f.ID] = f
	}
	if fig["records"].Value != float64(len(rows)) || fig["servers"].Value != float64(len(latest)) {
		t.Fatalf("records %v servers %v, recomputed %d %d", fig["records"].Value, fig["servers"].Value, len(rows), len(latest))
	}
	if fig["median_score"].Value != Median(scores) || fig["median_score"].Value != 80 {
		t.Fatalf("median %v, recomputed %v", fig["median_score"].Value, Median(scores))
	}
	if fig["grades"].Counts["A"] != grades["A"] || fig["grades"].Counts["C"] != grades["C"] || len(fig["grades"].Counts) != len(grades) {
		t.Fatalf("grades %v, recomputed %v", fig["grades"].Counts, grades)
	}
	// Server one's latest record is clean; server two fails protocol.ping.
	if fig["servers_with_a_failing_check"].Value != 1 || fig["failing_checks"].Counts["protocol.ping"] != 1 {
		t.Fatalf("failing figures %+v %+v", fig["servers_with_a_failing_check"], fig["failing_checks"])
	}
	// Every figure names records that are in the data.
	inData := map[string]bool{}
	for _, row := range rows {
		inData[row[8]] = true
	}
	for _, f := range onDisk.Figures {
		for _, d := range f.Records {
			if !inData[d] {
				t.Errorf("figure %s cites %s, which is not in data.csv", f.ID, d)
			}
		}
	}
	for _, name := range []string{"METHOD.md", "LICENSE-DATA.md"} {
		b, err := os.ReadFile(filepath.Join(out, name))
		if err != nil || (name == "LICENSE-DATA.md" && !strings.Contains(string(b), "CC-BY-4.0")) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows[1:]
}

func TestMedianAndEmptyYears(t *testing.T) {
	for _, c := range []struct {
		in   []float64
		want float64
	}{{nil, 0}, {[]float64{3}, 3}, {[]float64{4, 1, 3, 2}, 2.5}} {
		if got := Median(c.in); got != c.want {
			t.Errorf("Median(%v) = %v", c.in, got)
		}
	}
	rep, err := Build(t.TempDir(), 2030, filepath.Join(t.TempDir(), "out"))
	if err != nil || rep.Records != 0 {
		t.Fatalf("empty year: %+v %v", rep, err)
	}
	s := &site.Site{Dir: t.TempDir()}
	publish(t, s, "a/b", "1", "https://ab.example/mcp", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 1, "F", []attestation.Verdict{pass})
	_ = os.Remove(filepath.Join(s.Dir, site.RecordDir("a/b", "1"), site.StatementFile))
	rep, _ = Build(s.Dir, 2026, filepath.Join(t.TempDir(), "out"))
	if len(rep.Rejected) != 1 {
		t.Fatalf("a record with no statement must be rejected: %+v", rep)
	}
	names := []string{}
	for _, f := range figures(nil) {
		names = append(names, f.ID)
	}
	sort.Strings(names)
	if len(names) != 6 {
		t.Fatalf("figures %v", names)
	}
}
