# ZZA1 — pre/post-install live-prove fixture (A.1.1)

A throwaway KIDS build that ships a routine `ZZA1P` with `PRE` and `POST`
entry points, each setting a sentinel global node (`^ZZA1OUT("PRE"/"POST")=1`),
and declares them as the build's pre/post-install routines via the top-level
transport nodes `"INI")=PRE^ZZA1P` and `"INIT")=POST^ZZA1P`.

It exists to live-prove **install-fidelity A.1.1** (`docs/proposals/v-pkg-install-fidelity-spike.md`):
that `v pkg install` now creates the `#9.7` `INI`/`INIT` checkpoints (via the real
`$$NEWCP^XPDUTL`) so `EN^XPDIJ`'s `PRE^/POST^XPDIJ1` loops actually run the build's
pre/post-install routines.

`ZZA1.kids` is generated end-to-end from the build spec: `ZZA1.build.json`
declares `"preInstall": "PRE^ZZA1P"` and `"postInstall": "POST^ZZA1P"`, and
`v pkg build` emits the top-level `"INI")`/`"INIT")` transport nodes plus their
`"BLD",1,"INI")`/`"BLD",1,"INIT")` #9.6 manifest mirrors (coverage-analysis Track
**B.3**, now landed). Regenerate with:

```sh
v pkg build testdata/zza1-prepost/kids/ZZA1.build.json \
  --src testdata/zza1-prepost/src --out testdata/zza1-prepost/ZZA1.kids
```

## Re-running the live-prove (driver stack only)

```sh
# YDB (vehu)
export M_YDB_CONTAINER=vehu M_YDB_BIN=~/vista-cloud-dev/m-ydb/dist/m-ydb M_YDB_ROUTINES=/home/vehu/r
v pkg install testdata/zza1-prepost/ZZA1.kids --engine ydb --transport docker
m vista exec --engine ydb --transport docker 'W $G(^ZZA1OUT("PRE")),$G(^ZZA1OUT("POST"))'  # expect 11

# IRIS (foia-t12)
export M_IRIS_CONTAINER=foia-t12 M_IRIS_NAMESPACE=VISTA M_IRIS_BIN=~/vista-cloud-dev/m-iris/dist/m-iris
v pkg install testdata/zza1-prepost/ZZA1.kids --engine iris --transport docker
m vista exec --engine iris --transport docker 'W $G(^ZZA1OUT("PRE")),$G(^ZZA1OUT("POST"))'  # expect 11
```

Cleanup (per engine): `K ^ZZA1OUT`, delete `ZZA1P` (`S X="ZZA1P" X ^%ZOSF("DEL")`),
and `^DIK` the `#9.7`/`#9.6` `ZZA1*1.0*1` entries.
