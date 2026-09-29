#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: AGPL-3.0-only
#
# Fail unless every place that names the version being released names it:
# the CHANGELOG heading, the release highlights, the install lines in the
# README and docs, the version the README's ecosystem section states,
# CITATION.cff, the passmcp-reporting release go.mod requires, and the
# passmcp release the workflows and the trace run. The family moves in
# lockstep, so every one of them is the same version.
#
#   scripts/verify-release-versions.sh v0.0.1
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
tag="${1:-${GITHUB_REF_NAME:-}}"
[ -n "$tag" ] || { echo "usage: $0 vX.Y.Z" >&2; exit 2; }
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "not a release tag: $tag" >&2; exit 2; }
ver="${tag#v}"
fail=0
bad() { echo "$*" >&2; fail=1; }

grep -Eq "^## \[$ver\]" CHANGELOG.md || bad "CHANGELOG.md has no '## [$ver]' heading"

notes="docs/releases/v$ver.md"
if [ ! -f "$notes" ]; then
  bad "$notes is missing"
elif ! grep -Fq '## Highlights ⭐️' "$notes"; then
  bad "$notes has no '## Highlights ⭐️' section"
fi

# passmcp-registry's own install line and the passmcp it runs; at least one
# must be there, so a path that stops matching cannot pass silently.
installs=$(grep -Eoh 'satellion\.com/passmcp(-registry)?/cmd/passmcp(-registry)?@[^ `)]+' README.md docs/*.md || true)
if [ -z "$installs" ]; then
  bad "README.md names no go install line to check"
elif grep -v "@v$ver\$" <<<"$installs"; then
  bad "the docs pin a go install version other than v$ver"
fi

grep -Fq "Every component is released at **$ver**" README.md ||
  bad "README.md's ecosystem section does not state $ver"

grep -Eq "^version: \"?$ver\"?\$" CITATION.cff || bad "CITATION.cff does not say version $ver"

grep -Eq "satellion\.com/passmcp-reporting v$ver\$" go.mod ||
  bad "go.mod does not require satellion.com/passmcp-reporting v$ver"

# The passmcp release the scorecard and the integration test run.
for wf in .github/workflows/registry-run.yml .github/workflows/ci.yml; do
  pinned=$(sed -nE 's/^ *PASSMCP_VERSION: *(v[0-9.]+) *$/\1/p' "$wf")
  [ "$pinned" = "v$ver" ] || bad "$wf pins passmcp ${pinned:-<none>}, not v$ver"
done

# The acceptance-criteria trace is passmcp's tool, at the same release.
grep -Fq "satellion.com/passmcp/scripts/trace@v$ver " Makefile ||
  bad "Makefile does not run passmcp's trace at v$ver"

[ "$fail" -eq 0 ] && echo "release versions agree on $ver"
exit "$fail"
