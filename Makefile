# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: AGPL-3.0-only

.PHONY: all build test test-race coverage vet lint format spdx-check readme-check integration trace trace-check trace-refresh help name-guard

VERSION ?= dev

# Every gate CI runs that needs no network, cheap ones first.
all: format vet lint spdx-check readme-check test

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
TRACE := go run satellion.com/passmcp/scripts/trace@v0.0.1 -repo sebastienrousseau/passmcp-registry

trace:
	$(TRACE)

trace-check:
	$(TRACE) -check -run -report-dir build/trace

trace-refresh:
	$(TRACE) -refresh

help:
	@printf '%s\n' "targets: all build test test-race coverage vet lint format spdx-check readme-check integration trace trace-check trace-refresh"

# The project was renamed to passmcp: the old name may appear only in the
# provenance line (scripts/name-guard.sh).
name-guard:
	./scripts/name-guard.sh
