package buildspec

import (
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
		"number too low":  `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":200,"name":"ZZVSLFS"}]}}`,
		"number too high": `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":1000000,"name":"ZZVSLFS"}]}}`,
		"no name":         `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":999000}]}}`,
		"lower-case name": `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":999000,"name":"lower"}]}}`,
		"bad global root": `{"package":"ZZVSLFS","version":"1.0","components":{"files":[{"number":999000,"name":"ZZVSLFS","globalRoot":"DIZ999000"}]}}`,
	}
	for name, js := range cases {
		if _, err := Parse([]byte(js)); err == nil {
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
		"options":    `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"options":["ZZSKEL MENU"]}}`,
		"keys":       `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"keys":["ZZSKEL KEY"]}}`,
		"protocols":  `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"protocols":["ZZSKEL PROTO"]}}`,
		"templates":  `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"templates":["ZZSKEL TMPL"]}}`,
		"rpcs":       `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"rpcs":["ZZSKEL RPC"]}}`,
		"mailGroups": `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"mailGroups":["ZZSKEL MG"]}}`,
		"hl7":        `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"hl7":["ZZSKEL HL7"]}}`,
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

// `parameters` is documented as reference-only metadata (like `icrs`), not a
// shipped component — it must NOT trip the unsupported-component gate.
func TestParse_ReferenceParametersAccepted(t *testing.T) {
	js := `{"package":"ZZSKEL","version":"1.0","components":{"routines":["ZZSKEL"],"parameters":["ZZSKEL PARAM"]}}`
	if _, err := Parse([]byte(js)); err != nil {
		t.Errorf("reference-only parameters should be accepted, got %v", err)
	}
}
