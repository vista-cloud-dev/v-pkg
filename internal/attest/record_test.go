package attest

import (
	"crypto/ed25519"
	"strings"
	"testing"
)

// sampleRecord returns a representative install record (no chain/hash fields set).
func sampleRecord() Record {
	return Record{
		Schema:    Schema,
		Op:        "install",
		Action:    "snapshot+proceed",
		Name:      "ZZT*1.0*1",
		Class:     "pure-overwrite",
		Engine:    "ydb",
		Transport: "docker",
		Before:    map[string]string{"ZZT1": "B10838", "ZZT2": "absent"},
		After:     map[string]string{"ZZT1": "B20000", "ZZT2": "B30000"},
		Status:    3,
		Exit:      0,
	}
}

func TestSeal_HashIsDeterministic(t *testing.T) {
	a, err := Seal(sampleRecord(), "", nil)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	b, err := Seal(sampleRecord(), "", nil)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if a.Hash == "" {
		t.Fatal("sealed record has empty hash")
	}
	if a.Hash != b.Hash {
		t.Errorf("hash not deterministic: %s != %s", a.Hash, b.Hash)
	}
	if a.PrevHash != "" {
		t.Errorf("genesis prevHash = %q, want empty", a.PrevHash)
	}
}

func TestVerifyRecord_PristinePasses(t *testing.T) {
	r, _ := Seal(sampleRecord(), "", nil)
	if err := VerifyRecord(r); err != nil {
		t.Errorf("pristine record failed verify: %v", err)
	}
}

func TestVerifyRecord_TamperBreaksHash(t *testing.T) {
	r, _ := Seal(sampleRecord(), "", nil)
	r.After["ZZT1"] = "B99999" // edit a recorded field after sealing
	if err := VerifyRecord(r); err == nil {
		t.Error("tampered record passed verify; want hash-mismatch error")
	}
}

func TestVerifyChain_LinksAndDetectsBreak(t *testing.T) {
	r1, _ := Seal(sampleRecord(), "", nil)
	second := sampleRecord()
	second.Op = "uninstall"
	r2, _ := Seal(second, r1.Hash, nil)

	if r2.PrevHash != r1.Hash {
		t.Fatalf("r2.prevHash = %q, want %q", r2.PrevHash, r1.Hash)
	}
	if n, err := VerifyChain([]Record{r1, r2}); err != nil {
		t.Errorf("pristine chain failed verify at %d: %v", n, err)
	}

	// Break the link: tamper r1's hash so r2.PrevHash no longer matches.
	bad := []Record{r1, r2}
	bad[0].Hash = strings.Repeat("0", 64)
	if _, err := VerifyChain(bad); err == nil {
		t.Error("broken chain passed verify; want link error")
	}
}

func TestSign_DetachedSignatureVerifies(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	r, err := Seal(sampleRecord(), "", priv)
	if err != nil {
		t.Fatalf("seal+sign: %v", err)
	}
	if r.Signature == "" || r.PubKey == "" {
		t.Fatal("signed record missing signature/pubkey")
	}
	if err := VerifyRecord(r); err != nil {
		t.Errorf("signed record failed self-verify: %v", err)
	}
	// The recorded pubkey must be the signer's public key.
	if r.PubKey != hexKey(pub) {
		t.Errorf("recorded pubkey %q != signer %q", r.PubKey, hexKey(pub))
	}
	// Flip a byte of the signature → verify must fail.
	bad := r
	if bad.Signature[:2] == "00" {
		bad.Signature = "ff" + bad.Signature[2:]
	} else {
		bad.Signature = "00" + bad.Signature[2:]
	}
	if err := VerifyRecord(bad); err == nil {
		t.Error("record with corrupted signature passed verify")
	}
}

func TestVerifyChainTrusted_RequiresPinnedKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	pub := priv.Public().(ed25519.PublicKey)
	r, _ := Seal(sampleRecord(), "", priv)

	if err := VerifyChainTrusted([]Record{r}, pub); err != nil {
		t.Errorf("signed chain failed trusted verify with correct key: %v", err)
	}
	// A different key must be rejected.
	otherPub, _, _ := ed25519.GenerateKey(nil)
	if err := VerifyChainTrusted([]Record{r}, otherPub); err == nil {
		t.Error("trusted verify accepted a record signed by a different key")
	}
	// An unsigned record must be rejected when a key is pinned.
	unsigned, _ := Seal(sampleRecord(), "", nil)
	if err := VerifyChainTrusted([]Record{unsigned}, pub); err == nil {
		t.Error("trusted verify accepted an unsigned record")
	}
}
