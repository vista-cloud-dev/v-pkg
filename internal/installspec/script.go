package installspec

import (
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

// xtmpRoot is the open global reference the per-pair SETs hang off; MRef closes
// the paren. XPDA is an M variable (the #9.7 IEN), so it is unquoted here.
const xtmpRoot = `^XTMP("XPDI",XPDA,`

// InstallScript returns the M source that installs the build named name (whose
// transport-global contents are pairs) into a live VistA, non-interactively.
// header is the install header text recorded in #9.7 field 6 (cosmetic).
//
// The script: (1) sets the KIDS environment + full FM priv and opens HOME for
// IO(0); (2) refuses if the build is already loaded (INST would otherwise prompt
// with no stdin); (3) creates the #9.7 entry via $$INST^XPDIL1; (4) populates
// ^XTMP("XPDI",XPDA,…) from pairs; (5) runs EN^XPDIJ; (6) emits the #9.7 status
// as a ResultMarker line.
func InstallScript(name, header string, pairs []kids.Pair) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }

	nameLit := kids.MString(name)

	w(`S U="^",DUZ=1,DUZ(0)="@" S:'$D(DT) DT=$$DT^XLFDT`)
	// Refuse a re-install up front — INST^XPDIL1 prompts "OK to continue with
	// Load" when the name is already in #9.7, and there is no stdin over Exec.
	w(`I $D(` + `^XPD(9.7,"B",` + nameLit + `)) W "` + ResultMarker + `error=already-installed",! Q`)
	w(`D HOME^%ZIS`)
	w(`S XPDST=0,XPDIT=1,XPDST("H1")=` + kids.MString(header+"  ;Created on "))
	w(`S XPDA=$$INST^XPDIL1(` + nameLit + `)`)
	w(`S ^XTMP("XPDI",0)=$$FMADD^XLFDT(DT,7)_U_DT`)
	for _, p := range pairs {
		w(`S ` + p.Subs.MRef(xtmpRoot) + `=` + kids.MString(p.Value))
	}
	w(`D EN^XPDIJ`)
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
