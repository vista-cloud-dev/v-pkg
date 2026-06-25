---
title: "v-pkg: patching EXISTING routines (snapshot / restore / drift) — not just greenfield install"
status: PROPOSAL (2026-06-25) — design for review; not yet implemented
created: 2026-06-25
for: extending the v-pkg verb contract + build-spec schema so a build can safely PATCH an existing national routine and be reversed to stock
related: v-stdlib RPC-broker splice (VSLRPCWRAP / CALLP^XWBBRK), v-stdlib VSLTAPBO + FU-21 (the per-package hack this generalizes), docs/fileman-dd-install-plan.md
---

# v-pkg: patching existing routines, not just greenfield install

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
- **`uninstall` becomes pre-image aware** — **delete** greenfield components,
  **restore** patched ones from the captured pre-image. This is the bug fix, not
  just a feature: today's uninstall would brick the broker.
- **`verify`/`drift` extension** — answer "is my patch still applied to the live
  routine?" so a *later* national patch overwriting our splice is **detected**
  (generalizes FU-21's re-pin hook). Fits the org's registry-driven, drift-gated
  philosophy: the patch-applied check is a gate.

### Two back-out models (support both)

- **Snapshot / restore** — for dev / lab (vehu-style). Fast, local, exact stock
  restore. What the tap smoke-test needs *now*.
- **Forward back-out patch** — for production fidelity, mirroring real VistA: the
  reversal is itself a *forward* patch (cumulative, forward-only), not an
  uninstall. v-pkg should be able to emit one from a snapshot.

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

## Why now

This is a **verb-contract + build-spec-schema** change, so it deserves a decision
before more existing-routine patches land ad hoc. The XWBBRK splice is the first;
with 40k national routines, it will not be the last. Generalizing the tap's
per-package back-out into a v-pkg primitive is the difference between "every patch
re-invents its own reversal" and "patching an existing routine is a safe,
first-class, reversible v-pkg operation."
