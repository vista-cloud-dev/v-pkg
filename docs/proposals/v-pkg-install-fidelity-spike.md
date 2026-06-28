---
title: "v-pkg install-fidelity spike (Track A.1) — driving real KIDS phases non-interactively"
status: proposed
created: 2026-06-28
last_modified: 2026-06-28
revisions: 1
doc_type: [PROPOSAL, SPIKE]
grounding:
  - "Real XPD* routine source (WorldVistA/VistA-M, Kernel/Routines): XPDIJ, XPDIJ1, XPDI, XPDI1, XPDIL, XPDIL1, XPDIP, XPDIQ, XPDID — fetched verbatim, control flow traced"
  - "vdocs GOLD corpus — Kernel 8.0 Developer's Guide: KIDS UG (XU/krn_8_0_dg_kids_ug) + Systems Management KIDS UG (XU/krn_8_0_sm_kids_ug)"
  - "v-pkg source — internal/installspec/script.go, pkgcli/lifecycle.go (HEAD 2026-06-28)"
  - "prior live ground truth — kids-installation-automation.md §7.1 (ZZSKEL on YDB FOIA vehu, 2026-06-12)"
related:
  - kids-installation-automation.md
  - v-pkg-kids-coverage-analysis.md
  - implementation-plan.md (P5/P6)
---

# v-pkg install-fidelity spike (coverage-analysis Track A.1)

This is the grounded scoping spike the coverage analysis called "**THE pivotal
decision**": make `v pkg install` run real KIDS load/install semantics
(environment check, required-build enforcement, pre/post-install routines,
questions) instead of the `^XTMP`-populate + direct `EN^XPDIJ` shortcut (finding
F2). It ends in a **route recommendation**, not code.

## Verdict (recommendation)

**Recommended route: (c) augmented direct-populate — keep the proven
direct-`^XTMP` + `EN^XPDIJ` path and close F2's gaps by calling the *real* KIDS
phase functions explicitly, in priority order.** Neither the "drive the
interactive install option headless" route (a) nor the "expect-driven menus"
route (b) fits the driver's stdin-less `Exec` seam; route (c) does, stays inside
the waterline + the [[bespoke-installer-forbidden]] ban (it *invokes* KIDS, never
reimplements it), and lands incrementally with a live gate per step.

**Two findings up front that change the plan:**

1. **A.1 is NOT blocked on a missing document.** The roadmap flagged the *Kernel
   8.0 KIDS Developer Tools UG* as absent from the gold corpus and a prerequisite.
   It is **not a standalone guide** — its developer-variable content (`XPDENV`,
   `XPDQUIT`/`XPDABORT`, `XPDDIQ`, `XPDNOQUE`, `XPDQUES`, `EN^XPDIJ`/ICR 2243)
   lives as a *section inside* `krn_8_0_dg_kids_ug`, which **is** in the corpus.
   The routine source closes the rest. So this updates P6: the gap is largely
   illusory; we have what we need to scope and build A.1. *(Engine-specific
   validation is still required — see Risks.)*

2. **There is no documented headless "load + seed every answer + run" API.** KIDS
   is architected interactive (DG confirms: prompt-suppression is the build's own
   env-check pre-seeding `XPDDIQ`, and `EN^XPDIJ` only *tasks an already-loaded*
   install — ICR 2243). So "faithful non-interactive install" is necessarily a
   *reconstruction* of the pre-`XPDIJ` phase, not a single API call. Route (c)
   reconstructs it by calling the real sub-functions.

---

## The phase boundary (ground truth)

Traced from real `XPD*` source. **`EN^XPDIJ` is only the *filing* half of KIDS.**
Everything that decides *whether/how* to install runs earlier — in the LOAD phase
(`EN1^XPDIL`→`XPDIL1`) and the interactive install option (`EN^XPDI`→`XPDI1`).
`EN^XPDIJ` performs *actuations* (pre/post routines, inhibit-logons,
disable-options) but each is **gated on data the earlier phases must have already
written** into `^XPD(9.7,XPDA,…)` / `^XTMP("XPDI",XPDA,…)`.

| Step | Inside `EN^XPDIJ`? | Where it really lives / the catch |
|---|---|---|
| **Environment-check routine** | **NO** | `ENV^XPDIL1` only; called from `PKG^XPDIL1` (load, `XPDENV=0`) and `EN^XPDI` (`$$ENV^XPDIL1(1)`, `XPDENV=1`). Env-check name in `^XTMP("XPDI",XPDA,"PRE")`. |
| **Required builds (#9.611)** | **NO** | `REQB^XPDIL1`, reached *only* via `ENV`. Reads `^XTMP("XPDI",XPDA,"BLD",bld,"REQB",…)`; action code `XPDACT` 0=warn / 1=abort+kill / 2=abort → sets `XPDABORT`/`XPDREQAB`. |
| **Pre-install routine** | **YES** (conditional) | `PRE^XPDIJ1`: `D @XPDRTN` over `^XPD(9.7,XPDA,"INI",*,1)`. Fires *iff* the load phase created the `INI` checkpoint (`$$NEWCP^XPDUTL` in `PKG^XPDIL1`, name from `^XTMP("XPDI",XPDA,"INI")`). |
| **Post-install routine** | **YES** (conditional) | `POST^XPDIJ1`: `D @XPDRTN` over `^XPD(9.7,XPDA,"INIT",*,1)`. Same checkpoint dependency. |
| **Prompting the questions** | **NO** | `DIR^XPDIQ` in `EN^XPDI`/`QUES^XPDI1`, stored to #9.7 subfile 9.701. `EN^XPDIJ` only *reads* answers via `$$ANSWER^XPDIQ`. |
| **Disable options/protocols (actuate)** | **YES** (gated) | `OFF^XQOO1`, gated on `^XPD(9.7,XPDA,0)` pc 8 + `^XTMP("XQOO",set)` — written only in `EN^XPDI`. Absent ⇒ skipped. |
| **Inhibit logons (actuate)** | **YES** (gated) | `INHIBIT^XPDIJ1`, gated on `$$ANSWER^XPDIQ("XPI1")`. Question asked only in `EN^XPDI`. |

**Key reads `EN^XPDIJ` makes** (verbatim): `I $$ANSWER^XPDIQ("XPI1") D INHIBIT^XPDIJ1("Y")`
(inhibit), `I $$ANSWER^XPDIQ("XPO1") D …KIDS^XQ81` (menu rebuild). `$$ANSWER^XPDIQ`
returns `$G(^XPD(9.7,XPDA,"QUES",IEN,1))` found via the `"QUES","B"` cross-reference
— so an **unseeded** standard question returns `""` (falsy) ⇒ the actuation is
**skipped**, which is the *safe automation default* (no inhibit, no disable, no
forced menu rebuild).

---

## What v-pkg's install runs and skips today

`internal/installspec/script.go` + `pkgcli/lifecycle.go`: create the #9.7 entry
via `$$INST^XPDIL1`, stream the parsed `.KID` pairs into `^XTMP("VPKGI")`, MERGE
into `^XTMP("XPDI",XPDA)`, seed the KRN + FIA component-tracking nodes, then
`D EN^XPDIJ`. Mapping that against the boundary above:

| Phase | v-pkg today | Why |
|---|---|---|
| Component filing (routines, DD, data, KRN, FIA) | ✅ runs | `EN^XPDIJ` |
| **Pre/Post-install routines** | ❌ **silently skipped** | direct-populate never creates the `^XPD(9.7,XPDA,"INI"/"INIT",2,1)` checkpoints, so `PRE^/POST^XPDIJ1`'s `$O(…,"INI",…)` loop finds nothing and falls through (no error) |
| **Environment check** | ❌ skipped | no call to `ENV^XPDIL1`; jumps straight to `EN^XPDIJ` |
| **Required builds (#9.611)** | ❌ not enforced | `REQB` is reachable only via `ENV` |
| **Questions / answers** | ⚠️ defaults only | no answers seeded ⇒ `$$ANSWER` returns `""` ⇒ standard actuations skipped (safe), but **build-defined pre/post questions a routine reads via `XPDQUES` are empty** |
| Disable-options / inhibit-logons | ⏭️ intentionally off | the safe automation default |

So the install is **faithful for component filing but silently omits the build's
own gating + migration logic** — exactly F2. The sharpest single gap is
pre/post-install routines (a build's data migration / xref rebuild simply doesn't
run), and its mechanism is now known precisely.

---

## The three routes

### (a) Silent `XPD*` answer-variable seeding — drive `EN^XPDI` headless — REJECTED
Pre-seed answers + `XPDDIQ` and call the real interactive install option
non-interactively. **Why it doesn't fit:** `EN^XPDI` prompts via `DIR^XPDIQ` and
**always selects a device** (`DEVICE: HOME//`); the DG documents an `XPDDIQ`
suppression knob for only **two** prompts (`XPZ1` disable, `XPZ2` move-routines) —
**no documented knob for rebuild-menu-trees (`XPO1`), inhibit-logons (`XPI1`), or
the DEVICE prompt** (`XPDNOQUE` only *forbids queuing*, the device prompt remains).
The driver `Exec` is subprocess + JSON with **no interactive stdin**, so the
un-suppressible prompts have nothing to read and the option hangs/aborts. Driving
`EN^XPDI` headless means out-prompting a routine designed to prompt — brittle and
not fully achievable from docs.

### (b) Expect-driven pseudo-terminal — FALLBACK ONLY
Spawn a TTY, match prompt → send answer. Cross-engine brittle, needs a
stdin-capable transport the reference `mdriver.Client` doesn't expose, and re-couples
us to scraped output. Keep strictly as the last-resort cross-engine fallback
(matches the existing "Tier B"); not the path.

### (c) Augmented direct-populate — RECOMMENDED
Keep the proven, driver-friendly direct-populate + `EN^XPDIJ` core; **add explicit
calls to the real KIDS phase functions** so the skipped phases run natively. Each
addition seeds the *data structures real KIDS reads* (the same blessed pattern as
the existing KRN/FIA seeds) and/or *invokes a real KIDS entry* — never reimplements
KIDS logic. Lands as independent, live-gated increments:

- **A.1.1 — Pre/Post-install routines (highest value).** Reproduce the load-phase
  checkpoint creation: from the build's pre/post routine names (carried in the
  transport as `^XTMP("XPDI",XPDA,"INI"/"INIT")`, or read from #9.6), create the
  `^XPD(9.7,XPDA,"INI",2,0/1)` + `("INIT",2,0/1)` checkpoint nodes (preferably via
  the real `$$NEWCP^XPDUTL` / `PKG^XPDIL1` path, not a hand-SET) so `IN^XPDIJ1`'s
  `PRE`/`POST` loops fire `D @XPDRTN`. Result: a build's data migration / xref
  rebuild actually runs.
- **A.1.2 — Env-check + required-builds.** Before `EN^XPDIJ`, set `XPDENV=1` and call
  **`$$ENV^XPDIL1(1)`** (which runs the build's env-check routine *and* `REQB`
  enforcement in one). Honor `XPDQUIT` / `XPDABORT` / `XPDREQAB` → convert to a
  structured `<<VPKG>>error=…` and refuse to file. The inputs it needs
  (`^XTMP("XPDI",XPDA,"PRE")` env-check name; the `BLD…REQB` nodes) are already in
  the direct-populated transport.
- **A.1.3 — Question answers.** Add an `install-spec` answer map; for each answer
  seed `^XPD(9.7,XPDA,"QUES",IEN,0/1/"B")` (+ the mandatory `"QUES","B"` xref) —
  ideally via FileMan filing to subfile 9.701 so the xref builds itself. Standard
  `XPI1`/`XPZ1`/`XPO1` stay unseeded = safe-NO by default; build-defined pre/post
  questions become available to their routines via `XPDQUES`.
- **A.1.4 — (Track A.3) PACKAGE #9.4 footprint.** Write VERSION (#22) + PATCH
  APPLICATION HISTORY so downstream `$$PATCH^XPDUTL` stays honest. Sequenced after
  A.1.1–A.1.3 (tracked separately as A.3).

**Future, cleaner option to revisit:** if the SDK gains a native `SetGlobal` (and/or
a stdin-capable `Exec`), the *most* faithful path is to write the `.KID` to a host
file and call the **real `EN1^XPDIL` load** (which natively creates checkpoints +
runs the load-time env-check), eliminating the reconstruction in A.1.1. Out of
scope now (no SDK change available); route (c) needs none.

---

## Waterline & bespoke-installer compliance
Route (c) stays inside all four waterline rules: every engine touch is through
`mdriver.Client`; KIDS knowledge stays in v-pkg (the `v` layer); no transport is
hand-rolled. It is the **opposite of a bespoke installer**
([[bespoke-installer-forbidden]]) — it *adds* real KIDS phase calls
(`$$ENV^XPDIL1`, `$$NEWCP^XPDUTL`, `IN^XPDIJ1` via `EN^XPDIJ`, the build's own
`@XPDRTN`) the current shortcut omits. The only synthesized artifacts are *data
nodes KIDS reads* (#9.7 checkpoints / QUES answers), seeded to match what the real
load would write — the identical, already-accepted pattern as the KRN/FIA seeds.

## Risks & open questions
- **Live validation is mandatory before relying on any of this.** Every claim here
  is from routine source + docs; confirm on **vehu (YDB)** and **foia-t12 (IRIS)**
  via the driver stack, same discipline as §7.1. A pre/post routine that aborts
  (`S XPDABORT=1`) leaves options disabled with **no cleanup** (documented) — the
  tool must surface that loudly.
- **IRIS parity (P7).** `XPD*` are Kernel routines present on both engines, but
  checkpoint/`^XTMP` and `$$ENV` behavior under IRIS must be re-proven, not assumed.
- **`XPDQUES` phase-scoping.** The DG says `XPDQUES` is populated *only during*
  pre/post-install; A.1.3's seeded answers must land where `QUES^XPDIQ` reads them
  (`^XPD(9.7,XPDA,"QUES",…)`), verified live.
- **Idempotency / half-installs.** Keep the existing already-installed guard and the
  corrupt-#9.7 purge (§7.1 gotcha); A.1.2 aborts must leave a clean, restartable
  state.

## Live confirmation (2026-06-28) — A.1.1 mechanism verified on BOTH engines
Confirmed via the driver stack (`m vista exec --engine ydb|iris`, never raw
`docker exec`) on **vehu (YDB GT.M)** and **foia-t12 (IRIS, namespace VISTA)** —
identical on both:

- **`$$NEWCP^XPDUTL(name, callback, params)` exists** with the same source. It reads
  `XPDCP` + `XPDA` from scope, picks subfile **9.713** (pre, `XPDCP="INI"`) /
  **9.716** (post, `XPDCP="INIT"`), FileMan-files (`UPDATE^DIE`) `.01`=name,
  **field 2 = callback routine**, field 3 = params, is **idempotent** (returns the
  existing IEN via `$$FIND1^DIC`), and returns the checkpoint IEN.
- **Checkpoint grammar** (real example, both engines — KERNEL 8.0, #9.7 IEN 2):
  `^XPD(9.7,2,"INIT",2,0)="XPD POSTINSTALL STARTED^<FMtime>"`, `(2,1)="^XUINEND"`
  (the post-install routine). The base `"…COMPLETED"` checkpoint (#1) carries no
  routine; the `"…STARTED"` checkpoint (created only when the build has a pre/post
  routine) carries it at node `(c,1)`, which `PRE^/POST^XPDIJ1` reads and `D @`s
  (skipping it when `(c,1)=""` or the completion time `$P((c,0),U,2)` is set).
- A build with no pre/post routine has only the `COMPLETED` checkpoint (verified on
  XU\*8.0\*1, #9.7 IEN 3).

**So A.1.1 is exactly:** after the MERGE (XPDA in scope, `DUZ(0)="@"`), and before
`EN^XPDIJ`, mirror `PKG^XPDIL1`'s checkpoint block — `S XPDCP="INI"` then
`$$NEWCP^XPDUTL("XPD PREINSTALL COMPLETED")` and, iff `^XTMP("XPDI",XPDA,"INI")]""`,
`$$NEWCP^XPDUTL("XPD PREINSTALL STARTED",<that routine>)`; likewise `XPDCP="INIT"`
+ `…POSTINSTALL…` from `^XTMP("XPDI",XPDA,"INIT")`. These transport nodes are
already present after v-pkg's MERGE for any build that ships pre/post routines.
Calls the **real** `$$NEWCP^XPDUTL` (not a reimplementation) → inside the waterline.

## A.1.1 — DONE + live-proven on BOTH engines (2026-06-28)
Implemented in `internal/installspec/script.go` (`FinalInstallScript`): after the
MERGE / before `EN^XPDIJ`, set `XPDCP` and call the real `$$NEWCP^XPDUTL` for the
`INI`/`INIT` `COMPLETED` checkpoints plus the `STARTED` checkpoints carrying the
pre/post routine names read from `^XTMP("XPDI",XPDA,"INI"/"INIT")` (only when the
transport carries one). TDD: `TestFinalInstallScript_PrePostCheckpoints`;
lint/race/contract green.

**Live-proven through the real `v pkg install` path over the driver stack** with a
fixture build (`testdata/zza1-prepost/`) that ships `ZZA1P` with `PRE`/`POST`
entries setting `^ZZA1OUT("PRE"/"POST")`:

| Engine | binary | install | `^ZZA1OUT("PRE")` / `("POST")` |
|---|---|---|---|
| vehu (YDB) | **pre-A.1.1** | ok, status 3 | **empty / empty** — routines silently skipped (F2) |
| vehu (YDB) | **A.1.1** | ok, status 3 | **1 / 1** — both fired |
| foia-t12 (IRIS) | **A.1.1** | ok, status 3 | **1 / 1** — both fired |

The counterfactual is the proof: the same fixture on the same engine reports a
"successful" install (status 3) under the old binary yet never runs the routines —
A.1.1 is exactly what makes them fire, identically on YDB and IRIS. (Driver note:
the YDB `exec load` needs `M_YDB_ROUTINES` set to the container's source dir, e.g.
`/home/vehu/r`; IRIS needs `M_IRIS_NAMESPACE=VISTA`.)

## A.1.2 — DONE + live-proven on BOTH engines (2026-06-28)
Implemented in `internal/installspec/script.go` (`FinalInstallScript` gained a
`runEnvCheck bool`) + plumbed through `pkgcli` (`runInstall`/`liveInstall`/
`installCmd` with a new `--skip-env-check` flag; restore/back-out callers pass
`false`). After the MERGE / before the KRN-FIA-checkpoint-`EN^XPDIJ` filing block,
the script reconstructs the minimal install-phase scope `EN^XPDI` sets (XPDI.m:11)
— `XPDNM`, `XPDPKG`, and the **seeded `XPDT`** (without which `$$ENV`'s own tail
`'$O(XPDT(0))` self-rejects a clean build) — then calls the **real**
`$$ENV^XPDIL1(1)`, which runs the build's env-check routine (`^XTMP("XPDI",XPDA,
"PRE")`) and `REQB^XPDIL1` Required-Build (#9.611) enforcement. On a non-zero
reject it **purges the aborted `#9.7` entry** (`K ^XPD(9.7,"B",XPDNM,XPDA),
^XPD(9.7,XPDA)`, so a corrected retry is clean) and refuses to file with
`error=env-check-rejected^<rc>^<XPDREQAB>`. Invokes KIDS, never reimplements it
(route (c), inside the waterline + bespoke-installer ban). TDD:
`TestFinalInstallScript_EnvCheck` (run vs skip + ordering before `EN^XPDIJ`);
lint/race/contract green.

**Live-proven through the real `v pkg install` path over the driver stack** with a
required-build fixture (`testdata/zza2-reqb/`): `ZZA2.kids` declares an unmet
Required Build `ZZNOPE*1.0*1` (action "DON'T INSTALL, LEAVE GLOBAL" = #9.611 code
2); `ZZA2-ok.kids` is the same routine with no requirement.

| Engine | bogus required build (`ZZA2.kids`) | control (`ZZA2-ok.kids`) | A.1.1 regression (`ZZA1`, env-check ON) |
|---|---|---|---|
| vehu (YDB)     | **REFUSED** `env-check-rejected^2^2`, `#9.7` auto-purged (9.7B=0) | status 3 | `^ZZA1OUT PRE/POST=11` (status 3) |
| foia-t12 (IRIS)| **REFUSED** `env-check-rejected^2^2`, `#9.7` auto-purged (9.7B=0) | status 3 | `^ZZA1OUT PRE/POST=11` (status 3) |

`--skip-env-check` installs the bogus build anyway (status 3) — the bypass works,
and the A.1.1 regression confirms env-check-on did not break pre/post firing.

> **Scope note.** A.1.2 is proven here via **Required-Build enforcement** (no
> routine execution needed; `emitRequiredBuildManifest` already ships the #9.611
> nodes). The env-check **routine** path runs on the same `$$ENV` call but was not
> exercised at the time of writing. **It is now — see B.3 below**: the rejecting
> env-check routine `ZZA3RE` is SAVEd (`^%ZOSF("SAVE")`) and run by `ENV^XPDIL1`,
> sets `XPDABORT=1`, and `$$ENV` returns 1 → install refused `env-check-rejected^1^`,
> nothing filed, on both engines.

> **Follow-up finding — FIXED 2026-06-28.** `internal/kids/reversibility.go`'s
> `installRoleNames` map had the transport keys **`INI`/`PRE` swapped** vs ground
> truth. Corrected against the authoritative live **#9.6 DD** (field 913 ENVIRONMENT
> CHECK ROUTINE→`PRE`, 916 PRE-INSTALL ROUTINE→`INI`, 914 POST-INSTALL→`INIT`, 900
> PRE-TRANSPORTATION→`PRET`) — matching what `ENV^XPDIL1` (reads `"PRE"` as env-check)
> and `PRE^/POST^XPDIJ1` (read `"INI"/"INIT"`) do live. Display-only (classification
> keys on subnode *presence*, not the role name), so no class verdict changed; the
> `v pkg classify` `InstallCode` labels are now correct. Regression test
> `TestInstallCodeRoleLabels`.

## B.3 — DONE + live-proven on BOTH engines (2026-06-28)

`v pkg build` now AUTHORS the three install-hook routines from the build spec, so
the A.1.1/A.1.2 fixtures are reproducible **end-to-end with no hand-injection**:
`buildspec` gained `envCheck` (already), `preInstall`, `postInstall`;
`kids.emitInstallHooks` writes the top-level `"PRE")`/`"INI")`/`"INIT")` transport
nodes (mirrored under `"BLD",1,…`), unset hooks emit nothing (corpus DRIFT=0
preserved). Validation is shape-only — the routine may be shipped here or pre-exist
on the target. Fixture `testdata/zza3-hooks/` (`ZZA3` positive, `ZZA3R` rejecting
env-check), built via `v pkg build … --src src`.

| Engine | positive `ZZA3` (env-check pass + pre + post) | negative `ZZA3R` (rejecting env-check) |
|---|---|---|
| vehu (YDB)      | `^ZZA3OUT ENV/PRE/POST=1`, `#9.7` status 3 | **REFUSED** `env-check-rejected^1^`, nothing filed |
| foia-t12 (IRIS) | `^ZZA3OUT ENV/PRE/POST=1`, `#9.7` status 3 | **REFUSED** `env-check-rejected^1^`, nothing filed |

This closes the env-check-**routine** scope gap left by A.1.2 (positive sentinel
`ENV=1` proves the env-check routine RAN and passed; the rejecting routine proves
the reject arm). **Installer fix discovered here:** `FinalInstallScript` now
`K ^XTMP("XPDI",XPDA)` **before** the MERGE — a purged earlier install frees its
#9.7 IEN, `$$INST^XPDIL1` re-assigns it, and stale REQB nodes at that IEN survived
the MERGE and made `$$REQB` falsely reject (was the `^2^2` artifact). The real KIDS
load always starts from a clean node. Regression test
`TestFinalInstallScript_KillBeforeMerge`. **Gotcha:** env-check routine source must
be ASCII — `^%ZOSF("SAVE")` compiles it before running, and a non-ASCII comment
byte breaks the compile.

## A.1.3 — DONE + live-proven on BOTH engines (2026-06-28)

`v pkg install --answer NAME=VALUE` pre-answers a build's install questions so a
pre/post-install routine reads them via the real `$$ANSWER^XPDIQ` — the
non-interactive equivalent of the `EN^XPDIQ` question phase the direct-populate
path skips. Ground truth (from `ANSWER^XPDIQ`): the answer lives at
`^XPD(9.7,XPDA,"QUES",IEN,1)`, found via the `"QUES","B",NAME,IEN` xref. The
`"QUES"` subtree is **internal install scratch — not a #9.7 FileMan field** (no
`*QUEST*` field in the DD), so the seed is exactly three nodes per answer (`,IEN,0)`
name, `,IEN,1)` value, `"B",name,IEN)` xref) — no multiple header. `FinalInstallScript`
seeds them after the env-check and before `EN^XPDIJ` runs the pre/post routines;
`installspec.QuesAnswer` + the repeatable `--answer` flag (`parseAnswers` splits on
the first `=`, order-preserved → deterministic IENs) carry them.

| Engine | `--answer ZZA4Q=HELLO` | control (no `--answer`) |
|---|---|---|
| vehu (YDB)      | `^ZZA4OUT("Q")="HELLO"`, `#9.7` status 3 | `^ZZA4OUT("Q")=""` (default) |
| foia-t12 (IRIS) | `^ZZA4OUT("Q")="HELLO"`, `#9.7` status 3 | `^ZZA4OUT("Q")=""` (default) |

Fixture `testdata/zza4-ques/` (`ZZA4P` post-install does
`S ^ZZA4OUT("Q")=$$ANSWER^XPDIQ("ZZA4Q")`). The counterfactual proves the seed
reached `$$ANSWER^XPDIQ`. This closes the A.1 install-fidelity track (A.1.1
pre/post · A.1.2 env-check · A.1.3 questions).

## Recommended next steps
- The A.1 install-fidelity track + the `reversibility.go` role-label fix are
  complete. Remaining work is Track B (authoring) — see
  `v-pkg-kids-coverage-analysis.md`.

## Sources
- Real `XPD*` routine source: WorldVistA/VistA-M `Packages/Kernel/Routines/`
  (`XPDIJ`, `XPDIJ1`, `XPDI`, `XPDI1`, `XPDIL`, `XPDIL1`, `XPDIP`, `XPDIQ`,
  `XPDID`).
- `XU/krn_8_0_dg_kids_ug` (Developer's Guide: KIDS UG) — `XPDENV` (Table 6),
  `XPDQUIT`/`XPDABORT` (Table 7), `XPDDIQ` `XPZ1`/`XPZ2` (Tables 8–9), `XPDNOQUE`,
  `XPDQUES`, `required-builds` (Table 15), `EN^XPDIJ` (ICR 2243); the "Developer
  Tools" material is a section of *this* guide, not a separate UG.
- `XU/krn_8_0_sm_kids_ug` — the three install phases, re-answering questions.
- v-pkg: `internal/installspec/script.go`, `pkgcli/lifecycle.go`;
  `docs/kids-installation-automation.md` §7.1 (live ZZSKEL ground truth).
