# m-kids — Implementation Plan & Tracker

Single source of truth for m-kids status. Per the org Increment Protocol
(`~/vista-cloud-dev/CLAUDE.md`), update the tracking table **in the same change**
that lands work, add a note under the matching `§ Implementation notes` row,
append any new `§ Lessons learned`, and log directional decisions in `§ Q&A`.

**Last reconciled:** 2026-06-06 (plan created; P0–P3 backfilled as done; P4–P7 pending).

**Legend:** ☑ done · ◐ in progress / partial · ☐ not started · — n/a · 🔒 gated (needs a resource)

---

## 1. Tracking table

| ID | Workstream | Status | Evidence / head | Notes |
|---|---|---|---|---|
| **P0** | Go port scaffold + `clikit` contract (single static `CGO_ENABLED=0` binary, Kong grammar, `--output text\|json`, schema, exit-code ladder) | ☑ | `80c3acc` (stage 4.3) | [§P0](#p0) |
| **P1** | Core round-trip engine — `parse` / `decompose` / `assemble` / `roundtrip` / `canonicalize`, byte-identical to py-kids-vc on fixtures | ☑ | `internal/kids/*`, `kids_test.go` green | [§P1](#p1) |
| **P2** | PIKS data-class gate — `lint` (gate K2), **new in the Go port** | ☑ | `internal/kids/piks.go` | [§P2](#p2) |
| **P3** | Rename `kids-vc` → `m-kids` + docs + examples + merge to `main` + GitHub repo rename | ☑ | `593dcfe`, merge `040003d`, repo `m-kids` | [§P3](#p3) |
| **P4** | Package extraction (live VistA → filesystem `KIDComponents/` tree) | ☐ design only | `docs/package-extraction-design.md` | [§P4](#p4) |
| P4.1 | — Phase 1: inventory only (`#9.4`/`#9.6`/`#9.7` → `inventory.json`, zero writes, no PHI) | ☐ | — | [§P4](#p4) |
| P4.2 | — Phase 2: definition extraction (S2/S3) behind the PIKS airlock | ☐ | — | [§P4](#p4) |
| P4.3 | — Phase 3: S1 re-export → real `.KID` for `m-kids` round-trip | 🔒 | needs gold doc (P6) | [§P4](#p4) |
| **P5** | KIDS install automation (silent/non-interactive install of a `.KID`) | ◑ built — `v pkg install/verify/uninstall` over the driver; **install now streams the transport global in size-bounded chunks → staging global → MERGE + `EN^XPDIJ`** (2026-06-12), fixing a silent partial-install of large packages (one-mega-routine staging truncated). YDB live-proven incl. the full 15-routine MSL base (test-in-place 15/15 suites); IRIS live-validation of the chunked path owed (T0b.2 IRIS leg). **(2026-06-16) Non-routine components: `v pkg build`/`install`/`verify`/`uninstall` now handle a #8989.51 PARAMETER DEFINITION as a KIDS KRN component + Required Builds (#9.611) — the VSL T1.3 enabler. Key fix: the direct-populate install seeds `^XPD(9.7,XPDA,"KRN")` from the build manifest, else `KRN^XPDIK` GVUNDEFs (status stuck at 2). Install→verify→uninstall→verify-clean proven on BOTH engines (vehu YDB + foia-t12 IRIS); `testdata/zzparam` golden + roundtrip-clean. See `docs/memory/krn-param-def-component.md`.** | `pkgcli/lifecycle.go`, `internal/installspec/*`, `internal/kids/krncomp.go`, `internal/buildspec/*`, `docs/kids-installation-automation.md §7` | [§P5](#p5) |
| **P6** | Gold-doc gap — *Kernel 8.0 Developer's Guide: KIDS Developer Tools User Guide* (silent-install APIs + `XPD*` answer vars + re-export entry points) | ☐ | VDL fetch pending | [§P6](#p6) |
| **P7** | Engine parity for extraction + install (`^XTMP`/`XINDEX`/KIDS identical under YottaDB & IRIS) | 🔒 | depends on `m-ydb`/`m-iris` real-engine spikes | [§P7](#p7) |

Critical path for the next phase of work: **P6 (acquire gold doc) → P4.1 (inventory) → P4.2 → P5/P4.3 → P7**.
P4.1 has no dependency on P6 and is the safe first build.

---

## 2. Introduction — where this project came from

m-kids is the **third generation** of one idea: *put VistA KIDS distributions
under real version control by decomposing the monolithic `.KID` file into a
per-component tree.*

```
XPDK2VC  (Sam Habiel, MUMPS)  →  py-kids-vc  (Python)  →  m-kids  (Go)
```

- **XPDK2VC** — Sam Habiel's original MUMPS routine. Established the core idea
  and the critical *"do not include the build number"* fix (routine line 2
  carries volatile patch-list/date/build-number data that KIDS rewrites on every
  install; including it makes git diffs meaningless).
- **`py-kids-vc`** (`~/projects/py-kids-vc`, `github.com/rafael5/py-kids-vc`) —
  the Python port. Defined the `decompose`/`assemble`/`roundtrip` contract and
  the `KIDComponents/` on-disk layout that m-kids preserves verbatim. Its
  ADR-046 explicitly stops short of PIKS classification. **Deliberately
  preserved upstream** — not deleted by the Go port.
- **`m-kids`** (this repo) — the **Go** port, built as **stage 4.3 of the m-cli
  Go toolchain**. It is a *faithful, contract-stable* port: same contract, same
  layout, and `decompose`/`assemble`/`canonicalize` produce **byte-identical**
  output to `py-kids-vc` on the committed fixtures. It adds the `clikit` toolchain
  contract (versioned JSON envelope, deterministic errors, exit-code ladder) and
  one genuinely new capability — the **PIKS data-class gate** (`lint`).

A sibling Python project, **`py-kids-install`** (`~/projects/py-kids-install`),
explores the *install* side; its ideas feed workstream **P5** here.

Why Go / why now: m-kids joins the `m-*` toolchain family (`m-cli`, `m-stdlib`,
`m-ydb`, `m-iris`, `m-driver-sdk`) as a single static binary on the shared
`clikit` conventions, so it composes with the engine drivers — which is what
makes the live-system **extraction (P4)** and **install (P5)** workstreams
reachable: they ride on the `m-ydb`/`m-iris` driver contract rather than
reinventing engine access.

---

## 3. Implementation notes

### P0 — Go port scaffold + clikit contract {#p0}
Single typed Kong `CLI` struct in `main.go`; `schema` reflects the command/flag
tree for agent discovery. Honors the full toolchain contract: `--output
text|json|auto`, versioned JSON envelope, deterministic error objects, and the
exit-code ladder (`0` ok · `1` runtime · `2` usage · `3` check/drift · `4`
refused). Built `CGO_ENABLED=0` → one static executable. `clikit` is kept
**byte-identical** to the sibling repos (consistency-protocol rule).

### P1 — Core round-trip engine {#p1}
Files: `parse.go`, `decompose.go`, `assemble.go`, `roundtrip.go`,
`canonicalize.go`, plus helpers `subs.go` (substitutions), `ddcode.go` (FileMan
DD code), `encoding.go`, `piks.go`. The contract: `.KID` ⇄ `KIDComponents/`
tree (routines as `.m`, FileMan DDs as `.zwr`, Kernel components per-entry).
**Discovery captured in the round-trip guarantee:** equality is *semantic after
routine line-2 canonicalization, not byte-identity* — line 2
(`;;VER;PKG;**patches**;date;Build N`) is stripped to `;;VER;PKG;;` because KIDS
rewrites it at install time. `roundtrip` exits `3` on drift. Correctness is
pinned by **byte-identity tests against py-kids-vc output** on committed fixtures
(`testdata/DG_5_3_853.kid`, `OR_3_0_484.kid`, `XU_8_0_504.kid`, two VM fixtures).

### P2 — PIKS data-class gate (`lint`, gate K2) {#p2}
**New in the Go port — not in py-kids-vc.** Refuses (exit `3`) any
`DATA`/`FRV*` section touching a **Patient-** or **Institution-class** FileMan
file, so operational data / PHI never enters git; doubles as a PHI/PII tripwire
for the inbound/outbound airlock. Classification is at **file granularity**
(file *N* = Knowledge/System vs Patient/Institution), the correct granularity for
a guardrail. The authoritative PIKS model lives in **vista-meta**
(Patient/Institution/Knowledge/System over 8,261 files) — m-kids consumes it
rather than reinventing.

### P3 — Rename + docs + ship {#p3}
`kids-vc` → `m-kids` across dir, Go module, command, Makefile BIN/PKG, NOTICE,
remote. Added `docs/` (architecture + the two design proposals) and `examples/`
(roundtrip demo + transcript + committed `decomposed/` tree + USER_GUIDE, using
the real XU\*8.0\*504 KAAJEE proxy-logon fixture). Merged to `main` 2026-06-05
via `--allow-unrelated-histories -X theirs` (main was only GitHub's 3-file
scaffold → unrelated histories); resulting main tree byte-identical to the port
branch; build+tests green. GitHub repo renamed; port branch deleted. Testdata
fixtures intentionally **left with literal `kids-vc` content** to avoid
perturbing byte-identity round-trip tests.

### P4 — Package extraction (design only) {#p4}
Per `docs/package-extraction-design.md`: read the live `#9.4`/`#9.6`/`#9.7`
files to inventory installed packages, then extract definitions to a
`KIDComponents/` tree behind the PIKS airlock, then (where build entries allow)
re-export real `.KID`s for round-trip. Three strategies (S1 re-export / S2
definition walk / S3 component dump); recommended architecture runs over the
engine-neutral driver contract (P7). **Phase 1 (inventory-only) is zero-write,
no-PHI — ship first.** Open items tracked in `§ Q&A` (Q2, Q5).

### P5 — KIDS install automation (built; IRIS-live pending) {#p5}
Per `docs/kids-installation-automation.md`: drive the authoritative 3-phase KIDS
install (load distribution → install build → post-install) non-interactively.
Two tiers: **Tier A** = native silent-install APIs + `XPD*` answer variables
(needs P6 to confirm exact entry points); **Tier B** = expect-driven against the
interactive menus (the safe interim default). Transport globals live in `^XTMP`;
menus `[XPD MAIN]`/`[XPD INSTALLATION MENU]`; keys `XUPROG`/`XUPROGMODE`. Runs
over the driver contract (P7). Open items in `§ Q&A` (Q3, Q4).

**Built (2026-06-12):** the chosen automation route is the **direct-`^XTMP`
populate** path (§7.1) — `internal/installspec` generates the proven M (create the
`#9.7` entry via `$$INST^XPDIL1` → populate `^XTMP("XPDI",XPDA,…)` from the parsed
`.KID` pairs → synchronous `EN^XPDIJ`; `<<VPKG>>key=value` result markers), and
`pkgcli/lifecycle.go` mounts **`v pkg install` / `verify` / `uninstall`** on it.
The waterline split holds: the KIDS knowledge stays here, while reaching a live
engine goes through the shared `mdriver.Client` (m-driver-sdk v0.3.0) — each verb
stages its script as a scratch routine (`ZVPKG*`) via `exec load` and runs `EN^…`
via `exec run` (one process = one symbol table, so `XPDA` survives the SETs).
Markers are read off captured device output. **Live-proven on the YDB FOIA engine
(T0a.3/T0a.4).** Remaining: **T0a.5** — the same lifecycle live on the **IRIS
FOIA** engine (`foia` container) is the M0a exit gate (three invariants green on
both engines). `v pkg <verb>` takes the built `.KID` + `--engine ydb|iris`
`--transport local|docker|remote`.

### P6 — Gold-doc gap {#p6}
The *Kernel 8.0 Developer's Guide: KIDS Developer Tools User Guide* is **not** in
the `~/data/vdocs` gold corpus and is required for the exact silent-install APIs,
the full `XPD*` answer-variable set, and the `XPD TRANSPORT PACKAGE` /
re-export entry points (blocks Tier A in P5 and Phase 3 in P4.3). The companion
*KIDS User Guide* (`krn_8_0_sm_kids_ug`) **was** fetched from VDL during P3 and
staged. Note from that fetch: WebFetch's model can't read the PDF but `pdftotext`
extracts it cleanly.

### P7 — Engine parity {#p7}
Confirm `^XTMP`, `XINDEX`, and KIDS behave identically under YottaDB and IRIS via
the `m-ydb`/`m-iris` real-engine spikes. Gated on those drivers reaching the
relevant milestones; m-kids extraction/install consume them through the
`m-driver-sdk` contract, never vendor code directly.

---

## 4. Lessons learned

Append as the project teaches us something. Newest at the bottom.

- **L1 — Round-trip is semantic, not byte.** Routine line 2 carries volatile
  patch-list/date/build-number data KIDS rewrites every install. Strip it to
  `;;VER;PKG;;` or git diffs are noise. (XPDK2VC's "don't include the build
  number" fix — inherited verbatim.)
- **L2 — Byte-identity against the prior generation is the strongest port test.**
  Pinning `decompose`/`assemble`/`canonicalize` output to be byte-identical to
  py-kids-vc on real fixtures is what makes "faithful port" verifiable. Corollary:
  **don't rename literal strings in `testdata/`** — leaving `kids-vc` text in
  fixtures preserves the round-trip byte identity those tests depend on.
- **L3 — Don't reinvent classification.** The PIKS Patient/Institution/Knowledge/
  System model is owned by **vista-meta**; the guardrail consumes it at file
  granularity rather than re-deriving it.
- **L4 — Fresh GitHub repos create unrelated histories.** A repo whose only
  commit is the auto-generated `.gitignore`/`LICENSE`/`README` scaffold shares no
  merge base with a locally-rooted work branch. Merge with
  `--allow-unrelated-histories -X theirs` to take the real branch content; verify
  the merged tree is byte-identical to the branch (`git diff --stat main branch`
  empty) before pushing.
- **L5 — Some VA PDFs defeat WebFetch but not `pdftotext`.** When a VDL doc is
  needed and the fetch model returns nothing, `pdftotext` extracts the text.

---

## 5. Q&A — directional decisions

Claude logs decisions it needs from the user here; the user answers inline.
Format: **Q<n> (date, STATUS):** question — *recommendation*. **A:** answer.

- **Q1 (2026-06-06, OPEN):** Which workstream next — package **extraction (P4)**
  or **install automation (P5)**? *Recommendation: start P4.1 (inventory-only):
  zero writes, no PHI, high value, and it surfaces the data needed to scope
  everything else.* **A:** _<pending>_
- **Q2 (2026-06-06, OPEN):** IEN normalization policy for extraction — optimize
  for **cross-instance diff stability** (aggressive `canonicalize`, lossy) or
  **reinstall fidelity** (keep real IENs)? Or emit both views? **A:** _<pending>_
- **Q3 (2026-06-06, OPEN):** Can you source the *KIDS Developer Tools User Guide*
  from VDL (P6), or should Claude attempt the fetch + `pdftotext` extract into the
  gold corpus? It blocks Tier-A silent install and S1 re-export. **A:** _<pending>_
- **Q4 (2026-06-06, OPEN):** For P5, is **Tier B (expect-driven)** an acceptable
  shippable interim while Tier-A native APIs are unconfirmed, or hold P5 until P6
  lands? **A:** _<pending>_
- **Q5 (2026-06-06, OPEN):** What's the expected **build-entry completeness** of
  the target system(s)? (How much running code is *not* covered by a retained
  `#9.6` build entry?) Drives how much S2/S3 fallback P4.2 must implement.
  **A:** _<pending>_
- **Q6 (2026-06-25, OPEN):** Should the verbs grow first-class support for
  **patching an existing routine** (not just greenfield install)? The RPC-broker
  splice (`CALLP^XWBBRK`) exposed that `install` overwrites a national routine with
  no pre-image and `uninstall` *deletes* (bricks it) instead of restoring stock.
  *Recommendation: yes — but reversibility is a property of the whole patch, not
  the routine: KIDS patches run pre/post-install code + data with side-effects that
  have no generic inverse (this is why VA back-out is forward-only). So classify
  each patch (1 pure-overwrite=restore-able · 2 side-effecting=needs an authored
  back-out, VSLTAPBO-style · 3 forward-only=refuse, scaffold a forward back-out
  patch), add `snapshot`/`restore`, make `install`/`uninstall` class-aware, add
  patch-drift `verify`, and never over-claim reversibility. Full design:
  [`patch-existing-routines-proposal.md`](patch-existing-routines-proposal.md).
  **Now corpus-grounded** (2026-06-25): a static parse of all **2,404** WorldVistA
  KIDS distributions shows the pure-overwrite class is the **minority — 35%**;
  **64%** are side-effecting (51% run install code, 23% file FileMan entries, 23%
  ship DD/data, 96% declare required builds). So class-aware `uninstall` and the
  authored-back-out contract are the *common* path, not an edge case — see
  [`kids-corpus-findings.md`](kids-corpus-findings.md).*
  **A (partial, 2026-06-25):** keystone landed — `internal/kids/reversibility.go`
  + `v pkg classify` statically derive the reversibility class (no engine),
  corpus-validated (36%/63%). `make corpus` round-trips ALL 2,404 local KIDS
  (PASS=2404, DRIFT=0). `snapshot`/`restore` also DONE (pkgcli/snapshot.go,
  restore.go) — pre-image capture + class-aware honesty + preview-gated restore,
  live-proven on vehu (real 213-line XWBBRK). Next: class-aware `install`/
  `uninstall` (auto-restore only for class-1; run authored back-out for
  side-effecting), `verify --drift`, non-routine pre-image capture.

---

## References

- `docs/architecture.md` — assembly/disassembly process, data model, round-trip.
- `docs/package-extraction-design.md` — P4 source.
- `docs/kids-installation-automation.md` — P5 source.
- `examples/USER_GUIDE.md` — worked round-trip on XU\*8.0\*504.
- Upstream: `github.com/rafael5/py-kids-vc` (Python), Sam Habiel's XPDK2VC (MUMPS).
- `~/projects/py-kids-install` — sibling install-side Python project (feeds P5).
