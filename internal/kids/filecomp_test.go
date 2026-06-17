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
