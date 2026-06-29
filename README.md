# v pkg

**Modern package management for VistA.** `v pkg` is the `pkg` domain of the
`v` CLI — it builds, installs, verifies, backs out, and version-controls VistA
software packages over a live M engine. It is the plain-language front end for
what VistA calls **KIDS** (the Kernel Installation and Distribution System): the
same `.KID` distributions and `#9.6`/`#9.7` install records, surfaced under
guessable verbs (`build`, `install`, `verify`, `uninstall`, `decompose`,
`assemble`) instead of VistA's `XPD*` option names and menu trees.

This repo builds the standalone **`v-pkg`** binary; the same verbs mount under
the `v` umbrella as `v pkg <verb>` (static-pinned composition). It is a single
static (`CGO_ENABLED=0`) executable on the shared `clikit` conventions
(`--output text|json`, JSON envelope, deterministic errors, the exit-code
ladder, `schema`, `version`).

It reaches a live engine **only** through the `m-driver-sdk` seam (the YDB-VistA
and IRIS-VistA drivers), per the org's transport-monopoly rule — it never
hand-rolls `docker exec`, `mumps -direct`, or an `iris session`.

## What it does

KIDS bundles routines, FileMan DD changes, options, protocols, RPCs, and install
logic into one `.KID` text file, and installs it through a conversational,
terminal-driven menu. `v pkg` replaces that with a scriptable, contract-stable
command surface in four groups:

**Inspect**
- **`parse <kid>`** — summarize a `.KID`'s builds and sections without unpacking.
- **`classify <kid>`** — derive a `.KID`'s reversibility class (pure-overwrite
  vs side-effecting) from its structure — no engine.
- **`lint <kid>`** — the **PIKS data-class gate** (see below). Exit `3` on a
  blocked file.

**Transform** (the git round-trip — no engine)
- **`decompose <kid> <dir>`** — `.KID` → per-component tree (routines as `.m`,
  FileMan DDs as `.zwr`, Kernel components per-entry) so git diffs and merges at
  the granularity of a real change.
- **`assemble <dir> <kid>`** — the reverse: a reviewed tree back into an
  installable `.KID`.
- **`roundtrip <kid>`** — decompose → assemble → re-parse and verify the build
  is reproduced. Exit `3` on drift.
- **`canonicalize <dir>`** — substitute install-time IENs with the literal
  `"IEN"` placeholder for cross-instance diff stability (**LOSSY**; review-only).

**Build & install** (over the live engine)
- **`build <spec>`** — build a KIDS transport global from a declarative build
  spec (deterministic, normalized export).
- **`install <kid>`** — install a built `.KID` on a live engine over the driver
  (non-interactive KIDS load + install).
- **`verify <kid>`** — verify a `.KID`'s install on a live engine (`#9.7` status
  + per-routine presence and content).

**Back-out**
- **`snapshot <kid>`** — capture the live pre-image of a patch's routines into a
  restorable `.KID` (class-1 reversal).
- **`restore <kid>`** — re-apply a pre-image snapshot to revert routines to stock
  (preview by default; `--commit` installs).
- **`uninstall <kid>`** — uninstall a `.KID` from a live engine (routine-only
  back-out: routines + `#9.7`/`#9.6` footprint).

Every command honors the toolchain contract: `--output text|json|auto`, a
versioned JSON envelope, deterministic error objects, the exit-code ladder
(`0` ok · `1` runtime · `2` usage · `3` check/drift · `4` refused), `schema`,
and `version`.

## The round-trip guarantee

Round-trip is **semantic equality after routine line-2 canonicalization — not
byte-identity.** Routine line 2 (`;;VER;PKG;**patches**;date;Build N`) carries
the patch list, build date, and build number, all of which KIDS rewrites on
every install; `roundtrip`/decompose strip them to `;;VER;PKG;;` so diffs stay
meaningful. KIDS re-appends them at install time, so installed behavior is
unchanged. This is XPDK2VC's "do not include the build number" fix, inherited
verbatim.

## The PIKS data-class gate (`lint`)

`lint` enforces gate **K2** from the KIDS round-trip design: operational data
(`DATA`/`FRV*` sections) touching a **Patient-** or **Institution-class**
FileMan file is refused (exit `3`) — that data must never enter git. It also
doubles as a PHI/PII tripwire for the inbound/outbound airlock.

Classification is at **file granularity** (is file *N* Knowledge/System or
Patient/Institution?), which is the right granularity for a guardrail.

> The authoritative PIKS model lives in **vista-meta**
> (Patient/Institution/Knowledge/System over 8,261 FileMan files) and is
> consumed **by reference**, never vendored: `v pkg` ships only a small built-in
> seed of well-known files (e.g. File 2 PATIENT) and accepts the full model via
> `--piks <tsv>` (a `filenumber<TAB>class` table you can export from vista-meta).
> Files with no known class are warnings by default; `--strict` treats them as
> gate failures (fail-closed).

```sh
v pkg lint OR_3.0_484.KID                 # built-in seed only
v pkg lint OR_3.0_484.KID --piks piks.tsv # authoritative vista-meta table
v pkg lint OR_3.0_484.KID --strict        # fail-closed on unclassified files
```

## Usage

```sh
v pkg parse        OR_3.0_484.KID
v pkg decompose    OR_3.0_484.KID ./patches/
v pkg assemble     ./patches/ rebuilt.KID
v pkg roundtrip    OR_3.0_484.KID
v pkg canonicalize ./patches/
v pkg lint         OR_3.0_484.KID
v pkg schema | jq .
```

The standalone binary is invoked as `v-pkg <verb>`; under the `v` umbrella the
identical verbs are `v pkg <verb>`.

## Build

```sh
make build        # dist/v-pkg, static + trimmed + version-stamped
make test         # go test -race -cover ./...
make lint         # golangci-lint
go build -o v-pkg .
```

Prerequisites: Go 1.26+. Builds are pure-Go and `CGO_ENABLED=0`.

## Layout

```
v-pkg/
├── main.go                 # standalone CLI grammar (mounts the pkgcli verbs)
├── pkgcli/                 # the v pkg command surface (importable; the v umbrella mounts the same)
├── internal/
│   ├── kids/               # the format codec: parser, decompose, assemble, roundtrip, canonicalize, PIKS
│   ├── buildspec/          # declarative build → KIDS transport global
│   └── installspec/        # non-interactive load/install over the driver (streamed ^XTMP populate)
├── vcontract/              # the generated command/contract surface (drift-gated)
├── testdata/               # committed .KID / .kids fixtures
├── examples/               # runnable end-to-end demo + user guide + sample tree
├── docs/                   # architecture + lifecycle design docs (see below)
├── Makefile / .golangci.yml / .github/workflows/ci.yml
└── LICENSE
```

## Documentation

- **[`examples/USER_GUIDE.md`](examples/USER_GUIDE.md)** — hands-on end-to-end
  walkthrough (assemble / disassemble a real Kernel patch). Run
  `examples/roundtrip-demo.sh` for the live version.
- **[`docs/design/architecture.md`](docs/design/architecture.md)** — how the
  format codec works: the KIDS `.KID` format, the data model, and the
  assembly/disassembly process, with Mermaid diagrams.
- **[`docs/proposals/package-extraction-design.md`](docs/proposals/package-extraction-design.md)** —
  design proposal for automating extraction of a live VistA system's installed
  packages to the filesystem for analysis.
- **[`docs/design/kids-installation-automation.md`](docs/design/kids-installation-automation.md)**
  — design + procedure for automating KIDS build installation into a VistA
  instance (the `install`/`verify`/`uninstall` lifecycle).

All docs are grounded in official VA VistA documentation, with references at the
bottom of each.

## Heritage

The format codec (`decompose`/`assemble`/`roundtrip` and the `KIDComponents/`
layout) is a faithful Go port of
[`py-kids-vc`](https://github.com/rafael5/py-kids-vc) (itself a port of Sam
Habiel's XPDK2VC); decompose/assemble/canonicalize produce **byte-identical**
output to it on the committed fixtures (modulo the one-line saved-by header). The
build/install/verify/back-out lifecycle is new in `v pkg` — KIDS ships no generic
uninstall, so back-out is the tool's job.

## License

Apache-2.0 (final license reconciliation across the toolchain is deferred to
project completion).
