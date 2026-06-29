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

Design (kept orthogonal — the presence path is untouched, zero churn to its 14-arg
signature):
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

**Scope:** entry-type 0-nodes only (covers all ~15 KRN types + PARAMETER
DEFINITION uniformly). **Next slice:** FILE DD content-verify (field defs) — still
presence-only (`$D(^DD(file,0))`); and multi-node entry verify (e.g. param-def
1-node data type). Build-side unit/fake-driver proven; **not yet live-run** — that
lands with the MSL/VSL end-to-end install gate (the adversarial rec #1).

**Why/how to apply:** `v pkg verify` now means "the records I shipped are the
records that got filed," not just "names exist" — the gate worth depending on for
the in-org MSL/VSL ship loop. When adding a new entry type, set its `dataRoot` on
the `entryType` var and (only if FileMan re-files a 0-node piece) its `volatile`
mask; content-verify then covers it for free.
