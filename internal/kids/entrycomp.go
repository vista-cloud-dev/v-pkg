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
func buildEntryGroups(defs []ParamDef, opts []Option, keys []SecurityKey, protos []Protocol, rpcs []RPC, mgs []MailGroup, lts []ListTemplate, hfs []HelpFrame, h7s []HL7App, lls []LogicalLink) []entryGroup {
	var groups []entryGroup
	if len(lls) > 0 {
		groups = append(groups, entryGroup{logicalLinkEntryType, logicalLinkRecords(lls)})
	}
	if len(mgs) > 0 {
		groups = append(groups, entryGroup{mailGroupEntryType, mailGroupRecords(mgs)})
	}
	if len(hfs) > 0 {
		groups = append(groups, entryGroup{helpFrameEntryType, helpFrameRecords(hfs)})
	}
	if len(h7s) > 0 {
		groups = append(groups, entryGroup{hl7AppEntryType, hl7AppRecords(h7s)})
	}
	if len(lts) > 0 {
		groups = append(groups, entryGroup{listTemplateEntryType, listTemplateRecords(lts)})
	}
	if len(opts) > 0 {
		groups = append(groups, entryGroup{optionEntryType, optionRecords(opts)})
	}
	if len(keys) > 0 {
		groups = append(groups, entryGroup{securityKeyEntryType, securityKeyRecords(keys)})
	}
	if len(protos) > 0 {
		groups = append(groups, entryGroup{protocolEntryType, protocolRecords(protos)})
	}
	if len(rpcs) > 0 {
		groups = append(groups, entryGroup{rpcEntryType, rpcRecords(rpcs)})
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

// --- PROTOCOL (#101) — the fourth type on the generic core --------------------

const (
	protocolFile     = 101
	protocolFileName = "PROTOCOL"
	// protocolOrdTail is the national-constant tail of the #101 ORD install line
	// (pieces after "<file>;<ord>;"): the SEND/DELETE action routines KRN^XPDIK
	// runs to file/relink/delete a PROTOCOL. One form across the corpus.
	protocolOrdTail = ";;PRO^XPDTA;PROF1^XPDIA;PROE1^XPDIA;PROF2^XPDIA;;PRODEL^XPDIA"
)

var protocolEntryType = entryType{number: protocolFile, name: protocolFileName, ordTail: protocolOrdTail}

// Protocol is one #101 PROTOCOL record to ship as a KIDS KRN component
// (SEND-TO-SITE). Stored in ^ORD(101,; the node skeleton matches OPTION but the
// TYPE codes and data global are #101's own, and there is no uppercase-text xref.
// TypeCode is the #101 field 4 (TYPE) set-of-codes value (A action, X extended
// action, M menu, E event driver, …). The #101.01 ITEM multiple (menu items) and
// the extended menu-actions are a follow-up — this authors a base protocol.
type Protocol struct {
	Name        string // #101 .01 NAME (0;1)
	ItemText    string // #101 field 1 ITEM TEXT (0;2)
	TypeCode    string // #101 field 4 TYPE set-of-codes value
	EntryAction string // #101 field 20 ENTRY ACTION (M code); node 20
	ExitAction  string // #101 field 15 EXIT ACTION (M code); node 15
}

// protocolRecords packs each Protocol into the generic entry-record shape: the
// SEND XPDFL flag and the record image (0-node, optional ENTRY/EXIT action nodes).
func protocolRecords(protos []Protocol) []entryRec {
	recs := make([]entryRec, 0, len(protos))
	for _, p := range protos {
		img := []imageNode{
			// 0-node: .01 NAME ^ ITEM TEXT ^ (p3 reserved) ^ TYPE.
			{Subs{intSub(0)}, p.Name + "^" + p.ItemText + "^^" + p.TypeCode},
		}
		if p.ExitAction != "" {
			img = append(img, imageNode{Subs{intSub(15)}, p.ExitAction})
		}
		if p.EntryAction != "" {
			img = append(img, imageNode{Subs{intSub(20)}, p.EntryAction})
		}
		recs = append(recs, entryRec{name: p.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// ProtocolNames returns the #101 PROTOCOL component names in build order — what
// `v pkg verify`/`uninstall` probe and back out.
func (b *Build) ProtocolNames() []string { return b.entryNames(protocolFile) }

// --- REMOTE PROCEDURE / RPC (#8994) — the fifth type on the generic core ------

const (
	rpcFile     = 8994
	rpcFileName = "REMOTE PROCEDURE"
	// rpcOrdTail is the national-constant tail of the #8994 ORD install line
	// (pieces after "<file>;<ord>;"): the xref flag (1 → IX1^DIK rebuilds the "B"
	// xref) plus only a delete action routine (RPCDEL^XPDIA1) — an RPC needs no
	// menu/file relink actions. Dominant form across the corpus.
	rpcOrdTail = "1;;;;;;;RPCDEL^XPDIA1"
)

var rpcEntryType = entryType{number: rpcFile, name: rpcFileName, ordTail: rpcOrdTail}

// RPC is one #8994 REMOTE PROCEDURE record to ship as a KIDS KRN component
// (SEND-TO-SITE). Stored in ^XWB(8994,; the record is a single 0-node carrying the
// three DD-required fields (NAME/.01, ROUTINE/.03, RETURN VALUE TYPE/.04) plus TAG
// (.02, functionally required for the RPC to run). ReturnTypeCode is the #8994
// field .04 set-of-codes value (1 single value, 2 array, 3 word processing, …).
type RPC struct {
	Name           string // #8994 .01 NAME (0;1)
	Tag            string // #8994 .02 TAG — the M entry tag (0;2)
	Routine        string // #8994 .03 ROUTINE — the M routine (0;3)
	ReturnTypeCode string // #8994 .04 RETURN VALUE TYPE set-of-codes value (0;4)
}

// rpcRecords packs each RPC into the generic entry-record shape: the SEND XPDFL
// flag and a single 0-node NAME^TAG^ROUTINE^RETURN VALUE TYPE.
func rpcRecords(rpcs []RPC) []entryRec {
	recs := make([]entryRec, 0, len(rpcs))
	for _, r := range rpcs {
		recs = append(recs, entryRec{
			name:  r.Name,
			xpdfl: "0^1",
			image: []imageNode{{Subs{intSub(0)}, r.Name + "^" + r.Tag + "^" + r.Routine + "^" + r.ReturnTypeCode}},
		})
	}
	return recs
}

// RPCNames returns the #8994 REMOTE PROCEDURE component names in build order —
// what `v pkg verify`/`uninstall` probe and back out.
func (b *Build) RPCNames() []string { return b.entryNames(rpcFile) }

// --- MAIL GROUP (#3.8) — the sixth type on the generic core -------------------

const (
	mailGroupFile     = 3.8
	mailGroupFileName = "MAIL GROUP"
	// mailGroupOrdTail is the national-constant tail of the #3.8 ORD install line
	// (pieces after "<file>;<ord>;"): the SEND/DELETE action routines KRN^XPDIK runs
	// to file/relink/delete a MAIL GROUP. One form across the corpus; note the
	// trailing "(%)" on the delete action (its arg list differs from #19's).
	mailGroupOrdTail = ";;MAILG^XPDTA1;MAILGF1^XPDIA1;MAILGE1^XPDIA1;MAILGF2^XPDIA1;;MAILGDEL^XPDIA1(%)"
)

var mailGroupEntryType = entryType{number: mailGroupFile, name: mailGroupFileName, ordTail: mailGroupOrdTail}

// MailGroup is one #3.8 MAIL GROUP record to ship as a KIDS KRN component
// (SEND-TO-SITE). Stored in ^XMB(3.8,; the record is a single 0-node
// NAME^TYPE^ALLOW-SELF-ENROLLMENT. TypeCode is the #3.8 field 4 (TYPE) set-of-codes
// value (PU public / PR private) — a DD-REQUIRED field, so it always ships.
// AllowSelfEnroll is field 7 (y/n), optional. KIDS ships mail groups MEMBER-less
// (the #3.81 MEMBER multiple points to site-local #200 entries, added on site); the
// word-processing DESCRIPTION (field 3, node 2) is deferred — its header carries a
// volatile last-edited date that would defeat the deterministic-build invariant.
type MailGroup struct {
	Name            string // #3.8 .01 NAME (0;1)
	TypeCode        string // #3.8 field 4 TYPE set-of-codes value (0;2): PU / PR
	AllowSelfEnroll string // #3.8 field 7 ALLOW SELF ENROLLMENT (0;3): y / n / ""
}

// mailGroupRecords packs each MailGroup into the generic entry-record shape: the
// SEND XPDFL flag and a single 0-node NAME^TYPE^ALLOW-SELF-ENROLLMENT. TYPE
// defaults to "PU" (public) since #3.8 field 4 is required; the self-enrollment
// piece is emitted only when set, so a minimal group stays NAME^TYPE.
func mailGroupRecords(mgs []MailGroup) []entryRec {
	recs := make([]entryRec, 0, len(mgs))
	for _, m := range mgs {
		typ := m.TypeCode
		if typ == "" {
			typ = "PU"
		}
		zero := m.Name + "^" + typ
		if m.AllowSelfEnroll != "" {
			zero += "^" + m.AllowSelfEnroll
		}
		recs = append(recs, entryRec{
			name:  m.Name,
			xpdfl: "0^1",
			image: []imageNode{{Subs{intSub(0)}, zero}},
		})
	}
	return recs
}

// MailGroupNames returns the #3.8 MAIL GROUP component names in build order — what
// `v pkg verify`/`uninstall` probe and back out.
func (b *Build) MailGroupNames() []string { return b.entryNames(mailGroupFile) }

// --- LIST TEMPLATE (#409.61) — the seventh type on the generic core -----------

const (
	listTemplateFile     = 409.61
	listTemplateFileName = "LIST TEMPLATE"
	// listTemplateOrdTail is the national-constant tail of the #409.61 ORD install
	// line (pieces after "<file>;<ord>;"): piece 1 = 1 (rebuild the "B" xref), then
	// only an edit + delete action (LME1^XPDIA1 / LMDEL^XPDIA1) — a List Manager
	// template needs no menu/file relink. One form across the corpus.
	listTemplateOrdTail = "1;;;;LME1^XPDIA1;;;LMDEL^XPDIA1"
)

var listTemplateEntryType = entryType{number: listTemplateFile, name: listTemplateFileName, ordTail: listTemplateOrdTail}

// ListTemplate is one #409.61 LIST TEMPLATE (List Manager screen) record to ship as
// a KIDS KRN component (SEND-TO-SITE). Stored in ^SD(409.61,. The record is a fixed
// 14-piece 0-node (screen geometry + a PROTOCOL MENU pointer + the title) plus the
// List Manager callback nodes (HDR/INIT/FNL/HLP M code + the ARRAY global). Unlike
// the FileMan TEMPLATE family (#.4/.402/…), a list template carries NO compiled
// structure — every node is a plain string — so it authors cleanly from a spec.
// Margins/codes are pre-resolved strings (build.go applies the geometry defaults).
type ListTemplate struct {
	Name         string // #409.61 .01 NAME (0;1)
	ScreenTitle  string // .11 SCREEN TITLE (0;11)
	ProtocolMenu string // .1 PROTOCOL MENU (0;10) — #101 action-menu pointer by name
	RightMargin  string // .04 RIGHT MARGIN (0;4)
	TopMargin    string // .05 TOP MARGIN (0;5)
	BottomMargin string // .06 BOTTOM MARGIN (0;6)
	HeaderCode   string // field 100 HEADER CODE — node "HDR" (M code)
	EntryCode    string // field 106 ENTRY CODE — node "INIT" (M code)
	ExitCode     string // field 105 EXIT CODE — node "FNL" (M code)
	HelpCode     string // field 103 HELP CODE — node "HLP" (M code)
	ArrayName    string // field 107 ARRAY NAME — node "ARRAY" (the display global ref)
}

// listTemplateRecords packs each ListTemplate into the generic entry-record shape:
// the SEND XPDFL flag, the fixed 14-piece 0-node, and the callback nodes (emitted
// only when set). The 0-node's set-of-codes pieces are pinned to the dominant corpus
// values: TYPE OF LIST = 1 (PROTOCOL), OK TO TRANSPORT = 1, USE CURSOR CONTROL = 1,
// ALLOWABLE NUMBER OF ACTIONS = 1, AUTOMATIC DEFAULTS = 1.
func listTemplateRecords(lts []ListTemplate) []entryRec {
	recs := make([]entryRec, 0, len(lts))
	for _, lt := range lts {
		zero := strings.Join([]string{
			lt.Name,         // .01 NAME
			"1",             // .02 TYPE OF LIST = PROTOCOL
			"",              // .03 LEFT MARGIN
			lt.RightMargin,  // .04 RIGHT MARGIN
			lt.TopMargin,    // .05 TOP MARGIN
			lt.BottomMargin, // .06 BOTTOM MARGIN
			"1",             // .07 OK TO TRANSPORT?
			"1",             // .08 USE CURSOR CONTROL
			"",              // .09 ENTITY NAME
			lt.ProtocolMenu, // .1 PROTOCOL MENU
			lt.ScreenTitle,  // .11 SCREEN TITLE
			"1",             // .12 ALLOWABLE NUMBER OF ACTIONS
			"",              // .13 DATE RANGE LIMIT
			"1",             // .14 AUTOMATIC DEFAULTS
		}, "^")
		img := []imageNode{{Subs{intSub(0)}, zero}}
		for _, n := range []struct {
			sub, val string
		}{
			{"HDR", lt.HeaderCode},
			{"INIT", lt.EntryCode},
			{"FNL", lt.ExitCode},
			{"HLP", lt.HelpCode},
			{"ARRAY", lt.ArrayName},
		} {
			if n.val != "" {
				img = append(img, imageNode{Subs{strSub(n.sub)}, n.val})
			}
		}
		recs = append(recs, entryRec{name: lt.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// ListTemplateNames returns the #409.61 LIST TEMPLATE component names in build order
// — what `v pkg verify`/`uninstall` probe and back out.
func (b *Build) ListTemplateNames() []string { return b.entryNames(listTemplateFile) }

// --- HELP FRAME (#9.2) — the eighth type on the generic core ------------------

const (
	helpFrameFile     = 9.2
	helpFrameFileName = "HELP FRAME"
	// helpFrameOrdTail is the national-constant tail of the #9.2 ORD install line
	// (pieces after "<file>;<ord>;"): the SEND/DELETE action routines KRN^XPDIK runs
	// to file/relink/delete a HELP FRAME. One form across the corpus.
	helpFrameOrdTail = ";;HELP^XPDTA1;HLPF1^XPDIA1;HLPE1^XPDIA1;HLPF2^XPDIA1;;HLPDEL^XPDIA1"
)

var helpFrameEntryType = entryType{number: helpFrameFile, name: helpFrameFileName, ordTail: helpFrameOrdTail}

// HelpFrame is one #9.2 HELP FRAME record to ship as a KIDS KRN component
// (SEND-TO-SITE). Stored in ^DIC(9.2,. The record is the 0-node (.01 NAME ^ HEADER)
// plus the TEXT word-processing field (#9.2 field 2, subfile 9.21) at node 1 — the
// help content, which is the whole point of the type. The volatile DATE ENTERED
// (0;3) and DATE LAST UPDATED are OMITTED so the build stays deterministic, and the
// WP header is shipped date-less; the RELATED FRAME / INVOKED BY ROUTINE multiples
// are a follow-up.
type HelpFrame struct {
	Name   string   // #9.2 .01 NAME (0;1) — hyphen/space, 3–30 chars
	Header string   // #9.2 field 1 HEADER (0;2) — one-line summary
	Text   []string // #9.2 field 2 TEXT (WP, node 1) — the help body, in line order
}

// helpFrameRecords packs each HelpFrame into the generic entry-record shape: the
// SEND XPDFL flag, the 0-node, and the TEXT word-processing field (header
// ^^<lastSeq>^<count> + one node per line). The WP header is emitted date-less and
// the 0-node carries no DATE ENTERED, so identical input yields a byte-identical
// export (the deterministic-build invariant).
func helpFrameRecords(hfs []HelpFrame) []entryRec {
	recs := make([]entryRec, 0, len(hfs))
	for _, h := range hfs {
		img := []imageNode{
			{Subs{intSub(0)}, h.Name + "^" + h.Header},
		}
		if n := len(h.Text); n > 0 {
			ns := strconv.Itoa(n)
			img = append(img, imageNode{Subs{intSub(1), intSub(0)}, "^^" + ns + "^" + ns})
			for i, line := range h.Text {
				img = append(img, imageNode{Subs{intSub(1), intSub(int64(i + 1)), intSub(0)}, line})
			}
		}
		recs = append(recs, entryRec{name: h.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// HelpFrameNames returns the #9.2 HELP FRAME component names in build order — what
// `v pkg verify`/`uninstall` probe and back out.
func (b *Build) HelpFrameNames() []string { return b.entryNames(helpFrameFile) }

// --- HL7 APPLICATION PARAMETER (#771) — the ninth type on the generic core ----

const (
	hl7AppFile     = 771
	hl7AppFileName = "HL7 APPLICATION PARAMETER"
	// hl7AppOrdTail is the national-constant tail of the #771 ORD install line
	// (pieces after "<file>;<ord>;"): the SEND/DELETE action routines KRN^XPDIK runs
	// to file/relink/delete an HL7 application registration. One form across the
	// corpus; the trailing "(%)" mirrors MAIL GROUP's delete-action arg list.
	hl7AppOrdTail = ";;HLAP^XPDTA1;HLAPF1^XPDIA1;HLAPE1^XPDIA1;HLAPF2^XPDIA1;;HLAPDEL^XPDIA1(%)"
)

var hl7AppEntryType = entryType{number: hl7AppFile, name: hl7AppFileName, ordTail: hl7AppOrdTail}

// HL7App is one #771 HL7 APPLICATION PARAMETER record to ship as a KIDS KRN
// component (SEND-TO-SITE) — the canonical "register an HL7 application" entry.
// Stored in ^HL(771,. The record is a single 0-node:
// NAME ^ 2 ACTIVE/INACTIVE ^ 3 FACILITY NAME ^ ^ ^ ^ 7 COUNTRY CODE. The build
// pins ACTIVE = "a" (a shipped application registration is always active) and the
// country defaults to "USA"; the *HL7 SEGMENT/*HL7 MESSAGE legacy multiples are
// obsolete and never shipped. (The HLO registry #779.2 and logical link #870 — the
// other HL7-family files — are follow-ups; #870 carries site-specific network
// config and is not portably authorable.)
type HL7App struct {
	Name        string // #771 .01 NAME (0;1)
	Facility    string // #771 field 3 FACILITY NAME (0;3)
	CountryCode string // #771 field 7 COUNTRY CODE (0;7) — e.g. "USA"
}

// hl7AppRecords packs each HL7App into the generic entry-record shape: the SEND
// XPDFL flag and the single 7-piece 0-node, with ACTIVE pinned to "a".
func hl7AppRecords(apps []HL7App) []entryRec {
	recs := make([]entryRec, 0, len(apps))
	for _, a := range apps {
		zero := strings.Join([]string{a.Name, "a", a.Facility, "", "", "", a.CountryCode}, "^")
		recs = append(recs, entryRec{
			name:  a.Name,
			xpdfl: "0^1",
			image: []imageNode{{Subs{intSub(0)}, zero}},
		})
	}
	return recs
}

// HL7AppNames returns the #771 HL7 APPLICATION PARAMETER component names in build
// order — what `v pkg verify`/`uninstall` probe and back out.
func (b *Build) HL7AppNames() []string { return b.entryNames(hl7AppFile) }

// --- HL LOGICAL LINK (#870) — the tenth type on the generic core --------------

const (
	logicalLinkFile     = 870
	logicalLinkFileName = "HL LOGICAL LINK"
	// logicalLinkOrdTail is the national-constant tail of the #870 ORD install line
	// (pieces after "<file>;<ord>;"): piece 3 = 1 (this file's data ships with the
	// build), then the SEND/DELETE action routines KRN^XPDIK runs to file/relink/
	// delete a logical link. Lifted verbatim from real exports (e.g. PRC*5.1*167).
	logicalLinkOrdTail = "1;;HLLL^XPDTA1;;HLLLE^XPDIA1;;;HLLLDEL^XPDIA1(%)"
)

var logicalLinkEntryType = entryType{number: logicalLinkFile, name: logicalLinkFileName, ordTail: logicalLinkOrdTail}

// LogicalLink is one #870 HL LOGICAL LINK record to ship as a KIDS KRN component —
// an HL7 communication endpoint. Stored in ^HLCS(870,. The record is the sparse
// 0-node (NODE name ^ ^ LLP TYPE) plus, when a TCP param is set, the 400-node
// (^ PORT ^ SERVICE TYPE). LLP TYPE ships as the external #869.1 value ("TCP") and
// KIDS resolves it to the live IEN at install (the same external-pointer mechanism
// #771's COUNTRY CODE uses).
//
// Deliberately NO DNS DOMAIN / TCP/IP ADDRESS: the #870 install routine (HLLL/
// HLLLE^XPDIA1) re-files the record through FileMan, and those two fields are
// site-specific network config that the install does NOT carry — DNS DOMAIN's
// input transform DNS-resolves the host ($$ADDRESS^XLFNSLK) and drops itself (plus
// the coupled TCP/IP ADDRESS) when it can't, and a bare TCP/IP ADDRESS is dropped
// outright (live-proven on vehu). v-pkg ships only what actually lands — the link
// definition (name, protocol, port, role); the receiving site configures the
// endpoint. (The DESCRIPTION word-processing field #870.02 is a follow-up.)
type LogicalLink struct {
	Name        string // #870 .01 NODE (0;1) — 3..10 chars, no leading punctuation
	LLPType     string // #870 field 2 LLP TYPE (0;3) — #869.1 pointer, external e.g. "TCP"
	Port        string // #870 400.02 TCP/IP PORT (400;2)
	ServiceType string // #870 400.03 TCP/IP SERVICE TYPE (400;3) — C/S/M
}

// logicalLinkRecords packs each LogicalLink into the generic entry-record shape:
// the SEND XPDFL flag, the sparse 0-node, and (when a TCP param is set) the
// 400-node. Trailing empty caret-pieces are trimmed so identical input yields a
// byte-identical export.
func logicalLinkRecords(lls []LogicalLink) []entryRec {
	recs := make([]entryRec, 0, len(lls))
	for _, l := range lls {
		img := []imageNode{{Subs{intSub(0)}, caretJoin(map[int]string{1: l.Name, 3: l.LLPType})}}
		if l.Port != "" || l.ServiceType != "" {
			img = append(img, imageNode{Subs{intSub(400)}, caretJoin(map[int]string{2: l.Port, 3: l.ServiceType})})
		}
		recs = append(recs, entryRec{name: l.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// LogicalLinkNames returns the #870 HL LOGICAL LINK component names in build order
// — what `v pkg verify`/`uninstall` probe and back out.
func (b *Build) LogicalLinkNames() []string { return b.entryNames(logicalLinkFile) }

// caretJoin builds a ^-delimited FileMan node from a sparse 1-based piece map,
// trimming trailing empty pieces so the result is minimal and deterministic.
func caretJoin(pieces map[int]string) string {
	hi := 0
	for p, v := range pieces {
		if v != "" && p > hi {
			hi = p
		}
	}
	out := make([]string, hi)
	for p, v := range pieces {
		if p >= 1 && p <= hi {
			out[p-1] = v
		}
	}
	return strings.Join(out, "^")
}
