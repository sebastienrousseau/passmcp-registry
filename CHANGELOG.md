<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Changelog

All notable changes are documented here, in the format of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions move by
0.0.1 a release.

## [Unreleased]

### Added

- **A README demo**, rendered from `.github/demo.tape` by `make demo`: a
  scorecard run against a local registry listing two example servers: one
  record published, one withheld for its owner, and the published record
  verified.

## [0.0.3] — 2026-09-30

### Added

- **Fuzz targets for the two parsers that read untrusted input.**
  `FuzzListingPage` decodes arbitrary registry pages and checks that only
  an endpoint the job may contact reaches the check list;
  `FuzzWithhold` decodes arbitrary passmcp reports and checks that every
  reason names a DISCLOSURE.md rule with a bounded detail.

### Changed

- **The scorecard runs passmcp 0.0.3** (`PASSMCP_VERSION` in
  `registry-run.yml` and CI), a method change. Its checks, statuses and
  scoring are 0.0.2's; two outcomes can differ. A server whose list
  cursors loop now fails its catalogue with a named error instead of
  timing out, and a strict 2026-07-28 server that requires `Mcp-Param-*`
  headers now answers passmcp's tool calls instead of rejecting them.
  passmcp-reporting is required at v0.0.3.

### Fixed

- **A withholding reason is always valid UTF-8.** A reason quotes at most
  300 bytes of the server's text, and a cut inside a multi-byte character
  left half a character in the published record. The cut now lands on a
  character boundary, and `FuzzBound` checks it stays there.

## [0.0.2] — 2026-09-29

### Changed

- **The scorecard runs passmcp 0.0.2** (`PASSMCP_VERSION` in
  `registry-run.yml` and CI), a method change. Its checks, statuses,
  severities and scoring are those of 0.0.1; the one difference a record
  can show is the detail of a failed `protocol.id_echo`, which now names
  the id the server returned instead of a memory address.
- **The passmcp-reporting dependency is its v0.0.2 release**, not a
  pre-release commit, so the family resolves one version.
- **The product's run workflow is `registry-run.yml`.** `scorecard.yml` is
  now the OpenSSF Scorecard of this repository. The run no longer deploys
  Pages itself; it starts the Manual workflow, which publishes the manual,
  `coverage.json` and the scorecard under `/scorecard/` together. No record
  had been signed under the old workflow path.
- **The complexity ceilings are the portfolio's**: cyclomatic 10, cognitive
  15, 60 lines per function, and they now apply to tests too.
  `registry.Client.fetch`, `site.Site.Records`, the SPDX sweep and six
  tests were split to meet them, with no change in behaviour.

### Added

- A rendered manual (MkDocs, strict), `docs/ARCHITECTURE.md`, decision
  records, `CITATION.cff`, a dev container and a pre-commit configuration.
- A coverage badge published from CI, an OpenSSF Scorecard workflow, and a
  Release workflow that publishes the release page from a tag.
- `scripts/verify-release-versions.sh` (`make versions`), run in CI.

## [0.0.1] — 2026-09-29

The first release.

### Added

- **A signed public scorecard of the remote servers in the MCP Registry.**
  Each is checked read-only and without credentials by a pinned passmcp
  release, every result is an offline-verifiable attestation, and anything
  that would expose a vulnerability is withheld for its owner first.
