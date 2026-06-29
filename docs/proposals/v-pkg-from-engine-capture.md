---
title: "v-pkg --from-engine: read-live capture for the compiled-FileMan component family (templates / forms / dialogs)"
status: proposed
created: 2026-06-29
last_modified: 2026-06-29
revisions: 1
grounding:
  - "v-pkg source — internal/kids/entrycomp.go (generic KRN emitter), internal/installspec/script.go (read-live markers), pkgcli/snapshot.go (routine pre-image capture), internal/buildspec/buildspec.go (HEAD 2026-06-29)"
  - "coverage analysis — docs/proposals/v-pkg-kids-coverage-analysis.md (B.1 templates DEFERRED; corpus frequencies)"
  - "KIDS DIFROM behaviour — Kernel 8.0 Developer's Guide: KIDS UG (XU/krn_8_0_dg_kids_ug)"
related:
  - v-pkg-kids-coverage-analysis.md
  - v-pkg-install-fidelity-spike.md
  - ../archive/implementation-plan.md
---

# v-pkg `--from-engine`: read-live capture for the compiled-FileMan family

## Why this exists

The coverage analysis (B.1) landed ~15 KRN entry-component types that all author
from a **declarative spec** — OPTION, PROTOCOL, RPC, SECURITY KEY, MAIL GROUP,
LIST TEMPLATE, HELP FRAME, the HL7 family. It then hit a hard wall and **deferred
the template/form family** for one reason:

> "The transport mechanics generalize (one DIFROM ORD-tail covers all four +
> FUNCTION/DIALOG/BULLETIN), but the record image carries **compiled FileMan
> structures** (the `"DR"` edit string with embedded MUMPS, `"DIAB"` nodes,
> ScreenMan FORM/BLOCK subtrees) NOT derivable from a declarative spec. Needs a
> **read-live capture** image source (`--from-engine`)." — coverage analysis, B.1
> templates DEFERRED

This document scopes that read-live capture: the missing **image source**, not a
new transport. Everything downstream of the image — the `KRN` manifest, the ORD
install tail, the merge-and-file, verify, uninstall — already exists and is
type-agnostic.

## The target family and its reach

These are the spec-underivable types, in corpus-frequency order (per the
coverage analysis re-tally over 2,404 distributions):

| File # | Component | dists | % | Why not declarative |
|---|---|---:|---:|---|
| .4   | PRINT TEMPLATE  | 489 | 20%  | compiled `"DR"`/print-field image + embedded M |
| .402 | INPUT TEMPLATE  | 246 | 10%  | compiled `"DR"` edit string with embedded M |
| .401 | SORT TEMPLATE   | 114 | 4.7% | compiled sort/`"DIAB"` logic |
| .403 | FORM (ScreenMan) | 78 | 3.2% | nested FORM/BLOCK (#.404) subtree, compiled |
| .84  | DIALOG          | 53  | 2.2% | compiled dialog text + parameters |
| 3.6  | BULLETIN        | 50  | 2.1% | text + recipient logic (borderline — see §"Scope edges") |
| .5   | FUNCTION        | 34  | 1.4% | compiled function code |

The print/input/sort template trio alone is the dominant lever (~the next 20% of
authorable distributions after B.1's entry types). FORM pulls in its child BLOCK
(#.404) subtree.

## The core idea — one new image source, zero new transport

The generic emitter models a record as an ordered list of image nodes
(`internal/kids/entrycomp.go`):

```go
type imageNode struct { tail Subs; val string } // subscripts below "KRN",<file>,<seq>, + value
type entryRec  struct { name, xpdfl string; image []imageNode }
```

Every existing type produces `[]imageNode` from a Go struct (e.g.
`optionRecords()`). `--from-engine` adds a **second producer of the same
`[]imageNode`**: instead of packing fields from a spec, it reads the live
record's stored subtree off an engine and turns each `(subscript, value)` into an
`imageNode`. The manifest header, ORD tail, install merge, verify, and uninstall
are unchanged — they already iterate the subscripts opaquely.

`decompose` confirms the transport generalizes: it already maps these file
numbers (`internal/kids/decompose.go:549` — `.4 PRINT-TEMPLATE … .403 FORM .404
BLOCK`) and round-trips them losslessly out of an existing `.KID`. A live capture
is the same image, sourced from `^DIPT(…)`/`^DOPT(…)`/… instead of a `.KID` ZWR
block.

## The determinism problem — capture is a separate, explicit step

A `v pkg build` is **offline and byte-deterministic** (golden-tested, corpus
DRIFT=0). Reading a live engine mid-build would break that. The resolution
mirrors how routine pre-images are already handled (`pkgcli/snapshot.go` writes a
committed `<kid>.preimage.kids` sidecar): **split capture from build.**

```
  v pkg capture --engine ydb|iris \
                --type "PRINT TEMPLATE" --name ZZFOO \
                --out src/captured/zzfoo.print-template.json   # touches the engine, ONCE
  # → commit the captured image artifact into the package source tree

  v pkg build  ...                                             # offline, deterministic,
                                                               # consumes the committed image
```

- **`v pkg capture`** is the *only* verb that reads a live engine for authoring.
  It emits a frozen, human-diffable image artifact (the `[]imageNode` list +
  type + captured-from provenance).
- **`v pkg build`** gains a buildspec key (e.g. `capturedComponents` or a
  per-type `from: <path>` field) that loads those committed artifacts and feeds
  them straight into the generic emitter. The build stays offline, reproducible,
  and golden-testable; the captured artifact is the reproducibility anchor (like
  a vendored fixture).

This keeps the byte-deterministic build invariant intact and makes the
engine-dependent step explicit, reviewable, and version-controlled.

## Waterline — the key decision (needs your call)

Reaching the engine stays **only** through `mdriver.Client` (rule 3, the hard
transport monopoly) — `capture` stages a `ZVPKG*` scratch routine via
`exec load`/`exec run` and reads `<<VPKG>>…` markers back, exactly like every
other verb (`internal/installspec/script.go`, `pkgcli/lifecycle.go:runMScript`).
That part is non-negotiable and already proven.

The open question is the **read method inside that script:**

- **(a) DBS API (`$$GET1^DIQ`, `RETRIEVE^DILF`) — convention-pure but insufficient.**
  The org soft-convention is "read through the DBS API, never direct globals."
  But the DBS API returns *resolved field values*; it has **no method that
  returns the compiled `"DR"` edit string, the `"DIAB"` nodes, or a ScreenMan
  FORM/BLOCK subtree** — the very nodes that make this family spec-underivable. A
  DBS-only capture would silently lose the compiled image.
- **(b) Bounded `$QUERY` walk of the single entry's data subtree — faithful, and
  what KIDS itself does.** Real KIDS `DIFROM` reads these globals directly to
  build the transport. A read-only `$QUERY` loop scoped to **one entry's subtree**
  (resolve `name→IEN` via the `"B"` xref, then walk `^DIPT(IEN,…)` etc., emitting
  one marker per node) captures the literal stored image — exactly what the KRN
  transport must ship. The existing verify/snapshot scripts already read live
  state with `$D(^…)`/`$T(…)` probes, so a narrow read-only global walk is
  consistent with current practice.

**Recommendation: (b)**, framed explicitly as *"capture reads the entry's stored
image the same way KIDS DIFROM does — a bounded, read-only, single-entry global
walk over `mdriver.Client`."* The transport monopoly (the actual hard gate) is
untouched; only the soft DBS-read convention is bent, for the one case where it
is technically impossible to honor. **This is the decision to confirm before
building** — it sets precedent for any future read-live authoring (B.1 risk note
in the coverage analysis).

## Per-type storage globals — ground-truth at implementation time

The capture walk needs each type's storage global + the exact subtree to ship.
These must be **ground-truthed against a real KIDS export** at build time (the
B.1 method — never assert a node grammar unproven), but the starting map:

| Type | File # | Storage global (confirm) | Notable image nodes |
|---|---|---|---|
| PRINT TEMPLATE | .4 | `^DIPT(IEN,…)` | `0`-node, `"DR"`, print-field multiple, compiled lines |
| INPUT TEMPLATE | .402 | `^DIE(IEN,…)` | `0`-node, `"DR"` edit string, `.01` |
| SORT TEMPLATE  | .401 | `^DIBT(…)` *(confirm)* | sort spec, `"DIAB"` |
| FORM           | .403 | `^DIST(.403,IEN,…)` *(confirm)* | + nested BLOCK #.404 subtree |
| BLOCK          | .404 | `^DIST(.404,…)` *(confirm)* | field/caption geometry |
| DIALOG         | .84  | `^DI(.84,IEN,…)` *(confirm)* | text + params |
| FUNCTION       | .5   | `^DD("FUNC",…)` *(confirm)* | compiled function |

`FORM` is the hard one — it transports its child `BLOCK` records, so the capture
must follow the FORM→BLOCK link and ship both (the same multi-record packing
`buildEntryGroups` already does, just sourced live).

## IEN / pointer normalization — reuse the `"^"` resolver lesson

B.1-n/B.1-o established that KIDS transports a pointer as the IEN slot **plus a
parallel `"^"` NAME node**, and a re-filing type re-points from the name node at
install — the IEN is a build-local don't-care
([[option-entry-component]]). Capture must **preserve those `"^"` resolver
nodes** verbatim where the live image has them, and **must not** ship a hard
site-IEN as authoritative. Where a captured node carries a bare IEN with no name
resolver (some template internals), that is an open normalization question —
flag, don't guess.

## Safety — run the captured image through the PIKS gate

A captured template can embed references to Patient/Institution-class files (a
print template over file #2). The existing PIKS data-class gate (`internal/kids/piks.go`,
gate K2) must run over captured images so a capture that would drag PHI-class
structure into a build is refused — the same tripwire that protects extraction
(P4). Wire `capture` to lint its own output before writing the artifact.

## Phasing

1. **Decide the read method (§Waterline)** — recommend (b); confirm before code.
2. **`v pkg capture` skeleton** — name→IEN resolve + bounded `$QUERY` walk for
   **INPUT TEMPLATE #.402** (simplest compiled image: one `"DR"` string), emit
   the artifact; PIKS-gate it. Live-prove capture→build→install→verify→uninstall
   on both engines, golden the artifact + the resulting `.KID`.
3. **PRINT TEMPLATE #.4, SORT TEMPLATE #.401** — same single-record shape.
4. **FORM #.403 + BLOCK #.404** — the multi-record / child-subtree case.
5. **FUNCTION #.5, DIALOG #.84** — tail types; BULLETIN #3.6 see scope edge.

Each tier ships its golden + corpus-round-trip + live install-parity proof (the
B.1 cadence) and its tag→registry→red-gate triple (org contract).

## Scope edges

- **BULLETIN #3.6** is borderline spec-derivable (text + a recipient mail-group
  pointer). It may belong on the *declarative* path (like MAIL GROUP), not the
  capture path — evaluate it against a real export before assigning it here. If
  declarative, it stays a small B.1-style addition and drops off this list.
- **Extended menu-actions** (USE-AS-LINK/MERGE/ATTACH/DISABLE) are already scoped
  out as install-time menu management, not authoring — unaffected by this.
- **Capturing arbitrary entries of already-declarative types** (e.g. capture a
  live OPTION instead of authoring it) falls out of this work for free, but is
  not a goal — the declarative path stays primary for those.

## Open questions

1. **Waterline read method** (§Waterline) — (b) global walk vs (a) DBS. *The
   gating decision.*
2. **Capture artifact format** — standalone JSON sidecar per component (matches
   `snapshot`'s `.preimage.kids` precedent) vs inline in the buildspec. Recommend
   sidecar (diffable, PIKS-gateable, keeps buildspec declarative).
3. **Hard-IEN normalization** for template-internal pointers with no `"^"` name
   resolver — preserve-as-is vs refuse vs resolve-to-name. Needs a real-export
   look.
4. **Re-capture / drift** — should `v pkg verify` compare a built template against
   the live source it was captured from (template-drift, like routine-drift)?
   Likely a follow-up, not v1.
