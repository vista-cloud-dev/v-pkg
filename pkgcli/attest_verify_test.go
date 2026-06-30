package pkgcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vista-cloud-dev/clikit"
)

// writeLedger emits two chained records to a fresh ledger and returns its path.
func writeLedger(t *testing.T) string {
	t.Helper()
	kidFile := filepath.Join(t.TempDir(), "ZZT.KID")
	flags := attestFlags{}
	if err := emitAttestation(flags, kidFile, newRecord(attestInput{Op: "install", Action: "proceed", Name: "ZZT*1.0*1", Engine: "ydb", Transport: "docker", Status: 3})); err != nil {
		t.Fatalf("emit 1: %v", err)
	}
	if err := emitAttestation(flags, kidFile, newRecord(attestInput{Op: "uninstall", Action: "delete", Name: "ZZT*1.0*1", Engine: "ydb", Transport: "docker"})); err != nil {
		t.Fatalf("emit 2: %v", err)
	}
	return defaultAttestPath(kidFile)
}

func runAttestVerify(t *testing.T, ledger string) error {
	t.Helper()
	cc := clikit.NewContext(&clikit.Globals{Output: "json"}, "attest verify")
	cc.Stdout = &bytes.Buffer{}
	cmd := &attestVerifyCmd{Ledger: ledger}
	return cmd.Run(cc)
}

func TestAttestVerify_PristineLedgerPasses(t *testing.T) {
	if err := runAttestVerify(t, writeLedger(t)); err != nil {
		t.Errorf("pristine ledger failed verify: %v", err)
	}
}

func TestAttestVerify_TamperedRecordRefuses(t *testing.T) {
	ledger := writeLedger(t)
	raw, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	// Flip a recorded field in the first record (status 3 -> 9) WITHOUT re-hashing.
	tampered := bytes.Replace(raw, []byte(`"status":3`), []byte(`"status":9`), 1)
	if bytes.Equal(tampered, raw) {
		t.Fatal("test setup: no status field to tamper")
	}
	if err := os.WriteFile(ledger, tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runAttestVerify(t, ledger); err == nil {
		t.Error("tampered ledger passed verify; want refusal")
	}
}

func TestAttestVerify_MissingLedgerIsEmptyAndClean(t *testing.T) {
	// A non-existent ledger is an empty chain (0 records) — verifies clean.
	if err := runAttestVerify(t, filepath.Join(t.TempDir(), "none.jsonl")); err != nil {
		t.Errorf("empty ledger failed verify: %v", err)
	}
}
