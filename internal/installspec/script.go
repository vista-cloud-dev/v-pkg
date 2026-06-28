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

// QuesAnswer is one pre-answered KIDS install question (A.1.3): the question NAME
// (the build's #9.6 QUES .01, looked up by ENV/pre/post code via $$ANSWER^XPDIQ)
// and its INTERNAL answer value. It is seeded into the #9.7 INSTALL record's
// internal "QUES" scratch tree so the non-interactive direct-populate path answers
// build-specific questions that the interactive question phase (skipped here) would.
type QuesAnswer struct {
	Name  string // question name — the $$ANSWER^XPDIQ(name) lookup key
	Value string // internal answer value $$ANSWER^XPDIQ returns
}

// FinalInstallScript returns the constant-size install routine run after
// StageChunks has populated ^XTMP("VPKGI",…). nPairs is the expected staged-node
// count: the routine counts the staging global and refuses to install if it does
// not match (so a silently-truncated stage fails loudly instead of installing a
// partial package). header is the #9.7 install header (cosmetic). ques pre-answers
// the build's install questions (A.1.3) so pre/post routines read them via
// $$ANSWER^XPDIQ.
func FinalInstallScript(name, header string, nPairs int, runEnvCheck bool, ques []QuesAnswer) string {
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
	// Start from a CLEAN transport node, exactly as the real KIDS load does. A
	// purged earlier install can free its #9.7 IEN; $$INST^XPDIL1 then re-assigns
	// it, and any stale ^XTMP("XPDI",IEN,…) left at that IEN (e.g. a prior build's
	// REQB / "PRE" nodes) would survive the MERGE below and corrupt the env-check /
	// Required-Build enforcement. KILL first so the staged tree is the only content.
	w(`K ^XTMP("XPDI",XPDA)`)
	w(`M ^XTMP("XPDI",XPDA)=` + stageGbl) // staged tree → live transport global
	// A.1.2 env-check + required-builds (install-fidelity-spike). The
	// direct-populate path jumps to EN^XPDIJ (filing) and SKIPS the load/install
	// phase that runs the build's environment-check routine and enforces Required
	// Builds (#9.611). Reconstruct the minimal install-phase scope EN^XPDI sets
	// (XPDI.m:11) and call the REAL $$ENV^XPDIL1(1): it runs the env-check routine
	// named in ^XTMP("XPDI",XPDA,"PRE") and REQB^XPDIL1 over the BLD…REQB nodes,
	// returning 0=ok, non-zero=reject. XPDT MUST be seeded — ENV's own tail
	// self-rejects a clean build when '$O(XPDT(0)). On reject: purge the aborted
	// #9.7 entry (so a corrected retry is clean) and refuse to file. This INVOKES
	// KIDS, never reimplements it (route (c), inside the waterline + the
	// bespoke-installer ban). Skipped for the restore/back-out callers.
	if runEnvCheck {
		w(`S XPDNM=$P($G(^XPD(9.7,XPDA,0)),U),XPDPKG=+$P($G(^XPD(9.7,XPDA,0)),U,2)`)
		w(`S XPDT(XPDIT)=XPDA_U_XPDNM,XPDT("NM",XPDA)=XPDIT,XPDT("DA",XPDNM)=XPDIT`)
		w(`N XPDENV,XPDENVRC S XPDENV=1,XPDENVRC=$$ENV^XPDIL1(1)`)
		w(`I XPDENVRC K ^XPD(9.7,"B",XPDNM,XPDA),^XPD(9.7,XPDA),` + stageGbl + ` W "` + ResultMarker + `error=env-check-rejected^"_XPDENVRC_"^"_$G(XPDREQAB),! Q`)
	}
	// Pre-answer the build's install questions (A.1.3). The interactive question
	// phase (EN^XPDIQ, via EN^XPDI) is skipped by the direct-populate path, so a
	// pre/post-install routine that calls $$ANSWER^XPDIQ(name) would get "" (= NO)
	// for every question. $$ANSWER reads ^XPD(9.7,XPDA,"QUES","B",name,IEN) → IEN →
	// node 1 (live-ground-truthed); the "QUES" subtree is internal install scratch
	// (no #9.7 FileMan field), so the three nodes below are the whole faithful
	// record. Seeded after the env-check (only a build that passes is worth
	// answering) and before EN^XPDIJ runs the pre/post routines.
	for i, q := range ques {
		ien := strconv.Itoa(i + 1)
		nm := kids.MString(q.Name)
		w(`S ^XPD(9.7,XPDA,"QUES",` + ien + `,0)=` + nm +
			`,^XPD(9.7,XPDA,"QUES",` + ien + `,1)=` + kids.MString(q.Value) +
			`,^XPD(9.7,XPDA,"QUES","B",` + nm + `,` + ien + `)=""`)
	}
	// Seed the #9.7 INSTALL record's KRN component-tracking multiple from the build
	// manifest. A real KIDS load fills this; the direct-populate path does not, and
	// KRN^XPDIK reads ^XPD(9.7,XPDA,"KRN",file,0) without $G — an undefined node
	// faults the install (status stuck at 2). XPCOM then stamps each as installed.
	w(`N XPDBLD S XPDBLD=$O(^XTMP("XPDI",XPDA,"BLD",0))`)
	w(`M:$D(^XTMP("XPDI",XPDA,"BLD",XPDBLD,"KRN")) ^XPD(9.7,XPDA,"KRN")=^XTMP("XPDI",XPDA,"BLD",XPDBLD,"KRN")`)
	// FileMan FILE components (FIA): the interactive loader (GI^XPDIL) sets each
	// file's "file is new" flag (…,0,2) and builds the #9.7 FILE checkpoint via
	// XPCK^XPDIK (XPDIL1). The direct-populate path bypasses that, so do both here:
	// seed …,0,2)=1 (force new-file DD install) and run XPCK^XPDIK("FIA") — else
	// FIA^XPDIK reads ^XPD(9.7,XPDA,4,file,0) without $G and the install faults
	// (status stuck at 2), exactly like the KRN seed.
	w(`N XPF S XPF=0 I $D(^XTMP("XPDI",XPDA,"FIA")) F  S XPF=$O(^XTMP("XPDI",XPDA,"FIA",XPF)) Q:'XPF  S ^XTMP("XPDI",XPDA,"FIA",XPF,0,2)=1`)
	w(`D:$D(^XTMP("XPDI",XPDA,"FIA")) XPCK^XPDIK("FIA")`)
	// Pre/Post-install routine checkpoints (A.1.1, install-fidelity-spike). The
	// real load (PKG^XPDIL1) creates the #9.7 INI/INIT checkpoints; the
	// direct-populate path bypasses the load, so EN^XPDIJ's PRE^/POST^XPDIJ1 loops
	// find no checkpoint and the build's pre/post routines silently never run.
	// Mirror PKG^XPDIL1 exactly: with XPDA + XPDCP in scope, call the *real*
	// $$NEWCP^XPDUTL for the "...COMPLETED" base checkpoint, then the
	// "...STARTED" checkpoint carrying the routine name — but only when the
	// transport carries one (^XTMP("XPDI",XPDA,"INI"/"INIT")), so a build with no
	// pre/post routine just gets the (harmless) base checkpoint. $$NEWCP reads
	// XPDCP (INI→subfile 9.713, INIT→9.716) and files field 2 = the callback
	// routine, which PRE^/POST^XPDIJ1 D @ at install time.
	w(`S XPDCP="INI",XPDCPY=$$NEWCP^XPDUTL("XPD PREINSTALL COMPLETED")`)
	w(`S XPDCPR=$G(^XTMP("XPDI",XPDA,"INI")) I XPDCPR]"" S XPDCPY=$$NEWCP^XPDUTL("XPD PREINSTALL STARTED",XPDCPR)`)
	w(`S XPDCP="INIT",XPDCPY=$$NEWCP^XPDUTL("XPD POSTINSTALL COMPLETED")`)
	w(`S XPDCPR=$G(^XTMP("XPDI",XPDA,"INIT")) I XPDCPR]"" S XPDCPY=$$NEWCP^XPDUTL("XPD POSTINSTALL STARTED",XPDCPR)`)
	w(`D EN^XPDIJ`)
	w(`K ` + stageGbl)
	w(`W "` + ResultMarker + `status=",$P($G(^XPD(9.7,XPDA,0)),U,9),!`)
	return b.String()
}

// VerifyScript returns M source that reports whether name is installed: the
// #9.7 INSTALL presence + status (piece 9; 3 = "Install Completed"), per routine
// whether it is loaded ($T probe), per PARAMETER DEFINITION whether it is present
// in #8989.51 (the "B" index), per OPTION whether it is present in #19 (the "B"
// index), per SECURITY KEY whether it is present in #19.1 (the "B" index), per
// PROTOCOL whether it is present in #101 (the "B" index), per REMOTE PROCEDURE
// whether it is present in #8994 (the "B" index), per MAIL GROUP whether it is
// present in #3.8 (the "B" index), and per FileMan FILE whether its data dictionary
// installed (^DD(file,0) present). Each fact is a ResultMarker line.
func VerifyScript(name string, routines, paramDefs, options, keys, protocols, rpcs, mailGroups, files []string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }

	w(`S U="^"`)
	w(`S XPDA=$O(^XPD(9.7,"B",` + kids.MString(name) + `,0))`)
	w(`W "` + ResultMarker + `installed=",$S(+XPDA:1,1:0),!`)
	w(`W "` + ResultMarker + `status=",$P($G(^XPD(9.7,+XPDA,0)),U,9),!`)
	for _, r := range routines {
		w(`W "` + ResultMarker + `rtn:` + r + `=",$S($T(^` + r + `)]"":1,1:0),!`)
	}
	for _, p := range paramDefs {
		w(`W "` + ResultMarker + `param:` + p + `=",$S($D(^XTV(8989.51,"B",` + kids.MString(p) + `)):1,1:0),!`)
	}
	for _, o := range options {
		w(`W "` + ResultMarker + `option:` + o + `=",$S($D(^DIC(19,"B",` + kids.MString(o) + `)):1,1:0),!`)
	}
	for _, k := range keys {
		w(`W "` + ResultMarker + `key:` + k + `=",$S($D(^DIC(19.1,"B",` + kids.MString(k) + `)):1,1:0),!`)
	}
	for _, pr := range protocols {
		w(`W "` + ResultMarker + `protocol:` + pr + `=",$S($D(^ORD(101,"B",` + kids.MString(pr) + `)):1,1:0),!`)
	}
	for _, rp := range rpcs {
		w(`W "` + ResultMarker + `rpc:` + rp + `=",$S($D(^XWB(8994,"B",` + kids.MString(rp) + `)):1,1:0),!`)
	}
	for _, mg := range mailGroups {
		w(`W "` + ResultMarker + `mailgroup:` + mg + `=",$S($D(^XMB(3.8,"B",` + kids.MString(mg) + `)):1,1:0),!`)
	}
	for _, f := range files {
		w(`W "` + ResultMarker + `file:` + f + `=",$S($D(^DD(` + f + `,0)):1,1:0),!`)
	}
	return b.String()
}

// UninstallScript returns M source that reverses an install (T0a.4): delete each
// routine (^%ZOSF("DEL") removes the .m + .o), each PARAMETER DEFINITION from
// #8989.51 (FileMan DIK by IEN — clears its "B" and subfile xrefs), each OPTION
// from #19 (FileMan DIK by IEN), each SECURITY KEY from #19.1 (FileMan DIK by
// IEN), each PROTOCOL from #101 (FileMan DIK by IEN), each REMOTE PROCEDURE from
// #8994 (FileMan DIK by IEN), each MAIL GROUP from #3.8 (FileMan DIK by IEN), each
// FileMan FILE (its DD, data global, and dict-of-files pointer — KIDS ships no
// generic file uninstall), and the #9.7 INSTALL and #9.6 BUILD entries via DIK. The
// monotonic #9.x / #8989.51 / #19 / #19.1 / #101 / #8994 / #3.8 IEN counters are not
// rolled back (inherent to FileMan, not a leak).
func UninstallScript(name string, routines, paramDefs, options, keys, protocols, rpcs, mailGroups, files []string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }

	w(`S U="^",DUZ=1,DUZ(0)="@"`)
	for _, r := range routines {
		w(`S X=` + kids.MString(r) + ` X ^%ZOSF("DEL")`)
	}
	for _, p := range paramDefs {
		w(`S DA=$O(^XTV(8989.51,"B",` + kids.MString(p) + `,0)),DIK="^XTV(8989.51," I DA D ^DIK`)
	}
	for _, o := range options {
		w(`S DA=$O(^DIC(19,"B",` + kids.MString(o) + `,0)),DIK="^DIC(19," I DA D ^DIK`)
	}
	for _, k := range keys {
		w(`S DA=$O(^DIC(19.1,"B",` + kids.MString(k) + `,0)),DIK="^DIC(19.1," I DA D ^DIK`)
	}
	for _, pr := range protocols {
		w(`S DA=$O(^ORD(101,"B",` + kids.MString(pr) + `,0)),DIK="^ORD(101," I DA D ^DIK`)
	}
	for _, rp := range rpcs {
		w(`S DA=$O(^XWB(8994,"B",` + kids.MString(rp) + `,0)),DIK="^XWB(8994," I DA D ^DIK`)
	}
	for _, mg := range mailGroups {
		w(`S DA=$O(^XMB(3.8,"B",` + kids.MString(mg) + `,0)),DIK="^XMB(3.8," I DA D ^DIK`)
	}
	// FileMan FILE back-out: read the data global root + name BEFORE killing the
	// dictionary, then remove the DD (^DD/^DIC), the data global, and the
	// dict-of-files "B" pointer. @VG indirection kills the file's data global.
	for _, f := range files {
		// VG is the whole "GL" node (a global root like "^DIZ(999000,") — never $P
		// it on U: it starts with "^", so $P(…,"^",1) would be empty. VN (the file
		// name) is piece 1 of the ^DIC 0-node.
		w(`S VN=$P($G(^DIC(` + f + `,0)),U),VG=$G(^DIC(` + f + `,0,"GL"))`)
		w(`I VG]"" S VG=$E(VG,1,$L(VG)-1)_")" K @VG`)
		w(`I VN]"" K ^DIC("B",VN,` + f + `)`)
		w(`K ^DD(` + f + `),^DIC(` + f + `)`)
	}
	nameLit := kids.MString(name)
	w(`S DA=$O(^XPD(9.7,"B",` + nameLit + `,0)),DIK="^XPD(9.7," I DA D ^DIK`)
	w(`S DA=$O(^XPD(9.6,"B",` + nameLit + `,0)),DIK="^XPD(9.6," I DA D ^DIK`)
	w(`W "` + ResultMarker + `uninstalled=1",!`)
	return b.String()
}
