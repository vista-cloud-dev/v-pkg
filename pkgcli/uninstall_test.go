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
		foreign    bool // build declared a foreign overwrite (read offline from the .KID)
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
		// F1: a build that declared a foreign overwrite (read offline from the .KID)
		// and has NO pre-image must be REFUSED — deleting would brick the foreign
		// national routine, and there is no pre-image to restore it. This is the
		// brick path BB1's pre-image-keyed partition could not close on its own.
		{name: "F1: declared foreign + NO pre-image -> REFUSE (the brick-path fix)", class: pure, foreign: true, wantAction: actRefuse},
		{name: "F1: declared foreign + NO pre-image + --force -> delete greenfield subset only", class: pure, foreign: true, force: true, wantAction: actDelete},
		{name: "F1: declared foreign but --restore present -> partition (pre-image reverses it)", class: pure, restore: "pre.kids", greenfield: true, foreign: true, wantAction: actPartition},
		{name: "F1: side-effecting + declared foreign + no flags -> REFUSE", class: side, foreign: true, wantAction: actRefuse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := decideUninstall(tc.class, tc.restore, tc.backout, tc.force, tc.greenfield, tc.foreign)
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

// intersectRoutines drives the wrong-sidecar guard (F1): a declared foreign
// routine that lands in the delete set is one the pre-image failed to capture.
func TestIntersectRoutines(t *testing.T) {
	cases := []struct {
		name       string
		want, have []string
		out        []string
	}{
		{name: "foreign captured by pre-image -> empty (safe)", want: []string{"XWBPRS"}, have: []string{"VSLRTAP"}, out: nil},
		{name: "foreign in delete set -> flagged (wrong sidecar)", want: []string{"XWBPRS"}, have: []string{"VSLRTAP", "XWBPRS"}, out: []string{"XWBPRS"}},
		{name: "order follows want", want: []string{"AAA", "BBB", "CCC"}, have: []string{"CCC", "AAA"}, out: []string{"AAA", "CCC"}},
		{name: "no declaration -> empty", want: nil, have: []string{"VSLRTAP"}, out: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := intersectRoutines(tc.want, tc.have); !reflect.DeepEqual(got, tc.out) {
				t.Errorf("intersectRoutines = %v, want %v", got, tc.out)
			}
		})
	}
}
