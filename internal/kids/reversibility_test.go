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

// D2 (F1): ClassifyBuild records the build's declared foreign overwrites (the
// embedded ("VPKG","FOREIGN",<name>) nodes) so class-aware uninstall has the
// offline signal to refuse deleting a foreign routine with no pre-image. The
// declaration is read verbatim — NOT inferred from routine names.
func TestClassifyBuild_ForeignOverwrites(t *testing.T) {
	// The tap shape: VSLRT* greenfield + the declared foreign XWBPRS.
	mixed := buildFrom(
		[2]string{`"BLD",1,0)`, `VSLRT*1.0*1`},
		[2]string{`"RTN","VSLRTAP")`, `0^1`},
		[2]string{`"RTN","XWBPRS")`, `0^1`},
		[2]string{`"VPKG","FOREIGN","XWBPRS")`, `1`},
	)
	br := ClassifyBuild("VSLRT*1.0*1", mixed)
	if len(br.ForeignOverwrites) != 1 || br.ForeignOverwrites[0] != "XWBPRS" {
		t.Errorf("ForeignOverwrites = %v, want [XWBPRS]", br.ForeignOverwrites)
	}
	// Class is still pure-overwrite (routine-only, no install code) — foreignness is
	// orthogonal to the reversibility class.
	if br.Class != ClassPureOverwrite {
		t.Errorf("class = %v, want pure-overwrite", br.Class)
	}

	// A declaration-free build reports none (existing builds unaffected).
	plain := buildFrom(
		[2]string{`"BLD",1,0)`, `ZZSKEL*1.0*1`},
		[2]string{`"RTN","ZZSKEL")`, `0^1`},
	)
	if got := ClassifyBuild("ZZSKEL*1.0*1", plain).ForeignOverwrites; len(got) != 0 {
		t.Errorf("declaration-free build ForeignOverwrites = %v, want none", got)
	}
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
			// "INI" is the #9.6 PRE-INSTALL routine (field 916). The corpus build
			// happens to NAME its pre-install routine YS60PRE — a naming convention,
			// not the role (that name-vs-role confusion is what once mis-mapped INI).
			name: "pre-install routine (INI) -> side-effecting",
			build: buildFrom(
				[2]string{`"RTN","YS60")`, `0^1`},
				[2]string{`"BLD",19324,"INI")`, `YS60PRE`}, // pre-install
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
			// Routines are registered as KRN components under file #9.8 in EVERY
			// real build; that registration must NOT count as a side-effecting
			// FileMan entry (else every routine-bearing build is side-effecting).
			name: "routine registration as KRN 9.8 stays pure-overwrite",
			build: buildFrom(
				[2]string{`"RTN","XUSKAAJ1")`, `0^1`},
				[2]string{`"BLD",1103,"KRN",9.8,"NM",0)`, ``},
				[2]string{`"BLD",1103,"KRN",9.8,"NM",1,0)`, `XUSKAAJ1`},
			),
			wantClass: ClassPureOverwrite, wantRtns: true, wantKRN: false,
		},
		{
			// The reliable per-build signal: a build DECLARES a FileMan component
			// in its KRN multiple ("BLD",<n>,"KRN",<file>,"NM",<seq>,0)) even when
			// the top-level "KRN",<file>,<ien>,0) export region is absent — the
			// common real-.KID shape (corpus: OPTION 39% via NM vs 13% via the
			// top-level form). Keying only on the top-level form under-detects
			// FileMan entries and over-reports the reversible pure-overwrite class.
			name: "declared KRN component (BLD..KRN..NM, no top-level) -> side-effecting",
			build: buildFrom(
				[2]string{`"RTN","ZZ")`, `0^1`},
				[2]string{`"BLD",1,"KRN",0)`, ``},                  // KRN section header (not an entry)
				[2]string{`"BLD",1,"KRN",19,0)`, `OPTION`},         // component-type header (not an entry)
				[2]string{`"BLD",1,"KRN",19,"NM",0)`, ``},          // NM multiple header (not an entry)
				[2]string{`"BLD",1,"KRN",19,"NM",1,0)`, `ZZ MENU`}, // a declared OPTION entry
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

// TestInstallCodeRoleLabels locks the corrected #9.6 BUILD subnode → role labels
// (env-check "PRE" and pre-install "INI" were once swapped; the live #9.6 DD is
// authoritative — field 913 ENVIRONMENT CHECK ROUTINE=PRE, 916 PRE-INSTALL=INI,
// 914 POST-INSTALL=INIT, 900 PRE-TRANSPORTATION=PRET).
func TestInstallCodeRoleLabels(t *testing.T) {
	got := ClassifyBuild("ZZ*1.0*1", buildFrom(
		[2]string{`"RTN","ZZ")`, `0^1`},
		[2]string{`"BLD",1,"PRE")`, `ZZENV`},
		[2]string{`"BLD",1,"INI")`, `PRE^ZZP`},
		[2]string{`"BLD",1,"INIT")`, `POST^ZZP`},
		[2]string{`"BLD",1,"PRET")`, `ZZPRET`},
	))
	want := map[string]string{
		"environment-check": "ZZENV",
		"pre-install":       "PRE^ZZP",
		"post-install":      "POST^ZZP",
		"pre-transport":     "ZZPRET",
	}
	for role, entry := range want {
		if got.InstallCode[role] != entry {
			t.Errorf("InstallCode[%q] = %q, want %q", role, got.InstallCode[role], entry)
		}
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
