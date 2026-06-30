package kids

import (
	"strconv"
	"strings"
)

// This file ports VistA's routine-checksum surface so a shipped .KID can be
// tamper/corruption-checked OFFLINE before it is staged to an engine (the KIDS
// convention's "Verify Checksums in Transport Global"). A KIDS transport stores each
// routine's checksum in its 2-subscript RTN node value, `0^<numlines>^<checksum>…`,
// where <checksum> is `B<n>` — the "new", line-2-blind checksum $$SUMB^XPDRSUM /
// ^%ZOSF("RSUM1") computes. v-pkg's OWN builds strip that to "0" (real checksums are
// computed at install, see buildkids.go), so the verify is meaningful only for
// INGESTED/foreign .KIDs that carry a real B<n>; a 0/empty stored checksum is skipped,
// never false-failed.

// RoutineChecksumB computes VistA's "new" (line-2-blind) routine checksum over a
// routine's source lines — a bug-for-bug port of $$SUMB^XPDRSUM (≡ ^%ZOSF("RSUM1")):
//
//	S Y=0 F %=1,3:1 S %1=line(%),%3=$F(%1," ") Q:'%3
//	  S %3=$S($E(%1,%3)'=";":$L(%1), $E(%1,%3+1)=";":$L(%1), 1:%3-2)
//	  F %2=1:1:%3 S Y=$A(%1,%2)*(%2+%)+Y
//
// Line 2 (the ;;version/patch-list line KIDS rewrites at install) is excluded; a line
// with no space terminates the count (as in the M, where an empty line past the end
// makes $F return 0). A single-";" comment line contributes only its leading tag
// (everything before the first space); a ";;" line and code lines contribute in full.
// The recomputed value byte-matches the B<n> a real .KID carries, so a mismatch is a
// corrupted/tampered routine.
func RoutineChecksumB(lines []string) int64 {
	var y int64
	for n := 1; ; n++ { // % = 1, then 3, 4, 5, … (line 2 is skipped below)
		if n == 2 {
			continue
		}
		var line string
		if n-1 < len(lines) { // 1-indexed; past the end = "" (terminates)
			line = lines[n-1]
		}
		sp := strings.IndexByte(line, ' ') // 0-based index of the first space
		if sp < 0 {                        // $F(line," ")=0 → Q:'%3 ends the whole count
			break
		}
		p3 := sp + 2 // $F returns the 1-based position AFTER the space = (sp+1)+1
		switch {
		case mChar(line, p3) != ';': // not a comment right after the first space → whole line
			p3 = len(line)
		case mChar(line, p3+1) == ';': // a ";;" line → whole line
			p3 = len(line)
		default: // a single-";" comment → only the leading tag (before the space)
			p3 = p3 - 2
		}
		for c := 1; c <= p3; c++ { // $A(%1,%2)*(%2+%)
			if c-1 < len(line) {
				y += int64(line[c-1]) * int64(c+n)
			}
		}
	}
	return y
}

// mChar returns the byte at 1-based position k of s (M's $E(s,k)), or 0 when k is out
// of range — matching $E's "" (which compares unequal to ";").
func mChar(s string, k int) byte {
	if k >= 1 && k <= len(s) {
		return s[k-1]
	}
	return 0
}

// RoutineChecksum returns the checksum string stored in the build's transport RTN
// node for routine name — piece 3 of the 2-subscript `"RTN",<name>)` value
// (`0^<numlines>^<checksum>…`). "" if the build ships no such node. A v-pkg build
// stores "0" (real checksum computed at install); an ingested foreign .KID stores the
// real `B<n>`.
func (b *Build) RoutineChecksum(name string) string {
	for _, p := range b.Pairs() {
		s := p.Subs
		if len(s) == 2 && s[0].IsStr() && s[0].Str() == "RTN" && s[1].IsStr() && s[1].Str() == name {
			parts := strings.Split(p.Value, "^")
			if len(parts) >= 3 {
				return parts[2]
			}
			return ""
		}
	}
	return ""
}

// ChecksumVerdict grades one routine's transport-checksum check.
type ChecksumVerdict string

const (
	// ChecksumOK: the recomputed B-checksum matches the .KID's stored value.
	ChecksumOK ChecksumVerdict = "ok"
	// ChecksumMismatch: a stored B-checksum that the source does NOT reproduce — a
	// corrupted or tampered routine. This is the refuse signal.
	ChecksumMismatch ChecksumVerdict = "mismatch"
	// ChecksumNoStored: a "0"/empty stored checksum (a v-pkg build) — nothing to
	// verify; skipped, never false-failed.
	ChecksumNoStored ChecksumVerdict = "no-stored-checksum"
	// ChecksumUnknownFormat: a stored checksum in a form we cannot recompute (not "0"
	// and not "B<n>"). We cannot prove tamper, so we do not refuse — skipped + surfaced.
	ChecksumUnknownFormat ChecksumVerdict = "unknown-format"
)

// ChecksumResult is one routine's checksum verdict.
type ChecksumResult struct {
	Name     string          `json:"name"`
	Stored   string          `json:"stored"`             // the .KID's stored checksum (e.g. "B51345879" or "0")
	Computed string          `json:"computed,omitempty"` // the recomputed value (B-format only)
	Verdict  ChecksumVerdict `json:"verdict"`
}

// VerifyRoutineChecksum grades a stored checksum against the routine's shipped source.
// Only a "B<n>" stored value is recomputed (the format KIDS ships); "0"/"" is skipped
// (v-pkg build), and any other form is unknown-format (cannot verify → not a tamper).
func VerifyRoutineChecksum(stored string, lines []string) ChecksumResult {
	r := ChecksumResult{Stored: stored}
	switch {
	case stored == "" || stored == "0":
		r.Verdict = ChecksumNoStored
	case strings.HasPrefix(stored, "B"):
		got := strconv.FormatInt(RoutineChecksumB(lines), 10)
		r.Computed = "B" + got
		if got == stored[1:] {
			r.Verdict = ChecksumOK
		} else {
			r.Verdict = ChecksumMismatch
		}
	default:
		r.Verdict = ChecksumUnknownFormat
	}
	return r
}

// VerifyBuildChecksums grades every routine the build ships, in build order.
func VerifyBuildChecksums(b *Build) []ChecksumResult {
	names := b.RoutineNames()
	out := make([]ChecksumResult, 0, len(names))
	for _, name := range names {
		r := VerifyRoutineChecksum(b.RoutineChecksum(name), b.RoutineSource(name))
		r.Name = name
		out = append(out, r)
	}
	return out
}

// ChecksumMismatches returns just the results that failed verification (the refuse
// set). An empty slice means every shipped, checksummed routine verified (or there
// was nothing to verify).
func ChecksumMismatches(results []ChecksumResult) []ChecksumResult {
	var bad []ChecksumResult
	for _, r := range results {
		if r.Verdict == ChecksumMismatch {
			bad = append(bad, r)
		}
	}
	return bad
}
