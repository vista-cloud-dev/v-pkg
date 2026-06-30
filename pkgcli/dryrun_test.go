package pkgcli

import (
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

func TestClassifyRoutine(t *testing.T) {
	cases := []struct{ drift, want string }{
		{"absent", planNew},        // not on the engine — install adds it
		{"applied", planIdentical}, // live already matches the shipped source — no-op
		{"drifted", planChanged},   // resident differs from incoming — install overwrites
	}
	for _, c := range cases {
		if got := classifyRoutine(c.drift); got != c.want {
			t.Errorf("classifyRoutine(%q) = %q, want %q", c.drift, got, c.want)
		}
	}
}

func TestClassifyContent(t *testing.T) {
	cases := []struct{ content, want string }{
		{"absent", planWouldAdd},      // no record by this name — install would add it
		{"mismatch", planWouldChange}, // present but differs — install would change it
		{"ok", planIdentical},         // present and matches — no-op
	}
	for _, c := range cases {
		if got := classifyContent(c.content); got != c.want {
			t.Errorf("classifyContent(%q) = %q, want %q", c.content, got, c.want)
		}
	}
}

// TestAssembleDryRun checks the pure plan-assembly from canned engine verdicts:
// routines relabel from the drift probe; content-verify components use the 0-node
// compare; presence-only components (no content map entry) fall back to the
// "B"-index probe and NEVER claim would-change; FILE DD fields use the 0-node
// compare; the summary rolls up the four verdict buckets.
func TestAssembleDryRun(t *testing.T) {
	comps := []kids.Component{
		{File: 19, FileStr: "19", Names: []string{"ZZOPT"}},       // content-verify type
		{File: 0.402, FileStr: ".402", Names: []string{"ZZTMPL"}}, // presence-only type (present)
		{File: 3.6, FileStr: "3.6", Names: []string{"ZZBULL"}},    // presence-only type (absent)
	}
	files := []kids.FileContent{
		{File: 999001, FileStr: "999001", Field: "0.01"},
	}
	drift := map[string]string{"ZZR1": "absent", "ZZR2": "applied", "ZZR3": "drifted"}
	content := map[string]string{
		"19:ZZOPT":    "ok",     // content-verify -> identical
		"999001#0.01": "absent", // FILE DD field -> would-add
	}
	presence := map[string]bool{
		"19:ZZOPT":    true,  // ignored (content verdict wins)
		".402:ZZTMPL": true,  // presence-only present -> present
		"3.6:ZZBULL":  false, // presence-only absent -> would-add
	}

	rep := assembleDryRun("ZZ TEST 1.0", "pure-overwrite", drift, content, presence, comps, files)

	wantRtn := map[string]string{"ZZR1": planNew, "ZZR2": planIdentical, "ZZR3": planChanged}
	for k, want := range wantRtn {
		if got := rep.Routines[k]; got != want {
			t.Errorf("routine %s = %q, want %q", k, got, want)
		}
	}
	wantComp := map[string]string{
		"19:ZZOPT":    planIdentical,
		".402:ZZTMPL": planPresent,
		"3.6:ZZBULL":  planWouldAdd,
	}
	for k, want := range wantComp {
		if got := rep.Components[k]; got != want {
			t.Errorf("component %s = %q, want %q", k, got, want)
		}
	}
	if got := rep.Files["999001#0.01"]; got != planWouldAdd {
		t.Errorf("file 999001#0.01 = %q, want %q", got, planWouldAdd)
	}
	// Summary roll-up: New = ZZR1 + ZZBULL + FILE(999001) = 3; Changed = ZZR3 = 1;
	// Identical = ZZR2 + ZZOPT = 2; Present = ZZTMPL = 1.
	want := dryRunSummary{New: 3, Changed: 1, Identical: 2, Present: 1}
	if rep.Summary != want {
		t.Errorf("summary = %+v, want %+v", rep.Summary, want)
	}
	if rep.Name != "ZZ TEST 1.0" || rep.Class != "pure-overwrite" {
		t.Errorf("name/class = %q/%q", rep.Name, rep.Class)
	}
}
