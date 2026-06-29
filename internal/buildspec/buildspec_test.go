package buildspec

import (
	"fmt"
	"strings"
	"testing"
)

const zzskel = `{
  "package": "ZZSKEL",
  "version": "1.0",
  "patch": "1",
  "components": { "routines": ["ZZSKEL"] },
  "requiredBuilds": [
    { "name": "STD*1.0*5", "action": "DON'T INSTALL, LEAVE GLOBAL" }
  ],
  "envCheck": "ZZSKELEN",
  "icrs": [ { "number": 10063, "name": "$$PSET^%ZTLOAD", "custodian": "KERNEL" } ]
}`

func TestParse_Valid(t *testing.T) {
	s, err := Parse([]byte(zzskel))
	if err != nil {
		t.Fatalf("Parse valid spec: %v", err)
	}
	if got := s.InstallName(); got != "ZZSKEL*1.0*1" {
		t.Errorf("InstallName = %q, want ZZSKEL*1.0*1", got)
	}
	if len(s.Components.Routines) != 1 || s.Components.Routines[0] != "ZZSKEL" {
		t.Errorf("routines = %v", s.Components.Routines)
	}
	if len(s.RequiredBuilds) != 1 || s.RequiredBuilds[0].Action != ActionLeaveGlobal {
		t.Errorf("requiredBuilds = %+v", s.RequiredBuilds)
	}
	if s.EnvCheck != "ZZSKELEN" {
		t.Errorf("envCheck = %q", s.EnvCheck)
	}
}

const vslbase = `{
  "package": "VSLBASE",
  "version": "1.0",
  "patch": "1",
  "components": {
    "routines": ["VSLCFG"],
    "parameterDefinitions": [
      {
        "name": "VSL GREETING",
        "displayText": "VSL greeting string (read by VPNG)",
        "dataType": "free text",
        "entities": [ { "entity": "SYS", "precedence": 1 } ]
      }
    ]
  },
  "requiredBuilds": [
    { "name": "MSL*1.0*1", "action": "DON'T INSTALL, LEAVE GLOBAL" }
  ]
}`

func TestParse_ParameterDefinitions(t *testing.T) {
	s, err := Parse([]byte(vslbase))
	if err != nil {
		t.Fatalf("Parse VSL base spec: %v", err)
	}
	if len(s.Components.ParameterDefinitions) != 1 {
		t.Fatalf("parameterDefinitions = %+v", s.Components.ParameterDefinitions)
	}
	p := s.Components.ParameterDefinitions[0]
	if p.Name != "VSL GREETING" {
		t.Errorf("param name = %q", p.Name)
	}
	if p.DataType != "free text" {
		t.Errorf("param dataType = %q", p.DataType)
	}
	if len(p.Entities) != 1 || p.Entities[0].Entity != "SYS" || p.Entities[0].Precedence != 1 {
		t.Errorf("param entities = %+v", p.Entities)
	}
	// A spec carrying only parameter definitions (no routines) is still non-empty.
	if s.Components.empty() {
		t.Error("spec with a parameter definition must not be empty")
	}
}

func TestParse_ParameterDefinitions_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":      `{"package":"VSL","version":"1.0","components":{"parameterDefinitions":[{"displayText":"x"}]}}`,
		"bad name":     `{"package":"VSL","version":"1.0","components":{"parameterDefinitions":[{"name":"lower case"}]}}`,
		"bad datatype": `{"package":"VSL","version":"1.0","components":{"parameterDefinitions":[{"name":"VSL X","dataType":"bogus"}]}}`,
		"bad entity":   `{"package":"VSL","version":"1.0","components":{"parameterDefinitions":[{"name":"VSL X","entities":[{"entity":"NOPE"}]}]}}`,
		"neg prec":     `{"package":"VSL","version":"1.0","components":{"parameterDefinitions":[{"name":"VSL X","entities":[{"entity":"SYS","precedence":-1}]}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

const zzvslfs = `{
  "package": "ZZVSLFS",
  "version": "1.0",
  "patch": "1",
  "components": {
    "files": [
      { "number": 999000, "name": "ZZVSLFS", "globalRoot": "^DIZ(999000," }
    ]
  }
}`

func TestParse_Files(t *testing.T) {
	s, err := Parse([]byte(zzvslfs))
	if err != nil {
		t.Fatalf("Parse ZZVSLFS file spec: %v", err)
	}
	if len(s.Components.Files) != 1 {
		t.Fatalf("files = %+v", s.Components.Files)
	}
	f := s.Components.Files[0]
	if f.Number != 999000 || f.Name != "ZZVSLFS" || f.GlobalRoot != "^DIZ(999000," {
		t.Errorf("file = %+v", f)
	}
	// A spec carrying only a file (no routines) is still non-empty.
	if s.Components.empty() {
		t.Error("spec with a file component must not be empty")
	}
}

func TestParse_Files_DefaultGlobalRoot(t *testing.T) {
	s, err := Parse([]byte(`{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":999000,"name":"ZZVSLFS"}]}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// An omitted globalRoot defaults to ^DIZ(<file>, after validation.
	if got := s.Components.Files[0].GlobalRoot; got != "^DIZ(999000," {
		t.Errorf("default globalRoot = %q, want ^DIZ(999000,", got)
	}
}

func TestParse_Files_Invalid(t *testing.T) {
	cases := map[string]string{
		"zero number":       `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":0,"name":"ZZVSLFS","globalRoot":"^DIZ(0,"}]}}`,
		"non-integer":       `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":1.5,"name":"ZZVSLFS","globalRoot":"^DIZ(1,"}]}}`,
		"permanent no root": `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":662001,"name":"ZZVSLFS"}]}}`,
		"no name":           `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":999000}]}}`,
		"lower-case name":   `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":999000,"name":"lower"}]}}`,
		"bad global root":   `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":999000,"name":"ZZVSLFS","globalRoot":"DIZ999000"}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// A permanent (non-test-range) file number is allowed when the spec supplies an
// explicit global root — the test-range ^DIZ default never applies to a permanent
// file (it must declare where it lives; the namespace governance is an org concern).
func TestParse_Files_PermanentNumber(t *testing.T) {
	s, err := Parse([]byte(`{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":662001,"name":"ZZVSLFS","globalRoot":"^ZZVSL(662001,"}]}}`))
	if err != nil {
		t.Fatalf("permanent file number with explicit root should parse: %v", err)
	}
	if got := s.Components.Files[0].GlobalRoot; got != "^ZZVSL(662001," {
		t.Errorf("globalRoot = %q", got)
	}
}

// A data-bearing FILE component carries an action + records; the action name maps
// to the #9.64,222.8 SITE'S DATA send-options letter.
func TestParse_Files_Data(t *testing.T) {
	js := `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{
		"number":999000,"name":"ZZVSLFS","globalRoot":"^DIZ(999000,",
		"fields":[{"number":1,"label":"COUNT","type":"numeric","node":0,"piece":2,"width":6}],
		"data":{"action":"merge","records":[
			{"ien":1,"values":{".01":"ALPHA","1":"42"}},
			{"ien":2,"values":{".01":"BRAVO"}}
		]}
	}]}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("parse data-bearing file: %v", err)
	}
	d := s.Components.Files[0].Data
	if d == nil || d.Action != "merge" || len(d.Records) != 2 {
		t.Fatalf("data = %+v", d)
	}
	if DataActionCode[d.Action] != "m" {
		t.Errorf("action %q → code %q, want m", d.Action, DataActionCode[d.Action])
	}
}

func TestParse_Files_Data_Invalid(t *testing.T) {
	base := func(data string) string {
		return `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{` +
			`"number":999000,"name":"ZZVSLFS","globalRoot":"^DIZ(999000,",` +
			`"fields":[{"number":1,"label":"COUNT","type":"numeric","node":0,"piece":2,"width":6},` +
			`{"number":2,"label":"WHO","type":"pointer","node":0,"piece":3,"pointTo":200,"pointRoot":"^VA(200,"}],` +
			`"data":` + data + `}]}}`
	}
	cases := map[string]string{
		"unknown action":    base(`{"action":"clobber","records":[{"ien":1,"values":{".01":"A"}}]}`),
		"no records":        base(`{"action":"merge","records":[]}`),
		"bad ien":           base(`{"action":"merge","records":[{"ien":0,"values":{".01":"A"}}]}`),
		"unknown field":     base(`{"action":"merge","records":[{"ien":1,"values":{"7":"A"}}]}`),
		"missing dot01":     base(`{"action":"merge","records":[{"ien":1,"values":{"1":"42"}}]}`),
		"pointer-field val": base(`{"action":"merge","records":[{"ien":1,"values":{".01":"A","2":"5"}}]}`),
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

const zzvslAudit = `{
  "package": "ZZVSLAU",
  "version": "1.0",
  "patch": "1",
  "components": {
    "files": [
      {
        "number": 999001, "name": "ZZVSL AUDIT", "globalRoot": "^DIZ(999001,",
        "fields": [
          { "number": 1, "label": "USER NUMBER", "type": "numeric", "node": 0, "piece": 2, "width": 12, "decimals": 0, "min": 1 },
          { "number": 2, "label": "EVENT TYPE", "type": "set of codes", "node": 0, "piece": 3,
            "codes": [ { "internal": "I", "external": "INFO" }, { "internal": "W", "external": "WARN" } ] },
          { "number": 3, "label": "TIMESTAMP", "type": "date", "node": 0, "piece": 4, "time": true },
          { "number": 4, "label": "USER", "type": "pointer", "node": 0, "piece": 5, "required": true, "pointTo": 200, "pointRoot": "^VA(200," },
          { "number": 5, "label": "DETAIL", "type": "free text", "node": 0, "piece": 6, "maxLen": 245 }
        ]
      }
    ]
  }
}`

func TestParse_MultiFieldFile(t *testing.T) {
	s, err := Parse([]byte(zzvslAudit))
	if err != nil {
		t.Fatalf("Parse multi-field file spec: %v", err)
	}
	f := s.Components.Files[0]
	if len(f.Fields) != 5 {
		t.Fatalf("fields = %+v", f.Fields)
	}
	if f.Fields[0].Type != "numeric" || f.Fields[0].Width != 12 || f.Fields[0].Min == nil || *f.Fields[0].Min != 1 {
		t.Errorf("numeric field = %+v", f.Fields[0])
	}
	if f.Fields[1].Type != "set of codes" || len(f.Fields[1].Codes) != 2 || f.Fields[1].Codes[0].Internal != "I" {
		t.Errorf("set field = %+v", f.Fields[1])
	}
	if f.Fields[3].Type != "pointer" || !f.Fields[3].Required || f.Fields[3].PointTo != 200 {
		t.Errorf("pointer field = %+v", f.Fields[3])
	}
}

func TestParse_Fields_Invalid(t *testing.T) {
	base := `{"package":"ZZVSLAU","version":"1.0","components":{"files":[{"number":999001,"name":"ZZVSLAU","fields":[%s]}]}}`
	cases := map[string]string{
		"number not above .01": `{"number":0.01,"label":"X","type":"free text","piece":2}`,
		"duplicate number":     `{"number":1,"label":"A","type":"free text","piece":2},{"number":1,"label":"B","type":"free text","piece":3}`,
		"piece collides .01":   `{"number":1,"label":"A","type":"free text","node":0,"piece":1}`,
		"duplicate storage":    `{"number":1,"label":"A","type":"free text","piece":2},{"number":2,"label":"B","type":"free text","piece":2}`,
		"no piece":             `{"number":1,"label":"A","type":"free text"}`,
		"lower-case label":     `{"number":1,"label":"lower","type":"free text","piece":2}`,
		"unknown type":         `{"number":1,"label":"A","type":"blob","piece":2}`,
		"numeric no width":     `{"number":1,"label":"A","type":"numeric","piece":2}`,
		"min above max":        `{"number":1,"label":"A","type":"numeric","width":5,"piece":2,"min":9,"max":1}`,
		"set no codes":         `{"number":1,"label":"A","type":"set of codes","piece":2}`,
		"set bad code":         `{"number":1,"label":"A","type":"set of codes","piece":2,"codes":[{"internal":"I;","external":"INFO"}]}`,
		"pointer no file":      `{"number":1,"label":"A","type":"pointer","piece":2,"pointRoot":"^VA(200,"}`,
		"pointer bad root":     `{"number":1,"label":"A","type":"pointer","piece":2,"pointTo":200,"pointRoot":"VA200"}`,
		"free text too long":   `{"number":1,"label":"A","type":"free text","piece":2,"maxLen":300}`,
	}
	for name, fld := range cases {
		js := fmt.Sprintf(base, fld)
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_InstallHooks(t *testing.T) {
	js := `{"package":"ZZA1","version":"1.0","components":{"routines":["ZZA1P","ZZA1ENV"]},
	  "envCheck":"ZZA1ENV","preInstall":"PRE^ZZA1P","postInstall":"POST^ZZA1P"}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("parse install hooks: %v", err)
	}
	if s.EnvCheck != "ZZA1ENV" || s.PreInstall != "PRE^ZZA1P" || s.PostInstall != "POST^ZZA1P" {
		t.Errorf("hooks = %q/%q/%q", s.EnvCheck, s.PreInstall, s.PostInstall)
	}
}

func TestParse_InstallHooks_Invalid(t *testing.T) {
	base := `{"package":"ZZA1","version":"1.0","components":{"routines":["ZZA1P"]},%s}`
	cases := map[string]string{
		"envCheck has a tag":   `"envCheck":"PRE^ZZA1P"`, // env-check must be a bare routine
		"envCheck lowercase":   `"envCheck":"lower"`,
		"preInstall bad tag":   `"preInstall":"lower^ZZA1P"`,
		"preInstall empty rtn": `"preInstall":"PRE^"`,
		"postInstall 3 parts":  `"postInstall":"A^B^ZZA1P"`,
	}
	for name, frag := range cases {
		if _, err := Parse([]byte(fmt.Sprintf(base, frag))); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestInstallName_NoPatch(t *testing.T) {
	s, err := Parse([]byte(`{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"]}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := s.InstallName(); got != "ZZSKEL*1.0" {
		t.Errorf("InstallName = %q, want ZZSKEL*1.0", got)
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := map[string]string{
		"missing package":  `{"version":"1.0","components":{"routines":["ZZSKEL"]}}`,
		"bad version":      `{"package":"ZZSKEL","version":"1","components":{"routines":["ZZSKEL"]}}`,
		"no components":    `{"package":"ZZSKEL","version":"1.0","components":{}}`,
		"bad action":       `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"]},"requiredBuilds":[{"name":"STD*1.0*5","action":"MAYBE"}]}`,
		"bad routine name": `{"package":"ZZSKEL","version":"1.0","components":{"routines":["not a routine"]}}`,
		"bad env-check":    `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"]},"envCheck":"toolongname9"}`,
		"required no name": `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"]},"requiredBuilds":[{"action":"WARNING ONLY"}]}`,
		"unknown field":    `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"]},"oops":1}`,
		"bad icr number":   `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"]},"icrs":[{"number":0}]}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_NotJSON(t *testing.T) {
	if _, err := Parse([]byte("nope")); err == nil || !strings.Contains(err.Error(), "buildspec") {
		t.Errorf("want a buildspec parse error, got %v", err)
	}
}

func TestLoad_File(t *testing.T) {
	s, err := Load("testdata/zzskel.build.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.InstallName() != "ZZSKEL*1.0*1" {
		t.Errorf("InstallName = %q", s.InstallName())
	}
}

func TestLoad_Missing(t *testing.T) {
	if _, err := Load("testdata/nope.build.json"); err == nil {
		t.Error("missing file must error")
	}
}

// A >8-char routine name is rejected under the default (legacy SAC) policy.
func TestParse_LongName_RejectedByDefault(t *testing.T) {
	js := `{"package":"MSL","version":"0.1","components":{"routines":["STDASSERT"]}}`
	if _, err := Parse([]byte(js)); err == nil {
		t.Fatal("a 9-char routine name must be rejected without allowLongNames")
	}
}

// allowLongNames raises the cap from the legacy 8 to the engine limit (31),
// keeping M-naming character validation. See docs/background ADR
// "modern routine-name length policy".
func TestParse_AllowLongNames(t *testing.T) {
	js := `{"package":"MSL","version":"0.1","allowLongNames":true,
	  "components":{"routines":["STDASSERT","STDCOMPRESS","STDHTTPMSG"]},
	  "envCheck":"VSLLONGENVCHECK"}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("allowLongNames should admit ≤31-char M names: %v", err)
	}
	if !s.AllowLongNames {
		t.Error("AllowLongNames field not decoded")
	}
	if len(s.Components.Routines) != 3 {
		t.Errorf("routines = %v", s.Components.Routines)
	}
}

// Even with allowLongNames, the engine ceiling (31) and M-naming character
// rules still bind — the policy modernizes the limit, it does not remove it.
func TestParse_AllowLongNames_StillGated(t *testing.T) {
	cases := map[string]string{
		"over engine ceiling":  `{"package":"MSL","version":"0.1","allowLongNames":true,"components":{"routines":["` + strings.Repeat("A", 32) + `"]}}`,
		"lower-case still bad": `{"package":"MSL","version":"0.1","allowLongNames":true,"components":{"routines":["stdassert"]}}`,
		"spaces still bad":     `{"package":"MSL","version":"0.1","allowLongNames":true,"components":{"routines":["STD ASSERT"]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// A build path that silently drops a declared component manufactures false
// confidence: the build "succeeds" but the option/key/protocol/… never ships
// (coverage-analysis F1). Declaring a component type v-pkg cannot emit yet must
// be a hard error that NAMES the type, not a silent omission.
func TestParse_UnsupportedComponents_Rejected(t *testing.T) {
	cases := map[string]string{
		"templates": `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"templates":["ZZSKEL TMPL"]}}`,
	}
	for name, js := range cases {
		_, err := Parse([]byte(js))
		if err == nil {
			t.Errorf("%s: a declared-but-unemittable component should error, got nil", name)
			continue
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("%s: error must name the unsupported component type, got %v", name, err)
		}
	}
}

func TestParse_Options(t *testing.T) {
	js := `{"package":"ZZOPT","version":"1.0","patch":"1","components":{
	  "routines":["ZZOPTRT"],
	  "options":[{"name":"ZZOPT RUN ROUTINE","menuText":"ZZ Run Routine Demo","type":"run routine","routine":"EN^ZZOPTRT"}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse options spec: %v", err)
	}
	if len(s.Components.Options) != 1 {
		t.Fatalf("options = %+v", s.Components.Options)
	}
	o := s.Components.Options[0]
	if o.Name != "ZZOPT RUN ROUTINE" || o.Type != "run routine" || o.Routine != "EN^ZZOPTRT" {
		t.Errorf("option = %+v", o)
	}
	// A spec carrying only an option (no routines) is still non-empty.
	opt, err := Parse([]byte(`{"package":"ZZOPT","version":"1.0","components":{"options":[{"name":"ZZOPT A","menuText":"x","type":"action","entryAction":"W 1"}]}}`))
	if err != nil {
		t.Fatalf("Parse option-only spec: %v", err)
	}
	if opt.Components.empty() {
		t.Error("spec with an option must not be empty")
	}
}

func TestParse_Options_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":         `{"package":"ZZOPT","version":"1.0","components":{"options":[{"menuText":"x","type":"action"}]}}`,
		"bad name":        `{"package":"ZZOPT","version":"1.0","components":{"options":[{"name":"lower case","type":"action"}]}}`,
		"bad type":        `{"package":"ZZOPT","version":"1.0","components":{"options":[{"name":"ZZOPT A","type":"bogus"}]}}`,
		"run no routine":  `{"package":"ZZOPT","version":"1.0","components":{"options":[{"name":"ZZOPT A","type":"run routine"}]}}`,
		"bad routine ref": `{"package":"ZZOPT","version":"1.0","components":{"options":[{"name":"ZZOPT A","type":"run routine","routine":"EN^123BAD"}]}}`,
		"bad menu item":   `{"package":"ZZOPT","version":"1.0","components":{"options":[{"name":"ZZOPT M","type":"menu","menuItems":[{"name":"lower"}]}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// A menu OPTION's MENU items parse with name + synonym + display order.
func TestParse_Options_MenuItems(t *testing.T) {
	js := `{"package":"ZZOPT","version":"1.0","components":{"options":[
	  {"name":"ZZOPT MENU","type":"menu","menuItems":[{"name":"ZZOPT A","synonym":"A","displayOrder":1}]}]}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse option menu items: %v", err)
	}
	its := s.Components.Options[0].MenuItems
	if len(its) != 1 || its[0].Name != "ZZOPT A" || its[0].Synonym != "A" || its[0].DisplayOrder != 1 {
		t.Errorf("menu items = %+v", its)
	}
}

func TestParse_RPCs(t *testing.T) {
	js := `{"package":"ZZRPC","version":"1.0","patch":"1","components":{
	  "routines":["ZZRPCRT"],
	  "rpcs":[{"name":"ZZRPC ECHO","tag":"ECHO","routine":"ZZRPCRT","returnType":"single value"}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse rpcs spec: %v", err)
	}
	if len(s.Components.RPCs) != 1 {
		t.Fatalf("rpcs = %+v", s.Components.RPCs)
	}
	r := s.Components.RPCs[0]
	if r.Name != "ZZRPC ECHO" || r.Tag != "ECHO" || r.Routine != "ZZRPCRT" || r.ReturnType != "single value" {
		t.Errorf("rpc = %+v", r)
	}
	// returnType defaults to single value when omitted.
	d, err := Parse([]byte(`{"package":"ZZRPC","version":"1.0","components":{"rpcs":[{"name":"ZZRPC A","tag":"A","routine":"ZZRPCRT"}]}}`))
	if err != nil {
		t.Fatalf("Parse rpc without returnType: %v", err)
	}
	if d.Components.empty() {
		t.Error("spec with an rpc must not be empty")
	}
}

func TestParse_RPCs_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":     `{"package":"ZZR","version":"1.0","components":{"rpcs":[{"routine":"ZZRPCRT"}]}}`,
		"bad name":    `{"package":"ZZR","version":"1.0","components":{"rpcs":[{"name":"lower","routine":"ZZRPCRT"}]}}`,
		"no routine":  `{"package":"ZZR","version":"1.0","components":{"rpcs":[{"name":"ZZR A"}]}}`,
		"bad routine": `{"package":"ZZR","version":"1.0","components":{"rpcs":[{"name":"ZZR A","routine":"123BAD"}]}}`,
		"bad rtype":   `{"package":"ZZR","version":"1.0","components":{"rpcs":[{"name":"ZZR A","routine":"ZZRPCRT","returnType":"bogus"}]}}`,
		"bad param":   `{"package":"ZZR","version":"1.0","components":{"rpcs":[{"name":"ZZR A","routine":"ZZRPCRT","inputParameters":[{"name":"lower"}]}]}}`,
		"bad ptype":   `{"package":"ZZR","version":"1.0","components":{"rpcs":[{"name":"ZZR A","routine":"ZZRPCRT","inputParameters":[{"name":"INPUT","type":"bogus"}]}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// An RPC's INPUT PARAMETERs parse with their fields and survive validation.
func TestParse_RPCs_InputParameters(t *testing.T) {
	js := `{"package":"ZZRPC","version":"1.0","components":{"rpcs":[
	  {"name":"ZZRPC ECHO","routine":"ZZRPCRT","inputParameters":[
	    {"name":"INPUT","type":"literal","maxLength":80,"required":true,"sequence":1,"description":["d"]},
	    {"name":"FLAGS","type":"list","sequence":2}]}]}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse rpc input params: %v", err)
	}
	ps := s.Components.RPCs[0].InputParameters
	if len(ps) != 2 || ps[0].Name != "INPUT" || ps[0].Type != "literal" || ps[0].MaxLength != 80 || !ps[0].Required || ps[1].Name != "FLAGS" {
		t.Errorf("input parameters = %+v", ps)
	}
}

func TestParse_MailGroups(t *testing.T) {
	js := `{"package":"ZZMG","version":"1.0","patch":"1","components":{
	  "routines":["ZZMGRT"],
	  "mailGroups":[{"name":"ZZMG ALERTS","type":"private","allowSelfEnrollment":true}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse mailGroups spec: %v", err)
	}
	if len(s.Components.MailGroups) != 1 {
		t.Fatalf("mailGroups = %+v", s.Components.MailGroups)
	}
	m := s.Components.MailGroups[0]
	if m.Name != "ZZMG ALERTS" || m.Type != "private" || !m.AllowSelfEnrollment {
		t.Errorf("mailGroup = %+v", m)
	}
	// type defaults (to public) when omitted, and a mail-group-only spec is non-empty.
	d, err := Parse([]byte(`{"package":"ZZMG","version":"1.0","components":{"mailGroups":[{"name":"ZZMG A"}]}}`))
	if err != nil {
		t.Fatalf("Parse mail group without type: %v", err)
	}
	if d.Components.empty() {
		t.Error("spec with a mail group must not be empty")
	}
}

func TestParse_MailGroups_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":  `{"package":"ZZMG","version":"1.0","components":{"mailGroups":[{"type":"public"}]}}`,
		"bad name": `{"package":"ZZMG","version":"1.0","components":{"mailGroups":[{"name":"lower case"}]}}`,
		"bad type": `{"package":"ZZMG","version":"1.0","components":{"mailGroups":[{"name":"ZZMG A","type":"bogus"}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_ListTemplates(t *testing.T) {
	js := `{"package":"ZZLM","version":"1.0","patch":"1","components":{
	  "routines":["ZZLMRT"],
	  "listTemplates":[{"name":"ZZLM PATIENTS","screenTitle":"ZZ Patient List","rightMargin":80,"topMargin":3,"bottomMargin":20,"headerCode":"D HDR^ZZLMRT","entryCode":"D INIT^ZZLMRT","exitCode":"D EXIT^ZZLMRT","arrayName":"^TMP(\"ZZLM\",$J)"}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse listTemplates spec: %v", err)
	}
	if len(s.Components.ListTemplates) != 1 {
		t.Fatalf("listTemplates = %+v", s.Components.ListTemplates)
	}
	lt := s.Components.ListTemplates[0]
	if lt.Name != "ZZLM PATIENTS" || lt.ScreenTitle != "ZZ Patient List" || lt.RightMargin != 80 || lt.EntryCode != "D INIT^ZZLMRT" {
		t.Errorf("listTemplate = %+v", lt)
	}
	// A list-template-only spec (just a name) is non-empty.
	d, err := Parse([]byte(`{"package":"ZZLM","version":"1.0","components":{"listTemplates":[{"name":"ZZLM A"}]}}`))
	if err != nil {
		t.Fatalf("Parse minimal list template: %v", err)
	}
	if d.Components.empty() {
		t.Error("spec with a list template must not be empty")
	}
}

func TestParse_ListTemplates_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":     `{"package":"ZZLM","version":"1.0","components":{"listTemplates":[{"screenTitle":"x"}]}}`,
		"bad name":    `{"package":"ZZLM","version":"1.0","components":{"listTemplates":[{"name":"lower case"}]}}`,
		"bad margin":  `{"package":"ZZLM","version":"1.0","components":{"listTemplates":[{"name":"ZZLM A","rightMargin":-5}]}}`,
		"huge margin": `{"package":"ZZLM","version":"1.0","components":{"listTemplates":[{"name":"ZZLM A","rightMargin":999}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_HelpFrames(t *testing.T) {
	js := `{"package":"ZZHF","version":"1.0","patch":"1","components":{
	  "routines":["ZZHFRT"],
	  "helpFrames":[{"name":"ZZHF-MAIN","header":"ZZ Main Help","text":["First help line.","Second help line."]}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse helpFrames spec: %v", err)
	}
	if len(s.Components.HelpFrames) != 1 {
		t.Fatalf("helpFrames = %+v", s.Components.HelpFrames)
	}
	h := s.Components.HelpFrames[0]
	if h.Name != "ZZHF-MAIN" || h.Header != "ZZ Main Help" || len(h.Text) != 2 || h.Text[0] != "First help line." {
		t.Errorf("helpFrame = %+v", h)
	}
	// A help-frame-only spec (hyphenated name, no text) is non-empty.
	d, err := Parse([]byte(`{"package":"ZZHF","version":"1.0","components":{"helpFrames":[{"name":"ZZHF-A"}]}}`))
	if err != nil {
		t.Fatalf("Parse minimal help frame: %v", err)
	}
	if d.Components.empty() {
		t.Error("spec with a help frame must not be empty")
	}
	// A space in the name is allowed (help frame names permit spaces and hyphens).
	if _, err := Parse([]byte(`{"package":"ZZHF","version":"1.0","components":{"helpFrames":[{"name":"ZZHF MAIN MENU"}]}}`)); err != nil {
		t.Errorf("space in help frame name should be allowed: %v", err)
	}
}

func TestParse_HelpFrames_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":   `{"package":"ZZHF","version":"1.0","components":{"helpFrames":[{"header":"x"}]}}`,
		"too short": `{"package":"ZZHF","version":"1.0","components":{"helpFrames":[{"name":"ZZ"}]}}`,
		"lowercase": `{"package":"ZZHF","version":"1.0","components":{"helpFrames":[{"name":"zzhf-main"}]}}`,
		"all digit": `{"package":"ZZHF","version":"1.0","components":{"helpFrames":[{"name":"12345"}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_HL7Applications(t *testing.T) {
	js := `{"package":"ZZHL","version":"1.0","patch":"1","components":{
	  "routines":["ZZHLRT"],
	  "hl7Applications":[{"name":"ZZHL_APP","facility":"500","countryCode":"USA"}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse hl7Applications spec: %v", err)
	}
	if len(s.Components.HL7Applications) != 1 {
		t.Fatalf("hl7Applications = %+v", s.Components.HL7Applications)
	}
	a := s.Components.HL7Applications[0]
	if a.Name != "ZZHL_APP" || a.Facility != "500" || a.CountryCode != "USA" {
		t.Errorf("hl7App = %+v", a)
	}
	// An app-only spec (just a name; underscores allowed) is non-empty.
	d, err := Parse([]byte(`{"package":"ZZHL","version":"1.0","components":{"hl7Applications":[{"name":"ZZHL APP TWO"}]}}`))
	if err != nil {
		t.Fatalf("Parse minimal hl7 app: %v", err)
	}
	if d.Components.empty() {
		t.Error("spec with an hl7 application must not be empty")
	}
}

func TestParse_HL7Applications_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":   `{"package":"ZZHL","version":"1.0","components":{"hl7Applications":[{"facility":"500"}]}}`,
		"lowercase": `{"package":"ZZHL","version":"1.0","components":{"hl7Applications":[{"name":"zzhl app"}]}}`,
		"bad char":  `{"package":"ZZHL","version":"1.0","components":{"hl7Applications":[{"name":"ZZHL@APP"}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_HLOApplications(t *testing.T) {
	js := `{"package":"ZZHO","version":"1.0","patch":"1","components":{
	  "routines":["ZZHORT"],
	  "hloApplications":[{"name":"ZZHO_APP","messageTypes":[{"messageType":"ORU","event":"R01","actionTag":"START","actionRoutine":"ZZHORT","version":"2.4"}]}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse hloApplications spec: %v", err)
	}
	if len(s.Components.HLOApplications) != 1 {
		t.Fatalf("hloApplications = %+v", s.Components.HLOApplications)
	}
	a := s.Components.HLOApplications[0]
	if a.Name != "ZZHO_APP" || len(a.MessageTypes) != 1 {
		t.Fatalf("hloApp = %+v", a)
	}
	mt := a.MessageTypes[0]
	if mt.MessageType != "ORU" || mt.Event != "R01" || mt.ActionRoutine != "ZZHORT" || mt.Version != "2.4" {
		t.Errorf("messageType = %+v", mt)
	}
	// A name-only HLO app (no message types) is still a non-empty spec.
	d, err := Parse([]byte(`{"package":"ZZHO","version":"1.0","components":{"hloApplications":[{"name":"ZZHO_APP"}]}}`))
	if err != nil {
		t.Fatalf("Parse minimal hlo app: %v", err)
	}
	if d.Components.empty() {
		t.Error("spec with an hlo application must not be empty")
	}
}

func TestParse_HLOApplications_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":      `{"package":"ZZHO","version":"1.0","components":{"hloApplications":[{"messageTypes":[{"messageType":"ORU","event":"R01"}]}]}}`,
		"lower name":   `{"package":"ZZHO","version":"1.0","components":{"hloApplications":[{"name":"zzho app"}]}}`,
		"bad msg type": `{"package":"ZZHO","version":"1.0","components":{"hloApplications":[{"name":"ZZHO_APP","messageTypes":[{"messageType":"oru","event":"R01"}]}]}}`,
		"bad event":    `{"package":"ZZHO","version":"1.0","components":{"hloApplications":[{"name":"ZZHO_APP","messageTypes":[{"messageType":"ORU","event":"r 01"}]}]}}`,
		"bad routine":  `{"package":"ZZHO","version":"1.0","components":{"hloApplications":[{"name":"ZZHO_APP","messageTypes":[{"messageType":"ORU","event":"R01","actionRoutine":"123bad"}]}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_LogicalLinks(t *testing.T) {
	js := `{"package":"ZZLL","version":"1.0","patch":"1","components":{
	  "routines":["ZZLLRT"],
	  "logicalLinks":[{"name":"ZZLINK","llpType":"TCP","port":"5000","serviceType":"C"}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse logicalLinks spec: %v", err)
	}
	if len(s.Components.LogicalLinks) != 1 {
		t.Fatalf("logicalLinks = %+v", s.Components.LogicalLinks)
	}
	l := s.Components.LogicalLinks[0]
	if l.Name != "ZZLINK" || l.LLPType != "TCP" || l.Port != "5000" || l.ServiceType != "C" {
		t.Errorf("logicalLink = %+v", l)
	}
	// A skeleton link (just a name) is a non-empty spec.
	d, err := Parse([]byte(`{"package":"ZZLL","version":"1.0","components":{"logicalLinks":[{"name":"ZZBARE"}]}}`))
	if err != nil {
		t.Fatalf("Parse minimal logical link: %v", err)
	}
	if d.Components.empty() {
		t.Error("spec with a logical link must not be empty")
	}
}

func TestParse_LogicalLinks_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":      `{"package":"ZZLL","version":"1.0","components":{"logicalLinks":[{"llpType":"TCP"}]}}`,
		"too short":    `{"package":"ZZLL","version":"1.0","components":{"logicalLinks":[{"name":"ZZ"}]}}`,
		"too long":     `{"package":"ZZLL","version":"1.0","components":{"logicalLinks":[{"name":"ZZLINKTOOLONG"}]}}`,
		"lead punct":   `{"package":"ZZLL","version":"1.0","components":{"logicalLinks":[{"name":"-ZZLINK"}]}}`,
		"bad svc type": `{"package":"ZZLL","version":"1.0","components":{"logicalLinks":[{"name":"ZZLINK","serviceType":"X"}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

func TestParse_Protocols(t *testing.T) {
	js := `{"package":"ZZPROTO","version":"1.0","patch":"1","components":{
	  "routines":["ZZPRORT"],
	  "protocols":[{"name":"ZZPROTO ACTION","itemText":"ZZ Protocol Action Demo","type":"action","entryAction":"Q"}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse protocols spec: %v", err)
	}
	if len(s.Components.Protocols) != 1 {
		t.Fatalf("protocols = %+v", s.Components.Protocols)
	}
	p := s.Components.Protocols[0]
	if p.Name != "ZZPROTO ACTION" || p.Type != "action" || p.EntryAction != "Q" {
		t.Errorf("protocol = %+v", p)
	}
}

func TestParse_Protocols_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":  `{"package":"ZZP","version":"1.0","components":{"protocols":[{"type":"action"}]}}`,
		"bad name": `{"package":"ZZP","version":"1.0","components":{"protocols":[{"name":"lower case","type":"action"}]}}`,
		"bad type": `{"package":"ZZP","version":"1.0","components":{"protocols":[{"name":"ZZP A","type":"bogus"}]}}`,
		"bad item": `{"package":"ZZP","version":"1.0","components":{"protocols":[{"name":"ZZP M","type":"menu","items":[{"name":"lower"}]}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// A menu PROTOCOL's ITEMs parse with name + sequence.
func TestParse_Protocols_Items(t *testing.T) {
	js := `{"package":"ZZP","version":"1.0","components":{"protocols":[
	  {"name":"ZZP MENU","type":"menu","items":[{"name":"ZZP ACTION","sequence":5}]}]}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse protocol items: %v", err)
	}
	its := s.Components.Protocols[0].Items
	if len(its) != 1 || its[0].Name != "ZZP ACTION" || its[0].Sequence != 5 {
		t.Errorf("items = %+v", its)
	}
}

func TestParse_Keys(t *testing.T) {
	js := `{"package":"ZZKEY","version":"1.0","patch":"1","components":{
	  "routines":["ZZKEYRT"],
	  "keys":[{"name":"ZZKEY MANAGER"}]
	}}`
	s, err := Parse([]byte(js))
	if err != nil {
		t.Fatalf("Parse keys spec: %v", err)
	}
	if len(s.Components.Keys) != 1 || s.Components.Keys[0].Name != "ZZKEY MANAGER" {
		t.Fatalf("keys = %+v", s.Components.Keys)
	}
	// A spec carrying only a key (no routines) is still non-empty.
	k, err := Parse([]byte(`{"package":"ZZKEY","version":"1.0","components":{"keys":[{"name":"ZZKEY A"}]}}`))
	if err != nil {
		t.Fatalf("Parse key-only spec: %v", err)
	}
	if k.Components.empty() {
		t.Error("spec with a key must not be empty")
	}
}

func TestParse_Keys_Invalid(t *testing.T) {
	cases := map[string]string{
		"no name":  `{"package":"ZZKEY","version":"1.0","components":{"keys":[{}]}}`,
		"bad name": `{"package":"ZZKEY","version":"1.0","components":{"keys":[{"name":"lower case"}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}

// A single build may now ship BOTH options and parameter definitions — they share
// one computed "BLD",1,"KRN",0) manifest header across entry types (B.1).
func TestParse_Options_WithParamDefs_Accepted(t *testing.T) {
	js := `{"package":"ZZMIX","version":"1.0","components":{
	  "options":[{"name":"ZZMIX A","menuText":"x","type":"action","entryAction":"W 1"}],
	  "parameterDefinitions":[{"name":"ZZMIX P","dataType":"free text"}]
	}}`
	if _, err := Parse([]byte(js)); err != nil {
		t.Errorf("a build mixing options + parameterDefinitions should be accepted now, got %v", err)
	}
}

// `parameters` is documented as reference-only metadata (like `icrs`), not a
// shipped component — it must NOT trip the unsupported-component gate.
func TestParse_ReferenceParametersAccepted(t *testing.T) {
	js := `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"parameters":["ZZSKEL PARAM"]}}`
	if _, err := Parse([]byte(js)); err != nil {
		t.Errorf("reference-only parameters should be accepted, got %v", err)
	}
}
