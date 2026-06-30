package pkgcli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/attest"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

func TestRoutineChecksumMaps_BeforeAfter(t *testing.T) {
	// A build that overwrites ZZT1 (captured pre-image) and adds ZZT2 (greenfield).
	b := buildFromKID(t, map[string][]string{
		"ZZT1": {"ZZT1 ;new", "ZZT1 ;;1.0;;", " W 1", " Q", ""},
		"ZZT2": {"ZZT2 ;added", "ZZT2 ;;1.0;;", " Q", ""},
	})
	captured := []kids.RoutineSrc{{Name: "ZZT1", Lines: []string{"ZZT1 ;OLD", "ZZT1 ;;0.9;;", " W 0", " Q", ""}}}
	greenfield := []string{"ZZT2"}

	before, after := routineChecksumMaps(b, captured, greenfield)

	if before["ZZT1"] != kids.BChecksum(captured[0].Lines) {
		t.Errorf("before[ZZT1] = %q, want pre-image checksum", before["ZZT1"])
	}
	if before["ZZT2"] != "absent" {
		t.Errorf("before[ZZT2] = %q, want absent (greenfield)", before["ZZT2"])
	}
	if after["ZZT1"] != kids.BChecksum(b.RoutineSource("ZZT1")) {
		t.Errorf("after[ZZT1] = %q, want shipped-source checksum", after["ZZT1"])
	}
	if after["ZZT2"] != kids.BChecksum(b.RoutineSource("ZZT2")) {
		t.Errorf("after[ZZT2] = %q, want shipped-source checksum", after["ZZT2"])
	}
}

func TestDefaultAttestPath(t *testing.T) {
	if got := defaultAttestPath("/tmp/ZZT-1.0-1.KID"); got != "/tmp/ZZT-1.0-1.attest.jsonl" {
		t.Errorf("defaultAttestPath = %q", got)
	}
	// --attest override wins.
	f := attestFlags{Attest: "/var/log/v-pkg.jsonl"}
	if got := f.ledgerPath("/tmp/x.KID"); got != "/var/log/v-pkg.jsonl" {
		t.Errorf("ledgerPath override = %q", got)
	}
	if got := (attestFlags{}).ledgerPath("/tmp/x.KID"); got != "/tmp/x.attest.jsonl" {
		t.Errorf("ledgerPath default = %q", got)
	}
}

func TestEmitAttestation_AppendsAndChains(t *testing.T) {
	dir := t.TempDir()
	kidFile := filepath.Join(dir, "ZZT.KID")
	flags := attestFlags{} // attestation on by default, no signing

	in1 := attestInput{Op: "install", Action: "proceed", Name: "ZZT*1.0*1", Engine: "ydb", Transport: "docker", Status: 3, Exit: 0}
	if err := emitAttestation(flags, kidFile, newRecord(in1)); err != nil {
		t.Fatalf("emit 1: %v", err)
	}
	in2 := attestInput{Op: "uninstall", Action: "delete", Name: "ZZT*1.0*1", Engine: "ydb", Transport: "docker", Exit: 0}
	if err := emitAttestation(flags, kidFile, newRecord(in2)); err != nil {
		t.Fatalf("emit 2: %v", err)
	}

	recs, err := attest.Load(defaultAttestPath(kidFile))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("ledger has %d records, want 2", len(recs))
	}
	if n, err := attest.VerifyChain(recs); err != nil {
		t.Fatalf("chain invalid at %d: %v", n, err)
	}
	if recs[0].Op != "install" || recs[1].Op != "uninstall" {
		t.Errorf("unexpected op order: %s, %s", recs[0].Op, recs[1].Op)
	}
	if recs[0].Timestamp == "" {
		t.Error("emit did not stamp a timestamp")
	}
}

func TestEmitAttestation_NoAttestSuppresses(t *testing.T) {
	dir := t.TempDir()
	kidFile := filepath.Join(dir, "ZZT.KID")
	if err := emitAttestation(attestFlags{NoAttest: true}, kidFile, newRecord(attestInput{Op: "install"})); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if _, err := os.Stat(defaultAttestPath(kidFile)); !os.IsNotExist(err) {
		t.Error("--no-attest wrote a ledger anyway")
	}
}

// buildFromKID assembles a routine-only build from name->source-lines via the real
// build path, so the assembler is exercised on real RTN nodes.
func buildFromKID(t *testing.T, routines map[string][]string) *kids.Build {
	t.Helper()
	var rs []kids.RoutineSrc
	for name, lines := range routines {
		rs = append(rs, kids.RoutineSrc{Name: name, Lines: lines})
	}
	pairs := kids.MakeBuildPairs(kids.BuildInput{InstallName: "ZZT*1.0*1", Namespace: "ZZT", Routines: rs})
	out := filepath.Join(t.TempDir(), "b.KID")
	if err := kids.WriteKID([]string{"ZZT*1.0*1"}, map[string][]kids.Pair{"ZZT*1.0*1": pairs}, out); err != nil {
		t.Fatalf("write kid: %v", err)
	}
	_, b, err := loadBuild(out)
	if err != nil {
		t.Fatalf("load build: %v", err)
	}
	return b
}
