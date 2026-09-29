#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: AGPL-3.0-only
#
# Boot the container to a working `make`: the linter at the version CI
# pins, then the build and the tests. devcontainer.json (plain JSON, so
# check-json can parse it) pins the base image by digest and sets
# GOTOOLCHAIN=auto, which fetches the exact floor go.mod names.
set -euo pipefail

# Pinned, never @latest: ci.yml runs this version.
GOLANGCI_LINT_VERSION=v2.13.2

go version
go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
make build test

echo "Ready. DEVELOPMENT.md maps every CI gate to its local command."
