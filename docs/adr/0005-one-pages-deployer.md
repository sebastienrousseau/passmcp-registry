<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# 0005. One workflow deploys Pages: the manual, coverage and the scorecard

## Status

Accepted.

## Context

A repository has one GitHub Pages site, and each deployment replaces it
whole. Three things need publishing: the manual, the `coverage.json` the
README's coverage badge reads, and the scorecard the Registry Run workflow
produces. Two workflows deploying independently would each erase the
other's part.

## Decision

Only the [Manual workflow](https://github.com/sebastienrousseau/passmcp-registry/blob/main/.github/workflows/manual.yml) deploys Pages.
It builds the manual at the root, writes `coverage.json` beside it, and
copies `site/` from the `records` branch under `/scorecard/` when the
branch exists. It runs on every push to `main`; the Registry Run workflow
starts it with a `workflow_dispatch` after pushing new records, which a
`GITHUB_TOKEN` is allowed to do.

## Consequences

- The published site is at `https://sebastienrousseau.com/passmcp-registry/scorecard/`,
  not the root. The records branch, which verification reads, is unchanged.
- The Registry Run workflow needs `actions: write` and no Pages
  permissions.
- The run workflow was renamed from `scorecard.yml` to `registry-run.yml`,
  freeing that name for the OpenSSF Scorecard as in the rest of the family.
  No record had been signed under the old name, so no signature names it.
