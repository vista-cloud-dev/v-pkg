---
name: option-entry-component
description: v-pkg builds + installs/verifies/uninstalls a #19 OPTION as a KIDS KRN component via a GENERIC entry-component emitter (coverage-analysis B.1), live-proven both engines. Key gotcha — entryNames must match the file-number subscript numerically (int OR float): integer file numbers (#19/#101/#8994) re-parse as int from a .KID, so an IsFloat-only probe silently drops every entry on the live path.
metadata:
  type: project
---

# v-pkg: generic KIDS entry-component emitter + OPTION (#19) — B.1 (2026-06-28)

Coverage-analysis **Track B.1**: generalized the proven #8989.51 KRN mechanism
([[krn-param-def-component]]) into a **generic SEND/DELETE entry-component
emitter** and landed **OPTION (#19)** (39% of the corpus) on it, then **migrated
PARAMETER DEFINITION onto the same core and unified the manifest header across
entry types** — so one build can ship several KRN types. `internal/kids/entrycomp.go`
is the single generic core; `krncomp.go` keeps only the ParamDef/ReqBuild structs +
REQB/MBREQ/install-hook emitters.

## The generic core (`entrycomp.go`)
`entryType{number,name,ordTail}` + `entryRec{name,xpdfl,image []imageNode}`, grouped
as `entryGroup{et,recs}`. `buildEntryGroups(defs,opts)` collects every KRN type a
build ships, **ordered file-number ascending** — the single place that knows the
build's KRN types. `emitEntryManifest(groups)` writes ONE shared `"BLD",1,"KRN",0)`
header + per-type body; `emitEntryData(groups)` writes per-type ORD (ord = 1-based
group position) + record image; `(*Build).entryNames(file)`. Both OPTION and
PARAMETER DEFINITION ride this; new SEND/DELETE types just append in `buildEntryGroups`.

## Multi-type KRN manifest header (B.1-b, live-proven both engines)
`"BLD",1,"KRN",0)` = `^9.67PA^<last file#>^<type count>`. Real KIDS' "last IEN"
piece is **insertion-order** (a corpus build showed `^9.67PA^779.2^20` where 779.2
is NOT the max of its 20 types) — non-reproducible, so v-pkg uses a **deterministic
`max(file#)` + type-count** rule. This is **cosmetic to the install** (KRN^XPDIK
iterates the subscripts, not the header), confirmed by installing a build with BOTH
an OPTION (#19) and a PARAMETER DEFINITION (#8989.51): header `^9.67PA^8989.51^2`,
option at ORD 1, param-def at ORD 2, both file + verify + back out clean on vehu +
foia-t12. Per-type install order is each type's 1-based group position; the absolute
ord value is irrelevant (independent FileMan files), only the relative order matters,
and XPDIK loops ords ascending. The single-type goldens stay byte-identical
(`^9.67PA^19^1`, `^9.67PA^8989.51^1`) because max+count of one type is that type.
Param-def migration is byte-identical — proven by the unchanged `zzparam` golden,
`TestMakeBuildPairs_ParamDef_KRN`, and corpus DRIFT=0. The old buildspec
"can't-mix-options-and-paramDefs" guard is **removed**. Fixture `testdata/zzmix`.

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

## (resolved) Mixing options + param-defs
The earlier "reject a build mixing options + parameterDefinitions" guard is GONE —
the unified multi-type header (above) makes mixed KRN builds first-class and
live-proven. Next types append in `buildEntryGroups`.

## SECURITY KEY (#19.1) — third type, live-proven both engines (2026-06-28)
Adding a SEND/DELETE type is now ~30 lines: an `entryType` (file#, FileMan name,
national ORD action-routine tail), a `*Records` packer (the record image), the
`buildEntryGroups` line, a `*Names` reader, and the verify/uninstall threading.
SECURITY KEY (`testdata/zzkey`) is the proof-of-cheapness — its record image is
**minimal: `-1`=`0^1` + `0`=the .01 NAME alone** (a key is just a named token;
stored in `^DIC(19.1,`). ORD tail (52 corpus builds, one form):
`;;KEY^XPDTA1;KEYF1^XPDIA1;KEYE1^XPDIA1;KEYF2^XPDIA1;;KEYDEL^XPDIA1`. #19.1 is a
**float** file number so it never hit the int/float subscript gotcha. Shipped
**name-only** (optional DESCRIPTION word-processing field deferred — avoids the WP
volatile-date determinism work). Live install→verify→`--force` uninstall→clean on
vehu (YDB, `^DIC(19.1,797,0)=ZZKEY MANAGER`) + foia-t12 (IRIS). Verify probes
`^DIC(19.1,"B",<name>)`, uninstall is `DIK` on `^DIC(19.1,`. Remaining B.1 types
in frequency order: PROTOCOL #101 → RPC #8994 → templates (.4/.402/.401/.403) →
LIST TEMPLATE #409.61 → MAIL GROUP #3.8 → HL7 family.

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
