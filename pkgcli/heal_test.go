package pkgcli

import "testing"

// classifyHeal grades a #9.7 entry from the read-only heal-detect probe into one of
// three states. A healthy install (0-node present AND status 3) must NEVER be graded
// corrupt — heal is a targeted purge of a PROVEN-corrupt entry only.
func TestClassifyHeal(t *testing.T) {
	cases := []struct {
		name        string
		ien         int
		zeroPresent bool
		status      int
		want        healState
	}{
		{"no #9.7 entry -> not installed", 0, false, 0, healNotInstalled},
		{"complete install (0-node + status 3) -> healthy", 12, true, 3, healHealthy},
		{"xref present, NO 0-node -> corrupt", 12, false, 0, healCorrupt},
		{"xref + 0-node but status 2 (stuck mid-install) -> corrupt", 12, true, 2, healCorrupt},
		{"xref + 0-node but empty status -> corrupt", 12, true, 0, healCorrupt},
		// A stray B xref with no IEN (ien=0) can never be reached — ien gates everything.
		{"ien 0 ignores stale flags -> not installed", 0, true, 3, healNotInstalled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyHeal(tc.ien, tc.zeroPresent, tc.status); got != tc.want {
				t.Errorf("classifyHeal(%d,%v,%d) = %v, want %v", tc.ien, tc.zeroPresent, tc.status, got, tc.want)
			}
		})
	}
}

// decideHeal maps a state to the action `install --heal` takes: purge a corrupt
// entry then install, refuse a healthy one (nothing to heal), or proceed when there
// is no prior entry at all.
func TestDecideHeal(t *testing.T) {
	cases := []struct {
		name  string
		state healState
		want  healAction
	}{
		{"corrupt -> purge then proceed", healCorrupt, healPurgeProceed},
		{"healthy -> refuse (never touch a healthy install)", healHealthy, healRefuse},
		{"not installed -> proceed (nothing to heal)", healNotInstalled, healProceed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := decideHeal(tc.state)
			if got != tc.want {
				t.Errorf("decideHeal(%v) = %v (%q), want %v", tc.state, got, reason, tc.want)
			}
			if reason == "" {
				t.Errorf("decideHeal(%v): expected a non-empty reason", tc.state)
			}
		})
	}
}
