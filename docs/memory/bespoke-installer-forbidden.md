---
name: bespoke-installer-forbidden
description: HARD DIRECTIVE (owner, 2026-06-25) — never build or use a bespoke installer/patcher in v-pkg. Install + back-out is strictly and exclusively the generic v pkg install / v pkg uninstall KIDS lifecycle. The wrap-rpc / internal/wrapsplice splice was deleted.
metadata:
  type: feedback
---

**HARD DIRECTIVE (owner, 2026-06-25): never again build or use a bespoke
installer/patcher.** Installing onto, and backing out of, a live M engine is
**strictly and exclusively** the generic v-pkg KIDS lifecycle:

- **Install** → `v pkg install <build>.KID --engine {ydb,iris}` (`liveInstall` /
  `runInstall`).
- **Back out** → `v pkg uninstall <build>.KID --engine {ydb,iris}` (routine +
  #9.7/#9.6, optionally `--restore` a pre-image / `--backout` an authored inverse).

A package ships through a proper, drift-gated **KIDS build** (e.g. the RPC traffic
tap ships in v-stdlib's `kids/vsl.build.json` → `dist/kids/VSL.kids`, carrying the
`VSLTAP*` / `VSLRPC*` routines + #8989.51 PARAMETER DEFINITIONs). There is no
host-side splice, no content-anchored patch of a national routine, no second
back-out routine, no hand-rolled install M.

**What was removed (2026-06-25):** `v pkg wrap-rpc status|install|backout` and
`internal/wrapsplice` (the `CALLP^XWBBRK` splice) — deleted from v-pkg with their
tests; the orphaned `readRoutineSource` + `liveRestore` helpers went too. This
extends the earlier removal of the M-side bespoke install routines (VSLTAPBO /
VSLBLD / VSLENV — see v-stdlib `bespoke-install-routines-removed`).

**Why:** one install path = one thing that can drift, one thing that is gate-tested
and live-proven. Patching national routines off-engine is exactly the fragility the
KIDS lifecycle exists to avoid. **How to apply:** if a task says "tap / patch /
splice the broker", that is `v pkg install` of a KIDS build that *contains* the tap
routines — never a routine-splicing tool. Do not re-add a `wrap-rpc`-style command.
Historical detail of the deleted mechanism: [[fu5b2-xwbbrk-wrapsplice]]. Shared
org-level directive: [[never-use-bespoke-installer]].
