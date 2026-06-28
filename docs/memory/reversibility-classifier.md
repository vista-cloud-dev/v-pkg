---
name: reversibility-classifier
description: v-pkg reversibility classifier (Classify/ClassifyBuild) + `v pkg classify` verb + full-corpus roundtrip sweep — the keystone for patch reversal.
metadata:
  type: project
---

**Reversibility classifier + corpus harness (2026-06-25)** — the keystone the
patch-existing-routines proposal gates on, plus the full-corpus test the user
asked for. Grounded in the corpus findings (`docs/design/kids-corpus-findings.md`);
implements `docs/archive/patch-existing-routines-proposal.md`.

**`internal/kids/reversibility.go`** — `Classify(k)` / `ClassifyBuild(name,b)`
derive `ClassPureOverwrite` (class 1, restore-reversible) vs `ClassSideEffecting`
(class 2/3, no generic inverse) from the parsed `.KID` ALONE — no engine. Four
node-shape probes on `Build.Pairs()`:
- routine NAME: `s[0]=="RTN" && len==2 && s[1].IsStr()` (skip bare `"RTN")` header
  + `"RTN","X",n,0)` source lines).
- install code: `"BLD",<n>,"<ROLE>")` with NON-EMPTY value, ROLE ∈
  {INI=env-check, INIT=post-install, PRE=pre-install, PRET=pre-transport}. **INID
  is run-timing FLAGS (`^y^n`), NOT code — excluded.** Empty value = declared but
  no routine = not code.
- FileMan entry: top-level `"KRN",<file#>,<ien>,0)` (`len>=4`, s[1]&s[2] numeric,
  s[3] zero) — excludes the `"KRN",<file#>,0)` header and `"KRN",..,"B",..` xrefs.
- FILE/DD: `"BLD",<n>,4,<file#>,0)` (`len>=5`, s[2]==int 4, s[3] numeric, s[4]
  zero) — excludes the `"BLD",<n>,4,0)` multiple header. File numbers are decimals.
Class = SideEffecting iff installCode|fileManEntries|fileDD; overall = the
LEAST-reversible build governs. PureOverwrite requires nothing-but-routines (a
metadata-only build is trivially PureOverwrite).

**`v pkg classify <kid>`** (pkgcli/classify.go) — surfaces the class + per-build
signals; says whether snapshot/restore can reverse it. Adding the verb changed the
command surface → had to `make contract` to regenerate `dist/v-contract.json`
(golden test `TestContract_Golden`).

**`make corpus`** (CORPUS ?= ~/data/kids-patches/VistA/Packages) — `internal/kids/
corpus_test.go` `TestRoundtripCorpus`, gated on `VPKG_KIDS_CORPUS` (SKIPs in CI; no
corpus committed). Round-trips EVERY local KIDS + classifies each in one walk.
**Proven over all 2,404 WorldVistA distributions: PASS=2404, DRIFT=0, ERROR=0;
reversibility pure-overwrite=873 (36%), side-effecting=1531 (63%)** — matches the
independent analyze.py count (847/1557) within ~26 files (analyze.py over-counted
install code via human-readable strings; the node-based classifier is more precise).

**Why/how to apply:** `classify` is the prerequisite for the engine-bound verbs —
class-aware `uninstall` must auto-restore ONLY for PureOverwrite (the 36%), run the
authored back-out for SideEffecting, never silently delete-and-orphan. Next:
`snapshot`/`restore` (pre-image off the engine via the driver stack), class-aware
`install`/`uninstall`, `verify --drift`. Gotcha: `make test` needs `CGO_ENABLED=1`
(Makefile sets 0 but uses -race); airgapped build env vars (GOPROXY=file://…).
