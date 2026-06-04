package kids

import (
	"os"
	"path/filepath"
	"sort"
)

// CanonicalPairs returns a build's pairs sorted by the _sort_key collation,
// with routine line-2 values canonicalized — the form round-trip equality is
// checked against. Port of _canonical_pairs.
func CanonicalPairs(build *Build) []Pair {
	out := build.Sorted()
	for i := range out {
		s := out[i].Subs
		if len(s) >= 4 && s[0].IsStr() && s[0].str == "RTN" && numEq(s[2], 2) && numEq(s[3], 0) {
			out[i].Value = CanonicalizeRoutineLine2(out[i].Value)
		}
	}
	return out
}

// RoundtripResult summarizes a round-trip verification.
type RoundtripResult struct {
	File   string `json:"file"`
	Builds int    `json:"builds"`
	Pairs  int    `json:"pairs"`
	OK     bool   `json:"ok"`
	// Diff is populated on failure: the first divergence per mismatched build.
	Diff []RoundtripDiff `json:"diff,omitempty"`
}

// RoundtripDiff is the first divergence found in a build that failed to
// round-trip.
type RoundtripDiff struct {
	Build  string `json:"build"`
	PairsA int    `json:"pairsA"`
	PairsB int    `json:"pairsB"`
	FirstA string `json:"firstA,omitempty"`
	FirstB string `json:"firstB,omitempty"`
}

// pairsEqual compares two canonical pair lists with Python's tuple/value ==
// semantics (numerics equal across int/float; order significant).
func pairsEqual(a, b []Pair) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !subsEqual(a[i].Subs, b[i].Subs) || a[i].Value != b[i].Value {
			return false
		}
	}
	return true
}

func subsEqual(a, b Subs) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !subEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// Roundtrip parses, decomposes, reassembles, and re-parses kidPath, then
// verifies canonical equality. It is the oracle for the G6 gate. Port of
// roundtrip; the guarantee is semantic equality after line-2 canonicalization,
// not byte-identity.
func Roundtrip(kidPath string) (RoundtripResult, error) {
	tmp, err := os.MkdirTemp("", "m-kids-rt-")
	if err != nil {
		return RoundtripResult{}, err
	}
	defer os.RemoveAll(tmp)

	decompDir := filepath.Join(tmp, "decomposed")
	reassembled := filepath.Join(tmp, "reassembled.kid")

	parsed1, err := ParseKID(kidPath)
	if err != nil {
		return RoundtripResult{}, err
	}
	for name, build := range parsed1.Builds {
		dir := filepath.Join(decompDir, PatchDescriptorToDir(name), "KIDComponents")
		if err := DecomposeBuild(build, dir); err != nil {
			return RoundtripResult{}, err
		}
	}

	buildsPairs := map[string][]Pair{}
	for _, name := range parsed1.InstallNames {
		dir := filepath.Join(decompDir, PatchDescriptorToDir(name), "KIDComponents")
		ps, err := AssembleBuild(dir, name)
		if err != nil {
			return RoundtripResult{}, err
		}
		buildsPairs[name] = ps
	}

	if err := WriteKID(parsed1.InstallNames, buildsPairs, reassembled); err != nil {
		return RoundtripResult{}, err
	}
	parsed2, err := ParseKID(reassembled)
	if err != nil {
		return RoundtripResult{}, err
	}

	res := RoundtripResult{File: filepath.Base(kidPath), Builds: len(parsed1.Builds)}
	for _, b := range parsed1.Builds {
		res.Pairs += b.Len()
	}

	canon1 := map[string][]Pair{}
	for name, b := range parsed1.Builds {
		canon1[name] = CanonicalPairs(b)
	}
	canon2 := map[string][]Pair{}
	for name, b := range parsed2.Builds {
		canon2[name] = CanonicalPairs(b)
	}

	ok := len(canon1) == len(canon2)
	if ok {
		for name := range canon1 {
			if !pairsEqual(canon1[name], canon2[name]) {
				ok = false
				break
			}
		}
	}
	res.OK = ok
	if ok {
		return res, nil
	}

	// Build a first-divergence diff per mismatched install (sorted for
	// deterministic output).
	names := append([]string{}, parsed1.InstallNames...)
	sort.Strings(names)
	for _, name := range names {
		a, b := canon1[name], canon2[name]
		if pairsEqual(a, b) {
			continue
		}
		d := RoundtripDiff{Build: name, PairsA: len(a), PairsB: len(b)}
		n := len(a)
		if len(b) < n {
			n = len(b)
		}
		for i := 0; i < n; i++ {
			if !subsEqual(a[i].Subs, b[i].Subs) || a[i].Value != b[i].Value {
				d.FirstA = zwrLine(a[i].Subs, a[i].Value)
				d.FirstB = zwrLine(b[i].Subs, b[i].Value)
				break
			}
		}
		res.Diff = append(res.Diff, d)
	}
	return res, nil
}
