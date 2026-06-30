---
name: class-aware-uninstall
description: v-pkg `uninstall` is class-aware — refuses to silently delete a side-effecting patch; routes to restore/backout/delete by reversibility class.
metadata:
  type: project
---

**Class-aware `v pkg uninstall` (2026-06-25)** — the safety fix from the
patch-existing-routines proposal. Old uninstall blindly deleted routines+#9.7/#9.6,
which BRICKS a class-1 patched-over routine and ORPHANS the data/side-effects of a
side-effecting patch. Now it `Classify`s the `.KID` ([[reversibility-classifier]])
and routes via the pure
`decideUninstall(class, restoreKid, backoutKid, force, hasGreenfieldAdds)`:

- `--restore <pre-image.kids>` → **actRestore**: re-install the snapshot (the
  class-1 reversal for a patched-over routine — pairs with [[snapshot-restore]]).
- `--restore` **AND the build adds greenfield routines beyond the pre-image** →
  **actPartition** (see below).
- `--backout <kid>` → **actBackout**: install the authored inverse (class-2).
- both flags → **actRefuse** (ambiguous).
- side-effecting + no flags → **actRefuse** (exit 4 ExitRefused) — the core fix;
  `--force` overrides to **actDelete** with a loud "data/side-effects orphaned".
- class-1 + no flags → **actDelete** (greenfield routine-delete = the old path,
  backward compatible; reason warns to use --restore if it patched existing code).

**Partitioned uninstall (`actPartition`, 2026-06-30, the v-rpc-tap-scalable BB1
build-blocker).** A build can BOTH overwrite a foreign national routine (restore
it) AND add greenfield routines (delete them) — the v-rpc-tap shape: splices
`XWBPRS` *and* ships `VSLRT*`. A single restore ORPHANS the adds; a single delete
BRICKS the national routine. So `partitionRoutines(buildRoutines, preimage)` splits
the build by the **pre-image's routine set** (the snapshot captured exactly the
overwritten-foreign routines): in-pre-image → restore, not-in-pre-image → delete.
Then it runs BOTH halves, **ORDERED — restore FIRST, delete AFTER** (F-I/R21, so a
live caller never reaches a callee deleted before its caller is un-spliced). It
also clears the snapshot's `"<name> PREIMAGE"` #9.7/#9.6 footprint **before** the
restore (an existing #9.7 makes install a no-op → the pre-image wouldn't be put
back) and **after** completing (leave no ghost build) — so repeated install/
uninstall cycles are idempotent. `uninstallReport` gains `restored[]`/`deleted[]`.
The auto-snapshot sidecar makes the partition automatic: `install --auto-snapshot`
→ `uninstall` (auto-detects the sidecar → partition). Scope: the partition reverses
ROUTINE overwrites; a foreign NON-routine overwrite is still out of snapshot scope
(class-2, needs an authored back-out) — fine for the tap (its non-routine
components are all greenfield).

restore/backout/partition reuse `runInstall`; delete reuses `runUninstall`. The refuse path
returns BEFORE touching the engine (offline-decidable). `uninstallReport` JSON
carries name/class/action/reason/done/status. Added flags Restore/Backout/Force →
`make contract`. Single-build only (MULTI_BUILD guard).

**Smoke-proven offline** against a real side-effecting corpus patch
(OOPS*2.0*22): no flags → REFUSED exit 4 with guidance, engine untouched. Decision
logic unit-tested (pkgcli/uninstall_test.go, TestDecideUninstall 11 cases +
TestPartitionRoutines 5 cases). The partition is **dual-engine live-proven** by
`scripts/partition-uninstall-gate.sh` (17/17 on vehu YDB + foia-t12 IRIS through
the driver stack, idempotent): a synthetic ZZ* two-package fixture proves
overwrite-foreign→restore + greenfield-add→delete, ZZVPTF restored to v1 (drift
applied, NOT bricked) and ZZVPTG deleted.

**Why/how to apply:** the default is now SAFE — you cannot accidentally delete a
side-effecting patch. To reverse a patched-over national routine, snapshot first
(`v pkg snapshot`), then `uninstall --restore <pre-image>`. **Next:** auto-detect a
paired snapshot/back-out so the flags become optional; `verify-clean` after a
class-2 back-out; class-aware `install` (auto-snapshot before clobber).
