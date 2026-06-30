# v-pkg repo consistency audit — 2026-06-29

A comprehensive pass over **every** doc, folder, and file in `v-pkg` for
consistency, completeness, correctness, and relevance — flagging each stale /
superseded / misfiled / inaccurate item for a specific action. Run as a
documentation-and-hygiene increment (the `m-kids → v pkg` rename + the
lifecycle-landing left some residue to reconcile).

**Method:** deterministic link/grep checks (rename leftovers, deleted-feature
refs, broken relative `.md` links) + three parallel deep-read auditors over
(A) `docs/design` + `docs/proposals`, (B) `docs/memory` + `docs/archive`,
(C) root files + `examples` + `scripts` + `testdata` + Go code. `go vet ./...`
clean; offline gates (lint / build / test / contract) green.

**Tier:** D (live status). Archive this file once the flagged actions land.

---

## Headline

The repo is in good shape — **no stale `wrap-rpc`/`VSLRPCWRAP` feature claims in
live docs or code** (deleted feature is correctly confined to `docs/memory` +
`docs/archive` tombstones), **no `m-kids`-as-tool-name leaks** (every live doc
says "v pkg"; the only `m-kids` hits are frozen archive history and the
`py-kids-vc` predecessor + `krn_8_0_*_kids_ug` filenames, all legitimate). The
findings are reconciliation residue, not rot: a few completed proposals still
flagged `status: proposed`, some broken relative links from today's archive
move, one wrong license tag, and three orphan test fixtures.

---

## A. Fixed in this increment (safe, non-structural)

| # | File | Fix |
|---|---|---|
| A1 | `repo.meta.json` | `license` `AGPL-3.0` → **`Apache-2.0`** (LICENSE file + README both say Apache-2.0; the meta tag was the lone outlier — a real license-declaration inaccuracy). |
| A2 | `repo.meta.json` | `verification_commands` `./dist/m arch check .` → **`m arch check .`** (this repo's `dist/` only builds `v-pkg`; `./dist/m` never exists locally — the arch gate runs from PATH / CI). |
| A3 | `docs/memory/MEMORY.md` | Removed the **duplicate** `package-footprint.md` index line (the file was indexed twice — register-package angle + deregister angle; kept the combined line). |
| A4 | `docs/README.md` | Added the missing **`v-pkg-from-engine-capture.md`** proposal to the index; tagged the install-fidelity spike "(landed; pending archive)". |
| A5 | `docs/proposals/v-pkg-kids-coverage-analysis.md` | Fixed 4 broken `related:` frontmatter paths (`../design/…`, `../archive/…`) — they pointed at bare filenames that no longer resolve after the design/archive moves. |
| A6 | `docs/design/architecture.md` | Corpus count `≈2,406` → **`≈2,404`** (matched corpus-findings + coverage-analysis; numeric drift). |
| A7 | `pkgcli/commands.go` | Package doc-comment said the live lifecycle "(build/install/verify/uninstall/**status**) lands in M0a's later tasks" — it has **landed**, and there is no `status` verb. Updated to the real surface (… / snapshot / restore) over the m-driver-sdk seam. |

---

## B. Flagged for decision — structural moves (NOT executed; need a go-ahead)

These change file location or delete content; per the house caution on doc moves
they want an explicit OK rather than a silent rewrite.

### B1 — Archive the completed install-fidelity spike — ✅ DONE 2026-06-29
`git mv`'d `v-pkg-install-fidelity-spike.md` → `docs/archive/` (status →
`completed`); repointed all inbound links (`README.md`, `from-engine-capture`,
`coverage-analysis`, `implementation-plan`, the two memory refs + the zza1
fixture README). Original recommendation:
`docs/proposals/v-pkg-install-fidelity-spike.md` is front-matter
`status: proposed` but its body has **A.1.1 / A.1.2 / A.1.3 / B.3 all "DONE +
live-proven both engines"** and closes "the A.1 install-fidelity track … are
complete." It is a finished Tier-D effort sitting in `proposals/`, which the org
convention says must `git mv` to `docs/archive/` on landing.
- **Action:** `git mv` → `docs/archive/`, set `status: completed`, repoint the 2
  inbound links (`v-pkg-kids-coverage-analysis.md`, `v-pkg-from-engine-capture.md`
  link to it as a same-dir sibling → `../archive/…`) and the `docs/README.md`
  line.

### B2 — Resolve the implementation-plan archive — ✅ RESOLVED (stays archived)
**Decision 2026-06-29: leave it archived; nothing is orphaned.** P6 is an
*illusory* gap (the "missing" KIDS Developer Tools UG is a section of
`krn_8_0_dg_kids_ug`, already in the corpus). P7 (engine parity) is substantially
**proven** by the both-engine live gates (install-fidelity + live-package-gate on
vehu YDB + foia-t12 IRIS). P4 (live-VistA → filesystem extraction) is the only
genuinely-open item, and it is a **net-new capability** with its own live home —
`proposals/package-extraction-design.md` — not a deficiency in shipped v-pkg. So
no new tracker was carved. The 6 broken outbound links the move introduced are
**fixed** (repointed to `../proposals/`, `../design/`, archive-siblings). Original
note:
`docs/archive/implementation-plan.md` was archived today, but it still carries
**OPEN** items with no other live home: **P4** package-extraction (☐ design
only), **P6** gold-doc (☐), **P7** engine-parity (🔒), and Q&A Q1/Q3/Q4/Q5. Its
4 broken relative links (`proposals/…`, `design/…`, `archive/…`) are a symptom —
they resolve from `docs/` root, not from inside `docs/archive/`.
- **Action (choose one):** (a) confirm extraction/P4 is **deferred/out-of-scope**
  and leave archived, OR (b) carve the still-open P4/P6/P7 items into a slim live
  `docs/package-extraction-tracker.md` beside the existing
  `proposals/package-extraction-design.md`, then leave the plan archived. Either
  way, **repoint the 4 broken links** to `../proposals/…` / `../design/…` (or drop
  them — frozen docs may keep dead links, but these broke *today*).

### B3 — Trim the wrap-rpc tombstone — ✅ DONE 2026-06-29
Trimmed `fu5b2-xwbbrk-wrapsplice.md` from 12 KB to a ~20-line stub (points at
`bespoke-installer-forbidden` + git history); shrank its bloated `MEMORY.md`
index line to one line. Original note:
`docs/memory/fu5b2-xwbbrk-wrapsplice.md` is a **12 KB** memory file wholly
describing the **DELETED** `internal/wrapsplice`/`wrap-rpc`/`VSLRPCWRAP` patcher.
It self-labels "⛔ REMOVED 2026-06-25 … historical record only." Its one durable
lesson ("never build a bespoke installer") already lives in
`bespoke-installer-forbidden.md`; the rest is mechanics of code that no longer
exists (git + the proposal hold it). Fails keep-test (b).
- **Action:** delete, or trim to a ~3-line tombstone stub that points at
  `bespoke-installer-forbidden.md` + the git SHA.

### B4 — Three orphan test fixtures — ✅ DONE 2026-06-29 (`git rm`)
`git rm`'d `testdata/zza2-reqb/`, `testdata/zza3-hooks/`, `testdata/zza4-ques/`
(confirmed referenced by no Go test, no gate script, no Makefile target). The
install-fidelity behaviours they once scaffolded are already live-proven (memory
keeps the historical record). Original note:
`testdata/zza2-reqb/`, `testdata/zza3-hooks/`, `testdata/zza4-ques/` are
referenced by **no** Go test and **no** gate script (their feature commits built
the specs inline instead). They carry real Required-Build / hooks / questions
payloads worth gating.
- **Action (choose one):** wire a `*_test.go` case to each (preferred — they test
  real install-fidelity behavior), OR `git rm` the three dirs.

---

## C. Content refreshes — ✅ ALL SWEPT 2026-06-29

C1 `kids-installation-automation.md` §5/§11 marked landed (route (c); engine
parity proven) · C2 `coverage-analysis` status → accepted/largely-implemented +
stale "fold into tracker" note repointed · C3 `clikit-grouped-help` trimmed to
the durable gotchas (dropped version-pin journal) · C4 `streamed-install` Owed
IRIS item marked validated · C5 `snapshot-restore` dropped the deleted-wrap-rpc
lineage (both refs) · C6 `patch-existing` `related:` path → `../archive/` · C7
`architecture.md` §3 diagram adds `classify` + relabels the surface
`pkgcli.Commands` · C8 `package-extraction` "corpus gap" corrected (it's a
section of `krn_8_0_dg_kids_ug`, in corpus) · C9 `MEMORY.md` index trimmed to
true one-liners. Original table:

| # | File | Issue | Action |
|---|---|---|---|
| C1 | `docs/design/kids-installation-automation.md` | §5 still presents "Tier A vs Tier B, choose per environment" as an **open** design choice and §11 frames engine-parity as open — both **resolved/landed** (route (c) augmented direct-populate, live-proven both engines). | Refresh §5/§11 to "landed"; add forward pointer to the (archived) install-fidelity spike. Keep in `design/` (durable KIDS-phase reference). |
| C2 | `docs/proposals/v-pkg-kids-coverage-analysis.md` | `status: proposed` understates that the bulk (T0, A.1–A.4, B.1-a…o, B.2, B.3) has **landed**; closing note "fold into the tracker" points at the now-**archived** implementation-plan. | Set status to reflect near-completion; drop/repoint the stale "fold into tracker" note. Keep until the template family lands, then archive with `from-engine-capture`. |
| C3 | `docs/memory/clikit-grouped-help.md` | Durable bit (mount `ExploreCmd` at the standalone root) wrapped in version-pin journaling ("v0.1.0→v0.2.0, tag v0.4.0 …") that fails keep-test. | Trim to the durable gotcha. |
| C4 | `docs/memory/streamed-install.md` | "Owed: IRIS live-validation of the chunked path" is now **CLOSED** (live-package-gate covered it). | Drop the stale Owed item. |
| C5 | `docs/memory/snapshot-restore.md` | References "the same read path as **wrap-rpc's `readRoutineSource`**" — a **deleted** helper. | Reword to drop the dead-code lineage. |
| C6 | `docs/archive/patch-existing-routines-proposal.md` | Frontmatter `related:` → `docs/fileman-dd-install-plan.md` (now under `archive/`) — stale path (frozen doc, low priority). | Optional: repoint or leave (frozen). |
| C7 | `docs/design/architecture.md` | §3 grammar diagram omits the offline `classify` verb and labels the surface `main.go` (it's `pkgcli.Commands` mounted under `v`). | Optional minor. |
| C8 | `docs/proposals/package-extraction-design.md` | Ref [4]/§10.5 call the "KIDS Developer Tools UG" a corpus **gap**; the spike established it's a **section of** `krn_8_0_dg_kids_ug` (in the corpus). | Optional: update the gap note. |
| C9 | `docs/memory/MEMORY.md` | ~5 index entries are multi-paragraph status dumps, violating the file's own "one line per file" header rule (worst: option-entry-component, multi-field-dd-emitter, mixed-build-split-reversal). | Optional: trim each to one descriptive line. |

---

## D. Verified clean (no action)

- **No live `wrap-rpc` code/verb** — only intended drift-test fixture strings
  (`D req^VSLRPCWRAP`) in `internal/kids/drift_test.go`.
- **`testdata/zzrpc/`** is a legitimate RPC `#8994` component test (heavily
  referenced), unrelated to the deleted wrap-rpc feature.
- **Go version consistent** — `.go-version` `1.26.3` = `go.mod` = README "1.26+".
- **README verb surface matches `pkgcli/commands.go`** exactly across all 4
  groups; the hidden `schema` verb works.
- **`repo.meta.json` `layer: "v"`** correct (VistA-specific).
- **Makefile** targets all reference real scripts/files; both gate scripts use
  only real verbs/flags — no dead flags, no deleted-feature refs.
- **`docs-validate.yml`** validates `docs/` link/anchor integrity via the shared
  `doc-framework/validate.yml` — consistent with the actual layout.
- **No TODO/FIXME/DEPRECATED markers** in `internal/`, `pkgcli/`, `vcontract/`.
- **`examples/decomposed/`** committed `decompose` output is a documented browse
  artifact (no test drift-gates it — acceptable; optionally gate or `.gitignore`).
- **`docs/archive/fileman-dd-install-plan.md`** correctly retired (DONE,
  live-proven; only remaining work is a separate v-stdlib session).
