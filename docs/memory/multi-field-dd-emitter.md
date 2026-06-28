---
name: multi-field-dd-emitter
description: "2026-06-28: v-pkg AUTHORS a multi-field FileMan FILE DD (5 grounded scalar field types beyond .01) — coverage-analysis B.2-a. LIVE-INSTALL-PROVEN on both engines (a pointer-piece caret bug was found+fixed). Unblocks v-stdlib R3 (the VSL AUDIT file)."
metadata:
  type: project
---

# v-pkg: multi-field FileMan DD authoring — coverage-analysis B.2-a (2026-06-28)

Generalizes the single-`.01` FILE DD emitter [[fileman-dd-component]] to **multiple
typed fields**, the R3 enabler from the coverage-analysis proposal
(`docs/proposals/v-pkg-kids-coverage-analysis.md`, Track B item B.2).
TDD; landed on `main` (trunk-based). Unblocks v-stdlib R3's multi-field `VSL AUDIT`
file.

## Scope (what B.2-a does and does NOT do)
- **DOES:** emit a new file's DD with the `.01` free-text NAME **plus N typed
  fields**. Five grounded scalar types: **free text / numeric / date / set of
  codes / pointer**, each with `required` and an optional reader-help (`,N,3)`).
- **DOES NOT (still open):** ship file **DATA** + the 4 action codes
  (ADD-IF-NEW/MERGE/OVERWRITE/REPLACE) — the rest of B.2 (B.2-b); relax the
  test-range file-number restriction (permanent-number namespace policy needs org
  coordination).

## LIVE-INSTALL-PROVEN on both engines (2026-06-28)
Built via `v pkg build`, installed via `v pkg install` on vehu (YDB) + foia-t12
(IRIS): the multi-field DD (#999001 ZZVSL AUDIT, all 5 typed fields) reaches **#9.7
status 3**, registers in the dict-of-files (`^DIC("B")`), and **files a record**
exercising every type — `EVT-A^42^W^3060628.1430^1^boot complete` stored at the
right pieces, `.01` "B" xref built, `UPDATE^DIE` dierr=0. `v pkg verify` confirms.

**Bug found + fixed by the live-prove — the pointer-piece caret.** The pointer
field def node shipped as `USER^RP200'^^VA(200,^0;5^Q` (empty piece 3, root in
piece 4) instead of `USER^RP200'^VA(200,^0;5^Q`. Cause: `FileField.PointRoot` is
stored **with** a leading `^` (the buildspec global-root regex requires one), but
in the DD piece-3 the root has **no** caret (real fields: `STATE^P5'^DIC(5,^.11;5^Q`)
— a literal `^` is the piece delimiter, so it injected an empty piece and shoved
the storage location into piece 5 → `FIA^XPDIK` faulted with **NULSUBSC** and the
install stuck at **status 2** (DD filed but file left half-registered, no
`^DIC("B")`). Fix: `fieldDef` strips a leading `^` (`strings.TrimPrefix`). Status-2
with the DD present is the tell-tale of a malformed field def in this path.

**Known minor gap (not a blocker):** the pointed-to file's back-reference
`^DD(200,"PT",999001,4)` is NOT created (the FIA transport ships only the new
file's DD, not a modification to #200's). Records with pointer values file fine;
the missing PT xref affects reverse-navigation / delete-protection only. A future
item if full pointer fidelity is needed (would ship the PT node for the target).

## Grounded grammar (verbatim from a real new-file full DD: #8992.7 LOG4M CONFIG, Log4M_2p4.KID)
- **Header** `^DD(F,F,0)` = `FIELD^^<highest-field#>^<field-count>` — piece 1 is the
  **literal `FIELD`**, NOT the file name (confirmed across #8992.7/#125/#811.4). The
  proposal §8 wrote `<NAME>^^…`; that was a simplification — the existing tested
  emitter (`FIELD^^.01^1`) was right.
- **Field def** `^DD(F,F,<fld#>,0)` = `LABEL ^ TYPE ^ (set-list|ptr-root) ^ node;piece ^ input-transform`:
  - free text: `LABEL^F^^0;P^K:$L(X)>{max}!($L(X)<1) X`
  - numeric:   `LABEL^NJ{w},{d}^^0;P^K:+X'=X[!(X>{max})][!(X<{min})]!(X?.E1"."{d+1}.N) X`
  - date:      `LABEL^D^^0;P^S %DT="{E|ET}" D ^%DT S X=Y K:Y<1 X` (ET = with time)
  - set:       `LABEL^S^{int:ext;…;}^0;P^Q` (list is **`;`-terminated**)
  - pointer:   `LABEL^P{file}'^{pointed-root}^0;P^Q` — pointed-root in piece 3 has
    **NO leading `^`** (`DIC(5,`, `VA(200,`); the emitter strips the buildspec's caret.
  - **required** prefixes the type letter with `R` (RF / RS / RP{file}'…).
- **No `"GL"` map node per field** — real exports carry storage **inline** in piece 4
  of the def node. (The pre-existing `.01` `"GL","0;1",1,.01` node stays; new fields
  add none.) Real exports also ship `,N,"DT")` (a date — **omitted**, volatile, would
  break the deterministic-build invariant) and `,N,21,…)` WP descriptions (omitted —
  optional; only `,N,3)` help is supported).

## Where it lives
- `internal/kids/filecomp.go`: `FileDD.Fields []FileField`, `FileField`/`SetCode`,
  field-type consts, `fieldDef()` (the 5-piece builder), `mNum()` (M-canonical
  value-string number — strips the leading 0 of a fraction: `0.01`→`.01`, so the
  header `.01` and pointer `P200'` render right; **subscripts keep `0.01`** via
  `formatKIDSFloat`). Fields emitted in field-# order (deterministic).
- `internal/buildspec/buildspec.go`: `FileComp.Fields []FieldSpec` + `SetCodeSpec`,
  `validateFields`/`validateFieldType` (number > .01 & unique, label uppercase,
  known type, storage `node;piece` non-colliding with the reserved `0;1`,
  type-specific knobs; free-text maxLen ≤ 245).
- `pkgcli/build.go`: `resolveFields` maps spec→kids (storage carries straight
  through; spec is pre-validated).

## Gates (all green)
lint clean; `make test` (race) all pass (kids cov 83.9%); **corpus round-trip
DRIFT=0 over 2,404 dists** (losslessness preserved). New golden
`testdata/zzvslaudit/ZZVSLAU.kids` (#999001 ZZVSL AUDIT, all 5 field types) +
deterministic gate `TestBuild_ZZVSLAU_Deterministic`. Multi-field unit gate
`TestMakeBuildPairs_File_MultiField`.

Companion to [[fileman-dd-component]] (single-.01 lifecycle, live-proven) and
[[krn-param-def-component]]; roadmap item per the coverage-analysis proposal
(`docs/proposals/v-pkg-kids-coverage-analysis.md`). Engine
proof, when done, goes through [[engine-access-through-driver-stack]].
