# Kickoff prompt — verifiable-safety hardening (start at increment #1)

Paste the block below into a **fresh session started in `~/vista-cloud-dev/v-pkg`**
(one session ↔ one repo). Canon for the effort is the live tracker
[`docs/verifiable-safety-tracker.md`](../../../verifiable-safety-tracker.md); this
prompt just bootstraps a cold session into increment #1. Archive this folder with
the tracker when the effort lands.

---

```
You're continuing the "verifiable-safety hardening" effort on v-pkg — making `v pkg`
the most accurate, reliable, and verifiably-safe KIDS installer. Work in
~/vista-cloud-dev/v-pkg (this repo only).

READ FIRST (canon, do not re-derive):
- docs/verifiable-safety-tracker.md         ← the 4-increment plan + live-verified
                                              per-type global-root table. THIS GOVERNS.
- docs/design/kids-convention-vs-v-pkg.md    ← how v-pkg compares to the VA convention
- ../../docs/kids-classifee.md (docs repo)   ← the convention measured over 12,955
                                              vehu builds; its component table = the
                                              coverage checklist for increment #1
- internal/kids/reversibility.go + docs/memory/reversibility-classifier.md
- internal/kids/entrycomp.go, internal/installspec/script.go, pkgcli/lifecycle.go

GOAL: execute the tracker's increments IN ORDER, one verified increment at a time.
Start with #1 (component-type coverage 11 → all standard FileMan types). Do NOT
start #2 until #1 is committed and its tracker row is marked done.

INCREMENT #1 — the gap and the design:
- `install` is already complete (native KRN^XPDIK files every component type). The
  gap is `verify` (presence VerifyScript + content VerifyContentScript) and
  `uninstall` (UninstallScript DIK-delete), which hardcode only 11 types:
  19, 19.1, 101, 8994, 8989.51, 3.8, 409.61, 9.2, 771, 779.2, 870.
- So a build shipping INPUT TEMPLATE (#.402 — 779 builds, 3rd most common), PRINT
  TEMPLATE (#.4), SORT TEMPLATE (#.401), FORM (#.403), DIALOG (#.84), BULLETIN
  (#3.6), PARAMETER TEMPLATE (#8989.52), ENTITY (#1.5), HL LOWER-LEVEL (#869.2),
  XULM LOCK DICT (#8993) installs fine but uninstall ORPHANS it and verify can't
  assert it.
- Design: drive presence-verify + uninstall-delete generically from the
  `entryTypeByFile` registry (internal/kids/entrycomp.go) — one row per type — using
  each file's authoritative storage global. `(b *Build) entryNames(file)` already
  extracts names for ANY file#; `EntryContents()` already does generic content via
  `entryType.dataRoot`. The global roots are in the tracker's table (read live from
  `^DIC(file,0,"GL")`); re-confirm against vehu before relying on any.
- EXCLUDE #.5 FUNCTION (^DD("FUNC",…)) and #9002226 lexicon (no GL, multi-file) —
  they are NOT standard `^DIK` targets; refuse/warn, never mis-delete.

NON-NEGOTIABLE VALIDATION (why this isn't a casual edit):
- content-verify `volatile` masks per new type MUST be ground-truthed on a LIVE
  engine (wrong mask → false drift). Determine each type's FileMan-rewritten 0-node
  pieces by installing a fixture and diffing shipped vs filed, exactly as the 11
  existing masks were.
- `^DIK` deletion safety per type MUST be confirmed live (templates carry
  compiled-input/print subfiles; forms carry blocks).
- New testdata fixtures (e.g. testdata/zztmpl with an INPUT TEMPLATE).

ENGINE ACCESS — driver stack ONLY (org hard rule; raw `docker exec` is gate-denied):
- ad-hoc M probe: `M_YDB_CONTAINER=vehu m vista exec --engine ydb --transport docker '<one M cmd>'`
  Gotchas: set `S U="^"` (no Kernel context); wrap argumentless-FOR loops inside
  XECUTE (the bare FOR swallows the trailing WRITE); walk IENs with a bounded
  `F I=1:1:N`. Decode the JSON `data.stdout` (a small python `json.load` wrapper helps).
- suites/coverage: `m test`; live lifecycle gates: `make live-gate`, `make stress`
  (ENGINE=ydb vehu default; ENGINE=iris needs M_IRIS_* — creds are direnv-loaded).
- corpus round-trip + classifier shares: `CORPUS=~/data/kids-patches/VistA make corpus`.

PROCESS (org Increment Protocol — see ~/vista-cloud-dev/CLAUDE.md + v-pkg/CLAUDE.md):
- TDD is a HARD RULE: write the failing test first (offline-assert the generated M /
  the registry recognition), confirm RED, implement, confirm GREEN.
- Gates BEFORE commit: `make lint test` (CGO_ENABLED=1, -race), `make corpus`
  (DRIFT=0), and `make live-gate` / `make stress` for the engine-bound behavior.
- Commit straight to main (trunk-based, solo org); stage only files you touched;
  use the `Co-Authored-By: Claude …` trailer; never --no-verify / force-push main.
- At the close of the increment: persist the durable lesson to docs/memory/
  (update reversibility-classifier.md or a sibling, not a new status file), mark the
  #1 row done in docs/verifiable-safety-tracker.md, commit, push.

CONTEXT — already done (don't redo): the reversibility classifier's KRN probe was
fixed (commit 55adf9c — now also probes per-build "BLD",<n>,"KRN",<file>,"NM"
excluding #9.8); the tracker + reports are committed. Your job is increments #1→#4.

Begin with increment #1. Re-confirm the global-root table against vehu, then write
the first failing test.
```
