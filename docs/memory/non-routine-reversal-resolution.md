---
name: non-routine-reversal-resolution
description: v-pkg open Q2 resolved — non-routine components reverse via the authored back-out, not generic pre-image; snapshot itemizes uncaptured components.
metadata:
  type: project
---

**Non-routine pre-image — resolved by design (2026-06-25, proposal open Q2).** The
patch-existing-routines proposal asked snapshot to also capture pre-images of
`#8989.51` params / DDs / options. Resolution: **do NOT.** Generic pre-image
capture is correct only for routines (pure overwrites). Non-routine components are
filed by install code with no generic inverse, so a captured "value" would
OVER-CLAIM reversibility — the one thing the proposal forbids. Those reverse via
the patch's **authored back-out** (`v pkg uninstall --backout`, the class-2 path,
already built).

What `snapshot` does instead: `uncapturedComponents(params, fileManFiles,
fileDDFiles)` (pkgcli/snapshot.go) ITEMIZES every non-routine component it does
NOT capture (params by name; FileMan FILE/DD by #; other KRN entries by file #;
8989.51 listed once as params, not duplicated). The `snapshotResult.uncaptured`
field + the warning enumerate them, so the operator knows exactly what the
authored back-out must cover — nothing silently dropped. Unit-tested
(pkgcli/snapshot_test.go TestUncapturedComponents). JSON-only field → no contract
change.

Blocker for true non-routine capture (deferred, not needed): v-pkg has NO
engine-read path for FileMan records — only routines via `$TEXT`
(readRoutinePreimage). A future enhancement could capture the genuinely-restorable
sub-case (a FileMan-entry overwrite with NO surrounding install code) but needs
per-component-type engine readers + must still gate on "no install code" to stay
honest.

**Why/how to apply:** for a side-effecting patch, the reversal workflow is ship an
authored back-out KIDS and run `uninstall --backout <bo.kids> --verify`; do NOT
expect `snapshot`/`restore` to undo params/DD/options. See
[[snapshot-restore]]/[[class-aware-uninstall]]. This CLOSES the reversibility
lifecycle (classify/snapshot/restore/install/uninstall/verify --drift/pairing);
only #9.7 provenance recording (open Q1) remains, a separate registry concern.
