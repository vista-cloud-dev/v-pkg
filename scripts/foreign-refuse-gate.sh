#!/usr/bin/env bash
# foreign-refuse-gate.sh — live dual-engine proof of the F1 brick-path fix: a build
# that DECLARES a foreign-routine overwrite (foreignRoutines in the spec, embedded
# in the .KID) is REFUSED at uninstall when no pre-image is available, instead of
# falling through to a delete that would brick the foreign/national routine.
#
# Companion to scripts/partition-uninstall-gate.sh (which proves the WITH-sidecar
# partition path). This gate proves the WITHOUT-sidecar path:
#   P1 = {ZZFGN v1}                              the "foreign" routine, installed first
#   P2 = {ZZFGN v2 (overwrite), ZZVPTG} + foreignRoutines:["ZZFGN"]   the tap-shaped build
# then: install P2 --auto-snapshot (captures ZZFGN v1) -> REMOVE the sidecar ->
#   uninstall P2 (no flags) MUST be REFUSED (exit 4), ZZFGN left intact (not deleted);
#   uninstall P2 --force deletes ONLY ZZVPTG, leaving the declared-foreign ZZFGN intact.
# Plus an OFFLINE build-guard check: a foreignRoutines entry the build does not ship
# fails the build.
#
# All engine access goes through `v pkg <verb> --engine --transport` (the
# m-driver-sdk stack) — never a raw docker exec (waterline rule 3).
#
# Usage:
#   scripts/foreign-refuse-gate.sh                                    # ydb / vehu (default)
#   ENGINE=iris TRANSPORT=remote scripts/foreign-refuse-gate.sh       # IRIS / foia-t12 (needs M_IRIS_*)
set -euo pipefail

ENGINE="${ENGINE:-ydb}"
TRANSPORT="${TRANSPORT:-docker}"

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VPKG="${VPKG:-$here/dist/v-pkg}"
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
# jdata <python-index-expr>: read $WORK/out.json ["data"] (raw_decode tolerates a
# trailing error line printed after the result JSON).
jdata() { python3 -c 'import json,sys;d=json.JSONDecoder().raw_decode(open(sys.argv[2]).read().lstrip())[0]["data"];print(eval("d"+sys.argv[1],{"d":d}))' "$1" "$WORK/out.json"; }
# jexit: read the ["exit"] field of the (possibly error) result JSON.
jexit() { python3 -c 'import json,sys;print(json.JSONDecoder().raw_decode(open(sys.argv[1]).read().lstrip())[0]["exit"])' "$WORK/out.json"; }

############################################################################
note "foreign-overwrite refuse gate ($ENGINE / $TRANSPORT)"
############################################################################

# --- synthetic fixture -------------------------------------------------------
mkdir -p "$WORK/src1" "$WORK/src2"
cat > "$WORK/src1/ZZFGN.m" <<'EOF'
ZZFGN ;foreign routine — owned by "another package"
 ;;1.0
 ; baseline v1
 Q
EOF
cat > "$WORK/src2/ZZFGN.m" <<'EOF'
ZZFGN ;foreign routine — overwritten by the tap-shaped build
 ;;2.0
 ; OVERWRITTEN v2
 Q
EOF
cat > "$WORK/src2/ZZVPTG.m" <<'EOF'
ZZVPTG ;greenfield add
 ;;2.0
 Q
EOF
# P1 is the "foreign" routine's owning package (namespace ZZFGN), installed first.
cat > "$WORK/p1.build.json" <<'EOF'
{ "package": "ZZFGN", "version": "1.0", "components": { "routines": ["ZZFGN"] } }
EOF
# P2 is the tap-shaped build: package ZZVPT overwrites ZZFGN (declared foreign) and
# adds greenfield ZZVPTG.
cat > "$WORK/p2.build.json" <<'EOF'
{ "package": "ZZVPT", "version": "2.0", "components": { "routines": ["ZZFGN", "ZZVPTG"] }, "foreignRoutines": ["ZZFGN"] }
EOF
# A bad spec: declares a foreign routine the build does not ship.
cat > "$WORK/bad.build.json" <<'EOF'
{ "package": "ZZVPT", "version": "2.0", "components": { "routines": ["ZZVPTG"] }, "foreignRoutines": ["ZZFGN"] }
EOF

# --- OFFLINE build-time guard (no engine) ------------------------------------
note "build-time guard (offline)"
[[ "$(rc build "$WORK/bad.build.json" --src "$WORK/src2" --out "$WORK/bad.kids")" != 0 ]] \
  && ok "build FAILS when foreignRoutines names a routine the build does not ship" \
  || bad "build accepted an unshipped foreign declaration"

P1="$WORK/p1.kids"; P2="$WORK/p2.kids"
[[ "$(rc build "$WORK/p1.build.json" --src "$WORK/src1" --out "$P1")" == 0 ]] && ok "build P1 (foreign baseline)" || { bad "build P1"; cat "$WORK/out.json"; }
[[ "$(rc build "$WORK/p2.build.json" --src "$WORK/src2" --out "$P2")" == 0 ]] && ok "build P2 (declares foreignRoutines)" || { bad "build P2"; cat "$WORK/out.json"; }
grep -q '"VPKG","FOREIGN","ZZFGN")' "$P2" && ok "P2 .KID embeds the foreign declaration" || bad "P2 .KID missing the embedded foreign node"

# --- pre-clean (best effort) -------------------------------------------------
note "pre-clean (best-effort)"
rc uninstall "$P2" "${conn[@]}" --force --output json >/dev/null || true
rc uninstall "$P1" "${conn[@]}" --force --output json >/dev/null || true

# --- stage the foreign routine (P1) -----------------------------------------
note "install P1 -> ZZFGN v1 now lives on the engine (the 'foreign' routine)"
[[ "$(rc install "$P1" "${conn[@]}" --output json)" == 0 ]] && ok "install P1" || { bad "install P1"; cat "$WORK/out.json"; }

# --- install the tap-shaped build, capturing the pre-image ------------------
note "install P2 --auto-snapshot -> overwrites ZZFGN, adds ZZVPTG, captures pre-image"
[[ "$(rc install "$P2" "${conn[@]}" --auto-snapshot --output json)" == 0 ]] && ok "install P2 (auto-snapshot)" || { bad "install P2"; cat "$WORK/out.json"; }
SIDE="$WORK/p2.preimage.kids"
[[ -f "$SIDE" ]] && ok "pre-image sidecar written" || bad "no pre-image sidecar"

# A re-install probe sees ZZFGN and ZZVPTG as overwrites (both now present).
rc install "$P2" "${conn[@]}" --auto-snapshot --output json >/dev/null
[[ "$(jdata "['overwrites']")" == *ZZFGN* ]] && ok "ZZFGN present (overwrite) after install" || bad "overwrites = $(jdata "['overwrites']")"

# --- THE REFUSE: remove the sidecar, then bare uninstall --------------------
note "REMOVE the sidecar, then uninstall P2 (no flags) -> MUST be REFUSED"
rm -f "$SIDE"
[[ ! -f "$SIDE" ]] && ok "sidecar removed" || bad "sidecar still present"
r="$(rc uninstall "$P2" "${conn[@]}" --output json)"
[[ "$r" == 4 ]] && ok "bare uninstall REFUSED (exit 4 ExitRefused)" || { bad "bare uninstall exit = $r, want 4"; cat "$WORK/out.json"; }
[[ "$(jexit)" == "4" ]] && ok "result JSON reports exit 4" || bad "result exit = $(jexit)"

# --- prove the engine was UNTOUCHED -----------------------------------------
# Re-install probe: both ZZFGN and ZZVPTG must STILL be present as overwrites
# (refuse deleted nothing — neither falls into the greenfield/absent set).
note "independent live assertions — engine untouched by the refused uninstall"
rc install "$P2" "${conn[@]}" --auto-snapshot --output json >/dev/null
ovr="$(jdata "['overwrites']")"
[[ "$ovr" == *ZZFGN* ]]  && ok "ZZFGN still present (not bricked)" || bad "overwrites = $ovr"
[[ "$ovr" == *ZZVPTG* ]] && ok "ZZVPTG still present (refuse deleted nothing)" || bad "overwrites = $ovr (ZZVPTG missing -> deleted by a refuse?)"
rm -f "$SIDE"  # drop the sidecar the probe re-captured

# --- the --force escape hatch: delete greenfield ONLY, spare the foreign -----
note "uninstall P2 --force (no sidecar) -> delete ZZVPTG only, leave ZZFGN intact"
r="$(rc uninstall "$P2" "${conn[@]}" --force --output json)"
[[ "$r" == 0 ]] && ok "forced uninstall ran (exit 0)" || { bad "forced uninstall exit = $r"; cat "$WORK/out.json"; }
[[ "$(jdata "['deleted']")" == "['ZZVPTG']" ]] && ok "deleted = [ZZVPTG] (greenfield subset only)" || bad "deleted = $(jdata "['deleted']"), want ['ZZVPTG']"
# Probe: ZZFGN still present (foreign spared, an overwrite), ZZVPTG now absent
# (greenfield). After deleting ZZVPTG the greenfield key is present in the report.
rc install "$P2" "${conn[@]}" --auto-snapshot --output json >/dev/null
[[ "$(jdata "['overwrites']")" == *ZZFGN* ]] && ok "ZZFGN spared by --force (still present)" || bad "ZZFGN missing after --force: overwrites = $(jdata "['overwrites']")"
[[ "$(jdata "['greenfield']")" == *ZZVPTG* ]] && ok "ZZVPTG deleted by --force (now greenfield)" || bad "greenfield = $(jdata "['greenfield']"), want it to contain ZZVPTG"
rm -f "$SIDE"

# --- cleanup ----------------------------------------------------------------
note "cleanup"
rc uninstall "$P2" "${conn[@]}" --force --output json >/dev/null || true
rc uninstall "$P1" "${conn[@]}" --force --output json >/dev/null || true

############################################################################
printf '\033[1m== summary (%s)\033[0m  %d passed, %d failed\n' "$ENGINE" "$pass" "$fail"
[[ "$fail" == 0 ]]
