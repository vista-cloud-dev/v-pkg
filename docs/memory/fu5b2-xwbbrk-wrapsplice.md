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

## NEXT (5B.2b — owed; the live steps gated on owner go-ahead)
1. `readRoutineSource(ctx, cl, name) ([]string,error)` — read stock XWBBRK off the
   engine over the driver (`$TEXT` probe via `runMScript`/`ExecRun`; the first
   engine-READ capability in v-pkg).
2. A command (e.g. `v pkg wrap-rpc install|backout|status`) that: read stock →
   `Splice`/`Unsplice` → ship the patched/stock routine via the existing
   `kids.MakeBuildPairs`+`installspec` path (reuse, routines-only build).
3. The **non-interference proof** against the real dispatch (IRIS/foia first): wrap
   ON vs OFF → byte-identical result + FU-4 property + bounded resource deltas.
4. **Live install of patched national XWBBRK is hard-to-reverse → confirm with the
   owner before the first foia overwrite**, with the back-out proven first.
