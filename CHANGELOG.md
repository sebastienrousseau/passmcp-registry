<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Changelog

All notable changes are documented here, in the format of
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions move by
0.0.1 a release.

## [0.0.9] - 2026-11-12

The family's ninth release. passmcp-registry did not ship a 0.0.8, so it
moves from 0.0.7 to 0.0.9 and joins the family at its version.

### Changed

- **In lockstep with passmcp 0.0.9**: passmcp-reporting is required at
  v0.0.9 (whose `attestation` verifier is unchanged from v0.0.7), and
  passmcp's trace runs at v0.0.9.
- **The scorecard runs passmcp 0.0.9** (`PASSMCP_VERSION` in
  `registry-run.yml` and CI), a method change. The phases, flags and
  withholding rules are unchanged, so `MethodVersion` stays 1. passmcp
  0.0.8 changed no check; 0.0.9 changes checks within the phases the
  scorecard runs (`net,discovery,handshake,protocol,catalog`): it adds
  `protocol.unknown_resource` (SEP-2164 missing-resource error checks when
  `capabilities.resources` is declared), checks `catalog.cache_hints`
  (SEP-2549) across `tools/list`, `resources/list` and `prompts/list` on
  `2026-07-28`, and skips `protocol.invalid_params` and
  `protocol.unknown_tool` when `capabilities.tools` is not declared while
  selecting a declared list RPC for stateless first contact, so a server's
  score can differ from 0.0.7's. None of those checks names a
  DISCLOSURE.md rule, so what is withheld is unchanged.
- **GitHub Actions updated**: `github/codeql-action` 4.39.3,
  `sigstore/cosign-installer` 4.1.3, `DavidAnson/markdownlint-cli2-action`
  25.0.0, `actions/setup-python` 7.1.0, and `ossf/scorecard-action` 2.5.0
  ([#12](https://github.com/sebastienrousseau/passmcp-registry/pull/12),
  [#15](https://github.com/sebastienrousseau/passmcp-registry/pull/15),
  [#17](https://github.com/sebastienrousseau/passmcp-registry/pull/17)).

### Security

- **Floor `certifi` at 2026.10.19 in the manual's build lock.**
  `docs/requirements.in` sets `certifi>=2026.10.19` and
  `docs/requirements.txt` resolves 2026.10.19 (PYSEC-2026-4320,
  GHSA-6q48-6c7w-9gw2,
  [#14](https://github.com/sebastienrousseau/passmcp-registry/pull/14)).
  The Go module does not use Python.

## [0.0.7] - 2026-10-17

The family's seventh release. passmcp-registry did not ship a 0.0.6, so
it moves from 0.0.5 to 0.0.7 and joins the family at its version.

### Changed

- **In lockstep with passmcp 0.0.7**: passmcp-reporting is required at
  v0.0.7, and passmcp's trace runs at v0.0.7.
- **The scorecard runs passmcp 0.0.7** (`PASSMCP_VERSION` in
  `registry-run.yml` and CI), a method change. The phases, flags and
  withholding rules are unchanged, so `MethodVersion` stays 1. passmcp
  0.0.6 changed no check; 0.0.7 changes one the scorecard runs:
  `protocol.deprecated_features` now asks for `initialize` as a
  handshake, so a server that serves both protocol generations is no
  longer warned that it declares handshake revisions it does not serve,
  and its score can differ from 0.0.5's. passmcp 0.0.7's other check
  change, to `protocol.malformed_json`, applies over stdio only, and the
  scorecard checks HTTP endpoints. Neither check names a DISCLOSURE.md
  rule, so what is withheld is unchanged.
- **CodeQL action 4.38.3.** The CodeQL workflow's `init` and `analyze`
  steps and the Scorecard workflow's SARIF upload move from 4.38.2 to
  4.38.3, still pinned by commit (Dependabot #8).

### Fixed

- **`make versions` reads DEVELOPMENT.md.** Its passmcp install line
  still named v0.0.4 after the 0.0.5 release, because the release check
  only read install lines in `README.md` and `docs/`. The line now names
  v0.0.5 and the check covers it.
- **The manual lists release 0.0.5, and fails when a page is left
  out.** `docs/releases/v0.0.5.md` was published but missing from the
  navigation; mkdocs reported it at INFO, which the strict build ignores.
  `mkdocs.yml` now raises omitted pages to a warning, so `make manual`
  and the Manual workflow fail on the next one.

### Security

- **Build with Go 1.26.9.** The `go` directive moves from 1.26.8, which
  `govulncheck` reports as reachable by ten standard-library advisories
  (GO-2026-6603 to GO-2026-6617): seven in `net/http` and one each in
  `crypto/tls`, `os` and MIME header parsing (`net/textproto`, reached
  through `mime/multipart`), all fixed in 1.26.9. CI, CodeQL and the Registry Run
  workflow set up Go from `go.mod`, so they build with it too.

## [0.0.5] - 2026-10-12

### Changed

- **In lockstep with passmcp 0.0.5**: updates passmcp-reporting to v0.0.5
  and the passmcp runner dependency to v0.0.5.

## [0.0.4] — 2026-09-30

### Added

- **A README demo**, rendered from `.github/demo.tape` by `make demo`: a
  scorecard run against a local registry listing two example servers: one
  record published, one withheld for its owner, and the published record
  verified.

### Changed

- **The scorecard runs passmcp 0.0.4** (`PASSMCP_VERSION` in
  `registry-run.yml` and CI), a method change. The phases, flags and
  withholding rules are unchanged, so `MethodVersion` stays 1; passmcp's
  checks are not. Within the phases the scorecard runs, passmcp adds
  `protocol.notification_ack`, `protocol.content_type`,
  `protocol.missing_session`, `catalog.tools.schema_valid` and
  `catalog.tools.order`, so a score can differ from 0.0.3's for the same
  server. A server on MCP revision 2024-11-05 is now graded, with the
  checks it predates skipped, instead of refused. None of the new checks
  names a DISCLOSURE.md rule, so what is withheld is unchanged.
- **In lockstep with passmcp 0.0.4**: passmcp-reporting is required at
  v0.0.4, and passmcp's trace runs at v0.0.4.

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
