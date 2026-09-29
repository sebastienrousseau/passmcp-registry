<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Architecture

How one scorecard run turns the MCP Registry's listing into published,
verifiable records, and where each decision lives. For what is checked and
why, read the [method](method.md); for the decisions behind the shape, the
[decision records](adr/README.md).

## One run

`passmcp-registry run` builds a [`job.Config`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/job/job.go)
and calls `job.Run`, which does five things in order:

1. **List.** [`internal/registry`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/registry/registry.go) pages
   through the registry's v0 servers API, keeps the remote endpoints of the
   latest active version of every entry, and records the ones it cannot
   check (templated URLs, required headers, credentials in the URL) as
   skipped, with a reason. Pages and bodies are bounded.
2. **Select.** Opted-out names and namespaces
   ([`internal/policy`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/policy/policy.go), read from
   `opt-out.txt`) are dropped. One endpoint is kept per server version,
   preferring Streamable HTTP, so a record is about one endpoint.
3. **Check.** A small worker pool runs
   [`internal/checker`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/checker/checker.go) for each target.
   [`internal/limiter`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/limiter/limiter.go) spaces checks on one
   host at least `--per-host` apart. The checker runs the pinned passmcp
   program with the phases in `checker.Phases`, `--auth none` and an empty
   configuration file, then `passmcp attest` on its report.
4. **Assess and file.** Each outcome is one of three:
   - *unreachable*: the check failed, or the statement passmcp produced does
     not verify with passmcp-reporting or is not about the listed endpoint.
     It is listed in the index with the error.
   - *withheld*: `policy.Withhold` found a rule from
     [DISCLOSURE.md](https://github.com/sebastienrousseau/passmcp-registry/blob/main/DISCLOSURE.md) in the report.
     [`internal/disclosure`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/disclosure/disclosure.go) writes a
     private queue entry and a drafted notice, keeping the first date seen,
     and any published record of that version is removed.
   - *published*: [`internal/site`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/site/site.go) writes the
     record and the statement under `servers/<name>/<version>/`, and names
     the statement in `.to-sign`.
5. **Prune and index.** Records of opted-out servers are deleted, and
   `index.json` is rebuilt from the records on disk, never edited in place.

Stdout carries the run's one-line summary; every diagnostic goes to stderr.

## Outside the binary

The binary never signs, never encrypts and never talks to GitHub. The
[Registry Run workflow](https://github.com/sebastienrousseau/passmcp-registry/blob/main/.github/workflows/registry-run.yml), started by
hand, does that around it:

- fetches the `records` branch, which holds the published site and the
  encrypted queue, and decrypts the previous queue with the maintainers' age
  identity so the ninety days keep running from the first finding;
- runs `passmcp-registry run`, then seals the queue again with age;
- signs every statement named in `.to-sign` with cosign keyless, verifies
  every signature, and runs `passmcp-registry verify --require-bundles`;
- pushes the `records` branch and starts the
  [Manual workflow](https://github.com/sebastienrousseau/passmcp-registry/blob/main/.github/workflows/manual.yml), the one workflow that
  deploys GitHub Pages: the manual at the root, `coverage.json` beside it,
  and the site under `/scorecard/`.

## Reading the records

`passmcp-registry verify` reads every record, recomputes its statement's
SHA-256, parses the statement with passmcp-reporting and checks it covers
the listed endpoint. `passmcp-registry report` builds the yearly report in
[`internal/annual`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/annual/annual.go) from records that verify
the same way: the CSV, and every figure with its formula and the statement
digests behind it.

## Packages

| Package | Owns |
|---|---|
| `cmd/passmcp-registry` | Flags, the `run`, `verify`, `report` and `version` commands, exit codes |
| `internal/registry` | The registry client: paging, both response shapes, bounds |
| `internal/policy` | The opt-out list and the withholding rules W1 to W4 |
| `internal/checker` | Running passmcp with a fixed argument list and an empty configuration |
| `internal/limiter` | Per-host spacing between checks |
| `internal/job` | One run, and `MethodVersion` |
| `internal/site` | Records, the index, pruning, the list of statements to sign |
| `internal/disclosure` | The private queue and the drafted notices |
| `internal/annual` | The yearly report |
| `scripts/coveragebadge` | The coverage badge's shields.io endpoint document |

The one module dependency is
[passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting),
for parsing and checking statements.
