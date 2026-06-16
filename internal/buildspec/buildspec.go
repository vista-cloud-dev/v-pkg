// Package buildspec is the KIDS build-spec schema + validating loader
// (VSL T0a.1, CQ9): the declarative, diffable git source of a KIDS BUILD (#9.6),
// kids/<pkg>.build.json. It is the human-readable form of a package definition —
// component list, Required Builds, the environment-check routine, and the ICR
// list — that `v pkg build` consumes to produce a transport global / .KID.
// See msl-vsl-coordination-implementation-plan.md §7.2.
package buildspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Required-build install actions (KIDS BUILD #9.6 REQUIRED BUILD #11 multiple;
// architecture §3.1).
const (
	ActionWarn         = "WARNING ONLY"                 // warn, continue
	ActionLeaveGlobal  = "DON'T INSTALL, LEAVE GLOBAL"  // block unless prerequisite present; keep its global
	ActionRemoveGlobal = "DON'T INSTALL, REMOVE GLOBAL" // block unless prerequisite present; remove its global
)

var validActions = map[string]bool{ActionWarn: true, ActionLeaveGlobal: true, ActionRemoveGlobal: true}

// RequiredBuildActionCode maps a Required-Build action to its #9.611 ACTION
// set-of-codes value (0 = warn, 1 = don't install/remove global, 2 = don't
// install/leave global).
var RequiredBuildActionCode = map[string]int{
	ActionWarn:         0,
	ActionRemoveGlobal: 1,
	ActionLeaveGlobal:  2,
}

// Spec is one KIDS build definition (kids/<pkg>.build.json).
type Spec struct {
	Package        string          `json:"package"`         // NAMESPACE (e.g. ZZSKEL)
	Version        string          `json:"version"`         // e.g. 1.0
	Patch          string          `json:"patch,omitempty"` // e.g. 1 (optional)
	Components     Components      `json:"components"`
	RequiredBuilds []RequiredBuild `json:"requiredBuilds,omitempty"`
	EnvCheck       string          `json:"envCheck,omitempty"` // environment-check routine name (XPDENV)
	ICRs           []ICR           `json:"icrs,omitempty"`     // DBIA/ICR agreements the package relies on
}

// Components is the BUILD component list (#9.6). Each slice is omitempty so a
// spec lists only what it ships.
type Components struct {
	Routines             []string   `json:"routines,omitempty"`
	Files                []FileComp `json:"files,omitempty"`
	Options              []string   `json:"options,omitempty"`
	Keys                 []string   `json:"keys,omitempty"`                 // security keys
	Parameters           []string   `json:"parameters,omitempty"`           // XPAR parameter names (reference only)
	ParameterDefinitions []ParamDef `json:"parameterDefinitions,omitempty"` // XPAR #8989.51 PARAMETER DEFINITION components (shipped as data)
	Protocols            []string   `json:"protocols,omitempty"`
	Templates            []string   `json:"templates,omitempty"`
	RPCs                 []string   `json:"rpcs,omitempty"`
	MailGroups           []string   `json:"mailGroups,omitempty"`
	HL7                  []string   `json:"hl7,omitempty"`
}

func (c Components) empty() bool {
	return len(c.Routines) == 0 && len(c.Files) == 0 && len(c.Options) == 0 &&
		len(c.Keys) == 0 && len(c.Parameters) == 0 && len(c.ParameterDefinitions) == 0 &&
		len(c.Protocols) == 0 && len(c.Templates) == 0 && len(c.RPCs) == 0 &&
		len(c.MailGroups) == 0 && len(c.HL7) == 0
}

// ParamDef is an XPAR PARAMETER DEFINITION (#8989.51) shipped as a KIDS KRN
// component — the build creates the definition (not a value) at install time.
// DataType/Entity are human names resolved to their #8989.51 codes / #8989.518
// IENs by ParamDataTypeCode / ParamEntityIEN.
type ParamDef struct {
	Name        string        `json:"name"`                  // #8989.51 .01 NAME (uppercase, e.g. "VSL GREETING")
	DisplayText string        `json:"displayText,omitempty"` // #8989.51 .02 DISPLAY TEXT
	DataType    string        `json:"dataType,omitempty"`    // value data type; default "free text"
	Entities    []ParamEntity `json:"entities,omitempty"`    // ALLOWABLE ENTITIES (#8989.513) — where the param may be set
}

// ParamEntity is one ALLOWABLE ENTITIES row of a ParamDef: which entity the
// parameter may be set at, and its precedence.
type ParamEntity struct {
	Entity     string `json:"entity"`               // entity abbreviation: SYS, USR, PKG, …
	Precedence int    `json:"precedence,omitempty"` // #8989.513 .01 PRECEDENCE; default 1
}

// ParamDataTypeCode maps a human value-data-type name to the #8989.51 field 1.1
// (VALUE DATA TYPE) set-of-codes value. Free text is the default; the rest are
// accepted for completeness.
var ParamDataTypeCode = map[string]string{
	"free text":       "F",
	"numeric":         "N",
	"yes/no":          "Y",
	"set of codes":    "S",
	"date/time":       "D",
	"pointer":         "P",
	"word processing": "W",
}

// ParamEntityIEN maps a parameter-entity abbreviation to its #8989.518
// (PARAMETER ENTITY) IEN. #8989.518 is DINUM'd to the pointed-to file number, so
// these IENs are national constants — portable across YDB and IRIS VistA.
var ParamEntityIEN = map[string]string{
	"DEV": "3.5",  // DEVICE
	"DIV": "4",    // DIVISION
	"SYS": "4.2",  // SYSTEM (KERNEL SYSTEM PARAMETERS)
	"PKG": "9.4",  // PACKAGE
	"LOC": "44",   // LOCATION
	"SRV": "49",   // SERVICE
	"USR": "200",  // USER (NEW PERSON)
	"CLS": "8930", // CLASS
}

// FileComp is a FileMan file component (number may be fractional, e.g. 8989.51).
type FileComp struct {
	Number float64 `json:"number"`
	Name   string  `json:"name,omitempty"`
}

// RequiredBuild is a KIDS Required Build (#11) — a prerequisite build + the
// action KIDS takes when it is absent.
type RequiredBuild struct {
	Name   string `json:"name"`   // NAMESPACE*VERSION[*PATCH]
	Action string `json:"action"` // one of the Action* constants
}

// ICR is a DBIA/Integration Control Registration the package depends on.
type ICR struct {
	Number    int    `json:"number"`
	Name      string `json:"name,omitempty"`
	Custodian string `json:"custodian,omitempty"`
}

// InstallName is the KIDS install name NAMESPACE*VERSION[*PATCH].
func (s *Spec) InstallName() string {
	n := s.Package + "*" + s.Version
	if s.Patch != "" {
		n += "*" + s.Patch
	}
	return n
}

var (
	reNamespace = regexp.MustCompile(`^[A-Z%][A-Z0-9]*$`)
	reVersion   = regexp.MustCompile(`^\d+\.\d+$`)
	rePatch     = regexp.MustCompile(`^\d+$`)
	reRoutine   = regexp.MustCompile(`^%?[A-Z][A-Z0-9]*$`)                  // M routine name
	reReqBuild  = regexp.MustCompile(`^[A-Z%][A-Z0-9]*\*\d+\.\d+(\*\d+)?$`) // NS*VER[*PATCH]
	reParamName = regexp.MustCompile(`^[A-Z][A-Z0-9 ]*[A-Z0-9]$|^[A-Z]$`)   // #8989.51 NAME (uppercase, internal spaces)
)

// Load reads and validates a build spec from a kids/<pkg>.build.json file.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("buildspec: read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes a build spec from JSON (rejecting unknown fields — a typo'd key
// is an error, not a silent drop) and validates it.
func Parse(data []byte) (*Spec, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var s Spec
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("buildspec: parse: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks the spec is a usable KIDS build definition.
func (s *Spec) Validate() error {
	if !reNamespace.MatchString(s.Package) {
		return fmt.Errorf("buildspec: package %q is not a valid VistA namespace", s.Package)
	}
	if !reVersion.MatchString(s.Version) {
		return fmt.Errorf("buildspec: version %q must be MAJOR.MINOR (e.g. 1.0)", s.Version)
	}
	if s.Patch != "" && !rePatch.MatchString(s.Patch) {
		return fmt.Errorf("buildspec: patch %q must be numeric", s.Patch)
	}
	if s.Components.empty() {
		return fmt.Errorf("buildspec: %s has no components — a build must ship at least one", s.InstallName())
	}
	if err := validateRoutines(s.Components.Routines); err != nil {
		return err
	}
	if err := validateParamDefs(s.Components.ParameterDefinitions); err != nil {
		return err
	}
	for _, rb := range s.RequiredBuilds {
		if !reReqBuild.MatchString(rb.Name) {
			return fmt.Errorf("buildspec: required build name %q must be NAMESPACE*VERSION[*PATCH]", rb.Name)
		}
		if !validActions[rb.Action] {
			return fmt.Errorf("buildspec: required build %s action %q is not a KIDS install action", rb.Name, rb.Action)
		}
	}
	if s.EnvCheck != "" && !isRoutineName(s.EnvCheck) {
		return fmt.Errorf("buildspec: envCheck %q is not a valid routine name (≤8 chars)", s.EnvCheck)
	}
	for _, icr := range s.ICRs {
		if icr.Number <= 0 {
			return fmt.Errorf("buildspec: ICR number must be positive, got %d", icr.Number)
		}
	}
	return nil
}

// validateParamDefs checks each PARAMETER DEFINITION component: a valid #8989.51
// NAME (≤30 chars, uppercase), a known value data type, and known allowable
// entities with non-negative precedence.
func validateParamDefs(defs []ParamDef) error {
	for _, p := range defs {
		if p.Name == "" {
			return fmt.Errorf("buildspec: a parameterDefinition is missing its name")
		}
		if len(p.Name) > 30 || !reParamName.MatchString(p.Name) {
			return fmt.Errorf("buildspec: parameter name %q must be uppercase ≤30 chars (#8989.51 NAME)", p.Name)
		}
		if p.DataType != "" {
			if _, ok := ParamDataTypeCode[p.DataType]; !ok {
				return fmt.Errorf("buildspec: parameter %s data type %q is not a known #8989.51 value type", p.Name, p.DataType)
			}
		}
		for _, e := range p.Entities {
			if _, ok := ParamEntityIEN[e.Entity]; !ok {
				return fmt.Errorf("buildspec: parameter %s allowable entity %q is not a known parameter entity (SYS, USR, …)", p.Name, e.Entity)
			}
			if e.Precedence < 0 {
				return fmt.Errorf("buildspec: parameter %s entity %s precedence must be ≥ 0", p.Name, e.Entity)
			}
		}
	}
	return nil
}

func validateRoutines(rtns []string) error {
	for _, r := range rtns {
		if !isRoutineName(r) {
			return fmt.Errorf("buildspec: %q is not a valid routine name (≤8 chars, M naming)", r)
		}
	}
	return nil
}

// isRoutineName reports whether s is a valid VistA/M routine name (≤8 chars).
func isRoutineName(s string) bool {
	return len(s) >= 1 && len(s) <= 8 && reRoutine.MatchString(s) && !strings.ContainsAny(s, " \t")
}
