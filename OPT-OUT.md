<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Opting out

If you own a server listed in the MCP Registry and don't want it checked or
published by this scorecard, you can opt out. No reason is needed.

## How

Open an issue titled `opt-out: <registry name>`, for example
`opt-out: io.github.example/weather`, from the GitHub account or
organisation that owns the registry namespace. To opt out every server
under a namespace, write the namespace followed by `/*`, for example
`io.github.example/*`.

If you can't open a public issue, email the maintainer at the address in
[SECURITY.md](SECURITY.md) instead.

## What happens

- The name is added to [`opt-out.txt`](opt-out.txt) before the next run.
- **On the next run**, the server is not contacted at all. Every record the
  scorecard published about it, for every version, is deleted from the
  site, and it disappears from the index.
- Nothing about an opted-out server is kept in public data after that run.
  The published annual report is built from the records on the site, so
  later reports don't include it either.

Opting back in is the same issue, titled `opt-in: <registry name>`.
