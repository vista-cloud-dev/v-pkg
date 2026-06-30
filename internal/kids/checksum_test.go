package kids

import "testing"

// zztSource is a synthetic routine whose VistA checksums were ground-truthed on a
// live engine via $$SUMB^XPDRSUM / $$SUMA^XPDRSUM (vehu, 2026-06-30): SUMB=10838,
// SUMA=5458. RoutineChecksumB ports the SUMB ("B") surface, which is what KIDS
// stores in a transport RTN node.
var zztSource = []string{
	"ZZT ;test routine ;1.0",
	" ;;1.0;ZZT;;",
	" Q",
	"PING() ;",
	` Q "pong"`,
}

// RoutineChecksumB must reproduce the engine's $$SUMB^XPDRSUM value byte-for-byte —
// it is line-2-blind (the ;;version line is excluded from the count).
func TestRoutineChecksumB_GroundTruth(t *testing.T) {
	if got := RoutineChecksumB(zztSource); got != 10838 {
		t.Errorf("RoutineChecksumB = %d, want 10838 (engine $$SUMB^XPDRSUM)", got)
	}
}

// The checksum is line-2-blind: changing ONLY line 2 (the ;;version/patch line KIDS
// rewrites at install) must NOT change the value — that is the whole point.
func TestRoutineChecksumB_Line2Blind(t *testing.T) {
	mutated := append([]string(nil), zztSource...)
	mutated[1] = " ;;9.9;ZZT;**1,2,3**;Jan 1, 2026;Build 7"
	if got := RoutineChecksumB(mutated); got != 10838 {
		t.Errorf("RoutineChecksumB changed when only line 2 changed: got %d, want 10838", got)
	}
}

// Changing any COMMAND line (the checksum surface) MUST change the value — a tamper
// is detectable.
func TestRoutineChecksumB_CommandLineChanges(t *testing.T) {
	mutated := append([]string(nil), zztSource...)
	mutated[4] = ` Q "pongX"` // flip one command line
	if got := RoutineChecksumB(mutated); got == 10838 {
		t.Errorf("RoutineChecksumB unchanged after a command-line tamper (%d) — not detecting tamper", got)
	}
}

// RoutineChecksum reads the stored checksum (piece 3 of the 2-subscript RTN node) —
// "0" for a v-pkg build (real checksums computed at install), "B<n>" for an ingested
// foreign .KID.
func TestBuild_RoutineChecksum(t *testing.T) {
	// A v-pkg build stores 0/0 — nothing to verify.
	vpkg := MakeBuildPairs(BuildInput{
		InstallName: "ZZT*1.0*1", Namespace: "ZZT",
		Routines: []RoutineSrc{{Name: "ZZT", Lines: zztSource}},
	})
	b := newBuild()
	for _, p := range vpkg {
		b.Set(p.Subs, p.Value)
	}
	if got := b.RoutineChecksum("ZZT"); got != "0" {
		t.Errorf("v-pkg build RoutineChecksum(ZZT) = %q, want \"0\"", got)
	}
	if got := b.RoutineChecksum("NOSUCH"); got != "" {
		t.Errorf("RoutineChecksum(absent) = %q, want \"\"", got)
	}
}

// VerifyRoutineChecksum: a 0/empty stored checksum is a v-pkg build (no-stored,
// skipped); a matching B<n> is ok; a non-matching B<n> is a mismatch (tamper).
func TestVerifyRoutineChecksum(t *testing.T) {
	cases := []struct {
		name   string
		stored string
		lines  []string
		want   ChecksumVerdict
	}{
		{"v-pkg 0 -> no stored", "0", zztSource, ChecksumNoStored},
		{"empty -> no stored", "", zztSource, ChecksumNoStored},
		{"matching B value -> ok", "B10838", zztSource, ChecksumOK},
		{"wrong B value -> mismatch", "B99999999", zztSource, ChecksumMismatch},
		{"unrecognized format -> unknown (cannot verify, not a tamper)", "X123", zztSource, ChecksumUnknownFormat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := VerifyRoutineChecksum(tc.stored, tc.lines)
			if got.Verdict != tc.want {
				t.Errorf("VerifyRoutineChecksum(%q) verdict = %q, want %q (computed %q)", tc.stored, got.Verdict, tc.want, got.Computed)
			}
		})
	}
}
