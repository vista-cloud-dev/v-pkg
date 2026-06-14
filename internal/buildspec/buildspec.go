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
	Routines   []string   `json:"routines,omitempty"`
	Files      []FileComp `json:"files,omitempty"`
	Options    []string   `json:"options,omitempty"`
	Keys       []string   `json:"keys,omitempty"`       // security keys
	Parameters []string   `json:"parameters,omitempty"` // XPAR parameters
	Protocols  []string   `json:"protocols,omitempty"`
	Templates  []string   `json:"templates,omitempty"`
	RPCs       []string   `json:"rpcs,omitempty"`
	MailGroups []string   `json:"mailGroups,omitempty"`
	HL7        []string   `json:"hl7,omitempty"`
}

func (c Components) empty() bool {
	return len(c.Routines) == 0 && len(c.Files) == 0 && len(c.Options) == 0 &&
		len(c.Keys) == 0 && len(c.Parameters) == 0 && len(c.Protocols) == 0 &&
		len(c.Templates) == 0 && len(c.RPCs) == 0 && len(c.MailGroups) == 0 && len(c.HL7) == 0
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
