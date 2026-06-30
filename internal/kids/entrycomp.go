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
	// dataRoot is the storage global the type files into (e.g. "^DIC(19,") — where
	// a content-asserting verify resolves the site IEN via "B" and reads the live
	// 0-node back. volatile lists the 0-node ^-piece indices FileMan rewrites at
	// install (a pointer resolved to a site IEN, a set-of-codes external resolved to
	// its internal code), which content comparison must skip. Empty for types whose
	// 0-node files verbatim.
	dataRoot string
	volatile []int
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
func buildEntryGroups(defs []ParamDef, opts []Option, keys []SecurityKey, protos []Protocol, rpcs []RPC, mgs []MailGroup, lts []ListTemplate, hfs []HelpFrame, h7s []HL7App, hos []HLOApp, lls []LogicalLink) []entryGroup {
	var groups []entryGroup
	if len(lls) > 0 {
		groups = append(groups, entryGroup{logicalLinkEntryType, logicalLinkRecords(lls)})
	}
	if len(hos) > 0 {
		groups = append(groups, entryGroup{hloAppEntryType, hloAppRecords(hos)})
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

// entryTypeByFile maps a file number to its entry type — the registry a
// content-asserting verify uses to recover the storage global + transform mask of
// any shipped KRN record. Every type buildEntryGroups can emit is registered here.
var entryTypeByFile = map[float64]entryType{
	optionFile:       optionEntryType,
	paramDefFile:     paramDefEntryType,
	securityKeyFile:  securityKeyEntryType,
	protocolFile:     protocolEntryType,
	rpcFile:          rpcEntryType,
	mailGroupFile:    mailGroupEntryType,
	listTemplateFile: listTemplateEntryType,
	helpFrameFile:    helpFrameEntryType,
	hl7AppFile:       hl7AppEntryType,
	logicalLinkFile:  logicalLinkEntryType,
	hloAppFile:       hloAppEntryType,
}

// Component is one entry-component TYPE a build ships, for the registry-driven
// generic presence-verify + uninstall-delete path: the file number, its KIDS
// render (the marker-key disambiguator), the authoritative storage global root
// (which yields BOTH the "B" presence index `<DataRoot>"B"` and the ^DIK delete
// ref `<DataRoot>`), the FileMan file name (human label), and the shipped .01
// NAMEs in build order. One Component per registered type the build actually
// ships — the single place verify/uninstall learn "which records of which type by
// which name", so a new type becomes one registry row, not new per-type code.
type Component struct {
	File     float64
	FileStr  string
	DataRoot string
	Label    string
	Names    []string
}

// Components returns one Component per registered entry type (entryTypeByFile)
// the build ships, ordered by file number for deterministic output — the
// registry-driven replacement for the eleven per-type XxxNames() accessors the
// verify/uninstall scripts used to take as positional arguments.
func (b *Build) Components() []Component {
	files := make([]float64, 0, len(entryTypeByFile))
	for f := range entryTypeByFile {
		files = append(files, f)
	}
	sort.Float64s(files)
	var out []Component
	for _, f := range files {
		names := b.entryNames(f)
		if len(names) == 0 {
			continue
		}
		et := entryTypeByFile[f]
		out = append(out, Component{
			File: f, FileStr: formatKIDSFloat(f), DataRoot: et.dataRoot, Label: et.name, Names: names,
		})
	}
	return out
}

// EntryContent is what `v pkg verify` needs to assert a shipped KRN entry record
// is filed CORRECTLY, not merely present: the .01 NAME (the "B" lookup key), the
// data global it files into, the expected stored 0-node, and the 0-node ^-piece
// indices FileMan rewrites at install (skipped in comparison). FileStr is the
// file number rendered for the result marker key.
type EntryContent struct {
	File     float64
	FileStr  string
	Name     string
	DataRoot string
	Zero     string
	Volatile []int
}

// EntryContents returns one EntryContent per shipped KRN entry record (every entry
// type, in build order), read from the top-level "KRN",<file>,<seq>,0) 0-nodes
// with each type's storage global + transform mask attached. The presence probe
// answers "does a record by this name exist"; this is what lets verify answer "is
// the record we shipped the record that got filed."
func (b *Build) EntryContents() []EntryContent {
	var out []EntryContent
	for _, p := range b.Pairs() {
		s := p.Subs
		if len(s) == 4 && s[0].IsStr() && s[0].Str() == "KRN" &&
			s[1].IsNumeric() && s[2].IsInt() && s[3].IsZeroInt() {
			et, ok := entryTypeByFile[subNum(s[1])]
			if !ok {
				continue
			}
			name := p.Value
			if i := strings.IndexByte(name, '^'); i >= 0 {
				name = name[:i]
			}
			out = append(out, EntryContent{
				File: et.number, FileStr: formatKIDSFloat(et.number),
				Name: name, DataRoot: et.dataRoot, Zero: p.Value, Volatile: et.volatile,
			})
		}
	}
	return out
}

// ZeroMatch reports whether a live stored 0-node matches the one a build shipped,
// comparing ^-piece by ^-piece and skipping the volatile indices (the pieces
// FileMan rewrites at install — a pointer resolved to its site IEN, a set-of-codes
// external resolved to internal). Trailing empty pieces equal absent ones (FileMan
// trims them), so a shipped "NAME^^^" matches a stored "NAME". An empty live
// 0-node never matches — it signals the record is absent.
func ZeroMatch(expected, live string, volatile []int) bool {
	if live == "" {
		return false
	}
	skip := make(map[int]bool, len(volatile))
	for _, i := range volatile {
		skip[i] = true
	}
	ep := strings.Split(expected, "^")
	lp := strings.Split(live, "^")
	n := len(ep)
	if len(lp) > n {
		n = len(lp)
	}
	piece := func(ps []string, i int) string {
		if i < len(ps) {
			return ps[i]
		}
		return ""
	}
	for i := 1; i <= n; i++ {
		if skip[i] {
			continue
		}
		if piece(ep, i-1) != piece(lp, i-1) {
			return false
		}
	}
	return true
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

var optionEntryType = entryType{number: optionFile, name: optionFileName, ordTail: optionOrdTail, dataRoot: "^DIC(19,"}

// Option is one #19 OPTION record to ship as a KIDS KRN component (SEND-TO-SITE).
// The build files the option definition; TypeCode is the #19 field 4 (TYPE)
// set-of-codes value (e.g. "R" run routine, "A" action). Storage (ground-truthed
// against the live #19 DD): MENU TEXT 0;2, TYPE 0;4, EXIT ACTION node 15, ENTRY
// ACTION node 20, ROUTINE node 25, UPPERCASE MENU TEXT node "U".
type Option struct {
	Name        string           // #19 .01 NAME (0;1)
	MenuText    string           // #19 field 1 MENU TEXT (0;2)
	TypeCode    string           // #19 field 4 TYPE set-of-codes value
	Routine     string           // #19 field 25 ROUTINE entryref (run-routine type); node 25
	EntryAction string           // #19 field 20 ENTRY ACTION (M code); node 20
	ExitAction  string           // #19 field 15 EXIT ACTION (M code); node 15
	MenuItems   []OptionMenuItem // #19.01 MENU multiple (node 10), optional — menu children
}

// OptionMenuItem is one MENU entry (#19.01 subfile) of a menu option: the child
// OPTION it points to, its synonym, and its display order. The .01 ITEM is a #19
// pointer transported via the KIDS "^" resolver convention (placeholder IEN +
// "^" NAME node), same as PROTOCOL #101.01 — see entrycomp ProtocolItem.
type OptionMenuItem struct {
	Name         string // child OPTION .01 NAME (the #19 pointer target)
	Synonym      string // #19.01 field 2 SYNONYM (0;2)
	DisplayOrder string // #19.01 field 3 DISPLAY ORDER (0;3)
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
		if n := len(o.MenuItems); n > 0 {
			ns := strconv.Itoa(n)
			img = append(img, imageNode{Subs{intSub(10), intSub(0)}, "^19.01IP^" + ns + "^" + ns})
			for i, it := range o.MenuItems {
				seq := int64(i + 1)
				order := it.DisplayOrder
				if order == "" {
					order = strconv.FormatInt(seq, 10)
				}
				// Data node <pointer>^<synonym>^<display order>; pointer is a build-local
				// placeholder the install re-points from the "^" NAME node (#19.01 .01).
				img = append(img,
					imageNode{Subs{intSub(10), intSub(seq), intSub(0)},
						caretJoin(map[int]string{1: strconv.FormatInt(seq, 10), 2: it.Synonym, 3: order})},
					imageNode{Subs{intSub(10), intSub(seq), strSub("^")}, it.Name},
				)
			}
		}
		recs = append(recs, entryRec{name: o.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// OptionNames returns the #19 OPTION component names in build order — what `v pkg
// verify`/`uninstall` probe and back out.
func (b *Build) OptionNames() []string { return b.entryNames(optionFile) }

// --- PARAMETER DEFINITION (#8989.51) — migrated onto the generic core ---------

var paramDefEntryType = entryType{number: paramDefFile, name: paramDefFileName, ordTail: "1;;;;;;;", dataRoot: "^XTV(8989.51,"}

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

var securityKeyEntryType = entryType{number: securityKeyFile, name: securityKeyFileName, ordTail: securityKeyOrdTail, dataRoot: "^DIC(19.1,"}

// SecurityKey is one #19.1 SECURITY KEY record to ship as a KIDS KRN component
// (SEND-TO-SITE). A key is fundamentally just a named token holders are granted;
// the record image is minimal — the .01 NAME alone (stored in ^DIC(19.1,). The
// optional DESCRIPTION (word-processing) field is a follow-up.
type SecurityKey struct {
	Name        string   // #19.1 .01 NAME (0;1)
	Description []string // #19.1 field 1 DESCRIPTION (subfile 19.11 at node 1), optional
}

// securityKeyRecords packs each SecurityKey into the generic entry-record shape:
// the SEND XPDFL flag, a single 0-node carrying the key name, and (optionally) the
// DESCRIPTION word-processing field (subfile 19.11 at node 1).
func securityKeyRecords(keys []SecurityKey) []entryRec {
	recs := make([]entryRec, 0, len(keys))
	for _, k := range keys {
		img := append([]imageNode{{Subs{intSub(0)}, k.Name}}, wpNodes(1, "19.11", k.Description)...)
		recs = append(recs, entryRec{name: k.Name, xpdfl: "0^1", image: img})
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

var protocolEntryType = entryType{number: protocolFile, name: protocolFileName, ordTail: protocolOrdTail, dataRoot: "^ORD(101,"}

// Protocol is one #101 PROTOCOL record to ship as a KIDS KRN component
// (SEND-TO-SITE). Stored in ^ORD(101,; the node skeleton matches OPTION but the
// TYPE codes and data global are #101's own, and there is no uppercase-text xref.
// TypeCode is the #101 field 4 (TYPE) set-of-codes value (A action, X extended
// action, M menu, E event driver, …). The #101.01 ITEM multiple (menu items) and
// the extended menu-actions are a follow-up — this authors a base protocol.
type Protocol struct {
	Name        string         // #101 .01 NAME (0;1)
	ItemText    string         // #101 field 1 ITEM TEXT (0;2)
	TypeCode    string         // #101 field 4 TYPE set-of-codes value
	EntryAction string         // #101 field 20 ENTRY ACTION (M code); node 20
	ExitAction  string         // #101 field 15 EXIT ACTION (M code); node 15
	Items       []ProtocolItem // #101.01 ITEM multiple (node 10), optional — menu children
}

// ProtocolItem is one ITEM (#101.01 subfile) of a menu protocol: the name of the
// child PROTOCOL it points to, and the child's display sequence. The .01 ITEM is a
// #101 pointer; KIDS transports it as the source IEN in the data node PLUS a "^"
// resolver node carrying the child's NAME, which the install re-points at the
// target site.
type ProtocolItem struct {
	Name     string // child PROTOCOL .01 NAME (the #101 pointer target)
	Sequence string // #101.01 field 3 SEQUENCE (0;3)
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
		if n := len(p.Items); n > 0 {
			ns := strconv.Itoa(n)
			img = append(img, imageNode{Subs{intSub(10), intSub(0)}, "^101.01PA^" + ns + "^" + ns})
			for i, it := range p.Items {
				seq := int64(i + 1)
				itemSeq := it.Sequence
				if itemSeq == "" {
					itemSeq = strconv.FormatInt(seq, 10)
				}
				// Data node: <pointer>^^<sequence>^ — the pointer slot carries the
				// build-local seq as a placeholder; the install re-points it from the
				// "^" resolver node (the child's NAME).
				img = append(img,
					imageNode{Subs{intSub(10), intSub(seq), intSub(0)}, strconv.FormatInt(seq, 10) + "^^" + itemSeq + "^"},
					imageNode{Subs{intSub(10), intSub(seq), strSub("^")}, it.Name},
				)
			}
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

var rpcEntryType = entryType{number: rpcFile, name: rpcFileName, ordTail: rpcOrdTail, dataRoot: "^XWB(8994,"}

// RPC is one #8994 REMOTE PROCEDURE record to ship as a KIDS KRN component
// (SEND-TO-SITE). Stored in ^XWB(8994,; the record is a single 0-node carrying the
// three DD-required fields (NAME/.01, ROUTINE/.03, RETURN VALUE TYPE/.04) plus TAG
// (.02, functionally required for the RPC to run). ReturnTypeCode is the #8994
// field .04 set-of-codes value (1 single value, 2 array, 3 word processing, …).
type RPC struct {
	Name           string     // #8994 .01 NAME (0;1)
	Tag            string     // #8994 .02 TAG — the M entry tag (0;2)
	Routine        string     // #8994 .03 ROUTINE — the M routine (0;3)
	ReturnTypeCode string     // #8994 .04 RETURN VALUE TYPE set-of-codes value (0;4)
	InputParams    []RPCParam // #8994.02 INPUT PARAMETER multiple (node 2), optional
}

// RPCParam is one INPUT PARAMETER (#8994.02 subfile) of an RPC: its name, type,
// max length, required flag, sequence, and an optional DESCRIPTION word-processing
// field (subfile 8994.021).
type RPCParam struct {
	Name        string   // #8994.02 .01 INPUT PARAMETER (0;1)
	TypeCode    string   // #8994.02 .02 PARAMETER TYPE set value (0;2): 1 literal/2 list/3 WP/4 reference
	MaxLength   string   // #8994.02 .03 MAXIMUM DATA LENGTH (0;3)
	RequiredVal string   // #8994.02 .04 REQUIRED set value (0;4): 1 yes/0 no
	Sequence    string   // #8994.02 .05 SEQUENCE NUMBER (0;5)
	Description []string // #8994.02 field 1 DESCRIPTION (subfile 8994.021), optional
}

// rpcRecords packs each RPC into the generic entry-record shape: the SEND XPDFL
// flag, a 0-node NAME^TAG^ROUTINE^RETURN VALUE TYPE, and (optionally) the #8994.02
// INPUT PARAMETER multiple at node 2 — a header (^8994.02A^<n>^<n>), one data node
// per param, the param's optional DESCRIPTION WP, and the "B" (param name) and
// "PARAMSEQ" (sequence number) cross-references. The #8994 install is a verbatim
// KRN merge (no re-file routines), so the emitter ships these xrefs itself.
func rpcRecords(rpcs []RPC) []entryRec {
	recs := make([]entryRec, 0, len(rpcs))
	for _, r := range rpcs {
		img := []imageNode{{Subs{intSub(0)}, r.Name + "^" + r.Tag + "^" + r.Routine + "^" + r.ReturnTypeCode}}
		if n := len(r.InputParams); n > 0 {
			ns := strconv.Itoa(n)
			img = append(img, imageNode{Subs{intSub(2), intSub(0)}, "^8994.02A^" + ns + "^" + ns})
			for i, p := range r.InputParams {
				seq := int64(i + 1)
				seqNum := p.Sequence
				if seqNum == "" {
					seqNum = strconv.FormatInt(seq, 10)
				}
				img = append(img,
					imageNode{Subs{intSub(2), intSub(seq), intSub(0)},
						caretJoin(map[int]string{1: p.Name, 2: p.TypeCode, 3: p.MaxLength, 4: p.RequiredVal, 5: seqNum})},
					imageNode{Subs{intSub(2), strSub("B"), strSub(p.Name), intSub(seq)}, ""},
					imageNode{Subs{intSub(2), strSub("PARAMSEQ"), versionSub(seqNum), intSub(seq)}, ""},
				)
				img = append(img, wpNodesAt(Subs{intSub(2), intSub(seq), intSub(1)}, "", p.Description)...)
			}
		}
		recs = append(recs, entryRec{name: r.Name, xpdfl: "0^1", image: img})
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

var mailGroupEntryType = entryType{number: mailGroupFile, name: mailGroupFileName, ordTail: mailGroupOrdTail, dataRoot: "^XMB(3.8,"}

// MailGroup is one #3.8 MAIL GROUP record to ship as a KIDS KRN component
// (SEND-TO-SITE). Stored in ^XMB(3.8,; the record is a single 0-node
// NAME^TYPE^ALLOW-SELF-ENROLLMENT. TypeCode is the #3.8 field 4 (TYPE) set-of-codes
// value (PU public / PR private) — a DD-REQUIRED field, so it always ships.
// AllowSelfEnroll is field 7 (y/n), optional. KIDS ships mail groups MEMBER-less
// (the #3.81 MEMBER multiple points to site-local #200 entries, added on site); the
// word-processing DESCRIPTION (field 3, node 2) is deferred — its header carries a
// volatile last-edited date that would defeat the deterministic-build invariant.
type MailGroup struct {
	Name            string   // #3.8 .01 NAME (0;1)
	TypeCode        string   // #3.8 field 4 TYPE set-of-codes value (0;2): PU / PR
	AllowSelfEnroll string   // #3.8 field 7 ALLOW SELF ENROLLMENT (0;3): y / n / ""
	Description     []string // #3.8 field 3 DESCRIPTION (subfile 3.801 at node 2), optional
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
		img := append([]imageNode{{Subs{intSub(0)}, zero}}, wpNodes(2, "3.801", m.Description)...)
		recs = append(recs, entryRec{name: m.Name, xpdfl: "0^1", image: img})
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

var listTemplateEntryType = entryType{number: listTemplateFile, name: listTemplateFileName, ordTail: listTemplateOrdTail, dataRoot: "^SD(409.61,"}

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

var helpFrameEntryType = entryType{number: helpFrameFile, name: helpFrameFileName, ordTail: helpFrameOrdTail, dataRoot: "^DIC(9.2,"}

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
		img := append([]imageNode{{Subs{intSub(0)}, h.Name + "^" + h.Header}}, wpNodes(1, "", h.Text)...)
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

// hl7App #771 0-node piece 7 is COUNTRY CODE, a #779.004 pointer FileMan resolves
// from the shipped external value ("USA") to its IEN at install — skip it (B.1-i).
var hl7AppEntryType = entryType{number: hl7AppFile, name: hl7AppFileName, ordTail: hl7AppOrdTail, dataRoot: "^HL(771,", volatile: []int{7}}

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

// logicalLink #870 0-node piece 3 is LLP TYPE, a #869.1 pointer FileMan resolves
// from the shipped external value ("TCP") to its IEN at install — skip it (B.1-j).
var logicalLinkEntryType = entryType{number: logicalLinkFile, name: logicalLinkFileName, ordTail: logicalLinkOrdTail, dataRoot: "^HLCS(870,", volatile: []int{3}}

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
	Name        string   // #870 .01 NODE (0;1) — 3..10 chars, no leading punctuation
	LLPType     string   // #870 field 2 LLP TYPE (0;3) — #869.1 pointer, external e.g. "TCP"
	Port        string   // #870 400.02 TCP/IP PORT (400;2)
	ServiceType string   // #870 400.03 TCP/IP SERVICE TYPE (400;3) — C/S/M
	Description []string // #870 field 1 DESCRIPTION (subfile 870.02 at node 3), optional
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
		img = append(img, wpNodes(3, "870.02", l.Description)...)
		recs = append(recs, entryRec{name: l.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// LogicalLinkNames returns the #870 HL LOGICAL LINK component names in build order
// — what `v pkg verify`/`uninstall` probe and back out.
func (b *Build) LogicalLinkNames() []string { return b.entryNames(logicalLinkFile) }

// --- HLO APPLICATION REGISTRY (#779.2) — the eleventh type on the generic core --

const (
	hloAppFile     = 779.2
	hloAppFileName = "HLO APPLICATION REGISTRY"
	// hloAppOrdTail is the national-constant tail of the #779.2 ORD install line
	// (pieces after "<file>;<ord>;"): piece 3 = 1 (this file's data ships with the
	// build), then the SEND action routines (HLOAP gather / HLOE post-install). No
	// delete routine (trailing ";;;"). Lifted verbatim from real exports.
	hloAppOrdTail = "1;;HLOAP^XPDTA1;;HLOE^XPDIA1;;;"
)

var hloAppEntryType = entryType{number: hloAppFile, name: hloAppFileName, ordTail: hloAppOrdTail, dataRoot: "^HLD(779.2,"}

// HLOMsgType is one MESSAGE TYPE ACTIONS entry (#779.21 subfile) of an HLO
// application: which incoming HL7 message type/event the application handles and
// the action routine that processes it. MessageType (.01) and Event (.02) are free
// text and double as cross-reference keys; Version (.06) keys the "D" index when
// present.
type HLOMsgType struct {
	MessageType   string // #779.21 .01 HL7 MESSAGE TYPE (0;1) — e.g. "ORU"
	Event         string // #779.21 .02 HL7 EVENT (0;2) — e.g. "R01"
	ActionTag     string // #779.21 .04 ACTION TAG (0;4)
	ActionRoutine string // #779.21 .05 ACTION ROUTINE (0;5)
	Version       string // #779.21 .06 HL7 VERSION (0;6) — e.g. "2.4"
}

// HLOApp is one #779.2 HLO APPLICATION REGISTRY record to ship as a KIDS KRN
// component — the HL7-Optimized (HLO) counterpart to #771: it registers an
// application and maps the HL7 message types it handles to action routines. Stored
// in ^HLD(779.2,. The record is the 0-node (APPLICATION NAME) plus the #779.21
// MESSAGE TYPE ACTIONS multiple, each entry of which the emitter ships WITH its
// computed cross-references (the "B"/"C"/"D" indices KRN^XPDIK would otherwise
// rebuild) so the export is byte-identical to a native KIDS build.
type HLOApp struct {
	Name         string       // #779.2 .01 APPLICATION NAME (0;1)
	MessageTypes []HLOMsgType // #779.21 MESSAGE TYPE ACTIONS multiple
}

// hloAppRecords packs each HLOApp into the generic entry-record shape: the SEND
// XPDFL flag, the 0-node, and the #779.21 MESSAGE TYPE ACTIONS multiple — a header
// (^779.21I^<last>^<count>), one data node per entry, and the entry's computed
// cross-references. Every entry gets a "B" index on MSG TYPE; the (MSG TYPE, EVENT)
// lookup is keyed by version availability — a versioned entry gets the "D" index
// (MSG TYPE, EVENT, VERSION) and a versionless entry the "C" index (MSG TYPE,
// EVENT). C and D are mutually exclusive (live + corpus proven: the #779.21
// re-index builds exactly one of them). The version subscript stays numeric when
// canonical (2.4) and falls back to a string subscript otherwise (2.5.1).
func hloAppRecords(apps []HLOApp) []entryRec {
	recs := make([]entryRec, 0, len(apps))
	for _, a := range apps {
		img := []imageNode{{Subs{intSub(0)}, a.Name}}
		if n := len(a.MessageTypes); n > 0 {
			ns := strconv.Itoa(n)
			img = append(img, imageNode{Subs{intSub(1), intSub(0)}, "^779.21I^" + ns + "^" + ns})
			for i, mt := range a.MessageTypes {
				seq := int64(i + 1)
				img = append(img,
					imageNode{Subs{intSub(1), intSub(seq), intSub(0)},
						caretJoin(map[int]string{1: mt.MessageType, 2: mt.Event, 4: mt.ActionTag, 5: mt.ActionRoutine, 6: mt.Version})},
					imageNode{Subs{intSub(1), strSub("B"), strSub(mt.MessageType), intSub(seq)}, ""},
				)
				if mt.Version != "" {
					img = append(img, imageNode{Subs{intSub(1), strSub("D"), strSub(mt.MessageType), strSub(mt.Event), versionSub(mt.Version), intSub(seq)}, ""})
				} else {
					img = append(img, imageNode{Subs{intSub(1), strSub("C"), strSub(mt.MessageType), strSub(mt.Event), intSub(seq)}, ""})
				}
			}
		}
		recs = append(recs, entryRec{name: a.Name, xpdfl: "0^1", image: img})
	}
	return recs
}

// HLOAppNames returns the #779.2 HLO APPLICATION REGISTRY component names in build
// order — what `v pkg verify`/`uninstall` probe and back out.
func (b *Build) HLOAppNames() []string { return b.entryNames(hloAppFile) }

// versionSub renders an HL7 version as a subscript: a canonical M number stays
// numeric (unquoted, e.g. 2.4), anything else becomes a string subscript (e.g.
// "2.5.1") — matching how M collates the live #779.21 "D" index.
func versionSub(v string) Sub {
	if f, err := strconv.ParseFloat(v, 64); err == nil && formatKIDSFloat(f) == v {
		return fltSub(f)
	}
	return strSub(v)
}

// wpNodes packs a word-processing field's lines into image nodes hanging off the
// given parent storage node: a DATE-LESS header (^<subfile>^<last>^<count>) plus one
// node per line (<node>,<i>,0)=line). The header omits the install-stamped DATE
// (piece 5 in a native export) so identical input is byte-identical — the
// WP-determinism playbook first proven with HELP FRAME. subfile may be "" (the bare
// ^^<last>^<count> form some WP fields ship). Returns nil for no lines.
func wpNodes(node int64, subfile string, lines []string) []imageNode {
	return wpNodesAt(Subs{intSub(node)}, subfile, lines)
}

// wpNodesAt is wpNodes for a WP field hanging off an arbitrary subscript prefix —
// e.g. a per-row WP inside a multiple (RPC #8994.02 param DESCRIPTION at
// 2,<seq>,1,*). The header sits at <prefix>,0) and each line at <prefix>,<i>,0).
func wpNodesAt(prefix Subs, subfile string, lines []string) []imageNode {
	if len(lines) == 0 {
		return nil
	}
	at := func(tail ...Sub) Subs { return append(append(Subs{}, prefix...), tail...) }
	ns := strconv.Itoa(len(lines))
	out := make([]imageNode, 0, len(lines)+1)
	out = append(out, imageNode{at(intSub(0)), "^" + subfile + "^" + ns + "^" + ns})
	for i, line := range lines {
		out = append(out, imageNode{at(intSub(int64(i+1)), intSub(0)), line})
	}
	return out
}

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
