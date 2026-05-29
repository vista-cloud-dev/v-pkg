package kids

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// simpleSection is a top-level KIDS section that maps to a single flat .zwr
// file (XPDK2VC's GENOUT). The slice preserves the Python dict's order.
type simpleSection struct{ section, filename string }

var simpleSections = []simpleSection{
	{"BLD", "Build.zwr"},
	{"PKG", "Package.zwr"},
	{"VER", "KernelFMVersion.zwr"},
	{"PRE", "EnvironmentCheck.zwr"},
	{"INI", "PreInit.zwr"},
	{"INIT", "PostInstall.zwr"},
	{"MBREQ", "RequiredBuild.zwr"},
	{"QUES", "InstallQuestions.zwr"},
	{"TEMP", "TransportGlobal.zwr"},
}

// filemanSections are the sections co-located per FileMan file under Files/.
var filemanSections = []string{
	"FIA", "^DD", "^DIC", "SEC", "UP", "IX", "KEY", "KEYPTR",
	"PGL", "DATA", "FRV1", "FRVL", "FRV1K",
}

var unsafePathChars = regexp.MustCompile(`[\\/:!@#$%^&*()?<>" ]`)

// sanitize replaces filesystem-unsafe characters with "-" (XPDK2VC's rule).
func sanitize(name string) string {
	s := unsafePathChars.ReplaceAllString(name, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "unnamed"
	}
	return s
}

// PatchDescriptorToDir maps an install name to its directory name
// (VMTEST*1.0*1 → VMTEST_1.0_1), XPDK2VC's PD4FS convention.
func PatchDescriptorToDir(desc string) string {
	return strings.ReplaceAll(desc, "*", "_")
}

// CanonicalizeRoutineLine2 applies the diff-stability transform to a routine's
// `;;version` line: keep `;;VERSION;PACKAGE` (pieces 1–4), blank pieces 5 and 6
// (the patch list, date, and Build N — volatile on every install). Port of
// canonicalize_routine_line2.
func CanonicalizeRoutineLine2(line2 string) string {
	parts := strings.Split(line2, ";")
	if len(parts) < 4 {
		return line2
	}
	keep := append([]string{}, parts[:4]...)
	keep = append(keep, "", "")
	return strings.Join(keep, ";")
}

// numVal returns the numeric value of an int/float subscript element.
func (s Sub) numVal() (float64, bool) {
	switch s.kind {
	case kindInt:
		return float64(s.intV), true
	case kindFloat:
		return s.fltV, true
	default:
		return 0, false
	}
}

// display renders a numeric subscript for use in a directory name, matching
// Python's str(fnum).
func (s Sub) display() string {
	switch s.kind {
	case kindStr:
		return s.str
	case kindInt:
		return strconv.FormatInt(s.intV, 10)
	default:
		return formatKIDSFloat(s.fltV)
	}
}

// subEqual compares two subscript elements with Python's == semantics: numerics
// compare by value across int/float, strings compare exactly.
func subEqual(a, b Sub) bool {
	an, aok := a.numVal()
	bn, bok := b.numVal()
	if aok && bok {
		return an == bn
	}
	if !aok && !bok {
		return a.str == b.str
	}
	return false
}

// orderedSections buckets a build's pairs by top-level section, preserving
// insertion order within each section.
func bucketSections(build *Build) map[string]*Build {
	sections := map[string]*Build{}
	get := func(name string) *Build {
		b, ok := sections[name]
		if !ok {
			b = newBuild()
			sections[name] = b
		}
		return b
	}
	for _, p := range build.Pairs() {
		sec := "UNKNOWN"
		if len(p.Subs) > 0 && p.Subs[0].IsStr() {
			sec = p.Subs[0].str
		}
		get(sec).Set(p.Subs, p.Value)
	}
	return sections
}

// writeSortedZWR writes pairs (sorted by the _sort_key collation) to path, one
// `subs)="value"` line each.
func writeSortedZWR(path string, pairs []Pair) error {
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].Subs.less(pairs[j].Subs) })
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(zwrLine(p.Subs, p.Value))
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// DecomposeBuild decomposes one build's parsed data into a per-component file
// tree under outDir. Faithful port of decompose_build: no IEN substitution
// (subscripts preserved exactly); routine line 2 IS canonicalized.
func DecomposeBuild(build *Build, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	sections := bucketSections(build)

	// Simple GENOUT-style sections → one .zwr each.
	for _, ss := range simpleSections {
		sec, ok := sections[ss.section]
		if !ok {
			continue
		}
		if err := writeSortedZWR(filepath.Join(outDir, ss.filename), sec.Sorted()); err != nil {
			return err
		}
	}

	if err := decomposeRoutines(sections, outDir); err != nil {
		return err
	}

	// ORD — top-level single file.
	if sec, ok := sections["ORD"]; ok {
		if err := writeSortedZWR(filepath.Join(outDir, "ORD.zwr"), sec.Sorted()); err != nil {
			return err
		}
	}

	if err := decomposeKRN(sections, outDir); err != nil {
		return err
	}

	consumed, err := decomposeFIA(sections, outDir)
	if err != nil {
		return err
	}

	// Unclaimed fileman-section entries → Files/_unclaimed.zwr.
	var unclaimed []Pair
	for _, section := range filemanSections {
		if sec, ok := sections[section]; ok {
			for _, p := range sec.Pairs() {
				if !consumed[p.Subs.key()] {
					unclaimed = append(unclaimed, p)
				}
			}
		}
	}
	if len(unclaimed) > 0 {
		filesRoot := filepath.Join(outDir, "Files")
		if err := os.MkdirAll(filesRoot, 0o755); err != nil {
			return err
		}
		if err := writeSortedZWR(filepath.Join(filesRoot, "_unclaimed.zwr"), unclaimed); err != nil {
			return err
		}
	}

	// Catch-all for unrecognized top-level sections.
	known := map[string]bool{"RTN": true, "ORD": true, "KRN": true}
	for _, ss := range simpleSections {
		known[ss.section] = true
	}
	for _, s := range filemanSections {
		known[s] = true
	}
	var unknownNames []string
	for name := range sections {
		if !known[name] {
			unknownNames = append(unknownNames, name)
		}
	}
	if len(unknownNames) > 0 {
		sort.Strings(unknownNames)
		var misc []Pair
		for _, name := range unknownNames {
			misc = append(misc, sections[name].Sorted()...)
		}
		// Each section already sorted; the file concatenates them in section
		// order (matching Python's nested sorted loop).
		var b strings.Builder
		for _, p := range misc {
			b.WriteString(zwrLine(p.Subs, p.Value))
			b.WriteByte('\n')
		}
		if err := os.WriteFile(filepath.Join(outDir, "_misc.zwr"), []byte(b.String()), 0o644); err != nil {
			return err
		}
	}
	return nil
}

type routineBody struct {
	header string
	lines  map[int64]string
}

func decomposeRoutines(sections map[string]*Build, outDir string) error {
	sec, ok := sections["RTN"]
	if !ok {
		return nil
	}
	rtnDir := filepath.Join(outDir, "Routines")
	if err := os.MkdirAll(rtnDir, 0o755); err != nil {
		return err
	}

	routines := map[string]*routineBody{}
	var order []string
	var misc []Pair
	for _, p := range sec.Pairs() {
		if len(p.Subs) < 2 {
			misc = append(misc, p)
			continue
		}
		name := p.Subs[1].display()
		rb := routines[name]
		if rb == nil {
			rb = &routineBody{lines: map[int64]string{}}
			routines[name] = rb
			order = append(order, name)
		}
		switch len(p.Subs) {
		case 2:
			rb.header = p.Value
		case 4:
			if p.Subs[2].IsInt() {
				rb.lines[p.Subs[2].intV] = p.Value
			}
		}
	}

	if len(misc) > 0 {
		if err := writeSortedZWR(filepath.Join(rtnDir, "_index.zwr"), misc); err != nil {
			return err
		}
	}

	for _, name := range order {
		rb := routines[name]
		if err := os.WriteFile(filepath.Join(rtnDir, name+".header"), []byte(rb.header+"\n"), 0o644); err != nil {
			return err
		}
		lineNos := make([]int64, 0, len(rb.lines))
		for ln := range rb.lines {
			lineNos = append(lineNos, ln)
		}
		sort.Slice(lineNos, func(i, j int) bool { return lineNos[i] < lineNos[j] })
		var b strings.Builder
		for _, ln := range lineNos {
			content := rb.lines[ln]
			if ln == 2 {
				content = CanonicalizeRoutineLine2(content)
			}
			b.WriteString(content)
			b.WriteByte('\n')
		}
		if err := os.WriteFile(filepath.Join(rtnDir, name+".m"), []byte(b.String()), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func decomposeKRN(sections map[string]*Build, outDir string) error {
	sec, ok := sections["KRN"]
	if !ok {
		return nil
	}

	// Split into per-file numeric groups and non-numeric cross-refs.
	type fileGroup struct {
		fnum  Sub
		pairs []Pair
	}
	groups := map[string]*fileGroup{}
	var groupOrder []string
	var nonNumeric []Pair
	for _, p := range sec.Pairs() {
		if len(p.Subs) < 3 {
			nonNumeric = append(nonNumeric, p)
			continue
		}
		fnum := p.Subs[1]
		if !fnum.IsNumeric() {
			nonNumeric = append(nonNumeric, p)
			continue
		}
		fk := fnum.key()
		g := groups[fk]
		if g == nil {
			g = &fileGroup{fnum: fnum}
			groups[fk] = g
			groupOrder = append(groupOrder, fk)
		}
		g.pairs = append(g.pairs, p)
	}

	if len(nonNumeric) > 0 {
		krnDir := filepath.Join(outDir, "KRN")
		if err := os.MkdirAll(krnDir, 0o755); err != nil {
			return err
		}
		if err := writeSortedZWR(filepath.Join(krnDir, "_misc.zwr"), nonNumeric); err != nil {
			return err
		}
	}

	for _, fk := range groupOrder {
		g := groups[fk]
		fileName := resolveKernelFileName(g.fnum)
		fileDir := filepath.Join(outDir, "KRN", sanitize(fileName))
		if err := os.MkdirAll(fileDir, 0o755); err != nil {
			return err
		}

		// Header nodes: IEN=0 or non-integer position-2 cross-refs.
		var headers []Pair
		// Per-entry groups by integer IEN > 0.
		byIEN := map[int64][]Pair{}
		var ienOrder []int64
		for _, p := range g.pairs {
			if len(p.Subs) < 3 {
				continue
			}
			s2 := p.Subs[2]
			if s2.IsZeroInt() || !s2.IsInt() {
				headers = append(headers, p)
			}
			if !s2.IsInt() || s2.intV == 0 {
				continue
			}
			ien := s2.intV
			if _, ok := byIEN[ien]; !ok {
				ienOrder = append(ienOrder, ien)
			}
			byIEN[ien] = append(byIEN[ien], p)
		}

		if len(headers) > 0 {
			if err := writeSortedZWR(filepath.Join(fileDir, "FileHeader.zwr"), headers); err != nil {
				return err
			}
		}

		// Derive a name per IEN, then disambiguate sanitized-name collisions.
		sanitizedNames := map[int64]string{}
		nameCounts := map[string]int{}
		for _, ien := range ienOrder {
			n := sanitize(deriveKRNEntryName(ien, byIEN[ien]))
			sanitizedNames[ien] = n
			nameCounts[n]++
		}
		for _, ien := range ienOrder {
			name := sanitizedNames[ien]
			if nameCounts[name] > 1 {
				name = name + "__ien" + strconv.FormatInt(ien, 10)
			}
			if err := writeSortedZWR(filepath.Join(fileDir, name+".zwr"), byIEN[ien]); err != nil {
				return err
			}
		}
	}
	return nil
}

// deriveKRNEntryName derives a presentable entry name from a KRN entry's zero
// node (port of the nested _entry_zero_node/_derive_name helpers).
func deriveKRNEntryName(ien int64, entry []Pair) string {
	zn := krnEntryZeroNode(entry)
	if zn == "" {
		return "ien-" + strconv.FormatInt(ien, 10)
	}
	p := strings.Split(zn, "^")
	// If piece 1 looks like a storage spec (";" and "(" and ends ","), prefer
	// piece 2 (the meaningful name, e.g. File 8989.5 PARAMETER).
	if len(p) > 0 && strings.Contains(p[0], ";") && strings.Contains(p[0], "(") && strings.HasSuffix(p[0], ",") {
		if len(p) > 1 {
			return p[1]
		}
		return p[0]
	}
	return p[0]
}

func krnEntryZeroNode(entry []Pair) string {
	if len(entry) == 0 {
		return ""
	}
	first := entry[0].Subs
	if len(first) >= 3 {
		want := Subs{first[0], first[1], first[2], Sub{kind: kindInt, intV: 0}}
		for _, p := range entry {
			if p.Subs.key() == want.key() {
				return p.Value
			}
		}
	}
	for _, p := range entry {
		if len(p.Subs) == 4 && p.Subs[3].IsZeroInt() {
			return p.Value
		}
	}
	return ""
}

// decomposeFIA decomposes FileMan DD/data sections per file under Files/, and
// returns the set of consumed subscript keys. Port of the FIA block.
func decomposeFIA(sections map[string]*Build, outDir string) (map[string]bool, error) {
	consumed := map[string]bool{}

	// fia_files: {fnum → name} from the bare ("FIA", fnum) nodes.
	type fiaFile struct {
		fnum Sub
		name string
	}
	var fiaOrder []string
	fiaFiles := map[string]*fiaFile{}
	if sec, ok := sections["FIA"]; ok {
		for _, p := range sec.Pairs() {
			if len(p.Subs) == 2 {
				fk := p.Subs[1].key()
				if _, exists := fiaFiles[fk]; !exists {
					fiaOrder = append(fiaOrder, fk)
				}
				fiaFiles[fk] = &fiaFile{fnum: p.Subs[1], name: p.Value}
			}
		}
	}
	if len(fiaFiles) == 0 {
		return consumed, nil
	}

	filesRoot := filepath.Join(outDir, "Files")
	if err := os.MkdirAll(filesRoot, 0o755); err != nil {
		return nil, err
	}

	isDataSection := func(s string) bool {
		switch s {
		case "DATA", "FRV1", "FRVL", "FRV1K":
			return true
		}
		return false
	}

	for _, fk := range fiaOrder {
		ff := fiaFiles[fk]
		// _sanitize never returns "" (empty → "unnamed"), so Python's
		// `_sanitize(fname) or f"file-{fnum}"` always takes the sanitized name.
		safeName := sanitize(ff.name)
		fileDir := filepath.Join(filesRoot, ff.fnum.display()+"+"+safeName)
		if err := os.MkdirAll(fileDir, 0o755); err != nil {
			return nil, err
		}
		var ddPairs, dataPairs []Pair
		for _, section := range filemanSections {
			sec, ok := sections[section]
			if !ok {
				continue
			}
			for _, p := range sec.Pairs() {
				if !matchesFile(p.Subs, ff.fnum) {
					continue
				}
				if isDataSection(section) {
					dataPairs = append(dataPairs, p)
				} else {
					ddPairs = append(ddPairs, p)
				}
				consumed[p.Subs.key()] = true
			}
		}
		if len(ddPairs) > 0 {
			if err := writeSortedZWR(filepath.Join(fileDir, "DD.zwr"), ddPairs); err != nil {
				return nil, err
			}
			if err := extractDDCode(ddPairs, filepath.Join(fileDir, "DD-code")); err != nil {
				return nil, err
			}
		}
		if len(dataPairs) > 0 {
			if err := writeSortedZWR(filepath.Join(fileDir, "Data.zwr"), dataPairs); err != nil {
				return nil, err
			}
		}
	}
	return consumed, nil
}

// matchesFile reports whether subs belongs to file fnum, scanning positions 1
// and 2 (so both (SEC,fnum,…) and (SEC,"^DIC",fnum,…) shapes match).
func matchesFile(subs Subs, fnum Sub) bool {
	if len(subs) < 2 {
		return false
	}
	if subEqual(subs[1], fnum) {
		return true
	}
	if len(subs) >= 3 && subEqual(subs[2], fnum) {
		return true
	}
	return false
}

// wellKnownFiles maps a FileMan file number to a presentable directory name.
var wellKnownFiles = map[float64]string{
	0.4: "PRINT-TEMPLATE", 0.401: "SORT-TEMPLATE", 0.402: "INPUT-TEMPLATE",
	0.403: "FORM", 0.404: "BLOCK",
	3.7: "DEVICE", 3.8: "MAIL-GROUP", 3.9: "MAIL-MESSAGE",
	9.2: "HELP-FRAME", 9.4: "PACKAGE", 9.6: "KIDS-BUILD", 9.7: "KIDS-INSTALL", 9.8: "ROUTINE",
	19: "OPTION", 19.1: "SECURITY-KEY", 19.2: "OPTION-SCHEDULING",
	100: "ORDER", 101: "PROTOCOL", 101.41: "DIALOG",
	200: "NEW-PERSON", 2: "PATIENT",
	771: "HL7-APPLICATION", 870: "HL-LOGICAL-LINK", 871: "HL-FILE-EVENT", 872: "HL-LOWER-LEVEL-PROTOCOL",
	8989.51: "PARAMETER-DEFINITION", 8989.52: "PARAMETER-TEMPLATE",
	8993: "RPC-BROKER-SUBSCRIBER", 8994: "REMOTE-PROCEDURE",
}

func resolveKernelFileName(fnum Sub) string {
	if v, ok := fnum.numVal(); ok {
		if name, found := wellKnownFiles[v]; found {
			return name
		}
	}
	return "file-" + fnum.display()
}
