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
