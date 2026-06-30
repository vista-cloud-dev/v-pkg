---
name: snapshot-restore
description: v-pkg `snapshot`/`restore` verbs — pre-image capture off the live engine + class-aware honesty + preview-gated restore. Live-proven on vehu.
metadata:
  type: project
---

**`v pkg snapshot` / `v pkg restore` (2026-06-25)** — pre-image reversal, the
class-1 leg of the patch-existing-routines proposal. Builds on
[[reversibility-classifier]] for honesty. pkgcli/snapshot.go + restore.go.

**snapshot** `<kid> <out.kids> --engine ydb --transport docker`: parses the patch,
reads each routine it ships (`build.RoutineNames()`) off the LIVE engine via
`readRoutinePreimage` (`runMScript`+`readRoutineBody`, returning
`(lines, present, err)` so an ABSENT routine = greenfield, present=false, NO
error). Captured routines → `MakeBuildPairs`
→ `WriteKID` as a pre-image build named `"<orig> PREIMAGE"` (override `--name`).
HONESTY: `completeUndo=true` ONLY when class-1 pure-overwrite AND no greenfield
adds AND no non-routine components; else WARNS "provenance, not a reversal
guarantee." Result carries class + captured[] + absent[] + nonRoutine.

**restore** `<snapshot.kids> --engine … [--commit]`: restore = install-of-pre-image
→ reuses `runInstall`. PREVIEW by default (engine untouched); `--commit` overwrites
live routines (the gated step). `already-installed` (#9.7 identity exists) →
ExitRefused (re-restore needs the prior snapshot uninstalled or a fresh `--name`).

**Live-proven on vehu** (read-only legs): `snapshot` read the real 213-line
XWBBRK off the engine, wrote a re-parseable pre-image `.KID`, classified
`completeUndo:true`; `restore` preview listed XWBBRK, `committed:false`.
`--commit` deliberately NOT run (overwrites). vehu env:
`M_YDB_CONTAINER=vehu M_YDB_GBLDIR=/home/vehu/g/vehu.gld M_YDB_BIN=../m-ydb/dist/m-ydb
M_YDB_ROUTINES='/home/vehu/{p,s,r}/r2.02_x86_64*(…) …/lib/gtm/libgtmutil.so'` +
`--engine ydb --transport docker`.

**Tested** (pkgcli/snapshot_test.go) against the `fakeDriver`: name/namespace
derivation, pre-image build round-trip, present/absent read split, capture, and a
full snapshot→re-parse. Two new verbs changed the surface → `make contract`.
**Next:** non-routine pre-image (params/DD/options — proposal open Q2), `#9.7`/
content-address recording (Q1), and class-aware `install`/`uninstall` + `verify
--drift` (the engine-bound verbs that consume `snapshot`/`Classify`).
