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
and routes via the pure `decideUninstall(class, restoreKid, backoutKid, force)`:

- `--restore <pre-image.kids>` → **actRestore**: re-install the snapshot (the
  class-1 reversal for a patched-over routine — pairs with [[snapshot-restore]]).
- `--backout <kid>` → **actBackout**: install the authored inverse (class-2).
- both flags → **actRefuse** (ambiguous).
- side-effecting + no flags → **actRefuse** (exit 4 ExitRefused) — the core fix;
  `--force` overrides to **actDelete** with a loud "data/side-effects orphaned".
- class-1 + no flags → **actDelete** (greenfield routine-delete = the old path,
  backward compatible; reason warns to use --restore if it patched existing code).

restore/backout reuse `runInstall`; delete reuses `runUninstall`. The refuse path
returns BEFORE touching the engine (offline-decidable). `uninstallReport` JSON
carries name/class/action/reason/done/status. Added flags Restore/Backout/Force →
`make contract`. Single-build only (MULTI_BUILD guard).

**Smoke-proven offline** against a real side-effecting corpus patch
(OOPS*2.0*22): no flags → REFUSED exit 4 with guidance, engine untouched. Decision
logic unit-tested (pkgcli/uninstall_test.go, TestDecideUninstall, 7 cases).

**Why/how to apply:** the default is now SAFE — you cannot accidentally delete a
side-effecting patch. To reverse a patched-over national routine, snapshot first
(`v pkg snapshot`), then `uninstall --restore <pre-image>`. **Next:** auto-detect a
paired snapshot/back-out so the flags become optional; `verify-clean` after a
class-2 back-out; class-aware `install` (auto-snapshot before clobber).
