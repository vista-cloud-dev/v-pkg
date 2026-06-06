package kids

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtures are the committed .KID samples (VMDD/OR/XU/DG + synthetic VMTEST).
var fixtures = []string{
	"DG_5_3_853.kid",
	"OR_3_0_484.kid",
	"VMDD_1_0_1.kid",
	"VMTEST_1_0_1.kid",
	"XU_8_0_504.kid",
}

func fixturePath(name string) string { return filepath.Join("testdata", name) }

// TestRoundtripFixtures pins the round-trip property — the whole point — on
// every committed fixture: decompose → assemble → re-parse is canonically
// equal to the original parse (G6, fixture scope).
func TestRoundtripFixtures(t *testing.T) {
	for _, f := range fixtures {
		f := f
		t.Run(f, func(t *testing.T) {
			res, err := Roundtrip(fixturePath(f))
			if err != nil {
				t.Fatalf("Roundtrip(%s) error: %v", f, err)
			}
			if !res.OK {
				for _, d := range res.Diff {
					t.Errorf("build %s: %d→%d pairs\n  - %s\n  + %s",
						d.Build, d.PairsA, d.PairsB, d.FirstA, d.FirstB)
				}
				t.Fatalf("roundtrip FAILED for %s", f)
			}
			if res.Builds == 0 || res.Pairs == 0 {
				t.Fatalf("%s: expected non-empty parse, got builds=%d pairs=%d", f, res.Builds, res.Pairs)
			}
		})
	}
}

// TestSubscriptCodecRoundtrip checks the subscript parse/format inverse over
// representative shapes including the `""` escape and decimal file numbers.
// Float subscripts are normalized to Python's str(float) spelling on output
// (`.01`→`0.01`), so the codec is an inverse only for already-canonical input.
func TestSubscriptCodecRoundtrip(t *testing.T) {
	cases := []struct{ in, want string }{
		{`"BLD",1,0)`, `"BLD",1,0)`},
		{`"KRN",19,12345,0)`, `"KRN",19,12345,0)`},
		{`"^DD",999,.01,0)`, `"^DD",999,0.01,0)`},               // float normalized
		{`"BLD",19801,"KRN",.4,0)`, `"BLD",19801,"KRN",0.4,0)`}, // float normalized
		{`"^DD",999,0.01,0)`, `"^DD",999,0.01,0)`},              // already canonical
		{`"KRN",19,"NM","VMTEST MAIN MENU")`, `"KRN",19,"NM","VMTEST MAIN MENU")`},
		{`"PKG","VMTEST",0)`, `"PKG","VMTEST",0)`},
		{`"FIA",8989.51)`, `"FIA",8989.51)`},
		{`"B","SAY ""HI""",1)`, `"B","SAY ""HI""",1)`},
	}
	for _, c := range cases {
		subs := parseSubscriptLine(c.in)
		if got := formatSubscript(subs); got != c.want {
			t.Errorf("codec:\n  in:   %s\n  out:  %s\n  want: %s", c.in, got, c.want)
		}
	}
}

// TestDecodeUTF8Replace pins the maximal-subpart U+FFFD substitution against
// CPython's errors="replace" behavior (the .KID read path). The cases cover a
// stray continuation, a lone bad lead, two adjacent bad leads (→ two U+FFFD),
// a truncated multi-byte sequence (→ one U+FFFD), and well-formed sequences.
func TestDecodeUTF8Replace(t *testing.T) {
	const r = "�"
	cases := []struct {
		in   []byte
		want string
	}{
		{[]byte("plain ascii"), "plain ascii"},
		{[]byte{0xA7}, r},                         // stray continuation (§ in Latin-1)
		{[]byte{0xEF}, r},                         // lone 3-byte lead, no continuations
		{[]byte{0xEF, 0xEF}, r + r},               // two bad leads → two U+FFFD
		{[]byte{0xEF, 0xBF, 0x41}, r + "A"},       // truncated 3-byte → one U+FFFD, then 'A'
		{[]byte{0xC3, 0xA9}, "é"},                 // well-formed 2-byte
		{[]byte{0xE2, 0x82, 0xAC}, "€"},           // well-formed 3-byte
		{[]byte{0xE0, 0x80}, r + r},               // E0 then out-of-range cont → 2 U+FFFD (matches CPython)
		{[]byte{0x41, 0xA7, 0x42}, "A" + r + "B"}, // ascii, bad, ascii
	}
	for _, c := range cases {
		if got := decodeUTF8Replace(c.in); got != c.want {
			t.Errorf("decodeUTF8Replace(% x) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestParseVMTEST checks the parser against the known structure of the smallest
// fixture.
func TestParseVMTEST(t *testing.T) {
	k, err := ParseKID(fixturePath("VMTEST_1_0_1.kid"))
	if err != nil {
		t.Fatal(err)
	}
	if len(k.InstallNames) != 1 || k.InstallNames[0] != "VMTEST*1.0*1" {
		t.Fatalf("install names = %v", k.InstallNames)
	}
	b := k.Builds["VMTEST*1.0*1"]
	if b == nil {
		t.Fatal("missing VMTEST build")
	}
	// The fixture has RTN, BLD, KRN, ORD, VER, PKG, QUES content.
	if _, ok := b.Get(Subs{strSub("RTN"), strSub("VMTEST"), {kind: kindInt, intV: 1}, {kind: kindInt, intV: 0}}); !ok {
		t.Error("expected RTN VMTEST line 1 node")
	}
}

// TestCanonicalizeRoutineLine2 pins the line-2 stabilization transform.
func TestCanonicalizeRoutineLine2(t *testing.T) {
	in := ";;1.0;VMTEST;**20,27,48**;Apr 25, 1995;Build 3"
	want := ";;1.0;VMTEST;;"
	if got := CanonicalizeRoutineLine2(in); got != want {
		t.Errorf("CanonicalizeRoutineLine2:\n  got:  %q\n  want: %q", got, want)
	}
	// Fewer than 4 pieces is returned unchanged.
	if got := CanonicalizeRoutineLine2(";;x"); got != ";;x" {
		t.Errorf("short line changed: %q", got)
	}
}

// TestDecomposeAssembleLayout checks the on-disk decomposition produces the
// expected KIDComponents layout and that assemble reads it back.
func TestDecomposeAssembleLayout(t *testing.T) {
	k, err := ParseKID(fixturePath("VMTEST_1_0_1.kid"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	name := k.InstallNames[0]
	comp := filepath.Join(dir, PatchDescriptorToDir(name), "KIDComponents")
	if err := DecomposeBuild(k.Builds[name], comp); err != nil {
		t.Fatal(err)
	}
	// VMTEST has Build.zwr, Routines/VMTEST.m, ORD.zwr, KernelFMVersion.zwr.
	for _, rel := range []string{"Build.zwr", "Routines/VMTEST.m", "Routines/VMTEST.header", "ORD.zwr"} {
		if _, err := os.Stat(filepath.Join(comp, rel)); err != nil {
			t.Errorf("expected %s: %v", rel, err)
		}
	}
	// Routine line 2 must be canonicalized on disk.
	mb, err := os.ReadFile(filepath.Join(comp, "Routines/VMTEST.m"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(mb), "\n"), "\n")
	if len(lines) >= 2 && !strings.HasPrefix(lines[1], ";;1.0;VMTEST;;") {
		t.Errorf("line 2 not canonicalized: %q", lines[1])
	}
	pairs, err := AssembleBuild(comp, name)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) == 0 {
		t.Fatal("assemble produced no pairs")
	}
}

// TestCanonicalizeIENs checks IEN substitution on a decomposed VMTEST tree.
func TestCanonicalizeIENs(t *testing.T) {
	k, err := ParseKID(fixturePath("VMTEST_1_0_1.kid"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	name := k.InstallNames[0]
	comp := filepath.Join(dir, PatchDescriptorToDir(name), "KIDComponents")
	if err := DecomposeBuild(k.Builds[name], comp); err != nil {
		t.Fatal(err)
	}
	stats, err := CanonicalizeIENs(dir)
	if err != nil {
		t.Fatal(err)
	}
	// VMTEST has ("BLD",1,…) and ("KRN",19,1,…) — both IEN-substitutable.
	if stats.BLD == 0 {
		t.Errorf("expected BLD substitutions, got 0")
	}
	if stats.KRN == 0 {
		t.Errorf("expected KRN substitutions, got 0")
	}
	// The Build.zwr must now carry "IEN" in place of the build IEN.
	bld, err := os.ReadFile(filepath.Join(comp, "Build.zwr"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bld), `"BLD","IEN"`) {
		t.Errorf("Build.zwr not canonicalized:\n%s", bld)
	}
}

// TestLintDataClassClean checks the data-class gate passes a build with no
// operational data (VMTEST has no DATA section).
func TestLintDataClassClean(t *testing.T) {
	k, err := ParseKID(fixturePath("VMTEST_1_0_1.kid"))
	if err != nil {
		t.Fatal(err)
	}
	res := LintDataClass(k, NewPIKSClassifier(), false)
	if !res.OK {
		t.Errorf("expected clean gate, got %+v", res)
	}
}

// TestLintDataClassBlocked synthesizes a build with DATA on File 2 (PATIENT)
// and confirms the gate fails.
func TestLintDataClassBlocked(t *testing.T) {
	b := newBuild()
	b.Set(Subs{strSub("DATA"), {kind: kindInt, intV: 2}, {kind: kindInt, intV: 1}, {kind: kindInt, intV: 0}}, "JOHN DOE^...")
	k := &KID{InstallNames: []string{"X*1.0*1"}, Builds: map[string]*Build{"X*1.0*1": b}}

	res := LintDataClass(k, NewPIKSClassifier(), false)
	if res.OK {
		t.Fatal("expected gate failure for File 2 (PATIENT) data")
	}
	if res.Blocked != 1 || len(res.Findings) != 1 || res.Findings[0].Class != ClassPatient {
		t.Errorf("unexpected result: %+v", res)
	}
}

// TestLintDataClassStrict checks that an unclassified data file passes by
// default but fails under --strict.
func TestLintDataClassStrict(t *testing.T) {
	b := newBuild()
	// File 777777 is not in the seed → Unknown.
	b.Set(Subs{strSub("DATA"), {kind: kindInt, intV: 777777}, {kind: kindInt, intV: 1}, {kind: kindInt, intV: 0}}, "x")
	k := &KID{InstallNames: []string{"X*1.0*1"}, Builds: map[string]*Build{"X*1.0*1": b}}

	if res := LintDataClass(k, NewPIKSClassifier(), false); !res.OK || res.Unclassified != 1 {
		t.Errorf("lenient: expected OK with 1 unclassified, got %+v", res)
	}
	if res := LintDataClass(k, NewPIKSClassifier(), true); res.OK {
		t.Errorf("strict: expected gate failure for unclassified data file")
	}
}

// TestLoadPIKS checks an external classification table overrides the seed.
func TestLoadPIKS(t *testing.T) {
	dir := t.TempDir()
	tsv := filepath.Join(dir, "piks.tsv")
	if err := os.WriteFile(tsv, []byte("# file\tclass\n777777\tPatient\n9999\tK\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewPIKSClassifier()
	if err := c.LoadPIKS(tsv); err != nil {
		t.Fatal(err)
	}
	if c.Classify(777777) != ClassPatient {
		t.Errorf("777777 = %s, want Patient", c.Classify(777777))
	}
	if c.Classify(9999) != ClassKnowledge {
		t.Errorf("9999 = %s, want Knowledge", c.Classify(9999))
	}
}
