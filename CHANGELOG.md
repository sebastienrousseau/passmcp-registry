<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Changelog

All notable changes are documented here, in the format of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions move by
0.0.1 a release.

## [0.0.2]

### Changed

- **The passmcp-reporting dependency is its v0.0.1 release**, not a
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
