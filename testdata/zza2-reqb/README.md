# ZZA2 — required-build / env-check live-prove fixture (A.1.2)

Two throwaway KIDS builds of a trivial routine `ZZA2`, used to live-prove
**install-fidelity A.1.2** (`docs/proposals/v-pkg-install-fidelity-spike.md`): that
`v pkg install` now reconstructs the install-phase scope and calls the **real
`$$ENV^XPDIL1(1)`** before filing — running the build's environment-check routine
**and** enforcing Required Builds (#9.611, `REQB^XPDIL1`) — and **refuses to file**
(auto-purging the aborted `#9.7` entry) on a non-zero reject.

- `kids/ZZA2.build.json` → `ZZA2.kids` — declares a **bogus Required Build**
  `ZZNOPE*1.0*1` with action `DON'T INSTALL, LEAVE GLOBAL` (#9.611 code 2). Install
  must be **REFUSED** (`error=env-check-rejected^2^2`).
- `kids/ZZA2-ok.build.json` → `ZZA2-ok.kids` — the control, **no** Required Build.
  Install must **succeed** (status 3) — `$$ENV` returns 0 for a clean build.

The counterfactual is the proof: the same routine installs cleanly as the control
but is refused when it declares an unmet Required Build — and the refusal leaves
**no `#9.7` entry** (clean retry), identically on YDB and IRIS.

## Re-running the live-prove (driver stack only — never raw `docker exec`)

```sh
# build both .kids
v pkg build testdata/zza2-reqb/kids/ZZA2.build.json    --src testdata/zza2-reqb/src --out testdata/zza2-reqb/ZZA2.kids
v pkg build testdata/zza2-reqb/kids/ZZA2-ok.build.json --src testdata/zza2-reqb/src --out testdata/zza2-reqb/ZZA2-ok.kids

# YDB (vehu)
export M_YDB_CONTAINER=vehu M_YDB_BIN=~/vista-cloud-dev/m-ydb/dist/m-ydb M_YDB_ROUTINES=/home/vehu/r
v pkg install testdata/zza2-reqb/ZZA2.kids    --engine ydb --transport docker   # REFUSED: env-check-rejected^2^2
v pkg install testdata/zza2-reqb/ZZA2-ok.kids --engine ydb --transport docker   # status 3

# IRIS (foia-t12)
export M_IRIS_CONTAINER=foia-t12 M_IRIS_NAMESPACE=VISTA M_IRIS_BIN=~/vista-cloud-dev/m-iris/dist/m-iris
v pkg install testdata/zza2-reqb/ZZA2.kids    --engine iris --transport docker  # REFUSED: env-check-rejected^2^2
v pkg install testdata/zza2-reqb/ZZA2-ok.kids --engine iris --transport docker  # status 3

# bypass: --skip-env-check installs the bogus build anyway (status 3)
v pkg install testdata/zza2-reqb/ZZA2.kids --engine ydb --transport docker --skip-env-check
```

Cleanup (per engine): delete `ZZA2` (`S X="ZZA2" X ^%ZOSF("DEL")`) and kill any
`ZZA2*1.0*1` `#9.7` entries (`F  S DA=$O(^XPD(9.7,"B","ZZA2*1.0*1",0)) Q:'DA  K
^XPD(9.7,DA),^XPD(9.7,"B","ZZA2*1.0*1",DA)`).

## Note — env-check ROUTINE execution is not exercised here
A.1.2 also runs the build's env-check **routine** (named in
`^XTMP("XPDI",XPDA,"PRE")`), but that routine must already exist on the engine at
install time (it is not yet filed — authoring of the `"PRE"` transport node is a
follow-up, sibling of A.1.1's hand-injected `INI`/`INIT`). This fixture proves the
`$$ENV` path via **Required-Build enforcement**, which needs no routine execution
and is fully emitted by `v pkg build` today (`emitRequiredBuildManifest`).
