package kids

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestRoundtripCorpus runs the round-trip oracle over an ENTIRE local KIDS
// corpus — every real-world distribution a developer can point it at — to prove
// decompose→assemble→re-parse is canonically lossless against the full diversity
// of national patches, not just the five committed fixtures.
//
// It is gated on the VPKG_KIDS_CORPUS env var (a directory tree of .KID/.KIDS
// files; subdirectories are walked). When unset the test SKIPS, so CI — which
// has no corpus checked out — stays green; a developer with the WorldVistA mirror
// runs it with:
//
//	VPKG_KIDS_CORPUS=~/data/kids-patches/VistA/Packages go test ./internal/kids/ -run Corpus -v
//
// or `make corpus` (which sets the var). Every file that fails to round-trip is
// reported (file + first divergence), not just the first — so one run surfaces
// the full set of parser gaps to fix.
func TestRoundtripCorpus(t *testing.T) {
	root := os.Getenv("VPKG_KIDS_CORPUS")
	if root == "" {
		t.Skip("VPKG_KIDS_CORPUS not set — point it at a KIDS corpus dir to run the full sweep")
	}
	root = expandHome(root)
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		t.Fatalf("VPKG_KIDS_CORPUS=%q is not a readable directory: %v", root, err)
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext == ".kid" || ext == ".kids" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking corpus %q: %v", root, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no .KID/.KIDS files found under %q", root)
	}
	t.Logf("corpus: %d KIDS distributions under %s", len(files), root)

	var pass, drift, errs int
	var pure, side int
	var failures []string
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		res, err := Roundtrip(f)
		switch {
		case err != nil:
			errs++
			failures = append(failures, "ERROR "+rel+": "+err.Error())
		case !res.OK:
			drift++
			msg := "DRIFT " + rel
			for _, d := range res.Diff {
				msg += "\n    build " + d.Build + ":\n    - " + d.FirstA + "\n    + " + d.FirstB
			}
			failures = append(failures, msg)
		default:
			pass++
		}
		// Classify every parseable file so one sweep also validates the
		// reversibility classifier at corpus scale (and documents the split).
		if k, perr := ParseKID(f); perr == nil {
			if Classify(k).Class == ClassPureOverwrite {
				pure++
			} else {
				side++
			}
		}
	}

	t.Logf("roundtrip sweep: PASS=%d DRIFT=%d ERROR=%d (total %d)", pass, drift, errs, len(files))
	classified := pure + side
	if classified > 0 {
		t.Logf("reversibility:  pure-overwrite=%d (%d%%)  side-effecting=%d (%d%%)",
			pure, 100*pure/classified, side, 100*side/classified)
	}
	if len(failures) > 0 {
		for _, f := range failures {
			t.Errorf("%s", f)
		}
		t.Fatalf("%d of %d corpus files did not round-trip", drift+errs, len(files))
	}
}

// expandHome resolves a leading ~ to the user's home dir (test-only convenience
// so VPKG_KIDS_CORPUS=~/data/... works without shell expansion).
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
