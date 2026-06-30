// Package attest is the install-attestation core (verifiable-safety #4): a
// host-side, append-only, tamper-EVIDENT audit record for every engine-MUTATING
// v-pkg op (install / uninstall / restore). It turns "it installed" into provable
// provenance — a record an independent auditor can replay against the engine.
//
// Two protections, layered (kickoff: "the hash-chain is the floor, the signature
// is the ceiling"):
//   - Hash chain (ALWAYS): each record carries prevHash = the prior record's hash,
//     so the ledger is append-only and any edit to a past record breaks the chain.
//     This is SELF-consistency — like the #3b/#3c checksums it lives in the same
//     file an attacker controls, so it stops accidental/naive tampering, not a
//     determined forger who re-hashes the whole tail.
//   - Detached ed25519 signature (OPT-IN, --sign): each record is signed against an
//     EXTERNAL private key; verify pins the public key. This is the only thing that
//     stops a determined tamperer, because re-signing needs the private key.
//
// The package is ENGINE-NEUTRAL and offline: it has no driver/transport
// dependency, so it is fully unit-testable. Assembly from the live reports and the
// emit chokepoints live above it in pkgcli (which knows KIDS + the engine seam).
package attest

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// Schema is the attestation record schema tag. Bump on an incompatible record
// shape change (a new required field, a changed canonicalization) so an old
// verifier refuses a record it cannot fully check rather than mis-verifying it.
const Schema = "vpkg-attest/1"

// Record is one engine-mutating op's audit record. The change-set fields are a
// superset of the install/uninstall/restore reports they are assembled from; the
// chain/signature fields (PrevHash, Hash, Signature, PubKey) are added by Seal.
//
// Canonicalization (what the Hash and Signature cover) is the JSON encoding of the
// record with Hash and Signature ZEROED — every other field, PubKey and PrevHash
// included, is bound. So the chain link and the signing key cannot be swapped
// without detection, but Hash/Signature are themselves derived and excluded to
// avoid a self-reference.
type Record struct {
	Schema    string `json:"schema"`
	Op        string `json:"op"`              // install | uninstall | restore
	Action    string `json:"action"`          // the sub-action (proceed / snapshot+proceed / restore / partition / …)
	Name      string `json:"name"`            // build install-name
	Class     string `json:"class,omitempty"` // reversibility class
	Engine    string `json:"engine"`          // ydb | iris
	Transport string `json:"transport"`       // local | docker | remote

	// Before/After map each touched routine to its line-2-blind "B" checksum
	// ("B<n>") or "absent". BEFORE is the live pre-image read off the engine;
	// AFTER is the source this op filed. An auditor replays by reading the live
	// engine and confirming it still matches AFTER (no drift since this op).
	Before map[string]string `json:"before,omitempty"`
	After  map[string]string `json:"after,omitempty"`

	Components     []string `json:"components,omitempty"`     // non-routine components touched ("<file>:<name>")
	RequiredBuilds []string `json:"requiredBuilds,omitempty"` // #9.611 REQB prerequisite chain
	Snapshot       string   `json:"snapshot,omitempty"`       // pre-image sidecar ref, if any
	SnapshotHash   string   `json:"snapshotHash,omitempty"`   // #3c HashPairs content hash of the sidecar
	Status         int      `json:"status,omitempty"`         // #9.7 status (3 = Install Completed)
	Verify         string   `json:"verify,omitempty"`         // verify verdict (clean / dirty / drift grade), if checked
	Exit           int      `json:"exit"`                     // process exit code for the op
	Timestamp      string   `json:"timestamp,omitempty"`      // RFC3339 capture time (set by the caller; clock-free core)

	PrevHash  string `json:"prevHash"`            // hash of the prior ledger record ("" = genesis)
	Hash      string `json:"hash"`                // sha256(canonical form) — excluded from the canonical form
	Signature string `json:"signature,omitempty"` // detached ed25519 over the canonical form (hex), when --sign
	PubKey    string `json:"pubkey,omitempty"`    // ed25519 public key (hex) the signature verifies against
}

// canonical returns the deterministic bytes the Hash and Signature cover: the
// record with Hash and Signature zeroed, JSON-encoded. Go encodes struct fields in
// declaration order and map keys sorted, so the encoding is stable for equal
// content — no third-party canonical-JSON dependency needed.
func (r Record) canonical() ([]byte, error) {
	r.Hash = ""
	r.Signature = ""
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// hexKey renders an ed25519 public key as hex (the form stored in PubKey).
func hexKey(pub ed25519.PublicKey) string { return hex.EncodeToString(pub) }

// Seal links a record onto prevHash and computes its Hash; when priv is non-nil it
// also stamps PubKey and a detached ed25519 Signature over the same canonical form.
// PubKey is set BEFORE the canonical bytes are computed, so the signing key is part
// of what the hash and signature bind (it cannot be swapped). Hash and Signature
// are excluded from the canonical form (they are derived from it).
func Seal(r Record, prevHash string, priv ed25519.PrivateKey) (Record, error) {
	if r.Schema == "" {
		r.Schema = Schema
	}
	r.PrevHash = prevHash
	if priv != nil {
		r.PubKey = hexKey(priv.Public().(ed25519.PublicKey))
	}
	canon, err := r.canonical()
	if err != nil {
		return Record{}, err
	}
	sum := sha256.Sum256(canon)
	r.Hash = hex.EncodeToString(sum[:])
	if priv != nil {
		r.Signature = hex.EncodeToString(ed25519.Sign(priv, canon))
	}
	return r, nil
}

// VerifyRecord checks a record's self-consistency: its Hash must equal the sha256
// of its canonical form, and when it carries a Signature, the signature must verify
// against its recorded PubKey. It does NOT check the chain link (see VerifyChain) or
// pin the key to a trust anchor (see VerifyChainTrusted).
func VerifyRecord(r Record) error {
	if r.Schema != Schema {
		return fmt.Errorf("unknown record schema %q (want %q)", r.Schema, Schema)
	}
	canon, err := r.canonical()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(canon)
	if want := hex.EncodeToString(sum[:]); want != r.Hash {
		return fmt.Errorf("record hash mismatch (recorded %s, recomputed %s) — the record was altered", short(r.Hash), short(want))
	}
	if r.Signature != "" {
		if err := verifySignature(r, canon); err != nil {
			return err
		}
	}
	return nil
}

// verifySignature checks r's detached ed25519 signature over canon against its
// recorded PubKey.
func verifySignature(r Record, canon []byte) error {
	pub, err := hex.DecodeString(r.PubKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("record %s has an unusable signing key", short(r.Hash))
	}
	sig, err := hex.DecodeString(r.Signature)
	if err != nil {
		return fmt.Errorf("record %s has a malformed signature", short(r.Hash))
	}
	if !ed25519.Verify(pub, canon, sig) {
		return fmt.Errorf("record %s signature does not verify — the record was altered or forged", short(r.Hash))
	}
	return nil
}

// GenesisPrev is the prevHash of the first record in a ledger (an empty chain link).
const GenesisPrev = ""

// VerifyChain validates an ordered ledger: each record self-verifies (VerifyRecord)
// and its PrevHash equals the prior record's Hash (the first must be GenesisPrev). It
// returns the index of the first failing record (or len(records) on success) with the
// error, so a caller can point at exactly where a ledger was broken.
func VerifyChain(records []Record) (int, error) {
	prev := GenesisPrev
	for i, r := range records {
		if r.PrevHash != prev {
			return i, fmt.Errorf("record %d breaks the chain: prevHash %s does not link to the prior record (%s)", i, short(r.PrevHash), short(prev))
		}
		if err := VerifyRecord(r); err != nil {
			return i, fmt.Errorf("record %d: %w", i, err)
		}
		prev = r.Hash
	}
	return len(records), nil
}

// VerifyChainTrusted is VerifyChain plus a pinned trust anchor: EVERY record must
// carry a signature that verifies against expectedPub. This is the tamper-RESISTANT
// check (the hash chain alone is only tamper-evident) — an unsigned record, or one
// signed by any other key, is rejected.
func VerifyChainTrusted(records []Record, expectedPub ed25519.PublicKey) error {
	want := hexKey(expectedPub)
	if _, err := VerifyChain(records); err != nil {
		return err
	}
	for i, r := range records {
		if r.Signature == "" {
			return fmt.Errorf("record %d is unsigned, but a trusted key was required", i)
		}
		if r.PubKey != want {
			return fmt.Errorf("record %d is signed by an untrusted key (%s, want %s)", i, short(r.PubKey), short(want))
		}
	}
	return nil
}

// short trims a hex digest for human-readable diagnostics ("" stays "(none)").
func short(h string) string {
	if h == "" {
		return "(none)"
	}
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

// ErrNoKey is returned when a signed verify is requested but no key is available.
var ErrNoKey = errors.New("no signing key available")
