package installspec

import (
	"strings"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// zzskelPairs builds the same ZZSKEL transport-global pairs `v pkg build`
// produces, so the generated install script can be checked against the live-
// proven sequence (kids-installation-automation.md §7.1).
func zzskelPairs() []kids.Pair {
	return kids.MakeBuildPairs(kids.BuildInput{
		InstallName: "ZZSKEL*1.0*1",
		Namespace:   "ZZSKEL",
		Routines: []kids.RoutineSrc{{
			Name: "ZZSKEL",
			Lines: []string{
				"ZZSKEL ;VCD/VSL - throwaway test package (M0a ZZSKEL) ;1.0",
				" ;;1.0;ZZSKEL;;",
				" quit",
				"PING() ;",
				` quit "pong"`,
			},
		}},
	})
}

// StageChunks splits the transport global into size-bounded routine bodies that
// populate the ^XTMP("VPKGI","tok",…) staging global — never one routine big enough to
// exceed the driver's per-routine staging limit (which silently truncates large
// loads, T0b.2 discoveries P1).
func TestStageChunks_CoversAllPairsBounded(t *testing.T) {
	pairs := zzskelPairs()
	const maxBytes = 80
	chunks := StageChunks(pairs, maxBytes, "tok")
	if len(chunks) < 2 {
		t.Fatalf("want multiple chunks at maxBytes=%d, got %d", maxBytes, len(chunks))
	}
	// The first chunk clears any stale staging global before populating it.
	if !strings.HasPrefix(chunks[0], `K ^XTMP("VPKGI","tok")`) {
		t.Errorf("first chunk must clear the staging global, got:\n%s", chunks[0])
	}
	all := strings.Join(chunks, "\n")
	// Every pair is staged exactly once, into ^XTMP("VPKGI","tok",…) (not the live
	// ^XTMP("XPDI",XPDA,…) — that is filled by the finalize MERGE).
	for _, p := range pairs {
		ref := "S " + p.Subs.MRef(`^XTMP("VPKGI","tok",`) + "="
		if n := strings.Count(all, ref); n != 1 {
			t.Errorf("pair %q staged %d times, want 1", ref, n)
		}
	}
	// Embedded quotes in routine source are doubled.
	if !strings.Contains(all, `S ^XTMP("VPKGI","tok","RTN","ZZSKEL",5,0)=" quit ""pong"""`) {
		t.Errorf("RTN line 5 not M-escaped:\n%s", all)
	}
	// Each chunk stays bounded (a lone over-long SET may exceed maxBytes, but a
	// chunk never accumulates well past it).
	for i, c := range chunks {
		if len(c) > maxBytes+256 {
			t.Errorf("chunk %d = %d bytes, exceeds the bound", i, len(c))
		}
	}
}

// FinalInstallScript verifies the staged count, then installs in one process:
// INST → MERGE the staged tree into ^XTMP("XPDI",XPDA) → EN^XPDIJ.
func TestFinalInstallScript(t *testing.T) {
	got := FinalInstallScript("ZZSKEL*1.0*1", "ZZSKEL via v pkg install", 7, true, nil, nil, "tok")
	for _, want := range []string{
		`I $D(^XPD(9.7,"B","ZZSKEL*1.0*1"))`,            // already-installed guard
		`DUZ(0)="@"`,                                    // full FM priv for EN^XPDIJ
		ResultMarker + `staged=`,                        // staged-node count marker
		`I VC'=7`,                                       // truncation guard: count must match
		ResultMarker + `error=stage-incomplete`,         // …else refuse, do not install partial
		`S XPDA=$$INST^XPDIL1("ZZSKEL*1.0*1")`,          // real KIDS #9.7 entry
		`M ^XTMP("XPDI",XPDA)=^XTMP("VPKGI","tok")`,     // staged tree → live transport global
		`^XPD(9.7,XPDA,"KRN")=^XTMP("XPDI",XPDA,"BLD",`, // seed #9.7 KRN tracking (else XPDIK GVUNDEF)
		`^XTMP("XPDI",XPDA,"FIA",`,                      // FileMan FILE: seed new-file flag …,0,2)
		`XPCK^XPDIK("FIA")`,                             // …and the #9.7 FILE checkpoint (else FIA^XPDIK GVUNDEF)
		`D EN^XPDIJ`,                                    // synchronous install (same process)
		`K ^XTMP("VPKGI","tok")`,                        // clean the staging global
		ResultMarker + `status:tok=`,                    // #9.7 status marker (nonce-tagged)
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FinalInstallScript missing %q\n---\n%s", want, got)
		}
	}
}

// The transport node must be KILLed before the staged tree is MERGEd into it: a
// purged earlier install can free its #9.7 IEN, $$INST^XPDIL1 re-assigns it, and
// stale ^XTMP("XPDI",IEN,…) (e.g. a prior build's REQB nodes) would otherwise
// survive the MERGE and corrupt env-check / Required-Build enforcement. The real
// KIDS load always starts from a clean node. (Live-proven: a stale REQB made a
// hook-only build's env-check return 2 until the KILL was added.)
func TestFinalInstallScript_KillBeforeMerge(t *testing.T) {
	got := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, nil, nil, "tok")
	kill := strings.Index(got, `K ^XTMP("XPDI",XPDA)`)
	merge := strings.Index(got, `M ^XTMP("XPDI",XPDA)=^XTMP("VPKGI","tok")`)
	if kill < 0 {
		t.Fatalf("FinalInstallScript must KILL ^XTMP(\"XPDI\",XPDA) before the MERGE\n---\n%s", got)
	}
	if kill > merge {
		t.Errorf("KILL of the transport node (%d) must precede the MERGE (%d)", kill, merge)
	}
}

// A.3: with a package registration, after EN^XPDIJ the script must write the #9.4
// PACKAGE footprint — find-or-create the #9.4 entry by PREFIX, then $$PKGVER^XPDIP
// (VERSION multiple + CURRENT VERSION) and $$PKGPAT^XPDIP (PATCH APPLICATION
// HISTORY) — so downstream $$VER/$$PATCH^XPDUTL see the install.
func TestFinalInstallScript_PackageFootprint(t *testing.T) {
	reg := &PkgReg{Prefix: "ZZP", Name: "ZZ DEMO PACKAGE", Version: "1.0", Patch: "1"}
	got := FinalInstallScript("ZZP*1.0*1", "hdr", 7, true, nil, reg, "tok")
	for _, want := range []string{
		`^DIC(9.4,"C","ZZP",`,                     // resolve the package by PREFIX
		`XPDFDA(9.4,"+1,",.01)="ZZ DEMO PACKAGE"`, // create #9.4 entry (NAME) when absent
		`XPDFDA(9.4,"+1,",1)="ZZP"`,               // …with PREFIX
		`UPDATE^DIE(`,                             // file the new #9.4 entry
		`$$PKGVER^XPDIP(`,                         // VERSION multiple + CURRENT VERSION
		`$$PKGPAT^XPDIP(`,                         // PATCH APPLICATION HISTORY
		ResultMarker + `pkg=`,                     // footprint result marker
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FinalInstallScript(reg) missing %q\n---\n%s", want, got)
		}
	}
	// The footprint must run AFTER the build is filed (EN^XPDIJ).
	if strings.Index(got, `D EN^XPDIJ`) > strings.Index(got, `$$PKGVER^XPDIP(`) {
		t.Error("package footprint must come after D EN^XPDIJ")
	}
	// Without a registration, no footprint code leaks in.
	plain := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, nil, nil, "tok")
	if strings.Contains(plain, `$$PKGVER^XPDIP(`) {
		t.Error("a build with no package registration must emit no #9.4 footprint")
	}
}

// A patch-less registration writes the VERSION footprint but skips PATCH history.
func TestFinalInstallScript_PackageFootprint_NoPatch(t *testing.T) {
	reg := &PkgReg{Prefix: "ZZP", Name: "ZZ DEMO PACKAGE", Version: "2.0"}
	got := FinalInstallScript("ZZP*2.0", "hdr", 7, true, nil, reg, "tok")
	if !strings.Contains(got, `$$PKGVER^XPDIP(`) {
		t.Error("patch-less registration must still write the VERSION footprint")
	}
	if strings.Contains(got, `$$PKGPAT^XPDIP(`) {
		t.Error("patch-less registration must not write PATCH history")
	}
}

// A.1.1: before EN^XPDIJ, the script must create the INI/INIT checkpoints so the
// build's pre/post-install routines fire (PRE^/POST^XPDIJ1 D @ the routine read
// from the "...STARTED" checkpoint's callback). Mirrors PKG^XPDIL1: $$NEWCP^XPDUTL
// for the COMPLETED checkpoint, then the STARTED one (carrying the routine) only
// when the transport carries a pre/post routine name.
func TestFinalInstallScript_PrePostCheckpoints(t *testing.T) {
	got := FinalInstallScript("ZZSKEL*1.0*1", "ZZSKEL via v pkg install", 7, true, nil, nil, "tok")
	for _, want := range []string{
		`S XPDCP="INI"`, // pre-install checkpoint multiple (subfile 9.713)
		`$$NEWCP^XPDUTL("XPD PREINSTALL COMPLETED")`, // base checkpoint
		`$G(^XTMP("XPDI",XPDA,"INI"))`,               // pre-install routine name from the transport
		`$$NEWCP^XPDUTL("XPD PREINSTALL STARTED",`,   // routine-bearing checkpoint
		`S XPDCP="INIT"`,                             // post-install checkpoint multiple (subfile 9.716)
		`$$NEWCP^XPDUTL("XPD POSTINSTALL COMPLETED")`,
		`$G(^XTMP("XPDI",XPDA,"INIT"))`,
		`$$NEWCP^XPDUTL("XPD POSTINSTALL STARTED",`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FinalInstallScript missing %q\n---\n%s", want, got)
		}
	}
	// The checkpoints must be created BEFORE the install runs them.
	if iCP, iJ := strings.Index(got, `XPD PREINSTALL STARTED`), strings.Index(got, `D EN^XPDIJ`); iCP < 0 || iJ < 0 || iCP > iJ {
		t.Errorf("checkpoint creation (%d) must precede EN^XPDIJ (%d)", iCP, iJ)
	}
}

// A.1.2: with runEnvCheck=true the script must reconstruct the install-phase
// scope (XPDNM/XPDPKG + the seeded XPDT, without which ENV's tail self-rejects a
// clean build) and call the REAL $$ENV^XPDIL1(1) before filing, refusing to file
// on a non-zero reject. With runEnvCheck=false it must emit none of that (the
// restore/back-out callers must not re-run env-check).
func TestFinalInstallScript_EnvCheck(t *testing.T) {
	on := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, nil, nil, "tok")
	for _, want := range []string{
		`S XPDT(XPDIT)=XPDA_U_XPDNM`,                 // seed XPDT (else ENV self-rejects)
		`XPDPKG=+$P($G(^XPD(9.7,XPDA,0)),U,2)`,       // package pointer for the version check
		`S XPDENV=1,XPDENVRC=$$ENV^XPDIL1(1)`,        // the REAL env-check + REQB call
		ResultMarker + `error=env-check-rejected^`,   // refuse-to-file marker on reject
		`K ^XPD(9.7,"B",XPDNM,XPDA),^XPD(9.7,XPDA),`, // purge the aborted #9.7 entry on reject
	} {
		if !strings.Contains(on, want) {
			t.Errorf("runEnvCheck=true: FinalInstallScript missing %q\n---\n%s", want, on)
		}
	}
	// Env-check must run BEFORE filing.
	if iE, iJ := strings.Index(on, `$$ENV^XPDIL1(1)`), strings.Index(on, `D EN^XPDIJ`); iE < 0 || iJ < 0 || iE > iJ {
		t.Errorf("env-check (%d) must precede EN^XPDIJ (%d)", iE, iJ)
	}
	off := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, false, nil, nil, "tok")
	if strings.Contains(off, `$$ENV^XPDIL1`) {
		t.Errorf("runEnvCheck=false must NOT call $$ENV^XPDIL1\n---\n%s", off)
	}
}

// A.1.3: pre-answered install questions are seeded into the #9.7 "QUES" scratch
// tree — the three nodes $$ANSWER^XPDIQ reads (name .01, answer node 1, "B" xref),
// IEN per answer — BEFORE EN^XPDIJ runs the pre/post routines that read them. No
// answers → none of those sets (byte-identical to the pre-A.1.3 script).
func TestFinalInstallScript_QuesAnswers(t *testing.T) {
	got := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, []QuesAnswer{
		{Name: "ZZ4Q", Value: "HELLO"},
		{Name: "RUN MODE", Value: "1"},
	}, nil, "tok")
	for _, want := range []string{
		`S ^XPD(9.7,XPDA,"QUES",1,0)="ZZ4Q",^XPD(9.7,XPDA,"QUES",1,1)="HELLO",^XPD(9.7,XPDA,"QUES","B","ZZ4Q",1)=""`,
		`S ^XPD(9.7,XPDA,"QUES",2,0)="RUN MODE",^XPD(9.7,XPDA,"QUES",2,1)="1",^XPD(9.7,XPDA,"QUES","B","RUN MODE",2)=""`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FinalInstallScript missing QUES seed %q\n---\n%s", want, got)
		}
	}
	// The answers must be seeded BEFORE the install runs the pre/post routines.
	if iQ, iJ := strings.Index(got, `"QUES",1,0)`), strings.Index(got, `D EN^XPDIJ`); iQ < 0 || iJ < 0 || iQ > iJ {
		t.Errorf("QUES seed (%d) must precede EN^XPDIJ (%d)", iQ, iJ)
	}
	// No answers → no QUES sets at all.
	if none := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, nil, nil, "tok"); strings.Contains(none, `"QUES"`) {
		t.Errorf("hook-free build must not seed QUES\n---\n%s", none)
	}
}

// HealDetectScript is a READ-ONLY probe: it emits the #9.7 IEN, whether the entry
// has a 0-node, and the install status (piece 9) so the Go side can grade the entry
// (classifyHeal) without mutating anything.
func TestHealDetectScript(t *testing.T) {
	got := HealDetectScript("A2AP*1.0*0")
	for _, want := range []string{
		`S XPDA=$O(^XPD(9.7,"B","A2AP*1.0*0",0))`, // resolve the IEN from the "B" xref
		ResultMarker + `ien=`,                     // the IEN (0 = not present)
		`$D(^XPD(9.7,+XPDA,0))`,                   // 0-node presence test
		ResultMarker + `zero=`,                    // 0-node present? (the corrupt signal)
		ResultMarker + `status=`,                  // #9.7 status piece 9
	} {
		if !strings.Contains(got, want) {
			t.Errorf("HealDetectScript missing %q\n---\n%s", want, got)
		}
	}
	// It must be read-only — no KILL of the #9.7 entry or transport global.
	for _, forbidden := range []string{`K ^XPD(9.7`, `K ^XTMP`} {
		if strings.Contains(got, forbidden) {
			t.Errorf("HealDetectScript must be read-only, found %q\n---\n%s", forbidden, got)
		}
	}
}

// HealPurgeScript purges a PROVEN-corrupt #9.7 entry by IEN — the entry subtree, the
// "B" and "ASP" cross-references, and the staged ^XTMP("XPDI",ien) transport — so a
// clean reinstall can proceed (kids-installation-automation.md §7.1). Defense in
// depth: it re-confirms corruption engine-side and REFUSES a healthy (status 3)
// entry, so it can never blanket-delete a healthy install.
func TestHealPurgeScript(t *testing.T) {
	got := HealPurgeScript("A2AP*1.0*0")
	for _, want := range []string{
		`DUZ(0)="@"`, // FM priv for the global KILLs
		`S XPDA=$O(^XPD(9.7,"B","A2AP*1.0*0",0))`, // resolve the IEN
		`$P(^XPD(9.7,XPDA,0),U,9)=3`,              // re-confirm: a status-3 entry is healthy
		ResultMarker + `error=healthy-refused`,    // …refuse to purge it
		`K ^XPD(9.7,XPDA)`,                        // purge the entry subtree (ASP/INI/INIT subnodes)
		`K ^XPD(9.7,"B","A2AP*1.0*0",XPDA)`,       // the "B" xref
		`K ^XPD(9.7,"ASP",XPDA)`,                  // the "ASP" install-sequence xref
		`K ^XTMP("XPDI",XPDA)`,                    // the staged transport global
		ResultMarker + `healed=`,                  // outcome marker
	} {
		if !strings.Contains(got, want) {
			t.Errorf("HealPurgeScript missing %q\n---\n%s", want, got)
		}
	}
	// The healthy-refused guard must come BEFORE any KILL.
	if iG, iK := strings.Index(got, `error=healthy-refused`), strings.Index(got, `K ^XPD(9.7,XPDA)`); iG < 0 || iK < 0 || iG > iK {
		t.Errorf("healthy-refused guard (%d) must precede the purge (%d)", iG, iK)
	}
}

func TestVerifyScript(t *testing.T) {
	got := VerifyScript("ZZSKEL*1.0*1", []string{"ZZSKEL"}, nil, nil)
	for _, want := range []string{
		`S XPDA=$O(^XPD(9.7,"B","ZZSKEL*1.0*1",0))`,
		ResultMarker + `installed=`,
		ResultMarker + `status=`,
		`S VRN="+0^"_"ZZSKEL"`,          // routine name reaches the script only inside an M literal
		`$T(@VRN)`,                      // injection-safe indirection probe (was $T(^ZZSKEL))
		ResultMarker + `rtn:","ZZSKEL"`, // per-routine marker, name emitted as an escaped value
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript missing %q\n---\n%s", want, got)
		}
	}
}

// componentCases ground the registry-driven generic component path against every
// standard component global: each must presence-probe its "B" index and DIK-delete
// by IEN from its own storage global. A new type becomes one row here, no new code.
var componentCases = []struct {
	file                           float64
	fileStr, dataRoot, label, name string
}{
	{8989.51, "8989.51", "^XTV(8989.51,", "PARAMETER DEFINITION", "VSL GREETING"},
	{19, "19", "^DIC(19,", "OPTION", "ZZOPT RUN ROUTINE"},
	{19.1, "19.1", "^DIC(19.1,", "SECURITY KEY", "ZZKEY MANAGER"},
	{101, "101", "^ORD(101,", "PROTOCOL", "ZZPROTO ACTION"},
	{8994, "8994", "^XWB(8994,", "REMOTE PROCEDURE", "ZZRPC ECHO"},
	{3.8, "3.8", "^XMB(3.8,", "MAIL GROUP", "ZZMG ALERTS"},
	{409.61, "409.61", "^SD(409.61,", "LIST TEMPLATE", "ZZLM PATIENTS"},
	{9.2, "9.2", "^DIC(9.2,", "HELP FRAME", "ZZHF-MAIN"},
	{771, "771", "^HL(771,", "HL7 APPLICATION PARAMETER", "ZZHL_APP"},
	{779.2, "779.2", "^HLD(779.2,", "HLO APPLICATION REGISTRY", "ZZHO_APP"},
	{870, "870", "^HLCS(870,", "HL LOGICAL LINK", "ZZLINK"},
	// New types (increment #1) ride the SAME generic path — no per-type code.
	{0.402, "0.402", "^DIE(", "INPUT TEMPLATE", "ZZTMPL FILE #999000"},
	{0.403, "0.403", "^DIST(.403,", "FORM", "ZZFORM"},
}

func componentLit(i int) []kids.Component {
	c := componentCases[i]
	return []kids.Component{{File: c.file, FileStr: c.fileStr, DataRoot: c.dataRoot, Label: c.label, Names: []string{c.name}}}
}

// msliteral renders an M string literal the way kids.MString does, for the probe
// assertions (quote-wrapped, embedded quotes doubled).
func msliteral(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// VerifyScript probes each entry component by NAME in its storage file's "B"
// index, driven generically from the kids.Component registry — one
// comp:<file>:<name> marker per record.
func TestVerifyScript_Components(t *testing.T) {
	for i, tc := range componentCases {
		got := VerifyScript("ZZ*1.0*1", []string{"ZZRT"}, componentLit(i), nil)
		probe := `$D(` + tc.dataRoot + `"B",` + msliteral(tc.name) + `)`
		marker := ResultMarker + `comp:` + tc.fileStr + `:",` + msliteral(tc.name) // name emitted as an escaped value
		if !strings.Contains(got, probe) {
			t.Errorf("%s: VerifyScript missing presence probe %q\n---\n%s", tc.label, probe, got)
		}
		if !strings.Contains(got, marker) {
			t.Errorf("%s: VerifyScript missing marker %q\n---\n%s", tc.label, marker, got)
		}
	}
}

// VerifyContentScript reads back the LIVE 0-node of each shipped KRN entry record
// (resolving its site IEN via the data file's "B" index) so verify can assert
// content, not just presence — one z:<file>:<name> marker per record.
func TestVerifyContentScript(t *testing.T) {
	got := VerifyContentScript([]kids.EntryContent{
		{File: 19, FileStr: "19", Name: "ZZOPT RUN ROUTINE", DataRoot: "^DIC(19,", Zero: "ZZOPT RUN ROUTINE^x^^R"},
	}, nil)
	for _, want := range []string{
		`$O(^DIC(19,"B","ZZOPT RUN ROUTINE",0))`,    // resolve site IEN by name
		`S VR="^DIC(19,"_VIEN_",0)"`,                // build the 0-node ref by indirection
		ResultMarker + `z:19:","ZZOPT RUN ROUTINE"`, // emit the live 0-node; name as escaped value
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyContentScript missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyContentScript also reads each shipped FILE field's ^DD(file,fld,0)
// definition node back so verify can assert DD content, not just file presence.
func TestVerifyContentScript_File(t *testing.T) {
	got := VerifyContentScript(nil, []kids.FileContent{
		{File: 999001, FileStr: "999001", Field: "1", Zero: `USER NUMBER^NJ12,0^^0;2^K:+X'=X X`},
		{File: 999001, FileStr: "999001", Field: "0.01", Zero: "NAME^RF^^0;1^K:$L(X)>30 X"},
	})
	for _, want := range []string{
		`$G(^DD(999001,1,0))`,         // read the live field-def node
		`$G(^DD(999001,0.01,0))`,      // .01 renders as a canonical M numeric literal
		ResultMarker + `dd:999001#1=`, // emit per field
		ResultMarker + `dd:999001#0.01=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyContentScript (file) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each installed FileMan FILE by its data dictionary
// header node (^DD(file,0) present iff the DD installed).
func TestVerifyScript_File(t *testing.T) {
	got := VerifyScript("ZZVSLFS*1.0*1", nil, nil, []string{"999000"})
	for _, want := range []string{
		`$D(^DD(999000,0))`,
		ResultMarker + `file:999000=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (file) missing %q\n---\n%s", want, got)
		}
	}
}

func TestUninstallScript(t *testing.T) {
	got := UninstallScript("ZZSKEL*1.0*1", []string{"ZZSKEL"}, nil, nil)
	for _, want := range []string{
		`S X="ZZSKEL" X ^%ZOSF("DEL")`,                                        // routine delete
		`S DA=$O(^XPD(9.7,"B","ZZSKEL*1.0*1",0)),DIK="^XPD(9.7," I DA D ^DIK`, // #9.7
		`S DA=$O(^XPD(9.6,"B","ZZSKEL*1.0*1",0)),DIK="^XPD(9.6," I DA D ^DIK`, // #9.6
		ResultMarker + `uninstalled=1`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("UninstallScript missing %q\n---\n%s", want, got)
		}
	}
}

// UninstallScript backs out each entry component via FileMan DIK on its storage
// file (delete by the IEN resolved from the "B" index; DIK clears its xrefs and
// any compiled subfiles) — driven generically from the kids.Component registry.
func TestUninstallScript_Components(t *testing.T) {
	for i, tc := range componentCases {
		got := UninstallScript("ZZ*1.0*1", []string{"ZZRT"}, componentLit(i), nil)
		want := `S DA=$O(` + tc.dataRoot + `"B",` + msliteral(tc.name) + `,0)),DIK="` + tc.dataRoot + `" I DA D ^DIK`
		if !strings.Contains(got, want) {
			t.Errorf("%s: UninstallScript missing %q\n---\n%s", tc.label, want, got)
		}
	}
}

// UninstallScript also backs out each FileMan FILE: it reads the data global root
// + name from ^DIC before killing the DD (^DD/^DIC), the data global, and the
// dict-of-files "B" pointer — KIDS ships no generic file uninstall.
func TestUninstallScript_File(t *testing.T) {
	got := UninstallScript("ZZVSLFS*1.0*1", nil, nil, []string{"999000"})
	for _, want := range []string{
		`^DIC(999000,0,"GL")`,        // read the data global root first
		`K ^DD(999000),^DIC(999000)`, // remove the data dictionary
		`,"B",`,                      // remove the dict-of-files "B" pointer
	} {
		if !strings.Contains(got, want) {
			t.Errorf("UninstallScript (file) missing %q\n---\n%s", want, got)
		}
	}
}

// DeregisterScript removes the PACKAGE #9.4 patch-history footprint a prior
// install --register-package stamped (the inverse of the FinalInstallScript reg
// block): find package by PREFIX, VERSION #9.49 by value, PATCH APPLICATION
// HISTORY #9.4901 by value, then FileMan-DIK that entry.
func TestDeregisterScript(t *testing.T) {
	got := DeregisterScript(&PkgReg{Prefix: "MSL", Version: "0.1", Patch: "1"})
	for _, want := range []string{
		`$O(^DIC(9.4,"C","MSL",0))`, // find package by PREFIX ("C" xref)
		`,22,"B","0.1",`,            // find VERSION #9.49 by value
		`,"PAH","B","1",`,           // find PATCH APPLICATION HISTORY #9.4901 by value
		`D ^DIK`,                    // DIK the patch-history entry (clears its "B")
		ResultMarker + `dereg=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DeregisterScript missing %q\n---\n%s", want, got)
		}
	}
	// A patchless registration has no patch-history entry to clear; nil → nothing.
	if s := DeregisterScript(&PkgReg{Prefix: "MSL", Version: "0.1"}); s != "" {
		t.Errorf("patchless DeregisterScript = %q, want empty", s)
	}
	if s := DeregisterScript(nil); s != "" {
		t.Errorf("nil DeregisterScript = %q, want empty", s)
	}
}

// TestVerifyScripts_NoMInjection is the adversarial guard for the verify path: a
// crafted .KID can carry a routine/component/entry NAME containing a double-quote
// plus arbitrary M. Those names must reach the generated script ONLY inside escaped
// M string literals (via kids.MString), never as raw code — otherwise running
// `v pkg verify <untrusted.kid>` would execute attacker M in programmer context.
// Invariant: every generated line is balanced (even number of `"`); a raw splice of
// a name with one `"` makes the line odd (a literal breakout).
func TestVerifyScripts_NoMInjection(t *testing.T) {
	evil := `Z" S ^HACKED=1 W "` // an unbalanced quote + payload + reopen
	comps := []kids.Component{{File: 19, FileStr: "19", DataRoot: `^DIC(19,`, Names: []string{evil}}}
	contents := []kids.EntryContent{{File: 19, FileStr: "19", DataRoot: `^DIC(19,`, Name: evil}}
	scripts := map[string]string{
		"VerifyScript/routine":   VerifyScript("PKG*1.0*1", []string{evil}, nil, nil),
		"VerifyScript/component": VerifyScript("PKG*1.0*1", nil, comps, nil),
		"VerifyContentScript":    VerifyContentScript(contents, nil),
	}
	for label, s := range scripts {
		// Decisive invariant: every generated line is quote-balanced. A raw splice of a
		// name carrying one `"` makes its line odd (a literal breakout); MString doubles
		// the quote, keeping the line even.
		for i, line := range strings.Split(s, "\n") {
			if strings.Count(line, `"`)%2 != 0 {
				t.Errorf("%s line %d: unbalanced quotes — M-injection breakout: %q", label, i+1, line)
			}
		}
		// Positive confirmation: the evil name appears only in its MString-escaped form.
		if !strings.Contains(s, kids.MString(evil)) {
			t.Errorf("%s: the name was not emitted via kids.MString (escaping bypassed)", label)
		}
	}
}
