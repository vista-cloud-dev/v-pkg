package kids

import "testing"

// optionInput is a build with a routine and one #19 OPTION KRN component: a
// run-routine option (TYPE "R") whose ENTRY action is a routine the build ships.
func optionInput() BuildInput {
	return BuildInput{
		InstallName: "ZZOPT*1.0*1",
		Namespace:   "ZZOPT",
		Routines:    []RoutineSrc{{Name: "ZZOPTRT", Lines: []string{"ZZOPTRT ;x", " quit"}}},
		Options: []Option{{
			Name:     "ZZOPT RUN ROUTINE",
			MenuText: "ZZ Run Routine Demo",
			TypeCode: "R",
			Routine:  "EN^ZZOPTRT",
		}},
	}
}

// TestMakeBuildPairs_Option_KRN locks the transport shape for an OPTION (#19)
// shipped as a KRN component — the generic entry-component emitter (B.1).
// Ground-truthed against the WorldVistA corpus: the record image (-1 XPDFL flag,
// 0-node NAME^MENU TEXT^^TYPE, ROUTINE node 25, "U" uppercase-menu xref), the
// national-constant #19 ORD action-routine line, and the #9.6 BLD manifest.
func TestMakeBuildPairs_Option_KRN(t *testing.T) {
	got := map[string]string{}
	seen := map[string]bool{}
	for _, p := range MakeBuildPairs(optionInput()) {
		k := formatSubscript(p.Subs)
		if seen[k] {
			t.Errorf("duplicate subscript emitted: %s", k)
		}
		seen[k] = true
		got[k] = p.Value
	}

	want := map[string]string{
		// Top-level KRN record image — KRN^XPDIK merges it into ^DIC(19,DA).
		`"KRN",19,1,-1)`:  "0^1",                                      // XPDFL: 0=send/add-or-update
		`"KRN",19,1,0)`:   "ZZOPT RUN ROUTINE^ZZ Run Routine Demo^^R", // .01^MENU TEXT^^TYPE
		`"KRN",19,1,25)`:  "EN^ZZOPTRT",                               // ROUTINE (field 25)
		`"KRN",19,1,"U")`: "ZZ RUN ROUTINE DEMO",                      // UPPERCASE MENU TEXT xref
		// Install-order line — the #19 SEND/DELETE action routines XPDIK invokes.
		`"ORD",1,19)`:   "19;1;;;OPT^XPDTA;OPTF1^XPDIA;OPTE1^XPDIA;OPTF2^XPDIA;;OPTDEL^XPDIA",
		`"ORD",1,19,0)`: "OPTION",
		// BLD #9.6 manifest: the KRN component list (#9.67) + NM names.
		`"BLD",1,"KRN",0)`:                                 "^9.67PA^19^1",
		`"BLD",1,"KRN",19,0)`:                              "19",
		`"BLD",1,"KRN",19,"NM",0)`:                         "^9.68A^1^1",
		`"BLD",1,"KRN",19,"NM",1,0)`:                       "ZZOPT RUN ROUTINE^^0",
		`"BLD",1,"KRN",19,"NM","B","ZZOPT RUN ROUTINE",1)`: "",
		`"BLD",1,"KRN","B",19,19)`:                         "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

// TestMakeBuildPairs_SecurityKey_KRN locks the transport shape for a SECURITY KEY
// (#19.1) shipped as a KRN component — the third type on the generic emitter.
// Ground-truthed against the WorldVistA corpus: a key record is minimal (-1 XPDFL
// flag + 0-node = just the .01 NAME), plus the national-constant #19.1 ORD
// action-routine line and the #9.6 BLD manifest.
func TestMakeBuildPairs_SecurityKey_KRN(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZKEY*1.0*1",
		Namespace:   "ZZKEY",
		Routines:    []RoutineSrc{{Name: "ZZKEYRT", Lines: []string{"ZZKEYRT ;x", " quit"}}},
		Keys:        []SecurityKey{{Name: "ZZKEY MANAGER"}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		`"KRN",19.1,1,-1)`:                               "0^1",           // XPDFL: send/add-or-update
		`"KRN",19.1,1,0)`:                                "ZZKEY MANAGER", // .01 NAME — the whole record
		`"ORD",1,19.1)`:                                  "19.1;1;;;KEY^XPDTA1;KEYF1^XPDIA1;KEYE1^XPDIA1;KEYF2^XPDIA1;;KEYDEL^XPDIA1",
		`"ORD",1,19.1,0)`:                                "SECURITY KEY",
		`"BLD",1,"KRN",0)`:                               "^9.67PA^19.1^1",
		`"BLD",1,"KRN",19.1,0)`:                          "19.1",
		`"BLD",1,"KRN",19.1,"NM",0)`:                     "^9.68A^1^1",
		`"BLD",1,"KRN",19.1,"NM",1,0)`:                   "ZZKEY MANAGER^^0",
		`"BLD",1,"KRN",19.1,"NM","B","ZZKEY MANAGER",1)`: "",
		`"BLD",1,"KRN","B",19.1,19.1)`:                   "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	b := newBuild()
	for _, p := range MakeBuildPairs(in) {
		b.Set(p.Subs, p.Value)
	}
	if ks := b.KeyNames(); len(ks) != 1 || ks[0] != "ZZKEY MANAGER" {
		t.Errorf("KeyNames = %v, want [ZZKEY MANAGER]", ks)
	}
}

// TestMakeBuildPairs_Protocol_KRN locks the transport shape for a PROTOCOL (#101)
// shipped as a KRN component — the fourth type on the generic emitter. Same node
// skeleton as OPTION (0-node NAME^ITEM TEXT^^TYPE, ENTRY ACTION node 20) but its
// own data global (^ORD(101,), TYPE codes, ORD action-routine line, and NO "U"
// xref node. Ground-truthed against the WorldVistA corpus + the live #101 DD.
func TestMakeBuildPairs_Protocol_KRN(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZPROTO*1.0*1",
		Namespace:   "ZZPROTO",
		Routines:    []RoutineSrc{{Name: "ZZPRORT", Lines: []string{"ZZPRORT ;x", " quit"}}},
		Protocols:   []Protocol{{Name: "ZZPROTO ACTION", ItemText: "ZZ Protocol Action Demo", TypeCode: "A", EntryAction: "Q"}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		`"KRN",101,1,-1)`:             "0^1",
		`"KRN",101,1,0)`:              "ZZPROTO ACTION^ZZ Protocol Action Demo^^A", // .01^ITEM TEXT^^TYPE
		`"KRN",101,1,20)`:             "Q",                                         // ENTRY ACTION (field 20)
		`"ORD",1,101)`:                "101;1;;;PRO^XPDTA;PROF1^XPDIA;PROE1^XPDIA;PROF2^XPDIA;;PRODEL^XPDIA",
		`"ORD",1,101,0)`:              "PROTOCOL",
		`"BLD",1,"KRN",0)`:            "^9.67PA^101^1",
		`"BLD",1,"KRN",101,0)`:        "101",
		`"BLD",1,"KRN",101,"NM",1,0)`: "ZZPROTO ACTION^^0",
		`"BLD",1,"KRN",101,"NM","B","ZZPROTO ACTION",1)`: "",
		`"BLD",1,"KRN","B",101,101)`:                     "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	b := newBuild()
	for _, p := range MakeBuildPairs(in) {
		b.Set(p.Subs, p.Value)
	}
	if ps := b.ProtocolNames(); len(ps) != 1 || ps[0] != "ZZPROTO ACTION" {
		t.Errorf("ProtocolNames = %v, want [ZZPROTO ACTION]", ps)
	}
}

// TestMakeBuildPairs_RPC_KRN locks the transport shape for a REMOTE PROCEDURE
// (#8994) shipped as a KRN component — the fifth type on the generic emitter. The
// record is a single 0-node NAME^TAG^ROUTINE^RETURN VALUE TYPE (the three required
// fields + TAG), stored in ^XWB(8994,. Its ORD line carries the xref flag (1) and
// only a delete action routine (RPCDEL^XPDIA1) — RPCs need no menu relinking.
func TestMakeBuildPairs_RPC_KRN(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZRPC*1.0*1",
		Namespace:   "ZZRPC",
		Routines:    []RoutineSrc{{Name: "ZZRPCRT", Lines: []string{"ZZRPCRT ;x", " quit"}}},
		RPCs:        []RPC{{Name: "ZZRPC ECHO", Tag: "ECHO", Routine: "ZZRPCRT", ReturnTypeCode: "1"}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		`"KRN",8994,1,-1)`:                            "0^1",
		`"KRN",8994,1,0)`:                             "ZZRPC ECHO^ECHO^ZZRPCRT^1", // .01^.02 TAG^.03 ROUTINE^.04 RETURN TYPE
		`"ORD",1,8994)`:                               "8994;1;1;;;;;;;RPCDEL^XPDIA1",
		`"ORD",1,8994,0)`:                             "REMOTE PROCEDURE",
		`"BLD",1,"KRN",0)`:                            "^9.67PA^8994^1",
		`"BLD",1,"KRN",8994,0)`:                       "8994",
		`"BLD",1,"KRN",8994,"NM",1,0)`:                "ZZRPC ECHO^^0",
		`"BLD",1,"KRN",8994,"NM","B","ZZRPC ECHO",1)`: "",
		`"BLD",1,"KRN","B",8994,8994)`:                "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	b := newBuild()
	for _, p := range MakeBuildPairs(in) {
		b.Set(p.Subs, p.Value)
	}
	if rs := b.RPCNames(); len(rs) != 1 || rs[0] != "ZZRPC ECHO" {
		t.Errorf("RPCNames = %v, want [ZZRPC ECHO]", rs)
	}
}

// TestMakeBuildPairs_MixedEntryTypes proves the unified KRN manifest header spans
// multiple entry types in one build (B.1 next step): an OPTION (#19) AND a
// PARAMETER DEFINITION (#8989.51) share one "BLD",1,"KRN",0) header
// (^9.67PA^<max file#>^<type count>), each gets its own type body + a distinct
// ORD install order (file-number ascending), and both record images are emitted.
func TestMakeBuildPairs_MixedEntryTypes(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZMIX*1.0*1",
		Namespace:   "ZZMIX",
		Routines:    []RoutineSrc{{Name: "ZZMIXRT", Lines: []string{"ZZMIXRT ;x", " quit"}}},
		Options:     []Option{{Name: "ZZMIX RUN", MenuText: "ZZ Mix Run", TypeCode: "R", Routine: "EN^ZZMIXRT"}},
		ParamDefs: []ParamDef{{
			Name: "ZZMIX GREETING", DisplayText: "g", DataTypeCode: "F",
			Entities: []ParamEntity{{EntityIEN: "4.2", Precedence: 1}},
		}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		// One shared header: last-IEN = max file# (8989.51), count = 2 types.
		`"BLD",1,"KRN",0)`:                   "^9.67PA^8989.51^2",
		`"BLD",1,"KRN",19,0)`:                "19",
		`"BLD",1,"KRN",8989.51,0)`:           "8989.51",
		`"BLD",1,"KRN","B",19,19)`:           "",
		`"BLD",1,"KRN","B",8989.51,8989.51)`: "",
		// Distinct install orders, file-number ascending: option(19) then param(8989.51).
		`"ORD",1,19)`:        "19;1;;;OPT^XPDTA;OPTF1^XPDIA;OPTE1^XPDIA;OPTF2^XPDIA;;OPTDEL^XPDIA",
		`"ORD",2,8989.51)`:   "8989.51;2;1;;;;;;;",
		`"ORD",1,19,0)`:      "OPTION",
		`"ORD",2,8989.51,0)`: "PARAMETER DEFINITION",
		// Both record images present, each seq-numbered within its own type.
		`"KRN",19,1,0)`:      "ZZMIX RUN^ZZ Mix Run^^R",
		`"KRN",8989.51,1,0)`: "ZZMIX GREETING^g^",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	// Both name-readers work on the mixed build.
	b := newBuild()
	for _, p := range MakeBuildPairs(in) {
		b.Set(p.Subs, p.Value)
	}
	if o := b.OptionNames(); len(o) != 1 || o[0] != "ZZMIX RUN" {
		t.Errorf("OptionNames = %v", o)
	}
	if pd := b.ParamDefNames(); len(pd) != 1 || pd[0] != "ZZMIX GREETING" {
		t.Errorf("ParamDefNames = %v", pd)
	}
}

func TestBuild_OptionNames(t *testing.T) {
	b := newBuild()
	for _, p := range MakeBuildPairs(optionInput()) {
		b.Set(p.Subs, p.Value)
	}
	got := b.OptionNames()
	if len(got) != 1 || got[0] != "ZZOPT RUN ROUTINE" {
		t.Errorf("OptionNames = %v, want [ZZOPT RUN ROUTINE]", got)
	}
}

// A fresh build emits the file number 19 as a float subscript, but loading a .KID
// back (ParseKID → coerceNum) re-coerces a decimal-free number to an int — so the
// live verify/uninstall path sees an int-19 subscript. OptionNames must match it
// either way (regression: an IsFloat-only probe silently dropped every option on
// the live path while the in-memory test passed).
func TestBuild_OptionNames_AfterReparse(t *testing.T) {
	b := newBuild()
	for _, p := range MakeBuildPairs(optionInput()) {
		b.Set(parseSubscriptLine(formatSubscript(p.Subs)), p.Value)
	}
	got := b.OptionNames()
	if len(got) != 1 || got[0] != "ZZOPT RUN ROUTINE" {
		t.Errorf("OptionNames after .KID reparse = %v, want [ZZOPT RUN ROUTINE]", got)
	}
}
