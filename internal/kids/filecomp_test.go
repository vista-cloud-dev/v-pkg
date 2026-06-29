package kids

import (
	"strings"
	"testing"
)

// fileInput is a build that ships a single brand-new FileMan file DD: #999000
// ZZVSLFS, one free-text .01 NAME field, data global ^DIZ(999000,.
func fileInput() BuildInput {
	return BuildInput{
		InstallName: "ZZVSLFS*1.0*1",
		Namespace:   "ZZVSLFS",
		Files: []FileDD{{
			Number:     999000,
			Name:       "ZZVSLFS",
			GlobalRoot: "^DIZ(999000,",
		}},
	}
}

func TestMakeBuildPairs_File_FIAandDD(t *testing.T) {
	got := map[string]string{}
	seen := map[string]bool{}
	for _, p := range MakeBuildPairs(fileInput()) {
		k := formatSubscript(p.Subs)
		if seen[k] {
			t.Errorf("duplicate subscript emitted: %s", k)
		}
		seen[k] = true
		got[k] = p.Value
	}

	want := map[string]string{
		// FIA section — drives FIA^XPDIK (file name, data global, options, version).
		`"FIA",999000)`:        "ZZVSLFS",
		`"FIA",999000,0)`:      "^DIZ(999000,",
		`"FIA",999000,0,0)`:    "999000I",
		`"FIA",999000,0,1)`:    "y^n^f^^^^n^^n",
		`"FIA",999000,0,"VR")`: "1.0^ZZVSLFS",
		// ^DD image — the attribute dictionary (DDIN^DIFROMS files this). The file
		// number is doubled: ("^DD",<file>,<ddfile>,…).
		`"^DD",999000,999000,0)`:                      "FIELD^^.01^1",
		`"^DD",999000,999000,0,"IX","B",999000,0.01)`: "",
		`"^DD",999000,999000,0,"NM","ZZVSLFS")`:       "",
		`"^DD",999000,999000,0.01,0)`:                 "NAME^RF^^0;1^K:$L(X)>30!($L(X)<1) X",
		`"^DD",999000,999000,0.01,1,0)`:               "^.1",
		`"^DD",999000,999000,0.01,1,1,0)`:             "999000^B",
		`"^DD",999000,999000,0.01,1,1,1)`:             `S ^DIZ(999000,"B",$E(X,1,30),DA)=""`,
		`"^DD",999000,999000,0.01,1,1,2)`:             `K ^DIZ(999000,"B",$E(X,1,30),DA)`,
		`"^DD",999000,999000,0.01,3)`:                 "Answer must be 1-30 characters in length.",
		`"^DD",999000,999000,"B","NAME",0.01)`:        "",
		`"^DD",999000,999000,"GL","0;1",1,0.01)`:      "",
		// ^DIC image — dict-of-files registration (file level prefixes the subtree).
		`"^DIC",999000,999000,0)`:             "ZZVSLFS^999000",
		`"^DIC",999000,999000,0,"GL")`:        "^DIZ(999000,",
		`"^DIC",999000,"B","ZZVSLFS",999000)`: "",
		// BLD #9.64 FILE manifest — the build's self-description.
		`"BLD",1,4,0)`:                 "^9.64PA^999000^1",
		`"BLD",1,4,999000,0)`:          "999000",
		`"BLD",1,4,999000,222)`:        "y^n^f^^^^n^^n",
		`"BLD",1,4,"B",999000,999000)`: "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

// multiFieldInput is a build that ships a new file with a free-text .01 plus the
// five grounded field types (numeric, set-of-codes, date, pointer, free-text) —
// the shape v-stdlib R3's VSL AUDIT file needs.
func multiFieldInput() BuildInput {
	min1 := 1.0
	return BuildInput{
		InstallName: "ZZVSLAU*1.0*1",
		Namespace:   "ZZVSLAU",
		Files: []FileDD{{
			Number:     999001,
			Name:       "ZZVSLAUDIT",
			GlobalRoot: "^DIZ(999001,",
			Fields: []FileField{
				{Number: 1, Label: "USER NUMBER", Type: FieldNumeric, Node: 0, Piece: 2, Width: 12, Decimals: 0, Min: &min1},
				{Number: 2, Label: "EVENT TYPE", Type: FieldSet, Node: 0, Piece: 3, Codes: []SetCode{{"I", "INFO"}, {"W", "WARN"}, {"E", "ERROR"}}},
				{Number: 3, Label: "TIMESTAMP", Type: FieldDate, Node: 0, Piece: 4, HasTime: true},
				{Number: 4, Label: "USER", Type: FieldPointer, Node: 0, Piece: 5, Required: true, PointTo: 200, PointRoot: "^VA(200,"},
				{Number: 5, Label: "DETAIL", Type: FieldFreeText, Node: 0, Piece: 6, MaxLen: 245, Help: "Free-text detail, up to 245 characters."},
			},
		}},
	}
}

// TestFileContents proves the build exposes, per shipped FileMan FILE field, the
// ^DD(file,fld,0) definition node a content-asserting verify reads back and
// compares — covering the .01 plus every typed field, keyed by field number.
func TestFileContents(t *testing.T) {
	b := newBuild()
	for _, p := range MakeBuildPairs(multiFieldInput()) {
		b.Set(parseSubscriptLine(formatSubscript(p.Subs)), p.Value)
	}
	byField := map[string]FileContent{}
	for _, fc := range b.FileContents() {
		byField[fc.Field] = fc
	}
	if len(byField) != 6 { // .01 + 5 typed fields
		t.Fatalf("FileContents: got %d field defs, want 6: %v", len(byField), byField)
	}
	one, ok := byField["1"]
	if !ok || one.File != 999001 || one.FileStr != "999001" {
		t.Errorf("field 1 = %+v", one)
	}
	if one.Zero != `USER NUMBER^NJ12,0^^0;2^K:+X'=X!(X<1)!(X?.E1"."1.N) X` {
		t.Errorf("field 1 Zero = %q", one.Zero)
	}
	if _, ok := byField["0.01"]; !ok {
		t.Errorf("missing the .01 field def; have %v", byField)
	}
}

func TestMakeBuildPairs_File_MultiField(t *testing.T) {
	got := map[string]string{}
	seen := map[string]bool{}
	for _, p := range MakeBuildPairs(multiFieldInput()) {
		k := formatSubscript(p.Subs)
		if seen[k] {
			t.Errorf("duplicate subscript emitted: %s", k)
		}
		seen[k] = true
		got[k] = p.Value
	}

	want := map[string]string{
		// Header generalizes: highest field# = 5, field count = 6 (.01 + 5).
		`"^DD",999001,999001,0)`: "FIELD^^5^6",
		// Numeric: NJ<width>,<decimals>; transform carries the range + decimal guard.
		`"^DD",999001,999001,1,0)`: `USER NUMBER^NJ12,0^^0;2^K:+X'=X!(X<1)!(X?.E1"."1.N) X`,
		// Set-of-codes: piece 3 is the int:ext;… list (trailing ";"); transform "Q".
		`"^DD",999001,999001,2,0)`: "EVENT TYPE^S^I:INFO;W:WARN;E:ERROR;^0;3^Q",
		// Date with time: %DT flags "ET".
		`"^DD",999001,999001,3,0)`: `TIMESTAMP^D^^0;4^S %DT="ET" D ^%DT S X=Y K:Y<1 X`,
		// Required pointer: RP<file>'; piece 3 is the pointed-to global root WITHOUT
		// the leading "^" (the buildspec's "^VA(200," has its caret stripped — a
		// literal "^" would be the piece delimiter); storage 0;5 stays in piece 4.
		`"^DD",999001,999001,4,0)`: "USER^RP200'^VA(200,^0;5^Q",
		// Free-text with an explicit help string.
		`"^DD",999001,999001,5,0)`: `DETAIL^F^^0;6^K:$L(X)>245!($L(X)<1) X`,
		`"^DD",999001,999001,5,3)`: "Free-text detail, up to 245 characters.",
		// The .01 NAME field is still emitted (unchanged shape).
		`"^DD",999001,999001,0.01,0)`: "NAME^RF^^0;1^K:$L(X)>30!($L(X)<1) X",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}

	// Additional fields carry their storage location inline (^DD piece 4), like
	// real exports — they emit no separate "GL" map node. The only ^DD GL node is
	// the pre-existing .01 one; no field>.01 GL node leaks in.
	for k := range got {
		if strings.Contains(k, `"^DD"`) && strings.Contains(k, `,"GL",`) && !strings.Contains(k, `,1,0.01)`) {
			t.Errorf("unexpected per-field ^DD GL node emitted: %s", k)
		}
	}
}

// dataInput ships a new file (#999000) with a free-text .01 NAME, a numeric field
// at node 0 piece 2, and a free-text field at node 1 piece 1, plus two data records
// under a MERGE action. The data ships as the raw record storage subtree under
// ("DATA",<file>,<ien>,<node>) — exactly what DATAIN^DIFROMS (EN^DIFROMS4) installs.
func dataInput() BuildInput {
	return BuildInput{
		InstallName: "ZZVSLDATA*1.0*1",
		Namespace:   "ZZVSLDATA",
		Files: []FileDD{{
			Number:     999000,
			Name:       "ZZVSLDATA",
			GlobalRoot: "^DIZ(999000,",
			Fields: []FileField{
				{Number: 1, Label: "COUNT", Type: FieldNumeric, Node: 0, Piece: 2, Width: 6, Decimals: 0},
				{Number: 2, Label: "NOTE", Type: FieldFreeText, Node: 1, Piece: 1, MaxLen: 60},
			},
			Data: &FileData{
				Action: "m", // MERGE
				Records: []FileRecord{
					{IEN: 1, Nodes: map[int]map[int]string{0: {1: "ALPHA", 2: "42"}, 1: {1: "hello"}}},
					{IEN: 2, Nodes: map[int]map[int]string{0: {1: "BRAVO"}}},
				},
			},
		}},
	}
}

func TestMakeBuildPairs_File_Data(t *testing.T) {
	got := map[string]string{}
	for _, p := range MakeBuildPairs(dataInput()) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		// send-options now carry the data switch (piece 7 = y) + action (piece 8 = m).
		`"FIA",999000,0,1)`:     "y^n^f^^^^y^m^n",
		`"BLD",1,4,999000,222)`: "y^n^f^^^^y^m^n",
		// data records — the raw storage subtree per IEN, packed by node;piece.
		`"DATA",999000,1,0)`: "ALPHA^42",
		`"DATA",999000,1,1)`: "hello",
		`"DATA",999000,2,0)`: "BRAVO",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	// A DD-only file must NOT leak any DATA node.
	for _, p := range MakeBuildPairs(fileInput()) {
		if p.Subs.Section() == "DATA" {
			t.Errorf("DD-only build leaked a DATA node: %s", formatSubscript(p.Subs))
		}
	}
}

func TestMakeBuildPairs_File_Data_Deterministic(t *testing.T) {
	a := MakeBuildPairs(dataInput())
	b := MakeBuildPairs(dataInput())
	if len(a) != len(b) {
		t.Fatalf("length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if formatSubscript(a[i].Subs) != formatSubscript(b[i].Subs) || a[i].Value != b[i].Value {
			t.Fatalf("non-deterministic at %d", i)
		}
	}
}

func TestMakeBuildPairs_File_MultiField_Deterministic(t *testing.T) {
	a := MakeBuildPairs(multiFieldInput())
	b := MakeBuildPairs(multiFieldInput())
	if len(a) != len(b) {
		t.Fatalf("length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if formatSubscript(a[i].Subs) != formatSubscript(b[i].Subs) || a[i].Value != b[i].Value {
			t.Fatalf("non-deterministic at %d", i)
		}
	}
}

func TestMakeBuildPairs_File_Deterministic(t *testing.T) {
	a := MakeBuildPairs(fileInput())
	b := MakeBuildPairs(fileInput())
	if len(a) != len(b) {
		t.Fatalf("length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if formatSubscript(a[i].Subs) != formatSubscript(b[i].Subs) || a[i].Value != b[i].Value {
			t.Fatalf("non-deterministic at %d", i)
		}
	}
}

func TestBuild_FileNumbers(t *testing.T) {
	b := newBuild()
	for _, p := range MakeBuildPairs(fileInput()) {
		b.Set(p.Subs, p.Value)
	}
	got := b.FileNumbers()
	if len(got) != 1 || got[0] != 999000 {
		t.Errorf("FileNumbers = %v, want [999000]", got)
	}
}

// A routine-only build must stay byte-identical — no FIA/^DD/^DIC nodes leak in.
func TestMakeBuildPairs_RoutineOnly_NoFileSections(t *testing.T) {
	for _, p := range MakeBuildPairs(BuildInput{
		InstallName: "ZZSKEL*1.0*1", Namespace: "ZZSKEL",
		Routines: []RoutineSrc{{Name: "ZZSKEL", Lines: []string{"ZZSKEL ;x", " quit"}}},
	}) {
		switch p.Subs.Section() {
		case "FIA", "^DD", "^DIC":
			t.Errorf("routine-only build leaked a %s node: %s", p.Subs.Section(), formatSubscript(p.Subs))
		}
		if k := formatSubscript(p.Subs); strings.Contains(k, `"BLD",1,4`) {
			t.Errorf("routine-only build leaked a FILE manifest node: %s", k)
		}
	}
}
