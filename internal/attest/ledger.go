package attest

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// The ledger is a host-side append-only JSON Lines file (one Record per line),
// kept OUTSIDE the engine an install mutates so it is offline-auditable and not
// itself a target of the install it records. Default location and the --attest
// override are decided by the caller (pkgcli); this file is just the I/O.

// Append writes one record as a JSON line to path, creating the file if absent and
// appending otherwise. The record should already be Sealed (chained + hashed) by
// the caller, which reads LastHash(path) for the prevHash before sealing.
func Append(path string, r Record) error {
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// Load reads every record from a ledger file in order. A missing file is an empty
// ledger (no error). A malformed line fails loudly with its line number — a ledger
// that won't parse is itself evidence of corruption, not something to skip past.
func Load(path string) ([]Record, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []Record
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // records carry per-routine maps; allow a large line
	for line := 1; sc.Scan(); line++ {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("ledger %s line %d: %w", path, line, err)
		}
		out = append(out, r)
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return out, nil
}

// LastHash returns the Hash of the final record in a ledger (the prevHash the next
// Append must chain onto), or "" for a missing or empty ledger (genesis).
func LastHash(path string) (string, error) {
	recs, err := Load(path)
	if err != nil {
		return "", err
	}
	if len(recs) == 0 {
		return "", nil
	}
	return recs[len(recs)-1].Hash, nil
}
