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
`^DIC(19.1,"B",<name>)`, uninstall is `DIK` on `^DIC(19.1,`.

## PROTOCOL (#101) — fourth type, live-proven both engines (2026-06-28)
Same node skeleton as OPTION (0-node `NAME^ITEM TEXT^^TYPE`, ENTRY ACTION node 20,
EXIT ACTION node 15) but **its own data global `^ORD(101,`, its own TYPE codes, and
NO "U" xref node** (OPTION's uppercase-menu-text computed field has no #101
analog). TYPE set-of-codes are #101's own (`A`=action, `X`=extended action, `M`=menu,
`E`=event driver, `O`=protocol, `Q`=protocol menu, `L`=limited, `D`=dialog, `T`=term,
`S`=subscriber) — **do NOT reuse #19's TYPE list or its PACKAGE piece (0;14 in #19,
0;12 in #101)**. ORD tail (57 corpus builds): `;;PRO^XPDTA;PROF1^XPDIA;PROE1^XPDIA;
PROF2^XPDIA;;PRODEL^XPDIA`. Authored a **base protocol** (NAME + ITEM TEXT + TYPE +
ENTRY ACTION) — the #101.01 ITEM multiple (menu items) + extended menu-actions
(USE-AS-LINK/MERGE-ITEMS/ATTACH/DISABLE) are a follow-up. Live-proven
(`testdata/zzproto`, action protocol, ENTRY ACTION `Q`) on vehu (YDB,
`^ORD(101,7054,0)=ZZPROTO ACTION^…^^A`, node 20=`Q`) + foia-t12 (IRIS). Verify probes
`^ORD(101,"B",<name>)`, uninstall is `DIK` on `^ORD(101,`. #101 is integer-numbered
(re-parses as int) — the `subNum` numeric-match fix covers it.

## REMOTE PROCEDURE (#8994) — fifth type, live-proven both engines (2026-06-28)
The **simplest record of any type so far**: a single 0-node
`NAME^TAG^ROUTINE^RETURNTYPE` in its own data global `^XWB(8994,` — **no action/exit
nodes, no "U" xref**. `RPCComp{name,tag,routine,returnType}`; `returnType` is a human
name resolved to #8994 field **.04** set-of-codes via `RPCReturnTypeCode` (single
value→1, array→2, word processing→3, global array→4, global instance→5), **defaulting
to "single value" (1)** when omitted — field .04 is DD-required, so an empty value
would file a malformed RPC. ORD tail (corpus modal): `1;;;;;;;RPCDEL^XPDIA1`. `-1`
XPDFL = `0^1` (send). #8994 is integer-numbered → relies on the `subNum` numeric-match
fix. Live install→verify→`--force` uninstall→verify-clean on vehu (YDB) + foia-t12
(IRIS): live `^XWB(8994,…,0)=ZZRPC ECHO^ECHO^ZZRPCRT^1` **byte-identical on both**,
`^XWB(8994,"B",…)` gone after back-out. Verify probes `^XWB(8994,"B",<name>)`,
uninstall is `DIK` on `^XWB(8994,`. Fixture `testdata/zzrpc` (ZZRPC ECHO → `ECHO^ZZRPCRT`,
single value). RPC **input parameters** (#8994.02 multiple) are a follow-up.

## MAIL GROUP (#3.8) — sixth type, live-proven both engines (2026-06-28)
Another simple bespoke-action-routine type (NOT DIFROM — see the template note below).
Stored in `^XMB(3.8,`. Record: `-1)=0^1` (send), `0)=<NAME>^<TYPE>^<SELF-ENROLL>`.
**TYPE (#3.8 field 4, 0;2) is DD-REQUIRED** — set of codes `PU:public/PR:private`,
so it ALWAYS ships (default `PU`); ALLOW SELF ENROLLMENT (field 7, 0;3) is optional
`y/n`. NM node is the plain `<NAME>^^0` (no file# piece — unlike templates). ORD tail:
`;;MAILG^XPDTA1;MAILGF1^XPDIA1;MAILGE1^XPDIA1;MAILGF2^XPDIA1;;MAILGDEL^XPDIA1(%)` —
note the trailing `(%)` on the delete action. **KIDS ships mail groups MEMBER-LESS**:
the #3.81 MEMBER multiple points to site-local #200 (NEW PERSON) entries, added on
site — so a portable build never ships members. The word-processing DESCRIPTION
(field 3, record node `2`, subfile 3.801) is **deferred** — its header carries a
volatile last-edited date (`^3.801^n^n^<FMdate>`) that would defeat the
deterministic-build invariant (same reason KEY's DESCRIPTION is deferred). #3.8 is a
float file number so it never hit the int/float subscript gotcha. Live
install→verify→`--force` uninstall→verify-clean on vehu (YDB) + foia-t12 (IRIS):
live `^XMB(3.8,…,0)=ZZMG ALERTS^PU^y` byte-identical on both, B-index gone after
back-out. Fixture `testdata/zzmg`. Verify probes `^XMB(3.8,"B",<name>)`, uninstall is
`DIK` on `^XMB(3.8,`.

## LIST TEMPLATE (#409.61) — seventh type, live-proven both engines (2026-06-28)
The richest record so far, but still **all plain strings (no compiled structure)** —
so unlike the FileMan template family it DOES author from a spec. **Stored in
`^SD(409.61,` — NOT `^ORD(` (ground-truth the GL node: `^DIC(409.61,0,"GL")`; #409.61
is a List Manager file under the SD namespace, easy to assume wrong).** Record:
`-1)=0^1`, a fixed **14-piece 0-node** (`.01 NAME^.02 TYPE OF LIST^.03 LEFT MARGIN^
.04 RIGHT^.05 TOP^.06 BOTTOM^.07 OK-TO-TRANSPORT^.08 USE-CURSOR-CONTROL^.09 ENTITY^
.1 PROTOCOL MENU^.11 SCREEN TITLE^.12 #ACTIONS^.13 DATE-RANGE^.14 AUTO-DEFAULTS`),
plus the List Manager **callback nodes** at STRING subscripts `HDR`/`INIT`/`FNL`/`HLP`
(M code) + `ARRAY` (the display global ref). The set-of-codes pieces are pinned to the
dominant-corpus values: TYPE OF LIST=1 (PROTOCOL), OK-TO-TRANSPORT=1, USE-CURSOR=1,
#ACTIONS=1, AUTO-DEFAULTS=1. Screen geometry (right/top/bottom margins) defaults to
80/3/20 in `resolveListTemplates`. ORD tail: `1;;;;LME1^XPDIA1;;;LMDEL^XPDIA1` (piece
1 = xref-rebuild flag, then edit `LME1` + delete `LMDEL`). NM node plain `NAME^^0`.
The named-node subscripts come from the DD storage locs (`^DD(409.61,100,0)` HEADER
CODE @`HDR;E1,245`, 106 ENTRY=`INIT`, 105 EXIT=`FNL`, 103 HELP=`HLP`, 107 ARRAY=
`ARRAY`). PROTOCOL MENU (.1) + HIDDEN ACTION MENU (1;2) are #101 pointers — left
optional. Live install→verify→`--force` uninstall→verify-clean on vehu (YDB) +
foia-t12 (IRIS): live `^SD(409.61,…,0)=ZZLM PATIENTS^1^^80^3^20^1^1^^^ZZ Patient
List^1^^1` + `…,"HDR")=D HDR^ZZLMRT` byte-identical on both, B-index gone after
back-out. Fixture `testdata/zzlm`. Verify probes `^SD(409.61,"B",<name>)`, uninstall
is `DIK` on `^SD(409.61,`. The generic core's `imageNode.tail Subs` already supports
string-subscript nodes, so no core change was needed for the callback nodes.

Remaining B.1 types in frequency order: HELP FRAME #9.2 → HL7 family. Templates
(#.4/.402/…) still parked on read-live capture (below).

## HELP FRAME (#9.2) — eighth type, live-proven both engines (2026-06-28)
The first type whose **word-processing body is the point of the type** (unlike KEY's /
mail group's optional, deferred WP DESCRIPTION). Stored in `^DIC(9.2,`. Record:
`-1)=0^1`, `0)=NAME^HEADER` (.01 NAME ^ field 1 HEADER), and the **TEXT word-processing
field** (field 2, subfile 9.21) at **node 1**: header `1,0)=^^<lastSeq>^<count>` + one
node per line `1,<i>,0)`. ORD tail `;;HELP^XPDTA1;HLPF1^XPDIA1;HLPE1^XPDIA1;HLPF2^XPDIA1;;HLPDEL^XPDIA1`
(bespoke-action-routine family). NM node plain `NAME^^0`.

**Two determinism wins worth keeping (the WP-field playbook for future types):**
1. **Ship the WP header date-less** — `^^<n>^<n>`, dropping the volatile `^<FMdate>`
   piece real exports carry. It files cleanly: FileMan reads the line nodes, not the
   header date. Proven live — `^DIC(9.2,IEN,1,1,0)` came back as the shipped text.
2. **Omit the 0-node DATE ENTERED (0;3) + AUTHOR (0;4)** — ship only `NAME^HEADER`.
   **FileMan auto-stamps both at install time** (live record showed
   `…^3260628.221402^1` = today + DUZ), so the volatile data is added on-site, NOT in
   the deterministic transport. This is the correct split: never ship a field FileMan
   will populate itself.

**HELP FRAME names allow hyphens AND spaces** (`YS-PHY-EXAM-NORM`, `PRSA ENTER TW`),
3–30 chars, not all-digit — so they need their own `reHelpName` regex
(`^[A-Z][A-Z0-9 -]*[A-Z0-9]$`), not the space-only `reEntryName`. The RELATED FRAME
(field 3) / INVOKED BY ROUTINE (field 5) multiples are a follow-up. Live
install→verify→`--force` uninstall→verify-clean on vehu (YDB) + foia-t12 (IRIS):
`^DIC(9.2,…,0)` NAME^HEADER + the WP text byte-identical on both, B-index gone after
back-out. Fixture `testdata/zzhf`. Verify probes `^DIC(9.2,"B",<name>)`, uninstall is
`DIK` on `^DIC(9.2,`.

HL7 APPLICATION PARAMETER #771 + HL LOGICAL LINK #870 + HLO APPLICATION REGISTRY
#779.2 done (below). Templates (#.4/.402/…) still parked on read-live capture (below).

## PROTOCOL ITEM multiple — #101.01 + the KIDS "^" pointer-resolver convention (2026-06-28)
Added the #101.01 ITEM multiple to PROTOCOL #101 (field 10, subfile **101.01PA** @
node **10**) — a menu protocol's child list. Per item: data node
`10,<seq>,0)=<placeholder>^^<sequence>^` + a `10,<seq>,"^")=<CHILD NAME>` node.
`ProtocolItem{Name, Sequence}`; `ProtocolItemComp{Name, Sequence}`.

**The durable, broadly-reusable finding (live-proven, decisive experiment):** KIDS
transports a POINTER field in a KRN record as **two parallel nodes** — the pointer
slot carries a (source-system) IEN, and a sibling **`"^"` subscript node carries the
target's external NAME**. On install, a **re-filing** entry type (one whose ORD tail
has pre/post action routines — #101 runs PROF1/PROE1) **re-points the pointer from
the `"^"` name node**, so the IEN slot is a build-local DON'T-CARE. Proof: shipped
`10,1,0)=1^^5^` (placeholder `1`) + `10,1,"^")=ZZPROTO ACTION`; the live item became
`7054^^5^` on vehu and `5649^^5^` on foia-t12 — re-pointed to the sibling ACTION
protocol's real (engine-specific) IEN, filed in the SAME build (intra-build name
resolution). So v-pkg authors menu items by name without knowing target IENs. This
convention generalizes to ANY pointer field in a re-filing type (contrast the #771
COUNTRY / #870 LLP TYPE external-value resolution, which is the *input-transform*
path; this is the explicit `"^"`-node path KIDS uses for #101-pointer-to-#101 etc.).
Fixture `testdata/zzproto` is now self-contained (ACTION + MENU→ACTION). Basic
attach needs NO extended action (the menu attached cleanly without USE-AS-LINK/
MERGE/ATTACH/DISABLE — those remain a minor follow-up, as do OPTION #19.01 menu
items via the same `"^"` convention).

## RPC INPUT PARAMETERS — #8994.02 multiple (2026-06-28)
Extended RPC #8994 with the optional INPUT PARAMETER multiple (field 2, subfile
**8994.02A**, stored at node **2**). Per param: data `2,<seq>,0)=
NAME^TYPE^MAXLEN^REQUIRED^SEQNUM` (.02 TYPE set `1 literal/2 list/3 WP/4 reference`,
.04 REQUIRED set `1 yes/0 no`, .05 SEQUENCE NUMBER), an optional nested DESCRIPTION
WP (subfile 8994.021, header ships **empty-subfile** `^^<n>^<n>` like HELP FRAME),
and **two cross-references the emitter must ship itself**: `2,"B",<NAME>,<seq>)` and
`2,"PARAMSEQ",<seqnum>,<seq>)`. **Why ship the xrefs:** unlike #779.2 (which
re-indexes), the **#8994 install is a VERBATIM KRN MERGE** — its ORD tail
`1;;;;;;;RPCDEL^XPDIA1` has NO gather/pre/post re-file routines, so FileMan does NOT
rebuild xrefs; whatever the image carries is the live record. Live-proven on vehu +
foia-t12: the full param subtree (header, 2 data nodes, date-less description WP, B +
PARAMSEQ xrefs) is **byte-identical** to the shipped image on both — confirming the
verbatim-merge model. Generalized `wpNodes`→`wpNodesAt(prefix Subs, …)` for the
nested per-param WP. `RPCParamComp{Name,Type,MaxLength,Required,Sequence,Description}`;
seq + type default (position / "literal"). Fixture `testdata/zzrpc` now ships 2 params.

## DESCRIPTION word-processing fields — KEY #19.1 / MAIL GROUP #3.8 / #870 (2026-06-28)
Unblocked the three deferred DESCRIPTION WP fields with the date-less WP playbook
(proven with HELP FRAME). Added a shared `wpNodes(node, subfile, lines)` helper in
entrycomp.go — emits header `^<subfile>^<n>^<n>` (DATE-LESS; piece 5 is the volatile
install-stamp a native export carries) + one `<node>,<i>,0)=line`. Refactored HELP
FRAME's inline WP block to call it (subfile `""` → bare `^^<n>^<n>`; golden
unchanged). Field/subfile/node map (ground-truthed): **KEY #19.1** field 1 →
subfile **19.11** at node **1**; **MAIL GROUP #3.8** field 3 → **3.801** at node
**2**; **#870** field 1 → **870.02** at node **3**. Each `*Comp` gained an optional
`description []string`. **Live finding:** the engines file the date-less header
**verbatim** — live header stays `^19.11^1^1` (NO re-stamped date), so it is even
byte-identical in the live global, not just the shipped artifact. Live-proven on
vehu + foia-t12 for all three (install→verify→`--force` uninstall→clean), headers +
text byte-identical. Fixtures `testdata/zzkey|zzmg|zzll` now carry a description.

## HLO APPLICATION REGISTRY #779.2 (eleventh type, 2026-06-28) — first COMPUTED xrefs
The HL7-Optimized (HLO) counterpart to #771: registers an application and maps the
HL7 message types it handles to action routines. Global `^HLD(779.2,`. Record:
`-1)=0^1`, `0)=APPNAME` (.01 free text 3–60), plus the **#779.21 MESSAGE TYPE
ACTIONS multiple** at node 1 — header `1,0)=^779.21I^<n>^<n>`, data
`1,<seq>,0)=MSGTYPE^EVENT^^ACTIONTAG^ACTIONRTN^VERSION`. ORD tail
`1;;HLOAP^XPDTA1;;HLOE^XPDIA1;;;` (no DEL routine). MSG TYPE/EVENT are free text
(`reHL7Code`). This is the **first type with computed cross-reference nodes** — the
emitter ships the xrefs KRN^XPDIK would otherwise rebuild.

**The xref rule (live + corpus proven — got it WRONG first, live-prove caught it):**
each #779.21 entry ships a `"B"` index on MSG TYPE ALWAYS, plus EXACTLY ONE of the
(MSG TYPE, EVENT) lookups — **`"D"` `(MSGTYPE,EVENT,VERSION)` when a version is set,
else `"C"` `(MSGTYPE,EVENT)`. C and D are MUTUALLY EXCLUSIVE.** I first shipped C
unconditionally (misled by a live record that had both); install ground-truth showed
the #779.2 install **RE-INDEXES** (HLOAP/HLOE re-file through FileMan) — it dropped my
stray C and kept B+D. Corpus confirms native KIDS ships B+D for a versioned entry.
The VERSION subscript is **numeric when canonical** (`2.4` unquoted via
`versionSub`→`fltSub`), string otherwise (`"2.5.1"`). Because the install re-indexes,
the live xrefs are byte-identical to the shipped image only when the shipped set
matches FileMan's rebuild — so shipping the right C-XOR-D set is load-bearing, not
cosmetic. Pure-data type → bare uninstall OK (DIK on `^HLD(779.2,`). Fixture
`testdata/zzho`; verify probes `^HLD(779.2,"B",<app>)`. **Live install→verify→
`--force` uninstall→clean** on vehu (IEN 34) + foia-t12 (IEN 35): subtree
byte-identical (`0)=ZZHO_APP`, `1,1,0)=ORU^R01^^START^ZZHORT^2.4`, B + D xrefs),
B-index gone after back-out. This **closes the HL7 family** for v-pkg (#771 + #779.2 +
#870); remaining HL follow-ups are #870 DESCRIPTION WP (#870.02) and #779.2 multi-app
batches.

## HL LOGICAL LINK #870 (tenth type, 2026-06-28)
The HL7 communication-endpoint type. Global `^HLCS(870,`. Record: `-1)=0^1`, sparse
`0)=NODE^^LLPTYPE` (.01 NODE 0;1, field 2 LLP TYPE 0;3) + optional `400)=^PORT^SVC`
(400.02 PORT 400;2, 400.03 SERVICE TYPE 400;3). ORD tail
`1;;HLLL^XPDTA1;;HLLLE^XPDIA1;;;HLLLDEL^XPDIA1(%)` (piece 3 = 1, the data-ships flag,
unlike #771's empty piece). NODE name 3–10 chars, no leading punctuation
(`reLinkName`); SERVICE TYPE set `C`/`S`/`M` (`reSvcType`). Added a generic
`caretJoin(map[int]string)` helper (entrycomp.go) that builds a sparse `^`-node and
trims trailing empties — reused for both the 0-node and 400-node.

**Two durable findings (live-proven on vehu, byte-identical on foia-t12):**
1. **#870 install RE-FILES through FileMan, not a verbatim KRN merge.** Proof: LLP
   TYPE shipped external `"TCP"` lands as IEN **4** — a `#869.1` pointer resolved at
   install (the same external-pointer behavior as #771 COUNTRY `USA→1`). `#869.1`
   (`^HLCS(869.1,`) is nationally controlled — TCP=4 on BOTH engines.
2. **The network endpoint (DNS DOMAIN .08, TCP/IP ADDRESS 400.01) is NOT
   transported — it is site config the install DROPS.** DNS DOMAIN's input transform
   DNS-resolves the host (`$$ADDRESS^XLFNSLK`) and kills itself + the coupled IP
   (`$$IP^HLMA3`) when it can't; a bare TCP/IP ADDRESS is dropped outright too
   (verified IP-only — still dropped). So v-pkg ships ONLY what lands: name, LLP
   type, PORT, SERVICE TYPE (the link definition); the receiving site configures the
   endpoint. **Faithful-transport rule:** never emit a field the install silently
   drops — it would make the build lie. This is the concrete realization of the
   original "#870 is site-specific" deferral note: the link STRUCTURE is portable,
   the ENDPOINT is not. Pure-data type → bare uninstall OK (DIK on `^HLCS(870,`),
   like #771. Fixture `testdata/zzll`; verify probes `^HLCS(870,"B",<node>)`.
   Follow-ups: DESCRIPTION WP (#870.02); HLO registry #779.2 (multiple #779.21 WITH
   computed `"B"`/`"D"` xref nodes — genuinely new emitter capability).

## HL7 APPLICATION PARAMETER #771 (ninth type, 2026-06-28)
The portable, fully spec-derivable member of the HL7 family — `v pkg` ships it
author-from-spec like the other simple types. Global `^HL(771,`. Record image:
`-1)=0^1`, `0)=NAME^a^FACILITY^^^^COUNTRY`. Field map (ground-truthed via
`^DD(771,…`): field 2 STATUS = `a`(ACTIVE)/`i`(INACTIVE) at `0;2` — we always ship
`a`; field 3 FACILITY NAME at `0;3`; field 7 COUNTRY CODE at `0;7` is a `#779.004`
**pointer** — ship the plain `"USA"` and FileMan resolves it to its IEN at install
(live record shows `…^1`, not `…^USA`). Single 0-node only (no multiples shipped).
ORD tail `;;HLAP^XPDTA1;HLAPF1^XPDIA1;HLAPE1^XPDIA1;HLAPF2^XPDIA1;;HLAPDEL^XPDIA1(%)`
(note the `(%)` DEL arg). Names allow spaces AND underscores (`ZZHL_APP`, `VISTA_VTS`)
— own `reHL7Name` (`^[A-Z][A-Z0-9 _]*[A-Z0-9]$`, ≤30 chars). Build-spec: the
`hl7Applications` array `[{name,facility,countryCode}]` REPLACED the old unsupported
`"hl7"` string stub (removed from `unsupported()`, only `"templates"` remains).
`HL7AppComp{Name,Facility,CountryCode}`; resolver defaults CountryCode→`"USA"`.
Fixture `testdata/zzhl`. Verify probes `^HL(771,"B",<name>)`, uninstall is `DIK` on
`^HL(771,`. **Deferred HL7 follow-ups:** #779.2 (HLO message registry — multiples +
xrefs, more than a flat 0-node) and #870 (LOGICAL LINK — carries hardcoded IP/hostname,
site-specific, NOT portable / spec-derivable).

## DEFERRED: the template/form family (#.4/.401/.402/.403) needs read-live, not author-from-spec
Ground-truthed 2026-06-28 (user chose to defer + do MAIL GROUP instead). The transport
MECHANICS generalize fine (the record still ships as `"KRN",<file>,<ien>,…` with a `-1`
flag + an NM node, and ONE DIFROM ORD-tail covers all four template types plus FUNCTION
#.5 / DIALOG #.84 / BULLETIN #3.6: `<file>;<ord>;;;EDEOUT^DIFROMSO(<file>,DA,"",XPDA);
FPRE^DIFROMSI(…);EPRE^DIFROMSI(…);;EPOST^DIFROMSI(…);DEL^DIFROMSK(…)`). The BLOCKER is
the **record image**: unlike the simple types' few caret-pieces, a template carries
**compiled FileMan structures** — the `"DR",1,<file>)` edit string with embedded MUMPS
branching (the DR string IS the template), `"DIAB"` compiled field-action nodes, `"%D"`
description WP, and for many `.402`/`.403` records full ScreenMan FORM/BLOCK subtrees
(`.4031I`/`.4032IP`). These are NOT derivable from a declarative spec. The right path is
a **read-live capture** image source (`--from-engine`: copy a real template's
`^DIE(file,ien,*)` subtree verbatim) — templates are the forcing function for that
capability. NM node for templates also gains a piece: `<NAME>^<FILE#>^<sendflag>`
(file in piece 2). GOTCHA when grepping the corpus: a `"BLD",N,"KRN",<file>,0)` line
with just the file# and a `"NM",0)=^9.68A^^0` header is an EMPTY "all-components"
registration (#9.6 lists every possible component class even when the build ships none
of that type); a REAL shipped entry has `"NM",1,0)` (index ≥1).

Remaining B.1 types in frequency order: LIST TEMPLATE #409.61 → HELP FRAME #9.2 → HL7
family. Templates revisited once read-live capture exists.

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
