package pkgcli

import "testing"

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
