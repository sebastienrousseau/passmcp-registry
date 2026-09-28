<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Contributing

passmcp-registry publishes verdicts about servers other people run. A change
here is judged first by what it does to those servers and their owners,
and only then by what it adds.

## Before you start

- Open an issue for anything larger than a fix, so the method is agreed
  before the code.
- Read [AGENTS.md](AGENTS.md): it lists the rules that are expensive to
  discover from a diff.

## Making a change

1. Fork, branch from `main`, and keep the change to one concern.
2. Run the local gates:

   ```sh
   make            # format, vet, lint, headers, tests
   make test-race  # race detector, randomised order
   ```

3. Commit with a DCO sign-off (`git commit -s`) and a signed commit, with a
   [Conventional Commits](https://www.conventionalcommits.org/) subject of
   50 characters or fewer.
4. Open a pull request against `main`.

## What a change needs

- **85% statement coverage in every package** except `cmd/passmcp-registry`'s
  `main()`, which only wires signals and the exit code.
- **Tests against fixtures only.** A test never contacts the real registry
  or a real server; use a fake registry and fake MCP servers on loopback.
- **A method change bumps `MethodVersion`.** Changing the phases checked,
  the withholding rules or the record format changes what a published
  verdict means, and the version is how a reader tells old from new.
- **Anything that widens what the job does to a server needs an issue
  first.** That covers a new phase, a flag passed to passmcp, a higher rate
  or a new source of targets.

## Licence

Code contributions are licensed under AGPL-3.0-only; published data is
CC-BY-4.0.
