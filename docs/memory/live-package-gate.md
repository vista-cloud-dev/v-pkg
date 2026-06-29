---
name: live-package-gate
description: The live "ship the real package" gate (scripts/live-package-gate.sh + make live-gate) — builds MSL+VSL and drives install→content-verify→backout→verify-clean against a live engine. YDB/vehu AND IRIS/foia-t12 both proven 11/11 (NEG=1) 2026-06-29. Surfaced the #9.4-deregister gap.
metadata:
  type: project
---

**Live MSL+VSL install gate (2026-06-29)** — the integration proof that the unit /
fake-driver tests and the happy-path ZZ* fixtures could not give (adversarial rec
#1): build the org's two REAL packages from their committed specs and drive the
full lifecycle against a live engine through the driver stack. `scripts/live-package-gate.sh`
+ `make live-gate [ENGINE=ydb|iris] [NEG=1]`. Complements [[verify-content]],
[[package-footprint]], [[multi-build-install]].

**YDB/vehu: 11/11 PASS (NEG=1)** — build MSL+VSL → install MSL (registered) →
verify (content) → install VSL (Required-Build MSL satisfied) → verify (content) →
uninstall VSL (`--force --deregister`) → verify-clean → uninstall MSL
(`--force --deregister`) → verify-clean → **negative: install VSL alone → REFUSED**
(MSL deregistered). First-ever LIVE run of content-verify: VSL's
`8989.51:VPNG GREETING` graded `ok` (the live param-def 0-node matched the shipped
image).

**KEY GOTCHA — `v pkg --engine ydb --transport docker` env recipe for vehu.** The
driver's docker login shell sources the container's own gbldir/routines, BUT
v-pkg's *load* (staging the `ZVPKG*` scratch routine) needs the routine dir
explicitly or it stages nothing → "driver loaded no routine". Required exports:
`M_YDB_BIN`, `M_YDB_CONTAINER=vehu`, `M_YDB_TRANSPORT=docker`,
`M_YDB_DIST=/home/vehu/lib/gtm`, `M_YDB_GBLDIR=/home/vehu/g/vehu.gld`,
**`M_YDB_ROUTINES=/home/vehu/r`** (the one that was missing first). `m-ydb exec
eval` works without it (no staging); v-pkg verbs do not. The gate hardcodes these
vehu defaults; IRIS needs M_IRIS_* (base-url/user/password/namespace/container).

**GAP FOUND → CLOSED — uninstall now deregisters the PACKAGE #9.4 footprint
(2026-06-29).** Originally `uninstall` removed #9.7/#9.6/components but NOT the #9.4
footprint `--register-package` stamps, so after back-out `$$PATCH^XPDUTL("MSL*0.1*1")`
still returned 1 (a ghost) and VSL installed to status 3 with MSL's #9.7 absent —
the **negative** Required-Build direction couldn't fire. **Fixed by `uninstall
--deregister`** (see [[package-footprint]]): live-proven on vehu — `$$PATCH` 1→0,
PAH node gone, VERSION preserved; and with the ghost cleared, installing VSL alone
is correctly **refused** (env-check/required-build, exit 1). So the gate's `NEG=1`
is now a **real gated assertion** (11/11), and the back-out steps use `--force
--deregister` to stay $$PATCH-honest. (The register is still REQUIRED for the
positive test — the Required-Build check reads #9.4, not #9.7.)

**Side-effecting back-out reality (VSL).** VSL ships a FileMan file (#999001) +
param-def → class `side-effecting`. `uninstall --force` does a full delete
(routines + entries + file DD/data + #9.7). With a pre-image sidecar present
(`--auto-snapshot`), uninstall auto-detects it and does a routine **restore**
instead (reverts routines, LEAVES #9.7/param/file) — so a *clean* gate cycle must
install with `--allow-overwrite` (no sidecar), not `--auto-snapshot`.

**IRIS/foia-t12: 11/11 PASS (NEG=1, 2026-06-29)** — same gate, same assertions, now
run on IRIS via the **remote/Atelier transport** once the M_IRIS_* creds landed in
the environment (org `auth.env` + `source_up` in `v-pkg/.envrc`, commit dfa72db).
Invocation: `ENGINE=iris TRANSPORT=remote NEG=1 scripts/live-package-gate.sh` —
the gate defaults `TRANSPORT=docker` (vehu) so **IRIS must pass `TRANSPORT=remote`
explicitly** (the Atelier path is the credentialed login; docker-transport IRIS is
a separate thing). No IRIS-specific defaults are hardcoded in the gate — it reads
`M_IRIS_*` from the env. This also retires the owed **streamed-install IRIS
chunked-path validation**: MSL's full 15-routine install drives the size-bounded
`^XTMP("VPKGI")` chunk→MERGE→`EN^XPDIJ` path on IRIS and content-verified `ok`.

**Status:** BOTH engines proven 11/11 — the M0a "three invariants green on both
engines" gate is met for the live-package lifecycle. **Why/how to apply:** run
`make live-gate` (YDB) and `ENGINE=iris TRANSPORT=remote make live-gate` (IRIS)
after any change to the install/verify/uninstall path or the entry-component
emitters — it's the regression net that exercises the real packages end-to-end,
which CI cannot.
