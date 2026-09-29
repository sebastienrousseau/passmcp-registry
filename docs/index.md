<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# passmcp-registry documentation

A signed public scorecard of the remote servers in the MCP Registry: each
one checked read-only and without credentials by a pinned passmcp release,
every result an offline-verifiable attestation, and anything that would
expose a vulnerability withheld for its owner first.

| Document | Covers |
|---|---|
| [README](https://github.com/sebastienrousseau/passmcp-registry/blob/main/README.md) | Install, Quick Start, configuration, limitations |
| [Method](method.md) | What is checked, what is contacted, and how to verify a record |
| [Architecture](ARCHITECTURE.md) | How a run flows, the packages, and what the workflow does around the binary |
| [Decision records](adr/README.md) | Decisions made in this repository, and why |
| [Release 0.0.2](releases/v0.0.2.md) | The highlights of the second release |
| [Release 0.0.1](releases/v0.0.1.md) | The highlights of the first release |
| [DISCLOSURE.md](https://github.com/sebastienrousseau/passmcp-registry/blob/main/DISCLOSURE.md) | The withholding rules W1 to W4 and the 90 days |
| [OPT-OUT.md](https://github.com/sebastienrousseau/passmcp-registry/blob/main/OPT-OUT.md) | How a server's owner opts out |
| [DEVELOPMENT.md](https://github.com/sebastienrousseau/passmcp-registry/blob/main/DEVELOPMENT.md) | Toolchain, every CI gate locally, the release model |

## The published scorecard

The scorecard is served beside this manual under
[`/scorecard/`](https://sebastienrousseau.com/passmcp-registry/scorecard/)
once the first run has been published, with `index.json` at its root. Its
history is the
[`records` branch](https://github.com/sebastienrousseau/passmcp-registry/tree/records),
which `passmcp-registry verify` reads. The data is CC-BY-4.0. No run has
been published yet.

## Coverage

The README's coverage badge reads
[coverage.json](https://sebastienrousseau.com/passmcp-registry/coverage.json),
published with this site by the Manual workflow from the statement coverage
measured on `main`.

What each passmcp check means, and how to fix it, is passmcp's manual:
<https://satellion.com/passmcp/docs/>.
