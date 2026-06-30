package kids

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Sidecar integrity (verifiable-safety #3c). install --auto-snapshot writes a
// pre-image sidecar (<kid>.preimage.kids) that uninstall auto-restores. A tampered
// sidecar would silently restore the WRONG routine source. To detect that, the
// snapshot is stamped at capture with a content hash of its routine source, carried
// in a private ("VPKG","HASH") node that — like the foreign-overwrite declaration —
// EnginePairs strips before staging, so it rides in the .KID without polluting KIDS
// filing. restore / uninstall auto-restore recompute the hash and refuse a mismatch.

// hashSub1 is the second subscript of the private content-hash node
// ("VPKG","HASH"). VPKG (foreignSub0) is not a KIDS keyword, and EnginePairs strips
// any ("VPKG",…) node, so it never reaches an engine transport global.
const hashSub1 = "HASH"

// HashPairs returns a deterministic sha256 (hex) over a build's ENGINE-relevant pairs
// — EnginePairs(pairs), i.e. everything except v-pkg's private ("VPKG",…) metadata
// (so the hash node and any foreign-overwrite declaration are excluded, and the hash
// is stable whether or not the input is already stamped). Each pair is rendered as a
// canonical `(<subs>)=<value>` line; lines are sorted so pair order does not matter.
// This is the deterministic identity of the routine source a restore re-applies.
func HashPairs(pairs []Pair) string {
	eng := EnginePairs(pairs)
	lines := make([]string, 0, len(eng))
	for _, p := range eng {
		lines = append(lines, p.Subs.MRef("(")+"="+p.Value)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// StampHash returns pairs with a ("VPKG","HASH") node appended carrying HashPairs of
// the input. Because HashPairs excludes ("VPKG",…) nodes, re-stamping or re-hashing a
// stamped build yields the same value. Called at snapshot capture so the same capture
// always stamps the same hash.
func StampHash(pairs []Pair) []Pair {
	h := HashPairs(pairs)
	return append(pairs, Pair{Subs: Subs{strSub(foreignSub0), strSub(hashSub1)}, Value: h})
}

// VerifySidecarHash checks a parsed pre-image build's stamped content hash against its
// current engine content. present is false when the build carries no ("VPKG","HASH")
// node (an authored back-out, or a snapshot captured before #3c) — the caller then has
// nothing to verify. When present, ok is true iff the stored hash matches HashPairs of
// the build's current pairs; a false ok means the sidecar's routine source was altered
// after capture. stored is the stamped hash (for diagnostics).
func (b *Build) sidecarHash() (string, bool) {
	return b.Get(Subs{strSub(foreignSub0), strSub(hashSub1)})
}

// VerifySidecarHash is the package-level form over a *Build (see sidecarHash).
func VerifySidecarHash(b *Build) (stored string, ok bool, present bool) {
	stored, present = b.sidecarHash()
	if !present {
		return "", false, false
	}
	return stored, HashPairs(b.Pairs()) == stored, true
}
