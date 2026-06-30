# Kickoff prompt — verifiable-safety hardening (increment #3)

Paste the block below into a **fresh session started in `~/vista-cloud-dev/v-pkg`**
(one session ↔ one repo). Canon for the effort is the live tracker
[`docs/verifiable-safety-tracker.md`](../../../verifiable-safety-tracker.md); this
prompt bootstraps a cold session into increment #3. Archive this folder with the
tracker when the effort lands.

---

```
You're continuing the "verifiable-safety hardening" effort on v-pkg — making `v pkg`
the most accurate, reliable, and verifiably-safe KIDS installer. Work in
~/vista-cloud-dev/v-pkg (this repo only).

READ FIRST (canon, do not re-derive):
- docs/verifiable-safety-tracker.md            ← the 4-increment plan. §3 is THIS
                                                 increment. THIS GOVERNS.
- docs/memory/dry-run-compare.md               ← increment #2's durable lesson (the
                                                 read-only diff/plan you may reuse)
- docs/memory/class-aware-install.md + snapshot-restore.md + pairing-verify-clean.md
                                                 ← the install + snapshot/sidecar paths #3 hardens
- docs/memory/verify-drift.md                  ← the routine-source/line-2 checksum model
- docs/design/kids-installation-automation.md §7.1 ← the half-install corrupt-#9.7 gotcha
- pkgcli/lifecycle.go (runInstall, decideInstall, liveInstall, the install path),
  internal/installspec/script.go (FinalInstallScript — the already-installed guard
  at the top), pkgcli/snapshot.go + pkgcli/pairing.go (defaultPreimagePath,
  resolveAutoRestore, fileExists, buildSnapshotPairs), internal/kids/assemble.go
  (WriteKID), internal/kids/buildkids.go (RTN node value, RoutineDriftMatch)

GOAL: execute the tracker's increments IN ORDER. Increments #1 (component-type
coverage) and #2 (pre-install dry-run / `diff`) are DONE + on main (latest 5a18418).
Do increment #3 next; do NOT start #4 until #3 is committed and its tracker row is
marked done. #3 is three SEPARATE robustness fixes — land each as its own verified
increment (TDD → gates → commit), not one mega-commit.

INCREMENT #3 — robustness hardening (three sub-features, sequence by risk):

(3a) HALF-INSTALL HEAL. A prior aborted install leaves a #9.7 entry with the "B"
  xref (and "ASP"/"INI"/"INIT" subnodes) but NO usable 0-node / status; EN^XPDIJ
  silently bailed. The re-install guard (FinalInstallScript:
  `I $D(^XPD(9.7,"B",name))` → error=already-installed) then FALSELY reports
  already-installed, so a clean reinstall is impossible. FIX: detect the corrupt
  entry (xref present, 0-node absent / status not 3) and offer `v pkg install --heal`
  (purge by IEN: `^XPD(9.7,ien)` + `^XPD(9.7,"B",name,ien)` + `^XTMP("XPDI",ien)`)
  so a clean reinstall proceeds. Heal is a TARGETED purge of a PROVEN-corrupt entry
  only — never a blanket force-delete of a healthy install (that is uninstall's job).

(3b) TRANSPORT-CHECKSUM VERIFY (pre-install tamper/corruption). The convention's
  "Verify Checksums in Transport Global": before staging, recompute each shipped
  routine's checksum and compare to the checksum the .KID itself carries in its RTN
  node, refusing a mismatch (a corrupted/tampered .KID). GOTCHA you must handle:
  v-pkg's OWN builds STRIP routine checksums to 0 in the normalized artifact
  (buildkids.go ~L82: `0^<numlines>^<checksum>^<checksum>`, checksums 0 — real ones
  computed at install). So this check is meaningful for INGESTED/foreign .KIDs
  (the corpus patches) that carry real checksums; a 0/0 v-pkg build has nothing to
  verify (skip, don't false-fail). Compute the checksum the way VistA does
  (CHECK1^XTSUMBLD / the RSUM line-2-blind surface — see verify-drift.md) so a real
  patch's stored checksum matches; this is an OFFLINE check over the parsed .KID
  (no engine needed), which makes it cheap and a natural pre-stage gate in runInstall.

(3c) SNAPSHOT/SIDECAR INTEGRITY. install --auto-snapshot writes the pre-image
  sidecar <kid>.preimage.kids that uninstall auto-restores (pairing.go). A tampered
  sidecar would silently restore the WRONG routine source. FIX: stamp the sidecar
  with a content hash at capture (a private node, like the ("VPKG",…) foreign
  declaration that kids.EnginePairs strips — so it never reaches the engine) and
  VERIFY it on restore / auto-restore; refuse a hash mismatch. Reuse the EnginePairs
  metadata-node pattern so the hash rides in the .KID without polluting KIDS filing.

REUSE — do not build these from scratch:
- The already-installed guard + #9.7 read shapes are in FinalInstallScript /
  VerifyScript; heal reads the same #9.7 nodes (status piece 9, the 0-node, the "B"
  xref) READ-ONLY first to PROVE corruption before purging.
- The diff/plan from #2 (dryRunPlan) already reads resident routine + component
  state read-only — the heal-detect and a "this build is half-installed" signal can
  surface there too if it earns its keep (don't force it).
- Snapshot write/read = buildSnapshotPairs + kids.WriteKID + loadBuild; the sidecar
  path + auto-detect = pairing.go. Hash the pre-image PAIRS (deterministic) so the
  same capture always stamps the same hash.
- The metadata-ride-along pattern is kids.EnginePairs + ForeignRoutines (a private
  ("VPKG",…) node read offline, stripped before staging) — copy it for the hash.

DESIGN CALLs to make early (then proceed):
- Heal as a FLAG on install (`--heal`, reuses the build-load + connection) vs a
  standalone verb. Recommend the flag (heal is "make this install possible"), but a
  read-only `v pkg doctor <kid>`-style detector may be the honest first step.
- Transport-checksum: a refuse-by-default pre-stage gate in the install path vs an
  opt-in `--verify-checksums`. Recommend refuse-by-default for a .KID that CARRIES
  checksums (silent on 0/0 v-pkg builds), with an escape hatch to override.

NON-NEGOTIABLE VALIDATION (each sub-feature, live where it touches the engine):
- 3a: reproduce a corrupt #9.7 on vehu (seed the "B" xref + subnodes, no 0-node),
  prove the normal install REFUSES already-installed, then `--heal` purges it and a
  clean reinstall reaches status 3. Prove heal NEVER touches a healthy install.
- 3b: a pristine corpus .KID passes; a byte-tampered copy (flip one routine line)
  is REFUSED pre-stage with no engine write. Offline unit test from canned RTN
  checksum nodes (pristine vs tampered) — TDD red first.
- 3c: a pristine sidecar restores; a tampered sidecar (edit one routine line) is
  REFUSED on restore/auto-restore. Offline unit test of the hash stamp/verify.
- Wire the new gates into make live-gate / make stress; keep make corpus DRIFT=0.

ENGINE ACCESS — driver stack ONLY (org hard rule; raw `docker exec` is gate-denied):
- ad-hoc M probe: `M_YDB_CONTAINER=vehu m vista exec --engine ydb --transport docker '<one M cmd>'`
  Gotchas: set `S U="^"` (no Kernel context); wrap argumentless-FOR loops inside
  XECUTE; walk IENs with a bounded `F I=1:1:N`. Decode JSON data.stdout with python.
- live-gate env (vehu): export M_YDB_BIN/_CONTAINER=vehu/_TRANSPORT=docker/_DIST=
  /home/vehu/lib/gtm /_GBLDIR=/home/vehu/g/vehu.gld /_ROUTINES=/home/vehu/r.
- suites/coverage: `m test`; live: `make live-gate`, `make stress`; corpus:
  `make corpus` DRIFT=0.

PROCESS (org Increment Protocol — see ~/vista-cloud-dev/CLAUDE.md + v-pkg/CLAUDE.md):
- TDD is a HARD RULE: write the failing test first (offline-assert the heal-detect
  classification, the checksum compare, the sidecar hash verify from canned inputs),
  confirm RED, implement, confirm GREEN.
- Gates BEFORE commit: `make lint test` (CGO_ENABLED=1, -race), `make corpus`
  (DRIFT=0), `make live-gate` / `make stress` for engine-bound behavior, and
  `make contract` if you add a verb/flag (the command surface is drift-gated).
- Commit straight to main (trunk-based, solo org); stage only files you touched; use
  the `Co-Authored-By: Claude …` trailer; never --no-verify / force-push main. Land
  3a, 3b, 3c as separate gate-green commits.
- At the close: persist the durable lesson to docs/memory/ (siblings — e.g.
  half-install-heal.md / transport-checksum.md / sidecar-integrity.md, or extend an
  existing file; not a new status file), mark the #3 row done in
  docs/verifiable-safety-tracker.md, commit, push.

CONTEXT — already done (don't redo): #1 genericized verify/uninstall over the
kids.Component registry (+8 presence/uninstall-only types, DIK net-zero on vehu); #2
added `install --dry-run` + `v pkg diff` (read-only NEW/CHANGED/identical plan from
relabeled checkDrift/verifyContent/presence probes — a true no-op proven on vehu,
exit 0 always; wired into live-gate, 14/14). Your job is increments #3 → #4.

Begin with increment #3a (half-install heal). Decide flag-vs-verb, then write the
first failing test for the corrupt-#9.7 detection.
```
