package pkgcli

import (
	"slices"
	"strings"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/installspec"
)

func TestValidRoutineName(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"XWBBRK", true},
		{"%ZIS", true},
		{"XWBBRK2", true},
		{"A", true},
		{"VSLHL7TAP", true}, // 9 chars — allowLongNames package routine
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZ12345", true}, // 31 chars — the YDB/IRIS significant limit
		{"", false}, // empty
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZ123456", false}, // 32 chars > 31
		{"1ABC", false},    // leading digit
		{"XWB BRK", false}, // space — would break the M literal
		{`XWBBRK)`, false}, // injection attempt: closes the $TEXT ref
		{"X%Y", false},     // % only allowed leading
		{"XWB-BR", false},  // hyphen
	}
	for _, c := range cases {
		if got := validRoutineName(c.name); got != c.ok {
			t.Errorf("validRoutineName(%q) = %v, want %v", c.name, got, c.ok)
		}
	}
}

func TestReadRoutineBody_InterpolatesNameAndMarker(t *testing.T) {
	body := readRoutineBody("XWBBRK")
	if !strings.Contains(body, "$T(+I^XWBBRK)") {
		t.Errorf("body does not read the named routine via $TEXT:\n%s", body)
	}
	if !strings.Contains(body, installspec.ResultMarker+"l") {
		t.Errorf("body does not emit the line markers (%sl…):\n%s", installspec.ResultMarker, body)
	}
	// the end sentinel: a line that is empty AND followed by an empty line.
	if !strings.Contains(body, `$T(+(I+1)^XWBBRK)=""`) {
		t.Errorf("body lacks the two-empty end sentinel:\n%s", body)
	}
}

func TestParseRoutineLines_OrderedContiguousAndPreservesWhitespace(t *testing.T) {
	// markers as parseMarkers would return them (key l<n> → verbatim line).
	markers := map[string]string{
		"l1": `XWBBRK ;header`,
		"l2": ` N ERR,S`, // leading space must survive
		"l3": ` S:$L($G(XWBSEC)) ERR="-1^"_XWBSEC`,
		"l5": ` ORPHAN — beyond the contiguous run, must be ignored`,
		"x":  "noise",
	}
	got := parseRoutineLines(markers)
	want := []string{
		`XWBBRK ;header`,
		` N ERR,S`,
		` S:$L($G(XWBSEC)) ERR="-1^"_XWBSEC`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("parseRoutineLines:\n got: %#v\nwant: %#v (must stop at the l4 gap and keep order + leading spaces)", got, want)
	}
}

func TestParseRoutineLines_Empty(t *testing.T) {
	if got := parseRoutineLines(map[string]string{"other": "x"}); len(got) != 0 {
		t.Errorf("parseRoutineLines with no l<n> keys = %#v, want empty", got)
	}
}
