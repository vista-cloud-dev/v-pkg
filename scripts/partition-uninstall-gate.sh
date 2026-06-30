#!/usr/bin/env bash
# partition-uninstall-gate.sh — live dual-engine proof of the PARTITIONED
# uninstall (BB1, v-rpc-tap-scalable plan §2). The v-rpc-tap build BOTH overwrites
# a foreign national routine (XWBPRS — restore it) AND adds greenfield routines
# (VSLRT* — delete them); a single restore would orphan the adds and a single
# delete would BRICK the national routine. This gate proves `v pkg uninstall`
# partitions the two correctly and ORDERS them (restore foreign FIRST, delete
# greenfield AFTER — F-I/R21), on a live engine.
#
# It is self-contained: it stands up its own synthetic two-package fixture in a
# throwaway ZZ* namespace (no real VistA routine is ever touched) —
#   P1 = {ZZVPTF v1}                      the "foreign" routine, installed first
#   P2 = {ZZVPTF v2 (overwrite), ZZVPTG}  the tap-shaped build (overwrite + add)
# then: install P2 --auto-snapshot (captures ZZVPTF v1) -> uninstall P2
# (auto-detects the sidecar -> actPartition) -> asserts ZZVPTF is RESTORED to v1
# and ZZVPTG is DELETED.
#
# All engine access goes through `v pkg <verb> --engine --transport` (the
# m-driver-sdk stack) — never a raw docker exec (waterline rule 3).
#
# Usage:
#   scripts/partition-uninstall-gate.sh                       # ydb / vehu (default)
#   ENGINE=iris TRANSPORT=remote scripts/partition-uninstall-gate.sh   # IRIS / foia-t12 (needs M_IRIS_*)
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
# jdata <python-index-expr>: read $WORK/out.json ["data"]. Uses raw_decode so a
# trailing error line (a failing verb prints its result JSON THEN an error) is
# ignored — only the first JSON object is parsed.
jdata() { python3 -c 'import json,sys;d=json.JSONDecoder().raw_decode(open(sys.argv[2]).read().lstrip())[0]["data"];print(eval("d"+sys.argv[1],{"d":d}))' "$1" "$WORK/out.json"; }

############################################################################
note "partition-uninstall gate ($ENGINE / $TRANSPORT)"
############################################################################

# --- synthetic fixture: two packages in a throwaway ZZ* namespace -----------
# The "foreign" routine differs between v1 and v2 on a CODE line (line 3), NOT
# line 2 — so the drift check (which canonicalizes the ;; line) can tell them
# apart and prove the restore actually put v1 back.
mkdir -p "$WORK/src1" "$WORK/src2"
cat > "$WORK/src1/ZZVPTF.m" <<'EOF'
ZZVPTF ;partition gate foreign routine
 ;;1.0
 ; baseline v1
 Q
EOF
cat > "$WORK/src2/ZZVPTF.m" <<'EOF'
ZZVPTF ;partition gate foreign routine
 ;;2.0
 ; OVERWRITTEN v2
 Q
EOF
cat > "$WORK/src2/ZZVPTG.m" <<'EOF'
ZZVPTG ;partition gate greenfield add
 ;;2.0
 Q
EOF
cat > "$WORK/p1.build.json" <<'EOF'
{ "package": "ZZVPT", "version": "1.0", "components": { "routines": ["ZZVPTF"] } }
EOF
cat > "$WORK/p2.build.json" <<'EOF'
{ "package": "ZZVPT", "version": "2.0", "components": { "routines": ["ZZVPTF", "ZZVPTG"] } }
EOF

P1="$WORK/p1.kids"; P2="$WORK/p2.kids"
[[ "$(rc build "$WORK/p1.build.json" --src "$WORK/src1" --out "$P1")" == 0 ]] && ok "build P1 (foreign baseline)" || { bad "build P1"; cat "$WORK/out.json"; }
[[ "$(rc build "$WORK/p2.build.json" --src "$WORK/src2" --out "$P2")" == 0 ]] && ok "build P2 (overwrite + greenfield add)" || { bad "build P2"; cat "$WORK/out.json"; }

# --- pre-clean (best effort) -------------------------------------------------
note "pre-clean (best-effort)"
rc uninstall "$P2" "${conn[@]}" --force --output json >/dev/null || true
rc uninstall "$P1" "${conn[@]}" --force --output json >/dev/null || true

# --- stage the foreign routine (P1) -----------------------------------------
note "install P1 -> ZZVPTF v1 now lives on the engine (the 'foreign' routine)"
[[ "$(rc install "$P1" "${conn[@]}" --output json)" == 0 ]] && ok "install P1" || { bad "install P1"; cat "$WORK/out.json"; }

# --- install the tap-shaped build, capturing the pre-image ------------------
note "install P2 --auto-snapshot -> overwrites ZZVPTF, adds ZZVPTG, captures pre-image"
r="$(rc install "$P2" "${conn[@]}" --auto-snapshot --output json)"
if [[ "$r" == 0 ]]; then
  ok "install P2 (auto-snapshot)"
  [[ "$(jdata "['action']")" == "snapshot+proceed" ]] && ok "P2 install captured a pre-image (snapshot+proceed)" || bad "P2 install action = $(jdata "['action']"), want snapshot+proceed"
else bad "install P2 (auto-snapshot)"; cat "$WORK/out.json"; fi
[[ -f "$WORK/p2.preimage.kids" ]] && ok "pre-image sidecar written (p2.preimage.kids)" || bad "no pre-image sidecar"

# --- THE PARTITION: uninstall P2, auto-detecting the sidecar -----------------
note "uninstall P2 -> MUST partition: restore ZZVPTF (foreign), delete ZZVPTG (greenfield)"
r="$(rc uninstall "$P2" "${conn[@]}" --verify --output json)"
if [[ "$r" == 0 ]]; then ok "uninstall P2 (exit 0)"; else bad "uninstall P2 (exit $r)"; cat "$WORK/out.json"; fi
[[ "$(jdata "['action']")" == "partition" ]] && ok "action = partition (not delete, not restore)" || bad "action = $(jdata "['action']"), want partition"
[[ "$(jdata "['autoDetected']")" == "True" ]] && ok "pre-image auto-detected from the sidecar" || bad "pre-image NOT auto-detected"
[[ "$(jdata "['restored']")" == "['ZZVPTF']" ]] && ok "restored = [ZZVPTF] (the foreign routine)" || bad "restored = $(jdata "['restored']"), want ['ZZVPTF']"
[[ "$(jdata "['deleted']")" == "['ZZVPTG']" ]] && ok "deleted = [ZZVPTG] (the greenfield add)" || bad "deleted = $(jdata "['deleted']"), want ['ZZVPTG']"
[[ "$(jdata "['done']")" == "True" ]] && ok "partition reported done" || bad "partition not done"
[[ "$(jdata "['verifyClean']")" == "clean" ]] && ok "verify-clean: ZZVPTF matches the pre-image" || bad "verify-clean = $(jdata "['verifyClean']"), want clean"

# --- prove the LIVE engine state independently ------------------------------
# (clikit prints the data envelope only on SUCCESS, so both probes below use
# verbs that exit 0 — never a failing verify, whose output carries no data.)
note "independent live assertions"
# ZZVPTF must be BACK to v1: P1's #9.7 is still filed, so `verify P1 --drift`
# succeeds and its drift for ZZVPTF must read 'applied' (live == P1 source).
r="$(rc verify "$P1" "${conn[@]}" --drift --output json)"
[[ "$r" == 0 ]] && ok "verify P1 ok (ZZVPTF present, #9.7 intact)" || { bad "verify P1 failed (exit $r)"; cat "$WORK/out.json"; }
[[ "$(jdata "['drift']['ZZVPTF']")" == "applied" ]] && ok "ZZVPTF restored to v1 (drift = applied — NOT bricked/deleted)" || bad "ZZVPTF drift = $(jdata "['drift']['ZZVPTF']"), want applied"
# ZZVPTG must be GONE: re-installing P2 must now see ZZVPTG as GREENFIELD (absent)
# and ZZVPTF as an OVERWRITE (present). If the partition had failed to delete
# ZZVPTG, it would show up under 'overwrites', not 'greenfield'.
rc install "$P2" "${conn[@]}" --auto-snapshot --output json >/dev/null
[[ "$(jdata "['greenfield']")" == "['ZZVPTG']" ]] && ok "ZZVPTG was deleted (re-install sees it greenfield/absent)" || bad "greenfield = $(jdata "['greenfield']"), want ['ZZVPTG'] (ZZVPTG not deleted?)"
[[ "$(jdata "['overwrites']")" == "['ZZVPTF']" ]] && ok "ZZVPTF was kept (re-install sees it as an overwrite/present)" || bad "overwrites = $(jdata "['overwrites']"), want ['ZZVPTF']"

# --- cleanup ----------------------------------------------------------------
note "cleanup"
rc uninstall "$P2" "${conn[@]}" --output json >/dev/null || true
rc uninstall "$P1" "${conn[@]}" --force --output json >/dev/null || true

############################################################################
printf '\033[1m== summary (%s)\033[0m  %d passed, %d failed\n' "$ENGINE" "$pass" "$fail"
[[ "$fail" == 0 ]]
