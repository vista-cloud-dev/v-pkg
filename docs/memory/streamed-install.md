---
name: streamed-install
description: v pkg install now STREAMS the transport global in size-bounded chunks into a staging global, then MERGE + EN^XPDIJ in one process — fixing a silent partial-install of large packages (the one-mega-routine staging truncated at ~3 routines). YDB live-proven on the full 15-routine MSL base.
metadata:
  type: project
---

# Streamed KIDS install (fixes silent partial-install of large packages)

## The bug (T0b.2 discoveries P1, m-stdlib)
`v pkg install` built ONE scratch routine `ZVPKGINS` embedding every
`S ^XTMP("XPDI",XPDA,…)=…` pair and ran it. Fine for ZZSKEL (1 routine), but a
real package's transport global is large — the m-stdlib MSL base is **15 routines
/ ~6100 nodes / ~560 KB**, giving a ~560 KB install routine. The **driver's
routine staging silently truncates** a routine that big (m-ydb docker `exec load`
fails/partial-stages ≳60 KB; the eval path truncates a single command at ~21 KB),
so only the first ~3 routines (STDSTR/STDMATH/STDB64) landed — while `EN^XPDIJ`
still reported `#9.7` status 3. **v-pkg generated the correct full 6156-line
script** (`ParseKID().Pairs()` = all 6148 nodes); the loss was entirely in
staging the giant routine.

## The fix (this branch, `refile-v-pkg`)
Two phases, no SDK change:
1. **`installspec.StageChunks(pairs, maxBytes)`** — renders the pairs as several
   small routine bodies (≤40 KB each, `stageChunkBytes` in `lifecycle.go`) that
   `S ^XTMP("VPKGI",<subs>)=<val>`. The first body `K ^XTMP("VPKGI")`. `runInstall`
   runs each via `runMScript` (load+run); the staging global persists across the
   stateless driver processes, accumulating the whole tree.
2. **`installspec.FinalInstallScript(name, header, nPairs)`** — constant-size
   routine: counts the staged nodes ($QUERY loop) and **refuses with
   `error=stage-incomplete` unless the count == nPairs** (so a silent truncation
   now fails loudly, never installs a partial package); then `$$INST^XPDIL1` →
   `M ^XTMP("XPDI",XPDA)=^XTMP("VPKGI")` → `EN^XPDIJ` → `K ^XTMP("VPKGI")` → status
   marker. INST + MERGE + EN^XPDIJ stay in ONE process so XPDA survives (the MERGE
   makes the install routine size-independent of the package).

`runInstall` returns a Go error (not `already-installed`) on `stage-incomplete`.

## Proof
- Unit (TDD): `TestStageChunks_CoversAllPairsBounded`, `TestFinalInstallScript`,
  `TestRunInstall_MultiChunkStages` (asserts loads/runs == chunks+1). `go test
  ./... -race`, `go vet`, `gofmt`, contract no-drift — all green.
- **Live on YDB FOIA `vehu`** (m-ydb docker, after the gbldir fix `e5dcf85`): the
  full 15-routine MSL base installs (all `$T(^STD*)=1`, status 3, ~3 s);
  `scripts/kids-test-in-place.sh ydb` (m-stdlib) → **15/15 suites pass in place,
  1403 assertions, 0 fail**; uninstall reversible, verify-clean.

## Owed
- **IRIS live-validation of the chunked path** (m-iris/Atelier stages each chunk
  via PutDoc; ^XTMP persists across runner invocations — expected to work, not yet
  run). Will be exercised by the **T0b.2 IRIS leg** (m-stdlib session).
- A native `mdriver.Client.SetGlobal` would let the host populate `^XTMP` directly
  (no staging routines) — cleaner, but an SDK/coordinator change; not needed now.

See `docs/design/kids-installation-automation.md §7` and the m-stdlib T0b.2 tracker.
