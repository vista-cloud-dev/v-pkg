package kids

import (
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
// name, and data global root (e.g. "^DIZ(999000,"). Only the DD is created — never
// data (the consumer files its own records through the FileMan DBS API).
type FileDD struct {
	Number     int64
	Name       string
	GlobalRoot string // data global root, e.g. "^DIZ(999000,"
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
		b.Set(dd(intSub(0)), "FIELD^^.01^1") // FIELD^^<highest field#>^<field count>
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

		// --- ^DIC image: the dictionary-of-files registration. DDIN merges the whole
		// ("^DIC",<file>) subtree onto ^DIC, so the file level is a prefix and the
		// file's own node carries it twice (^DIC(<file>,0) ← "^DIC",file,file,0).
		b.Set(Subs{strSub("^DIC"), fnum, fnum, intSub(0)}, f.Name+"^"+num)
		b.Set(Subs{strSub("^DIC"), fnum, fnum, intSub(0), strSub("GL")}, gl)
		b.Set(Subs{strSub("^DIC"), fnum, strSub("B"), strSub(f.Name), fnum}, "")
	}
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
