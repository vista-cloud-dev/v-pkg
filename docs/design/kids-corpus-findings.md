---
title: "What 2,404 real KIDS patches actually do — corpus evidence for the v-pkg reversibility model"
status: FINDINGS (2026-06-25; CORRECTED 2026-06-28) — empirical grounding for patch-existing-routines-proposal.md
created: 2026-06-25
last_modified: 2026-06-28
corpus: WorldVistA/VistA (github), all Packages/**/*.KID[S] distributions
method: static parse of the KIDS BUILD (#9.6) transport global in each .KID — see ~/data/kids-patches/analyze.py
related: ../archive/patch-existing-routines-proposal.md (the design these numbers justify); ../proposals/v-pkg-kids-coverage-analysis.md (F5/T0.2 — the correction)
---

# What 2,404 real KIDS patches actually do

> **⚠ CORRECTED 2026-06-28 (coverage-analysis F5 / T0.2).** The first cut keyed
> three probes off node shapes that don't match the real transport global, so
> several headline numbers were wrong — all now fixed in `analyze.py` and below:
> - **FileMan entries** were read from a top-level `"KRN",<file>,<ien>,0)` node
>   (a rare installed-instance region), undercounting every type — OPTION read as
>   13%. The real per-build signal is `"BLD",<ien>,"KRN",<file>,"NM",…)`; OPTION
>   is **39%**, and non-routine entries overall are **50%** (was 23%).
> - **Required-builds** matched the bare substring `"REQB"`, hitting the empty
>   scaffold node in ~every build (96%). The real dependency node is
>   `"BLD",<ien>,"REQB",1,0)` → **79%** (1,890 of 2,404).
> - **Multi-build** lacked `re.M`, so it never matched → reported 0%. With `re.M`
>   → **3.66%** (88).
> - The file#→name map had `.4/.401/.402/.403`, `771/870/779.2`, `3.6` mislabeled;
>   corrected to the VistA Build Analyzer UG mapping.
>
> The corrected entry detection also lowers the pure-routine-only class
> (**35%→28%**) and raises side-effecting (**64%→72%**) — the forward-only thesis
> holds *a fortiori*. (The committed classifier `internal/kids/reversibility.go`
> still uses the older top-level-`KRN` probe → ~36%; aligning it is a follow-up,
> tracked separately — it gates real uninstall behavior, so it is not changed
> here.) The corrected `analyze.py` writes `analysis-report.txt`.

The v-pkg reversibility proposal ([patch-existing-routines-proposal.md](../archive/patch-existing-routines-proposal.md))
argued, from the single lived XWBBRK case, that **reversibility is a property of
the whole patch, not the routine** — and that snapshot/restore is sound only for
the narrow "pure routine-overwrite" class. To check that claim against reality we
cloned the entire **WorldVistA/VistA** patch corpus and statically parsed every
KIDS distribution's transport global.

**Corpus:** 2,404 `.KID`/`.KIDS` distributions across 157 packages (the full
WorldVistA mirror). Parser: `~/data/kids-patches/analyze.py` (not committed — it
operates on `~/data`, the persistent-data lake). Each file is the KIDS BUILD
(`#9.6`) global dump; we tally what each ships by node form (corrected probes):
`"RTN","<name>")` = routine; `"BLD",<n>,"KRN",<file#>,"NM",…)` = an exported
FileMan entry of `<file#>`; `"FIA",<file#>,…)` = a FileMan FILE (DD and/or data)
component; `"BLD",<n>,"REQB",1,0)` = a real required-build dependency;
`Environment Check`/`Pre-Install`/`Post-Install` = install-time code. (Each
transport node is stored as two lines — the subscript path, then its value.)

## Headline numbers

| What the patch ships / does | Distributions | % |
|---|---:|---:|
| Ship ≥ 1 routine | 2,343 | **97%** |
| **Routines ONLY** — no code, no entries, no DD/data | **674** | **28%** |
| **Side-effecting** — install code and/or FileMan entries and/or DD/data | **1,730** | **72%** |
| Ship install-time CODE (Env-Check / Pre / Post) | 1,235 | **51%** |
|  ⤷ Environment Check routine | 317 | 13% |
|  ⤷ Pre-Install routine | 109 | 4% |
|  ⤷ Post-Install routine | 293 | 12% |
| Ship non-routine FileMan entries (options/RPCs/protocols/keys/…) | **1,206** | **50%** |
| Ship a FileMan FILE (DD and/or data) | 579 | 24% |
|  ⤷ DD + data | 334 | 14% |
|  ⤷ data only (no DD) | 245 | 10% |
|  ⤷ DD only (no data) | 0 | 0% |
| Declare a real required-build dependency (`"REQB",1,0)`) | **1,890** | **79%** |
| Multi-build distribution (> 1 install name) | 88 | 3.66% |

Total routine instances shipped across the corpus: **14,288**.

### FileMan component types shipped (full, by distribution count)

`#9.8 ROUTINE` heads the list because builds register routines as `KRN`
components too; it is the routine-overwrite itself (counted separately above) and
is excluded from the side-effecting "non-routine FileMan entries" row.

| File # | Component | Distributions | % |
|---|---|---:|---:|
| 9.8 | ROUTINE | 2,286 | 95% |
| 19 | OPTION | 943 | **39%** |
| .4 | PRINT TEMPLATE | 489 | 20% |
| 19.1 | SECURITY KEY | 482 | 20% |
| 101 | PROTOCOL | 421 | 18% |
| .402 | INPUT TEMPLATE | 246 | 10% |
| 8994 | REMOTE PROCEDURE | 228 | 9% |
| 409.61 | LIST TEMPLATE (OE/RR) | 183 | 8% |
| 8989.51 | PARAMETER DEFINITION | 165 | 7% |
| 771 | HL7 APPLICATION PARAMETER | 132 | 5% |
| .401 | SORT TEMPLATE | 114 | 5% |
| 3.8 | MAIL GROUP | 108 | 4% |
| 870 | HL LOGICAL LINK | 103 | 4% |
| 9002226 | (lexicon family) | 84 | 3% |
| .403 | FORM | 78 | 3% |
| 8989.52 | PARAMETER TEMPLATE | 74 | 3% |
| 779.2 | HLO APPLICATION REGISTRY | 59 | 2% |
| 9.2 | HELP FRAME | 53 | 2% |
| .84 | DIALOG | 53 | 2% |
| 3.6 | BULLETIN | 50 | 2% |
| .5 | FUNCTION | 34 | 1% |
| 1.5 / 8993 / 1.62 | ENTITY / LOCK DICT / POLICY FN | ≤3 | — |

Each of these is a **data write into a FileMan file** that KIDS uninstall does
*not* reverse — installing an OPTION (#19) entry or a PARAMETER DEFINITION
(#8989.51) leaves a permanent row. Restoring a routine pre-image does nothing for
these.

## What this proves for v-pkg

1. **The pure-overwrite class is the minority — 28%, not the norm.** Only 674 of
   2,404 patches ship routines *and nothing else*. These are the only distributions
   for which "snapshot the routine, restore it to undo" is a *complete and sound*
   reversal. The XWBBRK splice that motivated the proposal is in this 28% — which is
   exactly why snapshot/restore is right *for it* and a trap if generalized. **A
   v-pkg that treated snapshot/restore as THE undo model would be wrong for ~7 of
   every 10 real patches.** (The first cut put this at 35% because it undercounted
   FileMan-entry patches; accurate detection makes the reversible class *smaller*.)

2. **The majority — 72% — are side-effecting and have no generic inverse.** 51%
   run arbitrary install-time M (Env-Check/Pre/Post); **50% file FileMan entries**
   (not the 23% first reported); 24% ship DD/data. For these, putting the old
   routine back removes the *code* but orphans every data row, DD change, parameter,
   queued job, and message the install produced — often leaving the system *worse*
   than the patched state. This is the corpus confirming, at scale, the user's
   correction: **VA's forward-only methodology is not a limitation of their tooling;
   it is forced by what patches actually do.** There is no inverse to compute
   because the forward step ran arbitrary, sometimes externally-irreversible, code.

3. **Patches are almost never standalone — 79% declare a required build.** Reversal
   and install both occur inside a dependency chain; a v-pkg uninstall (or install)
   that ignores `"REQB"` ordering is unsafe. Reversibility must be reasoned about
   over the chain, not the single build. (The first cut's 96% counted the empty
   `"REQB"` scaffold node present in nearly every build; the real-dependency node
   `"REQB",1,0)` is in 79%.)

4. **The reversibility class is statically derivable from the `.KID`** — exactly
   what the proposal's "detect the class, don't trust the tag" requirement needs.
   The four node-form probes above (routine-only? install code? FileMan entries?
   FileMan FILE?) are enough to bucket every one of the 2,404 distributions with no
   human judgement. That is the source-tag → registry → red-gate input: v-pkg can
   compute the class at build time and **red-gate any build that declares
   `reversible` while carrying install code, entries, or DD/data.**

## Direct consequences for the verb/schema design

These numbers tighten the proposal from "design for review" to "design with a
measured target":

- **`uninstall` MUST be class-aware by default**, because the default-shaped patch
  (72%) is the one snapshot/restore would corrupt. Auto-restore is only ever
  eligible for the 28% routine-only class — and only after the static probe
  confirms it. (Proposal §"Make `uninstall` honest, per class".)
- **The back-out contract is the common case, not the exception.** A side-effecting
  patch (72%) must ship an authored back-out (the `VSLTAPBO` generalization) or be
  declared class-3 forward-only. v-pkg cannot author the inverse for the majority.
- **Snapshot must cover entries + DD/data, not just routines** — **50% touch FileMan
  entries** and 24% touch a file, frequently alongside routines, so a routine-only
  pre-image is incomplete for well over half the corpus. (Proposal open-question #2,
  now answered: yes, multi-component pre-image is mandatory, not optional.)
- **Forward back-out scaffolding pays for itself.** With 72% of patches having no
  generic inverse, "scaffold a forward back-out build from recorded provenance" is
  the *primary* reversal path for real patches, not a fallback.

## Reproducing

```sh
# corpus (shallow clone, ~480 MB on disk):
git clone --depth 1 https://github.com/WorldVistA/VistA ~/data/kids-patches/VistA
python3 ~/data/kids-patches/analyze.py   # writes analysis-report.txt
```

The analyzer and the cloned corpus live under `~/data/kids-patches/` (the
persistent-data lake, not committed). This findings doc is the committed artifact;
re-running the analyzer against a refreshed clone should reproduce these figures
within sampling noise.
