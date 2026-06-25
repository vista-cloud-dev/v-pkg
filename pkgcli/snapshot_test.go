package pkgcli

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/installspec"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

func TestSnapshotName(t *testing.T) {
	cases := []struct{ orig, override, want string }{
		{"XWBBRK*1.0*1", "", "XWBBRK*1.0*1 PREIMAGE"},
		{"VSLTAP RPC WRAP 1.0", "", "VSLTAP RPC WRAP 1.0 PREIMAGE"},
		{"XWBBRK*1.0*1", "MY SNAP 1.0", "MY SNAP 1.0"}, // explicit override wins
	}
	for _, c := range cases {
		if got := snapshotName(c.orig, c.override); got != c.want {
			t.Errorf("snapshotName(%q,%q) = %q, want %q", c.orig, c.override, got, c.want)
		}
	}
}

func TestSnapshotNamespace(t *testing.T) {
	cases := []struct{ in, want string }{
		{"XWBBRK*1.0*1", "XWBBRK"},
		{"VSLTAP RPC WRAP 1.0", "VSLTAP"},
		{"OR*3.0*484", "OR"},
		{"*bad", "VPKG"}, // no leading namespace -> fallback
	}
	for _, c := range cases {
		if got := snapshotNamespace(c.in); got != c.want {
			t.Errorf("snapshotNamespace(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// buildSnapshotPairs builds an installable routine-only KIDS from captured
// pre-image source — it must carry the routine NAME node and its source lines.
func TestBuildSnapshotPairs_CarriesRoutine(t *testing.T) {
	captured := []kids.RoutineSrc{
		{Name: "XWBBRK", Lines: []string{"XWBBRK ;stock header", " N ERR", " Q"}},
	}
	pairs := buildSnapshotPairs("XWBBRK*1.0*1 PREIMAGE", "XWBBRK", captured)
	var names, srcLines []string
	for _, p := range pairs {
		s := p.Subs
		if len(s) == 2 && s[0].IsStr() && s[0].Str() == "RTN" && s[1].IsStr() {
			names = append(names, s[1].Str())
		}
		if len(s) == 4 && s[0].IsStr() && s[0].Str() == "RTN" {
			srcLines = append(srcLines, p.Value)
		}
	}
	if !slices.Equal(names, []string{"XWBBRK"}) {
		t.Errorf("routine name nodes = %v, want [XWBBRK]", names)
	}
	if !slices.Equal(srcLines, captured[0].Lines) {
		t.Errorf("source lines = %#v, want %#v", srcLines, captured[0].Lines)
	}
}

// readRoutinePreimage: present routine -> source + present=true.
func TestReadRoutinePreimage_Present(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "l1=XWBBRK ;hdr\n" +
		installspec.ResultMarker + "l2= N ERR\n"}
	lines, present, err := readRoutinePreimage(context.Background(), fakeClient(f), "XWBBRK")
	if err != nil {
		t.Fatalf("readRoutinePreimage: %v", err)
	}
	if !present {
		t.Errorf("present = false, want true")
	}
	if !slices.Equal(lines, []string{"XWBBRK ;hdr", " N ERR"}) {
		t.Errorf("lines = %#v", lines)
	}
}

// readRoutinePreimage: absent routine -> present=false, NO error (it is the
// greenfield case, not a fault).
func TestReadRoutinePreimage_Absent(t *testing.T) {
	f := &fakeDriver{runStdout: "no markers — routine not on the engine\n"}
	lines, present, err := readRoutinePreimage(context.Background(), fakeClient(f), "ZZNOPE")
	if err != nil {
		t.Fatalf("readRoutinePreimage (absent) returned error: %v", err)
	}
	if present || len(lines) != 0 {
		t.Errorf("absent routine: present=%v lines=%v, want false/empty", present, lines)
	}
}

// captureRoutinePreimages splits the target routines into captured (present) and
// absent (greenfield) sets.
func TestCaptureRoutinePreimages_Present(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "l1=XWBBRK ;hdr\n"}
	captured, absent, err := captureRoutinePreimages(context.Background(), fakeClient(f), []string{"XWBBRK"})
	if err != nil {
		t.Fatalf("captureRoutinePreimages: %v", err)
	}
	if len(captured) != 1 || captured[0].Name != "XWBBRK" || len(absent) != 0 {
		t.Errorf("captured=%v absent=%v", captured, absent)
	}
}

// End-to-end snapshot over the fake driver: reads the pre-image and writes a
// valid, re-parseable snapshot .KID.
func TestSnapshotWritesReparseableKID(t *testing.T) {
	f := &fakeDriver{runStdout: installspec.ResultMarker + "l1=XWBBRK ;stock\n" +
		installspec.ResultMarker + "l2= Q\n"}
	dir := t.TempDir()
	out := filepath.Join(dir, "preimage.kids")

	captured, _, err := captureRoutinePreimages(context.Background(), fakeClient(f), []string{"XWBBRK"})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	pairs := buildSnapshotPairs("XWBBRK*1.0*1 PREIMAGE", "XWBBRK", captured)
	if err := kids.WriteKID([]string{"XWBBRK*1.0*1 PREIMAGE"},
		map[string][]kids.Pair{"XWBBRK*1.0*1 PREIMAGE": pairs}, out); err != nil {
		t.Fatalf("WriteKID: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("snapshot not written: %v", err)
	}
	k, err := kids.ParseKID(out)
	if err != nil {
		t.Fatalf("re-parse snapshot: %v", err)
	}
	if len(k.InstallNames) != 1 || k.InstallNames[0] != "XWBBRK*1.0*1 PREIMAGE" {
		t.Errorf("snapshot install names = %v", k.InstallNames)
	}
}
