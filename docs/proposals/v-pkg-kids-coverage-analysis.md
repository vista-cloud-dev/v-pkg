---
title: "v-pkg adversarial coverage analysis — what it takes to build and install any of the ~2,400 real KIDS distributions"
status: proposed
created: 2026-06-28
last_modified: 2026-06-29
revisions: 2
doc_type: [PROPOSAL, ANALYSIS]
grounding:
  - "vdocs GOLD corpus — Kernel 8.0 KIDS Developer's Guide (XU/krn_8_0_dg_kids_ug) + Systems Management KIDS UG (XU/krn_8_0_sm_kids_ug) + VistA Build Analyzer UG (XU/vista_build_analyzer_ug)"
  - "WorldVistA/VistA KIDS corpus — 2,404 .KID/.KIDS distributions across 157 packages (~/data/kids-patches/), re-tallied with corrected node probes"
  - "real VistA code — ^DD/^DIC node grammar extracted from real KIDS file exports (FileMan #1.008, #8992.7, #.114) under ~/data/kids-patches/"
  - "MSL / VSL builds — m-stdlib/kids/std.build.json (MSL*0.1*1, 39 routines) + v-stdlib/kids/vsl.build.json (VSL*1.0*15: routines + #999001 VSL AUDIT + param-def + MSL required build)"
  - "v-pkg source — internal/buildspec, internal/kids, pkgcli, internal/installspec, pkgcli/commands.go verb surface (HEAD 2026-06-29)"
related:
  - kids-corpus-findings.md
  - patch-existing-routines-proposal.md
  - kids-installation-automation.md
  - fileman-dd-install-plan.md
  - "../v-stdlib/docs/proposals/v-stdlib-remediation-plan.md (R3 — the multi-field VSL AUDIT DD that triggered this analysis)"
---

# v-pkg coverage analysis: building and installing any real KIDS package

This is an adversarial review of v-pkg measured against a single, demanding bar:
**could it build and install any of the ~2,400 KIDS distributions that real
VistA packages actually ship?** The trigger was a concrete wall — v-stdlib's R3
needs a multi-field `VSL AUDIT` FileMan file, and v-pkg can only ship a
single-`.01` test-range file — but the right question is the general one, so
this document answers that and slots R3 in as one instance (§8).

Three evidence bases ground every claim: the **VA KIDS documentation** (the
authoritative target), the **2,404-distribution WorldVistA corpus** (what
packages really ship, by frequency), and the **real `^DD` node grammar** from
on-disk KIDS exports (what an installer must actually emit). v-pkg's own source
is the fourth.

## Contents

- [Verdict](#verdict)
- [Method and grounding](#method-and-grounding)
- [The target: the complete KIDS model](#the-target-the-complete-kids-model)
- [What v-pkg authors today](#what-v-pkg-authors-today)
- [Adversarial findings (F1–F8)](#adversarial-findings-f1f8)
- [Corrected corpus frequencies](#corrected-corpus-frequencies)
- [Coverage matrix — the corpus, MSL, and VSL](#coverage-matrix--the-corpus-msl-and-vsl)
- [Lifecycle coverage — the full KIDS build lifecycle](#lifecycle-coverage--the-full-kids-build-lifecycle)
- [Recommendations — two tracks to 100%](#recommendations--two-tracks-to-100)
- [The R3 case: a real multi-field DD](#the-r3-case-a-real-multi-field-dd)
- [Coverage accounting](#coverage-accounting)
- [Risks and open questions](#risks-and-open-questions)
- [Sources](#sources)

---

## Verdict

**v-pkg authors ~4 of the ~24 KIDS component types and can faithfully build and
install roughly the routine-only ~28% of real distributions.** The build path
emits Routines, one entry type (#8989.51 PARAMETER DEFINITION), a toy single-`.01`
FileMan file, and Required-Build *declarations*. Everything else KIDS can
transport — OPTION, SECURITY KEY, PROTOCOL, RPC, the template/form family,
MAIL GROUP, HL7, multi-field files, file **data**, install-time code — is either
schema-stubbed-but-silently-dropped, round-trip-only, or absent.

The sharpest finding is a **reframing of the goal** (F8). "Install any of the
2,000 packages" is mostly not an *authoring* problem: those packages already
exist as `.KID` files, and v-pkg's `decompose`/`assemble` already round-trips all
2,404 of them losslessly (DRIFT=0). The blocker for installing them is that the
**install path bypasses real KIDS load semantics** (F2) — it populates `^XTMP`
and calls `EN^XPDIJ` directly, so it never runs the build's Environment Check
(13% of the corpus), Pre/Post-Install routines (5% / 12%), install questions, or
Required-Build enforcement (79%). So the work splits cleanly into two tracks:

- **Install fidelity** — to install the ~2,400 *existing* distributions, run the
  real KIDS phases against a faithfully-loaded transport global. This is the
  dominant lever for the stated goal.
- **Authoring coverage** — to *build new* packages (VSL, including R3), emit the
  remaining component types from source. This is what unblocks v-stdlib.

A secondary but real finding: the committed corpus-frequency evidence
(`kids-corpus-findings.md`) is **wrong in several headline numbers** because the
analyzer's node probes mismatch the real transport layout (F5). Corrected
figures (below) reshuffle the authoring priority order.

---

## Method and grounding

| Evidence base | What it establishes | Locator |
|---|---|---|
| KIDS GOLD docs | The 100% target: every component type + the install engine's phases/mechanics | `XU/krn_8_0_dg_kids_ug`, `XU/krn_8_0_sm_kids_ug`, `XU/vista_build_analyzer_ug` |
| WorldVistA corpus (2,404 dists) | Real-world frequency → priority order | `~/data/kids-patches/` (re-tallied, §[Corrected frequencies](#corrected-corpus-frequencies)) |
| Real `^DD`/`^DIC` exports | The exact node grammar an emitter must produce | KIDS files #1.008, #8992.7, #.114 (§8) |
| v-pkg source | Current authoring + install surface | `internal/{buildspec,kids,installspec}`, `pkgcli/` |

**Note on the corpus re-tally.** The prior `analyze.py` keyed FileMan-entry
detection off a top-level `"KRN",<file>,<ien>,0)` node and `"REQB"` as a bare
substring. The real transport stores components under
`"BLD",<ien>,"KRN",<file>,"NM",…)` and dependencies under `"BLD",<ien>,"REQB",1,0)`.
The mismatch made the committed findings undercount OPTION (13%→**39%**), report
Required-Builds at 96% (real **79%**) and multi-build at 0% (real **3.66%**),
and omit PRINT TEMPLATE / FORM / SECURITY KEY from the entry ranking entirely.
The corrected probes (and the authoritative file#→name map from the Build
Analyzer UG) are used throughout; see F5.

---

## The target: the complete KIDS model

KIDS stores each application version in the **BUILD (#9.6)** file; its
**Build Components** multiple can transport these entity classes (file numbers
from the VistA Build Analyzer UG, *All Components*):

| Component | File # | Install action set |
|---|---|---|
| ROUTINE | 9.8 | load; skip/delete per-site via `$$RTNUP^XPDUTL` |
| FILE (DD ± data) | any | full/partial DD; data via 4 action codes (below) |
| OPTION | 19 | SEND / DELETE / **USE-AS-LINK / MERGE-ITEMS / ATTACH / DISABLE** |
| PROTOCOL | 101 | same extended menu-action set as OPTION |
| SECURITY KEY | 19.1 | SEND / DELETE |
| MAIL GROUP / BULLETIN | 3.8 / 3.6 | SEND / DELETE |
| PRINT / SORT / INPUT TEMPLATE | .4 / .401 / .402 | SEND / DELETE (compiled templates rebuilt) |
| FORM / BLOCK | .403 / .404 | SEND / DELETE |
| FUNCTION / DIALOG | .5 / .84 | SEND / DELETE |
| HELP FRAME | 9.2 | SEND / DELETE |
| LIST TEMPLATE | 409.61 | SEND / DELETE |
| HL7 APP PARAMETER / HLO REGISTRY / LOGICAL LINK / LLP | 771 / 779.2 / 870 / — | SEND / DELETE |
| REMOTE PROCEDURE | 8994 | SEND / DELETE |
| PARAMETER DEFINITION / TEMPLATE | 8989.51 / 8989.52 | SEND / DELETE |
| (auth family) ENTITY/ACTION/POLICY/LOCK DICT | 1.5 / 1.61 / 1.6 / 8993 | SEND / DELETE |

**Install engine — three phases** (Systems Management KIDS UG):

1. **Load** (`XPD LOAD DISTRIBUTION`): make an INSTALL (#9.7) entry per transport
   global, load into `^XTMP`, **run Environment Check (1st time)**.
2. **Questions** (`XPD INSTALL BUILD`): **Environment Check (2nd time)** — may
   cancel-this/cancel-all → pre-install questions → the 3 standard questions
   (rebuild menu trees / inhibit logons / disable options-protocols) →
   post-install questions → device/queue.
3. **Install**: disable requested options/protocols → delay → **pre-install
   routine** → **install components** → **post-install routine** → re-enable.

Mechanics an installer must honor:
- **Environment Check** (`XPDENV` var: 0=load, 1=install) can abort via
  `XPDQUIT`/`XPDABORT`; verifies prerequisites (`$$PATCH^XPDUTL`, `$$VER^XPDUTL`).
- **Pre/Post-Install** routines run in Phase 3 around component filing; abort in
  pre-install (`S XPDABORT=1`) leaves options disabled — no auto-cleanup.
- **Required Builds (#9.611)** verified at the site against #9.4 → VERSION (#22) →
  PATCH APPLICATION HISTORY (#9.49,1105): **WARNING ONLY** / **DON'T INSTALL,
  LEAVE GLOBAL** / **DON'T INSTALL, REMOVE GLOBAL**.
- **DD update**: unconditional if the file is new; if it exists, gated by the
  build's *Update DD* flag + *Screen to Determine DD Update*.
- **DATA action codes** (per file): **ADD-ONLY-IF-NEW / MERGE / OVERWRITE /
  REPLACE** (precise match + null-handling semantics differ; REPLACE deletes old
  XRefs, MERGE preserves non-null site values). "If you send data you must send
  FULL DD."
- **Package link (#9.4)**: KIDS records the install in VERSION (#22) +
  PATCH APPLICATION HISTORY, checks duplicate version/patch names.
- **Restart/checkpoints**: each phase checkpointed; `XPD RESTART INSTALL` resumes.

(Corpus does *not* document a private routine catalog beyond the public `XPD*`
entry points, the byte-level `^XTMP` layout, a "DIFQ" artifact, an "UPDATE" data
action, or a hard out-of-sequence rejection rule — those are not real KIDS
line-items and must not become spec.)

---

## What v-pkg authors today

| KIDS capability | v-pkg | Evidence | Note |
|---|---|---|---|
| ROUTINE | **FULL** | `internal/kids/buildkids.go:56`; install `installspec/script.go:74` | deterministic checksums; live-proven |
| PARAMETER DEFINITION #8989.51 | **FULL** | `internal/kids/krncomp.go:53` | the only entry type emitted |
| FILE — DD | **FULL** | `internal/kids/filecomp.go`; `buildspec.go` | `.01` + 5 typed fields (B.2-a); any positive file # (out-of-range needs explicit globalRoot) — live-proven both engines |
| FILE — DATA | **FULL** | `filecomp.go` `emitFileRecords` | records ship under `("DATA",file,ien,node)`; the 4 action codes a/m/o/r (B.2-b) — live-proven both engines |
| OPTION/KEY/PROTOCOL/RPC/TEMPLATE/MAILGROUP/HL7 | **NONE** | `buildspec.go:60–71` accepts, `build.go` ignores | **silently dropped** — see F1 |
| Required Builds | **PARTIAL** | manifest `krncomp.go:120`; install path runs no check | declared, **not enforced** — F4 |
| Package #9.4 link / patch history | **FULL** | `installspec/script.go` footprint; `--register-package` | A.3: $$PKGVER/$$PKGPAT after filing — live-proven both engines (≤4-char prefix for $$PATCH honesty) |
| Environment Check | **NONE** | `buildspec.go:44` accepted, never emitted/run | F2 |
| Pre / Post-Install routine | **NONE** | not emitted; `EN^XPDIJ` runs whatever is staged | F2 |
| Install questions (DIFQ/`XPDQUES`) | **PARTIAL (modeled, unwired)** | `installspec/installspec.go:36` defined, `lifecycle.go` doesn't consume it | F2 |
| Data action codes (a/m/o/r) | **FULL** | `filecomp.go` `fileSendOpts`; `buildspec.go` `DataActionCode` | send-opts p7/p8 per #9.64,222.7/222.8 (B.2-b) |
| Version/patch + already-installed guard | **PARTIAL** | `script.go:82` | refuses re-install; no patch-sequence check |
| Uninstall / reversibility | **PARTIAL (good, routine-centric)** | `reversibility.go`, `lifecycle.go:611` | strong static class model; pre-image = routines only |
| **Round-trip** any existing `.KID` | **FULL (opaque)** | `decompose.go` / `assemble.go` | carries *all* types as ZWR — but authors nothing |

**Net authoring surface: Routines + #8989.51 + toy-FILE + REQB-declaration.**
The install reaches the engine only through `mdriver.Client` (waterline rule 3,
`pkgcli/lifecycle.go:53`) — correct — but **bypasses `EN1^XPDIL`**: it stages the
parsed pairs into `^XTMP("VPKGI")`, MERGEs into `^XTMP("XPDI",XPDA)`, seeds the
KRN/FIA flags, and calls `EN^XPDIJ` directly (`installspec/script.go:69–110`).
Real `XPDIK`/`XPDIJ` does the filing; the entire *load/dialog/env-check/question/
required-build* layer is replaced by this custom populate.

---

## Adversarial findings (F1–F8)

**F1 — Declared components are silently dropped (correctness bug, red-gate-worthy).**
`buildspec` accepts `options/keys/protocols/rpcs/templates/mailGroups/hl7`
(`buildspec.go:60–71`), validates their namespace shape, then `build.go` emits
only `.Routines/.ParameterDefinitions/.Files`. A developer who writes
`"options": ["FOO MENU"]` gets a build with **no option and no error** — an
incomplete build that installs "successfully," manufacturing false confidence.
This violates the org's own source-tag→registry→red-gate discipline. **Fix
first, cheaply:** a declared-but-unemittable component must be a hard build
error, not a silent drop.

**F2 — The install bypasses real KIDS load semantics (the deepest gap).** By
populating `^XTMP` and calling `EN^XPDIJ` directly, the install never runs the
build's Environment Check (runs *twice* in real KIDS), Pre/Post-Install routines,
install questions, Required-Build enforcement, or the disable-options/inhibit-
logons safety steps. Consequence: even a side-effecting build v-pkg could
*author* would not install *faithfully*. For installing the existing corpus this
is **the** blocker — see F8.

**F3 — The FILE feature matches zero real usage.** Single `.01`, test-range,
**DD-only**. But in the corpus, of the 579 distributions that ship a FILE,
**every one ships DATA too** (DD-only = 0). So the current shape models a pattern
no real distribution uses, and multi-field (R3) is impossible.

**F4 — Required-builds: declared, never enforced.** 79% of distributions depend
on a prior build; v-pkg emits the #9.611 declaration but its bypassing install
runs no prerequisite check, so it will install out of order without complaint.

**F5 — The committed evidence base is wrong, and decisions rest on it.**
`kids-corpus-findings.md` (from the mis-probed `analyze.py`) undercounts OPTION
(13% vs **39%**), reports REQB 96% (vs **79%**) and multi-build 0% (vs
**3.66%**), and omits PRINT TEMPLATE / FORM / SECURITY KEY from the ranking. The
corrected order promotes SECURITY KEY and the template family. **Recommendation:**
fix `analyze.py`'s node probes and re-issue the findings; the priority order in
this document supersedes the old table.

**F6 — PACKAGE #9.4 link / patch history ✅ RESOLVED (A.3, 2026-06-28).** `v pkg
install --register-package` now stamps VERSION + PATCH APPLICATION HISTORY via
`$$PKGVER`/`$$PKGPAT^XPDIP` (live-proven both engines). The original finding stands:
real installs append VERSION (#22) + PATCH APPLICATION HISTORY; v-pkg previously
wrote none, so an install was invisible to later builds' `$$PATCH^XPDUTL` checks —
silently breaking the dependency chain.

**F7 — Multi-package builds ✅ RESOLVED (A.4, 2026-06-28).** `v pkg install` installs
each constituent in `**KIDS**`-header order, stopping on first failure (live-proven
both engines). The note below stands as the original finding. 3.66% of distributions are meta-builds
with an ordered install list; "any and all" requires them.

**F8 — "Install all 2,000 packages" is an install-fidelity problem, not an
authoring one (the reframing).** Those packages already exist as `.KID`s, and
v-pkg already round-trips all 2,404 losslessly. To *install* them, v-pkg does not
need new emitters — it needs to drive the **real KIDS load** (env-check / pre-post
/ questions / required-builds / data actions) against the faithfully-reconstructed
transport. The authoring emitters (F1/F3) matter for *building new* packages like
VSL. Conflating the two has hidden that the highest-leverage work for the stated
goal is **install fidelity (Track A)**, distinct from **authoring coverage
(Track B)**.

---

## Corrected corpus frequencies

Re-tallied over 2,404 distributions (corrected probes; file#→name per Build
Analyzer UG). This is the priority order.

**Component types shipped (by % of distributions):**

| File # | Component | dists | % |
|---|---|---:|---:|
| 9.8 | ROUTINE | 2286 | 95% |
| 19 | OPTION | 943 | **39%** |
| .4 | PRINT TEMPLATE | 489 | 20% |
| 19.1 | SECURITY KEY | 482 | 20% |
| 101 | PROTOCOL | 421 | 18% |
| .402 | INPUT TEMPLATE | 246 | 10% |
| 8994 | REMOTE PROCEDURE | 228 | 9% |
| 409.61 | LIST TEMPLATE (OE/RR) | 183 | 8% |
| 8989.51 | PARAMETER DEFINITION | 165 | 7% |
| 771 | HL7 APPLICATION PARAMETER | 132 | 5.5% |
| .401 | SORT TEMPLATE | 114 | 4.7% |
| 3.8 | MAIL GROUP | 108 | 4.5% |
| 870 | HL LOGICAL LINK | 103 | 4.3% |
| .403 | FORM | 78 | 3.2% |
| 8989.52 | PARAMETER TEMPLATE | 74 | 3.1% |
| 779.2 | HLO APPLICATION REGISTRY | 59 | 2.5% |
| 9.2 | HELP FRAME | 53 | 2.2% |
| .84 | DIALOG | 53 | 2.2% |
| 3.6 | BULLETIN | 50 | 2.1% |
| .5 | FUNCTION | 34 | 1.4% |

(plus a small tail: lexicon #9002226 84/3.5%, ENTITY #1.5, LOCK DICT #8993, …)

**Non-routine FileMan entries:** 1,206 distributions (**50%**) ship ≥1 entry
component (an OPTION/KEY/PROTOCOL/RPC/… — i.e. a permanent FileMan write), far
above the 23% first reported.
**Install-time code:** Environment Check 317 (13%) · Post-Install 293 (12%) ·
Pre-Install 109 (5%).
**FILE shipment:** 579 (24%) ship a FileMan FILE — **DD+data 334 (14%), data-only
245 (10%), DD-only 0**. 673 distinct file numbers; a long flat tail of
package-specific files.
**Dependencies:** real Required-Build dep 1,890 (**79%**); multi-build 88
(**3.66%**, up to 8 builds). Custom install questions are rare (~11); the
"questions" you see everywhere are the 3 standard boilerplate prompts.
**Reversibility:** with accurate entry detection, routine-only **674 (28%)**,
side-effecting **1,730 (72%)** — *lower* than the 35%/64% first reported, since
half the corpus ships an entry. (The committed `reversibility.go` classifier
still uses the older top-level-`KRN` probe → ~36% routine-only; aligning it is a
follow-up, not changed here.)

---

## Coverage matrix — the corpus, MSL, and VSL

Two questions, one matrix: **can v-pkg _author_ this from source (Track B)** and
**can v-pkg _install_ an existing `.KID` of it faithfully (Track A)**. Authoring
is per component type; install fidelity is per the build's phases (and applies to
*any* reconstructed transport, so it tracks Track A landing, not the type).

### Against the 2,404-distribution corpus, by component type

Frequencies from the corrected re-tally above; the two right columns are v-pkg's
state as of the work to date (B.1-a…o, B.2-a/b, B.3, A.1–A.4 all landed
2026-06-28).

| File # | Component | % dists | Authors (Track B) | Installs (Track A) |
|---|---|---:|---|---|
| 9.8     | ROUTINE              | 95%  | ✅ FULL                       | ✅ |
| 19      | OPTION               | 39%  | ✅ B.1-a (+menu B.1-o)        | ✅ |
| .4      | PRINT TEMPLATE       | 20%  | ⏳ deferred → `--from-engine` | ✅ |
| 19.1    | SECURITY KEY         | 20%  | ✅ B.1-c                      | ✅ |
| 101     | PROTOCOL             | 18%  | ✅ B.1-d (+item B.1-n)        | ✅ |
| .402    | INPUT TEMPLATE       | 10%  | ⏳ deferred → `--from-engine` | ✅ |
| 8994    | REMOTE PROCEDURE     | 9%   | ✅ B.1-e (+params B.1-m)      | ✅ |
| 409.61  | LIST TEMPLATE        | 8%   | ✅ B.1-g                      | ✅ |
| 8989.51 | PARAMETER DEFINITION | 7%   | ✅ FULL (B.1-b)               | ✅ |
| 771     | HL7 APPLICATION      | 5.5% | ✅ B.1-i                      | ✅ |
| .401    | SORT TEMPLATE        | 4.7% | ⏳ deferred → `--from-engine` | ✅ |
| 3.8     | MAIL GROUP           | 4.5% | ✅ B.1-f                      | ✅ |
| 870     | HL LOGICAL LINK      | 4.3% | ✅ B.1-j                      | ✅ |
| .403    | FORM / BLOCK (.404)  | 3.2% | ⏳ deferred → `--from-engine` | ✅ |
| 8989.52 | PARAMETER TEMPLATE   | 3.1% | ✖ not yet (declarative, small) | ✅ |
| 779.2   | HLO APP REGISTRY     | 2.5% | ✅ B.1-k                      | ✅ |
| 9.2     | HELP FRAME           | 2.2% | ✅ B.1-h                      | ✅ |
| .84     | DIALOG               | 2.2% | ⏳ deferred → `--from-engine` | ✅ |
| 3.6     | BULLETIN             | 2.1% | ⏳ deferred (likely declarative) | ✅ |
| .5      | FUNCTION             | 1.4% | ⏳ deferred → `--from-engine` | ✅ |
| any     | FILE (DD ± data)     | 24%  | ✅ B.2-a (DD) + B.2-b (data)  | ✅ |

Legend: ✅ done + live-proven both engines · ⏳ deferred (scoped, blocked on a
named follow-up) · ✖ unbuilt, unscoped. The **Installs** column is uniformly ✅
because Track A drives the *real* KIDS phases against any reconstructed transport
— it does not depend on v-pkg being able to author the type (F8). The authoring
gap is now exactly the **compiled-FileMan family** (templates / form / dialog /
function), all routed to
[`v-pkg-from-engine-capture.md`](v-pkg-from-engine-capture.md); plus PARAMETER
TEMPLATE #8989.52 (a small declarative addition, not yet built) and BULLETIN
(pending the declarative-vs-capture call).

### Against the two libraries that motivated this effort — MSL and VSL

These are the in-org packages v-pkg must build+install end-to-end. Both are
**100% covered today** — every component each ships is an authored, live-proven
type.

| Library | Build / spec | Components shipped | Authors | Installs |
|---|---|---|---|---|
| **MSL** (m-stdlib) | `MSL*0.1*1` — `m-stdlib/kids/std.build.json` | **39 routines** (STDARGS…STDXML); no entries, no files | ✅ FULL | ✅ FULL — live both engines |
| **VSL** (v-stdlib) | `VSL*1.0*15` — `v-stdlib/kids/vsl.build.json` | **6 routines** (VSLCFG/FS/IO/LOG/SEC/TASK) · **FILE #999001 `VSL AUDIT`** (`.01` + 4 typed fields: date / numeric / 2× free text) · **PARAMETER DEFINITION** `VPNG GREETING` (#8989.51, SYS entity) · **Required Build** `MSL*0.1*1` (DON'T INSTALL, LEAVE GLOBAL) | ✅ FULL | ✅ FULL |

VSL exercises four covered capabilities at once: routines, a **multi-field DD**
(B.2-a — the original R3 trigger, now a shipped reality), a parameter definition
(B.1-b), and a **required-build dependency** on MSL that Track A's enforcement
(A.2) honors at install. The `MSL → VSL` dependency direction matches the org
waterline (`v → m`); v-pkg installs MSL first, then VSL with its prerequisite
satisfied. Nothing either library needs is on the deferred list.

---

## Lifecycle coverage — the full KIDS build lifecycle

Component-type coverage (above) answers *what* v-pkg can carry. This answers
*which stages of a build's life* it can drive. v-pkg's 13 verbs
(`pkgcli/commands.go`) cover the whole arc; the engine-touching stages all go
through `mdriver.Client` (waterline rule 3).

| Stage | Verb(s) | What's covered | Residual gap | Status |
|---|---|---|---|---|
| **Assembly** (spec → transport) | `build` | Deterministic, normalized `.KID` from a declarative spec: routines, ~15 entry types, multi-field DD + data (4 action codes), install-time hooks (B.3); golden + corpus DRIFT=0 | compiled-template family (`--from-engine`); PARAMETER TEMPLATE #8989.52 | ◐ broad, one family deferred |
| **Disassembly** (transport → VC tree) | `decompose` · `assemble` · `roundtrip` · `canonicalize` | `.KID` ⇄ `KIDComponents/` tree, **all 2,404 corpus dists round-trip losslessly (DRIFT=0)** carrying every type opaquely as ZWR; semantic round-trip (line-2 canonicalized); IEN canonicalization for diff | round-trip is opaque (carries types it cannot author) | ✅ FULL (100% corpus) |
| **Inspection** | `parse` · `classify` · `lint` | Build/section summary; **reversibility class** derived statically (no engine); **PIKS data-class gate** (refuses PHI/Institution-class data into VC) | — | ✅ FULL |
| **Installation** | `install` | Real KIDS phases via route (c): load → `#9.7` entry → env-check → **Required-Build enforcement (A.2)** → seeded questions (`--answer`) → disable options → **pre-install routine** → file components → **post-install routine** (B.3 / A.1.1) → **#9.4 footprint (A.3)** → **multi-build header-order, stop-on-failure (A.4)**; class-aware overwrite guard + auto-snapshot | device/queue prompts not driven (headless `EN^XPDI` rejected); checkpoint *resume* (`XPD RESTART`) not modeled; custom (non-boilerplate) questions only pre-answered, not discovered | ◐ faithful filing + major phases |
| **Verification** | `verify` (`--drift`) | `#9.7` status-3; **routine drift** vs build source; **content assertion** for every entry type + param-def (live 0-node ^-piece compared to the shipped image, FileMan-transformed pieces masked) **and FILE DD field defs** (live `^DD(file,fld,0)` vs shipped `fieldDef`, filed verbatim) — not just `"B"`/`^DD(file,0)` presence (2026-06-29, **live-proven on vehu**: VSL param-def + all 5 `#999001` field defs) | entry verify is 0-node only (menu items / WP not yet compared); file's `^DIC(file,0)` name/GL still presence-only | ◐ content (entry 0-node + FILE DD, live) + routine-drift |
| **Back-out** | `uninstall` · `snapshot` · `restore` | **Class-aware** uninstall (routine + entry `DIK` delete; side-effecting **refused** without `--force`); **routine pre-image** capture (`snapshot`) + preview-gated `restore`; pre-image pairing (`--auto-snapshot` ↔ auto-detect); verify-clean; **`--deregister`** clears the PACKAGE #9.4 patch-history footprint (symmetric to `--register-package`) so `$$PATCH^XPDUTL` is honest after back-out (2026-06-29, live-proven — enables the negative Required-Build gate) | non-routine reversal is **by-design** the authored back-out's job (no generic inverse); #9.4 VERSION/CURRENT-VERSION footprint left intact (only patch history removed) | ◐ routine-centric; #9.4 patch-history deregister ✓ |

Legend: ✅ complete · ◐ substantial, with a documented residual.

**Reading the arc.** Disassembly and inspection are **complete and corpus-wide**
— v-pkg version-controls and classifies *any* of the 2,404 distributions today.
Assembly is broad with one deferred family. Installation drives the real KIDS
engine for the phases that matter (the F2 "bypass" is closed); the residuals are
interactive-device handling and restart-resume, neither on the path for
non-interactive package install. Back-out is routine-complete and, crucially,
**honest** about the side-effecting class it cannot generically reverse —
deferring that to an authored back-out rather than over-claiming
([[bespoke-installer-forbidden]]). No stage is a stub; every gap is named and
either scoped (templates) or a deliberate design boundary (non-routine reversal).

---

## Recommendations — two tracks to 100%

Sequenced by leverage. **Track A (install fidelity)** is the dominant lever for
"install any of the 2,000 existing packages"; **Track B (authoring coverage)** is
what builds new packages like VSL. Do **T0 first** — it is cheap and an integrity
fix.

### T0 — Integrity (do first; days) — ✅ DONE 2026-06-28
- **T0.1 Stop dropping declared components silently (F1). ✅** `buildspec.Validate`
  now rejects a populated `options/keys/protocols/templates/rpcs/mailGroups/hl7`
  slice with an error naming the type(s) — a confidence-destroying silent bug is
  now an honest "not yet supported" (TDD, buildspec cov 98.8%).
- **T0.2 Correct the evidence (F5). ✅** `analyze.py`'s probes fixed
  (`"BLD",ien,"KRN",file,"NM"`, `"REQB",1,0)`, `re.M` multi-build, 9.8 excluded
  from the side-effect set) + file#→name labels corrected; re-run to
  `analysis-report.txt`; `kids-corpus-findings.md` re-issued with the corrected
  tables above.

### Track A — Install fidelity (install the existing ~2,400)
The goal: take any reconstructed transport global and run it through *real* KIDS,
not the populate-and-`EN^XPDIJ` shortcut.
- **A.1 Drive the real load + phases (F2). ✅ SCOPED 2026-06-28 —
  [`v-pkg-install-fidelity-spike.md`](v-pkg-install-fidelity-spike.md).** The spike
  ground-truthed the KIDS phase boundary against real `XPD*` source: `EN^XPDIJ` is
  only the *filing* engine; env-check + required-builds (`ENV`/`REQB^XPDIL1`) and
  question-prompting (`EN^XPDI`) run earlier, and pre/post-install routines fire in
  `EN^XPDIJ` **only if** the load-phase `INI`/`INIT` checkpoints exist (which the
  direct-populate never creates — so they're silently skipped). **Recommended route
  (c): augmented direct-populate** — keep the proven core, add explicit calls to the
  *real* phase functions (`$$ENV^XPDIL1(1)`, `$$NEWCP^XPDUTL` checkpoints, seeded
  `#9.7` QUES answers). Route (a) "drive `EN^XPDI` headless" is rejected (the device
  + `XPO1`/`XPI1` prompts have no documented suppression and the driver `Exec` has no
  stdin); (b) expect stays the cross-engine fallback. **The flagged blocker is
  resolved:** the "KIDS Developer Tools UG" is a *section of* `krn_8_0_dg_kids_ug`
  (in the corpus), not a missing doc. Land A.1.1 (pre/post routines) first.
- **A.2 Enforce Required Builds (F4).** Run KIDS' #9.611 check (or replicate it
  against #9.4 → #22 → #9.49,1105) and honor WARNING / DON'T-INSTALL-LEAVE /
  DON'T-INSTALL-REMOVE before filing.
- **A.3 ✅ DONE + LIVE-PROVEN 2026-06-28 — PACKAGE #9.4 footprint (F6), both
  engines.** `v pkg install --register-package "<LONG NAME>"` writes the footprint
  via the REAL `$$PKGVER^XPDIP` (VERSION #9.49 + CURRENT VERSION #9.4 field 13) and
  `$$PKGPAT^XPDIP` (PATCH APPLICATION HISTORY #9.4901) after `EN^XPDIJ`, gated on
  #9.7 status 3 — find-or-creating the #9.4 entry by PREFIX (the `"C"` xref; KIDS
  itself never creates it, but an authored package must register itself). Makes
  downstream `$$PATCH^XPDUTL` honest. **KEY GOTCHA:** `$$LKPKG^XPDUTL` resolves a
  package by PREFIX only when `$L(prefix)<5` (else by NAME), and `$$PATCH`'s own
  install-name regex caps the prefix at **1-4 uppercase/numeric chars** — so the
  footprint is `$$PATCH`-honest only for a **≤4-char prefix** (a 6-char "ZZSKEL"
  filed fine but `$$PATCH` returned 0; re-proving with 3-char "ZZV" gave
  `$$PATCH(ZZV*1.0*3)` 0→1, negative `*99` stays 0, on both engines). Not hard-gated
  (the footprint still files; longer namespaces are valid for routines) —
  documented. Route (c): calls the real XPDIP, never reimplements KIDS (inside the
  waterline). TDD; lint/race/contract green. `internal/installspec/script.go`
  (`PkgReg`), `pkgcli/lifecycle.go` (`--register-package`, `packageReg`,
  `installResult.PackageIEN`). ([[package-footprint]])
- **A.4 ✅ DONE + LIVE-PROVEN 2026-06-28 — multi-package builds (F7), both engines.**
  `v pkg install` now installs a multi-build distribution: each constituent **in
  `**KIDS**`-header order** (which IS the install order — the header lists names
  `^`-separated in dependency order, preserved in `k.InstallNames`), **stopping at
  the first failure** (a failed prerequisite aborts the rest). Each constituent
  installs independently (its own #9.7 entry + transport) through the *same*
  class-aware `liveInstall` path — faithful to real KIDS (one `EN^XPDIJ` per
  constituent), no parallel installer (waterline). Build-specific flags are
  ambiguous across constituents, so `--register-package` and an explicit
  `--snapshot <path>` are refused; `--auto-snapshot` pairs each build to its own
  sidecar; `--answer`/`--allow-overwrite`/`--skip-env-check` apply to all. `MBREQ`
  is per-build metadata (doesn't block a standalone constituent install). VERIFY /
  UNINSTALL stay one-build-at-a-time (A.4 scope is install). Live-proven by merging
  two tiny builds (ZZM1+ZZM2) into one multi-build `.KID` (`WriteKID`'s multi-name
  header) — both reached status 3 in order on vehu + foia-t12. TDD
  (`installSequence` order + stop-on-failure via the fake driver); lint/race/contract
  green. `pkgcli/lifecycle.go`. ([[multi-build-install]])

### Track B — Authoring coverage (build new packages, incl. VSL)
- **B.1 A generic entry-component emitter.** Generalize the #8989.51 `KRN` path to
  every SEND-TO-SITE/DELETE-AT-SITE entry type and pack its `KRN` record image. One
  mechanism covers ~20 of the 24 types because they share SEND/DELETE semantics.
  Land in frequency order:
  **OPTION #19 → SECURITY KEY #19.1 → PROTOCOL #101 → RPC #8994 → templates
  (.4/.402/.401/.403) → LIST TEMPLATE #409.61 → MAIL GROUP #3.8 → HL7 family**.
  OPTION/PROTOCOL also need the extended actions (USE-AS-LINK / MERGE-ITEMS /
  ATTACH / DISABLE).
  - **B.1-a ✅ DONE + LIVE-INSTALL-PROVEN 2026-06-28 — generic emitter + OPTION
    (#19), both engines.** Extracted the generic `entryType`/`entryRec`/`imageNode`
    core (`internal/kids/entrycomp.go`: `emitEntryManifest`/`emitEntryData`/
    `entryNames`) from the proven #8989.51 path and landed **OPTION** on it (a
    run-routine option: `.01 NAME^MENU TEXT^^TYPE`, ROUTINE node 25, "U" xref; ORD
    action-routine line + `-1`=`0^1` ground-truthed as #19 national constants).
    **Design decision:** the image is **authored from a declarative `options` spec**
    (not read live via DBS) — keeps `v pkg build` offline + byte-deterministic
    (consistent with the param-def/file paths); the generic packer can take a
    read-live image source as a later alternate. TDD; lint/race/contract green;
    corpus DRIFT=0 (2,404); golden `testdata/zzoption/ZZOPTION.kids`. **Live
    install→verify→file-check→`--force` uninstall→clean on vehu (YDB) + foia-t12
    (IRIS)** via the driver stack. **Bug found+fixed by the live-prove:**
    `entryNames` must compare the file-number subscript **numerically** (int OR
    float) — integer file numbers (#19/#101/#8994) re-parse from a `.KID` as int, so
    an `IsFloat`-only probe passed unit tests but silently dropped every option on
    the live verify/uninstall path (`subNum()`; regression
    `TestBuild_OptionNames_AfterReparse`). ([[option-entry-component]])
  - **B.1-b ✅ DONE + LIVE-PROVEN 2026-06-28 — unified multi-type KRN manifest +
    param-def migrated onto the generic core, both engines.** One build can now ship
    several KRN entry types: `buildEntryGroups` collects all of them (file-number
    ascending) and a single `"BLD",1,"KRN",0)` header spans them
    (`^9.67PA^<max file#>^<type count>` — a deterministic stand-in for KIDS'
    insertion-order "last IEN", which a corpus build showed as `779.2` over 20 types;
    cosmetic to the install since KRN^XPDIK iterates the subscripts). PARAMETER
    DEFINITION (#8989.51) migrated onto the same `entryType`/`entryRec` core —
    **byte-identical** (unchanged `zzparam` golden, `TestMakeBuildPairs_ParamDef_KRN`,
    corpus DRIFT=0). The buildspec "can't-mix-options-and-paramDefs" guard is removed.
    **Live install→verify→uninstall→clean of a mixed OPTION + PARAMETER DEFINITION
    build** (`testdata/zzmix`, header `^9.67PA^8989.51^2`, option ORD 1 / param-def
    ORD 2, both types filed + verified + backed out) on vehu (YDB) + foia-t12 (IRIS).
    Next: **SECURITY KEY #19.1** (a one-`entryType` addition to `buildEntryGroups`).
    ([[option-entry-component]])
  - **B.1-c ✅ DONE + LIVE-PROVEN 2026-06-28 — SECURITY KEY (#19.1), both engines.**
    The third type on the generic core, and the proof that a new SEND/DELETE type is
    now ~30 lines (an `entryType` with its national ORD action-routine tail
    `;;KEY^XPDTA1;KEYF1^XPDIA1;KEYE1^XPDIA1;KEYF2^XPDIA1;;KEYDEL^XPDIA1`, a record
    packer, the `buildEntryGroups` line, a `KeyNames` reader, verify/uninstall
    threading). A key's record image is **minimal** — `-1`=`0^1` + `0`=the .01 NAME
    alone (stored in `^DIC(19.1,`); shipped **name-only** (the optional DESCRIPTION
    word-processing field is deferred). TDD; lint/race/contract green; corpus
    DRIFT=0; golden `testdata/zzkey/ZZKEY.kids`. **Live install→verify→`--force`
    uninstall→clean** on vehu (YDB, `^DIC(19.1,…,0)=ZZKEY MANAGER`) + foia-t12 (IRIS).
    Next: **PROTOCOL #101** (richer record image — items/types). ([[option-entry-component]])
  - **B.1-d ✅ DONE + LIVE-PROVEN 2026-06-28 — PROTOCOL (#101), both engines.** The
    fourth type. Mirrors OPTION's node skeleton (0-node `NAME^ITEM TEXT^^TYPE`, ENTRY
    ACTION node 20, EXIT ACTION node 15) but with its **own data global `^ORD(101,`,
    own TYPE set-of-codes** (A action / X extended action / M menu / E event driver /
    …, distinct from #19's), and **no "U" xref node**. ORD action-routine tail
    (57 corpus builds): `;;PRO^XPDTA;PROF1^XPDIA;PROE1^XPDIA;PROF2^XPDIA;;PRODEL^XPDIA`.
    Authored a **base protocol** (NAME + ITEM TEXT + TYPE + ENTRY ACTION); the #101.01
    ITEM multiple landed in B.1-n (extended menu-actions USE-AS-LINK / MERGE-ITEMS /
    ATTACH / DISABLE remain a minor follow-up). TDD; lint/race/contract green; corpus DRIFT=0; golden
    `testdata/zzproto/ZZPROTO.kids`. **Live install→verify→`--force` uninstall→clean**
    on vehu (YDB, `^ORD(101,…,0)=ZZPROTO ACTION^…^^A`, node 20=`Q`) + foia-t12 (IRIS).
    Next: **RPC #8994**. ([[option-entry-component]])
  - **B.1-e ✅ DONE + LIVE-PROVEN 2026-06-28 — REMOTE PROCEDURE (#8994), both
    engines.** The fifth type, and the simplest record yet: a single 0-node
    `NAME^TAG^ROUTINE^RETURNTYPE` in its own data global `^XWB(8994,` — no action/exit
    nodes, no "U" xref. ORD action-routine tail (corpus modal): `1;;;;;;;RPCDEL^XPDIA1`.
    `RPCComp` carries `tag`/`routine`/`returnType`; `returnType` is a human name
    resolved to its #8994 field .04 set-of-codes value (single value→1, array→2, word
    processing→3, global array→4, global instance→5), **defaulting to "single value"**
    when omitted (the field is DD-required). TDD; lint/race/contract green; corpus
    DRIFT=0; golden `testdata/zzrpc/ZZRPC.kids`. **Live install→verify→`--force`
    uninstall→verify-clean** on vehu (YDB) + foia-t12 (IRIS): live
    `^XWB(8994,…,0)=ZZRPC ECHO^ECHO^ZZRPCRT^1` **byte-identical on both**, B-index gone
    after back-out. (Input-parameter #8994.02 authoring landed in B.1-m.) Next:
    **template/form family** (#.4/.402/.401/.403). ([[option-entry-component]])
  - **B.1-f ✅ DONE + LIVE-PROVEN 2026-06-28 — MAIL GROUP (#3.8), both engines.** The
    sixth type. Bespoke-action-routine family (NOT DIFROM). Stored in `^XMB(3.8,`;
    record `-1)=0^1`, `0)=NAME^TYPE^SELF-ENROLL`. **TYPE (field 4) is DD-required**
    (set `PU:public/PR:private`, default `PU`); ALLOW SELF ENROLLMENT (field 7) is
    optional `y/n`. NM node plain `NAME^^0`. ORD tail
    `;;MAILG^XPDTA1;MAILGF1^XPDIA1;MAILGE1^XPDIA1;MAILGF2^XPDIA1;;MAILGDEL^XPDIA1(%)`.
    Shipped **member-less** (the #3.81 member list is site-local #200 pointers) and
    **description-less** (the field-3 WP header carries a volatile date — deferred,
    like KEY's). TDD; lint/race/contract green; corpus DRIFT=0; golden
    `testdata/zzmg/ZZMG.kids`. **Live install→verify→`--force` uninstall→verify-clean**
    on vehu (YDB) + foia-t12 (IRIS): live `^XMB(3.8,…,0)=ZZMG ALERTS^PU^y`
    byte-identical on both, B-index gone after back-out. ([[option-entry-component]])
  - **B.1-g ✅ DONE + LIVE-PROVEN 2026-06-28 — LIST TEMPLATE (#409.61), both engines.**
    The seventh type and the richest record yet — but all plain strings (no compiled
    structure), so it authors from a spec (unlike the FileMan templates). **Stored in
    `^SD(409.61,`, NOT `^ORD(`** (ground-truthed the GL node). Record: `-1)=0^1`, a
    fixed 14-piece 0-node (List Manager screen geometry + PROTOCOL MENU pointer +
    title) plus the callback nodes at string subscripts `HDR`/`INIT`/`FNL`/`HLP` (M
    code) + `ARRAY` (display global). Set-of-codes pieces pinned to the corpus dominant
    (TYPE OF LIST=1 PROTOCOL, etc.); margins default 80/3/20. ORD tail
    `1;;;;LME1^XPDIA1;;;LMDEL^XPDIA1`. The generic core already supported string-subscript
    image nodes — no core change. TDD; lint/race/contract green; corpus DRIFT=0; golden
    `testdata/zzlm/ZZLM.kids`. **Live install→verify→`--force` uninstall→verify-clean**
    on vehu (YDB) + foia-t12 (IRIS): live `^SD(409.61,…,0)=ZZLM PATIENTS^1^^80^3^20^1^1^^^ZZ
    Patient List^1^^1` + `…,"HDR")` byte-identical on both, B-index gone after back-out.
    ([[option-entry-component]])
  - **B.1-h ✅ DONE + LIVE-PROVEN 2026-06-28 — HELP FRAME (#9.2), both engines.** The
    eighth type and the first whose word-processing body IS the point (vs KEY's deferred
    optional WP). Stored in `^DIC(9.2,`. Record: `-1)=0^1`, `0)=NAME^HEADER`, and the
    TEXT word-processing field (field 2, subfile 9.21) at node 1 (header `^^<n>^<n>` +
    one node per line). ORD tail `;;HELP^XPDTA1;HLPF1^XPDIA1;HLPE1^XPDIA1;HLPF2^XPDIA1;;HLPDEL^XPDIA1`.
    Two determinism wins (the WP playbook): ship the WP header **date-less**, and **omit
    DATE ENTERED / AUTHOR** (FileMan auto-stamps both at install — proven live). Names
    allow hyphens AND spaces (own `reHelpName` regex). TDD; lint/race/contract green;
    corpus DRIFT=0; golden `testdata/zzhf/ZZHF.kids`. **Live install→verify→`--force`
    uninstall→verify-clean** on vehu (YDB) + foia-t12 (IRIS): live NAME^HEADER + the WP
    text byte-identical on both, B-index gone after back-out. ([[option-entry-component]])
  - **B.1 templates (#.4/.401/.402/.403) — DEFERRED 2026-06-28** (user decision). The
    transport mechanics generalize (one DIFROM ORD-tail covers all four + FUNCTION
    #.5/DIALOG #.84/BULLETIN #3.6), but the record image carries **compiled FileMan
    structures** (the `"DR"` edit string with embedded MUMPS, `"DIAB"` nodes, ScreenMan
    FORM/BLOCK subtrees) NOT derivable from a declarative spec. Needs a **read-live
    capture** image source (`--from-engine`); templates are its forcing function.
    **Scoped 2026-06-29: [`v-pkg-from-engine-capture.md`](v-pkg-from-engine-capture.md)**
    (one new image source, zero new transport; capture/build split for determinism;
    the DBS-vs-global-walk waterline decision is the gate). Remaining spec-derivable
    types: **HL7 family**. Detail: [[option-entry-component]].
  - **B.1-i ✅ DONE + LIVE-PROVEN 2026-06-28 — HL7 APPLICATION PARAMETER (#771), both
    engines.** The ninth type; the portable, fully spec-derivable member of the HL7
    family. Stored in `^HL(771,`. Record: `-1)=0^1`, `0)=NAME^a^FACILITY^^^^COUNTRY`
    (field 2 ACTIVE=`a`/INACTIVE=`i`; field 3 FACILITY NAME; field 7 COUNTRY CODE, a
    `#779.004` pointer that FileMan resolves from the shipped `"USA"` to its IEN at
    install). ORD tail `;;HLAP^XPDTA1;HLAPF1^XPDIA1;HLAPE1^XPDIA1;HLAPF2^XPDIA1;;HLAPDEL^XPDIA1(%)`.
    HL7 names allow spaces AND underscores (own `reHL7Name` regex; ≤30 chars). The
    `hl7Applications` build-spec key replaced the old unsupported `"hl7"` stub. TDD;
    lint/race/contract green; corpus DRIFT=0; golden `testdata/zzhl/ZZHL.kids`. **Live
    install→verify→`--force` uninstall→verify-clean** on vehu (YDB) + foia-t12 (IRIS):
    live 0-node `ZZHL_APP^a^500^^^^1` byte-identical on both (COUNTRY resolved to pointer
    `1`), B-index gone after `DIK` on `^HL(771,` back-out. HL7 follow-ups remain: #779.2
    (HLO message registry, multiples + xrefs) and #870 (logical link, hardcoded
    IP/hostname — site-specific, NOT portable). ([[option-entry-component]])
  - **B.1-j ✅ DONE + LIVE-PROVEN 2026-06-28 — HL LOGICAL LINK (#870), both engines.**
    The tenth type; the HL7 communication-endpoint. Stored in `^HLCS(870,`. Record:
    `-1)=0^1`, sparse `0)=NODE^^LLPTYPE` + optional `400)=^PORT^SVC`. ORD tail
    `1;;HLLL^XPDTA1;;HLLLE^XPDIA1;;;HLLLDEL^XPDIA1(%)` (piece 3 = `1`, the data-ships
    flag). Added a generic `caretJoin` sparse-node helper. **Two findings (live):**
    (1) the #870 install RE-FILES through FileMan (not a verbatim merge) — LLP TYPE
    ships external `"TCP"` and resolves to its `#869.1` IEN **4** at install (TCP=4 on
    BOTH engines, nationally controlled); (2) the network endpoint (DNS DOMAIN,
    TCP/IP ADDRESS) is **site config the install DROPS** (DNS input transform resolves
    the host and clears itself + the coupled IP; a bare IP is dropped too), so v-pkg
    ships only what lands — name, LLP type, PORT, SERVICE TYPE. This realizes the
    earlier "#870 is site-specific" note: the link *structure* is portable, the
    *endpoint* is not. TDD; lint/race/contract green; corpus DRIFT=0; golden
    `testdata/zzll/ZZLL.kids`. **Live install→verify→`--force` uninstall→verify-clean**
    on vehu (YDB, IEN 52) + foia-t12 (IRIS, IEN 103): live `ZZLINK^^4` + `^5000^C`
    identical on both, B-index gone after `DIK` on `^HLCS(870,`. Remaining HL7
    follow-ups: #779.2 (HLO registry — multiple #779.21 with computed `B`/`D` xref
    nodes) and #870 DESCRIPTION WP (#870.02). ([[option-entry-component]])
  - **B.1-k ✅ DONE + LIVE-PROVEN 2026-06-28 — HLO APPLICATION REGISTRY (#779.2), both
    engines.** The eleventh type and the **first with computed cross-reference nodes**.
    Stored in `^HLD(779.2,`. Record: `-1)=0^1`, `0)=APPNAME`, plus the #779.21 MESSAGE
    TYPE ACTIONS multiple (header `^779.21I^n^n`, data
    `MSGTYPE^EVENT^^TAG^RTN^VERSION`) with the emitter shipping each entry's computed
    xrefs. ORD tail `1;;HLOAP^XPDTA1;;HLOE^XPDIA1;;;` (no DEL routine). Added a generic
    `versionSub` (numeric subscript when canonical, e.g. `2.4`, string otherwise).
    **Xref rule (live-prove-corrected):** `"B"` on MSG TYPE always, plus EXACTLY ONE of
    `"D"` `(MSGTYPE,EVENT,VERSION)` when versioned / `"C"` `(MSGTYPE,EVENT)` when not —
    C and D are mutually exclusive. (Shipped C unconditionally first; install
    ground-truth showed #779.2 RE-INDEXES through FileMan and rebuilds B+D, dropping the
    stray C — corpus confirms native ships B+D.) TDD; lint/race/contract green; corpus
    DRIFT=0; golden `testdata/zzho/ZZHO.kids`. **Live install→verify→`--force`
    uninstall→verify-clean** on vehu (IEN 34) + foia-t12 (IEN 35): the full subtree
    (0-node, #779.21 entry, B + D xrefs) byte-identical to the shipped image on both,
    B-index gone after back-out. **This closes the HL7 family** (#771 + #779.2 + #870).
    (#779.2 multi-app batches: proven to work unchanged — the generic emitter packs N
    records with independent per-app #779.21 seq + per-entry C/D xref selection;
    regression-locked by `TestMakeBuildPairs_HLOApp_MultiApp` + a 2-app live install on
    both engines.) ([[option-entry-component]])
  - **B.1-l ✅ DONE + LIVE-PROVEN 2026-06-28 — optional DESCRIPTION WP for KEY #19.1,
    MAIL GROUP #3.8, HL LOGICAL LINK #870, both engines.** Closed the three deferred
    DESCRIPTION word-processing fields (B.1-c / B.1-f / B.1-j) with the date-less WP
    playbook. Added a shared `wpNodes(node, subfile, lines)` helper (header
    `^<subfile>^<n>^<n>` date-less + one node per line); refactored HELP FRAME's inline
    WP onto it (golden unchanged). Field→subfile→node: #19.1 f1→19.11@1, #3.8 f3→3.801@2,
    #870 f1→870.02@3. Each `*Comp` gained `description []string`. **Live finding:** the
    engines file the date-less header VERBATIM (live stays `^19.11^1^1`, no re-stamped
    date) — byte-identical in the live global too. TDD; lint/race/contract green; corpus
    DRIFT=0; goldens `testdata/zzkey|zzmg|zzll` regenerated. Live install→verify→`--force`
    uninstall→clean on vehu + foia-t12, headers + text byte-identical on both.
    ([[option-entry-component]])
  - **B.1-m ✅ DONE + LIVE-PROVEN 2026-06-28 — RPC INPUT PARAMETERS (#8994.02), both
    engines.** Closed the deferred RPC input-parameter authoring (from B.1-e). Added the
    optional #8994.02 multiple (field 2, subfile 8994.02A @ node 2): per-param
    `2,<seq>,0)=NAME^TYPE^MAXLEN^REQ^SEQNUM` + an optional nested DESCRIPTION WP
    (empty-subfile `^^n^n`) + the **"B" and "PARAMSEQ" cross-references the emitter
    ships itself**. **Key reason:** the #8994 install is a VERBATIM KRN MERGE (ORD tail
    has no re-file routines), so FileMan does NOT rebuild xrefs — the image must carry
    them. Generalized `wpNodes`→`wpNodesAt(prefix Subs, …)` for the nested per-param WP.
    `RPCParamComp{Name,Type,MaxLength,Required,Sequence,Description}` + `RPCParamTypeCode`
    map; type/seq default to literal/position. TDD; lint/race/contract green; corpus
    DRIFT=0; golden `testdata/zzrpc/ZZRPC.kids` regenerated (2 params). Live
    install→verify→`--force` uninstall→clean on vehu + foia-t12: the full param subtree
    (header, data nodes, date-less description WP, B + PARAMSEQ xrefs) **byte-identical**
    to the shipped image on both, B-index gone after back-out. ([[option-entry-component]])
  - **B.1-n ✅ DONE + LIVE-PROVEN 2026-06-28 — PROTOCOL ITEM multiple (#101.01), both
    engines.** Closed the deferred menu-item authoring (from B.1-d). Added the #101.01
    ITEM multiple (field 10, subfile 101.01PA @ node 10): per item a data node
    `10,<seq>,0)=<placeholder>^^<sequence>^` + a `10,<seq>,"^")=<CHILD NAME>` resolver
    node. **Decisive finding (broadly reusable):** KIDS transports a #101-pointer as
    the IEN slot + a parallel `"^"` NAME node, and a re-filing type (#101 runs
    PROF1/PROE1) **re-points from the name node** — the IEN slot is a build-local
    don't-care. Live-proven: shipped `1^^5^` + `"^")=ZZPROTO ACTION` became `7054^^5^`
    (vehu) / `5649^^5^` (foia-t12), re-pointed to the sibling ACTION's real
    engine-specific IEN filed in the SAME build. So menu items author by name with no
    target-IEN knowledge. `ProtocolItem{Name,Sequence}` + `ProtocolItemComp`. Fixture
    `testdata/zzproto` now self-contained (ACTION + MENU→ACTION). TDD; lint/race/contract
    green; corpus DRIFT=0; golden regenerated. Live install→verify→`--force` uninstall→
    clean both engines. Basic attach needs no extended action (USE-AS-LINK / MERGE /
    ATTACH / DISABLE remain a minor follow-up). ([[option-entry-component]])
  - **B.1-o ✅ DONE + LIVE-PROVEN 2026-06-28 — OPTION MENU multiple (#19.01), both
    engines.** Parity with B.1-n: the OPTION #19 MENU multiple (field 10, subfile
    19.01IP @ node 10) via the same KIDS `"^"` resolver convention — data
    `10,<seq>,0)=<placeholder>^<synonym>^<displayorder>` + `10,<seq>,"^")=<CHILD NAME>`.
    OPTION re-files (OPTF1/OPTE1) so it re-points by name AND rebuilds the menu "B"
    index (emitter ships neither). `OptionMenuItem{Name,Synonym,DisplayOrder}` +
    `OptionMenuItemComp`; validateOptions checks item names. Fixture `testdata/zzoption`
    now self-contained (MENU→RUN ROUTINE). TDD; lint/race/contract green; corpus
    DRIFT=0; golden regenerated. Live install→verify→`--force` uninstall→clean both
    engines: the menu item re-points to the sibling option's live IEN (17086 vehu /
    14290 foia-t12), synonym + order preserved. **The `"^"` resolver path is now proven
    for both #101- and #19-pointer menus.** ([[option-entry-component]])
  - **Extended menu-actions (USE-AS-LINK / MERGE / ATTACH / DISABLE) — INVESTIGATED &
    SCOPED OUT 2026-06-28 (not a B.1 gap).** Ground-truthed: these are the BLD-manifest
    NM piece-3 codes (#9.68 .03 ACTION set: `0 SEND`/`1 DELETE`/`2 USE-AS-LINK`/
    `3 MERGE MENU ITEMS`/`4 ATTACH TO MENU`/`5 DISABLE`). The emitter always ships
    `0 SEND`. Codes 2-4 are **install-time menu-management semantics against an EXISTING
    site menu** (link/merge/attach) — a patch concern, not authoring; 2/4 likely ship a
    reduced link-only shape, and none is live-provable without a pre-existing-menu
    fixture. Decision: a distinct future capability (like the parked template family),
    NOT a B.1 authoring gap. Rationale in [[option-entry-component]]; DELETE (code 1) is
    covered by the `--force` uninstall path.
- **B.2 Real FILE DD + DATA (F3; the R3 enabler).** Extend `FileComp` to a
  multi-field DD (see §8 for the grounded node-set) and add **DATA** export with
  the four action codes (ADD-IF-NEW / MERGE / OVERWRITE / REPLACE) and FULL/PARTIAL
  DD. Allow permanent (non-test-range) file numbers in a package's namespace.
  - **B.2-a ✅ DONE 2026-06-28 — multi-field DD authoring (the R3 unblock).**
    `FileDD.Fields`/`FileComp.Fields` emit the `.01` NAME plus N typed fields —
    the five grounded scalar types **free text / numeric / date / set of codes /
    pointer**, each with `required` + optional help. Grammar re-grounded against a
    real new-file full DD (#8992.7 LOG4M CONFIG): header piece 1 is the literal
    `FIELD` (not the file name — §8's `<NAME>^^…` was a simplification), and a
    field's storage is carried inline in piece 4 of its `,<fld#>,0)` def node — real
    exports ship **no per-field `"GL"` map node**. TDD; lint/race/contract green;
    **corpus round-trip DRIFT=0 over 2,404 dists**; deterministic golden
    `testdata/zzvslaudit/ZZVSLAU.kids` (#999001, all 5 types). `internal/kids/filecomp.go`,
    `internal/buildspec/buildspec.go`, `pkgcli/build.go`. ([[multi-field-dd-emitter]])
  - **B.2-a ✅ LIVE-INSTALL-PROVEN 2026-06-28 — both engines.** The multi-field DD
    (#999001, all 5 typed fields) installs to **#9.7 status 3**, registers
    (`^DIC("B")`), and **files a record** exercising every type (`EVT-A^42^W^…`
    stored at the right pieces, `.01` xref, `UPDATE^DIE` dierr=0); `v pkg verify`
    confirms; clean uninstall — on vehu (YDB) + foia-t12 (IRIS) via the driver
    stack. **Bug found+fixed:** the pointer field def shipped `RP200'^^VA(200,`
    (empty piece 3) because `PointRoot` carries a leading `^` (the piece delimiter)
    — DD piece-3 wants the root with NO caret; `fieldDef` now strips it (was
    `FIA^XPDIK` NULSUBSC → status 2, file half-registered). Minor open gap: the
    pointed-to back-ref `^DD(200,"PT",…)` is not shipped (records still file fine).
    ([[multi-field-dd-emitter]])
  - **B.2-b ✅ DONE + LIVE-PROVEN 2026-06-28 — file DATA + the 4 action codes +
    permanent file numbers (both engines).** Data records ship under
    **`("DATA",<file>,<ien>,<node>)`** (the raw record storage subtree, value =
    caret-joined field pieces). The send-options string is **9 pieces**: **p7 = DATA
    COMES WITH FILE (`y`/`n`, the data switch)** and **p8 = SITE'S DATA action** — the
    four letters **`a`/`m`/`o`/`r`** straight off `^DD(9.64,222.8,0)` (= ADD ONLY IF
    NEW FILE / MERGE / OVERWRITE / REPLACE). Install: `FIA^XPDIK` runs the data loop
    when p7=`y`, calling **`DATAIN^DIFROMS` → `EN^DIFROMS4`** (reads p8: `o`→overwrite,
    `r`→replace, else merge) then `I^DITR` files + **REINDEX rebuilds the xrefs** — so
    the emitter ships data nodes only (the "B" index is rebuilt live). **Permanent
    file numbers:** the hard 999000–999999 gate relaxed to any positive canonical
    integer, but an out-of-range file MUST declare an explicit `globalRoot` (the
    `^DIZ` scratch default applies only in-range); namespace *governance* stays an
    org-registry concern (code permits, policy governs). Live proof: #999002 with 3
    records under MERGE installed byte-identically on vehu (YDB) + foia-t12 (IRIS) —
    every piece exact, "B" xref rebuilt, clean uninstall (DIZ+DIC+DD gone). Pointer
    *data values* remain out of scope (no pointer-resolution name nodes). TDD;
    lint/race/contract green; corpus DRIFT=0; golden `testdata/zzdata`
    (`TestBuild_ZZDATA_Deterministic`). `internal/kids/filecomp.go`,
    `internal/buildspec/buildspec.go`, `pkgcli/build.go`. ([[multi-field-dd-emitter]])
- **B.3 Install-time code authoring.** Let a build declare and ship Environment
  Check / Pre / Post-Install routines (the spec already has `envCheck`; wire it to
  emit + register), so authored packages can gate and migrate.

### Cross-cutting
- **Keep the waterline + the bespoke-installer ban.** Every engine touch stays
  through `mdriver.Client`; install/back-out stays `v pkg install`/`uninstall` of a
  real KIDS build (`[[bespoke-installer-forbidden]]`). Track A makes the install
  *more* KIDS-native, not less — it is the opposite of a bespoke installer.
- **Each new component type ships its tag→registry→red-gate triple** (the org
  contract), and a round-trip corpus test (the `make corpus` harness already
  proves decompose/assemble; extend it to assert author-then-install parity per
  type).

---

## The R3 case: a real multi-field DD

v-stdlib R3 wants a dedicated `VSL AUDIT` file with structured fields (timestamp,
DUZ, host/$IO, event category, detail). That is exactly **B.2**. Grounded in real
KIDS `^DD`/`^DIC` exports, the **minimum node set** a multi-field DD emitter must
produce for a new file `#F` (data global `^G(F,`) with, say, `.01` NAME +
numeric + set-of-codes + pointer:

```
^DD(F,0)            = "<NAME>^^<high-field#>^<field-count>"
^DD(F,0,"NM","<NAME>") = ""
^DD(F,.01,0)        = "NAME^RF^^0;1^<transform>"
^DD(F,.01,1,0)      = "^.1^1^1"                         ; B xref multiple header
^DD(F,.01,1,1,0)    = "F^B"                             ; file^xref-name
^DD(F,.01,1,1,1)    = "S ^G(""B"",$E(X,1,30),DA)="""""  ; SET logic
^DD(F,.01,1,1,2)    = "K ^G(""B"",$E(X,1,30),DA)"       ; KILL logic
^DD(F,1,0)          = "<NUM>^NJ<n>,0^^0;2^<transform>"  ; numeric (p2 = NJ print-len,dec)
^DD(F,2,0)          = "<SET>^S^i:EXT;...;^0;3^Q"        ; set-of-codes (p3 = code list)
^DD(F,3,0)          = "<PTR>^P<g>'^<pointed-root>^0;4^Q"; pointer (p3 = pointed-to root)
^DIC(F,0)           = "<NAME>^F"
^DIC(F,0,"GL")      = "^G(F,"                           ; data-global root (mandatory)
^DIC("B","<NAME>",F)= ""
```

Field node piece layout: `LABEL ^ TYPE-spec ^ (set-list|pointer-root) ^ node;piece
^ INPUT-transform`; type codes `F/N/D/S/P<n>/V/W/K/C`, attribute letters `R`
(required), `J<n>,<d>` (print length). KIDS files the literal `^DD` you emit, then
post-install **re-indexes** to build the `^G("B",…)` entries — the emitter ships
the *definitions*, not the index data. (Grounded in KIDS exports of #1.008
full-DD-with-xref/help/WP, #8992.7 free-text+set+multiple, #.114 numeric.)

**B.2-a (multi-field DD authoring) landed 2026-06-28**, so R3 is **unblocked**: an
audit file ships no seed data (records are filed at runtime via the DBS API), so
the multi-field DD alone is sufficient — v-stdlib R3 can now declare a `VSL AUDIT`
file with `.01` plus typed fields (timestamp = date, DUZ = numeric/pointer, host =
free text, category = set of codes, detail = free text). The remaining B.2 work
(file DATA + the 4 action codes, permanent file numbers) is not on R3's path. One
caveat: B.2-a is build-side proven (unit/golden/corpus) but **not yet
live-install-proven** — a multi-field DD has not been installed on a live engine,
unlike the single-`.01` lifecycle.

---

## Coverage accounting

Cumulative share of the 2,404 distributions v-pkg could **build+install
faithfully** (authoring ∩ install-fidelity):

| After | Authors | Installs faithfully | ~Corpus reach |
|---|---|---|---|
| **today** | routines, #8989.51, toy-file | routine filing only | ~28% (routine-only, and only the routine effects) |
| **T0** | same (but honest about gaps) | same | ~28%, no longer silently lying |
| **+Track A** | same | **any existing `.KID`, real phases** | ~**95%+ install** of existing dists (authoring still limited) |
| **+B.1** | +all entry types | +entry components | entry-bearing dists now *authorable* (OPTION 39%, KEY 20%, …) |
| **+B.2** | +multi-field DD +data | +file DD/data | +file-bearing dists (24%); **R3 unblocked** |
| **+B.3 +A.2–A.4** | +install-code, +PKG, +multi-build | +enforcement, +history | **→ 100% author+install** |

The headline: **Track A alone gets "install any existing package" to ~95%+**;
the Track B tiers are what make v-pkg able to *author* the full component range
(and unblock VSL/R3).

---

## Risks and open questions

- **The install-engine fork (A.1) is the pivotal architectural decision.** Silent
  XPD\* answer-variable seeding vs expect-driven menus vs continuing to extend the
  populate path. The corpus does not document the Kernel silent-install API in
  full; resolving A.1 likely needs the Kernel 8.0 *KIDS Developer Tools* UG (noted
  absent from the gold corpus in `implementation-plan.md`). **Recommend a spike
  before committing.**
- **Reading live entries for authoring (B.1)** means v-pkg gains broad FileMan-read
  surface; keep every read through the DBS API + `mdriver.Client`, never direct
  global reads, to stay inside the waterline.
- **Permanent file numbers (B.2)** need a namespace policy (which non-test range a
  package owns); coordinate with the org namespace registry rather than minting
  numbers ad hoc.
- **This is a v-pkg roadmap, not a committed plan.** It reorders and extends the
  existing `implementation-plan.md` / `kids-installation-automation.md` threads
  (Tier A/B install, package extraction) with corrected evidence and the two-track
  framing; fold it into that tracker rather than forking a parallel plan.

---

## Sources

- KIDS model + install phases + mechanics: **Kernel 8.0 Developer's Guide: KIDS UG**
  (`XU/krn_8_0_dg_kids_ug`), **Systems Management: KIDS UG**
  (`XU/krn_8_0_sm_kids_ug`), **VistA Build Analyzer UG**
  (`XU/vista_build_analyzer_ug`).
- Real-world frequency: WorldVistA corpus, 2,404 distributions, re-tallied
  (`~/data/kids-patches/`; corrected probes supersede `kids-corpus-findings.md`).
- `^DD`/`^DIC` grammar: real KIDS exports of files #1.008, #8992.7, #.114
  (`~/data/kids-patches/VistA/Packages/…`).
- v-pkg current surface: `internal/buildspec/buildspec.go`,
  `internal/kids/{buildkids,krncomp,filecomp,reversibility}.go`,
  `internal/installspec/script.go`, `pkgcli/lifecycle.go`.
