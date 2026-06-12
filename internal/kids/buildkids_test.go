package kids

import (
	"strings"
	"testing"
)

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
