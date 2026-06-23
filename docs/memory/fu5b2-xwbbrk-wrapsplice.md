---
name: fu5b2-xwbbrk-wrapsplice
description: v-pkg internal/wrapsplice — host-side XWBBRK patcher (insert/back-out the VSLRPCWRAP traffic-tap side-calls) with content-anchored re-pin (FU-21); the RPC→S3 tap FU-5 5B.2 patcher core
metadata:
  type: project
---

# internal/wrapsplice — host-side XWBBRK wrap patcher (FU-5 5B.2a)

**DONE 2026-06-23.** Branch `s3tap-fu5b2-patcher`. The pure-Go, off-engine patch
core for the RPC→S3 traffic tap's broker wrap. Part of the cross-repo s3tap
workstream (the wrap glue `VSLRPCWRAP` lives in v-stdlib; this is the v-pkg
tooling that delivers it). Shared coordination: docs `docs/memory/rpc-traffic-s3-streaming-proposal.md`;
splice design: docs `docs/discoveries/fu-5b-callp-splice.md`.

## What it is
`internal/wrapsplice` patches the national RPC Broker routine **`CALLP^XWBBRK`** to
install — and exactly back out — the two `VSLRPCWRAP` traffic-tap side-calls. It is
**pure text surgery over `[]string` routine source, NO engine contact** — the
delivery decision (Option A): read stock XWBBRK off the engine via the driver, run
`Splice` host-side, ship the patched routine through the normal KIDS path.

API: `Splice(src) ([]string,error)` · `Unsplice(src) ([]string,error)` ·
`Validate(src) (reqIdx,respIdx,error)` · `IsSpliced(src) bool` · `Marker` (`;VSLTAPW`).

## Design decisions worth keeping
- **Content-anchored, not line-numbered** — the req anchor is matched by EXACT
  trimmed content (`S:$L($G(XWBSEC)) ERR="-1^"_XWBSEC` — XWBSEC also appears on
  lines 149/152, so an exact match keeps it unique); the resp anchor by the
  dispatch substring `D CAPI^XWBBRK2(`. **Both must occur exactly once or
  `Validate` REFUSES** — this is the **FU-21 per-XWB-patch re-validation gate** in
  code (a drifted/duplicated anchor is refused, never mis-patched).
- **Idempotent + exactly reversible** — every inserted line carries `Marker`;
  `Splice` refuses an already-spliced source, `Unsplice` removes only `Marker`
  lines, so `Splice→Unsplice` restores stock **byte-for-byte** (the back-out is the
  inverse, distinct from `VSLTAPBO` which clears the tap's `^XTMP` footprint).
- **Statement-level preserved** — each inserted line copies its anchor's leading
  whitespace+dot prefix (`linePrefix`): the `req` call lands top-level after the
  denial line (fires for every RPC, incl. a CHKPRMIT deny, known only there); the
  `resp` call lands at the dispatch's dot level so it stays INSIDE the `:155 IF…D`
  success block (success path only, before `:160 K XWB`).
- **Splice line is a plain `D req^VSLRPCWRAP`** (no guard) — VSLRPCWRAP owns the
  FU-4 fence + `$$captureOn` gate; a global-flag guard in the broker line would move
  the naked indicator before the fence runs (see the v-stdlib 5B.1 memory).

## Verification
`wrapsplice_test.go` table-driven against the **verbatim `CALLP^XWBBRK` 140-162**
fixture (captured live foia/IRIS, byte-identical vehu/YDB): both anchors found
uniquely; both side-calls placed correctly; **surgical** (removing the two Marker
lines reproduces stock — the XWBSEC/CAPI mentions on other lines untouched);
`Splice→Unsplice` byte-exact; idempotency + drift (missing/duplicated anchor)
refused. `make all` green (golangci-lint + `-race` tests + build); wrapsplice
coverage **97.7%**. GOTCHA (pre-existing, repo memory): `make test` needs
`CGO_ENABLED=1` (Makefile hard-sets 0 but `-race` needs cgo) — `make test`/`make all`
handle it.

## 5B.2b parts 1–2 DONE 2026-06-23 (read-capability + command; live install still held)
- **`readRoutineSource(ctx, cl, name)`** (pkgcli/lifecycle.go) — the **first
  engine-READ capability in v-pkg**: streams a routine's source back as one
  `<<VPKG>>l<n>=<line>` marker per line via a generated `$TEXT` loop (`readRoutineBody`),
  reconstructed in order by `parseRoutineLines`; `validRoutineName` guards the name
  against M injection before interpolation. Read-only.
- **`v pkg wrap-rpc status|install|backout`** (pkgcli/wraprpc.go) — read stock →
  `wrapsplice.Splice`/`Unsplice` → `kids.MakeBuildPairs` (routines-only) →
  **PREVIEW by default** (writes the patched `.m`+`.kids` to `--out`, engine NOT
  modified); the live KIDS install (reuse of the proven `runInstall`) runs **only
  under `--commit`**. `status` is read-only (reports spliced? + anchors-OK).
- **Live read-only smoke (foia/IRIS):** `wrap-rpc status` read the real **211-line**
  XWBBRK → `spliced:false, anchorsOk:true` (the content anchors are present + unique
  on the ACTUAL national routine, not just the fixture — strong FU-21 confirmation);
  `wrap-rpc install --out …` (no `--commit`) produced a 213-line patched routine
  (217 KIDS pairs) with `D req^VSLRPCWRAP ;VSLTAPW` at line 154 (after the denial
  line) and `. D resp^VSLRPCWRAP ;VSLTAPW` at line 160 (after the dispatch, dot
  level). `committed:false` — engine untouched.
- Gates: `make all` green; the command-surface contract golden (`dist/v-contract.json`)
  regenerated (`make contract`); wrapsplice cov 97.7%, pure helpers unit-tested.

## STILL HELD (5B.2c — the live mutation; gated on owner go-ahead)
1. The **live `--commit` install** of patched national XWBBRK on foia (IRIS first) —
   hard-to-reverse overwrite of the busiest national routine; back-out (`wrap-rpc
   backout --commit`) proven first.
2. The **non-interference proof** against the real dispatch: wrap ON vs OFF →
   byte-identical result + FU-4 property + bounded resource deltas (spec §6.4).
