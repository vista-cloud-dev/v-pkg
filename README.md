# kids-vc

**Version control for VistA KIDS distribution files.** Decompose a monolithic
`.KID` patch into a per-component tree you can track in git, and reassemble that
tree back into an installable `.KID`.

This is the **Go** port of [`py-kids-vc`](https://github.com/rafael5/py-kids-vc)
(itself a port of Sam Habiel's XPDK2VC), part of stage 4.3 of the m-cli Go
toolchain. It is a faithful, contract-stable port: the `decompose`/`assemble`/
`roundtrip` contract and the `KIDComponents/` layout are unchanged from the
Python tool, and decompose/assemble/canonicalize produce **byte-identical**
output to it on the committed fixtures. The binary is a single static
(`CGO_ENABLED=0`) executable built on the shared `clikit` conventions.

## What it does

KIDS bundles routines, FileMan DD changes, options, protocols, RPCs, and
install logic into one `.KID` text file. Git's line-based diff/merge is
destructive on this format because adjacent entries are semantically
independent. `kids-vc` provides:

- **`decompose <kid> <dir>`** — `.KID` → per-component tree (routines as `.m`,
  FileMan DDs as `.zwr`, Kernel components per-entry).
- **`assemble <dir> <kid>`** — the reverse.
- **`roundtrip <kid>`** — decompose → assemble → re-parse and verify the build
  is reproduced. Exit `3` on drift.
- **`canonicalize <dir>`** — substitute install-time IENs with the literal
  `"IEN"` placeholder for cross-instance diff stability (**LOSSY**; review-only).
- **`parse <kid>`** — summarize a `.KID` without decomposing.
- **`lint <kid>`** — the **PIKS data-class gate** (see below). Exit `3` on a
  blocked file.

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

> **This command is new in the Go port — it is not in py-kids-vc** (whose
> ADR-046 explicitly stops short of PIKS classification). The authoritative PIKS
> model lives in **vista-meta** (Patient/Institution/Knowledge/System over 8,261
> FileMan files) and is consumed **by reference**, never vendored: `kids-vc`
> ships only a small built-in seed of well-known files (e.g. File 2 PATIENT) and
> accepts the full model via `--piks <tsv>` (a `filenumber<TAB>class` table you
> can export from vista-meta). Files with no known class are warnings by
> default; `--strict` treats them as gate failures (fail-closed).

```sh
kids-vc lint OR_3.0_484.KID                 # built-in seed only
kids-vc lint OR_3.0_484.KID --piks piks.tsv # authoritative vista-meta table
kids-vc lint OR_3.0_484.KID --strict        # fail-closed on unclassified files
```

## Usage

```sh
kids-vc parse        OR_3.0_484.KID
kids-vc decompose    OR_3.0_484.KID ./patches/
kids-vc assemble     ./patches/ rebuilt.KID
kids-vc roundtrip    OR_3.0_484.KID
kids-vc canonicalize ./patches/
kids-vc lint         OR_3.0_484.KID
kids-vc schema | jq .
```

## Build

```sh
make build        # dist/kids-vc, static + trimmed + version-stamped
make test         # go test -race -cover ./...
make lint         # golangci-lint
go build -o kids-vc .
```

Prerequisites: Go 1.26+. Builds are pure-Go and `CGO_ENABLED=0`.

## Layout

```
kids-vc/
├── main.go                 # CLI grammar (Kong) + command bodies
├── clikit/                 # shared convention layer (vendored from go-cli-template)
├── internal/kids/          # the port: parser, codec, decompose, assemble, roundtrip, canonicalize, PIKS
│   └── testdata/*.kid      # the 5 committed fixtures (DG/OR/VMDD/VMTEST/XU)
├── Makefile / .golangci.yml / .github/workflows/ci.yml
└── LICENSE / NOTICE        # Apache-2.0
```

## Validation

- Round-trip is green on all 5 committed fixtures (`go test`).
- `decompose`, `assemble` (modulo the one-line saved-by header), and
  `canonicalize` produce **byte-identical** output to `py-kids-vc` on all 5
  fixtures.
- Full WorldVistA corpus (2,406 patches) is the **G6** gate — see the toolchain
  tracker for current status.

## Companion

`install` is **out of scope** — runtime install/verify against a live YottaDB
VistA lives in the separate `py-kids-install` component. Pipeline:
`decompose → edit/git → assemble` (kids-vc) → `install → verify`
(py-kids-install).

## License

Apache-2.0 (final license reconciliation across the toolchain is deferred to
project completion).
