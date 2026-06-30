---
name: verify-content
description: v-pkg `verify` asserts entry-record CONTENT (live 0-node piece-compare), not just "B"-index presence — with a per-type volatile-piece mask for FileMan-transformed pieces.
metadata:
  type: project
---

**Content-asserting `v pkg verify` (2026-06-29)** — closes the adversarial gap that
verify was **presence-only** for non-routine components: it probed `$D(<root>("B",name))`
("a record by this name exists"), never that the record was *correct*. Now verify
also reads each shipped KRN entry record's **live 0-node** back and compares it to
the image the build shipped. Pairs with [[verify-drift]] (routines) and
[[option-entry-component]] (the emitter whose output it checks).

Design (kept orthogonal — the presence path is a separate staged script;
presence + uninstall are now registry-driven over `kids.Component`, see
[[component-type-coverage]]):
- `internal/installspec.VerifyContentScript(contents)` — a SECOND staged script:
  per record resolve the site IEN via the data file's `"B"` index, then
  `W` the stored 0-node as a `z:<file>:<name>` marker (empty = absent). Read the
  **literal stored global** (indirection `S VR="<root>"_VIEN_",0)"`), not the DBS
  API — it's the same image the KRN transport shipped, so a byte diff is a real
  filing fault.
- `kids.Build.EntryContents()` — one `EntryContent{File,FileStr,Name,DataRoot,Zero,Volatile}`
  per shipped `"KRN",<file>,<seq>,0)` node, via the new `entryTypeByFile` registry.
- `kids.ZeroMatch(expected, live, volatile)` — `^`-piece compare; trailing empty
  pieces equal absent ones (FileMan trims them, so shipped `NAME^^^` matches stored
  `NAME`); empty live = absent (never matches).
- `pkgcli.verifyContent` grades each record `ok`/`mismatch`/`absent`;
  `verifyResult.Content` + `ok()` gate (a mismatch fails verify, exit 3).

**KEY GOTCHA — the volatile-piece mask.** Re-filing types rewrite some 0-node
pieces at install, so a raw compare false-fails. The transform is per-type knowledge
on `entryType.volatile`: **#771 HL7 APP piece 7** (COUNTRY external `USA` → #779.004
IEN) and **#870 LOGICAL LINK piece 3** (LLP TYPE external `TCP` → #869.1 IEN 4).
Those two are the only known 0-node transforms; everything else files its 0-node
verbatim (RPC #8994 is a verbatim MERGE; OPTION/KEY/PROTOCOL/MAILGROUP/etc. re-file
but their 0-node identity pieces are stable — pointer re-pointing lives in higher
nodes like menu items #101.01/#19.01, NOT the 0-node, so 0-node compare sidesteps it).

**Scope:** entry-type 0-nodes (all ~15 KRN types + PARAMETER DEFINITION) **plus
FILE DD field defs** (2026-06-29): `Build.FileContents()` yields each shipped
`("^DD",file,file,fld,0)` def node; `VerifyContentScript` reads the live
`^DD(file,fld,0)` back and `ZeroMatch` compares it. FileMan files the DD **verbatim**
(DDIN^DIFROMS moves the image in) — live-proven the live def node is byte-identical
to `fieldDef`, so no volatile mask is needed (unlike the #771/#870 0-nodes). Marker
key `dd:<file>#<field>` (e.g. `999001#0.01`); the field number is a canonical M
numeric literal so it addresses the live node directly (no indirection). **Both
content checks live-proven on vehu** via the [[live-package-gate]]: VSL's param-def
+ all 5 `#999001` field defs (`.01`/`1`/`2`/`3`/`4`) graded `ok`. **Remaining:** the
file's `^DIC(file,0)` name/GL and multi-node entry verify (e.g. param-def 1-node
data type) are still presence-only — a minor follow-up.

**Gotcha found live:** re-installing a file-bearing build with `--allow-overwrite`
over a half-cleaned state (file data global gone but #9.7 stale) can reach status 3
WITHOUT re-filing `^DD` — the file content-verify catches it (fields graded
`absent`). A clean greenfield install files the DD correctly. So FILE content-verify
also guards against a silently-incomplete DD install.

**Why/how to apply:** `v pkg verify` now means "the records I shipped are the
records that got filed," not just "names exist" — the gate worth depending on for
the in-org MSL/VSL ship loop. When adding a new entry type, set its `dataRoot` on
the `entryType` var and (only if FileMan re-files a 0-node piece) its `volatile`
mask; content-verify then covers it for free.
