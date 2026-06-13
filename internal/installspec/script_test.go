package installspec

import (
	"strings"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// zzskelPairs builds the same ZZSKEL transport-global pairs `v pkg build`
// produces, so the generated install script can be checked against the live-
// proven sequence (kids-installation-automation.md §7.1).
func zzskelPairs() []kids.Pair {
	return kids.MakeBuildPairs(kids.BuildInput{
		InstallName: "ZZSKEL*1.0*1",
		Namespace:   "ZZSKEL",
		Routines: []kids.RoutineSrc{{
			Name: "ZZSKEL",
			Lines: []string{
				"ZZSKEL ;VCD/VSL - throwaway test package (M0a ZZSKEL) ;1.0",
				" ;;1.0;ZZSKEL;;",
				" quit",
				"PING() ;",
				` quit "pong"`,
			},
		}},
	})
}

// StageChunks splits the transport global into size-bounded routine bodies that
// populate the ^XTMP("VPKGI",…) staging global — never one routine big enough to
// exceed the driver's per-routine staging limit (which silently truncates large
// loads, T0b.2 discoveries P1).
func TestStageChunks_CoversAllPairsBounded(t *testing.T) {
	pairs := zzskelPairs()
	const maxBytes = 80
	chunks := StageChunks(pairs, maxBytes)
	if len(chunks) < 2 {
		t.Fatalf("want multiple chunks at maxBytes=%d, got %d", maxBytes, len(chunks))
	}
	// The first chunk clears any stale staging global before populating it.
	if !strings.HasPrefix(chunks[0], `K ^XTMP("VPKGI")`) {
		t.Errorf("first chunk must clear the staging global, got:\n%s", chunks[0])
	}
	all := strings.Join(chunks, "\n")
	// Every pair is staged exactly once, into ^XTMP("VPKGI",…) (not the live
	// ^XTMP("XPDI",XPDA,…) — that is filled by the finalize MERGE).
	for _, p := range pairs {
		ref := "S " + p.Subs.MRef(`^XTMP("VPKGI",`) + "="
		if n := strings.Count(all, ref); n != 1 {
			t.Errorf("pair %q staged %d times, want 1", ref, n)
		}
	}
	// Embedded quotes in routine source are doubled.
	if !strings.Contains(all, `S ^XTMP("VPKGI","RTN","ZZSKEL",5,0)=" quit ""pong"""`) {
		t.Errorf("RTN line 5 not M-escaped:\n%s", all)
	}
	// Each chunk stays bounded (a lone over-long SET may exceed maxBytes, but a
	// chunk never accumulates well past it).
	for i, c := range chunks {
		if len(c) > maxBytes+256 {
			t.Errorf("chunk %d = %d bytes, exceeds the bound", i, len(c))
		}
	}
}

// FinalInstallScript verifies the staged count, then installs in one process:
// INST → MERGE the staged tree into ^XTMP("XPDI",XPDA) → EN^XPDIJ.
func TestFinalInstallScript(t *testing.T) {
	got := FinalInstallScript("ZZSKEL*1.0*1", "ZZSKEL via v pkg install", 7)
	for _, want := range []string{
		`I $D(^XPD(9.7,"B","ZZSKEL*1.0*1"))`,    // already-installed guard
		`DUZ(0)="@"`,                            // full FM priv for EN^XPDIJ
		ResultMarker + `staged=`,                // staged-node count marker
		`I VC'=7`,                               // truncation guard: count must match
		ResultMarker + `error=stage-incomplete`, // …else refuse, do not install partial
		`S XPDA=$$INST^XPDIL1("ZZSKEL*1.0*1")`,  // real KIDS #9.7 entry
		`M ^XTMP("XPDI",XPDA)=^XTMP("VPKGI")`,   // staged tree → live transport global
		`D EN^XPDIJ`,                            // synchronous install (same process)
		`K ^XTMP("VPKGI")`,                      // clean the staging global
		ResultMarker + `status=`,                // #9.7 status marker
	} {
		if !strings.Contains(got, want) {
			t.Errorf("FinalInstallScript missing %q\n---\n%s", want, got)
		}
	}
}

func TestVerifyScript(t *testing.T) {
	got := VerifyScript("ZZSKEL*1.0*1", []string{"ZZSKEL"})
	for _, want := range []string{
		`S XPDA=$O(^XPD(9.7,"B","ZZSKEL*1.0*1",0))`,
		ResultMarker + `installed=`,
		ResultMarker + `status=`,
		`$T(^ZZSKEL)`,                // routine-presence probe
		ResultMarker + `rtn:ZZSKEL=`, // per-routine marker
	} {
		if !strings.Contains(got, want) {
			t.Errorf("VerifyScript missing %q\n---\n%s", want, got)
		}
	}
}

func TestUninstallScript(t *testing.T) {
	got := UninstallScript("ZZSKEL*1.0*1", []string{"ZZSKEL"})
	for _, want := range []string{
		`S X="ZZSKEL" X ^%ZOSF("DEL")`,                                        // routine delete
		`S DA=$O(^XPD(9.7,"B","ZZSKEL*1.0*1",0)),DIK="^XPD(9.7," I DA D ^DIK`, // #9.7
		`S DA=$O(^XPD(9.6,"B","ZZSKEL*1.0*1",0)),DIK="^XPD(9.6," I DA D ^DIK`, // #9.6
		ResultMarker + `uninstalled=1`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("UninstallScript missing %q\n---\n%s", want, got)
		}
	}
}
