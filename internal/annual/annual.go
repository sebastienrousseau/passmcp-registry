// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package annual builds the yearly report from the scorecard's signed
// records: the raw data, every figure with the records it was computed
// from, and the method, all published under CC-BY-4.0.
//
// A figure that cannot be traced to the records it came from is not
// published. Each one in figures.json names its formula and the statement
// digests of every record behind it, so anyone holding the records can
// recompute it.
package annual

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"satellion.com/passmcp-reporting/attestation"

	"satellion.com/passmcp-registry/internal/site"
)

// Figure is one published number and where it came from.
type Figure struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Value   float64        `json:"value,omitempty"`
	Counts  map[string]int `json:"counts,omitempty"`
	Formula string         `json:"formula"`
	Records []string       `json:"records"`
}

// Report is the result of a build.
type Report struct {
	Year     int      `json:"year"`
	Records  int      `json:"records"`
	Rejected []string `json:"rejected"`
	Figures  []Figure `json:"figures"`
}

type verified struct {
	rec  site.Record
	stmt *attestation.Statement
}

// Build reads the site's records for one year and writes the report into
// out.
func Build(siteDir string, year int, out string) (Report, error) {
	rep := Report{Year: year, Rejected: []string{}}
	recs, err := (&site.Site{Dir: siteDir}).Records()
	if err != nil {
		return rep, err
	}
	var ok []verified
	for _, r := range recs {
		if r.CheckedAt.UTC().Year() != year {
			continue
		}
		v, err := verify(siteDir, r)
		if err != nil {
			rep.Rejected = append(rep.Rejected, fmt.Sprintf("%s@%s: %v", r.Name, r.Version, err))
			continue
		}
		ok = append(ok, v)
	}
	rep.Records = len(ok)
	rep.Figures = figures(ok)
	if err := os.MkdirAll(out, 0o750); err != nil {
		return rep, err
	}
	if err := writeCSV(filepath.Join(out, "data.csv"), ok); err != nil {
		return rep, err
	}
	if err := writeJSON(filepath.Join(out, "figures.json"), rep); err != nil {
		return rep, err
	}
	if err := os.WriteFile(filepath.Join(out, "METHOD.md"), []byte(method(year)), 0o600); err != nil {
		return rep, err
	}
	return rep, os.WriteFile(filepath.Join(out, "LICENSE-DATA.md"), []byte(licence), 0o600)
}

// verify checks that a record's statement is the one it names and that
// passmcp-reporting accepts it.
func verify(siteDir string, r site.Record) (verified, error) {
	b, err := os.ReadFile(filepath.Join(siteDir, filepath.FromSlash(r.Attestation))) // #nosec G304 -- under the site directory
	if err != nil {
		return verified{}, err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != r.StatementSHA256 {
		return verified{}, fmt.Errorf("the statement's digest is not the one the record names")
	}
	st, err := attestation.Parse(b)
	if err != nil {
		return verified{}, err
	}
	if !st.Covers("http", r.URL) {
		return verified{}, fmt.Errorf("the statement is not about %s", r.URL)
	}
	return verified{rec: r, stmt: st}, nil
}

// latest keeps each server's most recent verified record in the year.
func latest(vs []verified) []verified {
	by := map[string]verified{}
	for _, v := range vs {
		if prev, ok := by[v.rec.Name]; !ok || v.rec.CheckedAt.After(prev.rec.CheckedAt) {
			by[v.rec.Name] = v
		}
	}
	out := make([]verified, 0, len(by))
	for _, v := range by {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].rec.Name < out[j].rec.Name })
	return out
}

func figures(all []verified) []Figure {
	servers := latest(all)
	digests := func(vs []verified) []string {
		d := make([]string, 0, len(vs))
		for _, v := range vs {
			d = append(d, v.rec.StatementSHA256)
		}
		return d
	}
	var scores []float64
	grades := map[string]int{}
	failingChecks := map[string]int{}
	var withFail []verified
	for _, v := range servers {
		scores = append(scores, v.rec.Score)
		grades[v.rec.Grade]++
		failed := false
		for _, verdict := range v.stmt.Predicate.Verdicts {
			if verdict.Status == "fail" {
				failingChecks[verdict.ID]++
				failed = true
			}
		}
		if failed {
			withFail = append(withFail, v)
		}
	}
	return []Figure{
		{ID: "records", Title: "Verified records in the year", Value: float64(len(all)),
			Formula: "count of records whose statement matches its digest and verifies", Records: digests(all)},
		{ID: "servers", Title: "Servers checked", Value: float64(len(servers)),
			Formula: "count of distinct registry names, latest verified record each", Records: digests(servers)},
		{ID: "median_score", Title: "Median score", Value: Median(scores),
			Formula: "median of score over each server's latest record", Records: digests(servers)},
		{ID: "grades", Title: "Servers by grade", Counts: grades,
			Formula: "count of each server's latest record by grade", Records: digests(servers)},
		{ID: "servers_with_a_failing_check", Title: "Servers with at least one failing check", Value: float64(len(withFail)),
			Formula: "count of servers whose latest statement has a verdict with status fail", Records: digests(withFail)},
		{ID: "failing_checks", Title: "Servers failing each check", Counts: failingChecks,
			Formula: "per check id, count of servers whose latest statement has that verdict failing", Records: digests(servers)},
	}
}

// Median is the middle value, or the mean of the two middle values.
func Median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}

func writeCSV(path string, vs []verified) error {
	f, err := os.Create(path) // #nosec G304 -- the operator names the output directory
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"name", "version", "url", "checked_at", "method_version", "passmcp_version", "score", "grade", "statement_sha256"})
	for _, v := range vs {
		r := v.rec
		_ = w.Write([]string{r.Name, r.Version, r.URL, r.CheckedAt.UTC().Format("2006-01-02T15:04:05Z"), r.MethodVersion,
			r.PassmcpVersion, strconv.FormatFloat(r.Score, 'f', -1, 64), r.Grade, r.StatementSHA256})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func method(year int) string {
	return fmt.Sprintf(`# Method, %d

Every figure in figures.json was computed from the records in data.csv,
each of which is a statement the scorecard published and signed. A record
counts only if its statement's SHA-256 matches the digest the record names
and passmcp-reporting's verifier accepts it; records that fail are listed
under "rejected" and left out.

Each server was checked read-only and without credentials, with the passmcp
phases net, discovery, handshake, protocol and catalog, and no tool was
called. Results that exposed a vulnerability were withheld from
publication under DISCLOSURE.md, so they are not in this data either.

Server-level figures use each server's latest verified record in the year.
Each figure names its formula and the statement digests it was computed
from. To reproduce one, fetch those statements from the scorecard, check
each against its sigstore bundle, and apply the formula.
`, year)
}

const licence = `# Licence

The data in this directory (data.csv, figures.json and METHOD.md) is
published under the Creative Commons Attribution 4.0 International licence,
CC-BY-4.0: https://creativecommons.org/licenses/by/4.0/

Attribute it as: "passmcp-registry scorecard, https://github.com/sebastienrousseau/passmcp-registry".
`
