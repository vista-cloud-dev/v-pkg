package kids

import (
	"strconv"
	"testing"
)

func TestBChecksum_FormatsBPrefixed(t *testing.T) {
	lines := []string{
		"ZZT ;test",
		"ZZT ;;1.0;;",
		" W 1",
		" Q",
		"",
	}
	got := BChecksum(lines)
	want := "B" + strconv.FormatInt(RoutineChecksumB(lines), 10)
	if got != want {
		t.Errorf("BChecksum = %q, want %q", got, want)
	}
	if got == "" || got[0] != 'B' {
		t.Errorf("BChecksum %q not B-prefixed", got)
	}
}

func TestRequiredBuildNames_WalksREQB(t *testing.T) {
	// Mirror the BLD,1,"REQB" multiple emitRequiredBuildManifest writes.
	b := buildFrom(
		[2]string{`"BLD",1,0)`, `DEMO*1.0*1`},
		[2]string{`"BLD",1,"REQB",0)`, `^9.611^2^2`},
		[2]string{`"BLD",1,"REQB",1,0)`, `BASE*1.0*3^0`},
		[2]string{`"BLD",1,"REQB",2,0)`, `OTHER*2.0*1^2`},
	)
	got := b.RequiredBuildNames()
	want := []string{"BASE*1.0*3", "OTHER*2.0*1"}
	if len(got) != len(want) {
		t.Fatalf("RequiredBuildNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("RequiredBuildNames[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	none := buildFrom([2]string{`"BLD",1,0)`, `X*1.0*1`})
	if n := none.RequiredBuildNames(); len(n) != 0 {
		t.Errorf("RequiredBuildNames(no reqs) = %v, want empty", n)
	}
}
