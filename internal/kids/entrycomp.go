package kids

import (
	"sort"
	"strconv"
	"strings"
)

// This file is the generic SEND-TO-SITE/DELETE-AT-SITE KIDS entry-component
// emitter (coverage-analysis Track B.1) — the build-side counterpart to
// KRN^XPDIK for the ~20 FileMan entry types that share SEND/DELETE semantics
// (#19 OPTION, #19.1 SECURITY KEY, #101 PROTOCOL, #8994 RPC, the template family,
// …). It generalizes the #8989.51 PARAMETER DEFINITION path (krncomp.go): the
// transport is the same three parts — a #9.6 BLD KRN manifest, an "ORD" install
// line carrying the type's national-constant SEND/DELETE action routines, and the
// top-level "KRN",<file>,<seq> record image that XPDIK MERGEs into the live file
// (name-matched via $$DIC LAYGO) and re-indexes. The first type landed on it is
// OPTION (#19, the largest non-routine share of the corpus). Ground-truthed
// against real WorldVistA KIDS exports + the XPDIK/XPDIA source.

// entryType describes a KIDS KRN entry-component FileMan file: its number, its
// FileMan name (the ORD ,0) label XPDIK matches the live file by), and the
// national-constant tail of its ORD install line — the SEND/DELETE action
// routines (pieces 3+, after "<file>;<ord>;") that XPDIK invokes to file, relink,
// and delete an entry of this type. The tail is a per-type constant lifted
// verbatim from real exports, not synthesized.
type entryType struct {
	number  float64
	name    string
	ordTail string
}

// imageNode is one node of a record image: the subscripts BELOW the
// "KRN",<file>,<seq> prefix, and the value.
type imageNode struct {
	tail Subs
	val  string
}

// entryRec is one record of an entry type to ship: its .01 NAME (the manifest /
// verify / uninstall key), the -1 XPDFL flag value ("0^1" = send/add-or-update,
// "1^1" = delete-at-site), and the record image (the data nodes below
// "KRN",<file>,<seq>, in emit order).
type entryRec struct {
	name  string
	xpdfl string
	image []imageNode
}

// entryGroup is one entry type plus the records of that type a build ships. A
// build's groups are ordered file-number ascending (buildEntryGroups), which
// fixes both the shared manifest header and each type's install order.
type entryGroup struct {
	et   entryType
	recs []entryRec
}

// buildEntryGroups collects every KRN entry type a build ships into one ordered
// list (file-number ascending) — the single place that knows the build's KRN
// types, so the shared "BLD",1,"KRN",0) header and the ORD ordering are computed
// once across all of them. PARAMETER DEFINITION (#8989.51) and OPTION (#19) ride
// the same path; new SEND/DELETE types append here.
func buildEntryGroups(defs []ParamDef, opts []Option, keys []SecurityKey) []entryGroup {
	var groups []entryGroup
	if len(opts) > 0 {
		groups = append(groups, entryGroup{optionEntryType, optionRecords(opts)})
	}
	if len(keys) > 0 {
		groups = append(groups, entryGroup{securityKeyEntryType, securityKeyRecords(keys)})
	}
	if len(defs) > 0 {
		groups = append(groups, entryGroup{paramDefEntryType, paramDefRecords(defs)})
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].et.number < groups[j].et.number })
	return groups
}

// emitEntryManifest writes the BLD #9.6 KRN component list (#9.67) for every entry
// type the build ships — the documentation half (the install reads the data + ORD
// sections, not this). One shared header spans all types; each type then gets its
// own 0-node, NM name multiple, and "B" index. The header's last-IEN piece is the
// MAX file number and the count is the number of types — a deterministic stand-in
// for KIDS' insertion-order "last IEN" (cosmetic: KRN^XPDIK iterates the
// subscripts, not the header).
func emitEntryManifest(b *Build, groups []entryGroup) {
	if len(groups) == 0 {
		return
	}
	bld := Subs{strSub("BLD"), intSub(1)}
	krn := func(tail ...Sub) Subs { return append(append(Subs{}, bld...), append(Subs{strSub("KRN")}, tail...)...) }
	var maxFile float64
	for _, g := range groups {
		if g.et.number > maxFile {
			maxFile = g.et.number
		}
	}
	// KRN multiple header: ^9.67PA^<last file#>^<count of file types>.
	b.Set(krn(intSub(0)), "^9.67PA^"+formatKIDSFloat(maxFile)+"^"+strconv.Itoa(len(groups)))
	for _, g := range groups {
		b.Set(krn(fltSub(g.et.number), intSub(0)), formatKIDSFloat(g.et.number))
		b.Set(krn(fltSub(g.et.number), strSub("NM"), intSub(0)),
			"^9.68A^"+strconv.Itoa(len(g.recs))+"^"+strconv.Itoa(len(g.recs)))
		for i, r := range g.recs {
			seq := int64(i + 1)
			// NM node piece 3 = 0 (SEND); delete-at-site would be 1 (a follow-up).
			b.Set(krn(fltSub(g.et.number), strSub("NM"), intSub(seq), intSub(0)), r.name+"^^0")
			b.Set(krn(fltSub(g.et.number), strSub("NM"), strSub("B"), strSub(r.name), intSub(seq)), "")
		}
		b.Set(krn(strSub("B"), fltSub(g.et.number), fltSub(g.et.number)), "")
	}
}

// emitEntryData writes the install-driving ORD section and the top-level KRN
// record image for every entry type — the half KRN^XPDIK actually files. Each
// type's install order is its 1-based position in the group list; the ORD value
// is <file>;<ord>;<type's action-routine tail>, and each record image is the -1
// XPDFL flag followed by the record's nodes.
func emitEntryData(b *Build, groups []entryGroup) {
	for gi, g := range groups {
		ord := gi + 1
		b.Set(Subs{strSub("ORD"), intSub(int64(ord)), fltSub(g.et.number)},
			formatKIDSFloat(g.et.number)+";"+strconv.Itoa(ord)+";"+g.et.ordTail)
		b.Set(Subs{strSub("ORD"), intSub(int64(ord)), fltSub(g.et.number), intSub(0)}, g.et.name)
		for i, r := range g.recs {
			seq := int64(i + 1)
			rec := func(tail ...Sub) Subs {
				return append(Subs{strSub("KRN"), fltSub(g.et.number), intSub(seq)}, tail...)
			}
			// -1 = XPDFL action; killed from the live record after the merge.
			b.Set(rec(intSub(-1)), r.xpdfl)
			for _, n := range r.image {
				b.Set(rec(n.tail...), n.val)
			}
		}
	}
}

// entryNames returns the shipped .01 NAMEs of one entry type in build order, read
// from the top-level "KRN",<file>,<seq>,0) record 0-nodes (piece 1) — what `v pkg
// verify`/`uninstall` use to probe and back out each entry.
func (b *Build) entryNames(file float64) []string {
	var names []string
	for _, p := range b.Pairs() {
		s := p.Subs
		// The file-number subscript may come back as int OR float depending on the
		// number: a fresh build emits fltSub (8989.51, 19, …) but re-parsing a .KID
		// coerces a decimal-free number like 19 to an int. Compare numerically so an
		// integer-numbered entry file (#19, #101, #8994) is matched on either path.
		if len(s) == 4 && s[0].IsStr() && s[0].Str() == "KRN" &&
			s[1].IsNumeric() && subNum(s[1]) == file && s[2].IsInt() && s[3].IsZeroInt() {
			name := p.Value
			if i := strings.IndexByte(name, '^'); i >= 0 {
				name = name[:i]
			}
			names = append(names, name)
		}
	}
	return names
}

// --- OPTION (#19) — the first type on the generic emitter ---------------------

const (
	optionFile     = 19
	optionFileName = "OPTION"
	// optionOrdTail is the national-constant tail of the #19 ORD install line
	// (pieces after "<file>;<ord>;"): the SEND/DELETE action routines KRN^XPDIK
	// runs to file/relink/delete an OPTION. One form across the WorldVistA corpus.
	optionOrdTail = ";;OPT^XPDTA;OPTF1^XPDIA;OPTE1^XPDIA;OPTF2^XPDIA;;OPTDEL^XPDIA"
)

var optionEntryType = entryType{number: optionFile, name: optionFileName, ordTail: optionOrdTail}

// Option is one #19 OPTION record to ship as a KIDS KRN component (SEND-TO-SITE).
// The build files the option definition; TypeCode is the #19 field 4 (TYPE)
// set-of-codes value (e.g. "R" run routine, "A" action). Storage (ground-truthed
// against the live #19 DD): MENU TEXT 0;2, TYPE 0;4, EXIT ACTION node 15, ENTRY
// ACTION node 20, ROUTINE node 25, UPPERCASE MENU TEXT node "U".
type Option struct {
	Name        string // #19 .01 NAME (0;1)
	MenuText    string // #19 field 1 MENU TEXT (0;2)
	TypeCode    string // #19 field 4 TYPE set-of-codes value
	Routine     string // #19 field 25 ROUTINE entryref (run-routine type); node 25
	EntryAction string // #19 field 20 ENTRY ACTION (M code); node 20
	ExitAction  string // #19 field 15 EXIT ACTION (M code); node 15
}

// optionRecords packs each Option into the generic entry-record shape: the SEND
// XPDFL flag and the record image (0-node, optional action/routine nodes, and the
// "U" uppercase-menu xref). Nodes are emitted only when their field is set, so a
// minimal option stays minimal.
func optionRecords(opts []Option) []entryRec {
	recs := make([]entryRec, 0, len(opts))
	for _, o := range opts {
		img := []imageNode{
			// 0-node: .01 NAME ^ MENU TEXT ^ (p3 reserved) ^ TYPE.
			{Subs{intSub(0)}, o.Name + "^" + o.MenuText + "^^" + o.TypeCode},
		}
		if o.ExitAction != "" {
			img = append(img, imageNode{Subs{intSub(15)}, o.ExitAction})
		}
		if o.EntryAction != "" {
			img = append(img, imageNode{Subs{intSub(20)}, o.EntryAction})
		}
		if o.Routine != "" {
			img = append(img, imageNode{Subs{intSub(25)}, o.Routine})
		}
		if o.MenuText != "" {
			img = append(img, imageNode{Subs{strSub("U")}, strings.ToUpper(o.MenuText)})
		}
		recs = append(recs, entryRec{name: o.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// OptionNames returns the #19 OPTION component names in build order — what `v pkg
// verify`/`uninstall` probe and back out.
func (b *Build) OptionNames() []string { return b.entryNames(optionFile) }

// --- PARAMETER DEFINITION (#8989.51) — migrated onto the generic core ---------

var paramDefEntryType = entryType{number: paramDefFile, name: paramDefFileName, ordTail: "1;;;;;;;"}

// paramDefRecords packs each #8989.51 PARAMETER DEFINITION into the generic
// entry-record shape: the SEND XPDFL flag ("0" — a single piece, unlike OPTION's
// "0^1") and the record image (0-node NAME^DISPLAY^.03-empty, 1-node VALUE DATA
// TYPE, and the ALLOWABLE ENTITIES #30 multiple). Mirrors the live-proven prior
// emitter byte-for-byte; the build creates the definition only, never a value.
func paramDefRecords(defs []ParamDef) []entryRec {
	recs := make([]entryRec, 0, len(defs))
	for _, d := range defs {
		dt := d.DataTypeCode
		if dt == "" {
			dt = "F"
		}
		img := []imageNode{
			// 0 node: .01 NAME ^ .02 DISPLAY TEXT ^ .03 MULTIPLE VALUED (empty = single).
			{Subs{intSub(0)}, d.Name + "^" + d.DisplayText + "^"},
			// 1 node: 1.1 VALUE DATA TYPE ^ 1.2 VALUE DOMAIN ^ 1.3 VALUE HELP.
			{Subs{intSub(1)}, dt + "^^"},
		}
		if len(d.Entities) > 0 {
			img = append(img, imageNode{Subs{intSub(30), intSub(0)},
				"^" + entitySubfile + "I^" + strconv.Itoa(len(d.Entities)) + "^" + strconv.Itoa(len(d.Entities))})
			for j, e := range d.Entities {
				esub := int64(j + 1)
				prec := e.Precedence
				if prec == 0 {
					prec = 1
				}
				img = append(img, imageNode{Subs{intSub(30), intSub(esub), intSub(0)}, strconv.Itoa(prec) + "^" + e.EntityIEN})
				img = append(img, imageNode{Subs{intSub(30), strSub("B"), intSub(int64(prec)), intSub(esub)}, ""})
			}
		}
		recs = append(recs, entryRec{name: d.Name, xpdfl: "0", image: img})
	}
	return recs
}

// ParamDefNames returns the #8989.51 PARAMETER DEFINITION component names in build
// order — what `v pkg verify`/`uninstall` use to probe and back out each definition.
func (b *Build) ParamDefNames() []string { return b.entryNames(paramDefFile) }

// --- SECURITY KEY (#19.1) — the third type on the generic core ----------------

const (
	securityKeyFile     = 19.1
	securityKeyFileName = "SECURITY KEY"
	// securityKeyOrdTail is the national-constant tail of the #19.1 ORD install
	// line (pieces after "<file>;<ord>;"): the SEND/DELETE action routines
	// KRN^XPDIK runs to file/delete a SECURITY KEY. One form across the corpus.
	securityKeyOrdTail = ";;KEY^XPDTA1;KEYF1^XPDIA1;KEYE1^XPDIA1;KEYF2^XPDIA1;;KEYDEL^XPDIA1"
)

var securityKeyEntryType = entryType{number: securityKeyFile, name: securityKeyFileName, ordTail: securityKeyOrdTail}

// SecurityKey is one #19.1 SECURITY KEY record to ship as a KIDS KRN component
// (SEND-TO-SITE). A key is fundamentally just a named token holders are granted;
// the record image is minimal — the .01 NAME alone (stored in ^DIC(19.1,). The
// optional DESCRIPTION (word-processing) field is a follow-up.
type SecurityKey struct {
	Name string // #19.1 .01 NAME (0;1)
}

// securityKeyRecords packs each SecurityKey into the generic entry-record shape:
// the SEND XPDFL flag and a single 0-node carrying the key name.
func securityKeyRecords(keys []SecurityKey) []entryRec {
	recs := make([]entryRec, 0, len(keys))
	for _, k := range keys {
		recs = append(recs, entryRec{
			name:  k.Name,
			xpdfl: "0^1",
			image: []imageNode{{Subs{intSub(0)}, k.Name}},
		})
	}
	return recs
}

// KeyNames returns the #19.1 SECURITY KEY component names in build order — what
// `v pkg verify`/`uninstall` probe and back out.
func (b *Build) KeyNames() []string { return b.entryNames(securityKeyFile) }
