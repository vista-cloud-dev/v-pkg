package kids

import (
	"path/filepath"
	"testing"
)

func sidecarPairs() []Pair {
	return MakeBuildPairs(BuildInput{
		InstallName: "ZZT*1.0*1 PREIMAGE", Namespace: "ZZT",
		Routines: []RoutineSrc{{Name: "ZZT", Lines: []string{
			"ZZT ;pre-image ;1.0",
			" ;;1.0;ZZT;;",
			" Q",
		}}},
	})
}

// StampHash appends a private ("VPKG","HASH") node carrying a content hash of the
// engine-relevant pairs, and that node is stripped by EnginePairs (so it never
// reaches the engine — same ride-along pattern as the foreign-overwrite declaration).
func TestStampHash(t *testing.T) {
	stamped := StampHash(sidecarPairs())
	var hashVal string
	var found bool
	for _, p := range stamped {
		if len(p.Subs) == 2 && p.Subs[0].IsStr() && p.Subs[0].Str() == "VPKG" && p.Subs[1].IsStr() && p.Subs[1].Str() == "HASH" {
			hashVal, found = p.Value, true
		}
	}
	if !found {
		t.Fatal("StampHash must add a (\"VPKG\",\"HASH\") node")
	}
	if len(hashVal) != 64 {
		t.Errorf("hash value = %q (len %d), want a 64-char sha256 hex", hashVal, len(hashVal))
	}
	// EnginePairs strips the metadata node — it must never reach KIDS filing.
	for _, p := range EnginePairs(stamped) {
		if len(p.Subs) > 0 && p.Subs[0].IsStr() && p.Subs[0].Str() == "VPKG" {
			t.Error("EnginePairs must strip the (\"VPKG\",…) hash node")
		}
	}
}

// HashPairs is deterministic and EXCLUDES the hash node itself (so a stamped build
// re-hashes to the same value — no chicken-and-egg).
func TestHashPairs_Deterministic(t *testing.T) {
	p := sidecarPairs()
	if HashPairs(p) != HashPairs(sidecarPairs()) {
		t.Fatal("HashPairs is not deterministic across identical inputs")
	}
	// Hashing the stamped pairs equals hashing the unstamped pairs (the hash node is
	// excluded from its own computation).
	if HashPairs(StampHash(p)) != HashPairs(p) {
		t.Error("HashPairs must exclude the (\"VPKG\",\"HASH\") node from its own input")
	}
}

// VerifySidecarHash: a stamped build verifies OK; flipping one routine line makes the
// recomputed hash diverge → mismatch; a build with no stamp reports present=false.
func TestVerifySidecarHash(t *testing.T) {
	stamped := StampHash(sidecarPairs())
	b := newBuild()
	for _, p := range stamped {
		b.Set(p.Subs, p.Value)
	}
	if _, ok, present := VerifySidecarHash(b); !present || !ok {
		t.Errorf("stamped build must verify: present=%v ok=%v", present, ok)
	}

	// Tamper: change a routine source line, keep the old hash.
	b.Set(rtnLineSubs("ZZT", 3), " Q  ; tampered")
	if _, ok, present := VerifySidecarHash(b); !present || ok {
		t.Errorf("tampered build must FAIL verification: present=%v ok=%v", present, ok)
	}

	// No stamp: an authored back-out / pre-3c snapshot has nothing to verify.
	plain := newBuild()
	for _, p := range sidecarPairs() {
		plain.Set(p.Subs, p.Value)
	}
	if _, _, present := VerifySidecarHash(plain); present {
		t.Error("an unstamped build must report present=false")
	}
}

// The stamp survives the WriteKID → ParseKID round-trip: a sidecar written to disk
// and re-parsed still verifies (the hash is computed over content that round-trips).
func TestSidecarHash_RoundTrip(t *testing.T) {
	stamped := StampHash(sidecarPairs())
	path := filepath.Join(t.TempDir(), "pre.kids")
	if err := WriteKID([]string{"ZZT*1.0*1 PREIMAGE"}, map[string][]Pair{"ZZT*1.0*1 PREIMAGE": stamped}, path); err != nil {
		t.Fatal(err)
	}
	k, err := ParseKID(path)
	if err != nil {
		t.Fatal(err)
	}
	b := k.Builds["ZZT*1.0*1 PREIMAGE"]
	if _, ok, present := VerifySidecarHash(b); !present || !ok {
		t.Errorf("round-tripped sidecar must verify: present=%v ok=%v", present, ok)
	}
}
