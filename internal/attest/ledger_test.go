package attest

import (
	"path/filepath"
	"testing"
)

func TestLedger_AppendChainsAndLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.attest.jsonl")

	// First append: genesis (LastHash of a missing file is empty).
	last, err := LastHash(path)
	if err != nil {
		t.Fatalf("LastHash(absent): %v", err)
	}
	if last != "" {
		t.Fatalf("LastHash(absent) = %q, want empty", last)
	}
	r1, _ := Seal(sampleRecord(), last, nil)
	if err := Append(path, r1); err != nil {
		t.Fatalf("append r1: %v", err)
	}

	// Second append: chain onto the recorded last hash.
	last, err = LastHash(path)
	if err != nil {
		t.Fatalf("LastHash: %v", err)
	}
	if last != r1.Hash {
		t.Fatalf("LastHash = %q, want %q", last, r1.Hash)
	}
	second := sampleRecord()
	second.Op = "uninstall"
	r2, _ := Seal(second, last, nil)
	if err := Append(path, r2); err != nil {
		t.Fatalf("append r2: %v", err)
	}

	recs, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("loaded %d records, want 2", len(recs))
	}
	if n, err := VerifyChain(recs); err != nil {
		t.Errorf("loaded chain failed verify at %d: %v", n, err)
	}
}
