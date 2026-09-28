<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Development

Go, at the version `go.mod` names. `golangci-lint` v2, as CI runs it.

| Task | Command |
|---|---|
| Everything CI runs locally | `make` |
| Race detector | `make test-race` |
| Coverage per package | `make coverage` |
| Integration test with a real passmcp | `PASSMCP_BIN=$(command -v passmcp) make integration` |
| Build | `make build` |

The integration test runs the real passmcp binary against fake MCP servers on
127.0.0.1. It proves that an unlisted server receives nothing and that no
listed tool is called. CI runs it with passmcp at the pinned version.

## A run, locally, against fakes only

```sh
make build
./build/passmcp-registry run --registry http://127.0.0.1:<fake-registry-port> \
  --passmcp "$(command -v passmcp)" --site /tmp/site --private /tmp/private
./build/passmcp-registry verify --site /tmp/site
```

Never point `--registry` at the real registry from your machine. Live runs
happen only in the `scorecard` workflow, started by hand.
