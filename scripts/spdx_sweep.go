// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: AGPL-3.0-only

//go:build ignore

// spdx_sweep fails when a source file lacks a machine-readable license
// header, so REUSE compliance is a CI gate rather than a habit.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var skipDirs = map[string]bool{".git": true, "build": true, "dist": true, "vendor": true, "node_modules": true, "site": true}

func main() {
	missing, err := sweep(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(missing) > 0 {
		// REUSE-IgnoreStart
		fmt.Fprintln(os.Stderr, "files without an SPDX-License-Identifier in their first 5 lines:")
		// REUSE-IgnoreEnd
		for _, m := range missing {
			fmt.Fprintln(os.Stderr, "  "+m)
		}
		os.Exit(1)
	}
	fmt.Println("spdx-check: every source file carries a license header")
}

// sweep walks root, skipping skipDirs, and returns every wanted file that
// lacks a header.
func sweep(root string) ([]string, error) {
	var missing []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if wants(path) && !hasHeader(path) {
			missing = append(missing, path)
		}
		return nil
	})
	return missing, err
}

func wants(path string) bool {
	base := filepath.Base(path)
	switch base {
	case "go.mod", "go.sum", "LICENSE", "flake.lock", "CODEOWNERS", "CITATION.cff", ".gitignore", ".gitattributes", ".DS_Store":
		return false
	}
	if strings.HasPrefix(base, ".") && !checkedDotfile(base) {
		return false
	}
	switch filepath.Ext(path) {
	case ".go", ".sh", ".yml", ".yaml", ".md", ".toml", ".jsonc", ".nix", ".svg":
		return true
	}
	return base == "Makefile" || base == "GNUmakefile" || base == "Dockerfile" || strings.HasPrefix(base, ".editorconfig")
}

// checkedDotfile reports whether a dotfile is configuration that carries a
// header of its own.
func checkedDotfile(base string) bool {
	for _, p := range []string{".golangci", ".goreleaser", ".pre-commit", ".editorconfig", ".markdownlint", ".gitleaks"} {
		if strings.HasPrefix(base, p) {
			return true
		}
	}
	return false
}

func hasHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for i := 0; i < 5 && sc.Scan(); i++ {
		// REUSE-IgnoreStart
		if strings.Contains(sc.Text(), "SPDX-License-Identifier:") {
			return true
		}
		// REUSE-IgnoreEnd
	}
	return false
}
