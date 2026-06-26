---
name: clikit-grouped-help
description: v-pkg consumes clikit's discovery UX — grouped help (group:"" tags, v0.4.0) and the interactive `v-pkg explore` palette (clikit v0.3.2, mounted at the standalone root, v0.5.0). KIDS-lifecycle categories Inspect/Transform/Build & install/Back-out.
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

**Phase 2 — `v-pkg explore` (2026-06-26, tagged v0.5.0).** Repinned clikit
v0.2.0 → v0.3.2 and mounted `Explore clikit.ExploreCmd` at the **standalone root**
in `main.go` (NOT in the shared `pkgcli.Commands`): ExploreCmd always explores
from the app root, so mounting it inside pkgcli would make `v pkg explore`
confusingly browse the whole `v` tree. The standalone `v-pkg explore` browses the
v-pkg tree; the umbrella gets its own `v explore`. v-cli must now pin clikit v0.3.2
+ v-pkg v0.5.0.
