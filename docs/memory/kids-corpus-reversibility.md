---
name: kids-corpus-reversibility
description: Empirical KIDS-corpus evidence (2,404 patches) grounding the v-pkg reversibility taxonomy — pure-overwrite is the 35% minority, 64% side-effecting.
metadata:
  type: project
---

**KIDS corpus analysis (2026-06-25)** — to ground [[fu5b2-xwbbrk-wrapsplice]]'s
patch-reversibility design, cloned the full **WorldVistA/VistA** patch corpus and
statically parsed every `.KID` BUILD (#9.6) global. Findings doc:
`v-pkg/docs/kids-corpus-findings.md`; proposal it grounds:
`v-pkg/docs/patch-existing-routines-proposal.md`. Analyzer + corpus live in
`~/data/kids-patches/` (analyze.py + the clone — persistent-data lake, NOT
committed; findings doc is the committed artifact).

**Headline (2,404 distributions, 14,288 routine instances):**
- **35%** ship routines ONLY (no install code / no FileMan entries / no DD-data) —
  the *only* class where snapshot/restore is a sound undo (class 1). The XWBBRK
  splice that motivated the work is in this minority.
- **64%** are side-effecting (class 2/3, no generic inverse): **51%** run
  install-time code (Env-Check 13% / Pre 4% / Post 12%), **23%** file FileMan
  entries (options/RPCs/protocols/params/keys), **23%** ship a FileMan FILE
  (DD/data). **96%** declare required-build deps.

**Why it matters / how to apply:** this is the corpus confirming the user's
correction — VA's forward-only back-out is *forced* by what patches do, not a tool
limitation. So in v-pkg: `uninstall` MUST be class-aware by default (auto-restore
only for the 35% routine-only class, after a static probe confirms it); the
authored back-out contract (generalize VSLTAPBO) is the *common* path; snapshot
must cover entries+DD/data, not just routines; forward-back-out scaffolding is the
*primary* reversal path for real patches. The reversibility class is statically
derivable from the `.KID` (4 node-form probes: routine-only? install code? KRN
entries? FILE-multiple?) → the source-tag→registry→red-gate input.

**Node forms (for re-running / extending the analyzer):** routine = `"RTN","<name>")`
(name node, no `,0)`); exported FileMan entry = top-level `"KRN",<file#>,<ien>,0)`;
FileMan FILE/DD = `"BLD",<n>,4,<file#>,0)` (file numbers are decimals — must allow
`.` in the regex); install code = `Environment Check`/`Pre-Install`/`Post-Install`
strings. The build-DEFINITION nodes are `"BLD",<n>,"KRN",<file#>,...` (a header per
component type) — do NOT confuse with the top-level exported `"KRN",...` records.
