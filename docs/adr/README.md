<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Architecture Decision Records

Decisions about this repository that will be questioned later, with the
reasoning that produced them. Records are immutable once merged; a decision
that changes gets a new record superseding the old one.

Decisions about the checks, the score and the attestation format are
passmcp's, in [passmcp's ADRs](https://github.com/sebastienrousseau/passmcp/blob/main/docs/adr/README.md).
Records 0001 to 0004 were made with the first release and written down
afterwards, from the package documentation, the policies and the tests that
enforce them. Record 0005 was made when the manual was added.

| # | Decision | Status |
|---|---|---|
| [0001](0001-run-passmcp-as-a-program.md) | Run the pinned passmcp program; do not link its engine | Accepted |
| [0002](0002-only-non-invoking-phases-without-credentials.md) | Check only with phases that invoke nothing, and without credentials | Accepted |
| [0003](0003-withheld-means-absent.md) | A withheld result is absent from the site, and a person tells the owner | Accepted |
| [0004](0004-runs-start-by-hand.md) | Every run is started by hand; there is no schedule | Accepted |
| [0005](0005-one-pages-deployer.md) | One workflow deploys Pages: the manual, coverage and the scorecard | Accepted |
