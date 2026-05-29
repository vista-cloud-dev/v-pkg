package kids

import (
	"bufio"
	"os"
	"sort"
	"strconv"
	"strings"
)

// PIKSClass is the data-sensitivity class of a FileMan file under vista-meta's
// PIKS model: Patient, Institution, Knowledge, or System. Patient/Institution
// are operational data (refused from the versioned tree); Knowledge/System are
// definitional configuration (safely packageable). See kids-package-round-trip
// §3 and ADR-045 (data/code separation).
type PIKSClass string

const (
	ClassPatient     PIKSClass = "Patient"
	ClassInstitution PIKSClass = "Institution"
	ClassKnowledge   PIKSClass = "Knowledge"
	ClassSystem      PIKSClass = "System"
	ClassUnknown     PIKSClass = "Unknown"
)

// blocked reports whether a class is refused by the data-class gate (operational
// data: Patient or Institution).
func (c PIKSClass) blocked() bool {
	return c == ClassPatient || c == ClassInstitution
}

// dataSections are the KIDS sections carrying operational/instance data — the
// surface the K2 data-class gate inspects.
var dataSections = map[string]bool{
	"DATA": true, "FRV1": true, "FRVL": true, "FRV1K": true,
}

// builtinPIKS is a SEED classification of well-known FileMan files. It is NOT
// the authoritative model — vista-meta's full PIKS model covers 8,261 files and
// is consumed by reference via LoadPIKS / --piks. This seed lets the gate flag
// the canonical operational-data files (e.g. File 2 PATIENT) standalone, and
// mark the common definitional files (options, protocols, RPCs, templates) as
// safe so they don't read as Unknown.
var builtinPIKS = map[float64]PIKSClass{
	// Operational data — refused.
	2:   ClassPatient,     // PATIENT
	4:   ClassInstitution, // INSTITUTION
	405: ClassPatient,     // PATIENT MOVEMENT
	45:  ClassPatient,     // PTF
	63:  ClassPatient,     // LAB DATA
	100: ClassPatient,     // ORDER
	// Definitional configuration — safely packageable.
	0.4: ClassSystem, 0.401: ClassSystem, 0.402: ClassSystem, 0.403: ClassSystem, 0.404: ClassSystem,
	3.7: ClassSystem, // DEVICE
	9.2: ClassKnowledge, 9.4: ClassKnowledge, 9.6: ClassKnowledge, 9.7: ClassKnowledge, 9.8: ClassKnowledge,
	19: ClassKnowledge, 19.1: ClassSystem, 19.2: ClassKnowledge,
	101: ClassKnowledge, 101.41: ClassKnowledge,
	771: ClassKnowledge, 870: ClassSystem, 8989.51: ClassSystem, 8989.52: ClassSystem,
	8993: ClassKnowledge, 8994: ClassKnowledge,
}

// PIKSClassifier classifies FileMan file numbers. It layers an external table
// (authoritative, from vista-meta) over the built-in seed.
type PIKSClassifier struct {
	table map[float64]PIKSClass
}

// NewPIKSClassifier returns a classifier seeded with the built-in map.
func NewPIKSClassifier() *PIKSClassifier {
	t := make(map[float64]PIKSClass, len(builtinPIKS))
	for k, v := range builtinPIKS {
		t[k] = v
	}
	return &PIKSClassifier{table: t}
}

// LoadPIKS overlays an external PIKS table onto the classifier. The file is a
// TSV (or whitespace-separated) of `filenumber<sep>class` lines; `class` may be
// a single letter (P/I/K/S) or a full word (Patient/…). Blank lines and lines
// beginning with '#' are ignored. This is how vista-meta's authoritative PIKS
// export is supplied — kids-vc never vendors it.
func (c *PIKSClassifier) LoadPIKS(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.FieldsFunc(line, func(r rune) bool { return r == '\t' || r == ' ' })
		if len(fields) < 2 {
			continue
		}
		num, perr := strconv.ParseFloat(fields[0], 64)
		if perr != nil {
			continue
		}
		if cls := parsePIKSClass(fields[1]); cls != ClassUnknown {
			c.table[num] = cls
		}
	}
	return sc.Err()
}

func parsePIKSClass(s string) PIKSClass {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "P", "PATIENT":
		return ClassPatient
	case "I", "INSTITUTION":
		return ClassInstitution
	case "K", "KNOWLEDGE":
		return ClassKnowledge
	case "S", "SYSTEM":
		return ClassSystem
	default:
		return ClassUnknown
	}
}

// Classify returns the class of a file number (Unknown if unseen).
func (c *PIKSClassifier) Classify(fnum float64) PIKSClass {
	if cls, ok := c.table[fnum]; ok {
		return cls
	}
	return ClassUnknown
}

// LintFinding is one data-class gate finding against a single FileMan file.
type LintFinding struct {
	File     string    `json:"file"`     // file number as it appears
	Class    PIKSClass `json:"class"`    // resolved PIKS class
	Section  string    `json:"section"`  // KIDS data section it appeared in
	Severity string    `json:"severity"` // "error" (blocked) | "warning" (unclassified)
	Message  string    `json:"message"`
}

// LintResult is the outcome of the data-class gate.
type LintResult struct {
	Findings     []LintFinding `json:"findings"`
	DataFiles    int           `json:"dataFiles"`    // distinct files with operational data
	Blocked      int           `json:"blocked"`      // Patient/Institution findings
	Unclassified int           `json:"unclassified"` // files with no known class
	OK           bool          `json:"ok"`           // gate passed
}

// dataFileNumbers collects, per build, the file numbers that carry operational
// data (DATA/FRV* sections), keyed to the section they appeared in. The file
// number is the first numeric element at position 1 or 2 (mirroring
// matchesFile's scan of both shapes).
func dataFileNumbers(build *Build) map[float64]string {
	files := map[float64]string{}
	for _, p := range build.Pairs() {
		if len(p.Subs) < 2 || !p.Subs[0].IsStr() || !dataSections[p.Subs[0].str] {
			continue
		}
		section := p.Subs[0].str
		if v, ok := p.Subs[1].numVal(); ok {
			if _, seen := files[v]; !seen {
				files[v] = section
			}
			continue
		}
		if len(p.Subs) >= 3 {
			if v, ok := p.Subs[2].numVal(); ok {
				if _, seen := files[v]; !seen {
					files[v] = section
				}
			}
		}
	}
	return files
}

// LintDataClass runs the PIKS data-class gate (K2 / N9) over a parsed KID.
// Operational-data (DATA/FRV*) sections touching a Patient/Institution-class
// file are gate FAILURES (severity error). Files with no known class are
// reported as warnings; with strict=true they are also treated as gate
// failures (fail-closed when no authoritative classifier is supplied).
func LintDataClass(kid *KID, classifier *PIKSClassifier, strict bool) LintResult {
	type entry struct {
		fnum    float64
		section string
	}
	merged := map[float64]string{}
	for _, b := range kid.Builds {
		for fnum, section := range dataFileNumbers(b) {
			if _, ok := merged[fnum]; !ok {
				merged[fnum] = section
			}
		}
	}

	entries := make([]entry, 0, len(merged))
	for fnum, section := range merged {
		entries = append(entries, entry{fnum, section})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].fnum < entries[j].fnum })

	res := LintResult{DataFiles: len(entries), OK: true}
	for _, e := range entries {
		cls := classifier.Classify(e.fnum)
		num := strconv.FormatFloat(e.fnum, 'f', -1, 64)
		switch {
		case cls.blocked():
			res.Blocked++
			res.OK = false
			res.Findings = append(res.Findings, LintFinding{
				File: num, Class: cls, Section: e.section, Severity: "error",
				Message: "operational data for a " + string(cls) + "-class file must not be versioned (data-class gate)",
			})
		case cls == ClassUnknown:
			res.Unclassified++
			sev := "warning"
			msg := "unclassified file in a " + e.section + " section — supply --piks for authoritative PIKS classification"
			if strict {
				sev = "error"
				res.OK = false
				msg = "unclassified file in a " + e.section + " section refused under --strict (no authoritative classifier)"
			}
			res.Findings = append(res.Findings, LintFinding{
				File: num, Class: cls, Section: e.section, Severity: sev, Message: msg,
			})
		default:
			// Knowledge/System operational data is permitted; no finding.
		}
	}
	return res
}
