---
name: adversarial-stress-gate
description: scripts/adversarial-stress.sh + `make stress` — the live-gate's harder sibling; full MSL+VSL lifecycle (assembly→disassembly→install→verify+drift→back-out) with adversarial refusal probes. 56/56 both engines (YDB vehu + IRIS foia) 2026-06-30. Surfaced AND fixed the verify --drift TAB-indentation false-positive.
metadata:
  type: project
---

**Adversarial stress gate (2026-06-29)** — `scripts/adversarial-stress.sh` /
`make stress [ENGINE=ydb|iris] [TRANSPORT=] [OFFLINE=1]`. The live-package-gate
proves the happy path; this tries to BREAK things across the *whole* lifecycle and
asserts v-pkg REFUSES the unsafe moves. **56/56 on BOTH YDB/vehu and IRIS/foia (2026-06-30).**
Complements [[live-package-gate]], [[verify-drift]], [[class-aware-install]],
[[class-aware-uninstall]], [[install-attestation]], [[half-install-heal]], [[dry-run-compare]].

**Added 2026-06-30 (closing the V&V coverage boundaries):**
- **G7 heal corrupt-purge (live):** seeds a §7.1 corpse (`^XPD(9.7,ien)` with a `"B"`
  xref + `"ASP"` but NO `0`-node, at a fixed out-of-range IEN 9000001) via the approved
  ad-hoc `m vista exec` (`mseed` helper, MUST be `|| true` so a seed hiccup never aborts
  the `set -e` script; prefer the repo-local `../m-cli/dist/m`, not the PATH `m` wrapper),
  proves a normal install is FALSELY blocked (already-installed, exit 4), then `--heal`
  purges + reinstalls to status 3. Belt-and-suspenders K before AND after.
- **G6 dry-run CHANGED (live):** `diff ZZCK-bad` against the installed ZZCK-ok reads the
  one differing routine as `changed>=1` — the CHANGED case the standing live-gate (all-new
  / all-identical only) didn't cover.
- **Attestation #4 (8 probes):** install writes a record → chain verify → `--replay` →
  tampered-ledger refuse (exit 3) → uninstall chains a 2nd record. Now proven on IRIS too.

**Phase 1 — OFFLINE (22 asserts, no engine), per package:** build · determinism
(byte-identical rebuild) · parse (install-name) · classify (MSL=class-1 pure,
VSL=class-2 side-effecting) · lint (PIKS) · roundtrip · **decompose→assemble
disassembly** · component-lossless (reassembled section counts == original) ·
**tamper-faithfulness** (mutate a routine in the decomposed tree → reassembled
.KID must differ from original; packaging may not silently swallow a content change).

**Phase 2 — LIVE (15 asserts/engine), adversarial refusal probes:**
- install MSL --register → verify content → verify --drift = applied.
- **no-clobber:** `install MSL` with no `--allow-overwrite` over existing routines → REFUSE exit 4.
- install VSL (Required-Build MSL present) → verify content.
- **idempotency guard:** `install VSL --allow-overwrite` again, already filed in #9.7 →
  REFUSE exit 4 `error:"already-installed"`. (Install refuses to re-file an
  install-name already present even WITH --allow-overwrite — you must uninstall first.)
- **side-effecting back-out safety:** bare `uninstall VSL` (no flags) → REFUSE exit 4
  (don't orphan the #999001 file / param-def data); `--force` overrides.
- back out VSL `--force --deregister` → verify-clean (exit 3).
- **double back-out:** `uninstall VSL` again, already gone → graceful (exit 0, no panic/usage).
- back out MSL → verify-clean. **negative dependency:** `install VSL` alone, MSL
  deregistered → REFUSE exit 1.

**KEY FINDING (FOUND HERE, NOW FIXED) — `verify --drift` FALSE-POSITIVES on
TAB-indented routines (both engines).** All 6 v-stdlib (VSL) routines reported
`drifted` immediately after a CLEAN install, NO tampering; all 40 m-stdlib (MSL)
routines verified `applied`. Root cause: **v-stdlib was authored with leading TAB
indentation; m-stdlib with leading SPACES**, and an engine flattens a **leading TAB →
a single SPACE** on install (proven on vehu: shipped `'\t; doc...'` → live `' ; doc...'`;
same on foia-t12), so the live routine source diverged from the shipped `.KID` source
on *every* line. `RoutineDriftMatch` (`internal/kids/buildkids.go:141`) byte-compares
each line (except the canonicalized 2nd), so a uniform TAB→SPACE difference tripped
drift on all lines. **Content `verify` (no `--drift`) was unaffected** (it checks
entry-record content, not routine line bytes) — which is why this hid until the
adversarial gate ran drift on a clean install.

**FIX LANDED 2026-06-29 (both layers):** (1) **m-cli lint M-MOD-039** (`feat(lint)`,
m-cli `023d030`) — bans tabs in M source (Error, default profile) + an `m fmt`
canonical detab; closes the gate gap that documented "spaces only" but never enforced
it. (2) **v-stdlib detab** (`c830b48`) — the 6 `VSL*.m` (+ test routines) converted to
spaces, whitespace-only. With the shipped source now space-indented it matches what the
engine stores, so **VSL drifts `applied` on both engines**. The harness now ASSERTS
`verify --drift VSL` exit 0 (was a non-asserting observation) as the regression guard —
**37/37 both engines.**

**Why/how to apply:** run `make stress` (+ `ENGINE=iris TRANSPORT=remote`) after any
change to the lifecycle verbs, the class-aware decision logic, or the emitters — it's
the adversarial regression net beyond the happy-path live-gate.
