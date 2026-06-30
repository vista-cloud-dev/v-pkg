---
name: clikit-grouped-help
description: v-pkg consumes clikit's discovery UX — grouped help (group:"" tags) by KIDS-lifecycle category + the interactive explore palette. Composition gotchas: mount ExploreCmd at the standalone root; v-cli must co-pin clikit + a v-pkg tag that carries the groups.
metadata:
  type: project
---

**v-pkg consumes clikit's discovery UX.** `group:""` tags on the
`pkgcli.Commands` struct render the help surface as the KIDS lifecycle:

- **Inspect**: parse, classify, lint
- **Transform**: decompose, assemble, roundtrip, canonicalize
- **Build & install**: build, install, verify
- **Back-out**: snapshot, restore, uninstall
- (clikit's own schema/version/install-completions fall in the trailing
  "Commands" bucket.)

**Gotcha — mount `Explore clikit.ExploreCmd` at the STANDALONE root** (`main.go`),
**not** in the shared `pkgcli.Commands`. `ExploreCmd` always explores from the app
root, so mounting it inside `pkgcli` would make `v pkg explore` confusingly browse
the whole `v` tree. The standalone `v-pkg explore` browses the v-pkg tree; the
umbrella gets its own `v explore`.

**Composition gotcha — v-cli must co-pin.** v-cli mounts `pkgcli.Commands`
in-process and shares the `clikit.Context` type, so v-cli MUST pin the *same*
clikit version AND a v-pkg version that pins it. Go MVS alone won't surface
v-pkg's groups under `v pkg` unless v-cli's `require` names the v-pkg tag that
carries them.
