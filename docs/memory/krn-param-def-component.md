---
name: krn-param-def-component
description: v-pkg builds + installs/verifies/uninstalls a #8989.51 PARAMETER DEFINITION as a KIDS KRN component (+ Required Builds), proven on BOTH engines. Key fix — the direct-populate install must seed ^XPD(9.7,XPDA,"KRN") from the build manifest or KRN^XPDIK faults (status stuck at 2).
metadata:
  type: project
---

# v-pkg: non-routine KIDS components — #8989.51 PARAMETER DEFINITION + Required Builds (2026-06-16)

Extends `v pkg build`/`install`/`verify`/`uninstall` beyond routine-only to the
first **non-routine KIDS component**: an XPAR PARAMETER DEFINITION (#8989.51),
plus Required Builds (#9.611). Built for VSL/MSL **T1.3** (the v-pkg enabler;
v-stdlib authors `kids/vsl.build.json` on top). Branch `t1.3-kids-data-components`.

## #8989.51 is a KRN (Kernel) component, NOT a FileMan data-DD export
KIDS ships a parameter definition through the **KRN mechanism** (same path as
OPTIONs/RPCs/PROTOCOLs), installed natively by `KRN^XPDIK` under `EN^XPDIJ` — NOT
the generic file-data (`FIA`/`DAT`/DIFROM) export. The transport has three parts
(`internal/kids/krncomp.go` emits all three; routine-only builds stay byte-identical
because each is emitted only when its component is present):
1. **BLD #9.6 manifest** — `"BLD",1,"KRN",0)=^9.67PA^<maxfile>^<cnt>`,
   `…,"KRN",8989.51,0)=8989.51`, a `"NM"` sub-multiple naming each entry, and the
   `"KRN","B"` index. Plus `"BLD",1,"REQB",…` (#9.611 `NAME^action#`) + top `"MBREQ")`=count.
2. **`"ORD"` section** — drives the install order: `"ORD",<ord>,8989.51)` =
   `8989.51;<ord>;1;;;;;;;` (piece 3 = **xref flag = 1** → `IX1^DIK` rebuilds the "B"
   cross-reference) and `,0)=PARAMETER DEFINITION` (the file name `XPDIK` checks).
3. **Top-level `"KRN"` record image** — `"KRN",8989.51,<seq>,{-1,0,1,30…}`. `-1`=`0`
   (XPDFL action; `0`=send/add-or-update). `0`=`NAME^DISPLAY^` (.03 empty=single-valued).
   `1`=`F^^` (1.1 VALUE DATA TYPE; F=free text). `30`=ALLOWABLE ENTITIES (#8989.513):
   header `^8989.513I^n^n`, row `<precedence>^<entityIEN>`.
   `KRN^XPDIK` MERGEs this whole subtree into the live `^XTV(8989.51,DA)` (DA found/
   LAYGO'd by NAME via `$$DIC^XPDIK`), kills `-1`, then re-indexes.

## THE FIX — seed ^XPD(9.7,XPDA,"KRN") from the manifest (else status stuck at 2)
The non-interactive direct-populate install (`installspec.FinalInstallScript`) creates
the #9.7 entry via `$$INST^XPDIL1` and MERGEs the transport into `^XTMP("XPDI",XPDA)`,
but a **real KIDS load also copies the build's KRN component list into the #9.7 INSTALL
record**. `KRN^XPDIK` line 19 reads `^XPD(9.7,XPDA,"KRN",file,0)` **without `$G`** → an
undefined node faults (`%YDB-E-GVUNDEF`); `EN^XPDIJ`'s error trap swallows it and the
install stops at **#9.7 status 2** (the routine installs first, so the symptom is
"routine present, param absent, status 2"). Fix: before `EN^XPDIJ`, seed it —
`M:$D(^XTMP("XPDI",XPDA,"BLD",XPDBLD,"KRN")) ^XPD(9.7,XPDA,"KRN")=^XTMP(…,"BLD",XPDBLD,"KRN")`.
`$$XPCOM` then stamps each `8989.51,0)` → `8989.51^<dt>^<seq>` and status reaches 3.

## Verify / uninstall
`VerifyScript`/`UninstallScript`/`lifecycle.go` thread `Build.ParamDefNames()` (read
from the top-level KRN 0-nodes) alongside routines. Verify probes
`$D(^XTV(8989.51,"B",<name>))`; uninstall backs out with `DIK` on `^XTV(8989.51,`
(by IEN from "B") — clears the "B" + 30-subfile xrefs.

## Portability + proof
The SYS entity is `#8989.518` **IEN 4.2** (DINUM'd to the file number → national
constant, identical on YDB and IRIS). `buildspec` resolves entity abbrev→IEN
(`SYS`→`4.2`, `USR`→`200`, …) and data-type name→code (`free text`→`F`).
**Live-proven install→verify→uninstall→verify-clean on BOTH engines** over the driver
(`testdata/zzparam`, `ZZPARAM GREETING`): vehu (YDB, `--transport docker` +
`M_YDB_*`) and foia-t12 (IRIS, `--transport docker` + `M_IRIS_CONTAINER`/`_NAMESPACE`/
`_IRIS_INSTANCE`). Param lands in #8989.51 (free text, SYS 4.2) and backs out clean
(param/#9.7/routine all gone). Golden `.KID` committed + `v pkg roundtrip`-clean.

The structure was ground-truthed against vehu's live `^XTV(8989.51)` + the
`XPDIK`/`XPDIJ`/`XPDIJ1` source and the bundled `XU*8.0*504` reference build — see
[[engine-access-through-driver-stack]]; all engine work went through the driver stack.
Companion to [[streamed-install]].
