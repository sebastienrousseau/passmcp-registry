<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

<p align="center">
  <img src="https://raw.githubusercontent.com/sebastienrousseau/passmcp/main/.github/logo.svg" alt="passmcp-registry logo" width="128" />
</p>

<h1 align="center">passmcp-registry</h1>

<p align="center">
  A signed public scorecard of the remote servers in the MCP Registry: each one checked read-only and without credentials by a pinned passmcp release, every result an offline-verifiable attestation, and anything that would expose a vulnerability withheld for its owner first.
</p>

<p align="center">
  <a href="https://github.com/sebastienrousseau/passmcp-registry/actions"><img src="https://img.shields.io/github/actions/workflow/status/sebastienrousseau/passmcp-registry/ci.yml?style=for-the-badge&logo=github" alt="Build Status" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-registry/tree/records"><img src="https://img.shields.io/badge/data-CC--BY--4.0-fc8d62?style=for-the-badge&logo=github" alt="Published data" /></a>
  <a href="https://pkg.go.dev/satellion.com/passmcp-registry"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=for-the-badge&logo=go&logoColor=white" alt="Go Reference" /></a>
  <a href="https://scorecard.dev/viewer/?uri=satellion.com/passmcp-registry"><img src="https://img.shields.io/ossf-scorecard/satellion.com/passmcp-registry?style=for-the-badge&label=OpenSSF%20Scorecard&logo=openssf" alt="OpenSSF Scorecard" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0--only-blue?style=for-the-badge" alt="License: AGPL-3.0-only" /></a>
  <a href="#requirements"><img src="https://img.shields.io/github/go-mod/go-version/sebastienrousseau/passmcp-registry?style=for-the-badge&logo=go&logoColor=white&label=Go" alt="Minimum Go version" /></a>
</p>

---

## Contents

**Getting started**

- [Install](#install) — `go install`, or build from source
- [Requirements](#requirements) — toolchain floor, platforms
- [Quick Start](#quick-start) — verify the published scorecard offline

**The passmcp-registry ecosystem**

- [The passmcp-registry ecosystem](#the-passmcp-registry-ecosystem) — passmcp, passmcp-reporting, passmcp-registry

**Library reference**

- [Capabilities at a glance](#capabilities-at-a-glance) — the current surface by theme
- [Ecosystem comparison](#ecosystem-comparison) — how this differs from a registry listing
- [Benchmarks](#benchmarks) — why a run's duration is set by its pacing
- [Features](#features) — package-level capability list
- [Configuration](#configuration) — core options
- [Examples](#examples) — runnable commands

**Operational**

- [When not to use passmcp-registry](#when-not-to-use-passmcp-registry) — limitations
- [Development](#development) — make targets, CI
- [Security](#security) — guarantees and disclosure
- [Documentation](#documentation) — all reference docs
- [Stability guarantees](#stability-guarantees) — SemVer axis, record format, method version
- [License](#license)

---

## Install

### As a Go command

```sh
go install satellion.com/passmcp-registry/cmd/passmcp-registry@latest
```

From source: `make build` writes `build/passmcp-registry`. A run also needs
the passmcp binary at the version the scorecard pins
(`go install satellion.com/passmcp/cmd/passmcp@v0.0.1`);
verifying and reporting do not.

---

## Requirements

- Go at the version `go.mod` names, to build.
- Linux, macOS or Windows. The tests that stand in a shell script for passmcp
  run on Linux and macOS only.
- For a run: passmcp v0.0.1. For signing, the `scorecard` workflow uses
  cosign keyless and age; neither is needed to verify a record's statement.

---

## Quick Start

```sh
git clone --branch records https://github.com/sebastienrousseau/passmcp-registry records
passmcp-registry verify --site records/site --require-bundles
```

`verify` recomputes every record's statement digest, parses the statement
with passmcp-reporting and confirms it is about the listed endpoint. Check a
signature with `cosign verify-blob`, as [docs/method.md](docs/method.md)
shows.

---

## The passmcp-registry ecosystem

passmcp-registry runs passmcp; it adds nothing to what passmcp measures. It
chooses which servers to check, what may be published and what goes to
an owner first.

| Component | Purpose | Use case |
| :--- | :--- | :--- |
| [passmcp](https://github.com/sebastienrousseau/passmcp) | The MCP server diagnostic | Run the checks and produce the report and attestation |
| [passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting) | Report and attestation formats | Parse and verify each statement before it is published |
| **passmcp-registry** | The public scorecard | List the registry, publish, withhold, and build the yearly report |

---

## Capabilities at a glance

| Area | Capability | Status |
| :--- | :--- | :--- |
| Enumeration | Remote servers of the latest active version of every registry entry | Implemented |
| Checking | net, discovery, handshake, protocol and catalog phases; no credentials, no tool invoked | Implemented |
| Publishing | One signed in-toto statement per server version, and an index | Implemented; signing in the workflow |
| Opt-out | By registry name or namespace, removed on the next run | Implemented |
| Disclosure | W1 to W4 withheld, private encrypted queue, drafted notices | Implemented; notices are sent by a person |
| Yearly report | CSV, figures with formulas and statement digests | Implemented; needs a year of runs |

---

## Ecosystem comparison

The registry lists servers; it does not say how they behave. A person can
run passmcp against one server. passmcp-registry runs the same pinned checks
against every listed server and publishes the evidence, not only a score.

| Approach | Every listed server | Evidence anyone can verify offline | Owner told before publication |
| :--- | :---: | :---: | :---: |
| **passmcp-registry** | yes | yes: a signed statement per record | yes: 90 days |
| `passmcp check` by hand | no | only if the operator attests and shares it | no |
| The registry listing | yes | no | not applicable |

---

## Benchmarks

No benchmark is published. A run's duration is set by its pacing, which is
deliberate, not by the code: the defaults below bound it.

| Scenario | Result | Environment |
| :--- | ---: | :--- |
| Checks started against one host | at most 1 per 10 s | `--per-host` default |
| Requests within one check | at most 1 per second | `--rps` default |

---

## Features

- `internal/registry`: the v0 servers API, both response shapes, bounded
  pages and bodies; entries that cannot be checked are recorded with a
  reason.
- `internal/checker`: runs passmcp with a fixed argument list and an empty
  configuration, then `passmcp attest`.
- `internal/policy`: the opt-out list and the withholding rules W1 to W4.
- `internal/site`: records, the index and the list of statements to sign.
- `internal/disclosure`: the private queue and drafted notices; the 90
  days run from the first finding.
- `internal/limiter`: the per-host spacing.
- `internal/annual`: the yearly report from verified records only.

---

## Configuration

`passmcp-registry run` takes `--registry`, `--passmcp`, `--site`, `--private`,
`--opt-out`, `--workers` (4), `--per-host` (10s), `--rps` (1) and
`--timeout` (20s). The private queue must not be inside the site;
`run` refuses it.

The `scorecard` workflow needs `vars.DISCLOSURE_AGE_RECIPIENT` and, once a
queue exists, `secrets.DISCLOSURE_AGE_IDENTITY`. It refuses to run without
the recipient.

---

## Examples

```sh
passmcp-registry verify --site records/site
passmcp-registry report --site records/site --year 2027 --out report
passmcp-registry version
```

A run contacts real servers. Start it only through the `scorecard`
workflow; for local work, see [DEVELOPMENT.md](DEVELOPMENT.md), which runs
it against fakes.

---

## When not to use passmcp-registry

- To evaluate your own server: run passmcp, with your credentials and every
  phase. The scorecard checks only what an anonymous client sees.
- As a security audit: the checks are read-only and never invoke a tool.
  A good score is not a clean bill of health.
- For servers that need a header or credentials to connect: they are
  listed as skipped, not scored.

---

## Development

```bash
make            # format, vet, lint, spdx-check, readme-check, test
make test-race
make coverage
PASSMCP_BIN="$(go env GOPATH)/bin/passmcp" make integration
```

CI runs tests on three operating systems, the race detector, an 85%
per-package coverage gate, golangci-lint with complexity ceilings,
govulncheck, and the integration test with the pinned passmcp against fake
servers on loopback. See [DEVELOPMENT.md](DEVELOPMENT.md) and
[AGENTS.md](AGENTS.md).

---

## Security

- Everything a server returns is untrusted and bounded before it reaches
  a record, a notice or a log.
- A result that trips a rule in [DISCLOSURE.md](DISCLOSURE.md) is never
  published. It is queued, encrypted, for a Maintainer to tell the owner.
- Owners can opt out: [OPT-OUT.md](OPT-OUT.md).

Report vulnerabilities according to [`SECURITY.md`](SECURITY.md).

---

## Documentation

- [docs/method.md](docs/method.md): what is checked, what is contacted,
  and how to verify a record.
- [DISCLOSURE.md](DISCLOSURE.md): the withholding rules and the 90 days.
- [OPT-OUT.md](OPT-OUT.md): how an owner opts out.
- [CHANGELOG.md](CHANGELOG.md) and [docs/releases/](docs/releases/).

---

## Stability guarantees

Pre-1.0, every release moves the patch digit. Two things are versioned
separately from the code:

- The record format and `index.json`, which change only with a CHANGELOG
  entry.
- The method version and the passmcp version, both carried in every
  record. The method version changes when the phases, the withholding
  rules or the record format do. Records made under different method or
  passmcp versions are not compared.

---

## License

The code is licensed under [AGPL-3.0-only](LICENSE). The published
scorecard, its records and the yearly reports are licensed under
[CC-BY-4.0](LICENSES/CC-BY-4.0.txt). [REUSE.toml](REUSE.toml) says which
is which.
