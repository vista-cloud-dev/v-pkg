---
title: "v-pkg FileMan FILE-DD install enabler (VSL M3.T1)"
status: in-progress — scoped + corpus-grounded; implementation pending
created: 2026-06-16
for: teaching `v pkg build/install/verify/uninstall` the FileMan FILE-DD KIDS component
related: docs/memory/krn-param-def-component.md (the KRN sibling), vsl-implementation-tracker.md M3.T1
---

# v-pkg FileMan FILE-DD install enabler — VSL M3.T1

## Goal
Teach `v pkg` to **build and install a brand-new FileMan FILE (its data
dictionary)** as a KIDS component, non-interactively over the driver path, then
let v-stdlib's `VSLFS` re-run its M3 acceptance against a **dedicated throwaway
file** instead of the borrowed `#8989.51`. This is the "DD install" the M3
milestone bundles (kickoff option **(b)**, the R4 FileMan-impedance track that
was deliberately deferred so VSLFS could go green first on a borrowed file).

## What already exists (do not rebuild)
- **KRN component path is built** (T1.3, `docs/memory/krn-param-def-component.md`):
  `#8989.51` PARAMETER DEFINITION ships via the KRN mechanism (`KRN^XPDIK`), with
  the non-interactive direct-populate install seeding `^XPD(9.7,XPDA,"KRN")` from
  the BLD manifest. Routine-only builds stay byte-identical. **This is the
  template to mirror** for the file component.
- **`buildspec.Components.Files []FileComp`** exists (`{Number float64, Name
  string}`) and `empty()` counts it — but `Validate()` does **not** validate it,
  and nothing emits a file component into the transport. A `FileComp` is only a
  *reference* (file #, name); it carries no DD nodes.
- **The assembler already reads a file's DD** when present:
  `internal/kids/assemble.go` adds `Files/<dir>/DD.zwr` (and `Data.zwr`);
  `decompose.go` splits the FileMan sections (`FIA`, `^DD`, `^DIC`, `SEC`, `UP`,
  `IX`, `KEY`, `KEYPTR`). So the *assembly* side can carry DD nodes if authored;
  the *authoring* and *install-seeding* sides are the new work.

## Corpus grounding (2026-06-16, via corpus-researcher — cite before coding)
FM 22.2 DG (`DI/fm22_2dg`) + Kernel 8.0 KIDS UG (`XU/krn_8_0_dg_kids_ug`) + Kernel
TM (`XU/krn_8_0_tm`):
- **What a KIDS build transports for a file DD:** the attribute dictionary
  `^DD(file,...)` and dictionary-of-files `^DIC(file,...)` nodes.
  **Cross-references are NOT shipped** — they are rebuilt by reindexing on the
  target (`DI/fm22_2dg/difrom-builds-routines-containing-data-dictionaries`,
  `reindexing-files`). A new file requires a **FULL** DD
  (`XU/krn_8_0_dg_kids_ug/full-ddall-fields`).
- **New-file behavior:** if the target has no file with that number, "the DD is
  installed unconditionally" — the DD-update screen is skipped
  (`data-dictionary-update`, `determining-install-status-of-dds-and-data`). The
  **file number is preserved** (it's the literal `^DD(file,...)` subscript).
- **Installer routine:** **`XPDIK`** = "Install Kernel files **and VA FileMan
  files**" (`XU/krn_8_0_tm/production-account-routines`) — it handles BOTH KRN and
  FileMan files (no separate routine). Load = `EN1^XPDIL`; install =
  `EN^XPDIJ(xpda)` (ICR #2243).
- **DINUM:** file number is stable by construction; *record* IEN stability needs
  `DINUM` in the `.01` input transform (not needed for the VSLFS throwaway file).
- **CORPUS GAP (must read live `XPDIK`/`XPDIL` source via the driver):** the
  literal "FIA" array name and **`XPDIK`'s internal labels + the exact
  `^XPD(9.7,XPDA,...)` sub-nodes it reads for the file component are NOT
  documented.** Reverse-engineer from a real bundled build (as the KRN work used
  `XU*8.0*504`) + live source, exactly like the KRN seed fix was ground-truthed.

## Minimal throwaway DD (corpus-authored) — file #999000 `ZZVSLFS`, data `^DIZ(999000,`
```
^DIC(999000,0)              = "ZZVSLFS^999000"             ; name ^ file#
^DIC(999000,0,"GL")         = "^DIZ(999000,"              ; data global root
^DIC("B","ZZVSLFS",999000)  = ""                          ; dict-of-files B pointer
^DD(999000,0)               = "ZZVSLFS^NM^.01^1"          ; name^NM^highest-field#^field-count
^DD(999000,0,"NM","ZZVSLFS")= ""
^DD(999000,.01,0)           = "NAME^RF^^0;1^K:$L(X)>30!($L(X)<1) X"
                              ; p1 label, p2 RF=required free-text, p4 0;1 store, p5 input xform
^DD(999000,"B","NAME",.01)  = ""
^DIZ(999000,0)              = "ZZVSLFS^999000^^0"          ; name^file#^last-IEN^count
```
(`.01` is uppercase free-text 1–30 chars — transform-invariant, so VSLFS set→get
is byte-identical, same property the borrowed-#8989.51 test relied on.)

## TDD plan (leaf-first; v-pkg only this session, then a v-stdlib re-test session)
1. **buildspec**: validate `Files` (file # in a sane local range, name shape); a
   spec may ship a file with an authored DD. Red test first.
2. **DD authoring → transport**: emit the file component into the KIDS transport —
   the `#9.6 BLD` "FIA"/file manifest + `ORD` install-order rows + the `^DD`/`^DIC`
   image — mirroring `krncomp.go`. Ground the exact node shape from live `XPDIK`
   + a reference build. Deterministic/byte-identical build → golden `.KID`.
3. **install seeding**: the direct-populate `installspec` path seeds whatever
   `^XPD(9.7,XPDA,...)` sub-node `XPDIK`'s file arm reads (the FileMan analog of
   the `"KRN"` seed fix), so `EN^XPDIJ` reaches status 3 with `^DD(999000)`
   present.
4. **verify/uninstall**: verify probes `$D(^DD(999000,0))`; uninstall backs the
   file out (`^DIK` DD-delete or the documented file-deletion path) + the #9.7/#9.6.
5. **live dual-engine** (`vehu` YDB + `foia-t12` IRIS, driver stack only):
   install → `^DD(999000)` present → uninstall → clean.
6. **VSLFS re-test** (separate v-stdlib session): point `VSLFSTST` at #999000
   instead of #8989.51; re-prove 7/7 both engines.

## Risk / scope note
The hardest, highest-risk part is step 2–3 (the undocumented `XPDIK` file-install
internals), which requires live source reverse-engineering — the same depth as
the original KRN-param enabler, which was its own session. This is **not** a quick
tail; it is a genuine v-pkg feature. Tracked as VSL **M3.T1**.
