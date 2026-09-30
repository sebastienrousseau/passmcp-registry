# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: AGPL-3.0-only
#
# The README demo: a whole registry run on loopback. A fake registry
# (.github/demo/registry, served by python3's http.server) lists two of
# passmcp's example servers: the baseline, which is published, and
# any-origin, whose DNS-rebinding finding DISCLOSURE.md withholds and queues
# for its owner. Then `verify` checks the published record offline.
# Rendered by `make demo`, which installs passmcp and its example server at
# the lockstep release and runs this in build/demo/work; no real registry
# or server is contacted.
Output .github/demo.gif
Require passmcp
Require passmcp-registry
Require servers
Require python3
Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 260
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
# Everything stops itself after 90s: a hidden kill would still print a line.
Type "PS1='$ '; cd build/demo/work" Enter
Type "timeout 90 python3 -m http.server 17810 --bind 127.0.0.1 -d ../../../.github/demo/registry >/dev/null 2>&1 &" Enter
Type "timeout 90 servers -flaw baseline -addr 127.0.0.1:17811 >/dev/null 2>&1 &" Enter
Type "timeout 90 servers -flaw any-origin -addr 127.0.0.1:17812 >/dev/null 2>&1 &" Enter
# Let the set-up finish and clear it before recording starts.
Sleep 2s
Type "clear" Enter
Sleep 500ms
Show

Type "passmcp-registry run -registry http://127.0.0.1:17810 -per-host 1s" Sleep 400ms Enter
Sleep 5s
Type "passmcp-registry verify -site site" Sleep 400ms Enter
Sleep 2s
Type "ls site/servers private" Sleep 400ms Enter
Sleep 4s
