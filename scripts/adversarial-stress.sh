#!/usr/bin/env bash
# adversarial-stress.sh — full end-to-end ADVERSARIAL stress of v-pkg over the
# org's two REAL packages, MSL (m-stdlib) and VSL (v-stdlib).
#
# This is the live-package-gate's harder sibling. The gate proves the happy path
# (install -> verify -> back out -> clean). This harness tries to BREAK things:
# it exercises the *whole* lifecycle — assembly/packaging, DISASSEMBLY (decompose
# /assemble round-trip), installation, verification + DRIFT, de-installation and
# back-out — and asserts that v-pkg REFUSES the unsafe moves rather than silently
# doing damage.
#
#   Phase 1 (OFFLINE, no engine): assembly + disassembly + tamper-faithfulness.
#   Phase 2 (LIVE, per engine):   install/verify/drift/uninstall with adversarial
#                                 refusal + drift-catch + double-uninstall probes.
#
# All engine access goes through `v pkg <verb> --engine --transport` (the
# m-driver-sdk stack) — never a raw docker exec (waterline rule 3).
#
# Usage:
#   scripts/adversarial-stress.sh                       # ydb / vehu (default)
#   ENGINE=iris TRANSPORT=remote scripts/adversarial-stress.sh   # IRIS / foia-t12 (needs M_IRIS_*)
#   OFFLINE=1 scripts/adversarial-stress.sh             # phase 1 only, skip the engine
set -euo pipefail

ENGINE="${ENGINE:-ydb}"
TRANSPORT="${TRANSPORT:-docker}"
OFFLINE="${OFFLINE:-0}"

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VPKG="${VPKG:-$here/dist/v-pkg}"
MSL_DIR="${MSL_DIR:-$HOME/vista-cloud-dev/m-stdlib}"
VSL_DIR="${VSL_DIR:-$HOME/vista-cloud-dev/v-stdlib}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

if [[ "$ENGINE" == "ydb" ]]; then
  export M_YDB_BIN="${M_YDB_BIN:-$HOME/vista-cloud-dev/m-ydb/dist/m-ydb}"
  export M_YDB_CONTAINER="${M_YDB_CONTAINER:-vehu}"
  export M_YDB_TRANSPORT="${M_YDB_TRANSPORT:-docker}"
  export M_YDB_DIST="${M_YDB_DIST:-/home/vehu/lib/gtm}"
  export M_YDB_GBLDIR="${M_YDB_GBLDIR:-/home/vehu/g/vehu.gld}"
  export M_YDB_ROUTINES="${M_YDB_ROUTINES:-/home/vehu/r}"
fi

conn=(--engine "$ENGINE" --transport "$TRANSPORT")
pass=0 fail=0
ok()   { printf '  \033[32mPASS\033[0m %s\n' "$1"; pass=$((pass+1)); }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$1"; fail=$((fail+1)); }
note() { printf '\033[1m== %s\033[0m\n' "$1"; }

# rc <v-pkg args...>: run, echo the exit code, never abort the script.
rc() { local r=0; "$VPKG" "$@" >"$WORK/out.json" 2>&1 || r=$?; echo "$r"; }

# expect <want-exit> <label> <v-pkg args...>: live verb, compare exit.
expect() {
  local want="$1" label="$2"; shift 2
  local r; r="$(rc "$@" "${conn[@]}" --output json)"
  if [[ "$r" == "$want" ]]; then ok "$label (exit $r)"
  else bad "$label (exit $r, want $want)"; sed -n '1,25p' "$WORK/out.json"; fi
}

# jget <jsonpath-ish python expr> reads $WORK/out.json data
jdata() { python3 -c 'import json,sys;print(eval("d"+sys.argv[1],{"d":json.load(open(sys.argv[2]))["data"]}))' "$1" "$WORK/out.json"; }

############################################################################
note "PHASE 1 — OFFLINE assembly + disassembly (engine-independent)"
############################################################################
declare -A SPEC=( [MSL]="$MSL_DIR/kids/std.build.json" [VSL]="$VSL_DIR/kids/vsl.build.json" )
declare -A SRC=(  [MSL]="$MSL_DIR/src"                 [VSL]="$VSL_DIR/src" )
declare -A WANTCLASS=( [MSL]=1 [VSL]=2 )

for P in MSL VSL; do
  note "[$P] assembly + disassembly"
  K="$WORK/$P.kids"; K2="$WORK/${P}2.kids"

  # assemble / package
  [[ "$(rc build "${SPEC[$P]}" --src "${SRC[$P]}" --out "$K")" == 0 ]] && ok "$P build" || bad "$P build"

  # determinism — rebuild must be byte-identical (non-deterministic packaging = unauditable)
  rc build "${SPEC[$P]}" --src "${SRC[$P]}" --out "$K2" >/dev/null
  cmp -s "$K" "$K2" && ok "$P build deterministic (byte-identical rebuild)" || bad "$P build NOT deterministic"

  # parse — must report an install name
  if [[ "$(rc parse "$K" --output json)" == 0 ]] && [[ -n "$(jdata "['installNames'][0]")" ]]; then
    ok "$P parse ($(jdata "['installNames'][0]"))"
  else bad "$P parse"; fi
  sec_orig="$(jdata "['builds'][0]['sections']")"

  # classify — wrong class => wrong back-out strategy (the core safety contract)
  rc classify "$K" --output json >/dev/null
  cls="$(jdata "['class']")"
  [[ "$cls" == "${WANTCLASS[$P]}" ]] && ok "$P classify = class ${WANTCLASS[$P]} ($(jdata "['className']"))" \
    || bad "$P classify = $cls, want ${WANTCLASS[$P]}"

  # lint — PIKS data-class gate (no Patient/Institution data may leak)
  [[ "$(rc lint "$K")" == 0 ]] && ok "$P lint clean (PIKS gate)" || bad "$P lint blocked"

  # roundtrip — semantic self-consistency (parse->assemble reproduces, line-2 canon)
  [[ "$(rc roundtrip "$K")" == 0 ]] && ok "$P roundtrip (semantic)" || bad "$P roundtrip drift"

  # DISASSEMBLY: decompose -> tree -> assemble -> re.kids
  T="$WORK/${P}_tree"; RE="$WORK/${P}_re.kids"
  [[ "$(rc decompose "$K" "$T")" == 0 ]] && ok "$P decompose ($(find "$T" -type f|wc -l) files)" || bad "$P decompose"
  [[ "$(rc assemble "$T" "$RE")" == 0 ]] && ok "$P assemble (reassembled)" || bad "$P assemble"
  [[ "$(rc roundtrip "$RE")" == 0 ]] && ok "$P reassembled roundtrips" || bad "$P reassembled drift"

  # disassembly is COMPONENT-lossless: section counts must match the original
  rc parse "$RE" --output json >/dev/null
  sec_re="$(jdata "['builds'][0]['sections']")"
  [[ "$sec_re" == "$sec_orig" ]] && ok "$P disassembly component-lossless (sections match)" \
    || { bad "$P disassembly LOST/ADDED components"; echo "    orig=$sec_orig"; echo "    re  =$sec_re"; }

  # TAMPER faithfulness: mutate a routine in the tree -> reassembled MUST differ
  # (a packaging path that silently swallows a content change can't be trusted).
  RT="$(find "$T" -name 'STDARGS*.m' -o -name 'VSLCFG*.m' | head -1)"
  printf '\nZZDRIFT ; adversarial drift probe\n Q\n' >> "$RT"
  rc assemble "$T" "$WORK/${P}_tamper.kids" >/dev/null
  cmp -s "$K" "$WORK/${P}_tamper.kids" && bad "$P TAMPER swallowed (reassembled == original)" \
    || ok "$P tamper carried through (content change is auditable)"
done

if [[ "$OFFLINE" == "1" ]]; then
  note "summary — OFFLINE only"; printf '  %d passed, %d failed\n' "$pass" "$fail"; [[ "$fail" == 0 ]]; exit
fi

MSL="$WORK/MSL.kids"; VSL="$WORK/VSL.kids"

############################################################################
note "PHASE 2 — LIVE lifecycle + adversarial probes ($ENGINE)"
############################################################################
note "pre-clean — best-effort"
rc uninstall "$VSL" "${conn[@]}" --force --deregister --output json >/dev/null || true
rc uninstall "$MSL" "${conn[@]}" --force --deregister --output json >/dev/null || true

note "install + verify (happy path, dependency satisfied)"
expect 0 "install MSL (--register)"     install "$MSL" --allow-overwrite --register-package "M STANDARD LIBRARY"
expect 0 "verify MSL (content)"         verify  "$MSL"
expect 0 "verify --drift MSL (applied)" verify  "$MSL" --drift   # MSL is space-indented: drift round-trips clean

note "ADVERSARIAL: a second install with no --allow-overwrite must REFUSE (no silent clobber)"
expect 4 "install MSL again, no overwrite flag → refused" install "$MSL" --register-package "M STANDARD LIBRARY"

note "ADVERSARIAL: --heal must REFUSE a HEALTHY install (heal repairs only a corrupt half-install, never clobbers)"
expect 4 "install MSL --heal on a healthy install → refused" install "$MSL" --heal --allow-overwrite

note "ADVERSARIAL: transport-checksum gate (--verify-checksums) — pristine passes, tampered REFUSED (offline, pre-connect)"
# A foreign-style routine-only .KID carrying a REAL stored checksum (B10838 over the
# ground-truthed ZZT source). The tampered copy flips one command line so the source
# no longer reproduces the stored checksum.
mkck() { sed "s/__L5__/$2/" >"$1" <<'KID'
KIDS Distribution saved by v-pkg
v-pkg reassembled output
**KIDS**:ZZCK*1.0*1^

**INSTALL NAME**
ZZCK*1.0*1
"BLD",1,0)
ZZCK*1.0*1^ZZT^0^0
"RTN")
1
"RTN","ZZT")
0^5^B10838
"RTN","ZZT",1,0)
ZZT ;test routine ;1.0
"RTN","ZZT",2,0)
 ;;1.0;ZZT;;
"RTN","ZZT",3,0)
 Q
"RTN","ZZT",4,0)
PING() ;
"RTN","ZZT",5,0)
__L5__
"VER")
8.0^22.2
**END**
**END**
KID
}
mkck "$WORK/ZZCK-ok.kids"   ' Q "pong"'
mkck "$WORK/ZZCK-bad.kids"  ' Q "PONG"'
expect 4 "install tampered foreign .KID --verify-checksums → refused (CHECKSUM_MISMATCH)" install "$WORK/ZZCK-bad.kids" --verify-checksums
expect 0 "install pristine foreign .KID --verify-checksums → passes the gate + installs" install "$WORK/ZZCK-ok.kids" --verify-checksums
expect 0 "uninstall the throwaway ZZCK" uninstall "$WORK/ZZCK-ok.kids" --force

note "install VSL (Required-Build MSL present)"
expect 0 "install VSL (--register)" install "$VSL" --allow-overwrite --register-package "VISTA STANDARD LIBRARY"
expect 0 "verify VSL (content)"     verify  "$VSL"

note "ADVERSARIAL: re-installing an already-filed install-name must REFUSE (idempotency guard, even with --allow-overwrite)"
expect 4 "install VSL again (already filed in #9.7) → refused" install "$VSL" --allow-overwrite

# Regression guard for the tab-indentation false-drift (found by this harness,
# 2026-06-29): v-stdlib's routines were TAB-indented and an engine flattens a
# leading TAB->SPACE on install, so the live source diverged from the shipped
# source on every line and drift false-positived. Fixed by detabbing v-stdlib
# (spaces only) + m-cli lint M-MOD-039; VSL now drifts clean like MSL.
expect 0 "verify --drift VSL (applied; no tab false-drift)" verify "$VSL" --drift

note "ADVERSARIAL: VSL is side-effecting — bare uninstall must REFUSE (don't orphan file/param data)"
expect 4 "uninstall VSL, no flags → refused" uninstall "$VSL"

note "back out + verify clean"
expect 0 "uninstall VSL (--force --deregister)" uninstall "$VSL" --force --deregister
expect 3 "verify VSL clean (installed:false)"   verify    "$VSL"

note "ADVERSARIAL: double back-out of an already-removed package must be graceful (no usage/panic)"
r="$(rc uninstall "$VSL" "${conn[@]}" --force --deregister --output json)"
case "$r" in 0|1|3) ok "double-uninstall VSL graceful (exit $r)";; *) bad "double-uninstall VSL exit $r (want 0/1/3)";; esac

note "finish back-out — MSL"
expect 0 "uninstall MSL (--force --deregister)" uninstall "$MSL" --force --deregister
expect 3 "verify MSL clean (installed:false)"   verify    "$MSL"

note "ADVERSARIAL: negative dependency — VSL alone, MSL absent + deregistered, must be REFUSED"
expect 1 "install VSL alone → refused (Required-Build MSL absent)" install "$VSL" --allow-overwrite
rc uninstall "$VSL" "${conn[@]}" --force --deregister --output json >/dev/null || true

note "summary — $ENGINE"
printf '  %d passed, %d failed\n' "$pass" "$fail"
[[ "$fail" == 0 ]]
