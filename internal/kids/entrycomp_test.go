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

func TestMakeBuildPairs_MailGroup_KRN(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZMG*1.0*1",
		Namespace:   "ZZMG",
		Routines:    []RoutineSrc{{Name: "ZZMGRT", Lines: []string{"ZZMGRT ;x", " quit"}}},
		MailGroups:  []MailGroup{{Name: "ZZMG ALERTS", TypeCode: "PR", AllowSelfEnroll: "y"}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		`"KRN",3.8,1,-1)`:                             "0^1",
		`"KRN",3.8,1,0)`:                              "ZZMG ALERTS^PR^y", // .01 NAME^4 TYPE^7 ALLOW SELF ENROLLMENT
		`"ORD",1,3.8)`:                                "3.8;1;;;MAILG^XPDTA1;MAILGF1^XPDIA1;MAILGE1^XPDIA1;MAILGF2^XPDIA1;;MAILGDEL^XPDIA1(%)",
		`"ORD",1,3.8,0)`:                              "MAIL GROUP",
		`"BLD",1,"KRN",0)`:                            "^9.67PA^3.8^1",
		`"BLD",1,"KRN",3.8,0)`:                        "3.8",
		`"BLD",1,"KRN",3.8,"NM",1,0)`:                 "ZZMG ALERTS^^0",
		`"BLD",1,"KRN",3.8,"NM","B","ZZMG ALERTS",1)`: "",
		`"BLD",1,"KRN","B",3.8,3.8)`:                  "",
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
	if ms := b.MailGroupNames(); len(ms) != 1 || ms[0] != "ZZMG ALERTS" {
		t.Errorf("MailGroupNames = %v, want [ZZMG ALERTS]", ms)
	}
}

// A mail group with no explicit type defaults to public ("PU"); TYPE (#3.8 field 4)
// is DD-required so the 0-node must always carry a value there.
func TestMakeBuildPairs_MailGroup_DefaultType(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZMG*1.0*1",
		Namespace:   "ZZMG",
		MailGroups:  []MailGroup{{Name: "ZZMG A"}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	if v := got[`"KRN",3.8,1,0)`]; v != "ZZMG A^PU" {
		t.Errorf(`"KRN",3.8,1,0) = %q, want "ZZMG A^PU"`, v)
	}
}

func TestMakeBuildPairs_ListTemplate_KRN(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZLM*1.0*1",
		Namespace:   "ZZLM",
		Routines:    []RoutineSrc{{Name: "ZZLMRT", Lines: []string{"ZZLMRT ;x", " quit"}}},
		ListTemplates: []ListTemplate{{
			Name: "ZZLM PATIENTS", ScreenTitle: "ZZ Patient List",
			RightMargin: "80", TopMargin: "3", BottomMargin: "20",
			HeaderCode: "D HDR^ZZLMRT", EntryCode: "D INIT^ZZLMRT", ExitCode: "D EXIT^ZZLMRT",
			ArrayName: `^TMP("ZZLM",$J)`,
		}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		`"KRN",409.61,1,-1)`: "0^1",
		// 0-node pieces: .01 NAME^.02 TYPE(1=PROTOCOL)^.03 LMARGIN^.04 RMARGIN^.05 TOP^
		// .06 BOT^.07 OK-TRANSPORT^.08 CURSOR^.09 ENTITY^.1 PROTO MENU^.11 TITLE^
		// .12 #ACTIONS^.13 DATE-RANGE^.14 AUTO-DEFAULTS.
		`"KRN",409.61,1,0)`:                                "ZZLM PATIENTS^1^^80^3^20^1^1^^^ZZ Patient List^1^^1",
		`"KRN",409.61,1,"HDR")`:                            "D HDR^ZZLMRT",
		`"KRN",409.61,1,"INIT")`:                           "D INIT^ZZLMRT",
		`"KRN",409.61,1,"FNL")`:                            "D EXIT^ZZLMRT",
		`"KRN",409.61,1,"ARRAY")`:                          `^TMP("ZZLM",$J)`,
		`"ORD",1,409.61)`:                                  "409.61;1;1;;;;LME1^XPDIA1;;;LMDEL^XPDIA1",
		`"ORD",1,409.61,0)`:                                "LIST TEMPLATE",
		`"BLD",1,"KRN",0)`:                                 "^9.67PA^409.61^1",
		`"BLD",1,"KRN",409.61,0)`:                          "409.61",
		`"BLD",1,"KRN",409.61,"NM",1,0)`:                   "ZZLM PATIENTS^^0",
		`"BLD",1,"KRN",409.61,"NM","B","ZZLM PATIENTS",1)`: "",
		`"BLD",1,"KRN","B",409.61,409.61)`:                 "",
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
	if ls := b.ListTemplateNames(); len(ls) != 1 || ls[0] != "ZZLM PATIENTS" {
		t.Errorf("ListTemplateNames = %v, want [ZZLM PATIENTS]", ls)
	}
}

func TestMakeBuildPairs_HelpFrame_KRN(t *testing.T) {
	in := BuildInput{
		InstallName: "ZZHF*1.0*1",
		Namespace:   "ZZHF",
		Routines:    []RoutineSrc{{Name: "ZZHFRT", Lines: []string{"ZZHFRT ;x", " quit"}}},
		HelpFrames: []HelpFrame{{
			Name: "ZZHF-MAIN", Header: "ZZ Main Help",
			Text: []string{"First help line.", "Second help line."},
		}},
	}
	got := map[string]string{}
	for _, p := range MakeBuildPairs(in) {
		got[formatSubscript(p.Subs)] = p.Value
	}
	want := map[string]string{
		`"KRN",9.2,1,-1)`: "0^1",
		// 0-node: .01 NAME ^ HEADER (the volatile DATE ENTERED / AUTHOR pieces omitted).
		`"KRN",9.2,1,0)`: "ZZHF-MAIN^ZZ Main Help",
		// field 2 TEXT — a word-processing field at node 1 (subfile 9.21). Header is
		// date-less (^^<lastSeq>^<count>) for the deterministic-build invariant.
		`"KRN",9.2,1,1,0)`:                          "^^2^2",
		`"KRN",9.2,1,1,1,0)`:                        "First help line.",
		`"KRN",9.2,1,1,2,0)`:                        "Second help line.",
		`"ORD",1,9.2)`:                              "9.2;1;;;HELP^XPDTA1;HLPF1^XPDIA1;HLPE1^XPDIA1;HLPF2^XPDIA1;;HLPDEL^XPDIA1",
		`"ORD",1,9.2,0)`:                            "HELP FRAME",
		`"BLD",1,"KRN",0)`:                          "^9.67PA^9.2^1",
		`"BLD",1,"KRN",9.2,0)`:                      "9.2",
		`"BLD",1,"KRN",9.2,"NM",1,0)`:               "ZZHF-MAIN^^0",
		`"BLD",1,"KRN",9.2,"NM","B","ZZHF-MAIN",1)`: "",
		`"BLD",1,"KRN","B",9.2,9.2)`:                "",
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
	if hs := b.HelpFrameNames(); len(hs) != 1 || hs[0] != "ZZHF-MAIN" {
		t.Errorf("HelpFrameNames = %v, want [ZZHF-MAIN]", hs)
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
