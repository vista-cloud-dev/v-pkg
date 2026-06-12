package kids

import "strconv"

// This file builds a KIDS transport global from a declarative spec (VSL T0a.2,
// coordination plan §7.2) — the inverse of decompose/assemble, which work from
// an existing .KID. It lives in package kids because constructing subscripts
// needs the unexported Sub constructors.

// intSub builds an integer-kind subscript element.
func intSub(n int64) Sub { return Sub{kind: kindInt, intV: n} }

// RoutineSrc is one routine's name and source lines.
type RoutineSrc struct {
	Name  string
	Lines []string
}

// BuildInput is the normalized input to MakeBuildPairs. Volatile fields are NOT
// carried (install date/user/real checksums) — the export is byte-identical for
// identical inputs, the deterministic-build invariant (§7.2 #2).
type BuildInput struct {
	InstallName string       // NAMESPACE*VERSION[*PATCH]
	Namespace   string       // NAMESPACE
	Routines    []RoutineSrc // routine components, in build order
	Platform    string       // VER node (Kernel^FileMan), default "8.0^22.2"
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

	b.Set(Subs{strSub("RTN")}, strconv.Itoa(len(in.Routines)))
	for _, r := range in.Routines {
		// 0^<numlines>^<checksum>^<checksum> — checksums stripped (0) for the
		// normalized artifact; real checksums are computed at install time.
		b.Set(Subs{strSub("RTN"), strSub(r.Name)}, "0^"+strconv.Itoa(len(r.Lines))+"^0^0")
		for i, line := range r.Lines {
			b.Set(Subs{strSub("RTN"), strSub(r.Name), intSub(int64(i + 1)), intSub(0)}, line)
		}
	}

	plat := in.Platform
	if plat == "" {
		plat = "8.0^22.2"
	}
	b.Set(Subs{strSub("VER")}, plat)
	return b.Pairs()
}
