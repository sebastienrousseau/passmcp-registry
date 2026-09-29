<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# 0004. Every run is started by hand; there is no schedule

## Status

Accepted.

## Context

A run contacts every listed server and may produce findings that a person
has to disclose. With one maintainer, a schedule could produce findings
faster than they can be handled.

## Decision

The [Registry Run workflow](https://github.com/sebastienrousseau/passmcp-registry/blob/main/.github/workflows/registry-run.yml) has
`workflow_dispatch` only. Tests and local work use a fake registry and fake
servers on loopback; nothing points `--registry` at the real registry
outside that workflow.

## Consequences

- The scorecard is as fresh as the last run a maintainer started.
- Enabling a schedule waits for a second maintainer, as
  [GOVERNANCE.md](https://github.com/sebastienrousseau/passmcp-registry/blob/main/GOVERNANCE.md) records.
