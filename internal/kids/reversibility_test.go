package kids

import "testing"

// buildFrom constructs a Build from (subscriptLine, value) pairs, using the same
// parser the real loader uses — so the classifier is exercised on real subscript
// shapes, not hand-built Subs.
func buildFrom(pairs ...[2]string) *Build {
	b := newBuild()
	for _, p := range pairs {
		b.Set(parseSubscriptLine(p[0]), p[1])
	}
	return b
}

func TestClassifyBuild(t *testing.T) {
	cases := []struct {
		name      string
		build     *Build
		wantClass ReversibilityClass
		wantRtns  bool
		wantCode  bool
		wantKRN   bool
		wantDD    bool
	}{
		{
			name: "routines only -> pure overwrite (the XWBBRK class)",
			build: buildFrom(
				[2]string{`"BLD",1,0)`, `XWBBRK*1.0*1`},
				[2]string{`"RTN")`, ``},
				[2]string{`"RTN","XWBBRK")`, `0^1^B12345`},
				[2]string{`"RTN","XWBBRK",1,0)`, `XWBBRK ; broker`},
				[2]string{`"RTN","XWBBRK",2,0)`, ` ;;1.0`},
			),
			wantClass: ClassPureOverwrite, wantRtns: true,
		},
		{
			name: "post-install routine -> side-effecting",
			build: buildFrom(
				[2]string{`"RTN","SD53MY17")`, `0^1`},
				[2]string{`"BLD",10336,"INIT")`, `EN^SD53MY17`}, // post-install
			),
			wantClass: ClassSideEffecting, wantRtns: true, wantCode: true,
		},
		{
			name: "environment-check routine (INI) -> side-effecting",
			build: buildFrom(
				[2]string{`"RTN","YS60")`, `0^1`},
				[2]string{`"BLD",19324,"INI")`, `YS60PRE`}, // env-check
			),
			wantClass: ClassSideEffecting, wantRtns: true, wantCode: true,
		},
		{
			name: "empty PRE value is NOT install code",
			build: buildFrom(
				[2]string{`"RTN","ZZ")`, `0^1`},
				[2]string{`"BLD",19537,"PRE")`, ``},      // declared but empty
				[2]string{`"BLD",19537,"INID")`, `^y^n`}, // a flag, not code
			),
			wantClass: ClassPureOverwrite, wantRtns: true, wantCode: false,
		},
		{
			name: "exported OPTION entry (KRN) -> side-effecting",
			build: buildFrom(
				[2]string{`"RTN","ZZ")`, `0^1`},
				[2]string{`"KRN",19,0)`, `OPTION`},              // component header (not an entry)
				[2]string{`"KRN",19,8821,0)`, `ZZ MENU^^^^^^^`}, // an exported option record
			),
			wantClass: ClassSideEffecting, wantRtns: true, wantKRN: true,
		},
		{
			name: "FileMan FILE (DD/data) shipment -> side-effecting",
			build: buildFrom(
				[2]string{`"RTN","ZZ")`, `0^1`},
				[2]string{`"BLD",9217,4,0)`, ``},          // FILE multiple header (not an entry)
				[2]string{`"BLD",9217,4,408.21,0)`, `^^`}, // a shipped file
			),
			wantClass: ClassSideEffecting, wantRtns: true, wantDD: true,
		},
		{
			name:      "metadata only (no routines, no side-effects) -> pure overwrite (nothing to undo)",
			build:     buildFrom([2]string{`"BLD",1,0)`, `ZZ*1.0*1`}, [2]string{`"REQB",1,0)`, `XU*8.0*1`}),
			wantClass: ClassPureOverwrite, wantRtns: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyBuild(tc.name, tc.build)
			if got.Class != tc.wantClass {
				t.Errorf("Class = %v, want %v", got.Class, tc.wantClass)
			}
			if got.ShipsRoutines != tc.wantRtns {
				t.Errorf("ShipsRoutines = %v, want %v", got.ShipsRoutines, tc.wantRtns)
			}
			if got.HasInstallCode != tc.wantCode {
				t.Errorf("HasInstallCode = %v, want %v", got.HasInstallCode, tc.wantCode)
			}
			if got.ShipsFileManEntries != tc.wantKRN {
				t.Errorf("ShipsFileManEntries = %v, want %v", got.ShipsFileManEntries, tc.wantKRN)
			}
			if got.ShipsFileDD != tc.wantDD {
				t.Errorf("ShipsFileDD = %v, want %v", got.ShipsFileDD, tc.wantDD)
			}
		})
	}
}

// TestClassifyOverall: the least-reversible build governs the distribution.
func TestClassifyOverall(t *testing.T) {
	k := &KID{
		InstallNames: []string{"A*1.0*1", "B*1.0*1"},
		Builds: map[string]*Build{
			"A*1.0*1": buildFrom([2]string{`"RTN","A")`, `0^1`}),                                       // pure
			"B*1.0*1": buildFrom([2]string{`"RTN","B")`, `0^1`}, [2]string{`"BLD",2,"INIT")`, `EN^B`}), // side-effecting
		},
	}
	rev := Classify(k)
	if rev.Class != ClassSideEffecting {
		t.Errorf("overall Class = %v, want SideEffecting (least-reversible governs)", rev.Class)
	}
	if len(rev.Builds) != 2 {
		t.Fatalf("Builds = %d, want 2", len(rev.Builds))
	}
}

// TestClassifyRoutineNames: distinct routine names are collected for the snapshot
// pre-image (the set a class-1 restore must cover).
func TestClassifyRoutineNames(t *testing.T) {
	b := buildFrom(
		[2]string{`"RTN","XWBBRK")`, `0^1`},
		[2]string{`"RTN","XWBBRK",1,0)`, `X`},
		[2]string{`"RTN","XWBBRK2")`, `0^1`},
	)
	got := ClassifyBuild("t", b)
	if got.RoutineCount != 2 {
		t.Errorf("RoutineCount = %d, want 2", got.RoutineCount)
	}
}
