<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Security Policy

passmcp-registry checks other people's MCP servers and publishes the results.
Its security posture has two sides:

- **What the job does to the servers it checks.** It reads only, without
  credentials, only the servers the registry lists, and slowly.
- **What it publishes.** Nothing that hands an attacker a working
  vulnerability before the server's owner has heard about it.

This file is about vulnerabilities in passmcp-registry itself. How the
scorecard handles vulnerabilities it finds in the servers it checks is in
[DISCLOSURE.md](DISCLOSURE.md).

## Reporting a Vulnerability

Report security issues through [GitHub's private vulnerability reporting](https://github.com/sebastienrousseau/passmcp-registry/security/advisories/new). Do not open a public issue.

You will receive an acknowledgement within **72 hours**. A confirmed
vulnerability is fixed within **90 days** of the report, or sooner when a
fix is straightforward. If the window cannot be met, you will be told why
and given a revised date.

Each of these is a vulnerability:

- a way to make the job contact a host the registry does not list;
- a way to make it send a credential or invoke a tool;
- a way to make a withheld result reach the published site;
- a way to make the private disclosure queue readable by anyone but the
  maintainers.

## Supported Versions

Only the latest release is supported.

## Security Measures

Each item names the test that enforces it.

- **Listed servers only.** The job checks exactly the endpoints the
  registry lists, less the opted-out ones.
  `TestTheRunContactsOnlyTheListedServers`, plus
  `TestTheRealPassmcpContactsOnlyListedServersAndCallsNoTool`, which runs
  the real passmcp binary with an unlisted server on loopback and asserts it
  receives nothing.
- **No tool is called, and no credential is sent.** passmcp runs with the
  phases net, discovery, handshake, protocol and catalog only, with
  `--auth none` and an empty configuration file.
  `TestTheCheckRunsOnlyNonInvokingPhasesUnauthenticated`.
- **Withheld means absent.** A result that trips a DISCLOSURE.md rule is
  removed from the site and appears nowhere in the index.
  `TestAVulnerableResultIsWithheldAndQueuedNeverPublished`.
- **The disclosure queue is private.** It is written outside the site, and
  the workflow encrypts it with `age` before keeping it as an artifact,
  because artifacts of a public repository are public. The workflow
  refuses to run without the recipient key.
- **Records are verifiable offline.** Every statement is checked with
  [passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting)
  before it is published and again by `passmcp-registry verify`, and each is
  signed with cosign keyless in the workflow.
- **Supply chain.** One direct dependency, passmcp-reporting, which has none.
  CI runs `govulncheck` on every push, and passmcp is pinned to a release.
