package kids

import (
	"strconv"
	"strings"
)

// This file emits the KIDS transport pairs for two non-routine component kinds:
// XPAR PARAMETER DEFINITIONs (#8989.51, shipped as a KRN component) and Required
// Builds (#9.611). It is the build-side counterpart to KRN^XPDIK (the live
// installer) — the structures here were ground-truthed against a GT.M FOIA VistA
// (the XU*8.0*504 reference build + a live ^XTV(8989.51) record dump) and the
// XPDIK source itself: install loops ^XTMP("XPDI",ifn,"ORD",…) to order files,
// then MERGEs each ^XTMP(…,"KRN",file,seq) record image into the live file
// (name-matched via $$DIC LAYGO) and re-indexes it (IX1^DIK) when the ORD xref
// flag is set.

// paramDefFile is FileMan #8989.51 PARAMETER DEFINITION; entitySubfile is its
// ALLOWABLE ENTITIES (#30) multiple.
const (
	paramDefFile     = 8989.51
	entitySubfile    = "8989.513"
	paramDefFileName = "PARAMETER DEFINITION"
)

// ParamDef is one #8989.51 record to ship: NAME + DISPLAY TEXT, the value data
// type code (#8989.51 field 1.1, e.g. "F" for free text), and the ALLOWABLE
// ENTITIES rows. The build creates the definition only — never a value.
type ParamDef struct {
	Name         string
	DisplayText  string
	DataTypeCode string // #8989.51 1.1 set-of-codes value; "" → "F" (free text)
	Entities     []ParamEntity
}

// ParamEntity is one ALLOWABLE ENTITIES row: the #8989.518 entity IEN the
// parameter may be set at, and its precedence.
type ParamEntity struct {
	EntityIEN  string // #8989.518 IEN (DINUM to file #, e.g. "4.2" = SYS)
	Precedence int    // #8989.513 .01 PRECEDENCE; 0 → 1
}

// ReqBuild is one Required Build (#9.611): the prerequisite build name and the
// action KIDS takes when it is absent (0 warn, 1 remove global, 2 leave global).
type ReqBuild struct {
	Name   string
	Action int
}

// emitParamDefManifest writes the BLD #9.6 KRN component list (#9.67) naming the
// shipped #8989.51 entries — the documentation half of the build (the install
// reads the data + ORD sections, not this).
func emitParamDefManifest(b *Build, defs []ParamDef) {
	if len(defs) == 0 {
		return
	}
	bld := Subs{strSub("BLD"), intSub(1)}
	krn := func(tail ...Sub) Subs { return append(append(Subs{}, bld...), append(Subs{strSub("KRN")}, tail...)...) }
	// KRN multiple header: ^9.67PA^<last file#>^<count of file types>.
	b.Set(krn(intSub(0)), "^9.67PA^"+formatKIDSFloat(paramDefFile)+"^1")
	b.Set(krn(fltSub(paramDefFile), intSub(0)), formatKIDSFloat(paramDefFile))
	b.Set(krn(fltSub(paramDefFile), strSub("NM"), intSub(0)),
		"^9.68A^"+strconv.Itoa(len(defs))+"^"+strconv.Itoa(len(defs)))
	for i, d := range defs {
		seq := int64(i + 1)
		b.Set(krn(fltSub(paramDefFile), strSub("NM"), intSub(seq), intSub(0)), d.Name+"^^0")
		b.Set(krn(fltSub(paramDefFile), strSub("NM"), strSub("B"), strSub(d.Name), intSub(seq)), "")
	}
	b.Set(krn(strSub("B"), fltSub(paramDefFile), fltSub(paramDefFile)), "")
}

// emitParamDefData writes the install-driving ORD section and the top-level KRN
// record image for each PARAMETER DEFINITION — the half KRN^XPDIK actually files.
func emitParamDefData(b *Build, defs []ParamDef) {
	if len(defs) == 0 {
		return
	}
	const ord = 1 // single KRN file type → install order 1
	// ORD: <file#>;<ord>;<xref>;…;<action routines>. xref=1 makes XPDIK re-index
	// each entry (IX1^DIK), which builds the "B" cross-reference for a new entry.
	b.Set(Subs{strSub("ORD"), intSub(ord), fltSub(paramDefFile)},
		strings.Join([]string{formatKIDSFloat(paramDefFile), strconv.Itoa(ord), "1", "", "", "", "", "", "", ""}, ";"))
	b.Set(Subs{strSub("ORD"), intSub(ord), fltSub(paramDefFile), intSub(0)}, paramDefFileName)

	for i, d := range defs {
		seq := int64(i + 1)
		rec := func(tail ...Sub) Subs {
			return append(Subs{strSub("KRN"), fltSub(paramDefFile), intSub(seq)}, tail...)
		}
		dt := d.DataTypeCode
		if dt == "" {
			dt = "F"
		}
		// -1 = XPDFL action (0 = send/add-or-update); killed from the live record
		// after the merge.
		b.Set(rec(intSub(-1)), "0")
		// 0 node: .01 NAME ^ .02 DISPLAY TEXT ^ .03 MULTIPLE VALUED (empty = single).
		b.Set(rec(intSub(0)), d.Name+"^"+d.DisplayText+"^")
		// 1 node: 1.1 VALUE DATA TYPE ^ 1.2 VALUE DOMAIN ^ 1.3 VALUE HELP.
		b.Set(rec(intSub(1)), dt+"^^")
		// 30 multiple: ALLOWABLE ENTITIES (#8989.513).
		if len(d.Entities) > 0 {
			b.Set(rec(intSub(30), intSub(0)),
				"^"+entitySubfile+"I^"+strconv.Itoa(len(d.Entities))+"^"+strconv.Itoa(len(d.Entities)))
			for j, e := range d.Entities {
				esub := int64(j + 1)
				prec := e.Precedence
				if prec == 0 {
					prec = 1
				}
				b.Set(rec(intSub(30), intSub(esub), intSub(0)), strconv.Itoa(prec)+"^"+e.EntityIEN)
				b.Set(rec(intSub(30), strSub("B"), intSub(int64(prec)), intSub(esub)), "")
			}
		}
	}
}

// emitRequiredBuildManifest writes the BLD #9.6 REQB multiple (#9.611) — the
// required-build list KIDS checks at load time.
func emitRequiredBuildManifest(b *Build, reqs []ReqBuild) {
	if len(reqs) == 0 {
		return
	}
	bld := Subs{strSub("BLD"), intSub(1), strSub("REQB")}
	sub := func(tail ...Sub) Subs { return append(append(Subs{}, bld...), tail...) }
	b.Set(sub(intSub(0)), "^9.611^"+strconv.Itoa(len(reqs))+"^"+strconv.Itoa(len(reqs)))
	for i, r := range reqs {
		seq := int64(i + 1)
		b.Set(sub(intSub(seq), intSub(0)), r.Name+"^"+strconv.Itoa(r.Action))
		b.Set(sub(strSub("B"), strSub(r.Name), intSub(seq)), "")
	}
}

// emitMBREQ writes the top-level MBREQ count node (the transport summary of how
// many Required Builds the distribution carries).
func emitMBREQ(b *Build, reqs []ReqBuild) {
	if len(reqs) == 0 {
		return
	}
	b.Set(Subs{strSub("MBREQ")}, strconv.Itoa(len(reqs)))
}

// ParamDefNames returns the names of the #8989.51 PARAMETER DEFINITION components
// in build order, read from the top-level KRN record 0-nodes — what `v pkg
// verify`/`uninstall` use to probe and back out each parameter definition.
func (b *Build) ParamDefNames() []string {
	var names []string
	for _, p := range b.Pairs() {
		s := p.Subs
		if len(s) == 4 && s[0].IsStr() && s[0].Str() == "KRN" &&
			s[1].IsFloat() && s[1].fltV == paramDefFile && s[2].IsInt() && s[3].IsZeroInt() {
			name := p.Value
			if i := strings.IndexByte(name, '^'); i >= 0 {
				name = name[:i]
			}
			names = append(names, name)
		}
	}
	return names
}
