---
name: verify-drift
description: v-pkg `verify --drift` — detects whether a shipped patch is still applied to the live routine (FU-21 re-pin gate). Live-proven on vehu both ways.
metadata:
  type: project
---

**`v pkg verify --drift` (2026-06-25)** — the FU-21 re-pin gate: "is my patch
still applied to the live routine, or did a later national patch overwrite it?"
Pairs with [[class-aware-install]]/[[class-aware-uninstall]]/[[snapshot-restore]].

Implementation: `verifyCmd.Drift` flag → `checkDrift` reads each shipped routine
off the engine (`readRoutinePreimage`) and compares to the patch's own source
(`Build.RoutineSource(name)`, new) via `kids.RoutineDriftMatch` (new). The compare
canonicalizes the routine's **2nd line** (`CanonicalizeRoutineLine2` — the `;;`
version/patch-list/checksum line KIDS rewrites at install) so a real checksum
rewrite is NOT a false drift; any OTHER differing line, or a length mismatch, is
drift. Per-routine verdict: `applied` / `drifted` / `absent`. `verifyResult.ok()`
fails (exit 3 ExitCheck) if any routine is not `applied` when --drift is set; the
hint says "re-apply this patch (FU-21 re-pin)".

**Live-proven on vehu BOTH ways:** the patched XWBBRK .KID (215 lines, spliced) vs
the live stock routine (213 lines) → `drifted` (true positive — the wrap is NOT
currently on vehu, `wrap-rpc status` confirms spliced:False); the captured
pre-image .KID (213, stock) vs live → `applied`. So drift correctly distinguishes
applied-vs-not.

Pure logic unit-tested (internal/kids/drift_test.go: RoutineSource ordering,
RoutineDriftMatch line-2-tolerant match / body-change drift / length mismatch).
The --drift flag changed the verify surface → `make contract`.

**Why/how to apply:** run `v pkg verify --drift <patch.kids>` to gate that a patch
is still live before relying on it (e.g. the VSLTAP wrap after any XWB patch). A
`drifted` result means re-install the patch. Gotcha: RoutineSource matches the
4-subscript `"RTN",<name>,<n>,0)` source nodes (sorted by n), NOT the 2-sub name
node.
