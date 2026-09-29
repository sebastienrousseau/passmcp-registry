<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# 0002. Check only with phases that invoke nothing, and without credentials

## Status

Accepted.

## Context

The servers are not ours. Their owners have not consented to anything
beyond what an anonymous client of a public listing does.

## Decision

The checker runs the phases in `checker.Phases` (net, discovery,
handshake, protocol and catalog) with `--auth none` and an empty passmcp
configuration file, so no profile can add a credential or allow mutations.
Execution, performance and resilience call tools or hold sessions open, and
the auth phase presents a made-up credential; none of them runs. Checks on
one host start at least ten seconds apart, and passmcp makes at most one
request a second within a check.

## Consequences

- The scorecard shows only what an anonymous client sees. A server that
  needs a header or credentials to connect is listed as skipped, not
  scored.
- Widening any of this is a method change: it needs an issue first and a
  `MethodVersion` bump.
  `TestTheCheckRunsOnlyNonInvokingPhasesUnauthenticated` holds the line.
