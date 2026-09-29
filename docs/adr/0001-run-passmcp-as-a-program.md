<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# 0001. Run the pinned passmcp program; do not link its engine

## Status

Accepted.

## Context

The scorecard publishes verdicts about servers other people run. A reader
must be able to reproduce a verdict with a tool they can install, and the
safety properties passmcp has (no credential sent on an unauthenticated
probe, no mutating tool invoked without consent) must hold here without
being re-implemented.

## Decision

[`internal/checker`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/checker/checker.go) runs the passmcp
binary, at a version pinned in the workflow, with a fixed argument list
(`Passmcp.Args`), then `passmcp attest` on the report. The scorecard's
method is exactly that release plus those flags.

## Consequences

- An owner reproduces a finding with the `passmcp check` command in their
  notice, and gets the same answer.
- A new passmcp version is a method change: it is moved by hand, with a
  CHANGELOG entry, and every record carries the passmcp version it was
  produced by.
- The job depends on a binary on disk; the integration test runs the real
  one against fake servers on loopback.
