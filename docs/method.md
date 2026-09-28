<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Method (version 1)

The method version is `MethodVersion` in
[`internal/job`](../internal/job/job.go). Every published record names the
version it was produced by.

## What is checked

The job reads the official MCP Registry's v0 servers API. It takes the
remote (HTTP) endpoints of the latest version of every active server, and
leaves out:

- templated URLs;
- endpoints that require headers the user supplies;
- endpoints whose URL carries credentials;
- opted-out servers.

Each endpoint is checked by a pinned passmcp release with:

```sh
passmcp check <url> --phases net,discovery,handshake,protocol,catalog \
  --auth none --output json --rps 1 --timeout 20s \
  --header "User-Agent: passmcp-registry/<version> (+https://github.com/sebastienrousseau/passmcp-registry)"
```

passmcp runs with an empty configuration file, so no profile can add a
credential or switch on mutations.

- **No tool is called.** The phases that call tools or hold sessions open
  (execution, performance, resilience) are not run, and neither is the auth
  phase, which presents a made-up credential. The protocol phase sends
  `tools/call` twice: once with no tool name, and once for a name the
  server doesn't list (`passmcp_no_such_tool`). Each checks that the server
  refuses; neither invokes anything.
- **Only listed servers are contacted,** plus the authorization servers a
  listed server itself advertises during discovery.
- **Slowly.** passmcp makes at most one request a second within a check.
  Checks against the same host start at least ten seconds apart.
- **User-Agent.** The published User-Agent is sent on every request passmcp's
  client makes. passmcp's first-contact probe uses a transport that
  deliberately carries nothing (passmcp ADR 0001), so those requests currently
  arrive with Go's default User-Agent. A passmcp flag that sets it on both
  transports is needed to close that gap.

## What a record is

`passmcp attest` turns passmcp's report into an in-toto statement. The job
verifies it with passmcp-reporting and checks that it is about the listed
endpoint. Then:

- a result that trips a [DISCLOSURE.md](../DISCLOSURE.md) rule goes to the
  private queue;
- anything else is published as `servers/<name>/<version>/record.json`,
  beside `attestation.json` and, once signed, `attestation.sigstore.json`
  (cosign keyless, from the workflow's own identity).

To check a record yourself:

```sh
passmcp-registry verify --site <copy of the site> --require-bundles
cosign verify-blob --bundle attestation.sigstore.json \
  --certificate-identity https://github.com/sebastienrousseau/passmcp-registry/.github/workflows/scorecard.yml@refs/heads/main \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com attestation.json
```
