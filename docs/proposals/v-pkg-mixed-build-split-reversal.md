---
title: "v-pkg: mixed-build split reversal + declared-foreign overwrites — the high-risk national-routine-splice lifecycle"
status: PROPOSED (2026-06-28) — not built
created: 2026-06-28
for: extending v-pkg's class-aware install/uninstall so a build that BOTH overwrites a foreign national routine AND adds greenfield routines can be installed and FULLY reversed through the one sanctioned v-pkg lifecycle — no bespoke installer
related:
  - docs/patch-existing-routines-proposal.md (IMPLEMENTED — snapshot/restore/classify/class-aware uninstall this builds on)
  - docs/memory/class-aware-uninstall.md, docs/memory/reversibility-classifier.md, docs/memory/bespoke-installer-forbidden.md
  - "../../docs/proposals/v-rpc-tap-scalable.md" (the driving high-risk consumer — the XWBPRS splice)
  - "../../docs/proposals/considering/v-rpc-tap-scalable-deep-technical-analysis.md" (the analysis that surfaced the gap — N-5 + the uninstall finding)
---

# v-pkg: mixed-build split reversal + declared-foreign overwrites

> **One-line problem.** v-pkg's class-aware uninstall (`patch-existing-routines-proposal.md`,
> implemented) handles a build that is *either* greenfield-add *or* pure-overwrite. The
> high-risk RPC-broker splice is **both** — it overwrites the foreign national routine
> `XWBPRS` **and** adds the greenfield `VSLRTAP`/`VSLRTRP`/`VSLRTH`. For that **mixed** shape
> there is **no single back-out action that is correct**, and the default (`actDelete`)
> **deletes the live national `XWBPRS`**. This proposal closes that gap *inside* the v-pkg
> lifecycle — it does **not** add a bespoke installer ([[bespoke-installer-forbidden]]).

## Why not a dedicated installer (the question this answers)

The driving consumer is a deliberately high-risk operation: splice two `D` calls into
`XWBPRS`, a national, Directive-6402, checksum-audited Class-1 routine, on a live engine, and
be able to remove it byte-clean. The natural instinct is "such a dangerous change deserves its
own dedicated install/back-out tool." **Rejected**, for three standing reasons:

1. **Org directive** [[bespoke-installer-forbidden]]: install + back-out is *strictly and
   exclusively* `v pkg install` / `v pkg uninstall` of a drift-gated KIDS build. The previous
   `wrap-rpc` host-patcher was deleted for exactly this.
2. **Waterline rule 3 (transport monopoly):** the engine seam is exactly one artifact
   (`mdriver.Client` / the v-pkg lifecycle). A second installer that snapshots/patches/restores
   the engine is a second transport path → a red gate (and `engine-stack-guard.sh` denies it).
3. **A single, audited, drift-gated seam is what makes a dangerous change *safe*** — one
   snapshot store, one checksum gate, one back-out record, one CI-verified path. A parallel
   tool fragments that into a less-exercised code path: *more* risk on the highest-risk op.

The right kind of "dedication" is **package identity** (its own `VSL RPC TAP` build, its own
repo, out of any general v-stdlib build — already specified) plus a **hardened v-pkg
build-class** with stricter gates. That is this proposal.

---

## Problem — grounded in the real code

The splice build's component set is **mixed**:
- **overwrites** one foreign routine that already exists on the engine: `XWBPRS` (national);
- **adds** three greenfield routines in-namespace: `VSLRTAP`, `VSLRTRP`, `VSLRTH`.

### Failure 1 — the default uninstall DELETES the national routine
`internal/kids/reversibility.go:96 ClassifyBuild` derives class **statically from the `.KID`**
(side-effecting install code vs not). A routine-only splice build has no install code →
classified **`ClassPureOverwrite` (class 1)** (`reversibility.go:150-152`). The classifier
**cannot see that `XWBPRS` pre-existed** — it has no engine probe and no declared-foreign
signal. So at uninstall, `decideUninstall(class=1, restore="", …)` falls through to
**`actDelete`** (`pkgcli/lifecycle.go:684`), and `runUninstall` →
`installspec.UninstallScript` (`internal/installspec/script.go:146-153`) runs
`S X="XWBPRS" X ^%ZOSF("DEL")` over **every** build routine — **erasing the live national
`XWBPRS`**. The class-1 fallback only *warns* in its `reason` string ("if this patch
OVERWROTE existing routines, use --restore … delete would brick them"); it still proceeds.
Auto-detected pre-image (`resolveAutoRestore`) mitigates this *only if the `--auto-snapshot`
sidecar exists*; absent it, a plain `v pkg uninstall splice.kid` bricks `XWBPRS`.

### Failure 2 — the safe restore path ORPHANS the added routines
With `--restore <pre-image>` (or an auto-detected sidecar), `decideUninstall` →
**`actRestore`** (`lifecycle.go:672-673`), whose handler (`lifecycle.go:739-768`) only
re-installs the pre-image (restoring `XWBPRS` to stock) and **never deletes `VSLRTAP`/
`VSLRTRP`/`VSLRTH`**. They linger orphaned (harmless once stock `XWBPRS` no longer calls them,
but not a clean removal, and the `#9.7` accounting is left inconsistent). **There is no single
operation that restores the foreign overwrite AND deletes the greenfield adds.**

### Failure 3 — verify is line-2-blind, and the proposal it serves aimed it at the wrong line
`checkDrift` (`lifecycle.go:484-501`) compares via `kids.RoutineDriftMatch`
(`internal/kids/buildkids.go:124-138`), which canonicalizes line 2 through
`CanonicalizeRoutineLine2` (`internal/kids/decompose.go:56-64`) — keeping `;`-pieces 1–4,
**blanking pieces 5–6** (patch list / date / build number). `runVerify`
(`installspec/script.go:127-128`) is a presence (`$T`) probe only. So a "clean"/"applied"
verdict does **not** prove a foreign routine was restored byte-identical on line 2.

> **Correction from the v-rpc-tap deep analysis (N-5).** The consumer proposal called line 2
> "the checksum-audited line" and demanded restore "byte-identical including line 2" *because
> the national checksum hashes it*. **The corpus refutes that**: both VistA checksums exclude
> line 2 — `^%ZOSF("RSUM")` / `("RSUM1")` are documented *"the second line and comments are not
> included in the total"*; CHECK1 spec *"Line 2 is excluded from the count"* (Kernel
> OS-Interface DG / Toolkit DG). So a back-out with **command lines** restored byte-identical
> **passes** `[XU CHECKSUM REPORT]` (#9.8 gold) and `[XPD VERIFY INTEGRITY]` (#9.6) regardless
> of line 2 — and v-pkg's line-2-blind drift match is therefore **aligned** with the checksum.
> The real, separate concern is **patch-history / TRANSPORT BUILD NUMBER #63** (the 7th
> `;`-piece of line 2): a restore that leaves a wrong/blank patch list is invisible to today's
> verify. So the verify hardening below targets **command-line byte-equality (the checksum
> surface)** *and* **line-2 provenance pieces (patch history)** — two distinct claims, neither
> proven today.

This shape is **not tap-specific.** Any KIDS patch that ships helper routines alongside an
overwrite of an existing routine (common) hits Failures 1–2.

---

## Design — four additions to the existing lifecycle

All four stay inside the implemented `classify → snapshot → install → uninstall → verify`
machinery. Nothing reaches the engine outside `mdriver.Client`.

### D1 — Declared-foreign overwrites in the buildspec (closes F1 build-side)
Add an optional buildspec field:
```jsonc
"foreignRoutines": ["XWBPRS"]   // routines this build intentionally overwrites that are NOT in the package namespace
```
**Build-time guard** (extend `validateRoutines`, `internal/buildspec/buildspec.go`): every
routine whose name does **not** start with the package namespace token (`s.Package`, validated
at `buildspec.go:293`) **must** appear in `foreignRoutines`, or the build **fails**. This is
the F1 fix — today `isRoutineName` validates name syntax/length only and a build may list
`XWBPRS` with zero signal. The declaration makes the one intentional national overwrite
**explicit, reviewable, and machine-checked** (exactly one declared foreign routine for the
tap), and it is the offline signal the classifier and uninstaller need.

### D2 — A "mixed" reversibility shape (closes F1 classify-side, offline-decidable)
Extend `ClassifyBuild` (`reversibility.go`) to record, per build, the partition the install
path already computes at runtime (`captured` vs `greenfield` in `liveInstall`,
`lifecycle.go:317-325`) but currently discards at uninstall:
- `OverwriteForeign` = build routines listed in `foreignRoutines` (D1) — reversal = **restore
  pre-image** (delete would brick).
- `GreenfieldAdd` = the rest — reversal = **delete**.

A build with **both** non-empty is the **mixed** shape. This stays statically decidable from
the `.KID` + the `foreignRoutines` declaration — no engine probe required to *classify*
(engine probe is still used to *capture* the pre-image at install, as today). `CompleteUndo`
(snapshot.go:187) gains a mixed case: complete iff every `OverwriteForeign` routine has a
captured pre-image **and** every `GreenfieldAdd` is deletable (no non-routine components).

### D3 — `actSplitBackout`: the missing single operation (closes F2)
Add an uninstall action alongside `actDelete`/`actRestore`/`actBackout`/`actRefuse`
(`lifecycle.go:633-645`):

> **`actSplitBackout`** — for a mixed build: **(1) restore** every `OverwriteForeign`
> pre-image (reuse `runInstall`, as `actRestore` does), **then (2) delete** only the
> `GreenfieldAdd` routines (a `runUninstall` over the *greenfield subset*, NOT
> `b.RoutineNames()`). **Order is fixed: restore-foreign FIRST, delete-greenfield AFTER** — so
> the live `XWBPRS` is back to stock (and no longer references `VSLRT*`) before `VSLRT*` are
> removed; there is never a window where a spliced `XWBPRS` calls a deleted routine.

`decideUninstall` routing changes:
- **mixed build + pre-image available** (explicit `--restore` or auto-detected sidecar) →
  `actSplitBackout`.
- **mixed build + NO pre-image** → **`actRefuse`** (exit `ExitRefused`), never `actDelete`.
  This is the core safety fix: a build that overwrites a foreign routine can **never** fall
  through to delete-all. (`--force` may still allow `actDelete` of the greenfield subset only,
  with a loud "the foreign overwrite is NOT reversed" — it must **exclude** declared-foreign
  routines from the delete set even under `--force`.)
- pure greenfield / pure side-effecting paths are unchanged (backward compatible).

`UninstallScript` (`installspec/script.go:146`) is parameterized to delete a **subset** of
routines (it already takes a `routines []string` — pass the greenfield subset), so no new M
emitter is needed.

### D4 — National-overwrite verify mode (closes F3, N-5-correct)
For declared-foreign routines, add a stricter post-restore verification than the line-2-blind
`RoutineDriftMatch`:
- **(a) command-line byte-equality** — compare every line *except* line 2 byte-for-byte (the
  checksum surface; this is what makes `CHECK1^XTSUMBLD` match the FORUM #9.8 gold). Stronger
  than `RoutineDriftMatch`, which also blanks comment-text positions.
- **(b) line-2 provenance** — compare line-2 `;`-pieces (incl. the #63 build-stamp / patch
  list, pieces 5–7) so a restore that leaves a wrong/blank patch history is caught (the real
  line-2 concern, per N-5).
- **(c) optional engine checksum bracket** — capture `CHECK1^XTSUMBLD` for the foreign routine
  **pre-install** (== gold) and **post-uninstall**, and assert equality, as a belt-and-braces
  national-audit rehearsal (driver-stack call, off the hot path).

Surface as `v pkg uninstall --verify` reporting `verifyClean` plus a new
`foreignRestore: "exact" | "command-clean-provenance-drift" | "drift"` field.

### D5 (optional, generic) — pre-uninstall guard hook
The tap must be **disarmed** (mode flag cleared, reaper stopped) before its routines are
touched. Rather than tap-specific code, let a build declare a **pre-uninstall guard** entry
point (a routine label the lifecycle `D`-calls before any restore/delete, which must return
"safe to proceed"). Generic, reusable by any package with a runtime-arming safety state. If
out of scope for v1, the tap's host `disarm` is the operator precondition — documented, not
enforced.

---

## What does NOT change
- **Install is already safe.** `decideInstall` (`lifecycle.go:284-295`) already refuses a
  silent clobber of an existing routine unless `--snapshot`/`--auto-snapshot` (safe, enables
  restore) or `--allow-overwrite` (explicit unsafe). The splice install just uses
  `--auto-snapshot`. No change beyond the D1 build-time foreign declaration + an explicit
  foreign-load-LAST ordering note (today `V`<`X` collation is emergent — `EN^XPDIJ`,
  `installspec/script.go:108`; document the assumption or add an ordering hint).
- **The transport seam** stays `mdriver.Client`; **no bespoke installer.**

---

## Phasing
- **P1 — D1 + D2 (offline, no engine):** `foreignRoutines` schema + build-time guard;
  `ClassifyBuild` mixed shape + `OverwriteForeign`/`GreenfieldAdd` partition; `CompleteUndo`
  mixed case. Unit-test the classifier + guard (table-driven, like `TestDecideUninstall`).
- **P2 — D3:** `actSplitBackout` + `decideUninstall` routing (mixed+pre-image → split;
  mixed+no-pre-image → refuse; `--force` excludes foreign from delete). `UninstallScript`
  subset delete. Decision logic unit-tested offline (refuse path is engine-free, as today).
- **P3 — D4:** national-overwrite verify mode (command-byte + line-2 provenance + optional
  checksum bracket).
- **P4 — D5 (optional):** pre-uninstall guard hook.

## Acceptance — dual-engine, driver stack only (vehu YDB + foia IRIS)
The end-to-end gate, run through `m`/`mdriver` only (org engine-access rule — never raw
`docker exec`):
1. Build the splice `.KID` (overwrite `XWBPRS` + add `VSLRT*`), `foreignRoutines:["XWBPRS"]`.
2. `install --auto-snapshot` → `XWBPRS` patched, pre-image sidecar written, `VSLRT*` added.
3. `uninstall` (no flags) → routes to **`actSplitBackout`** (auto-detected sidecar): `XWBPRS`
   restored, `VSLRT*` deleted, ordered restore-first.
4. **Assert:** `XWBPRS` **command lines byte-identical to pre-install** (D4a) →
   `CHECK1^XTSUMBLD` == FORUM #9.8 gold; line-2 provenance restored (D4b); `VSLRT*` absent;
   `#9.7` consistent. On **both** engines.
5. **Negative:** `uninstall` of the same build with the sidecar **removed** → **REFUSED**
   (exit `ExitRefused`), engine untouched — proving the brick path is closed.
6. Build-time guard: a build listing `XWBPRS` **without** the foreign declaration → **build
   fails** (F1 closed).

## Why this is the right scope
This generalizes the same way `patch-existing-routines-proposal.md` did: that proposal turned a
per-package hand-hack (the `VSLTAPBO` back-out routine) into a first-class v-pkg capability.
This turns the next per-package hazard (a mixed overwrite-foreign + add-greenfield build that
today bricks or orphans on uninstall) into a first-class, gated capability — **the only
sanctioned way to install and cleanly reverse the high-risk splice, with no bespoke installer
and one audited seam.**
