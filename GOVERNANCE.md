<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Governance

passmcp-registry is one repository in the passmcp family and is governed the
way the family is: one Maintainer, decisions recorded in issues and pull
requests, and the rules in [AGENTS.md](AGENTS.md).

The Maintainer decides:

- **the method:** the phases, the withholding rules and the record format;
- **when the job runs:** it has no schedule, and every run is started by
  hand;
- **every owner notice:** each one is sent by a person, never by the job.

**Decisions that need a second maintainer.** Enabling a schedule, and
publishing a withheld finding after its window, both wait until a second
maintainer can share the disclosure load. Until then, every run is started
and reviewed by the Maintainer.
