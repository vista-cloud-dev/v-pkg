package kids

import (
	"sort"
	"strconv"
)

// This file builds a KIDS transport global from a declarative spec (VSL T0a.2,
// coordination plan §7.2) — the inverse of decompose/assemble, which work from
// an existing .KID. It lives in package kids because constructing subscripts
// needs the unexported Sub constructors.

// intSub builds an integer-kind subscript element.
func intSub(n int64) Sub { return Sub{kind: kindInt, intV: n} }

// fltSub builds a decimal (file-number) subscript element, e.g. 8989.51.
func fltSub(f float64) Sub { return Sub{kind: kindFloat, fltV: f} }

// RoutineSrc is one routine's name and source lines.
type RoutineSrc struct {
	Name  string
	Lines []string
}

// BuildInput is the normalized input to MakeBuildPairs. Volatile fields are NOT
// carried (install date/user/real checksums) — the export is byte-identical for
// identical inputs, the deterministic-build invariant (§7.2 #2).
type BuildInput struct {
	InstallName    string         // NAMESPACE*VERSION[*PATCH]
	Namespace      string         // NAMESPACE
	Routines       []RoutineSrc   // routine components, in build order
	ParamDefs      []ParamDef     // #8989.51 PARAMETER DEFINITION KRN components
	Options        []Option       // #19 OPTION KRN components (generic entry emitter, B.1)
	Keys           []SecurityKey  // #19.1 SECURITY KEY KRN components (B.1)
	Protocols      []Protocol     // #101 PROTOCOL KRN components (B.1)
	RPCs           []RPC          // #8994 REMOTE PROCEDURE KRN components (B.1)
	MailGroups     []MailGroup    // #3.8 MAIL GROUP KRN components (B.1)
	ListTemplates  []ListTemplate // #409.61 LIST TEMPLATE KRN components (B.1)
	HelpFrames     []HelpFrame    // #9.2 HELP FRAME KRN components (B.1)
	HL7Apps        []HL7App       // #771 HL7 APPLICATION PARAMETER KRN components (B.1)
	LogicalLinks   []LogicalLink  // #870 HL LOGICAL LINK KRN components (B.1)
	Files          []FileDD       // brand-new FileMan FILE DD components (FIA)
	RequiredBuilds []ReqBuild     // Required Builds (#9.611) — prerequisites
	EnvCheck       string         // environment-check routine (bare name) → top-level "PRE"
	PreInstall     string         // pre-install routine entryref → top-level "INI"
	PostInstall    string         // post-install routine entryref → top-level "INIT"
	Platform       string         // VER node (Kernel^FileMan), default "8.0^22.2"
}

// MakeBuildPairs constructs the ^XPD BUILD pairs for a routine-only KIDS package
// (BLD header + RTN sections + VER), with volatile fields NORMALIZED — the build
// date is 0 and routine checksums are stripped to 0 — so the same input yields a
// byte-identical export. This is the minimal correct shape for a routine package
// (the ZZSKEL throwaway); richer components (files/options/KRN/Required Builds)
// are added as the live install path (T0a.3+) validates them against real KIDS.
func MakeBuildPairs(in BuildInput) []Pair {
	b := newBuild()
	// BLD header: NAME^NAMESPACE^0^DATE — last field is the build date, normalized
	// to 0 (with the type field also 0) so the artifact is diffable + reproducible.
	b.Set(Subs{strSub("BLD"), intSub(1), intSub(0)}, in.InstallName+"^"+in.Namespace+"^0^0")

	// BLD #9.6 manifest for the non-routine components (emitted only when present
	// so a routine-only build stays byte-identical to the live-proven ZZSKEL form).
	// All KRN entry types (PARAMETER DEFINITION, OPTION, …) share one manifest +
	// ORD numbering, computed once over the ordered group list.
	groups := buildEntryGroups(in.ParamDefs, in.Options, in.Keys, in.Protocols, in.RPCs, in.MailGroups, in.ListTemplates, in.HelpFrames, in.HL7Apps, in.LogicalLinks)
	emitEntryManifest(b, groups)
	emitFileManifest(b, in.Files)
	emitRequiredBuildManifest(b, in.RequiredBuilds)
	emitInstallHooks(b, in.EnvCheck, in.PreInstall, in.PostInstall)

	b.Set(Subs{strSub("RTN")}, strconv.Itoa(len(in.Routines)))
	for _, r := range in.Routines {
		// 0^<numlines>^<checksum>^<checksum> — checksums stripped (0) for the
		// normalized artifact; real checksums are computed at install time.
		b.Set(Subs{strSub("RTN"), strSub(r.Name)}, "0^"+strconv.Itoa(len(r.Lines))+"^0^0")
		for i, line := range r.Lines {
			b.Set(Subs{strSub("RTN"), strSub(r.Name), intSub(int64(i + 1)), intSub(0)}, line)
		}
	}

	// Install-order + KRN record data + MBREQ count — what KRN^XPDIK consumes to
	// file the entry components (again, only when there are any).
	emitEntryData(b, groups)
	emitFileData(b, in.Files, fileVersion(in.InstallName), in.Namespace)
	emitMBREQ(b, in.RequiredBuilds)

	plat := in.Platform
	if plat == "" {
		plat = "8.0^22.2"
	}
	b.Set(Subs{strSub("VER")}, plat)
	return b.Pairs()
}

// RoutineNames returns the build's RTN component names in build order — the
// 2-subscript `"RTN",<name>` header pairs (the per-routine line nodes have 4
// subscripts, the RTN count node has 1). `v pkg verify`/`uninstall` use these to
// probe and delete each installed routine.
func (b *Build) RoutineNames() []string {
	var names []string
	for _, p := range b.Pairs() {
		if len(p.Subs) == 2 && p.Subs[0].IsStr() && p.Subs[0].Str() == "RTN" && p.Subs[1].IsStr() {
			names = append(names, p.Subs[1].Str())
		}
	}
	return names
}

// RoutineSource returns the shipped source lines of routine name, in line order —
// the `"RTN",<name>,<n>,0)` pairs collected by ascending n. Empty if the build
// ships no source for that routine. Used by `v pkg verify --drift` to compare the
// shipped patch against the live routine.
func (b *Build) RoutineSource(name string) []string {
	type ln struct {
		n   int64
		val string
	}
	var lines []ln
	for _, p := range b.Pairs() {
		s := p.Subs
		if len(s) == 4 && s[0].IsStr() && s[0].Str() == "RTN" && s[1].IsStr() && s[1].Str() == name &&
			s[2].IsInt() && s[3].IsZeroInt() {
			lines = append(lines, ln{n: s[2].Int(), val: p.Value})
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].n < lines[j].n })
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.val
	}
	return out
}

// RoutineDriftMatch reports whether a shipped routine and a live routine are the
// SAME code, ignoring the volatile second line (the `;;`-version/patch-list line
// KIDS rewrites at install with real checksums/patch history). It canonicalizes
// line 2 of each (CanonicalizeRoutineLine2) before comparing — so a patch that is
// still applied matches, and a later national patch that overwrote it does not.
func RoutineDriftMatch(shipped, live []string) bool {
	if len(shipped) != len(live) {
		return false
	}
	for i := range shipped {
		a, b := shipped[i], live[i]
		if i == 1 { // the 2nd line carries the volatile patch list / checksum
			a, b = CanonicalizeRoutineLine2(a), CanonicalizeRoutineLine2(b)
		}
		if a != b {
			return false
		}
	}
	return true
}
