---
name: multi-field-dd-emitter
description: "2026-06-28: v-pkg now AUTHORS a multi-field FileMan FILE DD (the 5 grounded scalar field types beyond .01) — coverage-analysis item B.2-a. Build-side only; not yet live-install-proven. Unblocks v-stdlib R3 (the VSL AUDIT file)."
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
  (ADD-IF-NEW/MERGE/OVERWRITE/REPLACE) — the rest of B.2; relax the test-range
  file-number restriction (permanent-number namespace policy needs org
  coordination); and it is **NOT yet live-install-proven on an engine** — the
  single-`.01` lifecycle was proven on both engines, but a multi-field DD has not
  been installed live yet (next step: install→verify→uninstall→file-a-record on
  vehu + foia-t12 via the driver stack).

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
  - pointer:   `LABEL^P{file}'^{pointed-root}^0;P^Q`
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
