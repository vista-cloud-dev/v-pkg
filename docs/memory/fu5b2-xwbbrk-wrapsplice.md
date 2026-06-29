---
name: fu5b2-xwbbrk-wrapsplice
description: Tombstone — the deleted v-pkg wrap-rpc / internal/wrapsplice XWBBRK patcher. Do not revive; see bespoke-installer-forbidden.
metadata:
  type: project
---

# internal/wrapsplice / `v pkg wrap-rpc` — DELETED (tombstone)

> **⛔ REMOVED 2026-06-25 (owner directive) — DO NOT REVIVE.**

`internal/wrapsplice` and the `v pkg wrap-rpc status|install|backout` command were
a **bespoke installer** — a content-anchored host-side splice of the two
`VSLRPCWRAP` traffic-tap side-calls into national `CALLP^XWBBRK` — and have been
**deleted** from v-pkg. Install + back-out of anything into a live VistA is
**strictly** `v pkg install` / `v pkg uninstall` of a proper KIDS build; no
hand-rolled patcher, ever.

The durable lesson lives in [[bespoke-installer-forbidden]]. The mechanics (the
splice anchors, the FU-21 re-pin gate, the read→splice→KIDS delivery) are gone
from the tree — recover them from git history (`git log --all -- internal/wrapsplice`)
if ever needed. The correct, KIDS-native way to ship the VSL RPC tap as a *mixed*
overwrite+greenfield build is the **proposed** split-reversal work, not this code:
see the central `docs/proposals/v-pkg-mixed-build-split-reversal.md`.
