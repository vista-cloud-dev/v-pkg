package pkgcli

import "testing"

// snapshotShouldWrite must NOT re-capture/overwrite the pre-image sidecar when the
// build is already installed: at that point the routines are in their post-install
// (overwritten) state, so a fresh snapshot captures the WRONG source and clobbers
// the genuine pre-image — the only artifact that can back out a foreign/national
// overwrite. Regression for the redundant-install-clobbers-back-out bug.
func TestSnapshotShouldWrite(t *testing.T) {
	cases := []struct {
		name             string
		action           installAction
		alreadyInstalled bool
		want             bool
	}{
		{name: "fresh overwrite -> write the pre-image", action: instSnapshotProceed, alreadyInstalled: false, want: true},
		{name: "ALREADY installed -> do NOT clobber the pre-image", action: instSnapshotProceed, alreadyInstalled: true, want: false},
		{name: "greenfield proceed -> nothing to snapshot", action: instProceed, alreadyInstalled: false, want: false},
		{name: "refused -> nothing to snapshot", action: instRefuse, alreadyInstalled: false, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := snapshotShouldWrite(tc.action, tc.alreadyInstalled); got != tc.want {
				t.Errorf("snapshotShouldWrite(%v,%v) = %v, want %v", tc.action, tc.alreadyInstalled, got, tc.want)
			}
		})
	}
}

func TestDecideInstall(t *testing.T) {
	cases := []struct {
		name           string
		hasExisting    bool
		snapshot       string
		allowOverwrite bool
		want           installAction
	}{
		{name: "pure greenfield -> proceed", hasExisting: false, want: instProceed},
		{name: "greenfield ignores flags -> proceed", hasExisting: false, snapshot: "s.kids", want: instProceed},
		{name: "overwrite, no flags -> REFUSE (no silent clobber)", hasExisting: true, want: instRefuse},
		{name: "overwrite with --snapshot -> snapshot then proceed", hasExisting: true, snapshot: "pre.kids", want: instSnapshotProceed},
		{name: "overwrite with --allow-overwrite -> proceed (unsafe ack)", hasExisting: true, allowOverwrite: true, want: instProceed},
		{name: "overwrite with both -> snapshot wins (safer)", hasExisting: true, snapshot: "pre.kids", allowOverwrite: true, want: instSnapshotProceed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := decideInstall(tc.hasExisting, tc.snapshot, tc.allowOverwrite)
			if got != tc.want {
				t.Errorf("decideInstall(%v,%q,%v) = %v (%q), want %v",
					tc.hasExisting, tc.snapshot, tc.allowOverwrite, got, reason, tc.want)
			}
			if reason == "" {
				t.Errorf("expected a non-empty reason for action %v", got)
			}
		})
	}
}
