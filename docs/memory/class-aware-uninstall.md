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

**Declared-foreign refuse (`F1`, 2026-06-30, closes the BB1 bare-uninstall hole).**
`actPartition` triggers off a PRE-IMAGE, so the auto-snapshot workflow is safe — but a
bare `uninstall <splice.kid>` with NO sidecar still fell through to `actDelete` and would
DELETE the foreign national routine (brick). Closed by an EXPLICIT declaration carried in
the build, not a name guess: the buildspec gains `foreignRoutines:[…]` (validated only as
valid + shipped names — `internal/buildspec` `validateForeignRoutines`), `MakeBuildPairs`
embeds it as a private `("VPKG","FOREIGN",<name>)` transport node, and `runInstall` strips
all `VPKG` nodes via **`kids.EnginePairs`** so the declaration NEVER reaches KIDS filing
(waterline rule 3 — only real KIDS content crosses the seam; it is v-pkg metadata that
rides in the `.KID` purely for offline reasoning). `ClassifyBuild` exposes it as
`ForeignOverwrites` (read verbatim from the node, via `Build.ForeignRoutines()`).
`decideUninstall` gains `hasForeignOverwrites` (checked BEFORE the side-effecting branch):
declared-foreign + NO pre-image → **`actRefuse`** (never delete); `--force` → `actDelete`
of the GREENFIELD subset ONLY (`partitionRoutines(build, foreign)` second half) — the
declared-foreign routine is excluded even under `--force`. Plus a wrong/incomplete-sidecar
guard: with a pre-image present, a declared-foreign routine that lands in the delete set
(not captured by the snapshot) → REFUSE (`intersectRoutines`). Dual-engine proven
(`scripts/foreign-refuse-gate.sh` 17/17 vehu+foia: build-guard offline, refuse exit 4 with
the engine byte-untouched, `--force` deletes greenfield-only). Implements D1+D2+D3 of
`../proposals/v-pkg-mixed-build-split-reversal.md`.

**National-overwrite verify (`D4`, 2026-06-30).** `uninstall --verify` of a restore/partition
that touched declared-foreign routines now adds a STRICTER grade than the line-2-blind
`RoutineDriftMatch`: `kids.RoutineRestoreVerdict(shipped, live)` →
`exact` | `command-clean-provenance-drift` | `drift`, surfaced as `uninstallReport.foreignRestore`.
The KEY FACT (N-5, corpus-confirmed): both VistA checksums (`$$RSUM`/`CHECK1^XTSUMBLD`) EXCLUDE
line 2, so the **command lines (every line except line 2) ARE the checksum surface** —
byte-equality there ⇒ the restored routine matches the FORUM #9.8 gold checksum. Line 2 carries
volatile patch-history provenance (patch list/date/Build N): `command-clean-provenance-drift` =
checksum-clean but provenance differs; `drift` = a command line differs or a line was
added/removed (the restore did NOT reinstate it — and this case ALSO trips the existing
line-2-blind `verifyClean=dirty` → `ExitCheck`, so `foreignRestore` adds diagnostic precision,
not a new exit). `foreignRestoreVerdict` (pkgcli) reads each foreign routine live and returns the
WORST verdict; absent → `drift`. Live verdict on a clean partition is always `exact`
(drift/provenance cases are inherently a broken restore — covered by unit tests, not the live
gate). Implements D4 of the proposal. `scripts/foreign-refuse-gate.sh` asserts `exact` (24/24
vehu+foia).

**GOTCHA — foreignness is NEVER inferred from routine names.** A package's routine
namespace need not match its package name (m-stdlib's package is `MSL` but it ships `STD*`
routines; v-stdlib's `VSL` ships `VSL*`). So the "routine whose name doesn't start with the
package namespace = foreign" prefix-heuristic the original F1 plan proposed is **wrong** —
it would fail the real `std.build.json`/`vsl.build.json` builds. Detection is by the
explicit `foreignRoutines` declaration ONLY (embedded in the `.KID`), keeping v-pkg's
name-agnostic robustness across the 2,000-package corpus. Do NOT re-propose a name guess.

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
