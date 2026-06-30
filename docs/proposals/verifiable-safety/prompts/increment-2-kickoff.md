# Kickoff prompt — verifiable-safety hardening (increment #2)

Paste the block below into a **fresh session started in `~/vista-cloud-dev/v-pkg`**
(one session ↔ one repo). Canon for the effort is the live tracker
[`docs/verifiable-safety-tracker.md`](../../../verifiable-safety-tracker.md); this
prompt bootstraps a cold session into increment #2. Archive this folder with the
tracker when the effort lands.

---

```
You're continuing the "verifiable-safety hardening" effort on v-pkg — making `v pkg`
the most accurate, reliable, and verifiably-safe KIDS installer. Work in
~/vista-cloud-dev/v-pkg (this repo only).

READ FIRST (canon, do not re-derive):
- docs/verifiable-safety-tracker.md            ← the 4-increment plan. §2 is THIS
                                                 increment. THIS GOVERNS.
- docs/memory/component-type-coverage.md       ← increment #1's durable lesson: the
                                                 registry-driven Components() path +
                                                 the contentVerify gate you'll reuse
- docs/memory/verify-drift.md + verify-content.md ← the two live-read paths you extend
- internal/installspec/script.go (VerifyContentScript), pkgcli/lifecycle.go
  (checkDrift, readRoutinePreimage, verifyContent, runVerify, the install path)
- internal/kids/entrycomp.go (Components, EntryContents, ZeroMatch), drift.go
  (RoutineDriftMatch / RoutineSource)

GOAL: execute the tracker's increments IN ORDER. Increment #1 (component-type
coverage) is DONE + on main (commits 683eace + de1c24b). Do increment #2 next; do
NOT start #3 until #2 is committed and its tracker row is marked done.

INCREMENT #2 — pre-install dry-run / compare-to-current:
- The gap: no verb previews a build's effect against the LIVE system BEFORE mutating
  it — the KIDS convention's "Verify Checksums in Transport Global" + "Compare
  Transport Global to Current System" (KIDS menu options 2–4). `classify` is static
  (no engine); `verify` asserts a COMPLETED install; `verify --drift` pre-install is
  a routines-only partial.
- Design (tracker §2): `v pkg install --dry-run` and/or a `v pkg diff <kid>` verb.
  Over the driver, READ-ONLY (never populate ^XTMP, never run EN^XPDIJ), report:
    * per shipped ROUTINE: NEW (absent on engine) / CHANGED (resident ≠ incoming) /
      IDENTICAL (no-op).
    * per COMPONENT / DD / data: would-add (absent) / would-change (present, differs)
      / identical.
  Output is a structured, machine-checkable plan; exit 0 ALWAYS (it is read-only).
  Pairs with #4 — the plan is the pre-image of the attestation.

REUSE — do not build these from scratch (they already read the live engine):
- Routines: `checkDrift` already returns absent/applied/drifted by comparing each
  shipped routine to the resident one (RoutineDriftMatch, line-2-canonicalized so a
  checksum rewrite isn't false drift). Dry-run is its PRE-install relabeling:
  absent→NEW, applied→IDENTICAL, drifted→CHANGED. Reuse readRoutinePreimage +
  RoutineSource; do NOT re-derive a checksum reader (source-compare is the proven
  path; the "B-checksum" is the convention being emulated, not a new dependency).
- Components/DD: `verifyContent` + `EntryContents`/`FileContents` + `ZeroMatch`
  already read the live 0-node / ^DD def and grade ok/mismatch/absent. Dry-run is the
  same probe run BEFORE install: absent→would-add, mismatch→would-change, ok→identical.
  Presence-only (contentVerify:false) types fall back to the `Components()` "B"-index
  probe (would-add if absent, else present) — they have no validated content claim
  (see component-type-coverage.md), so do NOT assert would-change for them.
- The plan struct should be JSON-first (cc.Result), like verifyResult.

DESIGN CALL to make early (then proceed): `--dry-run` flag on `install` (reuses the
build-load + multi-build + class logic) vs a standalone `v pkg diff <kid>` verb
(leaner, no install flags). Recommend: the flag as the engine, a thin `diff` alias if
it earns its keep. Keep the plan generation in one place either way.

NON-NEGOTIABLE VALIDATION (read-only, so cheaper than #1, but still live):
- Dry-run must be a TRUE no-op: assert it stages no ^XTMP("VPKGI"/"XPDI") and never
  reaches EN^XPDIJ. Prove on vehu that running --dry-run leaves #9.7 / the routines /
  the components UNCHANGED (diff the engine state before/after).
- Prediction fidelity: on a clean engine a greenfield build reads ALL-NEW; after a
  real install the same build reads ALL-IDENTICAL; after editing one shipped routine
  it reads CHANGED for exactly that routine. Wire this into make live-gate / make
  stress (the dry-run is the natural pre-flight before each install they already do).

ENGINE ACCESS — driver stack ONLY (org hard rule; raw `docker exec` is gate-denied):
- ad-hoc M probe: `M_YDB_CONTAINER=vehu m vista exec --engine ydb --transport docker '<one M cmd>'`
  Gotchas: set `S U="^"` (no Kernel context); wrap argumentless-FOR loops inside
  XECUTE; walk IENs with a bounded `F I=1:1:N`; the `^DIC(f,0,"GL")` GL node starts
  with "^" so never $P it on U. Decode the JSON `data.stdout` with a small python
  json.load wrapper.
- suites/coverage: `m test`; live: `make live-gate`, `make stress` (ENGINE=ydb vehu
  default; ENGINE=iris needs M_IRIS_* — direnv-loaded); corpus: `make corpus` DRIFT=0.

PROCESS (org Increment Protocol — see ~/vista-cloud-dev/CLAUDE.md + v-pkg/CLAUDE.md):
- TDD is a HARD RULE: write the failing test first (offline-assert the generated
  read-only M / the plan classification: NEW/CHANGED/IDENTICAL from canned markers),
  confirm RED, implement, confirm GREEN.
- Gates BEFORE commit: `make lint test` (CGO_ENABLED=1, -race), `make corpus`
  (DRIFT=0), and `make live-gate` / `make stress` for the engine-bound behavior.
- Commit straight to main (trunk-based, solo org); stage only files you touched; use
  the `Co-Authored-By: Claude …` trailer; never --no-verify / force-push main.
- At the close: persist the durable lesson to docs/memory/ (a sibling — e.g.
  dry-run-compare.md — or extend verify-drift.md; not a new status file), mark the #2
  row done in docs/verifiable-safety-tracker.md, commit, push.

CONTEXT — already done (don't redo): increment #1 genericized verify/uninstall over
the kids.Component registry (a new type is one entryTypeByFile row) and added 8
presence/uninstall-only types behind the contentVerify gate; DIK safety live-proven
net-zero on vehu. Your job is increments #2 → #4.

Begin with increment #2. Decide the flag-vs-verb shape, then write the first failing
test for the read-only plan classification.
```
