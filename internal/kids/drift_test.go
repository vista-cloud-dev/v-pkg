package kids

import (
	"slices"
	"testing"
)

func TestRoutineSource_OrderedByLineNumber(t *testing.T) {
	b := buildFrom(
		[2]string{`"RTN","XWBBRK")`, `0^3^0^0`},
		[2]string{`"RTN","XWBBRK",2,0)`, ` ;;1.0`},
		[2]string{`"RTN","XWBBRK",1,0)`, `XWBBRK ;hdr`},
		[2]string{`"RTN","XWBBRK",3,0)`, ` Q`},
		[2]string{`"RTN","OTHER",1,0)`, ` other`}, // a different routine, must be excluded
	)
	got := b.RoutineSource("XWBBRK")
	want := []string{`XWBBRK ;hdr`, ` ;;1.0`, ` Q`}
	if !slices.Equal(got, want) {
		t.Errorf("RoutineSource = %#v, want %#v", got, want)
	}
	if len(b.RoutineSource("NOPE")) != 0 {
		t.Errorf("RoutineSource of absent routine should be empty")
	}
}

func TestRoutineDriftMatch(t *testing.T) {
	shipped := []string{`XWBBRK ;hdr`, ` ;;1.0;XWB;**1**;date`, ` D req^VSLRPCWRAP`, ` Q`}
	cases := []struct {
		name string
		live []string
		want bool
	}{
		{
			name: "identical -> applied (match)",
			live: []string{`XWBBRK ;hdr`, ` ;;1.0;XWB;**1**;date`, ` D req^VSLRPCWRAP`, ` Q`},
			want: true,
		},
		{
			name: "only line 2 differs (real patch list/checksum) -> still applied",
			live: []string{`XWBBRK ;hdr`, ` ;;1.0;XWB;**1,7,42**;laterdate`, ` D req^VSLRPCWRAP`, ` Q`},
			want: true,
		},
		{
			name: "a body line changed (later patch overwrote our splice) -> DRIFTED",
			live: []string{`XWBBRK ;hdr`, ` ;;1.0;XWB;**1**;date`, ` ; our splice is gone`, ` Q`},
			want: false,
		},
		{
			name: "different length -> drifted",
			live: []string{`XWBBRK ;hdr`, ` ;;1.0`},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RoutineDriftMatch(shipped, tc.live); got != tc.want {
				t.Errorf("RoutineDriftMatch = %v, want %v", got, tc.want)
			}
		})
	}
}

// D4 (F1): a national-overwrite restore needs a STRICTER verdict than the
// line-2-blind RoutineDriftMatch. The COMMAND lines (every line except line 2) are
// the checksum surface — both VistA checksums (RSUM/CHECK1^XTSUMBLD) exclude line 2
// — so command-line byte-equality is what proves the restored routine matches the
// FORUM gold checksum. Line 2 carries the volatile patch-history provenance (patch
// list / date / Build N): a restore that leaves it wrong is checksum-clean but
// provenance-drifted. So the verdict is three-way.
func TestRoutineRestoreVerdict(t *testing.T) {
	shipped := []string{`XWBPRS ;hdr`, ` ;;1.0;XWB;**1,7**;Jul 10, 1995;Build 8`, ` D EN^XWBPRS`, ` Q`}
	cases := []struct {
		name string
		live []string
		want string
	}{
		{
			name: "byte-identical incl line 2 -> exact",
			live: []string{`XWBPRS ;hdr`, ` ;;1.0;XWB;**1,7**;Jul 10, 1995;Build 8`, ` D EN^XWBPRS`, ` Q`},
			want: "exact",
		},
		{
			name: "command lines identical, line-2 patch list differs -> provenance drift",
			live: []string{`XWBPRS ;hdr`, ` ;;1.0;XWB;**1**;older;Build 6`, ` D EN^XWBPRS`, ` Q`},
			want: "command-clean-provenance-drift",
		},
		{
			name: "a command line differs (checksum surface broken) -> drift",
			live: []string{`XWBPRS ;hdr`, ` ;;1.0;XWB;**1,7**;Jul 10, 1995;Build 8`, ` D EVIL^X`, ` Q`},
			want: "drift",
		},
		{
			name: "different length (line added/removed) -> drift",
			live: []string{`XWBPRS ;hdr`, ` ;;1.0;XWB;**1,7**;Jul 10, 1995;Build 8`, ` Q`},
			want: "drift",
		},
		{
			name: "line 1 (header) differs -> drift (line 1 is a command line)",
			live: []string{`XWBPRS ;TAMPERED`, ` ;;1.0;XWB;**1,7**;Jul 10, 1995;Build 8`, ` D EN^XWBPRS`, ` Q`},
			want: "drift",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RoutineRestoreVerdict(shipped, tc.live); got != tc.want {
				t.Errorf("RoutineRestoreVerdict = %q, want %q", got, tc.want)
			}
		})
	}
}
