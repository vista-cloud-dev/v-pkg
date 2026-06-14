// Package installspec is the declarative KIDS install spec + validating loader
// (VSL T0a.3): the answers `v pkg install` feeds to a non-interactive KIDS
// install — the source distribution, the environment-check choice, the standard
// KIDS questions, and the device/queue. It is the JSON form of the
// install-spec.yaml in docs/kids-installation-automation.md §5. The standard
// answers map onto the KIDS XPDDIQ answer codes (XPO1/XPI1/XPZ1) used to suppress
// the prompts (see that doc, Tier A).
package installspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// Spec is one install spec (kids/<pkg>.install.json or passed inline).
type Spec struct {
	Name         string        `json:"name"`                       // INSTALL NAME, e.g. ZZSKEL*1.0*1
	Source       Source        `json:"source"`                     // where the distribution comes from
	EnvCheck     string        `json:"environmentCheck,omitempty"` // "run" (default) | "skip"
	Backup       bool          `json:"backupTransportGlobal,omitempty"`
	Answers      Answers       `json:"answers"`
	Device       Device        `json:"device"`
	ExtraAnswers []ExtraAnswer `json:"extraAnswers,omitempty"` // build-specific pre/post questions
}

// Source is the distribution location.
type Source struct {
	Kind string `json:"kind"` // "hfs" (Host File) | "packman" (MailMan message)
	Path string `json:"path"`
}

// Answers are the four standard KIDS install questions.
type Answers struct {
	RebuildMenuTrees        bool `json:"rebuildMenuTrees"`
	InhibitLogons           bool `json:"inhibitLogons"`
	DisableOptionsProtocols bool `json:"disableOptionsProtocols"`
	DelayInstallMinutes     int  `json:"delayInstallMinutes"`
}

// XPDDIQ maps the standard answers onto the KIDS XPDDIQ answer codes (set in the
// environment-check to suppress the prompts): XPO1 = rebuild menu trees, XPI1 =
// inhibit logons, XPZ1 = disable options/protocols. "0" = NO, "1" = YES.
func (a Answers) XPDDIQ() map[string]string {
	yn := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	return map[string]string{
		"XPO1": yn(a.RebuildMenuTrees),
		"XPI1": yn(a.InhibitLogons),
		"XPZ1": yn(a.DisableOptionsProtocols),
	}
}

// Device selects the install output device / queueing.
type Device struct {
	Queue bool   `json:"queue"`
	At    string `json:"at,omitempty"` // ISO time when queued
}

// ExtraAnswer answers a build-specific pre/post-install question by prompt match.
type ExtraAnswer struct {
	PromptContains string `json:"promptContains"`
	Answer         string `json:"answer"`
}

var (
	reInstallName = regexp.MustCompile(`^[A-Z%][A-Z0-9]*\*\d+\.\d+(\*\d+)?$`)
	validSource   = map[string]bool{"hfs": true, "packman": true}
	validEnvCheck = map[string]bool{"": true, "run": true, "skip": true}
)

// Load reads + validates an install spec from a file.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("installspec: read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes an install spec from JSON (rejecting unknown fields) + validates.
func Parse(data []byte) (*Spec, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var s Spec
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("installspec: parse: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks the spec is usable for a non-interactive install.
func (s *Spec) Validate() error {
	if !reInstallName.MatchString(s.Name) {
		return fmt.Errorf("installspec: name %q must be NAMESPACE*VERSION[*PATCH]", s.Name)
	}
	if !validSource[s.Source.Kind] {
		return fmt.Errorf("installspec: source.kind %q must be hfs or packman", s.Source.Kind)
	}
	if s.Source.Path == "" {
		return fmt.Errorf("installspec: source.path is required")
	}
	if !validEnvCheck[s.EnvCheck] {
		return fmt.Errorf("installspec: environmentCheck %q must be run or skip", s.EnvCheck)
	}
	if d := s.Answers.DelayInstallMinutes; d < 0 || d > 60 {
		return fmt.Errorf("installspec: delayInstallMinutes %d out of range (0-60)", d)
	}
	for i, ea := range s.ExtraAnswers {
		if ea.PromptContains == "" {
			return fmt.Errorf("installspec: extraAnswers[%d] needs promptContains", i)
		}
	}
	return nil
}
