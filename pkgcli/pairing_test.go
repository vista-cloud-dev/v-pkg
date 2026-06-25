package pkgcli

import "testing"

func TestDefaultPreimagePath(t *testing.T) {
	cases := []struct{ kid, want string }{
		{"OR_3_0_484.kid", "OR_3_0_484.preimage.kids"},
		{"x/y/PATCH.KID", "x/y/PATCH.preimage.kids"},
		{"noext", "noext.preimage.kids"},
		{"a.kids", "a.preimage.kids"},
	}
	for _, c := range cases {
		if got := defaultPreimagePath(c.kid); got != c.want {
			t.Errorf("defaultPreimagePath(%q) = %q, want %q", c.kid, got, c.want)
		}
	}
}

func TestResolveAutoRestore(t *testing.T) {
	cases := []struct {
		name          string
		restore       string
		backout       string
		sidecarExists bool
		wantRestore   string
		wantAuto      bool
	}{
		{name: "explicit --restore wins, no auto", restore: "p.kids", sidecarExists: true, wantRestore: "p.kids", wantAuto: false},
		{name: "--backout present, no auto-restore", backout: "b.kids", sidecarExists: true, wantRestore: "", wantAuto: false},
		{name: "no flags + sidecar exists -> auto-detect", sidecarExists: true, wantRestore: "PATCH.preimage.kids", wantAuto: true},
		{name: "no flags + no sidecar -> nothing", sidecarExists: false, wantRestore: "", wantAuto: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotRestore, gotAuto := resolveAutoRestore(tc.restore, tc.backout, "PATCH.kid", tc.sidecarExists)
			if gotRestore != tc.wantRestore || gotAuto != tc.wantAuto {
				t.Errorf("resolveAutoRestore = (%q,%v), want (%q,%v)", gotRestore, gotAuto, tc.wantRestore, tc.wantAuto)
			}
		})
	}
}
