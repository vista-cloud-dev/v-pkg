package kids

import (
	"strings"
	"testing"
)

// vslInput is a build with a routine, one #8989.51 PARAMETER DEFINITION KRN
// component (SYS-settable free text), and a Required Build on the MSL base.
func vslInput() BuildInput {
	return BuildInput{
		InstallName: "VSLBASE*1.0*1",
		Namespace:   "VSLBASE",
		Routines:    []RoutineSrc{{Name: "VSLCFG", Lines: []string{"VSLCFG ;x", " quit"}}},
		ParamDefs: []ParamDef{{
			Name:         "VSL GREETING",
			DisplayText:  "VSL greeting string (read by VPNG)",
			DataTypeCode: "F",
			Entities:     []ParamEntity{{EntityIEN: "4.2", Precedence: 1}},
		}},
		RequiredBuilds: []ReqBuild{{Name: "MSL*1.0*1", Action: 2}},
	}
}

func TestMakeBuildPairs_ParamDef_KRN(t *testing.T) {
	got := map[string]string{}
	seen := map[string]bool{}
	for _, p := range MakeBuildPairs(vslInput()) {
		k := formatSubscript(p.Subs)
		if seen[k] {
			t.Errorf("duplicate subscript emitted: %s", k)
		}
		seen[k] = true
		got[k] = p.Value
	}

	want := map[string]string{
		// Top-level KRN data: the record image XPDIK merges into ^XTV(8989.51,DA).
		`"KRN",8989.51,1,-1)`:         "0",                                                // action: 0=send
		`"KRN",8989.51,1,0)`:          "VSL GREETING^VSL greeting string (read by VPNG)^", // .01^.02^.03(single)
		`"KRN",8989.51,1,1)`:          "F^^",                                              // 1.1 free text
		`"KRN",8989.51,1,30,0)`:       "^8989.513I^1^1",                                   // ALLOWABLE ENTITIES header
		`"KRN",8989.51,1,30,1,0)`:     "1^4.2",                                            // precedence^entity(SYS=4.2)
		`"KRN",8989.51,1,30,"B",1,1)`: "",                                                 // subfile B index on precedence
		// Install-order section: drives KRN^XPDIK; xref=1 → IX1^DIK rebuilds "B".
		`"ORD",1,8989.51)`:   "8989.51;1;1;;;;;;;",
		`"ORD",1,8989.51,0)`: "PARAMETER DEFINITION",
		// BLD #9.6 manifest: the KRN component list (#9.67) + NM names.
		`"BLD",1,"KRN",0)`:                                 "^9.67PA^8989.51^1",
		`"BLD",1,"KRN",8989.51,0)`:                         "8989.51",
		`"BLD",1,"KRN",8989.51,"NM",0)`:                    "^9.68A^1^1",
		`"BLD",1,"KRN",8989.51,"NM",1,0)`:                  "VSL GREETING^^0",
		`"BLD",1,"KRN",8989.51,"NM","B","VSL GREETING",1)`: "",
		`"BLD",1,"KRN","B",8989.51,8989.51)`:               "",
		// BLD #9.6 manifest: Required Build (#9.611) + top-level MBREQ count.
		`"BLD",1,"REQB",0)`:                 "^9.611^1^1",
		`"BLD",1,"REQB",1,0)`:               "MSL*1.0*1^2",
		`"BLD",1,"REQB","B","MSL*1.0*1",1)`: "",
		`"MBREQ")`:                          "1",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestMakeBuildPairs_ParamDef_Deterministic(t *testing.T) {
	a := MakeBuildPairs(vslInput())
	b := MakeBuildPairs(vslInput())
	if len(a) != len(b) {
		t.Fatalf("length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if formatSubscript(a[i].Subs) != formatSubscript(b[i].Subs) || a[i].Value != b[i].Value {
			t.Fatalf("non-deterministic at %d", i)
		}
	}
}

func TestBuild_ParamDefNames(t *testing.T) {
	b := newBuild()
	for _, p := range MakeBuildPairs(vslInput()) {
		b.Set(p.Subs, p.Value)
	}
	got := b.ParamDefNames()
	if len(got) != 1 || got[0] != "VSL GREETING" {
		t.Errorf("ParamDefNames = %v, want [VSL GREETING]", got)
	}
}

// Routine-only builds must stay byte-identical — no KRN/ORD/MBREQ/REQB nodes
// leak in, so the live-proven ZZSKEL path and its golden output are unchanged.
func TestMakeBuildPairs_RoutineOnly_NoExtraSections(t *testing.T) {
	for _, p := range MakeBuildPairs(BuildInput{
		InstallName: "ZZSKEL*1.0*1", Namespace: "ZZSKEL",
		Routines: []RoutineSrc{{Name: "ZZSKEL", Lines: []string{"ZZSKEL ;x", " quit"}}},
	}) {
		switch sec := p.Subs.Section(); sec {
		case "KRN", "ORD", "MBREQ":
			t.Errorf("routine-only build leaked a %s node: %s", sec, formatSubscript(p.Subs))
		}
		if k := formatSubscript(p.Subs); strings.Contains(k, `"REQB"`) || strings.Contains(k, `"KRN"`) {
			t.Errorf("routine-only build leaked a manifest node: %s", k)
		}
	}
}
