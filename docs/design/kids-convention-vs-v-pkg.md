---
title: "How v-pkg installs & uninstalls KIDS builds — measured against the VA historical convention"
status: analysis
created: 2026-06-30
related:
  - ../../../docs/kids-classifee.md   # the convention, measured from 12,955 installed builds in vehu
  - kids-installation-automation.md   # the install-automation design this realizes
  - kids-corpus-findings.md           # the parallel 2,404-distribution corpus evidence
  - ../memory/class-aware-uninstall.md
  - ../memory/reversibility-classifier.md
  - ../memory/bespoke-installer-forbidden.md
---

# How `v-pkg` Installs & Uninstalls KIDS Builds — vs. the VA Historical Convention

A comparative analysis. The companion report
[`docs/kids-classifee.md`](../../../docs/kids-classifee.md) measured **what KIDS builds
actually are** by classifying all **12,955 builds installed in `vehu`** (the BUILD file
#9.6). This document asks the next question: **given that convention, how does `v-pkg`'s
`install` / `uninstall` compare to it — what does it faithfully preserve, what does it
extend, and where does it deliberately diverge?**

> **One-sentence answer.** `v-pkg` **installs** by faithfully driving the *native* KIDS
> filer through its real M entry points (no bespoke installer) — honoring every parameter
> the convention defines — and **uninstalls** by adding the one thing the convention never
> had: a **class-aware, reversibility-typed back-out** that refuses to fake a reversal it
> cannot soundly compute.

---

## Table of Contents

1. [The convention, in one page](#1-the-convention-in-one-page)
2. [Two independent corpora, one finding](#2-two-independent-corpora-one-finding)
3. [How v-pkg installs — faithful automation of the native filer](#3-how-v-pkg-installs--faithful-automation-of-the-native-filer)
4. [Parameter-by-parameter: how install honors each KIDS convention](#4-parameter-by-parameter-how-install-honors-each-kids-convention)
5. [How v-pkg uninstalls — the deliberate divergence](#5-how-v-pkg-uninstalls--the-deliberate-divergence)
6. [The reversibility class vs. the kids-classifee taxonomy](#6-the-reversibility-class-vs-the-kids-classifee-taxonomy)
7. [Side-by-side: convention vs. v-pkg](#7-side-by-side-convention-vs-v-pkg)
8. [Exit codes — the safety contract the convention lacked](#8-exit-codes--the-safety-contract-the-convention-lacked)
9. [What v-pkg faithfully preserves, extends, and diverges from](#9-what-v-pkg-faithfully-preserves-extends-and-diverges-from)
10. [Honest gaps](#10-honest-gaps)
11. [Conclusion](#11-conclusion)

---

## 1. The convention, in one page

From `kids-classifee.md` and the Kernel KIDS User Guide, the VA historical convention is:

- **The unit of change is the whole routine, replaced under checksum.** A build ships the
  *complete* new routine source (action `0` = SEND TO SITE) plus a `B`-prefixed
  after-image checksum; install **overwrites** the resident routine (create if absent).
  Removal is explicit (action `1` = DELETE AT SITE). There is no line-level patching.
  In `vehu`: **88.9 % of builds carry routines**, 88,717 send-actions, 5,678 delete-actions.
- **Install is a three-phase, terminal-driven conversation** — **Load** (`[XPD LOAD
  DISTRIBUTION]` → `EN1^XPDIL`, creates an INSTALL #9.7 entry, lays the transport global
  into `^XTMP`) → **Questions** (environment check + standard KIDS questions + build-specific
  pre/post questions) → **Install** (`[XPD INSTALL BUILD]` → `EN^XPDIJ`, files components,
  runs the post-install routine, records status in #9.7).
- **The source of truth is FileMan.** INSTALL #9.7 STATUS (piece 9 = `3` "Install
  Completed") records *what installed and when*; PACKAGE #9.4 PATCH APPLICATION HISTORY
  records *which patch is applied* (`$$PATCH^XPDUTL`).
- **Builds carry guarded install logic and dependencies:** environment-check (#913),
  pre-install (#916), post-install (#914) routines; install questions (#50); required
  builds (#11, present on **74.6 %** of `vehu` builds).
- **The methodology is forward-only.** *KIDS ships no generic uninstall.* The only native
  rollback aid is "Backup a Transport Global" — which backs up **routines only**, not DD,
  data, or FileMan entries — and a per-patch manual back-out procedure (the DIBR guide).

The companion report's quantified punchline matters here: **most builds are not pure code.**
Post-install routines run on **37.1 %** of `vehu` builds, FileMan components ship in
**26.6 %**, file data/DD in **27.5 %**. A build that runs install code or files data has
**no computable inverse** — which is precisely *why* the convention is forward-only.

---

## 2. Two independent corpora, one finding

`v-pkg`'s design was grounded by an **independent** corpus analysis
([`kids-corpus-findings.md`](kids-corpus-findings.md)): a static parse of **2,404
WorldVistA `.KID` distributions on disk**. The `kids-classifee` report measured a
**different population by a different method** — **12,955 builds installed live in `vehu`**,
probed through the driver stack. They agree on the load-bearing fact:

| Measure | `kids-classifee` (vehu installed, n=12,955) | `kids-corpus-findings` (WorldVistA on-disk, n=2,404) |
|---|---:|---:|
| Ship ≥ 1 routine | 88.9 % | 97 % |
| **Pure routine-overwrite** (no install code, entries, or DD/data) | **38.9 %** | **28 %** (committed classifier ~36 %) |
| Side-effecting (the rest) | 61.1 % | 72 % |
| Ship install-time code (env/pre/post) | ~46 % (post 37 % + pre 10 % + env 11 %) | 51 % |
| Ship non-routine FileMan entries | 26.6 % | 50 % |
| Ship a FILE (DD/data) | 27.5 % | 24 % |
| Declare a required build | 74.6 % | 79 % |
| Multi-build distribution | 1.2 % | 3.7 % |

The absolute percentages differ — `vehu`'s installed set is full of small infrastructure
and local/`ZZ` patches, while the WorldVistA repo skews toward national clinical packages
with richer component payloads — but the **conclusion is identical and population-robust**:

> **The pure-overwrite, snapshot/restore-reversible class is a *minority* (≈ a third).
> The majority of real builds are side-effecting and have no generic inverse.**

This is the empirical foundation of v-pkg's uninstall design. A tool that treated
"snapshot the routine, restore it to undo" as *the* model would be wrong for the majority
of real patches — both corpora say so.

---

## 3. How v-pkg installs — faithful automation of the native filer

**Hard rule ([`bespoke-installer-forbidden.md`](../memory/bespoke-installer-forbidden.md)):
v-pkg never hand-rolls an installer or patcher.** It drives the *real* KIDS routines
(`XPDIL1`, `XPDIJ`, `XPDUTL`, `XPDIK`, `XPDIQ`, `XPDIP`) on the engine through the single
sanctioned seam, the `m-driver-sdk` `mdriver.Client`. The interactive `EN1^XPDIL` host-file
load cannot be fed over a stdin-less subprocess transport, so v-pkg uses **"route (c)
augmented direct-populate"**: stage the transport global itself, then call the same filer
the menu would.

### The install sequence (`FinalInstallScript`, emitted M)

| # | Step | Native KIDS API used | Purpose / convention honored |
|---|---|---|---|
| 1 | Stage transport in 40 KB chunks → `^XTMP("VPKGI")` | (`MERGE` target) | large transports truncate in one routine; chunk + count-guard makes silent partial-install fail **loudly** (`stage-incomplete`) |
| 2 | Re-install guard `$D(^XPD(9.7,"B",name))` | — | idempotency (convention §8); avoids the un-answerable "OK to continue with Load" prompt |
| 3 | Create the INSTALL #9.7 entry | **`$$INST^XPDIL1(name)`** → `XPDA` | the real #9.7 record, same as the Load phase |
| 4 | `K`+`MERGE ^XTMP("XPDI",XPDA)=^XTMP("VPKGI")` | — | lay the transport global into `^XTMP` exactly as native load does |
| 5 | Environment check + required builds | **`$$ENV^XPDIL1(1)`** + `REQB^XPDIL1` | runs the build's #913 env-check and enforces #11 REQB ordering; rejects → `env-check-rejected` |
| 6 | Seed + answer install questions | seed `^XPD(9.7,XPDA,"QUES")` → **`$$ANSWER^XPDIQ`** | `--answer NAME=VALUE` pre-answers #50 questions through the real reader |
| 7 | Seed KRN component multiple | `M ^XPD(9.7,XPDA,"KRN")=…` for **`KRN^XPDIK`** | options/protocols/RPCs/keys/… file via the real component filer |
| 8 | Seed FileMan FILE checkpoint | **`XPCK^XPDIK("FIA")`** | DD/data (#6 FILE) filed as the interactive `GI^XPDIL` would |
| 9 | Pre/post-install checkpoints | **`$$NEWCP^XPDUTL`** ("XPD PREINSTALL/POSTINSTALL …") | #916/#914 hooks fire via `PRE^/POST^XPDIJ1` `D @` only if checkpoints exist |
| 10 | **Run the install** | **`D EN^XPDIJ`** | the synchronous native KIDS install — files all components, runs pre/post |
| 11 | Read result | `$P(^XPD(9.7,XPDA,0),U,9)` | #9.7 STATUS `3` = "Install Completed" — the convention's own success oracle |
| 12 | (opt) PACKAGE #9.4 footprint | **`$$PKGVER^XPDIP` / `$$PKGPAT^XPDIP`** | stamp VERSION (#9.49) + PATCH HISTORY (#9.4901) so `$$PATCH^XPDUTL` is honest |

Every component lands through the **same FileMan/KIDS filer the VA convention uses** — v-pkg
adds orchestration and guards around it, not a parallel install path. The whole thing runs
in **one engine process** so `XPDA` and the symbol table survive (the SETs `EN^XPDIJ` needs).

### The class-aware front gate (`decideInstall`)

Before installing, v-pkg probes which target routines **already exist** on the engine
(`captureRoutinePreimages`) and refuses to silently clobber a national routine:

```
no existing routine            → proceed (greenfield)
existing + --snapshot <out>    → capture pre-image first, THEN install   (enables uninstall --restore)
existing + --allow-overwrite   → clobber without a pre-image             (uninstall cannot restore)
existing + neither             → REFUSE, exit 4 (INSTALL_REFUSED)
```

This is the first place v-pkg goes **beyond** the convention: native KIDS overwrites
resident routines unconditionally; v-pkg makes "you are about to overwrite something you
can't get back" a hard, exit-coded stop.

---

## 4. Parameter-by-parameter: how install honors each KIDS convention

Mapping the build parameters that `kids-classifee` enumerated to v-pkg's handling:

| KIDS parameter (kids-classifee) | Convention behavior | v-pkg install handling |
|---|---|---|
| **Routines** (#7 → 9.8, action 0/1) | whole-routine overwrite / delete under checksum | shipped in the transport, filed by `EN^XPDIJ`; checksums later asserted by `verify --drift` |
| **FILE / DD** (#6, full/partial) | DD update via the FILE multiple | `FIA` seed + `XPCK^XPDIK("FIA")` — real DD filer; multi-field DD emitter |
| **DATA COMES WITH FILE** (#6 p7) | file data ships and files | filed through the same FILE path; reversal flagged side-effecting |
| **BUILD COMPONENTS** (#7, KRN) | options/protocols/RPCs/templates via FileMan | `KRN` seed → `KRN^XPDIK`; generic entry-component emitter per file # |
| **ENVIRONMENT CHECK** (#913) | can abort the install | `$$ENV^XPDIL1(1)` runs it; non-zero → `env-check-rejected` refusal |
| **PRE / POST-INSTALL** (#916/#914) | data conversion hooks | `$$NEWCP^XPDUTL` checkpoints → real `PRE^/POST^XPDIJ1` execution |
| **INSTALL QUESTIONS** (#50) | interactive operator answers | `--answer NAME=VALUE`, read by the real `$$ANSWER^XPDIQ` |
| **REQUIRED BUILD** (#11) | prerequisite enforcement | `REQB^XPDIL1` over the `BLD…REQB` nodes |
| **MULTIPLE BUILD** (#10) | multi-package unit, install in order | `installSequence` runs constituents in **KIDS-header order**, stop-on-failure |
| **INSTALL #9.7 status** | the success oracle | read directly: status `3` = installed |
| **PACKAGE #9.4 history** | "is this patch applied?" | `--register-package` writes it; honest by request |

There is **no KIDS install parameter the convention defines that v-pkg bypasses or
re-implements.** It is automation *of* the convention.

---

## 5. How v-pkg uninstalls — the deliberate divergence

This is where v-pkg adds what the convention never had. Native KIDS ships **no generic
uninstall**; the convention's answer to "undo a patch" is a hand-written, per-patch DIBR
procedure. v-pkg replaces that with a **reversibility-typed** back-out, decided from the
`.KID` alone, that **refuses to fake a reversal it cannot soundly perform.**

### 5.1 Snapshot — capturing the pre-image the convention threw away

`install --auto-snapshot` (or `snapshot`) reads each target routine's **current source off
the live engine** *before* the overwrite (`readRoutineBody`: stream `$T(+I^name)` lines),
and writes a routine-only pre-image KIDS build named `"<orig> PREIMAGE"` to a sidecar
`<kid>.preimage.kids`. Crucially, the snapshot carries an **honest `completeUndo` flag**:
`true` **only** when the build is pure-overwrite, adds no greenfield routines, and touches
no non-routine component. Otherwise the snapshot is recorded as *provenance, not a reversal
guarantee*, and itemizes the `uncaptured` components an authored back-out must cover. This
is the convention's "Backup a Transport Global" — but **typed for honesty** (native backup
silently covered only routines and never said so).

### 5.2 The uninstall decision tree (`decideUninstall`)

```
both --restore and --backout ............................. REFUSE (exit 4)   "pick one"
--restore <preimage> + build adds greenfield routines .... actPartition       (restore foreign, then delete greenfield)
--restore <preimage> ..................................... actRestore         (re-install the pre-image)
--backout <authored.kid> ................................. actBackout         (install the authored inverse)
build declares a FOREIGN overwrite, no pre-image:
        with --force ..................................... actDelete          (greenfield subset ONLY; foreign LEFT in place)
        without --force .................................. REFUSE (exit 4)    "would BRICK the foreign/national routine"
class = SideEffecting (the 61–72% majority):
        with --force ..................................... actDelete          (orphans data — explicit override)
        without --force .................................. REFUSE (exit 4)    "provide --backout / --restore / forward back-out"
class = PureOverwrite (greenfield) ....................... actDelete          (routine-delete back-out)
```

- **`actRestore`** = install-of-the-pre-image (reuses the proven install path). Previews by
  default; **does not touch the engine without `--commit`** (it overwrites live national
  routines).
- **`actDelete`** (`UninstallScript`) = `^%ZOSF("DEL")` per **greenfield** routine + FileMan
  `^DIK` by IEN for each component type (params #8989.51, options #19, keys #19.1, protocols
  #101, RPCs #8994, …) + FILE DD/data + the #9.7/#9.6 records — the inverse of the filer.
- **`actPartition`** (mixed overwrite-foreign + greenfield, e.g. the v-rpc-tap shape):
  **restore the foreign routine FIRST, delete the greenfield AFTER** (so a live caller never
  reaches a deleted callee), keyed on which build routines appear in the pre-image; idempotent.
- **Foreignness is by *declaration*, never a name heuristic** (a package name ≠ its routine
  namespace — MSL ships `STD*`). A build that declares a foreign overwrite with no pre-image
  **refuses rather than brick** the national routine it cannot restore.

### 5.3 Symmetric footprint removal

`uninstall --deregister` removes the single #9.4 PATCH APPLICATION HISTORY entry (FileMan
`^DIK` on #9.4901) so `$$PATCH^XPDUTL` stops reporting the patch — the exact inverse of
`install --register-package`. It deliberately **leaves** the VERSION and package entries
(they may carry other/national patches; *KIDS itself never removes a package*). Native KIDS
had no symmetric operation at all.

### 5.4 Verification — proving the reversal

`uninstall --verify` (and `verify --drift`/`--content`) re-reads the affected routines off
the engine and grades them: `VerifyClean = clean|dirty` (exit 3 on dirty), plus the
declared-foreign restore fidelity `foreignRestore = exact | command-clean-provenance-drift |
drift` (line-2 patch-history provenance is excluded from the checksum surface, so a clean
restore that only differs in the comment line is graded honestly, not as drift). The
convention offered "Verify Checksums in Transport Global" as an *optional pre-install* step;
v-pkg makes post-operation verification a typed, exit-coded gate.

---

## 6. The reversibility class vs. the kids-classifee taxonomy

v-pkg's two reversibility classes are a **coarsening of the `kids-classifee` 10-class
taxonomy, projected onto the question "can this be reversed by restoring a pre-image?"**

| `kids-classifee` class | v-pkg reversibility class | Why | Uninstall route |
|---|---|---|---|
| C1 Routine-Only Code (no install hooks) | **PureOverwrite** | only routine overwrites | `--restore` pre-image (complete undo) / greenfield delete |
| C1 Routine-Only **with** pre/post hook | **SideEffecting** | hook ran arbitrary M | refuse → `--backout` / `--force` |
| C2 Code + Component | **SideEffecting** | FileMan entries filed | `--backout` (entries have no pre-image inverse) |
| C3 Code + Data | **SideEffecting** | DD/data filed | `--backout` / forward back-out |
| C4 Component / Config | **SideEffecting** | FileMan entries | `--backout` |
| C5 Data-Only Patch | **SideEffecting** | data rows filed | `--backout` / forward |
| C6 DD / Schema-Only | **SideEffecting** | DD change | `--backout` |
| C7 Component + Data | **SideEffecting** | entries + data | `--backout` |
| C8 Global Package | **SideEffecting** | whole-global load | `--backout` / forward |
| C9 Multi-Package Container | least-reversible member governs | a bundle is only as reversible as its worst build | per-constituent |
| C10 Informational / No-Payload | **PureOverwrite** (trivially) | nothing to reverse | delete #9.7/#9.6 |

The key insight both documents share: **the install hook is the silent reversibility
killer.** A build that ships nothing but routines is `kids-classifee` C1, yet if it also
carries a post-install routine (37 % of `vehu` builds do), v-pkg correctly downgrades it to
SideEffecting — because that hook may have re-indexed a file or queued a job that putting
the old routine back will not undo. The classifier keys exactly on this:
`SideEffecting if (install-code OR FileMan-entries OR DD/data)`.

---

## 7. Side-by-side: convention vs. v-pkg

| Dimension | VA historical convention (native KIDS) | `v-pkg` |
|---|---|---|
| **Install transport** | interactive 3-phase menu (`EN1^XPDIL`→`EN^XPDIJ`) | direct-populate `^XTMP` + the **same** `EN^XPDIJ` (no bespoke installer) |
| **Driver** | operator at a terminal | `mdriver.Client` over m-ydb/m-iris (engine-neutral) |
| **Routine change** | whole-routine overwrite under checksum | identical — filed by native KIDS; checksum re-asserted by `verify` |
| **Install questions** | typed at the device | `--answer NAME=VALUE` via `$$ANSWER^XPDIQ` |
| **Env-check / pre / post** | run by KIDS | run by KIDS (`$$ENV^XPDIL1`, `$$NEWCP^XPDUTL` checkpoints) |
| **Required builds** | checked by KIDS | `REQB^XPDIL1` enforced |
| **Success oracle** | #9.7 STATUS / #9.4 history (read by human) | same files, read programmatically; status `3` |
| **Overwrite safety** | none — silent clobber | **refuse** unless `--snapshot` or `--allow-overwrite` (exit 4) |
| **Uninstall** | **none** (manual per-patch DIBR) | **class-aware**: `--restore` / `--backout` / delete / **refuse** |
| **Pre-image backup** | "Backup a Transport Global" — routines only, silent scope | typed snapshot with honest `completeUndo`; covers the uncaptured-component gap explicitly |
| **Reversibility model** | "forward-only" by fiat | forward-only **proven** per build from the `.KID`; refuses to fake an inverse |
| **Footprint** | #9.4 written on install, never removed | symmetric `--register-package` / `--deregister` |
| **Verification** | optional pre-install checksum | typed post-op `verify --drift/--content`, exit-coded |

---

## 8. Exit codes — the safety contract the convention lacked

The native convention had no machine-checkable refusal; an operator simply *didn't* run a
dangerous step. v-pkg encodes the safety boundary as exit codes:

| Code | Name | When |
|---|---|---|
| 0 | OK | success |
| 1 | Runtime | engine/IO/parse failure (`INSTALL_FAILED`, `UNINSTALL_FAILED`, …) |
| 2 | Usage | bad flags (`MULTI_BUILD_REGISTER`, `BAD_ANSWER`, …) |
| 3 | Check | drift / not-verified-clean (`NOT_VERIFIED`, `VERIFY_CLEAN_FAILED`) |
| **4** | **Refused** | **the safety stop** — `INSTALL_REFUSED` (clobber w/o pre-image), `UNINSTALL_REFUSED` (foreign-brick / wrong sidecar / side-effecting), `ALREADY_INSTALLED`, `NO_DRIVER` |

Exit 4 is the codification of the convention's hard-won lesson: *do not perform an
irreversible engine mutation that you cannot honestly undo.* Gates assert it
(`scripts/foreign-refuse-gate.sh` 24/24, `partition-uninstall-gate.sh` 17/17 — vehu + foia).

---

## 9. What v-pkg faithfully preserves, extends, and diverges from

**Faithfully preserves (automation *of* the convention):**
- The native KIDS filer (`EN^XPDIJ`) and every component path — **no bespoke installer**.
- Whole-routine replacement under checksum; the #9.7/#9.4 source-of-truth files.
- Install questions, env-check/pre/post hooks, required-build ordering, multi-build order.
- Forward-only as the *default truth* for the side-effecting majority.

**Extends (adds what the convention left to humans):**
- Non-interactive, engine-neutral, idempotent install with a loud `stage-incomplete` guard.
- Pre-flight overwrite refusal (no silent clobber of a national routine).
- A typed pre-image **snapshot** with an honest completeness flag.
- A **class-aware uninstall** (`restore`/`backout`/`delete`/`partition`) the convention never had.
- Symmetric #9.4 footprint registration/deregistration.
- Typed post-operation **verification** (drift / content / foreign-restore fidelity).

**Deliberately diverges:**
- It **refuses** to overwrite-without-pre-image and to delete-a-side-effecting-patch — where
  native KIDS would simply proceed (install) or offer nothing (uninstall). The divergence is
  *toward* the convention's own forward-only honesty, made mechanical and unskippable.

---

## 10. Honest gaps

- **The committed classifier (`reversibility.go`) uses an older top-level-`KRN` probe (~36 %
  pure-overwrite); the corrected corpus probe is 28 %.** Aligning it is tracked separately —
  it gates real uninstall behavior, so it is changed deliberately, not casually. Either way
  the class is a minority; the conservative skew (calling *more* builds side-effecting) is
  the safe direction.
- **v-pkg cannot author the inverse for the 61–72 % side-effecting majority.** For those it
  requires an authored `--backout` or a declared forward back-out patch — the same burden the
  convention's DIBR places on a human, now explicit and typed rather than implicit.
- **Two populations, not one.** `kids-classifee` measured `vehu`'s installed set; the v-pkg
  corpus measured WorldVistA on-disk distributions. The shared conclusion is robust to the
  difference, but exact percentages are population-specific and should not be cross-cited as
  if from one dataset.
- **DD/data reversal remains the hardest case** — restoring a routine pre-image never reverses
  a filed DD change or a data row; snapshot itemizes these as `uncaptured`, but computing
  their inverse is out of scope (it is, fundamentally, what makes the convention forward-only).

---

## 11. Conclusion

`kids-classifee` showed that the VA's KIDS convention is a **forward-only, whole-routine,
checksum-verified distribution mechanism whose forward-only nature is *forced* by what
builds actually do** — the majority run install code or file data that has no inverse.

`v-pkg` is the faithful machine reading of that convention. Its **install** is automation
*of* the native filer — same entry points, same files, every parameter honored, zero
bespoke installation logic. Its **uninstall** is the convention's missing half, built
honestly: it **types each build's reversibility from the `.KID`**, performs a complete
pre-image restore only for the pure-overwrite minority, demands an authored back-out for the
side-effecting majority, and **refuses — with exit 4 — to perform any reversal it cannot
soundly compute.** Where the convention trusted an operator to not do the dangerous thing,
v-pkg makes the dangerous thing unrepresentable.

---

*Companion to [`docs/kids-classifee.md`](../../../docs/kids-classifee.md). Mechanism grounded
in `pkgcli/lifecycle.go`, `pkgcli/snapshot.go`, `internal/installspec/script.go`,
`internal/kids/reversibility.go`; corpus cross-validation in `kids-corpus-findings.md`.*
