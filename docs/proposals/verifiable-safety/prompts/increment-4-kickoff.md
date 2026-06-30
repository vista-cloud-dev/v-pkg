# Kickoff prompt — verifiable-safety hardening (increment #4)

Paste the block below into a **fresh session started in `~/vista-cloud-dev/v-pkg`**
(one session ↔ one repo). Canon for the effort is the live tracker
[`docs/verifiable-safety-tracker.md`](../../../verifiable-safety-tracker.md); this
prompt bootstraps a cold session into increment #4, the **final** increment.
**Increment #4 closes the effort — its landing commit must `git mv` this tracker AND
the whole `proposals/verifiable-safety/` folder (proposal + these prompts) into
`docs/archive/`, repointing inbound links** (org Increment Protocol step 2: a tracker
is Tier D — archive it when the effort lands).

---

```
You're completing the "verifiable-safety hardening" effort on v-pkg — making `v pkg`
the most accurate, reliable, and verifiably-safe KIDS installer. Work in
~/vista-cloud-dev/v-pkg (this repo only). This is the LAST increment (#4).

READ FIRST (canon, do not re-derive):
- docs/verifiable-safety-tracker.md            ← the 4-increment plan. §4 is THIS
                                                 increment. THIS GOVERNS. (§1–§3 done.)
- docs/memory/transport-checksum.md            ← #3b's KEY lesson: the checksum check
                                                 is a NON-cryptographic self-consistency
                                                 check (~1.6% of real KIDs are born
                                                 self-inconsistent); REAL tamper-
                                                 resistance is THIS increment.
- docs/memory/sidecar-integrity.md             ← #3c: kids.HashPairs / StampHash, the
                                                 ("VPKG",…) ride-along node pattern +
                                                 EnginePairs strip — REUSE for hashing.
- docs/memory/half-install-heal.md             ← #3a (the install path #4 instruments)
- docs/memory/class-aware-install.md + class-aware-uninstall.md + snapshot-restore.md
                                                 + dry-run-compare.md + verify-drift.md
                                                 ← the reports/probes #4 turns into an
                                                 attestation (they already carry most
                                                 of the change-set fields)
- pkgcli/lifecycle.go (liveInstall, runInstall, the installReport / uninstallReport
  shapes, checkDrift, captureRoutinePreimages), pkgcli/restore.go (restoreResult),
  pkgcli/heal.go, pkgcli/checksum.go, internal/kids/sidecar.go (HashPairs/StampHash),
  internal/kids/checksum.go (RoutineChecksumB), internal/kids/reversibility.go (Classify)

GOAL: execute the tracker's increment #4 (install attestation / audit record). #1–#3
are DONE + on main (latest ac823e9). #4 is the capstone — best landed last because it
records the richer change-set #1–#3 produce. Land it as ONE coherent verified
increment (or a small number of gate-green commits if it naturally splits), then CLOSE
the effort: archive the tracker + the proposals/verifiable-safety/ folder.

INCREMENT #4 — install attestation / audit record:

Each engine-MUTATING op (install / uninstall / restore — NOT read-only diff/verify)
appends a structured, tamper-EVIDENT attestation record so "it installed" becomes
provable provenance. A record captures, at minimum:
  - op (install|uninstall|restore) + sub-action (the existing Action string:
    proceed / snapshot+proceed / restore / partition / heal-purge / …)
  - build install-name(s), reversibility class, engine + transport
  - BEFORE/AFTER routine checksums (the real value of #4: prove what actually changed
    on the engine — reuse captureRoutinePreimages for BEFORE, checkDrift /
    RoutineChecksumB for AFTER)
  - components touched, Required-Build chain, snapshot/sidecar ref + its content hash
    (#3c HashPairs), #9.7 status, verify verdict, exit code
  - timestamp + a chain link (prev-record hash) so the ledger is append-only and
    tamper-EVIDENT on its own terms
The attestation is the pre-image of "provable provenance": an independent auditor can
replay it against the engine and confirm the recorded before/after.

This is ALSO where the REAL tamper-resistance #3b/#3c deferred belongs: those are
non-cryptographic SELF-consistency checks (the checksum lives in the same file an
attacker controls). #4 should support a DETACHED signature over each record against an
EXTERNAL key (the only thing that stops a determined tamperer) — design it in even if
signing is opt-in / a follow-up; the hash-chain is the floor, the signature is the
ceiling.

DESIGN CALLs to make early (then proceed):
- WHERE the ledger lives. Recommend a HOST-SIDE append-only file (JSON Lines), default
  next to the .KID or a configurable --attest <path> (and/or a per-repo default like
  ~/data/vista-cloud-dev/…). Host-side keeps it outside the engine an install mutates,
  and offline-auditable. (An optional engine-side copy in a ^XTMP node is a possible
  follow-up, NOT the primary store — keep the waterline clean.)
- HASH-CHAIN vs SIGNATURE. Recommend: each record carries prevHash (sha256 of the
  prior record's canonical form) → an append-only tamper-EVIDENT chain by default;
  an OPTIONAL detached ed25519 signature (--sign / key from env) is the real
  tamper-RESISTANCE. Be explicit in the record which protection applies.
- OPT-IN vs ALWAYS-ON. Recommend attestation ON by default for mutating ops (cheap,
  the whole point), with --no-attest to suppress; signing opt-in.
- A `v pkg attest verify <ledger>` read-only verb to validate the chain (+ signatures)
  and optionally replay before/after against the live engine. (New verb → contract.)

REUSE — do not build these from scratch:
- The change-set is ALREADY in installReport / uninstallReport / restoreResult
  (name/class/action/overwrites/greenfield/snapshot/heal/checksumWarnings/status/error)
  — the attestation record is a superset; assemble it from the report, don't re-probe.
- BEFORE checksums: captureRoutinePreimages already reads each target routine off the
  engine before mutation. AFTER: checkDrift / kids.RoutineChecksumB.
- Content hashing: kids.HashPairs (deterministic sha256 over EnginePairs) — reuse for
  the snapshot-ref hash and as the record-canonicalization primitive for the chain.
- Emit the record at the ONE chokepoint each op already funnels through (liveInstall,
  the uninstall switch, restore.Run) — never a parallel path.

NON-NEGOTIABLE VALIDATION (live where it touches the engine):
- An install on vehu writes an attestation record with correct BEFORE (absent→greenfield
  or the pre-image checksum) and AFTER (the installed routine's checksum) — replay
  confirms it. A second op appends, chaining to the first (prevHash matches).
- Tampering a ledger record (edit a field) breaks the chain → `attest verify` REFUSES
  (exit non-zero). A pristine ledger verifies clean. Offline unit test of the
  chain/canonicalization + (if built) signature verify — TDD red first.
- Read-only ops (diff/verify) write NO record (only mutating ops attest).
- Wire the new gate(s) into make stress / live-gate; keep make corpus DRIFT=0.

ENGINE ACCESS — driver stack ONLY (org hard rule; raw `docker exec` is gate-denied):
- ad-hoc M probe: `M_YDB_CONTAINER=vehu m vista exec --engine ydb --transport docker '<one M cmd>'`
  Gotchas: set `S U="^"`; wrap argumentless-FOR loops inside XECUTE (with doubled
  quotes); decode data.stdout with python. EACH Bash call is a fresh shell — re-export
  the M_YDB_* env every time (or prefix M_YDB_CONTAINER=vehu).
- live-gate env (vehu): export M_YDB_BIN=../m-ydb/dist/m-ydb /_CONTAINER=vehu
  /_TRANSPORT=docker /_DIST=/home/vehu/lib/gtm /_GBLDIR=/home/vehu/g/vehu.gld
  /_ROUTINES=/home/vehu/r.
- suites/coverage: `m test`; live: `make live-gate`, `make stress`; corpus:
  `make corpus` DRIFT=0.

PROCESS (org Increment Protocol — see ~/vista-cloud-dev/CLAUDE.md + v-pkg/CLAUDE.md):
- TDD is a HARD RULE: write the failing test first (chain canonicalization + verify,
  record assembly from a report, signature verify if built), confirm RED, implement,
  confirm GREEN.
- Gates BEFORE commit: `make lint test` (CGO_ENABLED=1, -race), `make corpus` (DRIFT=0),
  `make stress` / `make live-gate` for engine-bound behavior, and `make contract` if
  you add a verb/flag (the command surface is drift-gated).
- Commit straight to main (trunk-based, solo org); stage only files you touched; use
  the `Co-Authored-By: Claude …` trailer; never --no-verify / force-push main.
- At the close (this is the LAST increment): persist the durable lesson to
  docs/memory/ (e.g. install-attestation.md; extend, don't duplicate), mark the #4 row
  done, then `git mv` docs/verifiable-safety-tracker.md AND docs/proposals/verifiable-
  safety/ into docs/archive/ (repoint inbound links), commit, push. The effort is then
  complete.

CONTEXT — already done (don't redo): #1 genericized verify/uninstall over the
kids.Component registry; #2 added `install --dry-run` + `v pkg diff` (read-only plan);
#3a `install --heal` (corrupt half-install purge), #3b transport-checksum (WARN by
default / `--verify-checksums` hard gate — the SUMB port + the born-self-inconsistent
corpus finding), #3c sidecar integrity hash (kids.HashPairs/StampHash + the
("VPKG","HASH") ride-along). Your job is increment #4, then close the effort.

Begin with the WHERE/HASH-vs-SIGN/opt-in design calls, then write the first failing
test for the record canonicalization + hash-chain verify.
```
