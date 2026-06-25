---
title: "v-pkg: patching EXISTING routines (snapshot / restore / drift) — not just greenfield install"
status: PARTIALLY IMPLEMENTED (2026-06-25) — keystone `classify` verb landed; snapshot/restore/class-aware uninstall are next (engine-bound)
created: 2026-06-25
for: extending the v-pkg verb contract + build-spec schema so a build can safely PATCH an existing national routine and be reversed to stock
related: v-stdlib RPC-broker splice (VSLRPCWRAP / CALLP^XWBBRK), v-stdlib VSLTAPBO + FU-21 (the per-package hack this generalizes), docs/fileman-dd-install-plan.md
---

# v-pkg: patching existing routines, not just greenfield install

> **Implementation status (2026-06-25).** The keystone everything else gates on —
> static reversibility classification — is **built and corpus-validated**:
> `internal/kids/reversibility.go` (`Classify`/`ClassifyBuild`) + the `v pkg
> classify` verb derive PureOverwrite vs SideEffecting from the `.KID` alone (no
> engine), using four node-shape probes (routine name, install-code subnode
> INI/INIT/PRE/PRET, exported `KRN` entry, FILE-multiple). Cross-checked against
> all 2,404 WorldVistA distributions via `make corpus`: pure-overwrite **36%**,
> side-effecting **63%** — matching the independent analyze.py count (35/64). The
> remaining verbs (`snapshot`/`restore`, class-aware `install`/`uninstall`,
> `verify --drift`) are **engine-bound** and land next, now unblocked by the
> classifier.

## Problem

The current v-pkg verbs assume a **greenfield** package: `install` lays down new
`VSL*` routines; `uninstall` is "routine-only back-out" = **delete** them. That is
correct for a namespaced library shipped from scratch.

But VistA is **~40,000 existing routines**, and the dominant real-world operation
is **patching one of them** — shipping a *modified* copy of a national routine —
not adding greenfield code. KIDS patches do this constantly. The current verbs
have no first-class support for it, and the gap is not cosmetic: it leaves the
target **broken** on reversal.

### The motivating case (lived, 2026-06-25)

The v-stdlib traffic tap needs the RPC broker to call its glue. The mechanism is a
two-line splice into the **national** routine `CALLP^XWBBRK`:

```
 D req^VSLRPCWRAP          ; after the XWBSEC-denial line
 . D resp^VSLRPCWRAP       ; inside the success block, after the CAPI dispatch
```

Delivering that as a patched-`XWBBRK` KIDS artifact and installing it on `vehu`
exposed two concrete failures:

1. **`install` overwrote a national routine with no pre-image.** It clobbered
   vehu's stock `XWBBRK` and kept **no record of the original**. The only reason a
   revert is possible is that the operator (here, by hand) dumped `XWBBRK.stock.m`
   *before* installing. That manual snapshot is the missing primitive.

2. **`uninstall` deletes; it does not restore.** For a greenfield `VSL*` routine,
   delete is right. For a patched-over national routine, **delete leaves the
   system without a broker** — strictly worse than before the patch. `uninstall`
   conflates "remove the thing I added" with "undo the change I made to an
   existing thing."

The tap already felt this and hand-rolled a per-package answer (`VSLTAPBO`
back-out + the FU-21 "restore-to-stock CALLP + re-pin hook"). **That logic should
be a general v-pkg capability, not re-invented per build** — every future
existing-routine patch will need the same thing.

## Reversibility is a property of the whole patch, not the routine

A correction that reshapes the model: **a KIDS patch is not just modified
routines.** It also ships, and *runs*, code and data:

- **Environment check** (`XPDENV`) — runs at load/install; can abort.
- **Pre-install** (`XPDPRE`) and **post-install** (`XPDPOST`) routines — arbitrary
  M that runs around component install: data conversions, cross-reference
  re-indexing, FileMan DD changes, `#8989.51` parameter seeding, mail-group setup,
  queued TaskMan jobs, even outbound HL7 / MailMan messages.
- **Data + DD components** — file entries and data-dictionary changes, each with an
  install *action* (overwrite / merge / update-don't-overwrite). Merges are
  typically lossy w.r.t. the prior state.

This is **why VA's methodology is forward-only.** The install's effect is the
result of running arbitrary code with arbitrary — sometimes *external and
irreversible* — side effects (a sent message can't be unsent; a lossy data merge
can't be un-merged; a queued conversion may already have run). There is **no
generic inverse**, so "undo by putting the old routine back" is not merely
incomplete, it is *unsafe*: it removes the code but leaves every data change and
side-effect the post-install created — often worse than the patched state.

So reversibility is **not a capability v-pkg can assume**; it is a property of the
*specific patch*. The installer's job is to know which class a patch is in and
**never claim more reversal than is true.**

### Reversibility taxonomy (classify, then gate)

> **Corpus-measured shares** (all 2,404 WorldVistA KIDS distributions — see
> [kids-corpus-findings.md](kids-corpus-findings.md)): class 1 = **35%**, class
> 2/3 = **64%**. The pure-overwrite class is the *minority*; generalizing
> snapshot/restore as the undo model would be wrong for ~2 of every 3 real
> patches. 51% run install-time code, 23% file FileMan entries, 23% ship DD/data,
> 96% declare required-build deps. The taxonomy below is not a hypothesis — it is
> the measured shape of the real corpus.

1. **Pure-overwrite — reversible (≈35%).** Only component overwrites; **no**
   env/pre/post code, no data/DD, no side-effects. Fully reversible by pre-image
   restore. *The XWBBRK splice is this class* — which is why snapshot/restore is
   the right answer *for it*, and a trap if generalized.
2. **Side-effecting — reversible only with an authored back-out (the ≈64%
   common case).** Has install-time code and/or data, but the developer ships an
   explicit inverse.
   Reversal = run the patch's own back-out, then restore pre-images, then
   verify-clean. **This is exactly what `VSLTAPBO` already is** (it reverses the
   params / TaskMan / `^XTMP` / `^VSLTAP` footprint the post-install + runtime
   created). Generalize it: a side-effecting patch *must* ship a back-out entry;
   v-pkg orchestrates it but cannot author it.
3. **Forward-only — irreversible.** Effects with no authored inverse, or
   inherently irreversible external effects. v-pkg must **refuse to claim
   reversal**; the only sound back-out is a separately authored *forward back-out
   patch*.

### What this demands of the installer

- **Detect the class from the `.KID`, don't trust the tag.** Derive it from the
  presence of `XPDENV`/`XPDPRE`/`XPDPOST`, data/DD components, and parameter
  components; **red-gate** any build that declares `reversible` but carries
  install-time code or data (source-tag → registry → red-gate, the org pattern).
  A pure-routine patch with no callbacks is the *only* thing eligible for
  auto-`restore`.
- **Snapshot covers every component a patch touches** — routines *and* the
  pre-image of any DD/data/param it overwrites — and is **honest that it cannot
  capture the effects of arbitrary post-install code.** For class 1 the snapshot
  is complete; for class 2/3 it is provenance/audit, not a restore guarantee.
- **Standardize the back-out contract (generalize `VSLTAPBO`).** A side-effecting
  patch declares, in its build spec, a back-out entry (e.g. `backout^<RTN>`) and a
  `verifyClean^<RTN>` exit gate. v-pkg *orchestrates* reversal — run back-out →
  restore pre-image components → assert verify-clean — but never authors the
  inverse. No back-out entry on a side-effecting patch ⇒ it is class 3, and v-pkg
  says so.
- **Record full install provenance** (`#9.7` + a v-pkg registry): components, the
  env/pre/post routines that ran, the reversibility class, the snapshot hash, the
  back-out entry. This audit trail is what makes a forward back-out patch
  *authorable* later.
- **Make `uninstall` honest, per class:** class 1 → restore pre-image; class 2 →
  run the declared back-out + restore + verify-clean; class 3 → **refuse**, and
  point the operator at authoring/applying a forward back-out patch. What
  `uninstall` must *never* do on a side-effecting patch is today's behavior —
  silently remove routines and orphan the data/side-effects.
- **Help author the forward back-out patch.** From the recorded provenance +
  pre-image snapshot, v-pkg can *scaffold* a forward back-out build (the inverse
  components + a stub back-out routine the developer fills in) — turning
  "forward-only" from a dead end into a supported, first-class workflow.

## Proposed model

Make v-pkg **pre-image aware**: a build component is either **greenfield**
(create; reverse = delete) or a **patch** (overwrite-existing; reverse =
restore-to-stock). Capture the pre-image at install time so reversal is always
possible, and detect when a later national patch clobbers ours.

### Build-spec schema

Tag each routine (default `greenfield` to preserve today's behavior):

```jsonc
"routines": [
  "VSLRPCWRAP",                                   // greenfield (implicit)
  { "name": "XWBBRK", "kind": "patch" }           // patches an existing routine
]
```

The `kind` tag drives install/uninstall/verify behavior and is itself
drift-gateable (a `patch` component with no captured pre-image is a red gate).

### Verbs

- **`snapshot`** *(new)* — before patching, extract the **live pre-image** of every
  component the build will *overwrite* into a restorable artifact (itself just a
  `.KID` of the current routines). This is exactly the manual `XWBBRK.stock.m`
  step, promoted to a verb. Output is content-addressed + recorded in `#9.7`.
- **`restore`** *(new, cheap)* — re-apply a snapshot. Mechanically this **is**
  `install <snapshot.kid>` (restore = install-of-pre-image), so it is mostly a
  thin alias + the bookkeeping to mark the patch reversed.
- **`install` gains pre-image awareness** — classify each component greenfield vs
  overwrite; for any `patch` (or any silent overwrite of an existing routine),
  **auto-`snapshot` first**, or refuse without `--snapshot`/`--allow-overwrite`.
  No silent clobber of national code.
- **`uninstall` becomes class-aware** (see the reversibility taxonomy above) —
  **delete** greenfield components; **restore** class-1 patched ones from the
  pre-image; **run the declared back-out + verify-clean** for class 2; **refuse**
  for class 3 (forward-only) and point at a forward back-out patch. This is the bug
  fix, not just a feature: today's uninstall would brick the broker (class 1) and,
  worse, silently orphan data/side-effects (class 2/3).
- **`verify`/`drift` extension** — answer "is my patch still applied to the live
  routine?" so a *later* national patch overwriting our splice is **detected**
  (generalizes FU-21's re-pin hook). Fits the org's registry-driven, drift-gated
  philosophy: the patch-applied check is a gate.

### Back-out path is class-driven (per the taxonomy above)

The reversal mechanism is **not** one model — it follows the patch's reversibility
class:

- **Class 1** → snapshot / restore (exact stock restore). What the XWBBRK smoke
  test needs *now*.
- **Class 2** → run the patch's authored back-out (`VSLTAPBO`-style) + restore +
  verify-clean.
- **Class 3** → a separately authored **forward back-out patch** (cumulative,
  forward-only), which v-pkg can *scaffold* from the recorded provenance + snapshot
  but cannot auto-generate.

## Open questions

1. **Snapshot storage / provenance.** A `.KID` pre-image on disk, a `#9.7`
   install record, a content hash in a registry — or all three? (The org leans
   "generated, drift-gated artifact," so probably a recorded, hashed pre-image.)
2. **Multi-routine / non-routine components.** Pre-image capture must also cover
   `#8989.51` params, DDs, options, etc. — any component a patch overwrites, not
   just routines.
3. **Re-pin trigger.** Is `verify --drift` operator-run, scheduled (TaskMan), or
   wired into a gate? FU-21 wanted a hook on every XWB patch.
4. **`restore` vs `uninstall` surface.** Keep `restore` as a distinct verb, or
   fold it into a pre-image-aware `uninstall`? (Leaning: `uninstall` does the
   right thing automatically; `restore` stays for the "put stock back without
   removing my greenfield routines" case.)
5. **Idempotence + partial installs.** Re-running snapshot/restore must be safe
   (the VSLTAPBO `$text()`-guarded, fault-fenced pattern is the model).
6. **Class detection fidelity.** How reliably can v-pkg derive the reversibility
   class from a `.KID` (env/pre/post routines, data/DD, params)? Anything it can't
   classify with confidence must default to **forward-only** (fail safe — never
   over-claim reversibility).
7. **Back-out contract location.** Should the `backout`/`verifyClean` entry names
   live in the build spec, in a `#9.7`-recorded convention, or both? And should a
   missing back-out on a side-effecting patch be a *build-time* gate (refuse to
   build) or an *install-time* warning?

## Why now

This is a **verb-contract + build-spec-schema** change, so it deserves a decision
before more existing-routine patches land ad hoc. The XWBBRK splice is the first;
with 40k national routines, it will not be the last. Generalizing the tap's
per-package back-out into a v-pkg primitive is the difference between "every patch
re-invents its own reversal" and "patching an existing routine is a safe,
first-class, reversible v-pkg operation."
