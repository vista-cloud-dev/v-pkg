package pkgcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestContract_Golden is the §4 drift gate: the committed dist/v-contract.json
// must match the contract reflected from the live command tree. Regenerate with
// `make contract` (UPDATE_GOLDEN=1).
func TestContract_Golden(t *testing.T) {
	got, err := json.MarshalIndent(Contract(), "", "  ")
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	got = append(got, '\n')

	golden := filepath.Join("..", "dist", "v-contract.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatalf("mkdir dist: %v", err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run `make contract`): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("dist/v-contract.json drift — run `make contract`\n--- got ---\n%s", got)
	}
}

// TestContract_Invariants checks contract-level facts independent of the golden.
func TestContract_Invariants(t *testing.T) {
	m := Contract()
	if m.Domain != "pkg" {
		t.Errorf("domain = %q, want pkg", m.Domain)
	}
	if m.ContractVersion != ContractVersion {
		t.Errorf("contractVersion = %q, want %q", m.ContractVersion, ContractVersion)
	}
	if len(m.Commands) == 0 {
		t.Error("contract has no commands")
	}
	// The offline verbs must all be present.
	want := map[string]bool{"parse": true, "decompose": true, "assemble": true, "roundtrip": true, "canonicalize": true, "lint": true}
	for _, c := range m.Commands {
		if len(c.Path) == 1 {
			delete(want, c.Path[0])
		}
	}
	if len(want) != 0 {
		t.Errorf("contract missing verbs: %v", want)
	}
}
