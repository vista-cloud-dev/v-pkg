#!/usr/bin/env bash
# roundtrip-demo.sh — end-to-end assembly / disassembly walkthrough for v-pkg.
#
# Drives every v-pkg verb against a real Kernel patch (XU*8.0*504, the KAAJEE
# proxy-logon build) and proves the round-trip guarantee:
#
#     parse → decompose → (inspect tree) → assemble → re-parse → roundtrip → lint
#
# It is hermetic: it builds the binary, works in a temp directory, and tears
# down on exit. Run it from anywhere inside the repo:
#
#     examples/roundtrip-demo.sh
#
# Exit status is the status of the final `roundtrip` (0 = build reproduced).
set -euo pipefail

# --- locate repo root and the sample build -------------------------------
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/.." && pwd)"
KID="$HERE/XU_8.0_504.KID"
BIN="$ROOT/dist/v-pkg"

# --- build the binary if needed -------------------------------------------
if [[ ! -x "$BIN" ]]; then
  echo "==> building v-pkg"
  ( cd "$ROOT" && make build >/dev/null )
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

rule() { printf '\n=== %s ===\n' "$*"; }

rule "1. PARSE — summarize the .KID without unpacking it"
"$BIN" parse "$KID"

rule "2. DECOMPOSE — explode the .KID into a per-component tree"
"$BIN" decompose "$KID" "$WORK/tree"
echo "--- resulting KIDComponents/ tree ---"
( cd "$WORK/tree" && find . -type f | sort )

rule "3. INSPECT — a decomposed routine (.m) and a Kernel component (.zwr)"
echo "--- Routines/XU8P504.m (first lines) ---"
sed -n '1,6p' "$WORK/tree/XU_8.0_504/KIDComponents/Routines/XU8P504.m"
echo "--- KRN/OPTION/XUS-KAAJEE-WEB-LOGON.zwr ---"
cat "$WORK/tree/XU_8.0_504/KIDComponents/KRN/OPTION/XUS-KAAJEE-WEB-LOGON.zwr"

rule "4. ASSEMBLE — rebuild an installable .KID from the tree"
"$BIN" assemble "$WORK/tree" "$WORK/rebuilt.KID"
echo "--- rebuilt header ---"
sed -n '1,3p' "$WORK/rebuilt.KID"

rule "5. RE-PARSE — the rebuilt .KID carries the same build & subscripts"
"$BIN" parse "$WORK/rebuilt.KID"

rule "6. ROUNDTRIP — the oracle: decompose→assemble→re-parse, exit 3 on drift"
"$BIN" roundtrip "$KID"
echo "roundtrip exit = $?"

rule "7. LINT — PIKS data-class gate (exit 3 if operational PHI/PII data present)"
"$BIN" lint "$KID"
echo "lint exit = $?"

rule "DONE — the build round-tripped with no drift"
