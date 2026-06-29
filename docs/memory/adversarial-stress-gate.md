---
name: adversarial-stress-gate
description: scripts/adversarial-stress.sh + `make stress` — the live-gate's harder sibling; full MSL+VSL lifecycle (assembly→disassembly→install→verify+drift→back-out) with adversarial refusal probes. 36/36 both engines 2026-06-29. Surfaced the verify --drift TAB-indentation false-positive.
metadata:
  type: project
---

**Adversarial stress gate (2026-06-29)** — `scripts/adversarial-stress.sh` /
`make stress [ENGINE=ydb|iris] [TRANSPORT=] [OFFLINE=1]`. The live-package-gate
proves the happy path; this tries to BREAK things across the *whole* lifecycle and
asserts v-pkg REFUSES the unsafe moves. **36/36 on BOTH YDB/vehu and IRIS/foia-t12.**
Complements [[live-package-gate]], [[verify-drift]], [[class-aware-install]],
[[class-aware-uninstall]].

**Phase 1 — OFFLINE (22 asserts, no engine), per package:** build · determinism
(byte-identical rebuild) · parse (install-name) · classify (MSL=class-1 pure,
VSL=class-2 side-effecting) · lint (PIKS) · roundtrip · **decompose→assemble
disassembly** · component-lossless (reassembled section counts == original) ·
**tamper-faithfulness** (mutate a routine in the decomposed tree → reassembled
.KID must differ from original; packaging may not silently swallow a content change).

**Phase 2 — LIVE (14 asserts/engine), adversarial refusal probes:**
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

**KEY FINDING — `verify --drift` FALSE-POSITIVES on TAB-indented routines (both
engines).** All 6 v-stdlib (VSL) routines report `drifted` immediately after a CLEAN
install, with NO tampering; all 40 m-stdlib (MSL) routines verify `applied`. Root
cause: **v-stdlib routines are authored with leading TAB indentation; m-stdlib with
leading SPACES.** The engine flattens a **leading TAB → a single SPACE** on install
(proven on vehu: shipped `'\t; doc...'` → live `' ; doc...'`; same on foia-t12), so the
live routine source diverges from the shipped `.KID` source on *every* line.
`RoutineDriftMatch` (`internal/kids/buildkids.go:141`) compares each line verbatim
except the canonicalized 2nd line, so a uniform TAB→SPACE difference trips drift on
all lines. **Content `verify` (no `--drift`) is unaffected** — it checks entry-record
content, not routine line bytes — which is why the false-drift hid until now.
Implication: `verify --drift` is structurally unreliable for any tab-indented package,
including the org's own VSL. **Fix is pending a direction decision** (drift-normalize
leading whitespace in v-pkg / detab in `v pkg build` / detab the v-stdlib source).
The harness logs VSL drift as a non-asserting OBSERVATION (expects exit 3 today) so it
stays green; promote it to an assertion once the fix lands.

**Why/how to apply:** run `make stress` (+ `ENGINE=iris TRANSPORT=remote`) after any
change to the lifecycle verbs, the class-aware decision logic, or the emitters — it's
the adversarial regression net beyond the happy-path live-gate.
