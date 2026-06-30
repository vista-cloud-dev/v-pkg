---
title: "v-pkg User's Guide — the verifiably-safe KIDS installer for VistA"
status: guide
created: 2026-06-30
audience: VistA developers / release engineers / anyone installing or backing out KIDS builds
related:
  - ../design/kids-convention-vs-v-pkg.md   # gap analysis (full)
  - ../design/kids-corpus-findings.md       # corpus evidence (2,404 distributions)
  - ../design/kids-installation-automation.md
  - ../design/architecture.md
---

# v-pkg User's Guide

`v-pkg` is the `v pkg` domain of the VistA developer CLI: a **non-interactive,
engine-neutral, verifiably-safe** driver for the VistA/M software-distribution
format, **KIDS** (Kernel Installation & Distribution System). It builds, inspects,
installs, verifies, snapshots, backs out, and **attests** KIDS builds against a live
VistA engine — YottaDB *or* InterSystems IRIS — through one sanctioned transport
seam, never a bespoke installer.

This guide is in two parts. **Part I — Background Research** establishes, from a live
probe of two real VistA systems and a 2,404-distribution corpus analysis, *what KIDS
software actually is*, *how stock VistA installs it*, *where the stock process leaves
gaps*, and *how v-pkg closes each gap*. **Part II — Using v-pkg** is the operational
manual: setup, the full command summary, and the day-to-day workflows.

> One sentence: **v-pkg installs by faithfully driving the native KIDS filer through
> its real M entry points (no bespoke installer), and adds the safety the convention
> never had — pre-flight overwrite refusal, a class-aware back-out that refuses to
> fake a reversal it cannot soundly compute, and a tamper-evident install
> attestation ledger.**

> ### Reliability, in numbers
> **Proven, not promised — full detail in [§11 Validation & verification](#11-validation--verification--how-thoroughly-v-pkg-is-tested):**
>
> - **~500 automated pass/fail assertions** — **222 Go test functions → 291 subtests**, race-detector on, every CI push.
> - **2,404 real KIDS distributions** (the entire WorldVistA mirror, 157 packages) swept offline, plus **14,296 routine-checksum verifications**.
> - **Dual-engine live gates** on **YottaDB *and* InterSystems IRIS** — `stress` **56/56**, foreign-refuse **24/24**, partition **17/17** on *each* engine.
> - **Every command** covered at the unit tier and exercised live; every safety feature proven against an intentionally-broken patch.

---

## Table of Contents

- **Part I — Background Research**
    1. [What KIDS is, and how stock VistA installs software](#1-what-kids-is-and-how-stock-vista-installs-software)
    2. [The real VistA inventory — a live probe of two production-fidelity systems](#2-the-real-vista-inventory--a-live-probe-of-two-production-fidelity-systems)
    3. [Categories of packages installed in a real VistA](#3-categories-of-packages-installed-in-a-real-vista)
    4. [Gap analysis — how the stock installation process works, and where it falls short](#4-gap-analysis--how-the-stock-installation-process-works-and-where-it-falls-short)
    5. [How v-pkg remediates each gap](#5-how-v-pkg-remediates-each-gap)
- **Part II — Using v-pkg**
    6. [Setup — building the CLI and connecting to an engine](#6-setup--building-the-cli-and-connecting-to-an-engine)
    7. [Command summary — every verb, its action, and its use case](#7-command-summary--every-verb-its-action-and-its-use-case)
    8. [Core workflows](#8-core-workflows)
    9. [Install attestation (the audit ledger)](#9-install-attestation-the-audit-ledger)
    10. [Exit codes — the machine-checkable safety contract](#10-exit-codes--the-machine-checkable-safety-contract)
    11. [Validation & verification — how thoroughly v-pkg is tested](#11-validation--verification--how-thoroughly-v-pkg-is-tested)
    12. [Reference — environment, files, and flags](#12-reference--environment-files-and-flags)

---

# Part I — Background Research

## 1. What KIDS is, and how stock VistA installs software

VistA software ships as a **KIDS distribution** — a `.KID`/`.KIDS` text file containing
a *transport global*: the FileMan **BUILD** file (`#9.6`) serialized as
subscript→value pairs. A build carries, at minimum, the new/changed **routines** (M
program source), and may also carry FileMan **components** (options, RPCs, protocols,
templates, keys, …), whole **files** (data dictionary and/or data), install-time
**hook routines** (environment-check, pre-install, post-install), **install
questions**, and **required-build** prerequisites.

Stock installation is a **three-phase, terminal-driven conversation** an operator runs
from the KIDS menu:

1. **Load** (`[XPD LOAD DISTRIBUTION]` → `EN1^XPDIL`) — read the host file, create an
   **INSTALL** record (`#9.7`), and lay the transport global into `^XTMP`.
2. **Questions** — run the environment check, then ask the standard KIDS questions plus
   any build-specific install questions.
3. **Install** (`[XPD INSTALL BUILD]` → `EN^XPDIJ`) — file every component, run the
   pre/post-install hooks, and record the result in `#9.7` STATUS (piece 9 = `3`,
   "Install Completed").

The **source of truth is FileMan**: `#9.7` records *what installed and when*; the
**PACKAGE** file (`#9.4`) PATCH APPLICATION HISTORY records *which patch is applied*
(read by `$$PATCH^XPDUTL`). The methodology is **forward-only** — *KIDS ships no
generic uninstall.* The only native rollback aid is "Backup a Transport Global," which
backs up **routines only** (not data, DD, or FileMan entries), plus a hand-written,
per-patch back-out procedure.

## 2. The real VistA inventory — a live probe of two production-fidelity systems

To ground this guide in reality rather than theory, the BUILD (`#9.6`), INSTALL
(`#9.7`), and PACKAGE (`#9.4`) files were probed live on **two** production-fidelity
VistA systems through the v-pkg driver stack (the same engine seam the tool uses for
every operation — *not* a raw login):

| Live system | Engine | BUILD `#9.6` (builds present) | INSTALL `#9.7` (install events) | PACKAGE `#9.4` (packages) |
|---|---|---:|---:|---:|
| **vehu** (VistA-VEHU-M) | YottaDB | **12,955** | **14,345** | **470** |
| **foia** (FOIA VistA) | InterSystems IRIS | **13,829** | **13,100** | **157** |

Reproduce (vehu shown):
```sh
m vista exec --engine ydb --transport docker \
  'S U="^" W "BUILD=",$P(^XPD(9.6,0),U,4),", INSTALL=",$P(^XPD(9.7,0),U,4),", PKG=",$P(^DIC(9.4,0),U,4)'
```

**What this tells us.** A real VistA is not a handful of packages — it is the
accumulated sediment of **~13,000 KIDS builds and ~13,000–14,000 install events**, the
deposit of decades of national patches plus local modifications. The two systems differ
revealingly: **vehu** carries more *packages* (470) — it is a fuller VistA with local
and `ZZ`-namespace software — while **foia** (the FOIA public release on IRIS) carries
slightly more *builds* across far fewer (157) packages, skewing toward the national
clinical core. Any tool that installs or backs out software on a system of this scale
must be **non-interactive, idempotent, and refuse-by-default** — there is no operator
who can babysit 13,000 menu conversations.

> The 12,955 vehu builds are the same population the companion analysis
> [`kids-convention-vs-v-pkg.md`](../design/kids-convention-vs-v-pkg.md) classifies;
> the live re-probe here confirms that headline count from the current engine.

## 3. Categories of packages installed in a real VistA

"Category" has three useful axes for a real VistA. All three matter for installation
safety.

### 3a. By function — the VistA package taxonomy

The 157–470 packages in the `#9.4` PACKAGE file span the standard VistA functional
tiers. Representative members (namespace in parentheses):

| Tier | Representative packages (namespace) |
|---|---|
| **Infrastructure / Kernel** | Kernel (`XU`), FileMan (`DI`), MailMan (`XM`), RPC Broker (`XWB`), Task Manager (`XUT`), Health Level Seven (`HL`), Toolkit (`XT`) |
| **Patient administration** | Registration/ADT (`DG`), Scheduling (`SD`), PIMS, Master Patient Index (`MPI`) |
| **Clinical documentation / order entry** | CPRS/Order Entry (`OR`), Text Integration Utilities (`TIU`), Consult/Request Tracking (`GMRC`), Vitals (`GMRV`), Health Summary (`GMTS`), Clinical Reminders (`PXRM`), PCE (`PX`) |
| **Clinical ancillaries** | Laboratory (`LR`), Pharmacy — Outpatient (`PSO`) / Inpatient (`PSJ`) / Data (`PSS`) / ECME (`BPS`), Radiology (`RA`), Surgery (`SR`) |
| **Financial / administrative** | Integrated Billing (`IB`), Accounts Receivable (`PRCA`), Fee Basis (`FB`) |
| **Decision support / interfaces** | VPR (`VPR`), Web services (`WEBP`), Terminology, HL7 messaging (`HL`) |

A package is a **namespace owner**, not a build: one package (e.g. Kernel) accounts for
*hundreds* of the 12,955 builds — every `XU*8.0*NNN` patch is one build under the
Kernel package. This is why the build count dwarfs the package count.

### 3b. By payload — what a build actually ships

A static parse of **2,404 WorldVistA `.KID` distributions** (157 packages,
**14,288 routine instances**;
[`kids-corpus-findings.md`](../design/kids-corpus-findings.md)) categorizes builds by
their transport-global node forms:

| What the build ships / does | Distributions | % |
|---|---:|---:|
| Ship ≥ 1 routine | 2,343 | **97%** |
| **Routines ONLY** — no install code, no entries, no DD/data | **674** | **28%** |
| **Side-effecting** — install code and/or FileMan entries and/or DD/data | **1,730** | **72%** |
| Ship install-time CODE (Env-Check / Pre / Post) | 1,235 | 51% |
| Ship non-routine FileMan entries (options/RPCs/protocols/keys/…) | 1,206 | 50% |
| Ship a FileMan FILE (DD and/or data) | 579 | 24% |
| Declare a real required-build dependency | 1,890 | 79% |
| Multi-build distribution (> 1 install name) | 88 | 3.66% |

The non-routine **component types** shipped, by distribution count:

| File # | Component | % | File # | Component | % |
|---|---|---:|---|---|---:|
| 9.8 | ROUTINE | 95% | .401 | SORT TEMPLATE | 5% |
| 19 | OPTION | **39%** | 3.8 | MAIL GROUP | 4% |
| .4 | PRINT TEMPLATE | 20% | 870 | HL LOGICAL LINK | 4% |
| 19.1 | SECURITY KEY | 20% | .403 | FORM | 3% |
| 101 | PROTOCOL | 18% | 8989.52 | PARAMETER TEMPLATE | 3% |
| .402 | INPUT TEMPLATE | 10% | 779.2 | HLO APPLICATION | 2% |
| 8994 | REMOTE PROCEDURE (RPC) | 9% | 9.2 | HELP FRAME | 2% |
| 409.61 | LIST TEMPLATE | 8% | .84 | DIALOG | 2% |
| 8989.51 | PARAMETER DEFINITION | 7% | 3.6 | BULLETIN | 2% |
| 771 | HL7 APPLICATION | 5% | .5 | FUNCTION | 1% |

Each non-routine row is a **data write into a FileMan file that stock uninstall does
not reverse.** Installing an OPTION (#19) or a PARAMETER DEFINITION (#8989.51) leaves a
permanent row; restoring a routine pre-image does nothing for it.

### 3c. By reversibility — the category that governs back-out safety

Projecting the payload categories onto the question *"can this be undone by restoring a
pre-image?"* yields two reversibility classes — and the headline finding, robust across
**both** independent corpora (the 12,955 vehu installed set *and* the 2,404 WorldVistA
on-disk set):

> **The pure-overwrite, snapshot/restore-reversible class is a *minority* (≈ a third —
> 28–39%). The majority of real builds (61–72%) are *side-effecting* and have no
> generic inverse.**

| Reversibility class | vehu installed (n=12,955) | WorldVistA on-disk (n=2,404) | Can a routine-pre-image restore undo it? |
|---|---:|---:|---|
| **Pure overwrite** (routines only, no hooks/entries/DD) | 38.9% | 28% | **Yes — complete undo** |
| **Side-effecting** (install code, entries, or DD/data) | 61.1% | 72% | **No — orphans data/side-effects** |

The **install hook is the silent reversibility killer**: a build that ships only
routines is "pure" until you notice it also carries a post-install routine (37.1% of
vehu builds do) that may have re-indexed a file or queued a job. v-pkg's classifier
keys exactly on this: `SideEffecting if (install-code OR FileMan-entries OR DD/data)`.

## 4. Gap analysis — how the stock installation process works, and where it falls short

Stock KIDS is a sound *forward* mechanism, but it was built for an operator at a
terminal and leaves several gaps that matter for modern CI/CD, automation, and
auditable safety:

| # | Gap in the stock process | Why it bites |
|---|---|---|
| **G1** | **Interactive only.** Install is a multi-prompt menu conversation (`EN1^XPDIL` host-file load can't be fed over a stdin-less pipe). | Un-automatable; no CI, no agent, no scripted rollout across 13,000-build systems. |
| **G2** | **Silent overwrite.** Native install overwrites a resident (national) routine unconditionally, with no pre-image kept. | A patch can clobber code you can never get back — with no warning and no record. |
| **G3** | **No generic uninstall.** KIDS ships *no* back-out; the answer is a hand-written, per-patch DIBR procedure. | Every rollback is bespoke, manual, and error-prone — and most are never written. |
| **G4** | **Backup covers routines only, silently.** "Backup a Transport Global" captures routines and *says nothing* about the DD, data, and FileMan entries it omits. | A backup that looks complete but silently isn't — restoring it after a side-effecting patch leaves orphaned data. |
| **G5** | **No machine-checkable refusal.** Safety depended on an operator *not* running a dangerous step. | No exit code, no gate — automation has nothing to test against. |
| **G6** | **No pre-install preview.** "Compare Transport Global to Current System" exists but is an interactive, eyes-on step. | You can't programmatically assert "this install changes exactly these N routines and nothing else" before committing. |
| **G7** | **Half-install corruption.** An aborted install can leave a `#9.7` entry with a `"B"` cross-reference but no usable `0`-node; the re-install guard then falsely reports "already installed." | A package gets wedged — neither installed nor re-installable — with no native repair. |
| **G8** | **Checksums are an optional, eyes-on step.** "Verify Checksums in Transport Global" is a menu option an operator may skip. | Transit corruption or tampering of routine source can slip through. |
| **G9** | **No provenance.** "It installed" is a STATUS code in `#9.7`; there is no portable, tamper-evident, offline-auditable record of *what changed on the engine.* | No third-party-verifiable audit trail — you cannot prove, later and off-system, what an install actually did. |

## 5. How v-pkg remediates each gap

v-pkg's design is the **faithful machine reading** of the convention: its *install* is
automation **of** the native filer (same `EN^XPDIJ`, same files, every parameter
honored — **no bespoke installer**), and its *uninstall* and *attestation* are the
missing halves, built honestly. The gaps above map one-to-one onto remediations, most
delivered through the four-increment **verifiable-safety** hardening effort
(completed 2026-06-30):

| Gap | Remediation in v-pkg | Where |
|---|---|---|
| **G1** Interactive only | **Non-interactive, idempotent install** over the driver: stage the transport global into `^XTMP` in size-bounded chunks, then call the *same* `EN^XPDIJ` the menu would, in one engine process. `--answer NAME=VALUE` pre-answers install questions through the real `$$ANSWER^XPDIQ`. | `install` |
| **G2** Silent overwrite | **Pre-flight overwrite refusal.** v-pkg probes which target routines already exist and **refuses (exit 4)** to clobber a national routine unless you pass `--snapshot`/`--auto-snapshot` (capture a pre-image first) or `--allow-overwrite` (explicit, recorded). | `install` `decideInstall` |
| **G3** No uninstall | **Class-aware back-out.** `uninstall` types each build's reversibility from the `.KID` and chooses: `restore` a pre-image, install an authored `--backout`, `delete` (greenfield only), `partition` (restore-foreign-then-delete-greenfield), or **refuse**. | `uninstall` (increment #1: all standard FileMan component types) |
| **G4** Silent partial backup | **Typed snapshot with an honest `completeUndo` flag.** `snapshot`/`--auto-snapshot` captures the live routine pre-image and marks it a complete undo *only* when the build is pure-overwrite with no greenfield adds and no non-routine components — otherwise it itemizes the `uncaptured` components an authored back-out must cover. | `snapshot`, `restore` |
| **G5** No refusal contract | **Exit-coded safety boundary.** Exit `4` = *Refused* is the codification of "do not perform an irreversible mutation you cannot honestly undo." Gates assert it on vehu + foia. | all live verbs |
| **G6** No preview | **Read-only dry-run / diff** (increment #2). `install --dry-run` / `diff <kid>` previews the install against the live engine — per-routine NEW/CHANGED/identical and per-component would-add/would-change/identical — staging nothing, reaching `EN^XPDIJ` never, exit 0 always. | `diff`, `install --dry-run` |
| **G7** Half-install wedge | **`install --heal`** (increment #3a) detects a *proven-corrupt* `#9.7` entry (a `"B"` xref but no usable `0`-node) and purges it by IEN before reinstalling — never touching a healthy install. | `install --heal` |
| **G8** Optional checksums | **Transport-checksum verify** (increment #3b). v-pkg recomputes each routine's line-2-blind `B` checksum vs the `.KID`'s stored value. Default = **warn** (≈1.6% of real national patches are "born self-inconsistent"); `--verify-checksums` makes it a hard pre-connect refuse. Plus **sidecar-integrity** (#3c): a snapshot is content-hashed so a tampered pre-image is refused on restore (`SIDECAR_TAMPERED`). | `install --verify-checksums`, `restore`/`uninstall` |
| **G9** No provenance | **Install attestation** (increment #4). Every engine-mutating op appends a tamper-EVIDENT record (op, before/after routine checksums, components, snapshot ref, status, …) to a host-side append-only ledger, chained by `prevHash` (always) and optionally **ed25519-signed** (`--sign`). `attest verify` validates the chain + signatures and can **replay** the recorded after-state against the live engine. | `install`/`uninstall`/`restore` (default-on), `attest verify` |

The throughline: **where the convention trusted an operator not to do the dangerous
thing, v-pkg makes the dangerous thing unrepresentable** — and where the convention left
no record, v-pkg leaves a verifiable one. Full mechanism and parameter-by-parameter
mapping: [`kids-convention-vs-v-pkg.md`](../design/kids-convention-vs-v-pkg.md).

---

# Part II — Using v-pkg

## 6. Setup — building the CLI and connecting to an engine

`v-pkg` is a single static Go binary (also mountable as `v pkg <verb>` under the `v`
umbrella). Build it:

```sh
cd ~/vista-cloud-dev/v-pkg
make build           # → dist/v-pkg
```

**Engine access is through the driver stack only** — v-pkg never shells into a
container or runs a raw M interpreter. Every live verb takes `--engine ydb|iris` and
`--transport local|docker|remote`; the connection details (container, base URL,
credentials) are read by the driver from its `M_<ENGINE>_*` environment, so they never
appear on the command line. For the local YDB engine (vehu):

```sh
export M_YDB_BIN=../m-ydb/dist/m-ydb M_YDB_CONTAINER=vehu M_YDB_TRANSPORT=docker \
       M_YDB_DIST=/home/vehu/lib/gtm M_YDB_GBLDIR=/home/vehu/g/vehu.gld M_YDB_ROUTINES=/home/vehu/r
```

For IRIS (foia) the driver reads `M_IRIS_BASE_URL` / `_NAMESPACE` / `_USER` /
`_PASSWORD` (loaded from `~/data/vista-cloud-dev/auth.env` via direnv). Add
`--output json` to any verb for machine-readable output.

## 7. Command summary — every verb, its action, and its use case

`v pkg` has two families: **offline** verbs (pure `.KID` manipulation, no engine) and
**live** verbs (drive a real engine through the driver). Live verbs require
`--engine`/`--transport`.

| Command | Engine? | Action | Use case |
|---|:--:|---|---|
| `build <spec>` | no | Build a deterministic, normalized `.KID` from a declarative build spec (`--src`, `--out`). | Author a KIDS distribution from versioned source instead of a manual KIDS build. |
| `parse <kid>` | no | Summarize a `.KID`'s builds and section counts. | Quick inspection of an unknown distribution. |
| `classify <kid>` | no | Derive the reversibility class (pure-overwrite vs side-effecting) from structure alone. | Know *before* touching an engine whether a clean back-out is even possible. |
| `lint <kid>` | no | Run the PIKS data-class gate (`--piks`, `--strict`); exit 3 on a blocked file. | Refuse to version Patient/Institution-class operational data. |
| `decompose <kid> <dir>` | no | Split a `.KID` into a per-component `KIDComponents/` tree. | Diff/review a distribution component-by-component; put it under source control. |
| `assemble <dir> <kid>` | no | Reassemble a component tree back into a `.KID`. | Rebuild a distribution after editing the decomposed tree. |
| `roundtrip <kid>` | no | Verify decompose→assemble reproduces the build byte-for-byte (exit 3 on drift). | Prove a decompose/assemble cycle is lossless before trusting it. |
| `canonicalize <dir>` | no | Substitute install-time IENs with `"IEN"` in a tree (LOSSY; review-only). | Compare two distributions ignoring site-specific record numbers. |
| `diff <kid>` | **yes** | Read-only preview vs the live engine: per-artifact NEW/CHANGED/identical (exit 0 always). | Pre-flight "what would this install change?" without mutating anything. |
| `install <kid>` | **yes** | Install a built `.KID` (non-interactive native KIDS load+install). | The core operation — file a distribution onto a live engine, safely. |
| `verify <kid>` | **yes** | Confirm an install: `#9.7` status + per-routine presence, plus `--drift` / content. | Prove a distribution is fully installed and still applied (not overwritten by a later patch). |
| `snapshot <kid> <out>` | **yes** | Capture the live routine pre-image into a restorable `.KID` (`--name`). | Make a class-1 (pure-overwrite) patch reversible before installing it. |
| `restore <kid>` | **yes** | Re-apply a pre-image snapshot (preview by default; `--commit` installs). | Revert overwritten routines to their captured pre-patch source. |
| `uninstall <kid>` | **yes** | Class-aware back-out (routines + components + `#9.7`/`#9.6`). | Remove a distribution — safely, refusing what it cannot soundly reverse. |
| `attest verify <ledger>` | optional | Validate the attestation ledger's hash chain (+ `--trust` signatures); `--replay` against the live engine. | Audit, off-system, exactly what a sequence of installs/back-outs did — and prove it untampered. |

## 8. Core workflows

### 8a. Build → preview → install (the safe happy path)

```sh
# 1. Build a deterministic .KID from a spec
v pkg build mypkg/kids/MYPKG.build.json --src mypkg/src --out MYPKG.KID

# 2. (offline) Know the reversibility class up front
v pkg classify MYPKG.KID

# 3. (read-only) Preview the exact effect on the live engine — mutates nothing
v pkg diff MYPKG.KID --engine ydb --transport docker        # NEW/CHANGED/identical per artifact

# 4. Install. If it would overwrite existing routines, capture a pre-image first:
v pkg install MYPKG.KID --engine ydb --transport docker --auto-snapshot
#    (writes the pre-image sidecar MYPKG.preimage.kids and, by default, the
#     attestation ledger MYPKG.attest.jsonl)

# 5. Confirm the install
v pkg verify MYPKG.KID --engine ydb --transport docker --drift
```

`install` **refuses (exit 4)** to clobber an existing national routine unless you give
it `--auto-snapshot`/`--snapshot <out>` (the safe path — enables a later `restore`) or
`--allow-overwrite` (explicit, no pre-image). Other useful install flags:
`--register-package "NAME"` (stamp the `#9.4` footprint so `$$PATCH^XPDUTL` is honest),
`--heal` (purge a corrupt half-install first), `--verify-checksums` (hard checksum
gate), `--answer NAME=VALUE` (pre-answer an install question), `--dry-run` (preview and
stop, same as `diff`).

### 8b. Back-out — class-aware uninstall

The right back-out depends on the build's reversibility class, which `uninstall` decides
from the `.KID`:

```sh
# Pure-overwrite patch with a captured sidecar → clean restore of the pre-image:
v pkg uninstall MYPKG.KID --engine ydb --transport docker --verify
#   (auto-detects MYPKG.preimage.kids; --verify confirms the live routines match)

# Side-effecting patch → uninstall REFUSES (exit 4) unless you provide a reversal:
v pkg uninstall SIDEFX.KID --engine ydb --transport docker
#   → refused: "provide --backout <authored inverse> or --restore <pre-image>, or --force"

# Authored inverse for a side-effecting patch:
v pkg uninstall SIDEFX.KID --engine ydb --transport docker --backout SIDEFX-backout.KID

# Also clear the #9.4 patch-history footprint (symmetric to --register-package):
v pkg uninstall MYPKG.KID --engine ydb --transport docker --deregister
```

`--force` overrides a refusal (deletes routines anyway, leaving any declared-foreign
overwrite in place rather than bricking it) — use it only when you accept that
install-time data/side-effects will be orphaned.

### 8c. Decompose / review / reassemble (offline, version-control friendly)

```sh
v pkg decompose MYPKG.KID tree/      # → tree/MYPKG_1.0_1/KIDComponents/...
# ...review/diff the component files under source control...
v pkg assemble tree/ MYPKG2.KID
v pkg roundtrip MYPKG.KID            # prove the cycle is lossless (exit 3 on drift)
```

## 9. Install attestation (the audit ledger)

Every engine-**mutating** op (`install` / `uninstall` / `restore` — never a read-only
`diff`/`verify`) appends a structured record to a host-side append-only **JSON Lines
ledger**, default `<kid>.attest.jsonl` next to the distribution (`--attest <path>` to
override). A record captures the op + sub-action, build name + reversibility class,
engine/transport, **BEFORE/AFTER** per-routine checksums, components touched,
required-build chain, snapshot reference + content hash, `#9.7` status, verify verdict,
exit code, and timestamp.

Two layered protections:
- **Hash chain (always):** each record carries `prevHash` = the prior record's hash, so
  the ledger is append-only and *tamper-evident* — editing any past record breaks the
  chain.
- **ed25519 signature (opt-in, `--sign`):** each record is signed against an **external**
  private key (`$VPKG_ATTEST_KEY`, a hex 32-byte seed), making it *tamper-resistant* —
  the only protection against a determined forger who could re-hash the whole tail.

```sh
# Attestation is ON by default; suppress with --no-attest. Sign with --sign:
VPKG_ATTEST_KEY=<hex-seed> v pkg install MYPKG.KID --engine ydb --transport docker --sign

# Audit the ledger offline (chain + signatures):
v pkg attest verify MYPKG.attest.jsonl --trust <hex-pubkey>

# Replay the recorded after-state against the live engine (does it still match?):
v pkg attest verify MYPKG.attest.jsonl --replay --engine ydb --transport docker
```

`attest verify` exits non-zero (3) if the chain is broken, a signature fails the pinned
key, or a replay mismatches — making the audit a gate, not just a report. Replay checks
each routine's *net* recorded state (the most recent record that touched it), so a full
install→uninstall lifecycle replays clean. A record is written only when the op actually
mutated the engine; a refused / no-op / already-installed op attests nothing.

## 10. Exit codes — the machine-checkable safety contract

| Code | Name | When |
|---|---|---|
| **0** | OK | success |
| **1** | Runtime | engine / I/O / parse failure (`INSTALL_FAILED`, `UNINSTALL_FAILED`, …) |
| **2** | Usage | bad flags (`BAD_ANSWER`, `MULTI_BUILD_REGISTER`, …) |
| **3** | Check | drift / not-verified-clean / broken attestation (`NOT_VERIFIED`, `VERIFY_CLEAN_FAILED`, `ATTEST_VERIFY_FAILED`, `ROUNDTRIP_FAILED`) |
| **4** | **Refused** | the safety stop — `INSTALL_REFUSED` (clobber w/o pre-image), `UNINSTALL_REFUSED` (would brick a foreign routine / side-effecting / wrong sidecar), `ALREADY_INSTALLED`, `CHECKSUM_MISMATCH`, `SIDECAR_TAMPERED`, `NO_DRIVER` |

Exit `4` is the heart of the safety model: *do not perform an irreversible engine
mutation that cannot be honestly undone.* Scripts and CI gate on it directly.

## 11. Validation & verification — how thoroughly v-pkg is tested

v-pkg is built test-first (TDD), and reliability is not a claim — it is a number you
can re-run. Every release is held to **three independent tiers** of automated proof,
and the live tiers run against **two production-fidelity VistA systems on two different
M engines** (YottaDB *and* InterSystems IRIS), not a mock.

### By the numbers

| Tier | What it proves | Scale |
|---|---|---:|
| **Unit + integration** (Go, race-detector on, every CI push) | every function, decision tree, parser, emitter, and engine-interaction path (via a fake driver) | **222 test functions → 291 assertions** |
| **Offline corpus sweep** (`make corpus`) | parse / decompose / assemble / round-trip / checksum, over the **entire WorldVistA patch mirror** | **2,404 real KIDS distributions** (157 packages) — round-trip oracle + **14,296 routine-checksum verifications** |
| **Live adversarial gates** (`make stress` / `live-gate` + foreign-refuse + partition), **× 2 engines** | the full install→verify→back-out lifecycle *and* every safety refusal, against a real VistA | **97 assertions/engine × 2 engines = 194 live assertions** (+ 15 happy-path) |

That is **~500 automated pass/fail assertions plus ~16,700 corpus verifications across
2,404 real-world KIDS distributions**, re-runnable on demand — the unit + corpus tiers
on every push, the live tiers on the engine host.

### The two real engines

The live tiers are run end-to-end on **both** of VistA's supported M engines, against
real, fully-populated systems:

- **vehu** (VistA-VEHU-M) on **YottaDB** — 12,955 builds / 14,345 installs / 470 packages.
- **foia** (FOIA VistA) on **InterSystems IRIS** — 13,829 builds / 13,100 installs / 157 packages.

The same gate scripts pass on both (`make stress` **56/56**, foreign-refuse **24/24**,
partition **17/17** on each engine), so the safety guarantees are engine-independent.

### Per-command coverage

Every command is covered at the unit tier *and* exercised live; the safety-bearing
commands additionally have dedicated adversarial (intentionally-broken-input) probes.

| Command | Unit tests¹ | Offline corpus | Live (YDB + IRIS) | Adversarial / erroneous-input probes |
|---|---:|:--:|:--:|---|
| `build` | ~75 | — | ✓ (every gate builds its inputs) | deterministic-output + golden-KID drift |
| `parse` | ~11 | ✓ 2,404 | ✓ | malformed-section handling |
| `decompose`/`assemble`/`roundtrip`/`canonicalize` | ~15 | ✓ 2,404 round-trips | ✓ | **tamper-faithfulness** (an edit must survive the round-trip, exit 3 on drift) |
| `classify` | ~13 | ✓ 2,404 | — | mixed/foreign-overwrite class detection |
| `lint` (PIKS) | ~6 | — | — | blocked-data-class refusal (exit 3) |
| `diff` / `--dry-run` | ~6 | — | ✓ clean→NEW, installed→identical, **edited→CHANGED** | read-only no-op proof (engine byte-identical) |
| `install` | ~47 | — | ✓ | no-clobber refuse, already-installed, **checksum-mismatch**, **heal-on-healthy**, env-check/required-build, **half-install corpse → heal** |
| `verify` | ~38 | — | ✓ content + drift | not-installed (exit 3), drift detection |
| `snapshot` | ~9 | — | ✓ | honest `completeUndo` for side-effecting builds |
| `restore` | ~8 | — | ✓ | **tampered-sidecar refuse** (`SIDECAR_TAMPERED`) |
| `uninstall` | ~23 | — | ✓ | side-effecting refuse, **foreign-brick refuse** (24/24), **partition ordering** (17/17), double-uninstall grace |
| `attest verify` | ~21 | — | ✓ chain + **live replay** | **tampered-ledger refuse**, wrong-key / unsigned refuse |

¹ Approximate: counts the test functions exercising each command; shared suites
(e.g. the install/verify/uninstall lifecycle suite) are counted under each relevant
command, so the column sums to more than the 222 distinct functions.

### Intentionally-broken patches (the safety demos)

Every safety feature is proven against a deliberately-malformed input that v-pkg must
refuse — these run on **both** engines every `make stress`:

| Broken input | v-pkg must… | Result |
|---|---|---|
| Routine bytes that don't match the stored checksum | refuse `CHECKSUM_MISMATCH` (exit 4) | ✓ |
| Pre-image sidecar edited after capture | refuse `SIDECAR_TAMPERED` (exit 4) | ✓ |
| Attestation ledger record edited | `attest verify` fail (exit 3, chain broken) | ✓ |
| Install over a national routine with no pre-image | refuse `INSTALL_REFUSED` (exit 4) | ✓ |
| Bare uninstall of a side-effecting patch | refuse (would orphan data) | ✓ |
| Uninstall a foreign-overwrite with no pre-image | refuse (would brick a national routine) | ✓ 24/24 |
| `--heal` on a healthy install | refuse (heal ≠ clobber) | ✓ |
| A corrupt half-install (`#9.7` with a `"B"` xref but no `0`-node) | detect + **purge + reinstall** under `--heal` | ✓ |
| Required-build absent | refuse (env-check enforcement) | ✓ |
| Decompose→assemble of a tampered tree | carry the change through (auditable, no silent swallow) | ✓ |

### What is *not* claimed

In the spirit of v-pkg's own honesty principle: the live tiers prove the install
lifecycle on **two real national packages (MSL + VSL) plus 16 synthetic fixtures**, not
by installing all 2,404 corpus distributions on a live engine (infeasible — many have
unmet inter-package dependencies). The corpus-wide tier covers the *offline* path
(parse/round-trip/checksum/classify) across all 2,404. The live adversarial gates run on
the engine host (they need the Docker/remote engines), not inside the per-push CI, which
runs the unit tier + lint + vulnerability scan.

## 12. Reference — environment, files, and flags

**Environment (read by the driver, never on the command line):**
- YDB: `M_YDB_BIN`, `M_YDB_CONTAINER`, `M_YDB_TRANSPORT`, `M_YDB_DIST`, `M_YDB_GBLDIR`, `M_YDB_ROUTINES`
- IRIS: `M_IRIS_BASE_URL` (full Atelier path ending `…/api/atelier/v1/`), `M_IRIS_NAMESPACE`, `M_IRIS_USER`, `M_IRIS_PASSWORD`
- Attestation: `VPKG_ATTEST_KEY` (hex ed25519 seed, for `--sign`), `VPKG_ATTEST_PUBKEY` (hex public key, default for `attest verify --trust`)

**Files v-pkg reads/writes alongside a `.KID`:**
- `<kid>.preimage.kids` — the auto-snapshot pre-image sidecar (`install --auto-snapshot`; auto-detected by `uninstall`)
- `<kid>.attest.jsonl` — the attestation ledger (default; `--attest <path>` to relocate)

**Key flags by verb** (see `v pkg <verb> --help` for the full set):

| Verb | Notable flags |
|---|---|
| `install` | `--dry-run`, `--snapshot <out>`, `--auto-snapshot`, `--allow-overwrite`, `--heal`, `--verify-checksums`, `--skip-env-check`, `--answer NAME=VALUE`, `--register-package "NAME"`, `--attest <path>`, `--no-attest`, `--sign` |
| `uninstall` | `--restore <preimage>`, `--backout <kid>`, `--force`, `--verify`, `--deregister`, `--attest <path>`, `--no-attest`, `--sign` |
| `restore` | `--commit`, `--attest <path>`, `--no-attest`, `--sign` |
| `verify` | `--drift` |
| `snapshot` | `--name` |
| `diff` | (read-only; `--engine`/`--transport` only) |
| `attest verify` | `--trust <hexpub>`, `--replay`, `--engine`, `--transport` |
| `lint` | `--piks <tsv>`, `--strict` |
| `build` | `--src <dir>`, `--out <path>` |

---

*Background research sources: live `#9.6`/`#9.7`/`#9.4` probe of vehu (YDB) + foia
(IRIS) through the v-pkg driver stack (2026-06-30);
[`kids-corpus-findings.md`](../design/kids-corpus-findings.md) (2,404 WorldVistA
distributions); [`kids-convention-vs-v-pkg.md`](../design/kids-convention-vs-v-pkg.md)
(12,955-build vehu classification + full gap analysis);
[`kids-installation-automation.md`](../design/kids-installation-automation.md) (install
mechanism). Command surface generated from `dist/v-contract.json` (contract v1.0,
domain version 0.3.0).*
