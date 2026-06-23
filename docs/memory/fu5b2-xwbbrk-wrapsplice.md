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

## 5B.2c live sequence — step 1 DONE 2026-06-23 (stack deployed; patch HELD)
**Prerequisite finding (load-bearing):** the patched `XWBBRK` calls `D req^VSLRPCWRAP`
on EVERY RPC, so the **VSL tap stack must be installed BEFORE the `XWBBRK` patch** — else
the call is undefined and the broker's `$ETRAP` fails every RPC. Recon of foia (IRIS):
**m-stdlib already present** (`STDCRYPTO/STDJSON/STDB64/STDDATE`=1, from earlier M6/M6.5
work) but **no VSL routines**.
- **Step 1 done:** `v pkg install VSL.kids` (VSL\*1.0\*2, 15 routines from branch
  `s3tap-fu5b1-rpcwrap`) on foia → status 3; `VSLRPCWRAP/VSLRPCTAP/VSLTAP/VSLTAPHL/VSLS3`
  all present; `wrap-rpc status` confirms `XWBBRK` still **stock** (spliced:false, 211
  lines, anchorsOk:true). Broker untouched (no XWBBRK patch yet).
- The tap defaults **OFF** (`^VSLTAP("cfg","mode")` unset → `$$captureOn`=0), so once
  `XWBBRK` is patched the in-path is just the FU-4 fence + the gate read until armed.

- **Step 2 done 2026-06-23 — `XWBBRK` PATCHED LIVE on foia (tap OFF; broker intact).**
  `wrap-rpc install --commit` shipped the patched routine via KIDS (install name
  `VSLTAP RPC WRAP 1.0`, #9.7 status 3) — **proving the KIDS-overwrite-an-existing-
  national-routine path works**. Verified: `status` → spliced:true, 213 lines; the
  inserted `D req^VSLRPCWRAP` is at line 154 (after the denial line) and
  `. D resp^VSLRPCWRAP` at line 160 (after the `:159` dispatch, dot level); and
  `captureOn=0 / state=OFF` with the inserted calls running clean (`wrapcalls=ok`) —
  the patched routine compiled and the wrap calls are safe no-ops with the tap off, so
  the broker behaves as stock. **CURRENT LIVE STATE: foia XWBBRK is patched, tap OFF.**
  A raw stock backup is in the session scratchpad (`XWBBRK.stock.m`, 211 lines);
  deterministic back-out is `wrap-rpc backout --commit`.

## 5B.2c COMPLETE 2026-06-23 — non-interference PROVEN live; backed out byte-clean
**FU-5 is fully validated end-to-end on a live VistA (foia/IRIS).** The full sequence
ran: deploy stack → patch `XWBBRK` → non-interference proof → back-out.
- **Non-interference proof (foia), both paths PASS.** `CAPI^XWBBRK2` can't be called
  standalone (it `U`ses the broker-only devices `XWBNULL`/`XWBTDEV`, which exist only in a
  live TCP connection) — and the splice doesn't touch `CAPI` — so the proof brackets a
  **real RPC body** with the real `req`/`resp^VSLRPCWRAP` (the exact patched statements),
  OFF vs ARMED, on live foia with real m-stdlib (STDCRYPTO sha256 in-path):
  - **scalar** (`IMHERE^XWBLIB`, the real `XWB IM HERE` handler, `S RESULT=1`): result
    `1/1` identical off/armed; naked-ref preserved off/armed; `$T` preserved; ring `0/2`
    (armed actually captured req+resp); `$EC` clean.
  - **global-array / FU-17** (`XWBPTYPE=4`, `XWBP`=a closed global ref): result ref
    unchanged; **naked-ref preserved DESPITE the in-path `MERGE`** (the FU-4 fence holds
    for the global op on real IRIS); `$T` preserved; the snapshot byte-faithfully captured
    the source subtree (`alpha`/`beta`/`deep`); `$EC` clean.
  - Resource delta = bounded `^XTMP` churn (2 ring nodes/RPC, capped+trimmed), no journal
    impact (`^XTMP`); consistent with §6.4 (a fuller journal/CPU measurement is a follow-up,
    but the bounded-by-design property holds).
- **Back-out proven (FU-21 reverse):** `wrap-rpc backout --commit` → status 3; `status` →
  spliced:false, 211 lines; the restored `XWBBRK` is **byte-identical** to the pre-patch
  stock backup (`diff` empty). Tap state cleared (`^VSLTAP`/`^XTMP("VSLTAP")` killed,
  state=OFF). **foia is back to stock.** (The VSL stack routines remain installed — inert
  now that `XWBBRK` is stock; uninstall optional.)

## Remaining FU-5 follow-up (not blocking)
- **FU-21 hook:** the per-XWB-patch re-pin gate exists *in code* (`wrapsplice.Validate`
  refuses a drifted/ambiguous anchor); a scheduled CI/regression re-pin (run `wrap-rpc
  status` per XWB patch) is the formalization.
- A fuller journal/CPU resource-delta measurement under load (the IOC-site production
  validation, plan §13.4 step 8).
