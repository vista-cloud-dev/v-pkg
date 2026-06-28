package kids

import (
	"strings"
	"testing"
)

// B.3: a build that declares an env-check + pre/post-install routine emits the
// top-level "PRE"/"INI"/"INIT" transport nodes the install path reads, mirrored in
// the #9.6 BLD manifest. Env-check is a bare routine; pre/post are entryrefs.
func TestMakeBuildPairs_InstallHooks(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZA1*1.0*1", Namespace: "ZZA1",
		Routines:    []RoutineSrc{{Name: "ZZA1P", Lines: []string{"ZZA1P ;x", " quit"}}},
		EnvCheck:    "ZZA1ENV",
		PreInstall:  "PRE^ZZA1P",
		PostInstall: "POST^ZZA1P",
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		`"PRE")`:          "ZZA1ENV",    // top-level env-check routine (read by ENV^XPDIL1)
		`"INI")`:          "PRE^ZZA1P",  // top-level pre-install entryref
		`"INIT")`:         "POST^ZZA1P", // top-level post-install entryref
		`"BLD",1,"PRE")`:  "ZZA1ENV",    // #9.6 BLD manifest mirrors
		`"BLD",1,"INI")`:  "PRE^ZZA1P",
		`"BLD",1,"INIT")`: "POST^ZZA1P",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

// A build with no install hooks emits none of those nodes (byte-identical to the
// pre-B.3 routine-only shape).
func TestMakeBuildPairs_NoInstallHooks(t *testing.T) {
	for _, p := range MakeBuildPairs(BuildInput{
		InstallName: "ZZSKEL*1.0*1", Namespace: "ZZSKEL",
		Routines: []RoutineSrc{{Name: "ZZSKEL", Lines: []string{"ZZSKEL ;x", " quit"}}},
	}) {
		switch k := formatSubscript(p.Subs); k {
		case `"PRE")`, `"INI")`, `"INIT")`, `"BLD",1,"PRE")`, `"BLD",1,"INI")`, `"BLD",1,"INIT")`:
			t.Errorf("hook-free build leaked install-hook node: %s", k)
		}
	}
}

func TestMakeBuildPairs_Deterministic_And_Shape(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZSKEL*1.0*1",
		Namespace:   "ZZSKEL",
		Routines: []RoutineSrc{
			{Name: "ZZSKEL", Lines: []string{"ZZSKEL ; throwaway ;1.0", " quit"}},
		},
	}
	a := MakeBuildPairs(in)
	b := MakeBuildPairs(in)

	// Deterministic: identical inputs → identical pair sequence (invariant #2).
	if len(a) != len(b) {
		t.Fatalf("non-deterministic length: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if formatSubscript(a[i].Subs) != formatSubscript(b[i].Subs) || a[i].Value != b[i].Value {
			t.Fatalf("non-deterministic at %d: %q=%q vs %q=%q", i,
				formatSubscript(a[i].Subs), a[i].Value, formatSubscript(b[i].Subs), b[i].Value)
		}
	}

	// Shape: a BLD header with normalized (zeroed) date/checksum, the RTN count,
	// per-routine section with stripped checksums, the routine lines, and VER.
	got := map[string]string{}
	for _, p := range a {
		got[formatSubscript(p.Subs)] = p.Value
	}
	if v := got[`"BLD",1,0)`]; v != "ZZSKEL*1.0*1^ZZSKEL^0^0" {
		t.Errorf(`"BLD",1,0) = %q, want ZZSKEL*1.0*1^ZZSKEL^0^0 (date normalized)`, v)
	}
	if v := got[`"RTN")`]; v != "1" {
		t.Errorf(`"RTN") = %q, want 1`, v)
	}
	if v := got[`"RTN","ZZSKEL")`]; v != "0^2^0^0" {
		t.Errorf(`"RTN","ZZSKEL") = %q, want 0^2^0^0 (checksums stripped)`, v)
	}
	if v := got[`"RTN","ZZSKEL",1,0)`]; v != "ZZSKEL ; throwaway ;1.0" {
		t.Errorf(`routine line 1 = %q`, v)
	}
	if v := got[`"VER")`]; !strings.Contains(v, "8.0") {
		t.Errorf(`"VER") = %q, want a platform version`, v)
	}
}

// TestBuild_RoutineNames extracts the RTN component names (the 2-subscript
// `"RTN",<name>` header pairs), in build order — what `v pkg verify`/`uninstall`
// need to probe/delete each routine.
func TestBuild_RoutineNames(t *testing.T) {
	pairs := MakeBuildPairs(BuildInput{
		InstallName: "ZZSKEL*1.0*1",
		Namespace:   "ZZSKEL",
		Routines: []RoutineSrc{
			{Name: "ZZSKEL", Lines: []string{"ZZSKEL ;x", " quit"}},
			{Name: "ZZSKEL1", Lines: []string{"ZZSKEL1 ;y", " quit"}},
		},
	})
	b := newBuild()
	for _, p := range pairs {
		b.Set(p.Subs, p.Value)
	}
	got := b.RoutineNames()
	want := []string{"ZZSKEL", "ZZSKEL1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("RoutineNames = %v, want %v", got, want)
	}
}
