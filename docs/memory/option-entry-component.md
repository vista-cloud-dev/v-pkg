---
name: option-entry-component
description: v-pkg builds + installs/verifies/uninstalls a #19 OPTION as a KIDS KRN component via a GENERIC entry-component emitter (coverage-analysis B.1), live-proven both engines. Key gotcha — entryNames must match the file-number subscript numerically (int OR float): integer file numbers (#19/#101/#8994) re-parse as int from a .KID, so an IsFloat-only probe silently drops every entry on the live path.
metadata:
  type: project
---

# v-pkg: generic KIDS entry-component emitter + OPTION (#19) — B.1 (2026-06-28)

Coverage-analysis **Track B.1**: generalized the proven #8989.51 KRN mechanism
([[krn-param-def-component]]) into a **generic SEND/DELETE entry-component
emitter** and landed the first type on it — **OPTION (#19)**, the largest
non-routine share of the corpus (39%). `internal/kids/entrycomp.go` is the new
generic core; param-def stays on its own `krncomp.go` functions (migrating it is a
noted follow-up — left untouched to protect the live-proven path + golden).

## The generic core (`entrycomp.go`)
`entryType{number,name,ordTail}` + `entryRec{name,xpdfl,image []imageNode}` →
`emitEntryManifest` (BLD #9.6 `"KRN"` list) + `emitEntryData` (ORD line + top-level
`"KRN",<file>,<seq>` record image) + `(*Build).entryNames(file)`. Same three-part
transport as param-def, parameterized by file number. Each SEND/DELETE type is one
`entryType` with a national-constant `ordTail` (the per-type XPDIK action routines).

## OPTION specifics (ground-truthed: live ^DD(19) + WorldVistA corpus)
- **ORD action-routine line** (national constant, one form across 216 corpus
  builds): `19;<ord>;;;OPT^XPDTA;OPTF1^XPDIA;OPTE1^XPDIA;OPTF2^XPDIA;;OPTDEL^XPDIA`
  — differs from param-def's xref-flag form (`8989.51;<ord>;1;;…`); each type ships
  its own action-routine set, hardcoded from real exports.
- **`-1` XPDFL flag = `0^1`** (send/add-or-update) — NOT param-def's bare `0`
  (option carries a piece 2). Delete-at-site would be `1^1` (a follow-up).
- **Record image / #19 storage**: 0-node `.01 NAME^MENU TEXT^^TYPE`; ROUTINE node
  **25** (run-routine), ENTRY ACTION node **20**, EXIT ACTION node **15**,
  UPPERCASE MENU TEXT node **"U"** (shipped = `ToUpper(menuText)`). TYPE set-of-codes
  `OptionTypeCode`: action A, run routine R, menu M, edit E, inquire I, print P, …
- **NM manifest node** SEND = `<NAME>^^0` — byte-identical to param-def's, so the
  manifest skeleton is genuinely shared.

## THE GOTCHA — entryNames must compare the file number NUMERICALLY (int OR float)
A fresh build emits the file-number subscript via `fltSub` (→ kindFloat). But
`v pkg verify`/`uninstall` load the build from the `.KID` (`ParseKID`→`coerceNum`),
which coerces a **decimal-free number like 19 to an int**. So an `IsFloat`-only
probe (which param-def gets away with because 8989.51 always has a dot) **matched in
the in-memory unit test but returned EMPTY on the live path** — the option silently
absent from verify, never backed out. Fix: `subNum(Sub)` + `entryNames` compares
`subNum(s[1]) == file`, matching int OR float. **Essential for every
integer-numbered entry type (#19, #101 PROTOCOL, #8994 RPC, …)**; #19.1/#3.8 are
floats and would have hidden it. Regression test:
`TestBuild_OptionNames_AfterReparse` (re-parses each emitted subscript).

## Mixing guard (F1 honesty)
The shared `"BLD",1,"KRN",0)` header is not yet computed across multiple entry
types, so `buildspec.Validate` **rejects a build that ships both options AND
parameterDefinitions** rather than emit a wrong header. Generalizing that header
(so one build can carry several KRN types) is the follow-up that also unblocks
migrating param-def onto the generic core.

## Proof
**Live install→verify→uninstall→clean on BOTH engines** via the driver stack
(`testdata/zzoption`, `ZZOPTION RUN ROUTINE`, run-routine → `EN^ZZOPTRT`): vehu
(YDB) + foia-t12 (IRIS). Filed `^DIC(19,…,0)=…^^R`, ROUTINE node 25, "U" xref;
`v pkg verify` confirms (status 3, option present); a side-effecting build so bare
uninstall correctly REFUSES (class-aware), `--force` runs the authored `^DIC(19,`
DIK back-out → fully clean (option/routine/#9.7/#9.6 all gone). Build-side: golden
`testdata/zzoption/ZZOPTION.kids`, corpus DRIFT=0 (2404), lint/race/contract green.
Design decision (author-from-spec vs read-live): chose **author-from-spec** to keep
`v pkg build` offline + deterministic; the generic packer can later take a read-live
image source. See the proposal Track B.1 + [[krn-param-def-component]],
[[multi-field-dd-emitter]], [[kids-coverage-analysis]].
