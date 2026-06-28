---
name: kids-coverage-analysis
description: "2026-06-28: adversarial v-pkg coverage analysis vs the full KIDS model — authors ~4 of ~24 component types; the 'install 2000 packages' goal is an install-fidelity problem, not authoring"
metadata:
  type: project
---

**2026-06-28: deep adversarial coverage analysis** of v-pkg vs the complete KIDS
model — `docs/proposals/v-pkg-kids-coverage-analysis.md`. Triggered by v-stdlib
R3 (a multi-field `VSL AUDIT` DD v-pkg can't ship). Grounded in three bases the
owner named: the vdocs GOLD KIDS docs (authoritative target), the 2,404-distribution
WorldVistA corpus (re-tallied), and real `^DD` exports.

**The reframing (key insight):** "install any of the ~2,000 packages" is mostly
**not an authoring problem** — those `.KID`s already exist and v-pkg round-trips
all 2,404 losslessly. The blocker is that the install path **bypasses real KIDS
load** (`^XTMP`-populate + direct `EN^XPDIJ`), so it never runs the build's
Environment Check / Pre-Post-Install / questions / Required-Build enforcement. So
the roadmap splits into **Track A (install fidelity** — the dominant lever for the
existing corpus) and **Track B (authoring coverage** — to build new packages like
VSL). Do **T0 first** (integrity, cheap).

**Adversarial findings:** F1 the build **silently drops** declared
options/keys/protocols/rpcs/templates/mailGroups/hl7 (`buildspec.go:60–71`
accepted, `build.go` ignores — no error) → red-gate-worthy. F2 install bypasses
KIDS load semantics. F3 the FILE feature ships single-`.01` DD-only, but **100% of
the 579 file-shipping dists also ship DATA** (DD-only = 0). F4 Required-Builds
declared, never enforced (51.5% depend on one). F5 **the committed
`kids-corpus-findings.md` numbers are WRONG** — `analyze.py` mis-probes the
`"BLD",ien,"KRN",file,"NM"` / `"REQB",1,0)` layout: OPTION is **39% not 13%**, REQB
**51.5% not 96%**, multi-build **3.66% not 0%**, and FORM/PRINT-TEMPLATE/SECURITY-KEY
were omitted. F6 no PKG #9.4 patch-history footprint. F8 see reframing.

**Corrected authoring priority (by corpus %):** ROUTINE 95% » OPTION 39% » PRINT
TEMPLATE 20% ≈ SECURITY KEY 20% » PROTOCOL 18% » INPUT TEMPLATE 10% » RPC 9% »
LIST TEMPLATE 8% » PARAM DEF 7% » HL7-family/MAIL GROUP ~4–5%. One generic
SEND/DELETE `KRN` emitter (generalizing the #8989.51 path) covers ~20 of 24 types.

**R3 = Track B item B.2** (multi-field DD + DATA with the 4 action codes). The
proposal carries the grounded minimum `^DD`/`^DIC` node-set for a 4-field new file.
Until B.2 lands, R3's stopgap is the interim single-`.01` delimited record.

Reorders/extends the existing `implementation-plan.md` + `kids-installation-automation.md`
threads, not a fork. Waterline + [[bespoke-installer-forbidden]] preserved
(Track A is *more* KIDS-native, not a bespoke installer). The re-tally also owes a
fix to `~/data/kids-patches/analyze.py` (F5/T0.2). See [[reversibility-classifier]],
[[fileman-dd-component]], [[krn-param-def-component]].
