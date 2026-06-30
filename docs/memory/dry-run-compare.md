---
name: dry-run-compare
description: v-pkg `install --dry-run` / `diff <kid>` previews a build's effect against the LIVE engine read-only (NEW/CHANGED/identical), composed entirely from the existing checkDrift + verifyContent + "B"-presence probes — a true no-op, exit 0 always.
metadata:
  type: project
---

**Pre-install dry-run / compare-to-current (verifiable-safety increment #2, 2026-06-30)**
— closes the gap that no verb previewed a build's effect against the LIVE system
BEFORE mutating it (the KIDS convention's *Compare Transport Global to Current
System*, menu 2–4): `classify` is static, `verify` asserts a COMPLETED install.
Canon: `../archive/verifiable-safety-tracker.md` §2. Pairs with [[verify-drift]],
[[verify-content]], [[component-type-coverage]].

**Surface.** `v pkg install --dry-run` (the flag is the engine; reuses the
build-load + multi-build path) AND a thin `v pkg diff <kid>` alias (no install
flags). Both call `runDryRun` → `dryRunPlan` per build. JSON-first (`dryRunReport`
under `cc.Result`'s `.data`), one report per build. **Exit 0 ALWAYS** — read-only
preview, never a check failure.

**The whole thing is RELABELED EXISTING PROBES — no new engine reader was written**
(`pkgcli/dryrun.go`). Per build:
- routines ← `checkDrift` (already reads each routine via `readRoutinePreimage` +
  `RoutineDriftMatch`, line-2-blind): `absent→NEW`, `applied→identical`,
  `drifted→CHANGED` (`classifyRoutine`).
- content-verify components + FILE DD ← `verifyContent` (0-node compare via
  `ZeroMatch`): `absent→would-add`, `mismatch→would-change`, `ok→identical`
  (`classifyContent`).
- presence-only components (the increment-1 `contentVerify:false` types) ← the
  `VerifyScript` "B"-index probe (`runVerify`): present→`present`, absent→`would-add`.
  **They NEVER read `would-change`** — they have no validated content claim, so the
  plan honestly says only "exists" vs "would add" (the same verifiable-safety
  discipline as the contentVerify gate, see [[component-type-coverage]]).
The pure assembly (`assembleDryRun`) + the four-bucket `dryRunSummary`
(new/changed/identical/present) are unit-tested offline from canned verdict maps.

**KEY PROPERTY — it is a TRUE no-op (the non-negotiable).** The plan uses ONLY the
read-only probes, so it NEVER stages `^XTMP("VPKGI"/"XPDI")` and NEVER reaches
`EN^XPDIJ`. Live-proven on vehu: engine state (`$D(^XPD(9.7,"B",name))`, sample
routine `$T`, `$D(^XTMP("VPKGI"))`) is **byte-identical before/after** a `diff` run,
exit 0. So a dry-run can never accidentally install.

**Prediction fidelity — live-proven on vehu (MSL, 40 routines):** clean engine →
all-NEW (40); after a real install → all-identical (40); after editing exactly one
shipped routine on the engine → that one reads CHANGED, the other 39 identical.
Wired into `make live-gate` as a pre-flight (`diffck`): clean→all-new + installed→
all-identical for BOTH MSL and VSL (live-gate now 14/14; stress 37/37 unchanged).

**Why/how to apply:** run `v pkg diff <kid>` (or `install --dry-run`) to see what an
install would change before committing — the machine-checkable plan is also the
pre-image of the install attestation (increment #4). When adding a component type,
its plan verdict is automatic (registry-driven) — content-verify types get the
0-node compare, presence-only types the present/would-add fallback. Adding a verb +
flag changed the command surface → regenerate the contract golden (`make contract`).
