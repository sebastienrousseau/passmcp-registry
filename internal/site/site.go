// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

// Package site writes the published scorecard: one directory per server
// and version holding its record and its in-toto statement, and an index
// built from whatever records are on disk.
//
// The site is the record of every run, not only the last one, so the index
// is always rebuilt from the records rather than kept up to date by hand.
package site

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Record is what the scorecard publishes about one server version.
type Record struct {
	Name            string          `json:"name"`
	Version         string          `json:"version"`
	URL             string          `json:"url"`
	Transport       string          `json:"transport"`
	CheckedAt       time.Time       `json:"checked_at"`
	MethodVersion   string          `json:"method_version"`
	PassmcpVersion  string          `json:"passmcp_version"`
	Score           float64         `json:"score"`
	Grade           string          `json:"grade"`
	Counts          json.RawMessage `json:"counts,omitempty"`
	StatementSHA256 string          `json:"statement_sha256"`
	Attestation     string          `json:"attestation"`
	Bundle          string          `json:"bundle"`
}

// Unreachable is a listed server the run could not check.
type Unreachable struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	URL       string    `json:"url"`
	CheckedAt time.Time `json:"checked_at"`
	Error     string    `json:"error"`
}

// Skipped is a listed endpoint the method does not check.
type Skipped struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// Index is the site's entry point.
type Index struct {
	Licence        string        `json:"licence"`
	MethodVersion  string        `json:"method_version"`
	PassmcpVersion string        `json:"passmcp_version"`
	GeneratedAt    time.Time     `json:"generated_at"`
	Records        []Record      `json:"records"`
	Unreachable    []Unreachable `json:"unreachable"`
	Skipped        []Skipped     `json:"skipped"`
}

// Files a record directory holds, and the list the signing step reads.
const (
	RecordFile    = "record.json"
	StatementFile = "attestation.json"
	BundleFile    = "attestation.sigstore.json"
	ToSignFile    = ".to-sign"
	serversDir    = "servers"
)

// Site is a directory the scorecard publishes into.
type Site struct {
	Dir    string
	toSign []string
}

// Slug is a server name or version as one safe path segment.
func Slug(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		case r == '/':
			b.WriteString("~")
		default:
			b.WriteString("-")
		}
	}
	out := strings.Trim(b.String(), ".")
	if out == "" {
		return "-"
	}
	return out
}

// RecordDir is where a server version's record lives, relative to the site.
func RecordDir(name, version string) string {
	return filepath.Join(serversDir, Slug(name), Slug(version))
}

// Publish writes a record and its statement, replacing any earlier record
// for the same server version, and queues the statement for signing.
func (s *Site) Publish(r Record, statement []byte) error {
	rel := RecordDir(r.Name, r.Version)
	dir := filepath.Join(s.Dir, rel)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	sum := sha256.Sum256(statement)
	r.StatementSHA256 = hex.EncodeToString(sum[:])
	r.Attestation = filepath.ToSlash(filepath.Join(rel, StatementFile))
	r.Bundle = filepath.ToSlash(filepath.Join(rel, BundleFile))
	if err := os.WriteFile(filepath.Join(dir, StatementFile), statement, 0o600); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, RecordFile), r); err != nil {
		return err
	}
	s.toSign = append(s.toSign, r.Attestation)
	return nil
}

// Remove deletes the published record of one server version, if any. A
// result that has to go to the owner first replaces nothing in public.
func (s *Site) Remove(name, version string) error {
	return os.RemoveAll(filepath.Join(s.Dir, RecordDir(name, version)))
}

// Prune deletes every record whose server the predicate excludes, and
// returns the names removed.
func (s *Site) Prune(excludes func(name string) bool) ([]string, error) {
	recs, err := s.Records()
	if err != nil {
		return nil, err
	}
	var removed []string
	seen := map[string]bool{}
	for _, r := range recs {
		if !excludes(r.Name) || seen[r.Name] {
			continue
		}
		seen[r.Name] = true
		if err := os.RemoveAll(filepath.Join(s.Dir, serversDir, Slug(r.Name))); err != nil {
			return removed, err
		}
		removed = append(removed, r.Name)
	}
	return removed, nil
}

// Records reads every record on disk, sorted by name, version and time.
func (s *Site) Records() ([]Record, error) {
	// Walk through an os.Root so a symlink planted in the site cannot lead
	// the walk, or a read, outside it.
	root, err := os.OpenRoot(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	out, err := readRecords(root.FS())
	sortRecords(out)
	return out, err
}

// readRecords collects every record file under the servers directory of
// fsys; a site with no servers directory has no records.
func readRecords(fsys fs.FS) ([]Record, error) {
	var out []Record
	err := fs.WalkDir(fsys, serversDir, func(path string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) && path == serversDir {
			return fs.SkipAll
		}
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != RecordFile {
			return nil
		}
		r, err := readRecord(fsys, path)
		if err == nil {
			out = append(out, r)
		}
		return err
	})
	return out, err
}

// readRecord decodes one record file.
func readRecord(fsys fs.FS, path string) (Record, error) {
	var r Record
	b, err := fs.ReadFile(fsys, path)
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return r, fmt.Errorf("%s: %w", path, err)
	}
	return r, nil
}

// sortRecords orders records by name, then version, then check time.
func sortRecords(out []Record) {
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.CheckedAt.Before(b.CheckedAt)
	})
}

// WriteIndex rebuilds index.json from the records on disk, and writes the
// list of statements this run published for the signing step.
func (s *Site) WriteIndex(ix Index) error {
	recs, err := s.Records()
	if err != nil {
		return err
	}
	ix.Records = recs
	ix.Licence = "CC-BY-4.0"
	if ix.Unreachable == nil {
		ix.Unreachable = []Unreachable{}
	}
	if ix.Skipped == nil {
		ix.Skipped = []Skipped{}
	}
	if err := os.MkdirAll(s.Dir, 0o750); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(s.Dir, "index.json"), ix); err != nil {
		return err
	}
	list := strings.Join(s.toSign, "\n")
	if list != "" {
		list += "\n"
	}
	return os.WriteFile(filepath.Join(s.Dir, ToSignFile), []byte(list), 0o600)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}
