# ZZA1 — pre/post-install live-prove fixture (A.1.1)

A throwaway KIDS build that ships a routine `ZZA1P` with `PRE` and `POST`
entry points, each setting a sentinel global node (`^ZZA1OUT("PRE"/"POST")=1`),
and declares them as the build's pre/post-install routines via the top-level
transport nodes `"INI")=PRE^ZZA1P` and `"INIT")=POST^ZZA1P`.

It exists to live-prove **install-fidelity A.1.1** (`docs/proposals/v-pkg-install-fidelity-spike.md`):
that `v pkg install` now creates the `#9.7` `INI`/`INIT` checkpoints (via the real
`$$NEWCP^XPDUTL`) so `EN^XPDIJ`'s `PRE^/POST^XPDIJ1` loops actually run the build's
pre/post-install routines.

`ZZA1.kids` was built with `v pkg build` and then had the two top-level
`"INI")`/`"INIT")` nodes hand-injected, because authoring pre/post-install
routines is not yet supported (coverage-analysis Track **B.3**). Once B.3 lands,
this fixture should be regenerated end-to-end from the build spec.

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
