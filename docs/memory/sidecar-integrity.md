---
name: sidecar-integrity
description: v-pkg stamps the pre-image sidecar with a sha256 content hash in a private ("VPKG","HASH") node (stripped by EnginePairs) and refuses (SIDECAR_TAMPERED) a sidecar tampered after capture on restore / uninstall auto-restore.
metadata:
  type: project
---

**Sidecar integrity hash (verifiable-safety #3c, 2026-06-30)** — `install
--auto-snapshot` writes the pre-image sidecar `<kid>.preimage.kids` that uninstall
auto-restores ([[pairing-verify-clean]]). A sidecar tampered after capture would
silently restore the WRONG routine source. Now `buildSnapshotPairs` STAMPS the capture
with a content hash. Pairs with [[snapshot-restore]], [[class-aware-uninstall]].

**Mechanism (kids/sidecar.go):**
- `kids.HashPairs(pairs)` = sha256(hex) over `EnginePairs(pairs)`, each rendered as a
  canonical `(<subs>)=<value>` line, **sorted** (order-independent). Because it hashes
  EnginePairs, the `("VPKG",…)` metadata (the hash node itself + any foreign
  declaration) is EXCLUDED — so re-hashing a stamped build is stable (no chicken-egg).
- `kids.StampHash` appends `("VPKG","HASH")=<hex>`. Same ride-along pattern as the
  foreign-overwrite declaration ([[class-aware-uninstall]] F1): `EnginePairs` strips
  any `("VPKG",…)` node, so the hash never reaches KIDS filing; `WriteKID` writes it
  and `ParseKID` reads it back (string-sub nodes round-trip).
- `kids.VerifySidecarHash(b)` → (stored, ok, present). `verifySidecarIntegrity`
  (pkgcli/pairing.go) refuses **SIDECAR_TAMPERED** (exit 4) when present && !ok, and
  PASSES when not present (an authored `--backout` or a pre-#3c snapshot has nothing to
  verify). Wired into `restore` (before preview/commit) and uninstall's
  restore/auto-restore (after `loadBuild(restore)`, covering auto-detected sidecar AND
  explicit `--restore`).

**Why/how to apply:** sidecars produced by `install --auto-snapshot` / `snapshot` are
now tamper-evident automatically — no flag. Live-proven on vehu: a tampered sidecar
made uninstall auto-restore REFUSE and leave the routine untouched; the pristine
sidecar restored cleanly (verifyClean=clean). Gate: `make stress` snapshot→tamper→
restore-refused probe. Caveat (same as [[transport-checksum]]): this guards against
accidental/naive tampering of the sidecar's routine source; it is not a defense
against a signed-supply-chain attacker (that's the #4 attestation work).
