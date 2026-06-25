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
