---
name: clikit-grouped-help
description: v-pkg repinned to clikit v0.2.0 and its Commands now carry group:"" tags, so `v-pkg`/`v pkg` help shows KIDS-lifecycle categories (Inspect/Transform/Build & install/Back-out). Tagged v0.4.0 (2026-06-26).
metadata:
  type: project
---

**v-pkg picked up clikit's grouped help (2026-06-26, tagged v0.4.0).** Repinned
`github.com/vista-cloud-dev/clikit v0.1.0 → v0.2.0` (the styled, curated, grouped
help renderer + pager; see clikit's `cli-discovery-ux`) and added `group:""` tags
to the `pkgcli.Commands` struct so the help surface reads as the KIDS lifecycle:

- **Inspect**: parse, classify, lint
- **Transform**: decompose, assemble, roundtrip, canonicalize
- **Build & install**: build, install, verify
- **Back-out**: snapshot, restore, uninstall
- (clikit's own schema/version + install-completions fall in the trailing
  "Commands" bucket.)

**Coordination note:** v-cli mounts `pkgcli.Commands` in-process and shares the
`clikit.Context` type, so v-cli MUST pin the same clikit (v0.2.0) AND a v-pkg
version that pins it — hence tagging **v0.4.0** and bumping v-cli's `require
v-pkg v0.4.0`. (Go MVS would still build v-cli with clikit v0.2.0 if only v-cli
required it, but v-pkg's groups only appear once v-cli pins v0.4.0.)

Gates: `go build`/`go vet` clean, `go test -race ./...` green, `v-pkg help`
smoke-tested showing the grouped surface.
