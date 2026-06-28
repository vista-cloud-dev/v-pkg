---
name: install-hooks-authoring
description: B.3 — v pkg build authors env-check/pre/post install routines from a build spec; live-proven both engines; KILL-before-MERGE installer fix
metadata:
  type: project
---

**2026-06-28: B.3 — `v pkg build` AUTHORS install-time hook routines** (Track B
authoring, the install-side counterpart to [[install-fidelity-spike]]'s installer
work). A build spec can now declare three hook routines and the authored `.KID`
carries the transport nodes the live install honors:

- `envCheck` (bare routine name) → top-level `"PRE")` — `ENV^XPDIL1` does
  `D @("^"_name)`; 0 = proceed, non-zero (via `XPDABORT`) = reject.
- `preInstall` (entryref `TAG^RTN`) → top-level `"INI")` — `PRE^XPDIJ1` `D @`s it
  before component filing.
- `postInstall` (entryref) → top-level `"INIT")` → `POST^XPDIJ1` after filing.

Each is mirrored under `"BLD",1,…` (the #9.6 self-description). Emit is in
`kids.emitInstallHooks` (krncomp.go); unset hooks emit NOTHING, so a hook-free
build stays byte-identical (corpus DRIFT=0 preserved). `buildspec.validateInstallHooks`
is **shape-only** (env-check = bare name, no tag; pre/post = `RTN` or `TAG^RTN`):
the routine may be shipped in this build OR pre-exist on the target — a missing one
surfaces at install time, so requiring it shipped was dropped (it broke real specs
that point at system routines).

**Live-proven end-to-end via `v pkg build` (NO hand-injection) on BOTH engines**
(vehu YDB + foia-t12 IRIS), fixture `testdata/zza3-hooks/`:
- Positive (ZZA3, env-check passes + pre + post): `^ZZA3OUT("ENV"/"PRE"/"POST")=1`,
  #9.7 status 3. Proves the env-check **routine** RAN and passed, pre/post fired.
- Negative (ZZA3R, rejecting env-check): install refused `env-check-rejected^1^`,
  nothing filed. Proves A.1.2's env-check-**routine** reject arm (distinct from the
  Required-Build arm proven by `testdata/zza2-reqb`).

**Gotchas (hard-won):**
1. **Env-check routine source MUST be ASCII.** `^%ZOSF("SAVE")` compiles the
   env-check routine (from `^XTMP(…,"RTN",…)`) before `D @("^"_name)` runs it; a
   non-ASCII byte (e.g. a `→` in a comment) breaks the compile and the install gets
   a confusing reject. M routines are ASCII anyway.
2. **A rejected install rolls back its transaction** — the negative case's
   `^ZZA3OUT` sentinel does NOT persist (only device output / the refusal marker
   survives). Don't try to read a global to confirm a *rejected* hook ran; read the
   refusal marker. A *successful* install commits, so the positive sentinels persist.
3. **Installer fix — KILL before MERGE** (`FinalInstallScript`, script.go): the
   direct-populate path did `M ^XTMP("XPDI",XPDA)=^XTMP("VPKGI")` with no prior
   `K`. A purged earlier install frees its #9.7 IEN; `$$INST^XPDIL1` re-assigns it;
   stale `^XTMP("XPDI",IEN,…)` (e.g. a prior build's REQB nodes) survived the MERGE
   and made the hook-only build's `$$REQB` return 2 (false reject). Now
   `K ^XTMP("XPDI",XPDA)` precedes the MERGE, exactly as the real KIDS load starts
   clean. Regression test: `TestFinalInstallScript_KillBeforeMerge`.
4. Engine cleanup of a side-effecting fixture (pre/post make it class 2/3) needs
   `uninstall --force`; `^%ZOSF("DEL")` reads **uppercase** `X` (M is case-sensitive).

See [[install-fidelity-spike]] (A.1.1 checkpoints, A.1.2 env-check), [[kids-coverage-analysis]].
