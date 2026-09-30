# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: AGPL-3.0-only

.PHONY: all build test test-race coverage coverage-json vet lint format spdx-check readme-check integration trace trace-check trace-refresh help name-guard versions manual

VERSION ?= dev

# Every gate CI runs that needs no network, cheap ones first.
all: format vet lint spdx-check readme-check name-guard versions test

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Version=$(VERSION)" \
	  -o build/passmcp-registry ./cmd/passmcp-registry

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on -count=1 ./...

# The gate is 85% statement coverage in every package; ci.yml applies it.
coverage:
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# The shields.io endpoint document behind the README's coverage badge:
# statement coverage across the module, as CI measured it. The Manual
# workflow publishes it with GitHub Pages as coverage.json.
coverage-json: coverage
	@mkdir -p build
	go run ./scripts/coveragebadge -profile coverage.out > build/coverage.json
	@cat build/coverage.json

vet:
	go vet ./...

lint:
	golangci-lint run ./...

format:
	gofmt -l -w .

spdx-check:
	go run ./scripts/spdx_sweep.go

readme-check:
	scripts/readme-check.sh

# Every file that names the version names the newest CHANGELOG release:
# the family moves in lockstep, so passmcp and passmcp-reporting are pinned
# at that same release (scripts/verify-release-versions.sh).
versions:
	scripts/verify-release-versions.sh "v$$(grep -Eo '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | tr -d '#[] ')"

# The rendered manual, strictly: a broken link or nav entry fails. Needs
# the hash-locked requirements: pip install --require-hashes -r docs/requirements.txt
manual:
	mkdocs build --strict --site-dir public

# The real passmcp, named by PASSMCP_BIN, against fake MCP servers on
# 127.0.0.1. Nothing here leaves loopback.
integration:
	@test -n "$(PASSMCP_BIN)" || { echo "integration: set PASSMCP_BIN to a passmcp binary"; exit 2; }
	PASSMCP_BIN="$(PASSMCP_BIN)" go test -tags integration -count=1 -v -run TestTheRealPassmcp ./internal/job/

# The acceptance-criteria trace. This repository's user stories are
# filed in passmcp's issues and name it
# ("**Repository:** sebastienrousseau/passmcp-registry"), so the trace is passmcp's own
# tool, run at a pinned version: a closed story with an untested
# criterion fails here, and passmcp's stories are left to passmcp.
TRACE := go run satellion.com/passmcp/scripts/trace@v0.0.3 -repo sebastienrousseau/passmcp-registry

trace:
	$(TRACE)

trace-check:
	$(TRACE) -check -run -report-dir build/trace

trace-refresh:
	$(TRACE) -refresh

help:
	@printf '%s\n' "targets: all build test test-race coverage coverage-json vet lint format spdx-check readme-check" \
	  "         versions manual name-guard integration trace trace-check trace-refresh"

# A retired product name may not appear anywhere in the tree
# (scripts/name-guard.sh).
name-guard:
	./scripts/name-guard.sh
