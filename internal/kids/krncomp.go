package kids

import (
	"strconv"
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

// emitInstallHooks writes the install-time routine declarations (B.3): the
// environment-check ("PRE"), pre-install ("INI"), and post-install ("INIT")
// routines. Each is written both as the top-level transport node the install path
// reads (ENV^XPDIL1 reads "PRE"; PKG^XPDIL1/$$NEWCP read "INI"/"INIT") and as the
// #9.6 BLD manifest mirror (the build's self-description / round-trip). Env-check
// is a bare routine name (ENV does D @("^"_name)); pre/post are entryrefs that
// PRE^/POST^XPDIJ1 D @. Emits nothing when a hook is unset, so a hook-free build
// stays byte-identical.
func emitInstallHooks(b *Build, envCheck, preInstall, postInstall string) {
	set := func(field, val string) {
		if val == "" {
			return
		}
		b.Set(Subs{strSub(field)}, val)
		b.Set(Subs{strSub("BLD"), intSub(1), strSub(field)}, val)
	}
	set("PRE", envCheck)
	set("INI", preInstall)
	set("INIT", postInstall)
}
