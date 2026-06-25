package kids

import (
	"sort"
	"strconv"
)

// ReversibilityClass is the statically-derived reversibility of a KIDS build —
// the keystone the proposal (docs/patch-existing-routines-proposal.md) requires:
// "detect the class from the .KID, don't trust the tag." It is derived ONLY from
// the transport global's structure, with no live engine.
//
// Corpus evidence (docs/kids-corpus-findings.md, 2,404 WorldVistA distributions):
// the pure-overwrite class is the MINORITY (~35%); ~64% are side-effecting. So
// the safe default for anything carrying install code, FileMan entries, or
// DD/data is SideEffecting — snapshot/restore is sound ONLY for PureOverwrite.
type ReversibilityClass int

const (
	// ClassPureOverwrite (class 1): ships only routine/metadata overwrites — no
	// install-time code, no exported FileMan entries, no DD/data. A pre-image
	// snapshot+restore fully reverses it. This is the XWBBRK-splice class.
	ClassPureOverwrite ReversibilityClass = iota + 1
	// ClassSideEffecting (class 2/3): runs install-time code (env-check /
	// pre-install / post-install) and/or files FileMan entries and/or ships
	// DD/data. There is no generic inverse — reversal needs an authored back-out
	// (class 2) or a forward back-out patch (class 3); restoring a routine
	// pre-image is incomplete and unsafe.
	ClassSideEffecting
)

func (c ReversibilityClass) String() string {
	switch c {
	case ClassPureOverwrite:
		return "pure-overwrite"
	case ClassSideEffecting:
		return "side-effecting"
	default:
		return "unknown"
	}
}

// installRoleNames maps the #9.6 BUILD subnode key to the install-time role it
// carries. A NON-EMPTY value at any of these means the build runs that code at
// install (ground-truthed against the corpus: "INI"=env-check "YS60PRE",
// "INIT"=post-install "YS60POST", "PRE"/"PRET"=pre-install/pre-transport). The
// "INID" node is run-timing FLAGS ("^y^n"), not code, and is excluded.
var installRoleNames = map[string]string{
	"INI":  "environment-check",
	"INIT": "post-install",
	"PRE":  "pre-install",
	"PRET": "pre-transport",
}

// BuildReversibility is the per-build static analysis used to drive class-aware
// install / uninstall / snapshot.
type BuildReversibility struct {
	Build               string             `json:"build"`
	Class               ReversibilityClass `json:"class"`
	ClassName           string             `json:"className"`
	ShipsRoutines       bool               `json:"shipsRoutines"`
	RoutineCount        int                `json:"routineCount"`
	Routines            []string           `json:"routines,omitempty"`
	HasInstallCode      bool               `json:"hasInstallCode"`
	InstallCode         map[string]string  `json:"installCode,omitempty"` // role -> entry tag
	ShipsFileManEntries bool               `json:"shipsFileManEntries"`
	FileManFiles        []string           `json:"fileManFiles,omitempty"` // file numbers with exported entries
	ShipsFileDD         bool               `json:"shipsFileDD"`
	FileDDFiles         []string           `json:"fileDDFiles,omitempty"` // file numbers shipped as a FILE (DD/data)
}

// Reversibility is the whole-distribution analysis: per-build plus the overall
// class, where the LEAST-reversible build governs (a multi-build patch is only
// as reversible as its most side-effecting member).
type Reversibility struct {
	Class     ReversibilityClass   `json:"class"`
	ClassName string               `json:"className"`
	Builds    []BuildReversibility `json:"builds"`
}

// numString renders a numeric subscript (int or decimal file number) the way it
// appears in the transport.
func numString(s Sub) string {
	if s.IsInt() {
		return strconv.FormatInt(s.intV, 10)
	}
	if s.IsFloat() {
		return formatKIDSFloat(s.fltV)
	}
	return s.str
}

// ClassifyBuild derives the reversibility of a single parsed build by walking
// its transport pairs and probing four node shapes (routine name, install-code
// subnode, exported FileMan entry, FileMan FILE multiple).
func ClassifyBuild(name string, b *Build) BuildReversibility {
	r := BuildReversibility{Build: name, InstallCode: map[string]string{}}
	rtns := map[string]bool{}
	krnFiles := map[string]bool{}
	ddFiles := map[string]bool{}

	for _, p := range b.Pairs() {
		s := p.Subs
		if len(s) == 0 || !s[0].IsStr() {
			continue
		}
		switch s[0].str {
		case "RTN":
			// routine NAME node: "RTN","<name>") — len 2, string name. (The
			// bare "RTN") header and "RTN","X",n,0) source lines are skipped.)
			if len(s) == 2 && s[1].IsStr() {
				rtns[s[1].str] = true
			}
		case "KRN":
			// exported FileMan entry: "KRN",<file#>,<ien>,0) — distinguishes a
			// real record from the "KRN",<file#>,0) component header and the
			// "KRN",<file#>,"B",... cross-ref/name nodes.
			if len(s) >= 4 && s[1].IsNumeric() && s[2].IsNumeric() && s[3].IsZeroInt() {
				krnFiles[numString(s[1])] = true
			}
		case "BLD":
			switch {
			// install-code subnode: "BLD",<bld#>,"<ROLE>") with a non-empty entry.
			case len(s) == 3 && s[2].IsStr() && p.Value != "":
				if role, ok := installRoleNames[s[2].str]; ok {
					r.HasInstallCode = true
					r.InstallCode[role] = p.Value
				}
			// FileMan FILE (DD/data): "BLD",<bld#>,4,<file#>,0) — a numbered
			// entry under the FILE multiple (field 4). The "BLD",<bld#>,4,0)
			// header is len 4 and excluded.
			case len(s) >= 5 && s[2].IsInt() && s[2].Int() == 4 && s[3].IsNumeric() && s[4].IsZeroInt():
				ddFiles[numString(s[3])] = true
			}
		}
	}

	r.ShipsRoutines = len(rtns) > 0
	r.RoutineCount = len(rtns)
	r.Routines = sortedKeys(rtns)
	r.ShipsFileManEntries = len(krnFiles) > 0
	r.FileManFiles = sortedKeys(krnFiles)
	r.ShipsFileDD = len(ddFiles) > 0
	r.FileDDFiles = sortedKeys(ddFiles)
	if len(r.InstallCode) == 0 {
		r.InstallCode = nil
	}

	if r.HasInstallCode || r.ShipsFileManEntries || r.ShipsFileDD {
		r.Class = ClassSideEffecting
	} else {
		r.Class = ClassPureOverwrite
	}
	r.ClassName = r.Class.String()
	return r
}

// Classify derives reversibility for every build in a KID. The overall class is
// the least-reversible build (SideEffecting if ANY build is side-effecting).
func Classify(k *KID) Reversibility {
	rev := Reversibility{Class: ClassPureOverwrite}
	for _, name := range k.InstallNames {
		br := ClassifyBuild(name, k.Builds[name])
		rev.Builds = append(rev.Builds, br)
		if br.Class == ClassSideEffecting {
			rev.Class = ClassSideEffecting
		}
	}
	rev.ClassName = rev.Class.String()
	return rev
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
