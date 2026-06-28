---
name: install-fidelity-spike
description: "2026-06-28: Track A.1 install-fidelity spike — ground-truthed the KIDS phase boundary from real XPD* source. EN^XPDIJ is filing-only; env-check/required-builds/questions run earlier; pre/post routines need load-phase INI/INIT checkpoints. Recommends route (c) augmented direct-populate. The 'missing Developer Tools UG' blocker is resolved."
metadata:
  type: project
---

# v-pkg Track A.1 install-fidelity spike (2026-06-28)

Scoping spike for coverage-analysis F2 (the install bypasses real KIDS load).
Deliverable: `docs/proposals/v-pkg-install-fidelity-spike.md`. Grounded in **real
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

Risk: all from source/docs — **confirm live on vehu (YDB) + foia-t12 (IRIS)** before
relying; IRIS `^XTMP`/checkpoint/`$$ENV` parity unproven. Companion to
[[kids-coverage-analysis]], [[fileman-dd-component]]; builds on
kids-installation-automation.md §7.1.
