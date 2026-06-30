---
name: component-type-coverage
description: v-pkg verify/uninstall presence + ^DIK are registry-driven over kids.Component (one row per type, not 11 hand-coded cases); a contentVerify gate keeps unvalidated 0-node masks out of content-verify.
metadata:
  type: project
---

**Registry-driven verify/uninstall + increment-1 coverage (2026-06-30)** — closes
the gap that `v pkg install` files every component type natively (KRN^XPDIK) but
`verify`/`uninstall` only knew **11** hand-coded types, so a build shipping an
INPUT TEMPLATE (#.402, the 3rd-commonest component in vehu), FORM, BULLETIN, …
installed fine yet **uninstall orphaned it and verify couldn't assert it**. Canon
for the 4-part effort: `../verifiable-safety-tracker.md`. Pairs with
[[verify-content]], [[class-aware-uninstall]], [[fileman-dd-component]].

**The mechanism — one registry, three consumers.** `entryTypeByFile`
(internal/kids/entrycomp.go) is the single source. `(b *Build) Components()`
returns one `kids.Component{File,FileStr,DataRoot,Label,Names}` per type the build
ships (file-number ascending, names via the existing `entryNames`). `VerifyScript`
+ `UninstallScript` (internal/installspec) now take `[]kids.Component` instead of
14 positional name-slices, and loop generically:
- presence: `$D(<DataRoot>"B",<name>)` → `comp:<file>:<name>` marker.
- uninstall: `S DA=$O(<DataRoot>"B",<name>,0)),DIK="<DataRoot>" I DA D ^DIK`.
`verifyResult.Components map[string]bool` replaced the 11 typed maps. **DataRoot is
the whole contract** — it yields BOTH the "B" presence index and the ^DIK ref, so a
new type is **one registry row, no new code in three places.**

**The contentVerify gate (the verifiable-safety bit).** `entryType.contentVerify`
gates the 0-node CONTENT assertion (`EntryContents` skips `!contentVerify`). The 11
original types are `true` (their volatile masks are live-proven, see
[[verify-content]]). The 8 increment-1 types are **`false`**: they get
presence-verify + ^DIK back-out (the orphan fix) but **no content claim**, because
their 0-node mask is not ground-truthed — asserting it would risk false drift
(templates' DATE pieces are install-volatile; their real content lives in DR/DIAB
subnodes, not the 0-node). Flip a type to `true` only after a live shipped-vs-filed
0-node diff. **Do not add a new type to `entryTypeByFile` with contentVerify:true
and an empty/guessed mask** — that re-opens the false-drift hole this gate closes.

**Types added (presence + ^DIK only):** #.402 INPUT TEMPLATE `^DIE(`, #.4 PRINT
`^DIPT(`, #.401 SORT `^DIBT(`, #.403 FORM `^DIST(.403,`, #.84 DIALOG `^DI(.84,`,
#3.6 BULLETIN `^XMB(3.6,`, #8989.52 PARAMETER TEMPLATE `^XTV(8989.52,`, #1.5 ENTITY
`^DDE(`. Roots are live `^DIC(f,0,"GL")` (vehu 2026-06-30).
**Excluded on purpose:** #.5 FUNCTION (`^DD("FUNC",` — not a standard ^DIK target);
**#8993 XULM** (its .01 lives at node `E1,245)`, not `0)`, so the 0-node name probe
would miss — and ships in 1 corpus KID); **#869.2** + lexicon **#9002226** (never
shipped as a top-level `KRN",<file>,<seq>` image in the 2,404-KID corpus, so
untestable / dead code). The "all standard FileMan types" goal = the types that
actually ship as KRN images, by frequency.

**DIK safety — live-proven (kickoff's non-negotiable).** Method that needs NO real
package install and leaves NO residue on the gold master: for each type, seed a
throwaway record + its "B" xref (+ a fake compiled subfile node for the templates)
in its storage global via `m vista exec --engine ydb --docker vehu`, run the EXACT
generated `…DIK="<root>" D ^DIK`, then assert `$D(<root>"B",name)=0` AND
`$D(<root>IEN)=0`. All 8 came back B0/R0 — DIK removes the record **subtree**
(compiled template subfiles included, since DIK kills `^GLOBAL(DA)` wholesale) and
the xref. Net-zero (seed+delete in one probe). Gates: `make corpus` DRIFT=0 (2404),
`make stress` 37/37, `make live-gate` 10/10 — the live gates already exercise the
generic DIK path for the existing types (MSL/VSL param-defs).

**Follow-up (not blocking the orphan fix):** enable content-verify for the types
whose 0-node IS the content once masked live — BULLETIN #3.6 and ENTITY #1.5 have
no 0-node pointers (mask empty); PARAMETER TEMPLATE #8989.52 pieces 3 (USE ENTITY
FROM, variable-ptr) + 4 (USE INSTANCE FROM → #8989.51) are pointer-volatile (mask
{3,4}). Template family + DIALOG stay presence-only (compiled, fragile 0-node).
