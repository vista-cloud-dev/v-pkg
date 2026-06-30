package pkgcli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vista-cloud-dev/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// Transport-checksum verification (verifiable-safety #3b — the KIDS convention's
// "Verify Checksums in Transport Global"). For each shipped routine we recompute the
// line-2-blind checksum (kids.RoutineChecksumB) and compare it to the value the .KID
// stores in its RTN node. A mismatch means the routine's source and its stored
// checksum disagree — accidental transit corruption, a botched edit, or tampering.
//
// IMPORTANT (corpus-grounded, 2026-06-30): this is a SELF-consistency check — the
// source and the checksum both live in the .KID — over a NON-cryptographic sum
// (Σ ascii·(pos+line)). It catches accidental corruption of routine bytes, but a
// deliberate tamperer who also recomputes the (forgeable) checksum defeats it; real
// tamper-resistance needs a signature over the whole .KID against an external key
// (increment #4, attestation). And ~6% of real foreign corpus patches are "born
// self-inconsistent" — their stored checksum was computed on a slightly different
// routine version than what shipped — indistinguishable offline from a tamper. So a
// mismatch WARNS by default (never false-fails a legitimate install); the hard
// `--verify-checksums` gate is opt-in for operators who want to refuse on any
// mismatch. v-pkg's own builds store "0" and are silent here.

// checksumScan returns, per install name, the routines whose stored checksum does NOT
// reproduce from their shipped source. An empty map means everything verified (or
// nothing carried a real checksum — a v-pkg 0/0 build).
func checksumScan(k *kids.KID) map[string][]kids.ChecksumResult {
	out := map[string][]kids.ChecksumResult{}
	for _, name := range k.InstallNames {
		if bad := kids.ChecksumMismatches(kids.VerifyBuildChecksums(k.Builds[name])); len(bad) > 0 {
			out[name] = bad
		}
	}
	return out
}

// checksumLines renders a scan's mismatches as deterministic, human-readable lines
// ("<build>:<routine> (stored …, recomputed …)"), sorted so output is stable.
func checksumLines(scan map[string][]kids.ChecksumResult) []string {
	var lines []string
	for name, bad := range scan {
		for _, m := range bad {
			lines = append(lines, fmt.Sprintf("%s:%s (stored %s, recomputed %s)", name, m.Name, m.Stored, m.Computed))
		}
	}
	sort.Strings(lines)
	return lines
}

// checksumRefusal is the --verify-checksums hard gate: a *clikit.Error refusing the
// install when any routine's checksum mismatches, else nil. Used only in opt-in
// strict mode; the default path warns instead (checksumLines surfaced on the report).
func checksumRefusal(scan map[string][]kids.ChecksumResult) *clikit.Error {
	lines := checksumLines(scan)
	if len(lines) == 0 {
		return nil
	}
	return clikit.Fail(clikit.ExitRefused, "CHECKSUM_MISMATCH",
		"refusing to install (--verify-checksums): transport checksum mismatch — the .KID is corrupted, tampered, or carries an inconsistent stored checksum: "+strings.Join(lines, "; "),
		"re-fetch the .KID from a trusted source; drop --verify-checksums to install with a warning instead")
}
