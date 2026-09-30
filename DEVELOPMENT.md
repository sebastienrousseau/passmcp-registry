<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Development

The single entry point for working on passmcp-registry: toolchain, how to
reproduce every CI gate locally, and how a release is cut. If a gate fails
in CI and you cannot reproduce it from this file, that is a bug in this
file.

## Requirements

| Tool | Version | Why |
|---|---|---|
| Go | 1.26.8 or later, the `go` directive in `go.mod` | `GOTOOLCHAIN=auto` downloads it; CI tests on that version and on latest stable |
| make | any | Task runner for everything below |
| passmcp | the version in `CHANGELOG.md` | Only for `make integration` and a run: `go install satellion.com/passmcp/cmd/passmcp@v0.0.4` |

Optional, only for the gate that uses it: `golangci-lint` v2 (`make lint`),
Python 3.12 with the hash-locked `docs/requirements.txt` (`make manual`),
`markdownlint-cli2`, `codespell` and `lychee` (the Docs Lint workflow and
`pre-commit`).

The [dev container](.devcontainer/devcontainer.json) has Go at the
`go.mod` floor and golangci-lint at the version CI pins, and runs
`make build test` when it is created.

## Reproducing every CI gate

| CI job | Local command |
|---|---|
| Test (three OSes × two Go versions) | `make test` |
| Race & Shuffled Tests | `make test-race` |
| Coverage Gate (85% per package but `cmd/passmcp-registry`) | `make coverage` |
| Lint, with the complexity ceilings | `gofmt -l .` and `make lint` |
| Vulnerability Scan | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |
| Repository Checks | `make trace-check spdx-check name-guard readme-check versions` |
| Real passmcp against fake servers | `PASSMCP_BIN=$(command -v passmcp) make integration` |
| Licence Headers | `make spdx-check` |
| Markdown & Spelling | `make readme-check`, `markdownlint-cli2 '**/*.md'` and `codespell` |
| Link Check | `lychee --offline --include-fragments '**/*.md'` |
| DCO check | `git log --format=%B origin/main.. \| grep Signed-off-by` |
| Manual (build, and on `main` deploy with `coverage.json`) | `make manual` after `pip install --require-hashes -r docs/requirements.txt`, and `make coverage-json` |
| CodeQL | not reproducible locally; runs on push, pull request and weekly |
| OpenSSF Scorecard | not reproducible locally; runs on push to `main` and weekly, and publishes to [scorecard.dev](https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-registry) |

`make` with no target runs the gates that need no network, cheap ones
first. `make trace-check` fetches passmcp's trace tool at the pinned
release.

The integration test runs the real passmcp binary against fake MCP servers
on 127.0.0.1. It proves that an unlisted server receives nothing and that
no listed tool is called.

## Complexity

`make lint` enforces the portfolio's per-function ceilings through
golangci-lint: cyclomatic complexity 10 (`gocyclo`), cognitive complexity
15 (`gocognit`) and 60 lines (`funlen`), in tests as in the rest of the
code. Every function is under them, so there is no baseline of existing
offenders; a new one fails the Lint job. `scripts/spdx_sweep.go` is
`//go:build ignore`, so the linter does not see it; check it with
`go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -over 10 scripts` and
`go run github.com/uudashr/gocognit/cmd/gocognit@latest -over 15 scripts`.
Halstead difficulty has no golangci-lint analyser and is not gated.

## Coverage

The gate is 85% statement coverage in every package with statements but
`cmd/passmcp-registry`, whose `main` only wires signals and the process
exit around the tested `run`. `make coverage` writes `coverage.out`;
ci.yml's Coverage Gate checks each package against the threshold.

The README's coverage badge is a separate, whole-module number:
`make coverage-json` runs the tests with a cover profile and
`scripts/coveragebadge` turns it into a
[shields.io endpoint document](https://shields.io/badges/endpoint-badge),
`build/coverage.json`. It is brightgreen from 90%, green from 85%, yellow
from 70% and red below, truncated rather than rounded so the badge never
shows the gate as met early. The Manual workflow publishes it on every
push to `main` at
<https://sebastienrousseau.com/passmcp-registry/coverage.json>, which the
badge reads.

## A run, locally, against fakes only

```sh
make build
./build/passmcp-registry run --registry http://127.0.0.1:<fake-registry-port> \
  --passmcp "$(command -v passmcp)" --site /tmp/site --private /tmp/private
./build/passmcp-registry verify --site /tmp/site
```

Never point `--registry` at the real registry from your machine. Live runs
happen only in the Registry Run workflow
(`.github/workflows/registry-run.yml`), started by hand.

`make demo` is such a run, recorded: it serves `.github/demo/registry` as a
fake registry with `python3 -m http.server`, lists two of passmcp's example
servers on loopback, and renders the README demo, `.github/demo.gif`, from
`.github/demo.tape` with [VHS](https://github.com/charmbracelet/vhs) (`vhs`,
`ttyd` and `ffmpeg` on `PATH`). Regenerate it whenever what the run prints
changes; its site and queue stay in `build/demo/work`. Leave 90 seconds
between renders: the servers a render starts stop themselves then, and hold
their ports until they do.

## GitHub Pages

One workflow deploys Pages, the Manual workflow
([ADR 0005](docs/adr/0005-one-pages-deployer.md)): the manual at the root,
`coverage.json` beside it, and the `records` branch's `site/` under
`/scorecard/`. The Registry Run workflow starts it after pushing records.

## Test layout

Each package's tests sit beside it. The registry, the job and the command
use a fake registry and fake MCP servers on loopback;
`internal/checker` and `cmd/passmcp-registry` stand a shell script in for
passmcp and are `//go:build !windows`, so on Windows they report no test
files. `internal/job/live_passmcp_test.go` is the integration test, behind
the `integration` build tag. `scripts/coveragebadge/main_test.go` covers the
badge document.

## Release model

The family moves in lockstep: every repository is released at passmcp's
version, after passmcp and passmcp-reporting, on a `feat/vX.Y.Z` branch.

1. Move the `## [Unreleased]` entries under a `## [X.Y.Z] — date` heading
   in `CHANGELOG.md`, and write `docs/releases/vX.Y.Z.md` with a
   `## Highlights ⭐️` section of two to four bullets.
2. Update the version in the README's install lines and ecosystem
   sentence, in `CITATION.cff` (`version` and `date-released`), in
   `PASSMCP_VERSION` in `ci.yml` and `registry-run.yml`, and in the
   Makefile's `TRACE`; move `go.mod` to passmcp-reporting's `vX.Y.Z`. A new
   passmcp version is a method change: say so in the CHANGELOG.
3. `make versions` (it runs `scripts/verify-release-versions.sh` for the
   newest CHANGELOG release), and optionally the Release workflow's dry run.
4. Push a signed annotated tag `vX.Y.Z` with the message
   `passmcp-registry vX.Y.Z`. The Release workflow verifies the versions
   again and publishes the release page: the highlights, the generated
   "What's Changed", the checksums section and the full-changelog link.
   There are no binaries: `go install` fetches the module from the proxy.
5. Read the tag and the release page back before calling it done.

## Conventions

- Stdout carries a command's result; every diagnostic goes to stderr.
- Every exported identifier is documented.
- Anything the registry or a server returns is untrusted input, and is
  bounded before it reaches a record, a notice or a log.
