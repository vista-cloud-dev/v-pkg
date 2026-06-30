# Verifiable-safety hardening — tracker

Live tracker (Tier D) for the four-part effort to make `v pkg` the most accurate,
reliable, and *verifiably* safe KIDS installer. Motivated by the corpus reports
[`docs/kids-classifee.md`](../../docs/kids-classifee.md) (the VA convention measured
over 12,955 installed builds) and
[`docs/design/kids-convention-vs-v-pkg.md`](design/kids-convention-vs-v-pkg.md)
(how v-pkg compares). Sequenced by value / risk / dependency; each row is a
verified increment (TDD + gates + commit), most needing dual-engine (vehu/foia)
validation. Archive this doc to `docs/archive/` when all four land.

| # | Increment | Value | Status |
|---|---|---|---|
| 1 | Component-type coverage 11 → all standard FileMan types (verify + uninstall) | reliability — no orphan-on-uninstall, complete verify | **in progress** |
| 2 | Pre-install dry-run / compare-to-current | verifiable-in-advance | planned |
| 3 | Robustness: half-install heal · transport-checksum · sidecar integrity | reliability | planned |
| 4 | Install attestation / audit record | third-party-verifiable | planned |

---

## 1 — Component-type coverage (11 → all standard FileMan types)

**Gap.** `VerifyScript` (presence), `VerifyContentScript` (via `entryTypeByFile`),
and `UninstallScript` (DIK delete) cover only **11** component types
(`19, 19.1, 101, 8994, 8989.51, 3.8, 409.61, 9.2, 771, 779.2, 870`). Install is
already complete (native `KRN^XPDIK` files every type), so a build that ships an
**INPUT TEMPLATE (#.402 — 779 builds, the 3rd most common component in `vehu`)**,
PRINT TEMPLATE, FORM, DIALOG, BULLETIN, … installs fine but **uninstall silently
orphans it and verify can't assert it.** The `kids-classifee` component census is
the prioritized checklist; `(b *Build) entryNames(file)` already extracts any
type's names generically — only the per-type storage global is missing downstream.

**Design — registry-driven, DD-sourced.** Extend the `entryTypeByFile` registry
(content-verify) and drive presence-verify + uninstall-delete generically from a
`file# → {dataRoot, "B"-index, DIK ref}` table, rather than 11 hand-coded cases. A
new type becomes one table row. The authoritative storage global is the live DD's
`^DIC(file,0,"GL")` — captured below (vehu, 2026-06-30):

| File# | Component | Global root (`^DIC(f,0,"GL")`) | DIK-safe? | Notes |
|---|---|---|---|---|
| .402 | INPUT TEMPLATE | `^DIE(` | yes | 779 builds — top priority |
| .4 | PRINT TEMPLATE | `^DIPT(` | yes | 451 |
| .401 | SORT TEMPLATE | `^DIBT(` | yes | 221 |
| .403 | FORM | `^DIST(.403,` | yes (subfile-aware) | 177 |
| .84 | DIALOG | `^DI(.84,` | yes | 125 |
| 3.6 | BULLETIN | `^XMB(3.6,` | yes | 117 |
| 8989.52 | PARAMETER TEMPLATE | `^XTV(8989.52,` | yes | 31 |
| 1.5 | ENTITY | `^DDE(` | verify-live | 23 |
| 8993 | XULM LOCK DICTIONARY | `^XLM(8993,` | verify-live | rare |
| 869.2 | HL LOWER LEVEL PROTOCOL | `^HLCS(869.2,` | yes | 13 |
| **.5** | **FUNCTION** | `^DD("FUNC",` | **NO** | not a standard DIK file — exclude / special |
| **9002226** | lexicon family | *(no `GL`)* | **NO** | multi-file family — exclude / special |

**Validation required before merge (why this is not a casual edit):**
- **`volatile` masks** for content-verify must be ground-truthed per type on the
  live engine (a wrong mask → false drift). Determine each type's FileMan-rewritten
  0-node pieces by installing a fixture and diffing shipped vs filed (as the
  existing 11 were).
- **DIK safety** per type on a live engine — confirm `^DIK` by IEN cleanly removes
  the record + its xrefs/subfiles (templates carry compiled-input/print subfiles;
  forms carry blocks). Exclude `.5 FUNCTION` and `9002226 lexicon` (not standard
  DIK targets) — refuse-or-warn rather than mis-delete.
- Dual-engine `live-gate` + `stress` green; new fixtures (`testdata/zztmpl`, …).

**Files:** `internal/kids/entrycomp.go` (registry + per-type `entryType`),
`internal/installspec/script.go` (generic presence-verify + uninstall loops),
`pkgcli/lifecycle.go` (collect generic component list), new `testdata/` fixtures.

## 2 — Pre-install dry-run / compare-to-current

**Gap.** No verb previews a build's effect against the **live** system before
mutating it — the convention's *Verify Checksums in Transport Global* + *Compare
Transport Global to Current System* (KIDS menu 2–4). `classify` is static (no
engine); `verify` asserts a *completed* install. (`verify --drift` run pre-install
is a routines-only partial.)

**Design.** `v pkg install --dry-run` (and/or `v pkg diff <kid>`): over the driver,
for each shipped routine report **NEW / CHANGED (resident checksum ≠ incoming) /
IDENTICAL (no-op)**; for each DD/data/component report would-add / would-change /
identical — **without** populating `^XTMP`/running `EN^XPDIJ`. Reuses
`checkDrift`/`readRoutineBody` (already read routines off the engine) + the
transport's shipped `B`-checksums. Output is a structured, machine-checkable plan;
exit 0 always (it is read-only). Pairs naturally with #4 (the plan is the
pre-image of the attestation).

## 3 — Robustness hardening

- **Half-install heal.** A prior aborted install leaves a `#9.7` entry with
  `"ASP"/"INI"/"INIT"` xrefs but no `0`-node; `EN^XPDIJ` silently bails and the
  re-install guard falsely reports `already-installed`. Detect the corrupt entry
  (xrefs present, 0-node absent) and offer `--heal` (purge by IEN +
  `^XTMP("XPDI",ien)`) so a clean reinstall can proceed. (Gotcha documented in
  `docs/design/kids-installation-automation.md` §7.1.)
- **Transport checksum verify (pre-install tamper/corruption).** Before staging,
  recompute each shipped routine's checksum and compare to the build's stored
  `B`-checksum (the convention's *Verify Checksums in Transport Global*). Refuse a
  mismatch (corrupted/tampered `.KID`).
- **Snapshot/sidecar integrity.** Stamp `<kid>.preimage.kids` with a content hash
  (and verify it on `restore`/auto-restore) so a tampered pre-image cannot silently
  restore the wrong routine source.

## 4 — Install attestation / audit record

**Design.** Each engine-mutating op appends a structured, signed-or-hashed
attestation (op, build, timestamp, reversibility class, before/after routine
checksums, components touched, REQB chain, snapshot ref, verify verdict, exit
code). Append-only, independently auditable — turns "it installed" into provable
provenance. Aligns with the org `source-tag → registry → red-gate` discipline (the
attestation is the install's red-gate evidence). Best landed last so it records the
richer change-set from #1–#3.

---

*Reports that motivate this: [`docs/kids-classifee.md`](../../docs/kids-classifee.md),
[`docs/design/kids-convention-vs-v-pkg.md`](design/kids-convention-vs-v-pkg.md).
Each increment follows the org Increment Protocol (TDD → gates → memory → commit).*
