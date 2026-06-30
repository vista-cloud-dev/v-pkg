---
name: install-uninstall-hardening
description: Adversarial-review hardening of v-pkg install/uninstall (2026-06-30) — per-op token isolates concurrent installs, #9.7 LOCK + ledger flock + status nonce, uninstall folds attestation Before into the foreign set, driver meta-caps handshake. Plus the verify-path M-injection fix.
metadata:
  type: project
---

**Install/uninstall hardening from an adversarial design review (2026-06-30).** Four
parallel investigators probed the install M-script, attestation, reversibility/uninstall,
and the seam/concurrency. Findings + fixes (commits `824610a`, `2882b20`, `a55fd08`):

**Verify-path M-injection (RCE-class) — see [[verify-script-m-injection]].** The one
true vulnerability: build-controlled names spliced raw into `Verify*Script` markers.
Fixed via `kids.MString` + `$T(@VRN)` indirection.

**Concurrency (no locking; org runs parallel agent sessions).** The install path used
FIXED scratch routine names + a shared `^XTMP("VPKGI")` + a same-tip ledger append:
- **Per-process `opToken`** (`crypto/rand`, pkgcli/lifecycle.go) now suffixes the
  scratch routine names (`ZVINS<tok>`…) AND subscripts the staging global
  `^XTMP("VPKGI",<tok>,…)`, so concurrent installs use DISJOINT scratch routines +
  staging subtrees. The staged-count loop is bounded to the token subtree with
  **`$QSUBSCRIPT(VR,1)="VPKGI" & $QS(VR,2)=token`** (else a concurrent op's subtree
  would inflate the count). Live-proven: two concurrent installs of distinct builds
  both reach #9.7 status 3, no cross-clobber.
- **`#9.7` dup-entry TOCTOU:** `FinalInstallScript` holds `L +^XTMP("VPKGLOCK",name):5`
  across the already-installed `$D` guard AND `$$INST^XPDIL1`. **KEY GOTCHA: an engine
  LOCK only spans a SINGLE ExecRun process** — the chunk Loads + finalize are separate
  processes, so the LOCK works *because* the guard+create are both inside the one
  finalize script. Released on every exit path; 5s timeout → `install-locked` refusal.
- **Ledger chain-fork:** two emits read the same `LastHash` tip → same `prevHash` →
  forked chain (a false-positive "tampering" on verify). Fixed with a **cross-process
  flock** (`attest.WithLedgerLock`, build-tagged unix/other) around read-tip→seal→
  append. (A host-side file lock, NOT an engine LOCK — the ledger is host-side.)
- **Status-marker forge (nonce, #5):** the success oracle is nonce-tagged
  `status:<opToken>=`; a build's pre/post M (runs inside `EN^XPDIJ`, before our write)
  can't pre-forge it. The Go side trusts only the token-tagged status line.

**Undeclared foreign-overwrite brick (#3).** Foreignness was self-declared only; an
undeclared national overwrite (installed via `--allow-overwrite`) was DELETED (bricked)
on a bare uninstall. Fix: `uninstall` folds the **durable install-time overwrite set**
— routines the attestation ledger's latest install record shows pre-existed
(`Before != "absent"`) — into the effective foreign set (`mergeForeign`), so the
existing F1 refuse/partition logic protects them. Degrades to declaration-only when no
ledger. The classifier-false-negative "load-time side effects" finding was INVALID (M
doesn't run a routine on filing — only when called; install side effects are the PRE/
INI/INIT hooks the classifier already detects).

**Driver contract handshake (#4).** `engineConn.checkDriver` calls `meta caps` before a
mutating op and refuses a driver whose contract MAJOR ≠ the SDK's (`mdriver.ContractVersion`)
or whose engine is wrong — catches accidental driver/version skew early. NOT a defense
against a forged driver (it owns the seam by definition; that's the attestation
signature's job).

**Threat-model framing (kept for future reviews):** most attestation "gaps" the review
raised (host-side ledger deletion, AFTER-from-filed-source, co-located signing key) are
DOCUMENTED tradeoffs — the hash chain is tamper-EVIDENT, `--sign` + external verify is
tamper-RESISTANT. Don't re-raise them as bugs. Validation: `make stress` 56/56 dual-engine
(vehu + foia), `TestWithLedgerLock_ConcurrentAppendsStayChained` (30 goroutines → one
valid chain). See [[adversarial-stress-gate]], [[class-aware-uninstall]], [[install-attestation]].
