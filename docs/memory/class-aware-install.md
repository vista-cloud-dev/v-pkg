---
name: class-aware-install
description: v-pkg `install` is pre-image aware — refuses to silently overwrite an existing routine; --snapshot auto-captures the pre-image before clobbering.
metadata:
  type: project
---

**Class-aware `v pkg install` (2026-06-25)** — closes the no-silent-clobber side
of the patch-existing-routines proposal (pairs with [[class-aware-uninstall]],
[[snapshot-restore]]). Old install ran `runInstall` immediately, overwriting
national routines with no pre-image. Now it PROBES the engine first.

Flow (pkgcli/lifecycle.go installCmd): parse → `captureRoutinePreimages` reads
each shipped routine off the engine, splitting **existing** (present = overwrite
targets) vs **greenfield** (absent = new) → pure `decideInstall(hasExisting,
snapshot, allowOverwrite)`:

- no existing (pure greenfield) → **instProceed** (the old, safe path — backward
  compatible; ZZSKEL/params/files installs unaffected).
- existing + `--snapshot <out.kids>` → **instSnapshotProceed**: capture the
  overwrite targets' pre-image (`buildSnapshotPairs` over `existing`, `WriteKID`)
  BEFORE `runInstall` (enables `uninstall --restore`).
- existing + `--allow-overwrite` → **instProceed** (explicit unguarded clobber;
  uninstall cannot restore).
- existing + neither → **instRefuse** (exit 4 ExitRefused).

`installReport` JSON carries name/class/action/reason/overwrites/greenfield/
snapshot/installed/status. New flags Snapshot/AllowOverwrite → `make contract`.
Single-build only (MULTI_BUILD guard). Overwrite detection NEEDS the engine (unlike
uninstall's offline refuse), so it runs the read-only probe before deciding.

**Live-proven on vehu:** `install xwbbrk-patch.kids` (no flags) → REFUSED exit 4
("would overwrite existing national routine(s)") because XWBBRK is present —
read-only probe, engine untouched. The mutating `--snapshot`-then-install demo was
DENIED by the shared-engine guardrail (correct); the snapshot-capture leg is
unit-tested and the snapshot verb + runInstall are independently live-proven.

Decision logic unit-tested (pkgcli/install_test.go, TestDecideInstall, 6 cases).

**Why/how to apply:** to patch a national routine safely now:
`v pkg install <patch.kids> --snapshot pre.kids` (captures the rollback), and to
reverse: `v pkg uninstall <patch.kids> --restore pre.kids`. **Next:** `verify
--drift` (FU-21 re-pin gate), non-routine pre-image, paired snapshot/back-out
auto-detect.
