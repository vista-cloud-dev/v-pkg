---
name: install-fidelity-spike
description: "2026-06-28: Track A.1 install-fidelity spike — ground-truthed the KIDS phase boundary from real XPD* source. EN^XPDIJ is filing-only; env-check/required-builds/questions run earlier; pre/post routines need load-phase INI/INIT checkpoints. Recommends route (c) augmented direct-populate. The 'missing Developer Tools UG' blocker is resolved."
metadata:
  type: project
---

# v-pkg Track A.1 install-fidelity spike (2026-06-28)

Scoping spike for coverage-analysis F2 (the install bypasses real KIDS load).
Deliverable: `docs/archive/v-pkg-install-fidelity-spike.md`. Grounded in **real
`XPD*` routine source** (WorldVistA/VistA-M `Packages/Kernel/Routines/`: XPDIJ,
XPDIJ1, XPDI, XPDIL1, XPDIP, XPDIQ, XPDID) + the gold corpus
`krn_8_0_dg_kids_ug`.

## The KIDS phase boundary (the core finding)
`EN^XPDIJ` is **only the filing engine**. The deciding phases run earlier:
- **Env-check** = `ENV^XPDIL1` (name in `^XTMP("XPDI",XPDA,"PRE")`); called from
  the LOAD phase (`PKG^XPDIL1`, `XPDENV=0`) and from `EN^XPDI` (`$$ENV^XPDIL1(1)`,
  `XPDENV=1`). **Not in `EN^XPDIJ`.**
- **Required builds (#9.611)** = `REQB^XPDIL1`, reachable **only via `ENV`**
  (`XPDACT` 0=warn/1=abort+kill/2=abort → `XPDABORT`/`XPDREQAB`). Not in `EN^XPDIJ`.
- **Questions** prompted by `DIR^XPDIQ` in `EN^XPDI`, stored to #9.7 subfile 9.701.
  `EN^XPDIJ` only *reads* via `$$ANSWER^XPDIQ` = `$G(^XPD(9.7,XPDA,"QUES",IEN,1))`
  found through the `"QUES","B"` xref → **unseeded standard question = `""` = falsy
  = actuation skipped** (the safe automation default).
- **Pre/Post-install routines** = `PRE^/POST^XPDIJ1` (`D @XPDRTN` over
  `^XPD(9.7,XPDA,"INI"/"INIT",*,1)`) — these DO run inside `EN^XPDIJ`, **but only if
  the load-phase created the `INI`/`INIT` checkpoints** (`$$NEWCP^XPDUTL` in
  `PKG^XPDIL1`, routine name from `^XTMP("XPDI",XPDA,"INI"/"INIT")`).

## What this means for v-pkg's current path
The direct-populate + `EN^XPDIJ` install is faithful for **component filing** but
silently skips: pre/post-install routines (it never creates the INI/INIT
checkpoints — the sharpest gap), env-check, and required-builds. Questions default
to safe-NO (fine), but build-defined pre/post questions a routine reads via
`XPDQUES` are empty.

## Two findings that change the plan
1. **A.1 is NOT blocked on a missing doc.** The "Kernel 8.0 KIDS Developer Tools
   UG" is a *section inside* `krn_8_0_dg_kids_ug` (in the corpus), not a standalone
   guide. (Corrects P6.)
2. **No documented headless "load + seed answers + run" API exists** — KIDS is
   architected interactive; `XPDDIQ` suppresses only `XPZ1`/`XPZ2` (not `XPO1`/`XPI1`/
   device), `XPDNOQUE` only forbids queuing, `EN^XPDIJ` (ICR 2243) only tasks an
   *already-loaded* install. So faithful non-interactive install is a
   *reconstruction* of the pre-`XPDIJ` phase, not one call.

## Recommendation: route (c) augmented direct-populate
Keep the proven core; add explicit calls to the **real** phase functions (never
reimplement — stays inside [[bespoke-installer-forbidden]] + the waterline):
- **A.1.1** create the `INI`/`INIT` checkpoints (via `$$NEWCP^XPDUTL`/`PKG^XPDIL1`)
  so pre/post routines fire — **do first, live-gated both engines** (highest value).
- **A.1.2** call `$$ENV^XPDIL1(1)` (env-check + REQB together) before filing; honor
  `XPDQUIT`/`XPDABORT`/`XPDREQAB`.
- **A.1.3** seed `^XPD(9.7,XPDA,"QUES",IEN,0/1/"B")` from an install-spec (FileMan
  file to subfile 9.701 so the xref builds itself); standard 3 stay safe-NO.
- **A.1.4** (= Track A.3) PACKAGE #9.4 patch history.
Route (a) "drive `EN^XPDI` headless" rejected (un-suppressible prompts vs stdin-less
driver `Exec`); (b) expect = cross-engine fallback only. **Future cleaner option:**
if the SDK gains `SetGlobal`/stdin-`Exec`, call the real `EN1^XPDIL` load (natively
creates checkpoints + runs load env-check), retiring A.1.1's reconstruction.

## A.1.1 mechanism LIVE-CONFIRMED 2026-06-28 (vehu YDB + foia-t12 IRIS, identical)
Via the driver stack (`m vista exec --engine ydb|iris`; IRIS needs
`M_IRIS_NAMESPACE=VISTA` + `M_IRIS_CONTAINER=foia-t12` + the built
`m-iris/dist/m-iris` as `M_IRIS_BIN`; YDB needs `M_YDB_CONTAINER=vehu`):
- **`$$NEWCP^XPDUTL(name,callback,params)`** exists on both: reads `XPDCP`+`XPDA`;
  subfile **9.713** (`XPDCP="INI"`) / **9.716** (`"INIT"`); FileMan-files `.01`=name,
  **field 2=callback routine** (→ node `(c,1)`), field 3=params; idempotent
  (`$$FIND1^DIC`); returns the checkpoint IEN.
- **Grammar** (KERNEL 8.0, #9.7 IEN 2, both engines): `…,"INIT",2,0)="XPD
  POSTINSTALL STARTED^<FMtime>"`, `(2,1)="^XUINEND"`. `"…COMPLETED"` = base (no
  routine); `"…STARTED"` carries the routine; `PRE^/POST^XPDIJ1` `D @`s it when
  `(c,1)]""` and `$P((c,0),U,2)=""`.
- **A.1.1 impl** = after MERGE / before `EN^XPDIJ`, mirror `PKG^XPDIL1`: `S
  XPDCP="INI"` → `$$NEWCP^XPDUTL("XPD PREINSTALL COMPLETED")`, then iff
  `^XTMP("XPDI",XPDA,"INI")]""` → `$$NEWCP^XPDUTL("XPD PREINSTALL STARTED",<rtn>)`;
  same for `INIT`/POST. Calls real KIDS (not a reimpl). Only when the build ships a
  pre/post routine (no-init builds stay byte-identical).

Gotcha (driver quirk): IRIS `m vista exec` returns empty stdout unless
`M_IRIS_NAMESPACE=VISTA` is set; ZWR over the driver GVUNDEF'd on some nodes — use
explicit `$O`/`$G` walks with full (non-naked) refs. The m-iris driver binary must
be built first (`cd m-iris && make build`).

## A.1.1 DONE + live-proven 2026-06-28 (vehu YDB + foia-t12 IRIS)
`internal/installspec/script.go` `FinalInstallScript` now emits the checkpoint
block (real `$$NEWCP^XPDUTL` for INI/INIT COMPLETED + STARTED-with-routine) before
`EN^XPDIJ`. TDD `TestFinalInstallScript_PrePostCheckpoints`; gates green.
Live-proven through the real `v pkg install` over the driver stack with fixture
`testdata/zza1-prepost/` (ships `ZZA1P` PRE/POST setting `^ZZA1OUT`):
- **vehu pre-A.1.1 binary:** install status 3 but `^ZZA1OUT` empty → routines
  silently skipped (F2 confirmed live).
- **vehu + foia-t12 A.1.1 binary:** `^ZZA1OUT("PRE")=1`, `("POST")=1` → both fired,
  identical on YDB and IRIS. The old-vs-new counterfactual is the airtight proof.

**Driver gotchas learned (live):** YDB `v pkg install` (`exec load`) needs
`M_YDB_ROUTINES` = the container source dir (`/home/vehu/r`) — eval doesn't, only
load/compile does; "driver loaded no routine" = this is unset. IRIS needs
`M_IRIS_NAMESPACE=VISTA` + the built `m-iris/dist/m-iris` (`M_IRIS_BIN`). foia-t12
container was stopped — `docker start foia-t12` (lifecycle, allowed; not exec).

## A.1.2 DONE + live-proven 2026-06-28 (vehu YDB + foia-t12 IRIS)
`FinalInstallScript` gained `runEnvCheck bool`; plumbed through `runInstall`/
`liveInstall`/`installCmd` (`--skip-env-check` flag; restore/back-out callers pass
`false`). After the MERGE / before filing, the script reconstructs the install-phase
scope `EN^XPDI` sets (XPDI.m:11) — `XPDNM`, `XPDPKG`, and **`XPDT` (MUST seed: `$$ENV`'s
own tail `'$O(XPDT(0))` self-rejects a clean build otherwise)** — then calls the REAL
`$$ENV^XPDIL1(1)` (env-check routine in `^XTMP("XPDI",XPDA,"PRE")` + `REQB^XPDIL1`
#9.611 enforcement). Non-zero return → **purge the aborted #9.7 entry** (`K
^XPD(9.7,"B",XPDNM,XPDA),^XPD(9.7,XPDA)` → clean retry) + refuse with
`error=env-check-rejected^<rc>^<XPDREQAB>`. TDD `TestFinalInstallScript_EnvCheck`;
lint/race/contract green (new flag needs `make contract`).

Live-proven via the real `v pkg install` over the driver stack, fixture
`testdata/zza2-reqb/` (bogus Required Build `ZZNOPE*1.0*1` action code 2 vs a control
with none): **both engines** — bogus → REFUSED `env-check-rejected^2^2` + `#9.7`
auto-purged (9.7B=0); control → status 3; `--skip-env-check` bypasses (status 3).
**A.1.1 regression green** (zza1 `^ZZA1OUT PRE/POST=11` with env-check ON, both engines).

**GOTCHAS:** (1) seed `XPDT` or a clean build self-rejects (the non-obvious one). (2)
clean a stuck #9.7 with **direct global kill** `K ^XPD(9.7,DA),^XPD(9.7,"B",name,DA)` —
`^DIK` over `m vista exec` needs `DUZ(0)="@"` set or it silently no-ops. (3) the env-check
**routine** path runs in the same `$$ENV` call but isn't fixture-proven (a build's own
env routine isn't filed at env-check time; emitting the `"PRE"` node from
`buildspec.envCheck` is a B.3 follow-up). (4) **FOLLOW-UP BUG (cosmetic, not fixed):**
`internal/kids/reversibility.go` `installRoleNames` has `INI`/`PRE` **swapped** vs ground
truth — live `ENV^XPDIL1` reads `"PRE"`=env-check and `"INI"`=pre-install (A.1.1-proven),
but the map says `INI`=environment-check / `PRE`=pre-install. Display-only (classification
keys on subnode presence, not name).

Next: A.1.3 (seed #9.7 QUES from the `installspec` `Answers.XPDDIQ()`), B.3 authoring
(emit pre/post + env-check `"PRE"` nodes so fixtures are reproducible end-to-end), and
the `reversibility.go` INI/PRE label fix. Companion to the coverage-analysis proposal
(`docs/proposals/v-pkg-kids-coverage-analysis.md`), [[fileman-dd-component]]; builds on
`docs/design/kids-installation-automation.md` §7.1.
