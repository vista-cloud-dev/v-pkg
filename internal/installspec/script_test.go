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
// populate the ^XTMP("VPKGI",…) staging global — never one routine big enough to
// exceed the driver's per-routine staging limit (which silently truncates large
// loads, T0b.2 discoveries P1).
func TestStageChunks_CoversAllPairsBounded(t *testing.T) {
	pairs := zzskelPairs()
	const maxBytes = 80
	chunks := StageChunks(pairs, maxBytes)
	if len(chunks) < 2 {
		t.Fatalf("want multiple chunks at maxBytes=%d, got %d", maxBytes, len(chunks))
	}
	// The first chunk clears any stale staging global before populating it.
	if !strings.HasPrefix(chunks[0], `K ^XTMP("VPKGI")`) {
		t.Errorf("first chunk must clear the staging global, got:\n%s", chunks[0])
	}
	all := strings.Join(chunks, "\n")
	// Every pair is staged exactly once, into ^XTMP("VPKGI",…) (not the live
	// ^XTMP("XPDI",XPDA,…) — that is filled by the finalize MERGE).
	for _, p := range pairs {
		ref := "S " + p.Subs.MRef(`^XTMP("VPKGI",`) + "="
		if n := strings.Count(all, ref); n != 1 {
			t.Errorf("pair %q staged %d times, want 1", ref, n)
		}
	}
	// Embedded quotes in routine source are doubled.
	if !strings.Contains(all, `S ^XTMP("VPKGI","RTN","ZZSKEL",5,0)=" quit ""pong"""`) {
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
	got := FinalInstallScript("ZZSKEL*1.0*1", "ZZSKEL via v pkg install", 7, true, nil)
	for _, want := range []string{
		`I $D(^XPD(9.7,"B","ZZSKEL*1.0*1"))`,            // already-installed guard
		`DUZ(0)="@"`,                                    // full FM priv for EN^XPDIJ
		ResultMarker + `staged=`,                        // staged-node count marker
		`I VC'=7`,                                       // truncation guard: count must match
		ResultMarker + `error=stage-incomplete`,         // …else refuse, do not install partial
		`S XPDA=$$INST^XPDIL1("ZZSKEL*1.0*1")`,          // real KIDS #9.7 entry
		`M ^XTMP("XPDI",XPDA)=^XTMP("VPKGI")`,           // staged tree → live transport global
		`^XPD(9.7,XPDA,"KRN")=^XTMP("XPDI",XPDA,"BLD",`, // seed #9.7 KRN tracking (else XPDIK GVUNDEF)
		`^XTMP("XPDI",XPDA,"FIA",`,                      // FileMan FILE: seed new-file flag …,0,2)
		`XPCK^XPDIK("FIA")`,                             // …and the #9.7 FILE checkpoint (else FIA^XPDIK GVUNDEF)
		`D EN^XPDIJ`,                                    // synchronous install (same process)
		`K ^XTMP("VPKGI")`,                              // clean the staging global
		ResultMarker + `status=`,                        // #9.7 status marker
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
	got := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, nil)
	kill := strings.Index(got, `K ^XTMP("XPDI",XPDA)`)
	merge := strings.Index(got, `M ^XTMP("XPDI",XPDA)=^XTMP("VPKGI")`)
	if kill < 0 {
		t.Fatalf("FinalInstallScript must KILL ^XTMP(\"XPDI\",XPDA) before the MERGE\n---\n%s", got)
	}
	if kill > merge {
		t.Errorf("KILL of the transport node (%d) must precede the MERGE (%d)", kill, merge)
	}
}

// A.1.1: before EN^XPDIJ, the script must create the INI/INIT checkpoints so the
// build's pre/post-install routines fire (PRE^/POST^XPDIJ1 D @ the routine read
// from the "...STARTED" checkpoint's callback). Mirrors PKG^XPDIL1: $$NEWCP^XPDUTL
// for the COMPLETED checkpoint, then the STARTED one (carrying the routine) only
// when the transport carries a pre/post routine name.
func TestFinalInstallScript_PrePostCheckpoints(t *testing.T) {
	got := FinalInstallScript("ZZSKEL*1.0*1", "ZZSKEL via v pkg install", 7, true, nil)
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
	on := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, nil)
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
	off := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, false, nil)
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
	})
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
	if none := FinalInstallScript("ZZSKEL*1.0*1", "hdr", 7, true, nil); strings.Contains(none, `"QUES"`) {
		t.Errorf("hook-free build must not seed QUES\n---\n%s", none)
	}
}

func TestVerifyScript(t *testing.T) {
	got := VerifyScript("ZZSKEL*1.0*1", []string{"ZZSKEL"}, nil, nil, nil, nil, nil, nil, nil, nil)
	for _, want := range []string{
		`S XPDA=$O(^XPD(9.7,"B","ZZSKEL*1.0*1",0))`,
		ResultMarker + `installed=`,
		ResultMarker + `status=`,
		`$T(^ZZSKEL)`,                // routine-presence probe
		ResultMarker + `rtn:ZZSKEL=`, // per-routine marker
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each PARAMETER DEFINITION by NAME in #8989.51's "B"
// index (XPDIK builds it via IX1^DIK on install).
func TestVerifyScript_ParamDef(t *testing.T) {
	got := VerifyScript("VSLBASE*1.0*1", []string{"VSLCFG"}, []string{"VSL GREETING"}, nil, nil, nil, nil, nil, nil, nil)
	for _, want := range []string{
		`$D(^XTV(8989.51,"B","VSL GREETING"))`,
		ResultMarker + `param:VSL GREETING=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (param) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each OPTION by NAME in #19's "B" index (KRN^XPDIK
// builds it when it files the option record).
func TestVerifyScript_Option(t *testing.T) {
	got := VerifyScript("ZZOPT*1.0*1", []string{"ZZOPTRT"}, nil, []string{"ZZOPT RUN ROUTINE"}, nil, nil, nil, nil, nil, nil)
	for _, want := range []string{
		`$D(^DIC(19,"B","ZZOPT RUN ROUTINE"))`,
		ResultMarker + `option:ZZOPT RUN ROUTINE=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (option) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each SECURITY KEY by NAME in #19.1's "B" index.
func TestVerifyScript_Key(t *testing.T) {
	got := VerifyScript("ZZKEY*1.0*1", []string{"ZZKEYRT"}, nil, nil, []string{"ZZKEY MANAGER"}, nil, nil, nil, nil, nil)
	for _, want := range []string{
		`$D(^DIC(19.1,"B","ZZKEY MANAGER"))`,
		ResultMarker + `key:ZZKEY MANAGER=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (key) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each PROTOCOL by NAME in #101's "B" index (^ORD(101,).
func TestVerifyScript_Protocol(t *testing.T) {
	got := VerifyScript("ZZPROTO*1.0*1", []string{"ZZPRORT"}, nil, nil, nil, []string{"ZZPROTO ACTION"}, nil, nil, nil, nil)
	for _, want := range []string{
		`$D(^ORD(101,"B","ZZPROTO ACTION"))`,
		ResultMarker + `protocol:ZZPROTO ACTION=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (protocol) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each REMOTE PROCEDURE by its #8994 "B" index entry.
func TestVerifyScript_RPC(t *testing.T) {
	got := VerifyScript("ZZRPC*1.0*1", []string{"ZZRPCRT"}, nil, nil, nil, nil, []string{"ZZRPC ECHO"}, nil, nil, nil)
	for _, want := range []string{
		`$D(^XWB(8994,"B","ZZRPC ECHO"))`,
		ResultMarker + `rpc:ZZRPC ECHO=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (rpc) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each MAIL GROUP by its #3.8 "B" index entry.
func TestVerifyScript_MailGroup(t *testing.T) {
	got := VerifyScript("ZZMG*1.0*1", []string{"ZZMGRT"}, nil, nil, nil, nil, nil, []string{"ZZMG ALERTS"}, nil, nil)
	for _, want := range []string{
		`$D(^XMB(3.8,"B","ZZMG ALERTS"))`,
		ResultMarker + `mailgroup:ZZMG ALERTS=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (mail group) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each LIST TEMPLATE by its #409.61 "B" index entry
// (stored under ^SD(409.61,, not ^ORD).
func TestVerifyScript_ListTemplate(t *testing.T) {
	got := VerifyScript("ZZLM*1.0*1", []string{"ZZLMRT"}, nil, nil, nil, nil, nil, nil, []string{"ZZLM PATIENTS"}, nil)
	for _, want := range []string{
		`$D(^SD(409.61,"B","ZZLM PATIENTS"))`,
		ResultMarker + `listtemplate:ZZLM PATIENTS=`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript (list template) missing %q\n---\n%s", want, got)
		}
	}
}

// VerifyScript also probes each installed FileMan FILE by its data dictionary
// header node (^DD(file,0) present iff the DD installed).
func TestVerifyScript_File(t *testing.T) {
	got := VerifyScript("ZZVSLFS*1.0*1", nil, nil, nil, nil, nil, nil, nil, nil, []string{"999000"})
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
	got := UninstallScript("ZZSKEL*1.0*1", []string{"ZZSKEL"}, nil, nil, nil, nil, nil, nil, nil, nil)
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

// UninstallScript also backs out each PARAMETER DEFINITION via FileMan DIK on
// #8989.51 (delete by IEN resolved from the "B" index; DIK clears its xrefs).
func TestUninstallScript_ParamDef(t *testing.T) {
	got := UninstallScript("VSLBASE*1.0*1", []string{"VSLCFG"}, []string{"VSL GREETING"}, nil, nil, nil, nil, nil, nil, nil)
	want := `S DA=$O(^XTV(8989.51,"B","VSL GREETING",0)),DIK="^XTV(8989.51," I DA D ^DIK`
	if !strings.Contains(got, want) {
		t.Errorf("UninstallScript (param) missing %q\n---\n%s", want, got)
	}
}

// UninstallScript also backs out each OPTION via FileMan DIK on #19 (delete by
// IEN resolved from the "B" index; DIK clears its xrefs).
func TestUninstallScript_Option(t *testing.T) {
	got := UninstallScript("ZZOPT*1.0*1", []string{"ZZOPTRT"}, nil, []string{"ZZOPT RUN ROUTINE"}, nil, nil, nil, nil, nil, nil)
	want := `S DA=$O(^DIC(19,"B","ZZOPT RUN ROUTINE",0)),DIK="^DIC(19," I DA D ^DIK`
	if !strings.Contains(got, want) {
		t.Errorf("UninstallScript (option) missing %q\n---\n%s", want, got)
	}
}

// UninstallScript also backs out each SECURITY KEY via FileMan DIK on #19.1.
func TestUninstallScript_Key(t *testing.T) {
	got := UninstallScript("ZZKEY*1.0*1", []string{"ZZKEYRT"}, nil, nil, []string{"ZZKEY MANAGER"}, nil, nil, nil, nil, nil)
	want := `S DA=$O(^DIC(19.1,"B","ZZKEY MANAGER",0)),DIK="^DIC(19.1," I DA D ^DIK`
	if !strings.Contains(got, want) {
		t.Errorf("UninstallScript (key) missing %q\n---\n%s", want, got)
	}
}

// UninstallScript also backs out each PROTOCOL via FileMan DIK on #101 (^ORD(101,).
func TestUninstallScript_Protocol(t *testing.T) {
	got := UninstallScript("ZZPROTO*1.0*1", []string{"ZZPRORT"}, nil, nil, nil, []string{"ZZPROTO ACTION"}, nil, nil, nil, nil)
	want := `S DA=$O(^ORD(101,"B","ZZPROTO ACTION",0)),DIK="^ORD(101," I DA D ^DIK`
	if !strings.Contains(got, want) {
		t.Errorf("UninstallScript (protocol) missing %q\n---\n%s", want, got)
	}
}

// UninstallScript also backs out each REMOTE PROCEDURE via FileMan DIK on #8994
// (^XWB(8994,).
func TestUninstallScript_RPC(t *testing.T) {
	got := UninstallScript("ZZRPC*1.0*1", []string{"ZZRPCRT"}, nil, nil, nil, nil, []string{"ZZRPC ECHO"}, nil, nil, nil)
	want := `S DA=$O(^XWB(8994,"B","ZZRPC ECHO",0)),DIK="^XWB(8994," I DA D ^DIK`
	if !strings.Contains(got, want) {
		t.Errorf("UninstallScript (rpc) missing %q\n---\n%s", want, got)
	}
}

// UninstallScript also backs out each MAIL GROUP via FileMan DIK on #3.8 (^XMB(3.8,).
func TestUninstallScript_MailGroup(t *testing.T) {
	got := UninstallScript("ZZMG*1.0*1", []string{"ZZMGRT"}, nil, nil, nil, nil, nil, []string{"ZZMG ALERTS"}, nil, nil)
	want := `S DA=$O(^XMB(3.8,"B","ZZMG ALERTS",0)),DIK="^XMB(3.8," I DA D ^DIK`
	if !strings.Contains(got, want) {
		t.Errorf("UninstallScript (mail group) missing %q\n---\n%s", want, got)
	}
}

// UninstallScript also backs out each LIST TEMPLATE via FileMan DIK on #409.61
// (^SD(409.61,).
func TestUninstallScript_ListTemplate(t *testing.T) {
	got := UninstallScript("ZZLM*1.0*1", []string{"ZZLMRT"}, nil, nil, nil, nil, nil, nil, []string{"ZZLM PATIENTS"}, nil)
	want := `S DA=$O(^SD(409.61,"B","ZZLM PATIENTS",0)),DIK="^SD(409.61," I DA D ^DIK`
	if !strings.Contains(got, want) {
		t.Errorf("UninstallScript (list template) missing %q\n---\n%s", want, got)
	}
}

// UninstallScript also backs out each FileMan FILE: it reads the data global root
// + name from ^DIC before killing the DD (^DD/^DIC), the data global, and the
// dict-of-files "B" pointer — KIDS ships no generic file uninstall.
func TestUninstallScript_File(t *testing.T) {
	got := UninstallScript("ZZVSLFS*1.0*1", nil, nil, nil, nil, nil, nil, nil, nil, []string{"999000"})
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
