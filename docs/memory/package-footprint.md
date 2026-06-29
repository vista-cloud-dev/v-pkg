---
name: package-footprint
description: "2026-06-28: A.3 — v-pkg install writes the PACKAGE #9.4 footprint (VERSION #9.49 + PATCH APPLICATION HISTORY #9.4901) via $$PKGVER/$$PKGPAT^XPDIP so downstream $$PATCH^XPDUTL is honest. LIVE-PROVEN both engines. KEY GOTCHA: honesty needs a ≤4-char package prefix."
metadata:
  type: project
---

# v-pkg: PACKAGE #9.4 footprint — coverage-analysis A.3 / F6 (2026-06-28)

The direct-populate install (`EN^XPDIJ`) files the build but writes **no #9.4
footprint**, so a v-pkg install was invisible to later builds'
`$$PATCH^XPDUTL`/`$$VER^XPDUTL` checks — silently breaking the dependency chain
(F6). A.3 closes it. Opt-in via **`v pkg install --register-package "<LONG NAME>"`**;
LIVE-PROVEN on vehu (YDB) + foia-t12 (IRIS). TDD; trunk-based.

## The footprint mechanism (ground-truthed from XPDIP `PKGV` + `$$PKGVER`/`$$PKGPAT`)
The real install line (`PKGV^XPDIP`) reads a `"PKG"` section in the transport global
and calls two extrinsics; v-pkg replicates them directly after `EN^XPDIJ` (route (c),
inside the waterline — **calls the REAL XPDIP, never reimplements KIDS**):
- **`$$PKGVER^XPDIP(pkgIEN, version^^installDate^DUZ)`** — ensures the VERSION
  multiple (#9.49) and sets **CURRENT VERSION** (#9.4 field 13, `…,"VERSION")` p1).
- **`$$PKGPAT^XPDIP(pkgIEN, version, patch^installDate^DUZ)`** — appends to PATCH
  APPLICATION HISTORY (#9.4901). Skipped when the install has no patch.

Both use `$$MDIC` → `UPDATE^DIE`, so the version "B" xref (`…,22,"B",ver,vIEN`) and
the PAH "B" xref (`…,22,vIEN,"PAH","B",patch,pIEN`) are built — exactly what
`$$PATCH^XPDUTL` reads. **KIDS itself never creates the #9.4 entry** (the developer
links the #9.6 BUILD to a pre-existing #9.4 PACKAGE); an authored package has none,
so v-pkg find-or-creates it: `$O(^DIC(9.4,"C",<prefix>,0))` (the PREFIX "C" xref),
else `UPDATE^DIE` a new entry (.01 NAME + field 1 PREFIX). Gated on **#9.7 status 3**
(only a build that actually filed).

## KEY GOTCHA — the footprint is honest to `$$PATCH^XPDUTL` ONLY for a ≤4-char prefix
`$$PATCH^XPDUTL(X)` resolves the package via **`$$LKPKG^XPDUTL`** (XPDUTL line 183):
- `$L(X)<5` → look up by **PREFIX** (`^DIC(9.4,"C",X)`, then `"C2"`).
- `$L(X)>3` → look up by **NAME** (`^DIC(9.4,"B",X)`).

And `$$PATCH` itself pattern-matches the install name with the prefix as **1-4
uppercase/numeric chars** (`X'?1.4UN1"*"…` → `Q 0`). So a package whose install-name
prefix is **>4 chars** (e.g. the 6-char "ZZSKEL") fails the pattern AND can't be
found by PREFIX — `$$PATCH` returns 0 even though the footprint filed correctly.
**Lesson:** for `$$PATCH` honesty the package namespace (= install-name prefix) must
be **≤4 chars** (real VistA convention). v-pkg does NOT hard-gate this (the footprint
still files; longer namespaces are valid for routines/install) — it's a documented
constraint. The first live-prove used "ZZSKEL" and `$$PATCH` returned 0; re-proving
with the 3-char "ZZV" gave `$$PATCH(ZZV*1.0*3)` **0 → 1** (and a non-installed patch
`*99` stayed 0) on both engines.

## Where it lives
- `internal/installspec/script.go`: `PkgReg{Prefix,Name,Version,Patch}` +
  `FinalInstallScript`'s footprint block (find-or-create #9.4 by PREFIX, `$$PKGVER`,
  conditional `$$PKGPAT`, `<<VPKG>>pkg=<ien>` marker). The block is a dotted `D`
  routine-structure (the script is staged as a real routine, so dotted blocks work).
- `pkgcli/lifecycle.go`: `installCmd.RegisterPackage` flag; `packageReg(installName,
  regName)` parses PREFIX*VERSION[*PATCH]; threaded `liveInstallInput.pkgReg` →
  `runInstall` → `FinalInstallScript`; `installResult/installReport.PackageIEN`
  surfaces the stamped #9.4 IEN. `restore.go` + the uninstall re-install path pass nil.

Companion to [[install-fidelity-spike]] (A.1 phase boundary), [[install-hooks-authoring]]
(B.3), [[install-question-answers]] (A.1.3). Roadmap item A.3 in
`docs/proposals/v-pkg-kids-coverage-analysis.md`. Engine access via
[[engine-access-through-driver-stack]].
