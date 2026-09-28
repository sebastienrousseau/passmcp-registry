<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: AGPL-3.0-only -->

# Disclosure policy

The scorecard publishes verdicts about servers other people run. Some
results would hand an attacker a working way in before the owner could fix
it. Those results are **withheld**: not published, and sent to the owner
first.

## What is withheld

A server's result is withheld when any of these holds. The rules are
[`internal/policy`](internal/policy/policy.go) in code, and changing one
changes the method version.

| Rule | What it means |
|---|---|
| W1 | `tools/list` answered with no credentials, and at least one tool is not declared `readOnlyHint: true`. Anyone who can reach the endpoint can call it. |
| W2 | `protocol.origin` failed: the server serves requests from a foreign browser `Origin`, so any web page can drive it through DNS rebinding. |
| W3 | Any `catalog.text.*` check failed: tool text carries hidden, encoded or injected instructions. That is tool poisoning, and it can mean the server is compromised. |
| W4 | `discovery.as.https` failed: the server's authorization server is reached over plain HTTP, so tokens can be read in transit. |

A withheld server doesn't appear anywhere in the published site. The index
doesn't list it even as withheld, because that label alone would point at
a vulnerable server. An earlier published record of the same version is
removed.

## What happens next

1. The job writes a private queue entry and a drafted notice for the owner.
   The queue never touches the site. In GitHub Actions it's encrypted with
   `age` before it's kept, because artifacts of a public repository are
   public.
2. **A person sends the notice.** The job sends nothing. The notice goes to
   the owner's security contact, or to their repository's private
   vulnerability reporting, or failing both, their GitHub profile's email.
3. **The owner has 90 days**, counted from the first run that found it.
   Running the job again doesn't restart the clock.
4. When a later run no longer finds the problem, the server's record is
   published as usual.
5. When 90 days pass without a fix, the Maintainer decides whether to
   publish. The owner is told first. Until there's a second maintainer, no
   withheld result is published after its window without a fresh review.

## Reproducing a finding

Every notice includes the exact command the scorecard ran, so the owner can
reproduce the finding before and after a fix:

```sh
passmcp check <url> --phases net,discovery,handshake,protocol,catalog --auth none
```
