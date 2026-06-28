package kids

import (
	"sort"
	"strconv"
	"strings"
)

// This file emits the KIDS transport for a brand-new FileMan FILE shipped as its
// data dictionary (VSL M3.T1). It is the build-side counterpart to FIA^XPDIK (the
// live installer) — the structures here were ground-truthed against a GT.M FOIA
// VistA (the real DG_5_3_853 file-bearing build + the live XPDIK source): the
// install loops ^XTMP("XPDI",XPDA,"FIA",file), force-installs the DD for a new
// file, and DDIN^DIFROMS moves the ^DD/^DIC image into the live engine.
//
// The supported shape is the minimal transform-invariant file: a single free-text
// .01 NAME field. Cross-references are rebuilt by FileMan on the target; the DD
// ships only ^DD (attribute dictionary) + ^DIC (dict-of-files) nodes.

// fileMaxNameLen bounds the .01 NAME input transform (1-30 chars), the
// transform-invariant property VSLFS relies on.
const fileMaxNameLen = 30

// FileDD is one brand-new FileMan FILE to ship as a data dictionary: its number,
// name, data global root (e.g. "^DIZ(999000,"), and zero or more fields beyond the
// implicit free-text .01 NAME. Only the DD is created — never data (the consumer
// files its own records through the FileMan DBS API).
type FileDD struct {
	Number     int64
	Name       string
	GlobalRoot string      // data global root, e.g. "^DIZ(999000,"
	Fields     []FileField // additional fields beyond the .01 NAME (empty = single-.01 file)
}

// Field type names — the five scalar shapes a multi-field DD emitter grounds
// against real KIDS exports (see filecomp_test / coverage-analysis §8). They map
// to the ^DD definition node's piece-2 type spec.
const (
	FieldFreeText = "free text"
	FieldNumeric  = "numeric"
	FieldDate     = "date"
	FieldSet      = "set of codes"
	FieldPointer  = "pointer"
)

// SetCode is one internal:external pair of a set-of-codes field (^DD piece 3 is
// the ";"-terminated list of these).
type SetCode struct {
	Internal string
	External string
}

// FileField is one FileMan field beyond the implicit .01 NAME, shipped in a
// brand-new file's DD. Node/Piece are the storage location ("<Node>;<Piece>", the
// ^DD definition node's piece 4); Type selects the piece-2 grammar and the input
// transform. The .01 is always a free-text NAME emitted separately; FileField
// covers fields numbered above it.
type FileField struct {
	Number   float64 // field number (> .01)
	Label    string  // field LABEL (^DD piece 1)
	Type     string  // one of the Field* constants
	Node     int     // storage node (the N in "N;piece")
	Piece    int     // storage piece (the P in "node;P")
	Required bool    // R attribute (RF / RS / RP… prefix)
	Help     string  // optional reader help → ,<field#>,3) node

	MaxLen int // free text: input-transform length cap

	Width    int      // numeric: NJ<width>,<decimals> print spec
	Decimals int      // numeric: decimal places (0 = integer)
	Min      *float64 // numeric: lower bound (nil = unbounded)
	Max      *float64 // numeric: upper bound (nil = unbounded)

	HasTime bool // date: allow a time component (%DT "ET" vs "E")

	Codes []SetCode // set of codes: the value list, in order

	PointTo   float64 // pointer: pointed-to file number → P<file>'
	PointRoot string  // pointer: pointed-to global root (^DD piece 3)
}

// emitFileManifest writes the BLD #9.6 FILE component list (#9.64) naming the
// shipped files — the build's self-description (the install reads the FIA + ^DD
// sections, not this), mirroring emitParamDefManifest.
func emitFileManifest(b *Build, files []FileDD) {
	if len(files) == 0 {
		return
	}
	var maxFile int64
	for _, f := range files {
		if f.Number > maxFile {
			maxFile = f.Number
		}
	}
	bld := Subs{strSub("BLD"), intSub(1), intSub(4)}
	sub := func(tail ...Sub) Subs { return append(append(Subs{}, bld...), tail...) }
	// FILE multiple header: ^9.64PA^<highest file#>^<count>.
	b.Set(sub(intSub(0)), "^9.64PA^"+strconv.FormatInt(maxFile, 10)+"^"+strconv.Itoa(len(files)))
	for _, f := range files {
		b.Set(sub(intSub(f.Number), intSub(0)), strconv.FormatInt(f.Number, 10))
		// #9.64 field 222 (SEND FULL/PARTIAL DD …) — the same options string as the
		// FIA node; "y^n^p^^^^n^^n" = send DD, no security, no data.
		b.Set(sub(intSub(f.Number), intSub(222)), fileSendOpts)
		b.Set(sub(strSub("B"), intSub(f.Number), intSub(f.Number)), "")
	}
}

// fileSendOpts is the DD-only send-options string (FIA …,0,1 and BLD #9.64 222):
// piece 1 "y" = install/update the DD; piece 2 "n" = no security code; piece 3
// "f" = FULL definition (DIFROMS2 reads piece 3: "f"=full new-file DD, "p"=partial
// field update — a new file MUST be "f" or DDIN errors "Partial DD/File does not
// exist"); piece 7 "n" = do not send data.
const fileSendOpts = "y^n^f^^^^n^^n"

// emitFileData writes the FIA section + the ^DD/^DIC image for each file — the
// half FIA^XPDIK actually installs. version/namespace stamp the FIA VR node.
func emitFileData(b *Build, files []FileDD, version, namespace string) {
	if len(files) == 0 {
		return
	}
	for _, f := range files {
		fnum := intSub(f.Number)
		num := strconv.FormatInt(f.Number, 10)
		gl := f.GlobalRoot

		// --- FIA section: the file-information array FIA^XPDIK loops. ---
		b.Set(Subs{strSub("FIA"), fnum}, f.Name)
		b.Set(Subs{strSub("FIA"), fnum, intSub(0)}, gl)
		b.Set(Subs{strSub("FIA"), fnum, intSub(0), intSub(0)}, num+"I") // DIFROM internal-format tag
		b.Set(Subs{strSub("FIA"), fnum, intSub(0), intSub(1)}, fileSendOpts)
		b.Set(Subs{strSub("FIA"), fnum, intSub(0), strSub("VR")}, version+"^"+namespace)

		// --- ^DD image: the attribute dictionary for a single free-text .01 NAME. ---
		// DDIN^DIFROMS reads it as ("^DD",<file>,<ddfile>,…) — the file number is
		// DOUBLED (DIFROMS2 loops DIFRD=$O("^DD",file,DIFRD); a single level would
		// land DIFRD=0 and Q:DIFRD'>0 skip the whole DD). For a top file the inner
		// dd-file number equals the file number.
		dd := func(tail ...Sub) Subs { return append(Subs{strSub("^DD"), fnum, fnum}, tail...) }
		// Header: FIELD^^<highest field#>^<field count>. The .01 NAME is always
		// present, plus any declared fields (a single-.01 file → "FIELD^^.01^1").
		hi := 0.01
		for _, fld := range f.Fields {
			if fld.Number > hi {
				hi = fld.Number
			}
		}
		b.Set(dd(intSub(0)), "FIELD^^"+mNum(hi)+"^"+strconv.Itoa(1+len(f.Fields)))
		// Register the "B" index so the FileMan DBS filer (UPDATE^DIE) fires it.
		b.Set(dd(intSub(0), strSub("IX"), strSub("B"), fnum, fltSub(0.01)), "")
		b.Set(dd(intSub(0), strSub("NM"), strSub(f.Name)), "")
		b.Set(dd(fltSub(0.01), intSub(0)), "NAME^RF^^0;1^K:$L(X)>"+strconv.Itoa(fileMaxNameLen)+"!($L(X)<1) X")
		b.Set(dd(fltSub(0.01), intSub(1), intSub(0)), "^.1")
		b.Set(dd(fltSub(0.01), intSub(1), intSub(1), intSub(0)), num+"^B")
		b.Set(dd(fltSub(0.01), intSub(1), intSub(1), intSub(1)), "S "+gl+`"B",$E(X,1,`+strconv.Itoa(fileMaxNameLen)+`),DA)=""`)
		b.Set(dd(fltSub(0.01), intSub(1), intSub(1), intSub(2)), "K "+gl+`"B",$E(X,1,`+strconv.Itoa(fileMaxNameLen)+`),DA)`)
		b.Set(dd(fltSub(0.01), intSub(3)), "Answer must be 1-"+strconv.Itoa(fileMaxNameLen)+" characters in length.")
		b.Set(dd(strSub("B"), strSub("NAME"), fltSub(0.01)), "")
		b.Set(dd(strSub("GL"), strSub("0;1"), intSub(1), fltSub(0.01)), "")

		// --- additional fields (beyond .01), in field-number order for a
		// deterministic export. Each emits its ,<field#>,0) definition node and an
		// optional ,<field#>,3) help node — matching real exports, which carry the
		// storage location inline in the definition (no separate "GL" map node).
		fields := append([]FileField(nil), f.Fields...)
		sort.Slice(fields, func(i, j int) bool { return fields[i].Number < fields[j].Number })
		for _, fld := range fields {
			fsub := fltSub(fld.Number)
			b.Set(dd(fsub, intSub(0)), fieldDef(fld))
			if fld.Help != "" {
				b.Set(dd(fsub, intSub(3)), fld.Help)
			}
		}

		// --- ^DIC image: the dictionary-of-files registration. DDIN merges the whole
		// ("^DIC",<file>) subtree onto ^DIC, so the file level is a prefix and the
		// file's own node carries it twice (^DIC(<file>,0) ← "^DIC",file,file,0).
		b.Set(Subs{strSub("^DIC"), fnum, fnum, intSub(0)}, f.Name+"^"+num)
		b.Set(Subs{strSub("^DIC"), fnum, fnum, intSub(0), strSub("GL")}, gl)
		b.Set(Subs{strSub("^DIC"), fnum, strSub("B"), strSub(f.Name), fnum}, "")
	}
}

// fieldDef builds the 5-piece ^DD definition node value for one field:
// LABEL ^ TYPE-spec ^ (set-list|pointer-root) ^ node;piece ^ input-transform.
// Type-spec and transform follow the grounded grammar (coverage-analysis §8);
// a Required field prefixes the type letter with "R" (RF / RS / RP…).
func fieldDef(f FileField) string {
	storage := strconv.Itoa(f.Node) + ";" + strconv.Itoa(f.Piece)
	var typ, p3, xform string
	switch f.Type {
	case FieldNumeric:
		typ = "N"
		if f.Width > 0 {
			typ += "J" + strconv.Itoa(f.Width) + "," + strconv.Itoa(f.Decimals)
		}
		xform = "K:+X'=X"
		if f.Max != nil {
			xform += "!(X>" + mNum(*f.Max) + ")"
		}
		if f.Min != nil {
			xform += "!(X<" + mNum(*f.Min) + ")"
		}
		xform += `!(X?.E1"."` + strconv.Itoa(f.Decimals+1) + `.N) X`
	case FieldDate:
		typ = "D"
		flags := "E"
		if f.HasTime {
			flags = "ET"
		}
		xform = `S %DT="` + flags + `" D ^%DT S X=Y K:Y<1 X`
	case FieldSet:
		typ = "S"
		var b strings.Builder
		for _, c := range f.Codes {
			b.WriteString(c.Internal + ":" + c.External + ";")
		}
		p3 = b.String()
		xform = "Q"
	case FieldPointer:
		typ = "P" + mNum(f.PointTo) + "'"
		p3 = f.PointRoot
		xform = "Q"
	default: // FieldFreeText
		typ = "F"
		maxLen := f.MaxLen
		if maxLen <= 0 {
			maxLen = fileMaxNameLen
		}
		xform = "K:$L(X)>" + strconv.Itoa(maxLen) + "!($L(X)<1) X"
	}
	if f.Required {
		typ = "R" + typ
	}
	return f.Label + "^" + typ + "^" + p3 + "^" + storage + "^" + xform
}

// mNum renders a FileMan field/file number the way M canonicalizes a numeric
// literal inside a value string: whole numbers plain ("5"), fractions with the
// leading zero stripped (".01"). Subscripts keep their "0.01" rendering via
// formatKIDSFloat — this is only for value strings (the ^DD header, pointer type
// spec, numeric range).
func mNum(f float64) string {
	s := formatKIDSFloat(f)
	if strings.HasPrefix(s, "0.") {
		s = s[1:]
	}
	return s
}

// FileNumbers returns the FileMan file numbers shipped by the build, read from the
// bare ("FIA",<file>) section nodes — what `v pkg verify`/`uninstall` use to probe
// and back out each installed file.
func (b *Build) FileNumbers() []int64 {
	var nums []int64
	for _, p := range b.Pairs() {
		s := p.Subs
		if len(s) == 2 && s[0].IsStr() && s[0].Str() == "FIA" && s[1].IsNumeric() {
			if v, ok := s[1].numVal(); ok {
				nums = append(nums, int64(v))
			}
		}
	}
	return nums
}

// fileVersion extracts the VERSION piece of an install name (NS*VER[*PATCH]) for
// the FIA VR node; it defaults to "1.0" when the name is malformed.
func fileVersion(installName string) string {
	parts := strings.Split(installName, "*")
	if len(parts) >= 2 && parts[1] != "" {
		return parts[1]
	}
	return "1.0"
}
