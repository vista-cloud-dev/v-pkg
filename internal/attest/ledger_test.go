package attest

import (
	"path/filepath"
	"strconv"
	"sync"
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

// TestWithLedgerLock_ConcurrentAppendsStayChained proves the lock serializes
// read-tip→seal→append: 30 goroutines appending concurrently to one ledger must
// produce a VALID chain of exactly 30 records. Without the lock they would read the
// same tip, seal with the same prevHash, and fork the chain (VerifyChain would break).
func TestWithLedgerLock_ConcurrentAppendsStayChained(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.attest.jsonl")
	const n = 30
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- WithLedgerLock(path, func() error {
				last, err := LastHash(path)
				if err != nil {
					return err
				}
				r := sampleRecord()
				r.Name = "B*1.0*" + strconv.Itoa(i)
				sealed, err := Seal(r, last, nil)
				if err != nil {
					return err
				}
				return Append(path, sealed)
			})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent append: %v", err)
		}
	}
	recs, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(recs) != n {
		t.Fatalf("got %d records, want %d (a lost/forked append)", len(recs), n)
	}
	if at, err := VerifyChain(recs); err != nil {
		t.Errorf("concurrent ledger chain invalid at %d: %v", at, err)
	}
}
