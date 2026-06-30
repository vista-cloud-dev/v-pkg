package pkgcli

import (
	"reflect"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

func TestDecideUninstall(t *testing.T) {
	const (
		pure = kids.ClassPureOverwrite
		side = kids.ClassSideEffecting
	)
	cases := []struct {
		name       string
		class      kids.ReversibilityClass
		restore    string
		backout    string
		force      bool
		greenfield bool // build adds routines beyond the pre-image (only meaningful with a pre-image)
		wantAction uninstallAction
	}{
		{name: "class-1, no flags -> greenfield delete", class: pure, wantAction: actDelete},
		{name: "class-1 with --restore (pure overwrite) -> restore pre-image", class: pure, restore: "pre.kids", wantAction: actRestore},
		{name: "side-effecting, no flags -> REFUSE (the safety fix)", class: side, wantAction: actRefuse},
		{name: "side-effecting with --backout -> run authored back-out", class: side, backout: "bo.kids", wantAction: actBackout},
		{name: "side-effecting with --restore -> restore (routines only)", class: side, restore: "pre.kids", wantAction: actRestore},
		{name: "side-effecting with --force -> delete anyway", class: side, force: true, wantAction: actDelete},
		{name: "both --restore and --backout -> refuse (ambiguous)", class: pure, restore: "p.kids", backout: "b.kids", wantAction: actRefuse},
		// BB1: a build that BOTH overwrote a foreign routine (in the pre-image) AND
		// added greenfield routines (not in the pre-image) must PARTITION — restore
		// the foreign, delete the greenfield — not just restore (orphans the adds)
		// nor delete-all (bricks the foreign national routine).
		{name: "class-1 with --restore AND greenfield adds -> PARTITION", class: pure, restore: "pre.kids", greenfield: true, wantAction: actPartition},
		{name: "side-effecting with --restore AND greenfield adds -> PARTITION", class: side, restore: "pre.kids", greenfield: true, wantAction: actPartition},
		{name: "greenfield flag without --restore is ignored -> delete", class: pure, greenfield: true, wantAction: actDelete},
		{name: "--restore+--backout with greenfield still refuses (ambiguous)", class: pure, restore: "p.kids", backout: "b.kids", greenfield: true, wantAction: actRefuse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := decideUninstall(tc.class, tc.restore, tc.backout, tc.force, tc.greenfield)
			if got != tc.wantAction {
				t.Errorf("decideUninstall = %v (%q), want %v", got, reason, tc.wantAction)
			}
			if reason == "" {
				t.Errorf("expected a non-empty reason for action %v", got)
			}
		})
	}
}

func TestPartitionRoutines(t *testing.T) {
	cases := []struct {
		name        string
		build       []string
		preimage    []string
		wantRestore []string
		wantDelete  []string
	}{
		{
			name:        "tap build: XWBPRS overwritten (restore), VSLRT* added (delete)",
			build:       []string{"VSLRTAP", "VSLRTH", "VSLRTRP", "XWBPRS"},
			preimage:    []string{"XWBPRS"},
			wantRestore: []string{"XWBPRS"},
			wantDelete:  []string{"VSLRTAP", "VSLRTH", "VSLRTRP"},
		},
		{
			name:        "pure greenfield (no pre-image) -> all delete, none restore",
			build:       []string{"VSLRTAP", "VSLRTH"},
			preimage:    nil,
			wantRestore: nil,
			wantDelete:  []string{"VSLRTAP", "VSLRTH"},
		},
		{
			name:        "pure overwrite (every routine in pre-image) -> all restore, none delete",
			build:       []string{"XWBPRS", "XWBSEC"},
			preimage:    []string{"XWBPRS", "XWBSEC"},
			wantRestore: []string{"XWBPRS", "XWBSEC"},
			wantDelete:  nil,
		},
		{
			name:        "build order is preserved within each partition",
			build:       []string{"AAA", "BBB", "CCC", "DDD"},
			preimage:    []string{"BBB", "DDD"},
			wantRestore: []string{"BBB", "DDD"},
			wantDelete:  []string{"AAA", "CCC"},
		},
		{
			name:        "pre-image routines not in the build are ignored",
			build:       []string{"VSLRTAP", "XWBPRS"},
			preimage:    []string{"XWBPRS", "SOMETHINGELSE"},
			wantRestore: []string{"XWBPRS"},
			wantDelete:  []string{"VSLRTAP"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restore, del := partitionRoutines(tc.build, tc.preimage)
			if !reflect.DeepEqual(restore, tc.wantRestore) {
				t.Errorf("restore = %v, want %v", restore, tc.wantRestore)
			}
			if !reflect.DeepEqual(del, tc.wantDelete) {
				t.Errorf("delete = %v, want %v", del, tc.wantDelete)
			}
		})
	}
}
