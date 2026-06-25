---
title: "What 2,404 real KIDS patches actually do — corpus evidence for the v-pkg reversibility model"
status: FINDINGS (2026-06-25) — empirical grounding for patch-existing-routines-proposal.md
created: 2026-06-25
corpus: WorldVistA/VistA (github), all Packages/**/*.KID[S] distributions
method: static parse of the KIDS BUILD (#9.6) transport global in each .KID — see ~/data/kids-patches/analyze.py
related: patch-existing-routines-proposal.md (the design these numbers justify)
---

# What 2,404 real KIDS patches actually do

The v-pkg reversibility proposal ([patch-existing-routines-proposal.md](patch-existing-routines-proposal.md))
argued, from the single lived XWBBRK case, that **reversibility is a property of
the whole patch, not the routine** — and that snapshot/restore is sound only for
the narrow "pure routine-overwrite" class. To check that claim against reality we
cloned the entire **WorldVistA/VistA** patch corpus and statically parsed every
KIDS distribution's transport global.

**Corpus:** 2,404 `.KID`/`.KIDS` distributions across 157 packages (the full
WorldVistA mirror). Parser: `~/data/kids-patches/analyze.py` (not committed — it
operates on `~/data`, the persistent-data lake). Each file is the KIDS BUILD
(`#9.6`) global dump; we tally what each ships by node form:
`"RTN","<name>")` = routine; top-level `"KRN",<file#>,<ien>,0)` = an exported
FileMan entry; `"BLD",<n>,4,<file#>,0)` = a FileMan FILE (DD/data) component;
`Environment Check`/`Pre-Install`/`Post-Install` = install-time code.

## Headline numbers

| What the patch ships / does | Distributions | % |
|---|---:|---:|
| Ship ≥ 1 routine | 2,343 | **97%** |
| **Routines ONLY** — no code, no entries, no DD/data | **847** | **35%** |
| **Side-effecting** — install code and/or FileMan entries and/or DD/data | **1,557** | **64%** |
| Ship install-time CODE (Env-Check / Pre / Post) | 1,235 | **51%** |
|  ⤷ Environment Check routine | 317 | 13% |
|  ⤷ Pre-Install routine | 109 | 4% |
|  ⤷ Post-Install routine | 293 | 12% |
| Ship FileMan entries (options/RPCs/protocols/params/keys/…) | 559 | 23% |
| Ship a FileMan FILE (DD and/or data) | 576 | 23% |
| Declare required-build dependencies (`"REQB"`) | 2,318 | **96%** |

Total routine instances shipped across the corpus: **14,288**. Multi-build
distributions in this single-build slice: ~0% (multi-builds live under
`Packages/MultiBuilds/` and are counted as their constituent builds).

### FileMan component types shipped (top of the long tail)

| File # | Component | Distributions | % |
|---|---|---:|---:|
| 19 | OPTION | 323 | 13% |
| 8994 | REMOTE PROCEDURE | 144 | 5% |
| .402 | SORT TEMPLATE | 123 | 5% |
| 101 | PROTOCOL | 96 | 3% |
| 8989.51 | PARAMETER DEFINITION | 74 | 3% |
| 19.1 | SECURITY KEY | 74 | 3% |
| 409.61 | OE/RR component | 57 | 2% |
| .4 | FORM | 54 | 2% |
| 3.8 | MAIL GROUP | 37 | 1% |

Each of these is a **data write into a FileMan file** that KIDS uninstall does
*not* reverse — installing an OPTION (#19) entry or a PARAMETER DEFINITION
(#8989.51) leaves a permanent row. Restoring a routine pre-image does nothing for
these.

## What this proves for v-pkg

1. **The pure-overwrite class is the minority — 35%, not the norm.** Only 847 of
   2,404 patches ship routines *and nothing else*. These are the only distributions
   for which "snapshot the routine, restore it to undo" is a *complete and sound*
   reversal. The XWBBRK splice that motivated the proposal is in this 35% — which is
   exactly why snapshot/restore is right *for it* and a trap if generalized. **A
   v-pkg that treated snapshot/restore as THE undo model would be wrong for ~2 of
   every 3 real patches.**

2. **The majority — 64% — are side-effecting and have no generic inverse.** 51%
   run arbitrary install-time M (Env-Check/Pre/Post); 23% file FileMan entries; 23%
   ship DD/data. For these, putting the old routine back removes the *code* but
   orphans every data row, DD change, parameter, queued job, and message the install
   produced — often leaving the system *worse* than the patched state. This is the
   corpus confirming, at scale, the user's correction: **VA's forward-only
   methodology is not a limitation of their tooling; it is forced by what patches
   actually do.** There is no inverse to compute because the forward step ran
   arbitrary, sometimes externally-irreversible, code.

3. **Patches are almost never standalone — 96% declare required builds.** Reversal
   and install both occur inside a dependency chain; a v-pkg uninstall that ignores
   `"REQB"` ordering is unsafe. Reversibility must be reasoned about over the chain,
   not the single build.

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
  (64%) is the one snapshot/restore would corrupt. Auto-restore is only ever
  eligible for the 35% routine-only class — and only after the static probe
  confirms it. (Proposal §"Make `uninstall` honest, per class".)
- **The back-out contract is the common case, not the exception.** A side-effecting
  patch (64%) must ship an authored back-out (the `VSLTAPBO` generalization) or be
  declared class-3 forward-only. v-pkg cannot author the inverse for the majority.
- **Snapshot must cover entries + DD/data, not just routines** — 23% touch FileMan
  entries and 23% touch a file, frequently alongside routines, so a routine-only
  pre-image is incomplete for nearly half the corpus. (Proposal open-question #2,
  now answered: yes, multi-component pre-image is mandatory, not optional.)
- **Forward back-out scaffolding pays for itself.** With 64% of patches having no
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
