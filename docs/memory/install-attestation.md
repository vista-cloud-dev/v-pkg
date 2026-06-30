---
name: install-attestation
description: v-pkg #4 install attestation — every engine-mutating op appends a tamper-evident record (sha256 prevHash chain always + opt-in ed25519 sig) to a host-side JSONL ledger; AFTER is the FILED-source checksum (no extra engine read), replay checks NET per-routine state.
metadata:
  type: project
---

**Install attestation (verifiable-safety #4, 2026-06-30 — FINAL increment, closes
the effort).** Each engine-MUTATING op (install / uninstall / restore — never a
read-only diff/verify) appends a structured, tamper-EVIDENT record to a host-side
append-only JSON Lines ledger, so "it installed" becomes provable provenance. Core
is the engine-neutral `internal/attest` package; assembly + emit chokepoints are
`pkgcli/attest.go` (`liveInstall`/`runMulti`, the uninstall switch, `restore.Run`).
New verb `v pkg attest verify <ledger>`. Builds on [[transport-checksum]] (#3b) +
[[sidecar-integrity]] (#3c): those are SELF-consistency checks (sum lives in the
file an attacker controls); #4's signature is the real tamper-RESISTANCE they
deferred. Pairs with [[dry-run-compare]] (the dry-run plan is the record's pre-image).

**Durable design decisions (the non-obvious ones):**
- **AFTER = checksum of the source the op FILED, not a re-read from the engine.** The
  record is assembled with NO extra engine round-trip: BEFORE from the pre-image
  `captureRoutinePreimages` already reads (or `absent` for greenfield), AFTER from
  `b.RoutineSource(name)`. The kickoff floated "checkDrift/RoutineChecksumB for
  AFTER" (a live read) — rejected: the live read is the AUDITOR's job (`attest verify
  --replay`), not the writer's. Replay reads the engine and must still match AFTER.
- **Two tiers, layered:** a sha256 `prevHash` chain ALWAYS (append-only,
  tamper-evident — `VerifyChain`); an OPT-IN detached **ed25519** signature
  (`--sign`, key = `$VPKG_ATTEST_KEY` hex 32-byte seed) is tamper-RESISTANT
  (`attest verify --trust <hexpub>` / `$VPKG_ATTEST_PUBKEY`, `VerifyChainTrusted`).
- **Canonicalization needs NO third-party canonical-JSON dep:** the hash/signature
  cover `json.Encode(record with Hash+Signature zeroed)` — Go encodes struct fields
  in declaration order and map keys sorted, so it's deterministic. PubKey + PrevHash
  are INSIDE the canonical form (bound, can't be swapped); Hash/Signature are derived
  so excluded (no self-reference). Set PubKey BEFORE computing canonical when signing.
- **Replay checks NET per-routine state, not every historical record.** A routine
  installed then uninstalled has expected state = the AFTER of the LAST record that
  recorded one. Replaying every record instead false-fails: the install record's
  AFTER (`B19553`) no longer holds once a later uninstall removed the routine. Net
  state makes a full lifecycle ledger replay clean. (Caught live on vehu, fixed.)
- **Emit only when the op actually MUTATED the engine** (`res.Installed` / `res.Done`).
  A refused / no-op / already-installed op changed nothing → writes NO record. ON by
  default; `--no-attest` suppresses; a FAILED ledger write is surfaced loudly
  (`ATTEST_FAILED`) since the engine was already mutated.
- **Line-2-blind B checksum (`kids.BChecksum`) is what makes replay robust:** KIDS
  rewrites a routine's `;;version` line (line 2) at install, but `RoutineChecksumB`
  skips line 2, so the recorded AFTER still matches the live routine.

**Ledger:** host-side `<kid>.attest.jsonl` next to the .KID by default, `--attest
<path>` override. Same .KID's install + uninstall chain onto one ledger out of the box.

**Validation (all live-proven on vehu + offline):** greenfield install record (BEFORE
absent / AFTER B19553, genesis prevHash); chain verify + `--replay` clean; uninstall
appends a chained record (prevHash matches); a tampered field → chain breaks →
`attest verify` exits 3; `--sign` + `--trust` correct-key passes / wrong-key/unsigned
refused. Wired into [[adversarial-stress-gate]] (8 probes, `make stress` 51/51) and
`make live-gate` (replay assertion, 15/15); `make corpus` DRIFT=0 (2404) unchanged.
