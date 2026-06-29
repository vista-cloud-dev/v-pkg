---
name: multi-build-install
description: "2026-06-28: A.4 — v pkg install now installs a multi-build (meta) distribution: each constituent in **KIDS**-header order, stopping on first failure. LIVE-PROVEN both engines. Header order IS install order."
metadata:
  type: project
---

# v-pkg: multi-build distribution install — coverage-analysis A.4 / F7 (2026-06-28)

`v pkg install` (and its loader/uninstall) previously **refused** any `.KID` with
more than one build. A.4 lifts that for INSTALL: a multi-build distribution now
installs each constituent **in `**KIDS**`-header order**, stopping at the first
failure. LIVE-PROVEN on vehu (YDB) + foia-t12 (IRIS). TDD; trunk-based.

## Multi-build structure (ground-truthed from the corpus `MultiBuilds/` dir)
- A multi-build `.KID` header lists every constituent in order, `^`-separated:
  **`**KIDS**:EAS*1.0*96^IVM*2.0*156^`** — and **that order IS the install order**
  (the distribution is written in dependency order). `kids.ParseKID` preserves it in
  `k.InstallNames` (built by splitting the header on `^`); `k.Builds[name]` holds each.
- Each constituent carries an **`MBREQ`** ("Multi-Build REQuirement") section
  (`^XTMP("XPDI",XPDA,"MBREQ")=0`) — per-build metadata; it does NOT block installing
  a single constituent standalone.
- The corpus has a dedicated `~/data/kids-patches/VistA/Packages/MultiBuilds/` folder;
  3.66% of distributions are meta-builds (F7).

## How v-pkg installs it
**Each constituent installs independently** — its own #9.7 INSTALL entry, its own
transport — through the *same* class-aware `liveInstall` path used for a single
build (no parallel installer; waterline rule). This is faithful: real KIDS also runs
`EN^XPDIJ` per constituent. `installSequence` loops `k.InstallNames` in order and
**stops at the first build that fails** (engine/refuse error, or a build that does
not reach #9.7 status 3) — a failed prerequisite must abort the rest, never install
dependents against a missing base.

**Flag handling for multi-build** (build-specific flags are ambiguous across
constituents): `--register-package` and an explicit `--snapshot <path>` are
**refused** (install constituents one at a time to register / snapshot to a named
path); `--auto-snapshot` pairs each build to its **own** sidecar
(`<kid>.preimage.kids.<NAME_with_*→_>`); `--answer NAME=VALUE` applies to every build
(KIDS scopes a question per build, so a NAME matches whichever build defines it);
`--allow-overwrite` / `--skip-env-check` apply to all.

## Where it lives
- `pkgcli/lifecycle.go`: `installSequence(ctx, cl, names, mk)` (the ordered loop,
  stop-on-failure, returns `multiInstallResult{Builds, Installed}`); `installCmd.Run`
  branches to `runMulti` when `len(InstallNames) > 1`; `sanitizeName`/`countInstalled`
  helpers. Single-build path unchanged (same JSON shape). VERIFY/UNINSTALL still
  one-build-at-a-time (A.4 scope is install only).
- Unit-tested with the fake driver: `TestInstallSequence_AllInOrder` (order + all
  installed), `TestInstallSequence_StopsOnFailure` (status≠3 on build 1 → only 1
  report, second never attempted).
- Live-prove built two tiny single builds (ZZM1, ZZM2) and merged them into one
  multi-build `.KID` via `kids.WriteKID`'s multi-name header (a Python text-merge of
  the two `**INSTALL NAME**` blocks under one `**KIDS**:` header) — both reached
  #9.7 status 3 in order, routines present, on both engines.

Roadmap item A.4 in `docs/proposals/v-pkg-kids-coverage-analysis.md`. Engine access
via [[engine-access-through-driver-stack]]. Companion install-fidelity items:
[[install-fidelity-spike]] (A.1), [[package-footprint]] (A.3).
