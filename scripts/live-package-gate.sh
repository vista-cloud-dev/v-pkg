#!/usr/bin/env bash
# live-package-gate.sh — the "ship the real package" end-to-end gate.
#
# Builds the org's two real KIDS packages — MSL (m-stdlib) and VSL (v-stdlib) —
# from their committed specs and drives the FULL lifecycle against a LIVE engine
# through the m-driver-sdk stack (waterline rule 3): install -> content-verify ->
# back out -> verify clean, with the VSL->MSL Required-Build dependency exercised
# both ways. This is the integration proof that complements the unit/fake-driver
# tests: it exercises the real packages, the real driver transport, and the real
# KIDS phases together — surfacing the messiness that synthetic ZZ* fixtures hide.
#
# It is NOT a cloud-CI gate (it needs the local Docker engines vehu/foia-t12);
# run it locally / on the engine host.
#
# The cycle is fully self-cleaning: `--register-package` stamps a PACKAGE #9.4
# footprint and the back-out uses `uninstall --force --deregister` to clear it, so
# $$PATCH^XPDUTL is honest afterward (no ghost). The register is REQUIRED for the
# POSITIVE dependency test (VSL's Required-Build reads $$PATCH/$$VER^XPDUTL = #9.4,
# so MSL must be registered to satisfy it); the deregister is what makes the
# NEGATIVE test real — with MSL's footprint cleared, installing VSL alone is
# correctly refused. Both directions are now gated assertions (NEG=1).
#
# Engine connection is read from the M_<ENGINE>_* environment (same knobs as
# `v pkg` / `m vista`), NEVER hardcoded here. For YDB+vehu the docker login shell
# sources the container's own env, so only the paths below are needed; for IRIS
# the caller must export M_IRIS_* (base-url, user, password, namespace, container).
#
# Usage:
#   scripts/live-package-gate.sh                 # ENGINE=ydb (vehu) by default
#   ENGINE=iris scripts/live-package-gate.sh     # needs M_IRIS_* exported
#   NEG=1 scripts/live-package-gate.sh           # also run the negative dependency test
#
set -euo pipefail

ENGINE="${ENGINE:-ydb}"
TRANSPORT="${TRANSPORT:-docker}"
NEG="${NEG:-0}"

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VPKG="${VPKG:-$here/dist/v-pkg}"
MSL_DIR="${MSL_DIR:-$HOME/vista-cloud-dev/m-stdlib}"
VSL_DIR="${VSL_DIR:-$HOME/vista-cloud-dev/v-stdlib}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# YDB+vehu connection defaults (the driver's docker login shell sources the rest).
# Override by exporting M_YDB_* before calling; IRIS has no safe default (creds).
if [[ "$ENGINE" == "ydb" ]]; then
  export M_YDB_BIN="${M_YDB_BIN:-$HOME/vista-cloud-dev/m-ydb/dist/m-ydb}"
  export M_YDB_CONTAINER="${M_YDB_CONTAINER:-vehu}"
  export M_YDB_TRANSPORT="${M_YDB_TRANSPORT:-docker}"
  export M_YDB_DIST="${M_YDB_DIST:-/home/vehu/lib/gtm}"
  export M_YDB_GBLDIR="${M_YDB_GBLDIR:-/home/vehu/g/vehu.gld}"
  export M_YDB_ROUTINES="${M_YDB_ROUTINES:-/home/vehu/r}"   # required: else load stages nothing
fi

conn=(--engine "$ENGINE" --transport "$TRANSPORT")
pass=0 fail=0
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$1"; pass=$((pass+1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$1"; fail=$((fail+1)); }
note() { printf '\033[1m== %s\033[0m\n' "$1"; }

# run <expect-exit> <label> <v-pkg args...>: run a verb, compare exit to expected.
run() {
  local want="$1" label="$2"; shift 2
  local rc=0
  "$VPKG" "$@" "${conn[@]}" --output json >"$WORK/out.json" 2>&1 || rc=$?
  if [[ "$rc" == "$want" ]]; then ok "$label (exit $rc)"; else
    bad "$label (exit $rc, want $want)"; sed -n '1,30p' "$WORK/out.json"; fi
  return 0
}

note "build the real packages"
"$VPKG" build "$MSL_DIR/kids/std.build.json" --src "$MSL_DIR/src" --out "$WORK/MSL.kids" >/dev/null && ok "build MSL.kids" || bad "build MSL.kids"
"$VPKG" build "$VSL_DIR/kids/vsl.build.json" --src "$VSL_DIR/src" --out "$WORK/VSL.kids" >/dev/null && ok "build VSL.kids" || bad "build VSL.kids"
MSL="$WORK/MSL.kids"; VSL="$WORK/VSL.kids"

note "pre-clean ($ENGINE) — best-effort, ignore if absent"
"$VPKG" uninstall "$VSL" "${conn[@]}" --force --deregister --output json >/dev/null 2>&1 || true
"$VPKG" uninstall "$MSL" "${conn[@]}" --force --deregister --output json >/dev/null 2>&1 || true

note "install + content-verify (happy path; VSL->MSL dependency satisfied)"
run 0 "install MSL"            install "$MSL" --allow-overwrite --register-package "M STANDARD LIBRARY"
run 0 "verify MSL (content)"   verify  "$MSL"
run 0 "install VSL (req-build MSL present)" install "$VSL" --allow-overwrite --register-package "VISTA STANDARD LIBRARY"
run 0 "verify VSL (content)"   verify  "$VSL"

note "back out + verify clean (--deregister clears the #9.4 footprint too)"
run 0 "uninstall VSL (--force --deregister)" uninstall "$VSL" --force --deregister
run 3 "verify VSL clean (installed:false)"   verify    "$VSL"
run 0 "uninstall MSL (--force --deregister)" uninstall "$MSL" --force --deregister
run 3 "verify MSL clean (installed:false)"   verify    "$MSL"

if [[ "$NEG" == "1" ]]; then
  note "negative dependency test — VSL alone, MSL absent + deregistered, expect REFUSED"
  # The back-out above deregistered MSL (cleared its #9.4 footprint), so
  # $$PATCH^XPDUTL("MSL*0.1*1")=0: VSL's Required-Build MSL*0.1*1 must now block the
  # install (env-check / required-build enforcement, exit 1). Live-proven on vehu
  # 2026-06-29 — the deregister is what makes this direction testable.
  run 1 "install VSL refused (req-build MSL absent)" install "$VSL" --allow-overwrite
  "$VPKG" uninstall "$VSL" "${conn[@]}" --force --deregister --output json >/dev/null 2>&1 || true
fi

note "summary — $ENGINE"
printf '  %d passed, %d failed\n' "$pass" "$fail"
[[ "$fail" == 0 ]]
