package pkgcli

import (
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
		wantAction uninstallAction
	}{
		{name: "class-1, no flags -> greenfield delete", class: pure, wantAction: actDelete},
		{name: "class-1 with --restore -> restore pre-image", class: pure, restore: "pre.kids", wantAction: actRestore},
		{name: "side-effecting, no flags -> REFUSE (the safety fix)", class: side, wantAction: actRefuse},
		{name: "side-effecting with --backout -> run authored back-out", class: side, backout: "bo.kids", wantAction: actBackout},
		{name: "side-effecting with --restore -> restore (routines only)", class: side, restore: "pre.kids", wantAction: actRestore},
		{name: "side-effecting with --force -> delete anyway", class: side, force: true, wantAction: actDelete},
		{name: "both --restore and --backout -> refuse (ambiguous)", class: pure, restore: "p.kids", backout: "b.kids", wantAction: actRefuse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := decideUninstall(tc.class, tc.restore, tc.backout, tc.force)
			if got != tc.wantAction {
				t.Errorf("decideUninstall = %v (%q), want %v", got, reason, tc.wantAction)
			}
			if reason == "" {
				t.Errorf("expected a non-empty reason for action %v", got)
			}
		})
	}
}
