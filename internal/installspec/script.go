package installspec

import (
	"strconv"
	"strings"

	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// This file generates the M install script for the non-interactive
// direct-populate path (T0a.3, live-proven on YDB FOIA — see
// kids-installation-automation.md §7.1). Rather than driving the interactive
// EN1^XPDIL host-file load (which prompts and cannot be fed over the driver's
// subprocess+JSON Exec), it creates the KIDS #9.7 INSTALL entry via the real
// $$INST^XPDIL1, populates ^XTMP("XPDI",XPDA,…) directly from the parsed .KID
// pairs (those pairs ARE the transport-global contents), then runs the
// synchronous EN^XPDIJ install. The script is one execution unit: the driver
// must run it with a persistent symbol table so XPDA survives across the SETs.

// ResultMarker prefixes the script's machine-readable result lines on the
// principal device — the driver layer scans stdout for it. Format:
// `<<VPKG>>key=value`.
const ResultMarker = "<<VPKG>>"

// The install is delivered in two phases so it never embeds the whole transport
// global in one routine. A real package's transport global is large (the MSL
// base is ~6100 nodes / ~560 KB); one routine that big silently truncates when
// the driver stages it (T0b.2 discoveries P1), installing only a prefix. Instead:
//
//  1. StageChunks streams the pairs into a staging global ^XTMP("VPKGI",…) as a
//     sequence of small, size-bounded routine bodies (each stages reliably).
//  2. FinalInstallScript verifies the staged node count, then — in ONE process,
//     so XPDA survives — creates the #9.7 entry, MERGEs the staged tree into
//     ^XTMP("XPDI",XPDA), and runs EN^XPDIJ.
//
// The MERGE makes the install routine constant-size regardless of package size.

// stageOpen is the open global reference the staging SETs hang off (MRef closes
// the paren); stageGbl is the same global, used to clear/merge/count it whole.
const (
	stageOpen = `^XTMP("VPKGI",`
	stageGbl  = `^XTMP("VPKGI")`
)

// StageChunks renders the transport-global pairs as M routine bodies that
// populate the staging global ^XTMP("VPKGI",…). Each body is kept at or below
// maxBytes (a lone over-long SET is its own chunk), so no single staged routine
// is large enough to hit the driver's silent-truncation limit. The first body
// clears any stale staging global. Run each body in order; the global persists
// across the (stateless) driver processes, accumulating the whole tree.
func StageChunks(pairs []kids.Pair, maxBytes int) []string {
	var chunks []string
	var b strings.Builder
	b.WriteString("K " + stageGbl + "\n") // clear stale staging before the first batch
	for _, p := range pairs {
		line := "S " + p.Subs.MRef(stageOpen) + "=" + kids.MString(p.Value) + "\n"
		if b.Len() > 0 && b.Len()+len(line) > maxBytes {
			chunks = append(chunks, b.String())
			b.Reset()
		}
		b.WriteString(line)
	}
	if b.Len() > 0 {
		chunks = append(chunks, b.String())
	}
	return chunks
}

// FinalInstallScript returns the constant-size install routine run after
// StageChunks has populated ^XTMP("VPKGI",…). nPairs is the expected staged-node
// count: the routine counts the staging global and refuses to install if it does
// not match (so a silently-truncated stage fails loudly instead of installing a
// partial package). header is the #9.7 install header (cosmetic).
func FinalInstallScript(name, header string, nPairs int) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }
	nameLit := kids.MString(name)

	w(`S U="^",DUZ=1,DUZ(0)="@" S:'$D(DT) DT=$$DT^XLFDT`)
	// Refuse a re-install up front — INST^XPDIL1 prompts "OK to continue with
	// Load" when the name is already in #9.7, and there is no stdin over Exec.
	w(`I $D(^XPD(9.7,"B",` + nameLit + `)) K ` + stageGbl + ` W "` + ResultMarker + `error=already-installed",! Q`)
	// Count the staged nodes ($QUERY over the staging subtree) and refuse unless
	// every pair arrived — guards against a silently-truncated chunk stage.
	w(`N VC,VR S VC=0,VR=` + kids.MString(stageGbl))
	w(`F  S VR=$Q(@VR) Q:(VR="")!(VR'[` + kids.MString("VPKGI") + `)  S VC=VC+1`)
	w(`W "` + ResultMarker + `staged=",VC,!`)
	w(`I VC'=` + strconv.Itoa(nPairs) + ` K ` + stageGbl + ` W "` + ResultMarker + `error=stage-incomplete",! Q`)
	w(`D HOME^%ZIS`)
	w(`S XPDST=0,XPDIT=1,XPDST("H1")=` + kids.MString(header+"  ;Created on "))
	w(`S XPDA=$$INST^XPDIL1(` + nameLit + `)`)
	w(`S ^XTMP("XPDI",0)=$$FMADD^XLFDT(DT,7)_U_DT`)
	w(`M ^XTMP("XPDI",XPDA)=` + stageGbl) // staged tree → live transport global
	w(`D EN^XPDIJ`)
	w(`K ` + stageGbl)
	w(`W "` + ResultMarker + `status=",$P($G(^XPD(9.7,XPDA,0)),U,9),!`)
	return b.String()
}

// VerifyScript returns M source that reports whether name is installed: the
// #9.7 INSTALL presence + status (piece 9; 3 = "Install Completed") and, per
// routine, whether it is loaded ($T probe). Each fact is a ResultMarker line.
func VerifyScript(name string, routines []string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }

	w(`S U="^"`)
	w(`S XPDA=$O(^XPD(9.7,"B",` + kids.MString(name) + `,0))`)
	w(`W "` + ResultMarker + `installed=",$S(+XPDA:1,1:0),!`)
	w(`W "` + ResultMarker + `status=",$P($G(^XPD(9.7,+XPDA,0)),U,9),!`)
	for _, r := range routines {
		w(`W "` + ResultMarker + `rtn:` + r + `=",$S($T(^` + r + `)]"":1,1:0),!`)
	}
	return b.String()
}

// UninstallScript returns M source that reverses a routine-only install
// (T0a.4): delete each routine (^%ZOSF("DEL") removes the .m + .o) and the #9.7
// INSTALL and #9.6 BUILD entries via FileMan DIK (which also clears their
// cross-references). KIDS ships no generic uninstall — back-out is the tool's
// job. The monotonic #9.x IEN counters are not rolled back (inherent to
// FileMan, not a leak).
func UninstallScript(name string, routines []string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }

	w(`S U="^",DUZ=1,DUZ(0)="@"`)
	for _, r := range routines {
		w(`S X=` + kids.MString(r) + ` X ^%ZOSF("DEL")`)
	}
	nameLit := kids.MString(name)
	w(`S DA=$O(^XPD(9.7,"B",` + nameLit + `,0)),DIK="^XPD(9.7," I DA D ^DIK`)
	w(`S DA=$O(^XPD(9.6,"B",` + nameLit + `,0)),DIK="^XPD(9.6," I DA D ^DIK`)
	w(`W "` + ResultMarker + `uninstalled=1",!`)
	return b.String()
}
