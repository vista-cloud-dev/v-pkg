---
name: snapshot-clobber-and-lowercase-tag
description: Two v-pkg hardening fixes found by stress-testing the v-rpc-tap foreign-overwrite build (splices national XWBPRS + ships VSLRT* + #19 OPTION) on live vehu — a serious pre-image-clobber back-out bug and an over-strict lowercase-tag validator.
metadata:
  type: project
---

Both found 2026-06-30 by driving the **v-rpc-tap** KIDS build (overwrites the
national `XWBPRS` via the splice + ships greenfield `VSLRT*` + a `#19` OPTION)
through `v pkg install/uninstall` on live `vehu` and trying to BREAK it. See
[[pairing-verify-clean]], [[class-aware-install]], [[class-aware-uninstall]],
[[adversarial-stress-gate]].

## 1. SERIOUS — `install --auto-snapshot` clobbered the pre-image on a redundant install
**Symptom:** a *second* `install --auto-snapshot` of an ALREADY-installed
foreign-overwrite build silently **destroyed the back-out**. The redundant install
re-captured the snapshot from the routines in their *post-install* (already-spliced)
state and overwrote the conventional sidecar `<kid>.preimage.kids` — **before** the
install refused as already-installed. After that the sidecar held the SPLICED
XWBPRS, so `uninstall --restore` could never restore the original national routine.
Worse: because the clobbered pre-image then contained *all* build routines (none
looked greenfield relative to it), `decideUninstall` picked `actRestore` (not
`actPartition`) → install-of-pre-image → hit the already-installed guard → no-op
(`done:false`, NOT_UNINSTALLED). The build became effectively un-backoutable via the
normal sidecar path (recovery needed a hand-built XWBPRS-only original artifact to
force the partition path).

**Root cause:** in `liveInstall` (pkgcli/lifecycle.go) the capture + `WriteKID`
sidecar write ran *before* `runInstall` detected already-installed.

**Fix:** probe install state (`probeHeal` → `healHealthy`) when a snapshot is
requested, and gate the sidecar write on the pure helper
`snapshotShouldWrite(action, alreadyInstalled) = action==instSnapshotProceed &&
!alreadyInstalled`. A snapshot is now captured **exactly once**, against the true
pre-install state; a redundant install preserves the genuine sidecar and just
reports already-installed. Regression: unit `TestSnapshotShouldWrite` + live on
vehu (sidecar still 0 tap-calls after the redundant install; partition uninstall
byte-clean). No effect on first-install paths (the existing gates install on clean
engines → `alreadyInstalled=false`).

## 2. `#19` OPTION / pre-post-install entryref validator rejected modern lowercase tags
`reLabel = ^[A-Z][A-Z0-9]*$` rejected a valid lowercase M line label, so a
run-routine OPTION with `routine:"run^VSLRTRP"` (and `preInstall`/`postInstall`
hooks with lowercase tags) failed `buildspec` validation — even though lowercase
labels are valid M and the reaper's own `ZTRTN="run^VSLRTRP"` runs under live
TaskMan. **Fix:** `reLabel = ^[A-Za-z][A-Za-z0-9]*$` (tag case-relaxed; the ROUTINE
half stays uppercase — routine files are uppercase, so `run^vslrtrp` is still
rejected). The pre-existing `lower^ZZA1P` "bad tag" test case encoded the old
assumption and was retargeted to `bad-tag^ZZA1P` (hyphen). Live: the complete
v-rpc-tap build now files `^DIC(19,…,25)="run^VSLRTRP"` (type R) and uninstalls clean.
