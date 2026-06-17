---
name: fileman-dd-component
description: v-pkg builds + installs/verifies/uninstalls a brand-new FileMan FILE (its data dictionary) as a KIDS FIA component, proven on BOTH engines (VSL M3.T1). Key facts — the FIA transport, the DOUBLED ^DD file-number level, the "full vs partial" piece-3 flag (must be "f" for a new file), and the install seed (XPCK^XPDIK("FIA") + …,0,2)=1).
metadata:
  type: project
---

# v-pkg: FileMan FILE-DD install enabler — #999000 ZZVSLFS (VSL M3.T1, 2026-06-17)

Extends `v pkg build/install/verify/uninstall` to a **brand-new FileMan FILE shipped
as its data dictionary** — the FIA mechanism (`FIA^XPDIK` / `DDIN^DIFROMS`), distinct
from the KRN mechanism [[krn-param-def-component]]. Throwaway file **#999000 ZZVSLFS**,
single free-text `.01` NAME, data global `^DIZ(999000,`. Built so VSLFS can be
re-proven on a dedicated file instead of the borrowed `#8989.51`. Branch
`m3t1-fileman-dd`. Live-proven install→verify→uninstall→clean on **vehu (YDB) +
foia-t12 (IRIS)** over the driver stack, and a record files via `UPDATE^DIE` with the
`"B"` xref firing on both engines.

## What ships — `internal/kids/filecomp.go` (emitted only when Components.Files present; routine/param builds stay byte-identical)
1. **FIA section** (drives `FIA^XPDIK`): `"FIA",file)`=name, `…,0)`=data global,
   `…,0,0)`=`<file>I`, `…,0,1)`=options, `…,0,"VR")`=`ver^ns`.
2. **`^DD` image** — the file number is **DOUBLED**: `("^DD",file,ddfile,…)` (for a top
   file ddfile=file). DIFROMS2 loops `DIFRD=$O(@SA@("^DD",file,DIFRD))` and
   `Q:DIFRD'>0`; a single `("^DD",file,…)` level lands `DIFRD=0` and **silently
   installs nothing** (status still reaches 3 — the trap that wasted the first run).
3. **`^DIC` image** — `("^DIC",file,…)` (the file level prefixes the `^DIC(file,…)`
   subtree; the `"B"` index is `("^DIC",file,"B",name,file)`).
4. **BLD #9.64 FILE manifest** — the build's self-description (not read by the
   direct-populate install).

## THE TWO FIXES that make it actually install (both ground-truthed on live XPDIK/DIFROMS2)
- **Options piece 3 = "f" (FULL), not "p".** `DIFROMS2` sets
  `DIFRFDD=$P(opts,"^",3)'="p"`. Real builds (e.g. DG_5_3_853 file #2) use `y^n^p^…`
  because they **add a field to an existing file** (partial). A **brand-new file MUST
  use `y^n^f^^^^n^^n`** or DDIN errors *"Partial DD/File does not exist"*. `fileSendOpts`
  in filecomp.go.
- **Install seed (`installspec.FinalInstallScript`), the FIA analog of the KRN seed:**
  after the MERGE, when `$D(^XTMP("XPDI",XPDA,"FIA"))` — seed each
  `^XTMP("XPDI",XPDA,"FIA",file,0,2)=1` (the "file is new" flag `GI^XPDIL` would set;
  XPFIL2 read at `…,0,2)` via naked ref) **and** `D XPCK^XPDIK("FIA")` (builds the #9.7
  FILE checkpoint #9.714 — `FIA^XPDIK` reads `^XPD(9.7,XPDA,4,file,0)` WITHOUT `$G`, so an
  absent checkpoint GVUNDEFs). XPDIL1 does `XPCK^XPDIK("FIA"),XPCK^XPDIK("KRN")` at load;
  we bypass the interactive load so we do it ourselves. `EN^XPDIJ`→`XPDIJ1` FMDATA→`FIA^XPDIK`.

## Verify / uninstall — `installspec.{VerifyScript,UninstallScript}` (+ `Build.FileNumbers()` threaded in lifecycle.go)
- Verify probes `$D(^DD(file,0))` → `<<VPKG>>file:<n>=1`.
- Uninstall reads name + data global from `^DIC(file,0)` / `^DIC(file,0,"GL")` **before**
  killing, then removes `^DD(file)`, `^DIC(file)`, the data global, and `^DIC("B",name,file)`.
  **GOTCHA:** the `"GL"` node value (`"^DIZ(999000,"`) **starts with `^`**, so take it
  WHOLE — `$P(…,U,1)` returns "" (the empty piece before the leading caret) and the data
  global never gets killed.

## DD validity
The minimal DD (`^DD(file,0)="FIELD^^.01^1"`, a free-text `.01` with a `"B"` xref + the
`^DD(file,0,"IX","B",file,.01)` index registration) files a record via `UPDATE^DIE` and
the `"B"` cross-reference fires **after a real DDIN install** (DDIN's `IXALL^DIK` reindex
registers it; a hand-SET DD does not auto-fire it). `^DD(file,0)`="FIELD^^…" (NOT the file
name — the name lives in `^DIC(file,0)`); VMDD's `"^DD",file,0,0)="<name>…"` is a synthetic
fixture shape and is WRONG — don't copy it.

Companion to [[streamed-install]] + [[krn-param-def-component]]; engine work via
[[engine-access-through-driver-stack]] (driver stack only). Deterministic golden
`testdata/zzvslfs/ZZVSLFS.kids` (26 pairs, roundtrip-clean).
