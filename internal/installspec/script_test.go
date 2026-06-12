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

func TestInstallScript_ProvenSequence(t *testing.T) {
	got := InstallScript("ZZSKEL*1.0*1", "ZZSKEL via v pkg install", zzskelPairs())

	// The non-interactive direct-populate skeleton (all four steps, in order).
	wantInOrder := []string{
		`D HOME^%ZIS`,                                // IO(0) for INIT^XPDID
		`S XPDST=0,XPDIT=1`,                          // INST prerequisites
		`S XPDA=$$INST^XPDIL1("ZZSKEL*1.0*1")`,       // real KIDS #9.7 entry
		`S ^XTMP("XPDI",0)=$$FMADD^XLFDT(DT,7)_U_DT`, // transport expiration
		`S ^XTMP("XPDI",XPDA,"BLD",1,0)="ZZSKEL*1.0*1^ZZSKEL^0^0"`,
		`D EN^XPDIJ`, // synchronous install
	}
	last := -1
	for _, want := range wantInOrder {
		i := strings.Index(got, want)
		if i < 0 {
			t.Errorf("generated script missing %q\n---\n%s", want, got)
			continue
		}
		if i < last {
			t.Errorf("statement %q out of order", want)
		}
		last = i
	}

	// Routine line 5 carries embedded quotes — they must be doubled.
	if !strings.Contains(got, `S ^XTMP("XPDI",XPDA,"RTN","ZZSKEL",5,0)=" quit ""pong"""`) {
		t.Errorf("RTN line 5 not M-escaped:\n%s", got)
	}

	// Full FM privilege is required for EN^XPDIJ.
	if !strings.Contains(got, `DUZ(0)="@"`) {
		t.Errorf("script must grant full FM priv (DUZ(0)=\"@\")")
	}

	// It must emit a parseable status marker the driver layer reads back.
	if !strings.Contains(got, ResultMarker+"status=") {
		t.Errorf("script must emit %sstatus=", ResultMarker)
	}
}

// An already-present build must be refused before INST (which would otherwise
// prompt "OK to continue with Load" with no stdin over the driver Exec).
func TestInstallScript_GuardsAlreadyInstalled(t *testing.T) {
	got := InstallScript("ZZSKEL*1.0*1", "h", zzskelPairs())
	if !strings.Contains(got, `I $D(^XPD(9.7,"B","ZZSKEL*1.0*1"))`) {
		t.Errorf("script must guard on an existing #9.7 entry:\n%s", got)
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
