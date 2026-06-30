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

// The staging global is subscripted by a per-OP token — ^XTMP("VPKGI",<token>,…) —
// so two concurrent installs against the same engine stage into DISJOINT subtrees and
// cannot clobber or contaminate each other's staged transport (the count guard alone
// could not distinguish interleaved nodes). stageOpenFor is the open ref the staging
// SETs hang off (MRef closes the paren); stageGblFor is the same subtree whole
// (clear/merge/count). The token is escaped as an M string subscript.
func stageOpenFor(token string) string { return `^XTMP("VPKGI",` + kids.MString(token) + `,` }
func stageGblFor(token string) string  { return `^XTMP("VPKGI",` + kids.MString(token) + `)` }

// StageChunks renders the transport-global pairs as M routine bodies that populate
// this op's staging subtree ^XTMP("VPKGI",token,…). Each body is kept at or below
// maxBytes (a lone over-long SET is its own chunk), so no single staged routine is
// large enough to hit the driver's silent-truncation limit. The first body clears any
// stale staging subtree FOR THIS TOKEN. Run each body in order; the global persists
// across the (stateless) driver processes, accumulating the whole tree.
func StageChunks(pairs []kids.Pair, maxBytes int, token string) []string {
	stageOpen, stageGbl := stageOpenFor(token), stageGblFor(token)
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
// PkgReg is an optional PACKAGE #9.4 registration: the footprint a build writes so
// downstream $$VER/$$PATCH^XPDUTL checks see the install. Prefix is the #9.4 PREFIX
// (the namespace), Name the #9.4 .01 long NAME (used to create the entry when it is
// absent), Version/Patch the install's version + patch. A nil *PkgReg writes no
// footprint (matching a "NO package" build).
type PkgReg struct {
	Prefix  string
	Name    string
	Version string
	Patch   string
}

func FinalInstallScript(name, header string, nPairs int, runEnvCheck bool, ques []QuesAnswer, reg *PkgReg, token string) string {
	stageGbl := stageGblFor(token)
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }
	nameLit := kids.MString(name)

	w(`S U="^",DUZ=1,DUZ(0)="@" S:'$D(DT) DT=$$DT^XLFDT`)
	// Serialize same-build installs: hold an advisory LOCK across the
	// already-installed guard AND the $$INST^XPDIL1 entry-create below, so two
	// concurrent installs of the SAME name cannot both pass the $D check and file two
	// #9.7 entries (the duplicate-entry TOCTOU). The LOCK is incremental on a v-pkg
	// node (not a KIDS global, to avoid colliding with KIDS' own locks); a 5s timeout
	// refuses rather than blocking forever. Held for the whole process (released at Q).
	w(`L +^XTMP("VPKGLOCK",` + nameLit + `):5 E  K ` + stageGbl + ` W "` + ResultMarker + `error=install-locked",! Q`)
	// Refuse a re-install up front — INST^XPDIL1 prompts "OK to continue with
	// Load" when the name is already in #9.7, and there is no stdin over Exec.
	w(`I $D(^XPD(9.7,"B",` + nameLit + `)) L -^XTMP("VPKGLOCK",` + nameLit + `) K ` + stageGbl + ` W "` + ResultMarker + `error=already-installed",! Q`)
	// Count the staged nodes ($QUERY over THIS op's token subtree only) and refuse
	// unless every pair arrived — guards a silently-truncated chunk stage, and the
	// $QSUBSCRIPT bound keeps a concurrent op's subtree from inflating the count.
	w(`N VC,VR S VC=0,VR=` + kids.MString(stageGbl))
	w(`F  S VR=$Q(@VR) Q:(VR="")!($QS(VR,1)'=` + kids.MString("VPKGI") + `)!($QS(VR,2)'=` + kids.MString(token) + `)  S VC=VC+1`)
	w(`W "` + ResultMarker + `staged=",VC,!`)
	w(`I VC'=` + strconv.Itoa(nPairs) + ` L -^XTMP("VPKGLOCK",` + nameLit + `) K ` + stageGbl + ` W "` + ResultMarker + `error=stage-incomplete",! Q`)
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
		w(`I XPDENVRC L -^XTMP("VPKGLOCK",` + nameLit + `) K ^XPD(9.7,"B",XPDNM,XPDA),^XPD(9.7,XPDA),` + stageGbl + ` W "` + ResultMarker + `error=env-check-rejected^"_XPDENVRC_"^"_$G(XPDREQAB),! Q`)
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
	w(`L -^XTMP("VPKGLOCK",` + nameLit + `)`)
	// Nonce-tag the success oracle with the per-op token (status:<token>): the token is
	// host-random and lives only in this op's scratch routine, so a build's pre/post
	// routine (which runs INSIDE EN^XPDIJ, before this line) cannot pre-forge a
	// `status:<token>=3` marker. The Go side trusts only the token-tagged line.
	w(`W "` + ResultMarker + `status:` + token + `=",$P($G(^XPD(9.7,XPDA,0)),U,9),!`)
	// A.3 PACKAGE #9.4 footprint (F6). The real install line (XPDIP, PKGV) stamps the
	// package's VERSION (#9.49) + PATCH APPLICATION HISTORY (#9.4901) from the build's
	// PACKAGE FILE LINK; the direct-populate path files the build but writes no #9.4
	// footprint, so a v-pkg install is invisible to later $$VER/$$PATCH^XPDUTL checks.
	// When the build declares a registration, replicate it via the REAL XPDIP
	// extrinsics (route (c), inside the waterline): find the #9.4 entry by PREFIX (the
	// "C" xref), CREATE it (NAME + PREFIX) when absent — KIDS itself requires the entry
	// to pre-exist, but an authored package must register itself — then $$PKGVER^XPDIP
	// (VERSION + CURRENT VERSION) and, when a patch is present, $$PKGPAT^XPDIP (patch
	// history). Only when the build actually filed (status 3).
	if reg != nil {
		w(`I $P($G(^XPD(9.7,XPDA,0)),U,9)=3 D`)
		w(` . N XPDPDA,XPDPV,XPDPP`)
		w(` . S XPDPDA=+$O(^DIC(9.4,"C",` + kids.MString(reg.Prefix) + `,0))`)
		w(` . I 'XPDPDA D`)
		w(` . . N XPDFDA,XPDPIEN`)
		w(` . . S XPDFDA(9.4,"+1,",.01)=` + kids.MString(reg.Name) + `,XPDFDA(9.4,"+1,",1)=` + kids.MString(reg.Prefix))
		w(` . . D UPDATE^DIE("","XPDFDA","XPDPIEN")`)
		w(` . . S XPDPDA=+$G(XPDPIEN(1))`)
		w(` . Q:'XPDPDA`)
		w(` . S XPDPV=$$PKGVER^XPDIP(XPDPDA,` + kids.MString(reg.Version) + `_"^^"_DT_"^"_DUZ)`)
		if reg.Patch != "" {
			w(` . S XPDPP=$$PKGPAT^XPDIP(XPDPDA,` + kids.MString(reg.Version) + `,` + kids.MString(reg.Patch) + `_"^"_DT_"^"_DUZ)`)
		}
		w(` . W "` + ResultMarker + `pkg=",XPDPDA,!`)
	}
	return b.String()
}

// HealDetectScript returns READ-ONLY M source that probes a #9.7 INSTALL entry for
// the half-install-corruption signal (kids-installation-automation.md §7.1): a prior
// aborted install can leave the "B" xref (so the re-install guard falsely reports
// already-installed) but no usable 0-node / a status that never reached 3. It emits
// the IEN, whether the 0-node exists, and the status (piece 9) as ResultMarker lines;
// the Go side grades them (classifyHeal). It mutates nothing — the purge is a separate,
// guarded step (HealPurgeScript).
func HealDetectScript(name string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }
	nameLit := kids.MString(name)
	w(`S U="^"`)
	w(`S XPDA=$O(^XPD(9.7,"B",` + nameLit + `,0))`)
	w(`W "` + ResultMarker + `ien=",+XPDA,!`)
	w(`W "` + ResultMarker + `zero=",$S($D(^XPD(9.7,+XPDA,0)):1,1:0),!`)
	w(`W "` + ResultMarker + `status=",$P($G(^XPD(9.7,+XPDA,0)),U,9),!`)
	return b.String()
}

// HealPurgeScript returns M source that purges a PROVEN-corrupt #9.7 entry by IEN so
// a clean reinstall can proceed: the entry subtree (which carries the "ASP"/"INI"/
// "INIT" subnodes a half-install wrote), the "B" + "ASP" cross-references, and the
// staged ^XTMP("XPDI",ien) transport global (§7.1's documented manual purge — DIK
// cannot clean an entry whose 0-node is gone). Heal is a TARGETED repair, never a
// blanket force-delete: the script re-confirms corruption engine-side and REFUSES a
// healthy (status 3) entry, so a healthy install can never be purged even if the Go
// side were wrong about the state. Removing a healthy install is uninstall's job.
func HealPurgeScript(name string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }
	nameLit := kids.MString(name)
	w(`S U="^",DUZ=1,DUZ(0)="@"`)
	w(`S XPDA=$O(^XPD(9.7,"B",` + nameLit + `,0))`)
	// Nothing to heal — no entry at all.
	w(`I 'XPDA W "` + ResultMarker + `healed=0",! Q`)
	// Defense in depth: never purge a healthy install (0-node present AND status 3).
	w(`I $D(^XPD(9.7,XPDA,0)),$P(^XPD(9.7,XPDA,0),U,9)=3 W "` + ResultMarker + `error=healthy-refused",! Q`)
	w(`K ^XPD(9.7,XPDA)`)
	w(`K ^XPD(9.7,"B",` + nameLit + `,XPDA)`)
	w(`K ^XPD(9.7,"ASP",XPDA)`)
	w(`K ^XTMP("XPDI",XPDA)`)
	w(`W "` + ResultMarker + `healed=1",!`)
	return b.String()
}

// VerifyScript returns M source that reports whether name is installed: the
// #9.7 INSTALL presence + status (piece 9; 3 = "Install Completed"), per routine
// whether it is loaded ($T probe), per entry COMPONENT whether a record by that
// .01 NAME is present in its storage file's "B" index, and per FileMan FILE
// whether its data dictionary installed (^DD(file,0) present). Each fact is a
// ResultMarker line. The component probes are driven generically from the
// kids.Component registry (one `comp:<file>:<name>` marker per shipped record),
// so a new component type needs no change here.
func VerifyScript(name string, routines []string, comps []kids.Component, files []string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }

	w(`S U="^"`)
	w(`S XPDA=$O(^XPD(9.7,"B",` + kids.MString(name) + `,0))`)
	w(`W "` + ResultMarker + `installed=",$S(+XPDA:1,1:0),!`)
	w(`W "` + ResultMarker + `status=",$P($G(^XPD(9.7,+XPDA,0)),U,9),!`)
	for _, r := range routines {
		// The routine NAME is build-controlled, so it must never be spliced into the
		// script as raw M — it reaches the line ONLY inside escaped M string literals
		// (kids.MString). The existence probe uses entryref indirection ($T(@VRN)) so
		// the name is a string operand, not code: a name carrying a `"` cannot break
		// out of the literal. Equivalent to the old `$T(^<r>)]""` (first line present).
		rl := kids.MString(r)
		w(`S VRN="+0^"_` + rl)
		w(`W "` + ResultMarker + `rtn:",` + rl + `,"=",$S($T(@VRN)]"":1,1:0),!`)
	}
	// Per entry component: a record by this .01 NAME is present iff it has a
	// "B"-index entry under the type's storage global (KRN^XPDIK builds it on
	// install). The "B" subscript + name hang off the type's DataRoot.
	for _, c := range comps {
		for _, n := range c.Names {
			// c.FileStr/c.DataRoot are registry-fixed (safe); the .01 NAME n is
			// build-controlled, so it is written as an escaped value (kids.MString),
			// never as raw literal text — both the label and the "B" subscript.
			nl := kids.MString(n)
			w(`W "` + ResultMarker + `comp:` + c.FileStr + `:",` + nl + `,"=",$S($D(` + c.DataRoot + `"B",` + nl + `)):1,1:0),!`)
		}
	}
	for _, f := range files {
		w(`W "` + ResultMarker + `file:` + f + `=",$S($D(^DD(` + f + `,0)):1,1:0),!`)
	}
	return b.String()
}

// DeregisterScript returns M that REMOVES the PACKAGE #9.4 patch-history footprint a
// prior `install --register-package` stamped — the inverse of FinalInstallScript's
// reg block, so $$PATCH^XPDUTL no longer reports the patch after a back-out. It
// finds the package by PREFIX (the "C" xref), the VERSION (#9.49) by value, and the
// PATCH APPLICATION HISTORY (#9.4901) entry by value, then FileMan-DIKs that entry
// (which also clears its "B" xref). It deliberately LEAVES the VERSION + package
// entries intact — they may carry other patches or be national, and KIDS itself
// never removes a package; the confirmed gap is only the $$PATCH ghost. A nil reg,
// or one with no patch (no patch-history entry exists), writes nothing.
func DeregisterScript(reg *PkgReg) string {
	if reg == nil || reg.Patch == "" {
		return ""
	}
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }
	w(`S U="^",DUZ=1,DUZ(0)="@"`)
	w(`N XPDPDA,XPDPV,XPDPP,DA,DIK`)
	w(`S XPDPDA=+$O(^DIC(9.4,"C",` + kids.MString(reg.Prefix) + `,0))`)
	w(`S XPDPV=$S(XPDPDA:+$O(^DIC(9.4,XPDPDA,22,"B",` + kids.MString(reg.Version) + `,0)),1:0)`)
	w(`S XPDPP=$S(XPDPV:+$O(^DIC(9.4,XPDPDA,22,XPDPV,"PAH","B",` + kids.MString(reg.Patch) + `,0)),1:0)`)
	w(`I XPDPP S DA(2)=XPDPDA,DA(1)=XPDPV,DA=XPDPP,DIK="^DIC(9.4,"_XPDPDA_",22,"_XPDPV_","_$C(34)_"PAH"_$C(34)_"," D ^DIK`)
	w(`W "` + ResultMarker + `dereg=",$S(XPDPP:1,1:0),!`)
	return b.String()
}

// VerifyContentScript returns M that reads back the LIVE 0-node of each shipped
// KRN entry record so `verify` can assert CONTENT, not just presence: per record
// it resolves the site IEN via the data file's "B" index, then writes the stored
// 0-node as a `z:<file>:<name>` marker (empty when the record is absent). The Go
// side compares each against the shipped image (kids.ZeroMatch), skipping the
// FileMan-transformed pieces. Reading the literal stored node (not the DBS API) is
// deliberate: it is the same image the KRN transport shipped, so a byte difference
// is a real filing fault.
func VerifyContentScript(contents []kids.EntryContent, files []kids.FileContent) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }
	for _, c := range contents {
		nameLit := kids.MString(c.Name)
		// VIEN = the site IEN for this name; VR = the 0-node ref built by indirection
		// (the data global root is per-type, so the node is assembled at runtime).
		w(`S VIEN=+$O(` + c.DataRoot + `"B",` + nameLit + `,0))`)
		w(`S VR="` + c.DataRoot + `"_VIEN_",0)"`)
		// c.Name is build-controlled — write it as an escaped value (nameLit), not raw
		// literal text, so a name carrying a `"` cannot break out into executable M.
		w(`W "` + ResultMarker + `z:` + c.FileStr + `:",` + nameLit + `,"=",$S(VIEN:$G(@VR),1:""),!`)
	}
	// FILE DD content: read each shipped field's ^DD(file,fld,0) definition node
	// back (filed verbatim by DDIN^DIFROMS, so it matches the shipped fieldDef). The
	// field number is a canonical M numeric literal, so it addresses the live node
	// directly — no indirection needed.
	for _, f := range files {
		w(`W "` + ResultMarker + `dd:` + f.FileStr + `#` + f.Field + `=",$G(^DD(` + f.FileStr + `,` + f.Field + `,0)),!`)
	}
	return b.String()
}

// UninstallScript returns M source that reverses an install (T0a.4): delete each
// routine (^%ZOSF("DEL") removes the .m + .o), each entry COMPONENT record from
// its storage file (FileMan DIK by the IEN resolved from the "B" index — clears
// its xrefs and any compiled subfiles), each FileMan FILE (its DD, data global,
// and dict-of-files pointer — KIDS ships no generic file uninstall), and the #9.7
// INSTALL and #9.6 BUILD entries via DIK. The component deletes are driven
// generically from the kids.Component registry (one `^DIK` per shipped record,
// keyed on the type's storage global), so a new component type needs no change
// here — closing the orphan-on-uninstall gap for every registered type. The
// monotonic #9.x and per-file FileMan IEN counters are not rolled back (inherent
// to FileMan, not a leak).
func UninstallScript(name string, routines []string, comps []kids.Component, files []string) string {
	var b strings.Builder
	w := func(line string) { b.WriteString(line); b.WriteByte('\n') }

	w(`S U="^",DUZ=1,DUZ(0)="@"`)
	for _, r := range routines {
		w(`S X=` + kids.MString(r) + ` X ^%ZOSF("DEL")`)
	}
	// Per entry component: resolve the site IEN from the type's "B" index and DIK it
	// by IEN (which also clears its cross-references and compiled subfiles). The "B"
	// lookup root and the DIK ref are both the type's DataRoot.
	for _, c := range comps {
		for _, n := range c.Names {
			nameLit := kids.MString(n)
			w(`S DA=$O(` + c.DataRoot + `"B",` + nameLit + `,0)),DIK="` + c.DataRoot + `" I DA D ^DIK`)
		}
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
