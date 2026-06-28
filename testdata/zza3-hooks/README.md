# zza3-hooks — B.3 install-hook end-to-end fixture

Throwaway packages that prove **B.3**: `v pkg build` can AUTHOR install-time hook
routines (environment-check, pre-install, post-install) from a declarative build
spec, and a live KIDS install honors them — built **end-to-end via `v pkg build`**,
with **no hand-injection** of transport nodes (contrast `zza1-prepost`, which
hand-injects `"INI"/"INIT"` to prove only the installer's checkpoint path).

## Builds

- **`kids/ZZA3.build.json`** (positive) — ships `ZZA3P` + `ZZA3ENV`; declares
  `envCheck:"ZZA3ENV"`, `preInstall:"PRE^ZZA3P"`, `postInstall:"POST^ZZA3P"`.
  Authored `.KID` carries top-level `"PRE")=ZZA3ENV`, `"INI")=PRE^ZZA3P`,
  `"INIT")=POST^ZZA3P` (mirrored under `"BLD",1,…`).
- **`kids/ZZA3R.build.json`** (negative) — ships only `ZZA3RE`; declares
  `envCheck:"ZZA3RE"`. Exercises A.1.2's env-check **routine** reject arm (distinct
  from `zza2-reqb`, which proves the Required-Build reject arm).

## Routines (`src/`)

- **`ZZA3P.m`** — `PRE` sets `^ZZA3OUT("PRE")=1`; `POST` sets `^ZZA3OUT("POST")=1`.
- **`ZZA3ENV.m`** — passing env-check: sets `^ZZA3OUT("ENV")=1`, does NOT set
  `XPDABORT`, so `$$ENV^XPDIL1` returns 0 and the install proceeds.
- **`ZZA3RE.m`** — rejecting env-check: sets `^ZZA3OUT("ENV")=1` AND `XPDABORT=1`,
  so `$$ENV^XPDIL1` returns 1 (reject + kill global) and KIDS refuses the install.

All routine source is **ASCII-only** — `^%ZOSF("SAVE")` compiles the env-check
routine before running it, and a non-ASCII byte (e.g. a `→`) breaks that compile.

## Live-proven (2026-06-28) — both engines, via the driver stack

Built with `v pkg build … --src src`, installed with `v install … --engine
{ydb|iris} --transport docker`:

| Engine        | Positive (ZZA3)                     | Negative (ZZA3R)                          |
|---------------|-------------------------------------|-------------------------------------------|
| YDB (vehu)    | ENV=1, PRE=1, POST=1; #9.7 status 3 | `env-check-rejected^1^`; nothing filed    |
| IRIS (foia-t12)| ENV=1, PRE=1, POST=1; #9.7 status 3 | `env-check-rejected^1^`; nothing filed    |

A rejected install rolls back its transaction, so the negative case's `^ZZA3OUT`
sentinel does not persist — the proof is the refusal marker + `#9.7` staying
absent. The KILL-before-MERGE fix in `FinalInstallScript` (stale-IEN guard) was
discovered here.
