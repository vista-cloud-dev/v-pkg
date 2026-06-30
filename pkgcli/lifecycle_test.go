package pkgcli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
	"github.com/vista-cloud-dev/v-pkg/internal/installspec"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// --- fake driver ------------------------------------------------------------

// fakeDriver is a CmdRunner that answers the two verbs the lifecycle path uses
// (`exec load`, `exec run`) with canned clikit envelopes, recording the argv so
// tests can assert the staged path and the entryref.
type fakeDriver struct {
	loadArgs, runArgs []string
	runStdout         string
	loadEng, runEng   *mdriver.EngineError
	loadEmpty         bool // driver stages nothing (no fault) — the silent no-op case
	loads, runs       int  // call counts (the chunked install stages many times)
}

func (f *fakeDriver) run(_ context.Context, _ string, args []string) (stdout, stderr []byte, exit int, err error) {
	switch {
	case len(args) >= 2 && args[0] == "exec" && args[1] == "load":
		f.loadArgs, f.loads = args, f.loads+1
		loaded := []string{"EN"}
		if f.loadEmpty {
			loaded = nil
		}
		return envBytes("exec load", mdriver.LoadResult{Loaded: loaded}, f.loadEng), nil, 0, nil
	case len(args) >= 2 && args[0] == "exec" && args[1] == "run":
		f.runArgs, f.runs = args, f.runs+1
		return envBytes("exec run", mdriver.ExecResult{Stdout: f.runStdout}, f.runEng), nil, 0, nil
	}
	return envBytes("?", nil, nil), nil, 0, nil
}

func envBytes(command string, data any, eng *mdriver.EngineError) []byte {
	raw, _ := json.Marshal(data)
	env := map[string]any{
		"schemaVersion": "1.0", "command": command, "ok": eng == nil, "exit": 0,
		"data": json.RawMessage(raw),
	}
	if eng != nil {
		env["engineError"] = eng
	}
	b, _ := json.Marshal(env)
	return b
}

func fakeClient(f *fakeDriver) *mdriver.Client {
	return mdriver.NewClient("/bin/m-ydb", "ydb", "local", nil, f.run)
}

func zzskelPairs() []kids.Pair {
	return kids.MakeBuildPairs(kids.BuildInput{
		InstallName: "ZZSKEL*1.0*1", Namespace: "ZZSKEL",
		Routines: []kids.RoutineSrc{{Name: "ZZSKEL", Lines: []string{"ZZSKEL ;x", " quit"}}},
	})
}

// --- pure helpers -----------------------------------------------------------

func TestWrapRoutine(t *testing.T) {
	got := wrapRoutine("ZVPKGINS", "S X=1\nW \"hi\",!")
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if lines[0] != "ZVPKGINS ;v-pkg generated lifecycle routine — safe to delete" {
		t.Errorf("header line = %q", lines[0])
	}
	if lines[1] != "EN ;" {
		t.Errorf("label line = %q", lines[1])
	}
	if lines[2] != " S X=1" || lines[3] != ` W "hi",!` {
		t.Errorf("body not indented by one space: %q / %q", lines[2], lines[3])
	}
	if lines[len(lines)-1] != " Q" {
		t.Errorf("missing trailing quit: %q", lines[len(lines)-1])
	}
}

func TestParseMarkers(t *testing.T) {
	// Markers may appear mid-line (after KIDS device output) and over CRLF.
	out := "Some KIDS chatter" + installspec.ResultMarker + "status=3\r\n" +
		"junk\n" + installspec.ResultMarker + "installed=1\n" +
		installspec.ResultMarker + "rtn:ZZSKEL=0\n"
	m := parseMarkers(out)
	if m["status"] != "3" || m["installed"] != "1" || m["rtn:ZZSKEL"] != "0" {
		t.Errorf("parseMarkers = %v", m)
	}
}

func TestLoadBuild_ZZSKEL(t *testing.T) {
	name, b, err := loadBuild(filepath.Join("..", "testdata", "zzskel", "ZZSKEL.kids"))
	if err != nil {
		t.Fatalf("loadBuild: %v", err)
	}
	if name != "ZZSKEL*1.0*1" {
		t.Errorf("name = %q", name)
	}
	if got := b.RoutineNames(); len(got) != 1 || got[0] != "ZZSKEL" {
		t.Errorf("routines = %v", got)
	}
}

// --- install ----------------------------------------------------------------

func TestRunInstall_Success(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "status:" + opToken + "=3\n"}
	res, err := runInstall(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", "hdr", zzskelPairs(), true, nil, nil)
	if err != nil {
		t.Fatalf("runInstall: %v", err)
	}
	if !res.Installed || res.Status != 3 {
		t.Errorf("res = %+v, want installed status 3", res)
	}
	// The script was staged as a .m and run via its EN entry.
	if !anySuffix(f.loadArgs, rtnInstall+".m") {
		t.Errorf("load argv missing …/%s.m: %v", rtnInstall, f.loadArgs)
	}
	if !contains(f.runArgs, "EN^"+rtnInstall) {
		t.Errorf("run argv missing EN^%s: %v", rtnInstall, f.runArgs)
	}
}

// A.4: a multi-build distribution installs each constituent in header order. When
// every build files, all reports come back in order with Installed=true.
func TestInstallSequence_AllInOrder(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "status:" + opToken + "=3\n"}
	names := []string{"EAS*1.0*96", "IVM*2.0*156"}
	mk := func(name string) liveInstallInput {
		return liveInstallInput{name: name, header: name, pairs: zzskelPairs(), runEnvCheck: false}
	}
	got, ferr := installSequence(context.Background(), fakeClient(f), names, mk)
	if ferr != nil {
		t.Fatalf("installSequence: %v", ferr)
	}
	if !got.Installed || len(got.Builds) != 2 {
		t.Fatalf("got %+v, want 2 builds all installed", got)
	}
	if got.Builds[0].Name != "EAS*1.0*96" || got.Builds[1].Name != "IVM*2.0*156" {
		t.Errorf("builds out of header order: %s, %s", got.Builds[0].Name, got.Builds[1].Name)
	}
}

// A.4: if a constituent build fails to reach status 3, the sequence stops — later
// builds (which may depend on it) are never attempted.
func TestInstallSequence_StopsOnFailure(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "status:" + opToken + "=2\n"} // never completes
	names := []string{"EAS*1.0*96", "IVM*2.0*156"}
	mk := func(name string) liveInstallInput {
		return liveInstallInput{name: name, header: name, pairs: zzskelPairs(), runEnvCheck: false}
	}
	got, ferr := installSequence(context.Background(), fakeClient(f), names, mk)
	if ferr != nil {
		t.Fatalf("installSequence: %v", ferr)
	}
	if got.Installed {
		t.Error("sequence must report not-installed when a build fails")
	}
	if len(got.Builds) != 1 {
		t.Errorf("want the sequence to stop after the first failed build, got %d reports", len(got.Builds))
	}
}

// A package whose transport global exceeds one chunk must stage in several
// load+run cycles (none big enough to truncate), then finalize once.
func TestRunInstall_MultiChunkStages(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "status:" + opToken + "=3\n"}
	lines := make([]string, 0, 4000)
	lines = append(lines, "ZZBIG ;big")
	for i := 0; i < 4000; i++ {
		lines = append(lines, fmt.Sprintf(" S X=%d ; padding line %d to grow the transport global", i, i))
	}
	pairs := kids.MakeBuildPairs(kids.BuildInput{
		InstallName: "ZZBIG*1.0*1", Namespace: "ZZBIG",
		Routines: []kids.RoutineSrc{{Name: "ZZBIG", Lines: lines}},
	})
	chunks := installspec.StageChunks(pairs, stageChunkBytes, opToken)
	if len(chunks) < 2 {
		t.Fatalf("test needs a multi-chunk build, got %d chunks for %d pairs", len(chunks), len(pairs))
	}
	res, err := runInstall(context.Background(), fakeClient(f), "ZZBIG*1.0*1", "hdr", pairs, true, nil, nil)
	if err != nil {
		t.Fatalf("runInstall: %v", err)
	}
	if !res.Installed || res.Status != 3 {
		t.Errorf("res = %+v, want installed status 3", res)
	}
	// One load+run per chunk, plus the finalize routine.
	if want := len(chunks) + 1; f.loads != want || f.runs != want {
		t.Errorf("loads=%d runs=%d, want %d each (chunks+finalize)", f.loads, f.runs, want)
	}
}

func TestRunInstall_AlreadyInstalled(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "error=already-installed\n"}
	res, err := runInstall(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", "hdr", zzskelPairs(), true, nil, nil)
	if err != nil {
		t.Fatalf("runInstall: %v", err)
	}
	if res.Installed || res.Error != "already-installed" {
		t.Errorf("res = %+v, want refused already-installed", res)
	}
}

func TestRunInstall_EngineError(t *testing.T) {
	f := &fakeDriver{runEng: &mdriver.EngineError{Mnemonic: "%GTM-E-UNDEF", Text: "XPDA"}}
	_, err := runInstall(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", "hdr", zzskelPairs(), true, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "UNDEF") {
		t.Errorf("want engine fault surfaced, got %v", err)
	}
}

func TestRunInstall_LoadError(t *testing.T) {
	f := &fakeDriver{loadEng: &mdriver.EngineError{Mnemonic: "%GTM-E-ZLINKFILE", Text: "bad"}}
	_, err := runInstall(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", "hdr", zzskelPairs(), true, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "stage") {
		t.Errorf("want stage error, got %v", err)
	}
}

func TestRunInstall_SilentLoadNoOp(t *testing.T) {
	// Driver reports no fault but stages nothing (e.g. no routine source dir) —
	// must fail at staging, not proceed to a confusing run-time link error.
	f := &fakeDriver{loadEmpty: true}
	_, err := runInstall(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", "hdr", zzskelPairs(), true, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "loaded no routine") {
		t.Errorf("want staging no-op surfaced, got %v", err)
	}
	if f.runArgs != nil {
		t.Errorf("must not run EN after an empty load, ran: %v", f.runArgs)
	}
}

// --- verify -----------------------------------------------------------------

func TestRunVerify(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "installed=1\n" +
		installspec.ResultMarker + "status=3\n" +
		installspec.ResultMarker + "rtn:ZZSKEL=1\n"}
	res, err := runVerify(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", []string{"ZZSKEL"}, nil, nil)
	if err != nil {
		t.Fatalf("runVerify: %v", err)
	}
	if !res.Installed || res.Status != 3 || !res.Routines["ZZSKEL"] {
		t.Errorf("res = %+v", res)
	}
}

// A verify carrying entry components reads each one's generic comp:<file>:<name>
// presence marker, keys it into res.Components, and folds it into ok() — a missing
// record of ANY type defeats verification. One table covers every registered type
// (the per-type maps are gone; a new type is a registry row, no new test).
func TestRunVerify_Components(t *testing.T) {
	cases := []struct {
		file                    float64
		fileStr, dataRoot, name string
	}{
		{8989.51, "8989.51", "^XTV(8989.51,", "VSL GREETING"},
		{19, "19", "^DIC(19,", "ZZOPT RUN ROUTINE"},
		{19.1, "19.1", "^DIC(19.1,", "ZZKEY MANAGER"},
		{101, "101", "^ORD(101,", "ZZPROTO ACTION"},
		{8994, "8994", "^XWB(8994,", "ZZRPC ECHO"},
		{3.8, "3.8", "^XMB(3.8,", "ZZMG ALERTS"},
		{409.61, "409.61", "^SD(409.61,", "ZZLM PATIENTS"},
		{9.2, "9.2", "^DIC(9.2,", "ZZHF-MAIN"},
		{771, "771", "^HL(771,", "ZZHL_APP"},
		{779.2, "779.2", "^HLD(779.2,", "ZZHO_APP"},
		{870, "870", "^HLCS(870,", "ZZLINK"},
		{0.402, "0.402", "^DIE(", "ZZTMPL FILE #999000"}, // new type, same path
	}
	for _, tc := range cases {
		key := tc.fileStr + ":" + tc.name
		f := &fakeDriver{runStdout: installspec.ResultMarker + "installed=1\n" +
			installspec.ResultMarker + "status=3\n" +
			installspec.ResultMarker + "rtn:ZZRT=1\n" +
			installspec.ResultMarker + "comp:" + key + "=1\n"}
		comps := []kids.Component{{File: tc.file, FileStr: tc.fileStr, DataRoot: tc.dataRoot, Names: []string{tc.name}}}
		res, err := runVerify(context.Background(), fakeClient(f), "ZZ*1.0*1", []string{"ZZRT"}, comps, nil)
		if err != nil {
			t.Fatalf("%s: runVerify: %v", key, err)
		}
		if !res.Components[key] || !res.ok() {
			t.Errorf("%s: res = %+v, want component present and ok()", key, res)
		}
		res.Components[key] = false
		if res.ok() {
			t.Errorf("%s: ok() must be false when a component is missing", key)
		}
	}
}

// A verify carrying a FileMan FILE reads its DD-present marker and folds it into
// ok() — a missing file dictionary means the install is not fully verified.
func TestRunVerify_File(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "installed=1\n" +
		installspec.ResultMarker + "status=3\n" +
		installspec.ResultMarker + "file:999000=1\n"}
	res, err := runVerify(context.Background(), fakeClient(f), "ZZVSLFS*1.0*1", nil, nil, []string{"999000"})
	if err != nil {
		t.Fatalf("runVerify: %v", err)
	}
	if !res.Files["999000"] || !res.ok() {
		t.Errorf("res = %+v, want file present and ok()", res)
	}
	res.Files["999000"] = false
	if res.ok() {
		t.Error("ok() must be false when a file dictionary is missing")
	}
}

func TestRunVerify_NotInstalled(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "installed=0\n" +
		installspec.ResultMarker + "status=\n" +
		installspec.ResultMarker + "rtn:ZZSKEL=0\n"}
	res, err := runVerify(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", []string{"ZZSKEL"}, nil, nil)
	if err != nil {
		t.Fatalf("runVerify: %v", err)
	}
	if res.Installed || res.Routines["ZZSKEL"] {
		t.Errorf("res = %+v, want all-false", res)
	}
}

// --- content verification ---------------------------------------------------

// verifyContent reads the live 0-node of each shipped entry record and grades it
// against the shipped image: ok (matches aside from FileMan-transformed pieces),
// mismatch (a record by that name exists but differs), or absent (no record).
func TestVerifyContent(t *testing.T) {
	contents := []kids.EntryContent{
		{File: 19, FileStr: "19", Name: "ZZOPT", DataRoot: "^DIC(19,", Zero: "ZZOPT^Demo^^R"},
		{File: 19.1, FileStr: "19.1", Name: "ZZKEY", DataRoot: "^DIC(19.1,", Zero: "ZZKEY"},
		{File: 771, FileStr: "771", Name: "ZZHL", DataRoot: "^HL(771,", Zero: "ZZHL^a^500^^^^USA", Volatile: []int{7}},
	}
	files := []kids.FileContent{
		{File: 999001, FileStr: "999001", Field: "1", Zero: `USER NUMBER^NJ12,0^^0;2^K:+X'=X X`},
		{File: 999001, FileStr: "999001", Field: "2", Zero: "EVENT^S^I:INFO;^0;3^Q"},
	}
	f := &fakeDriver{runStdout: installspec.ResultMarker + "z:19:ZZOPT=ZZOPT^Demo^^R\n" + // exact match
		installspec.ResultMarker + "z:19.1:ZZKEY=ZZKEY WRONG\n" + // record exists but differs
		installspec.ResultMarker + "z:771:ZZHL=ZZHL^a^500^^^^1\n" + // country USA resolved to pointer 1 (volatile)
		installspec.ResultMarker + `dd:999001#1=USER NUMBER^NJ12,0^^0;2^K:+X'=X X` + "\n" + // DD field matches
		installspec.ResultMarker + "dd:999001#2=EVENT^S^I:DIFFERENT;^0;3^Q\n"} // DD field differs
	got, err := verifyContent(context.Background(), fakeClient(f), contents, files)
	if err != nil {
		t.Fatalf("verifyContent: %v", err)
	}
	want := map[string]string{
		"19:ZZOPT": "ok", "19.1:ZZKEY": "mismatch", "771:ZZHL": "ok",
		"999001#1": "ok", "999001#2": "mismatch",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("content %s = %q, want %q", k, got[k], v)
		}
	}
	// A field with no marker (the DD field never filed) grades as absent.
	g2, err := verifyContent(context.Background(), fakeClient(&fakeDriver{runStdout: ""}), contents[:1], files[:1])
	if err != nil {
		t.Fatalf("verifyContent (absent): %v", err)
	}
	if g2["19:ZZOPT"] != "absent" || g2["999001#1"] != "absent" {
		t.Errorf("absent case = %v, want both absent", g2)
	}
}

// A content mismatch defeats ok(): an install that filed a wrong record is not a
// verified install, even though the #9.7 status reads complete and the name exists.
func TestVerifyResultOK_ContentGate(t *testing.T) {
	r := verifyResult{Installed: true, Status: 3, Content: map[string]string{"19:ZZOPT": "mismatch"}}
	if r.ok() {
		t.Error("ok() must be false when a shipped record's content mismatches")
	}
	r.Content["19:ZZOPT"] = "ok"
	if !r.ok() {
		t.Error("ok() must be true when content matches and the install completed")
	}
}

// --- uninstall --------------------------------------------------------------

func TestRunUninstall(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "uninstalled=1\n"}
	res, err := runUninstall(context.Background(), fakeClient(f), "ZZSKEL*1.0*1", []string{"ZZSKEL"}, nil, nil)
	if err != nil {
		t.Fatalf("runUninstall: %v", err)
	}
	if !res.Uninstalled {
		t.Errorf("res = %+v, want uninstalled", res)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func anySuffix(ss []string, suffix string) bool {
	for _, s := range ss {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// --- deregister (clear the #9.4 footprint) ----------------------------------

func TestDeregReg(t *testing.T) {
	r := deregReg("MSL*0.1*1")
	if r == nil || r.Prefix != "MSL" || r.Version != "0.1" || r.Patch != "1" {
		t.Errorf("deregReg(MSL*0.1*1) = %+v", r)
	}
	if r := deregReg("MSL*0.1"); r == nil || r.Patch != "" {
		t.Errorf("deregReg(MSL*0.1) = %+v, want patch empty", r)
	}
	if r := deregReg("bogus"); r != nil {
		t.Errorf("deregReg(bogus) = %+v, want nil", r)
	}
}

func TestRunDeregister(t *testing.T) {
	reg := &installspec.PkgReg{Prefix: "MSL", Version: "0.1", Patch: "1"}
	f := &fakeDriver{runStdout: installspec.ResultMarker + "dereg=1\n"}
	ok, err := runDeregister(context.Background(), fakeClient(f), reg)
	if err != nil {
		t.Fatalf("runDeregister: %v", err)
	}
	if !ok {
		t.Error("want deregistered=true for dereg=1")
	}
	// No footprint found → false (not an error).
	f2 := &fakeDriver{runStdout: installspec.ResultMarker + "dereg=0\n"}
	if ok2, _ := runDeregister(context.Background(), fakeClient(f2), reg); ok2 {
		t.Error("want deregistered=false for dereg=0")
	}
	// A patchless reg has nothing to clear → false, no engine call.
	if ok3, _ := runDeregister(context.Background(), fakeClient(f), &installspec.PkgReg{Prefix: "MSL", Version: "0.1"}); ok3 {
		t.Error("want deregistered=false for patchless reg")
	}
}
