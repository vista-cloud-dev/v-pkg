package pkgcli

import (
	"path/filepath"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

const preimageName = "ZZT*1.0*1 PREIMAGE"

// sidecarBuild writes pairs to a temp .KID and re-parses them — the same
// WriteKID→ParseKID path the real sidecar takes — and returns the parsed build.
func sidecarBuild(t *testing.T, pairs []kids.Pair) *kids.Build {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pre.kids")
	if err := kids.WriteKID([]string{preimageName}, map[string][]kids.Pair{preimageName: pairs}, path); err != nil {
		t.Fatal(err)
	}
	k, err := kids.ParseKID(path)
	if err != nil {
		t.Fatal(err)
	}
	return k.Builds[preimageName]
}

func preimagePairs() []kids.Pair {
	return kids.MakeBuildPairs(kids.BuildInput{
		InstallName: preimageName, Namespace: "ZZT",
		Routines: []kids.RoutineSrc{{Name: "ZZT", Lines: []string{
			"ZZT ;pre-image ;1.0", " ;;1.0;ZZT;;", " Q",
		}}},
	})
}

// verifySidecarIntegrity must: pass a pristine stamped sidecar, REFUSE a tampered one
// (SIDECAR_TAMPERED), and pass (nothing to verify) an unstamped one.
func TestVerifySidecarIntegrity(t *testing.T) {
	// Pristine stamped sidecar → nil.
	if ferr := verifySidecarIntegrity(sidecarBuild(t, kids.StampHash(preimagePairs())), "pre.kids"); ferr != nil {
		t.Errorf("pristine stamped sidecar must pass, got: %v", ferr)
	}

	// Tampered: stamp, then alter a routine line — the recomputed hash diverges.
	tampered := sidecarBuild(t, kids.StampHash(preimagePairs()))
	for _, p := range preimagePairs() {
		if len(p.Subs) == 4 { // a routine source line node "RTN",name,n,0
			tampered.Set(p.Subs, p.Value+" ; evil")
			break
		}
	}
	ferr := verifySidecarIntegrity(tampered, "pre.kids")
	if ferr == nil {
		t.Fatal("tampered sidecar must be REFUSED")
	}
	if ferr.Code != "SIDECAR_TAMPERED" {
		t.Errorf("tampered sidecar code = %q, want SIDECAR_TAMPERED", ferr.Code)
	}

	// Unstamped (authored back-out / pre-3c snapshot): nothing to verify → nil.
	if ferr := verifySidecarIntegrity(sidecarBuild(t, preimagePairs()), "backout.kids"); ferr != nil {
		t.Errorf("unstamped sidecar must pass (nothing to verify), got: %v", ferr)
	}
}
