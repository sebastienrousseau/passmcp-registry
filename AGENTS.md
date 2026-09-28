<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Working on passmcp-registry as an AI agent

These are the invariants for AI-assisted contributions. Read
[DEVELOPMENT.md](DEVELOPMENT.md) for the toolchain.

## Hard gates

| Gate | Command |
|---|---|
| 85% statement coverage, every package | `make coverage` |
| Race detector, randomised order | `make test-race` |
| Lint at zero findings, complexity ceilings included | `make lint` |
| SPDX header on every source file | `make spdx-check` |
| README follows the template | `make readme-check` |

## Things that are load-bearing

- **Never contact a real server or the real registry from a test or a
  manual run.** Use a fake registry and fake MCP servers on loopback. A live
  run is the Maintainer's decision, made in the workflow.
- **Only the listed servers, only these phases.** The job checks exactly
  what the registry lists, less opt-outs, with `checker.Phases`, `--auth
  none` and an empty passmcp configuration. Widening any of that is a method
  change: it needs an issue and a `MethodVersion` bump.
- **Withheld means absent.** A result that trips a DISCLOSURE.md rule
  appears nowhere in the site or its index, not even as "withheld".
- **The disclosure queue is private.** Never write it under the site, never
  upload it unencrypted, and never have the job send a notice.
- **A record is published only if its statement verifies** with
  passmcp-reporting and is about the listed endpoint.
- **The ninety days run from the first finding.** Re-running must not
  restart the clock.

## Commits

Signed, with a DCO sign-off, and a Conventional Commits subject of 50
characters or fewer. Never rewrite pushed history.
