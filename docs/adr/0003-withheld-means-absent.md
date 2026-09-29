<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# 0003. A withheld result is absent from the site, and a person tells the owner

## Status

Accepted.

## Context

Some results would hand an attacker a working way in before the owner
could fix it. Even a "withheld" label in a public index would point at a
vulnerable server.

## Decision

A result that trips a rule in [DISCLOSURE.md](https://github.com/sebastienrousseau/passmcp-registry/blob/main/DISCLOSURE.md) appears
nowhere in the site or its index, and any earlier record of that version is
removed. [`internal/disclosure`](https://github.com/sebastienrousseau/passmcp-registry/blob/main/internal/disclosure/disclosure.go)
writes a private queue entry and a drafted notice outside the site; the
workflow encrypts the queue with age before keeping it, because anything a
public repository's workflow keeps is public. The job sends nothing: a
person sends each notice. The ninety days run from the first run that
found the problem, and running again does not restart them.

## Consequences

- The published scorecard under-reports vulnerable servers during their
  window, by design.
- The workflow refuses to run without the age recipient.
- `TestAVulnerableResultIsWithheldAndQueuedNeverPublished` holds the line.
