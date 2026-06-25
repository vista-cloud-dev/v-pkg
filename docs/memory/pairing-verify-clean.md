---
name: pairing-verify-clean
description: v-pkg install/uninstall pre-image pairing via the <kid>.preimage.kids sidecar convention + uninstall --verify (verify-clean).
metadata:
  type: project
---

**Pre-image pairing + verify-clean (2026-06-25)** — makes the install→uninstall
reversal work without re-specifying the snapshot path. Pairs with
[[class-aware-install]]/[[class-aware-uninstall]]/[[verify-drift]].

`defaultPreimagePath(kid)` = the `.KID` path with its ext swapped to
`.preimage.kids` (next to the patch). `install --auto-snapshot` writes the
pre-image there (instead of needing `--snapshot <path>`); `uninstall` with NO
`--restore`/`--backout` auto-detects that sidecar via `resolveAutoRestore`
(explicit flags always win; `--backout` suppresses auto-restore) and restores from
it. So the whole safe round-trip is just:

    v pkg install   <patch.kid> --auto-snapshot
    v pkg uninstall <patch.kid>            # auto-restores <patch>.preimage.kids

`uninstall --verify` (verify-clean): after a restore/back-out install, runs
`checkDrift` against the RE-APPLIED artifact and reports `verifyClean` clean/dirty
(exit 3 ExitCheck on dirty) — confirms the live routines actually match what was
re-applied. Reuses the [[verify-drift]] machinery.

Pure logic unit-tested (pkgcli/pairing_test.go: defaultPreimagePath,
resolveAutoRestore precedence). New flags (install --auto-snapshot, uninstall
--verify) → `make contract`. Engine legs (runInstall, checkDrift) are independently
live-proven; the auto-detect/verify wiring is unit-tested (engine demo skipped — it
mutates the shared engine).

**Why/how to apply:** prefer `install --auto-snapshot` for any national-routine
patch so uninstall is a clean one-liner. `uninstall --verify` to gate that the
reversal actually landed.
