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

// TestChecksumCorpus validates the RoutineChecksumB port (the SUMB / line-2-blind
// "B" checksum, #3b) against the ENTIRE local corpus: nearly every routine whose RTN
// node carries a real B<n> must recompute byte-for-byte from its shipped source. A
// SUMB port bug would mis-flag THOUSANDS of pristine patches, so this is the
// at-scale regression guard.
//
// It does NOT require zero mismatches: ~1.6% of real national patches are "born
// self-inconsistent" (their stored checksum was computed on a slightly different
// routine version than what shipped — confirmed against the engine's own
// $$SUMB^XPDRSUM, which agrees with this port over the transport source). That is a
// corpus property, not a port bug, and is exactly why the install gate WARNS by
// default rather than refusing (see pkgcli/checksum.go). The guard therefore asserts
// the mismatch RATE stays low (a port regression would blow past it). Gated on
// VPKG_KIDS_CORPUS, same as the round-trip sweep (`make corpus` sets it).
func TestChecksumCorpus(t *testing.T) {
	root := os.Getenv("VPKG_KIDS_CORPUS")
	if root == "" {
		t.Skip("VPKG_KIDS_CORPUS not set — point it at a KIDS corpus dir to run the checksum sweep")
	}
	root = expandHome(root)

	var files []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if ext := strings.ToLower(filepath.Ext(path)); ext == ".kid" || ext == ".kids" {
				files = append(files, path)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walking corpus %q: %v", root, err)
	}
	sort.Strings(files)

	var checked, mismatch, skipped int
	var failures []string
	for _, f := range files {
		k, err := ParseKID(f)
		if err != nil {
			continue // parse coverage is the round-trip test's job
		}
		rel, _ := filepath.Rel(root, f)
		for _, name := range k.InstallNames {
			for _, r := range VerifyBuildChecksums(k.Builds[name]) {
				switch r.Verdict {
				case ChecksumOK:
					checked++
				case ChecksumMismatch:
					mismatch++
					if len(failures) < 40 {
						failures = append(failures, "MISMATCH "+rel+" "+name+":"+r.Name+" stored="+r.Stored+" recomputed="+r.Computed)
					}
				default: // no-stored (v-pkg builds don't appear here) / unknown-format
					skipped++
				}
			}
		}
	}
	total := checked + mismatch
	t.Logf("checksum sweep: OK=%d MISMATCH=%d SKIPPED=%d (no-stored/unknown) over %d files", checked, mismatch, skipped, len(files))
	if total > 0 {
		t.Logf("mismatch rate: %.2f%% (born-inconsistent corpus KIDs; ~1.6%% expected)", 100*float64(mismatch)/float64(total))
	}
	if checked == 0 {
		t.Fatalf("no B-checksummed routines verified in the corpus — the sweep proved nothing")
	}
	// A correct port reproduces the overwhelming majority; a port bug would flip
	// thousands from OK to MISMATCH. Allow generous headroom over the observed ~1.6%
	// (born-inconsistent KIDs) while still catching a real regression.
	if rate := float64(mismatch) / float64(total); rate > 0.05 {
		for _, m := range failures {
			t.Logf("%s", m)
		}
		t.Fatalf("checksum mismatch rate %.2f%% exceeds 5%% — RoutineChecksumB likely regressed", 100*rate)
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
