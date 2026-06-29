# v-pkg docs

Documentation for **v-pkg** — the VistA KIDS round-trip + lifecycle tool fronted
by `v pkg` (decompose / assemble / build / install / verify / uninstall /
snapshot / restore / classify). Layout follows the org standard folder set
(`~/vista-cloud-dev/CLAUDE.md`).

## Folders

- **`design/`** — this repo's own design + reference notes:
  - [`architecture.md`](design/architecture.md) — how `v pkg` assembles /
    disassembles a `.KID`, the data model, and the round-trip guarantee.
  - [`kids-installation-automation.md`](design/kids-installation-automation.md) —
    non-interactive KIDS install design + driver contract (P5).
  - [`kids-corpus-findings.md`](design/kids-corpus-findings.md) — corpus evidence
    (2,404 WorldVistA `.KID` builds) grounding the reversibility model.
- **`proposals/`** — not-yet-built designs and active proposals:
  - [`package-extraction-design.md`](proposals/package-extraction-design.md) —
    live-VistA → filesystem extraction (P4, design only).
  - [`v-pkg-kids-coverage-analysis.md`](proposals/v-pkg-kids-coverage-analysis.md) —
    adversarial coverage analysis vs the full KIDS model.
  - [`v-pkg-from-engine-capture.md`](proposals/v-pkg-from-engine-capture.md) —
    `v pkg capture --from-engine` to author the deferred compiled-FileMan
    template family (the remaining unbuilt authoring work).
- **`memory/`** — auto-memory (durable facts only); see
  [`memory/MEMORY.md`](memory/MEMORY.md) for the index.
- **`archive/`** — retired / completed docs (frozen, kept for history), incl.
  the landed `v-pkg-install-fidelity-spike.md` (Track A install fidelity) and the
  `implementation-plan.md` tracker.

## Live trackers

- [`repo-consistency-audit.md`](repo-consistency-audit.md) — 2026-06-29
  comprehensive doc/folder/file consistency audit + flagged actions (Tier-D;
  archive once the flagged moves land).
