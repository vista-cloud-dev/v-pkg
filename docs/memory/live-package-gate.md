---
name: live-package-gate
description: The live "ship the real package" gate (scripts/live-package-gate.sh + make live-gate) — builds MSL+VSL and drives install→content-verify→backout→verify-clean against a live engine. YDB/vehu proven 10/10 2026-06-29. Surfaced the #9.4-deregister gap.
metadata:
  type: project
---

**Live MSL+VSL install gate (2026-06-29)** — the integration proof that the unit /
fake-driver tests and the happy-path ZZ* fixtures could not give (adversarial rec
#1): build the org's two REAL packages from their committed specs and drive the
full lifecycle against a live engine through the driver stack. `scripts/live-package-gate.sh`
+ `make live-gate [ENGINE=ydb|iris] [NEG=1]`. Complements [[verify-content]],
[[package-footprint]], [[multi-build-install]].

**YDB/vehu: 10/10 PASS** — build MSL+VSL → install MSL (registered) → verify
(content) → install VSL (Required-Build MSL satisfied) → verify (content) →
uninstall VSL (`--force`) → verify-clean → uninstall MSL (`--force`) → verify-clean.
First-ever LIVE run of content-verify: VSL's `8989.51:VPNG GREETING` graded `ok`
(the live param-def 0-node matched the shipped image).

**KEY GOTCHA — `v pkg --engine ydb --transport docker` env recipe for vehu.** The
driver's docker login shell sources the container's own gbldir/routines, BUT
v-pkg's *load* (staging the `ZVPKG*` scratch routine) needs the routine dir
explicitly or it stages nothing → "driver loaded no routine". Required exports:
`M_YDB_BIN`, `M_YDB_CONTAINER=vehu`, `M_YDB_TRANSPORT=docker`,
`M_YDB_DIST=/home/vehu/lib/gtm`, `M_YDB_GBLDIR=/home/vehu/g/vehu.gld`,
**`M_YDB_ROUTINES=/home/vehu/r`** (the one that was missing first). `m-ydb exec
eval` works without it (no staging); v-pkg verbs do not. The gate hardcodes these
vehu defaults; IRIS needs M_IRIS_* (base-url/user/password/namespace/container).

**CONFIRMED GAP — uninstall does not deregister the PACKAGE #9.4 footprint.**
`--register-package` (A.3) stamps VERSION (#9.49) + PATCH APPLICATION HISTORY
(#9.4901); `uninstall` removes #9.7/#9.6/components but NOT #9.4. Live-confirmed on
vehu: after uninstalling MSL, `^XPD(9.7,"B","MSL*0.1*1")=0` (gone) but
`$$PATCH^XPDUTL("MSL*0.1*1")=1` and `^DIC(9.4,"B","M STANDARD LIBRARY")` persist
(ghost). Consequences: (1) the **negative** Required-Build direction can't be
exercised — VSL installs to status 3 with MSL's #9.7 absent because the ghost #9.4
satisfies the `$$PATCH/$$VER` check; (2) the register is REQUIRED for the positive
test (the check reads #9.4, not #9.7), so the asymmetry is load-bearing. **Fix
(follow-up):** a symmetric `uninstall --deregister` (or auto when the install
registered) that clears the #9.49/#9.4901 footprint. Until then the gate's `NEG=1`
is an informational known-gap probe, not a gated assertion.

**Side-effecting back-out reality (VSL).** VSL ships a FileMan file (#999001) +
param-def → class `side-effecting`. `uninstall --force` does a full delete
(routines + entries + file DD/data + #9.7). With a pre-image sidecar present
(`--auto-snapshot`), uninstall auto-detects it and does a routine **restore**
instead (reverts routines, LEAVES #9.7/param/file) — so a *clean* gate cycle must
install with `--allow-overwrite` (no sidecar), not `--auto-snapshot`.

**Status:** YDB proven; **IRIS (foia-t12) not run this session** (needs M_IRIS_*
creds) — same driver-abstracted path, per-type IRIS parity already proven
elsewhere. **Why/how to apply:** run `make live-gate` after any change to the
install/verify/uninstall path or the entry-component emitters — it's the
regression net that exercises the real packages end-to-end, which CI cannot.
