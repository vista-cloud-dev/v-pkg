---
name: verify-script-m-injection
description: VerifyScript/VerifyContentScript spliced build-controlled routine/component NAMES raw into W marker literals + $T(^<rtn>) — a crafted .KID gave arbitrary M execution via `v pkg verify`. Fixed 2026-06-30 by emitting every name via kids.MString + entryref-indirection routine probe.
metadata:
  type: project
---

**M-injection in the verify path (found by adversarial review, fixed 2026-06-30,
`824610a`).** `internal/installspec/script.go` `VerifyScript` and
`VerifyContentScript` interpolated the **build-controlled** routine names and entry
`.01` component NAMES **raw** into the generated `W "<<VPKG>>…<name>=",…` marker
literals (and `VerifyScript` spliced the routine name raw into `$T(^<name>)`). The
names come straight from the `.KID` (`"RTN",<name>` subscripts / KRN `.01` names),
and `parseSubscriptLine` decodes `""`→`"`, so a crafted name carrying a `"` plus M
closed the literal and executed **arbitrary M in programmer context** (`DUZ(0)="@"`)
the moment an operator ran `v pkg verify` / `diff` / `--dry-run` content pass on an
untrusted `.KID`. Real RCE-class supply-chain vector (you verify a `.KID` before
trusting it — exactly when it bites).

**Why it slipped:** the escaping discipline existed but was applied inconsistently —
`UninstallScript`/`DeregisterScript` route every name through `kids.MString`, and
`readRoutineBody` is guarded by `validRoutineName` at its sole caller — but the
`Verify*Script` generators were the unguarded outliers. The `"B"`-subscript use of
the name WAS escaped; only the **marker-label literal** (and the `$T` ref) were raw.

**Fix (the durable rule): a build-controlled string must reach a generated M script
ONLY inside an escaped M string literal (`kids.MString`), never as raw format text
or a raw routine ref.**
- Names are now written as VALUES: `W "<<VPKG>>rtn:",<MString(name)>,"=",…` — the
  printed device label is identical (so the `<<VPKG>>` marker protocol and the Go
  `markers["rtn:"+name]` keys are unchanged), but the name can't break out.
- The routine-existence probe uses **entryref indirection**: `S VRN="+0^"_<MString(
  name)>` then `$T(@VRN)` (was `$T(^<name>)`) — same "first line present" semantics,
  name is a string operand not code. Works on YDB + IRIS.
- File/field NUMBERS (`FileStr`/`Field`) are numeric-by-construction
  (`strconv.FormatInt`), not a string vector — left as direct subscripts.

**Guard test (regression net):** `TestVerifyScripts_NoMInjection` asserts the
**quote-balance invariant** — every generated line has an even number of `"` (a raw
splice of a name with one `"` makes an odd line) — plus that the name appears only in
`MString` form. This is a *structural* proof (a name confined to a balanced literal
cannot execute), not a sample. Live-proven no regression: `make live-gate` 15/15 on
vehu (verify content + drift still pass; device output byte-identical).

**Apply generally:** any future generated-M path (new verb, new probe) must pass
build-controlled strings through `kids.MString` (or `validRoutineName` for true M
routine names) — audit every `W "…" + <buildString> + "…"` and every `^<buildString>`.
