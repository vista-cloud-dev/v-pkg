package pkgcli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vista-cloud-dev/clikit"
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

func TestInstallTimeOverwrites_FromLedger(t *testing.T) {
	dir := t.TempDir()
	kid := filepath.Join(dir, "P.KID")
	// Install record: overwrote X (pre-existed → checksum), added Y (greenfield → absent).
	rec := newRecord(attestInput{Op: "install", Name: "P*1.0*1",
		Before: map[string]string{"X": "B123", "Y": "absent"},
		After:  map[string]string{"X": "B999", "Y": "B888"}})
	if err := emitAttestation(attestFlags{}, kid, rec); err != nil {
		t.Fatal(err)
	}
	got := installTimeOverwrites(attestFlags{}, kid, "P*1.0*1")
	if len(got) != 1 || got[0] != "X" {
		t.Errorf("installTimeOverwrites = %v, want [X] (only the pre-existing routine)", got)
	}
	if n := installTimeOverwrites(attestFlags{}, kid, "OTHER*1.0*1"); len(n) != 0 {
		t.Errorf("a different build name must see no overwrites, got %v", n)
	}
	if n := installTimeOverwrites(attestFlags{}, filepath.Join(dir, "none.KID"), "P*1.0*1"); len(n) != 0 {
		t.Errorf("a missing ledger must degrade to empty, got %v", n)
	}
}

func TestMergeForeign_Union(t *testing.T) {
	got := mergeForeign([]string{"A", "B"}, []string{"B", "C"})
	if len(got) != 3 || got[0] != "A" || got[1] != "B" || got[2] != "C" {
		t.Errorf("mergeForeign = %v, want [A B C]", got)
	}
}

// TestUninstall_RefusesUndeclaredOverwrite is the #3 end-to-end proof (offline — the
// refuse is decided before any engine call): a build that did NOT self-declare a
// foreign overwrite, but whose attestation ledger records that a routine pre-existed
// at install (an --allow-overwrite of a national routine), must REFUSE a bare uninstall
// rather than delete (brick) that routine.
func TestUninstall_RefusesUndeclaredOverwrite(t *testing.T) {
	dir := t.TempDir()
	kid := filepath.Join(dir, "ZZSKEL.KID")
	pairs := kids.MakeBuildPairs(kids.BuildInput{InstallName: "ZZSKEL*1.0*1", Namespace: "ZZSKEL",
		Routines: []kids.RoutineSrc{{Name: "ZZSKEL", Lines: []string{"ZZSKEL ;x", " Q"}}}})
	if err := kids.WriteKID([]string{"ZZSKEL*1.0*1"}, map[string][]kids.Pair{"ZZSKEL*1.0*1": pairs}, kid); err != nil {
		t.Fatal(err)
	}
	// Ledger records the install OVERWROTE the pre-existing ZZSKEL (Before != absent).
	if err := emitAttestation(attestFlags{}, kid, newRecord(attestInput{
		Op: "install", Name: "ZZSKEL*1.0*1", Before: map[string]string{"ZZSKEL": "B10838"},
	})); err != nil {
		t.Fatal(err)
	}

	cc := clikit.NewContext(&clikit.Globals{Output: "json"}, "uninstall")
	cc.Stdout = &bytes.Buffer{}
	cmd := &uninstallCmd{engineConn: engineConn{Engine: "ydb", Transport: "docker"}, KidFile: kid}
	err := cmd.Run(cc)
	var ce *clikit.Error
	if !errors.As(err, &ce) || ce.Exit != clikit.ExitRefused {
		t.Fatalf("bare uninstall of an undeclared overwrite: got %v, want a Refused (exit 4) error", err)
	}

	// With --force, the overwrite guard must NOT fire (it proceeds past the offline
	// refuse toward the engine, then fails later on no-driver / handshake — a DIFFERENT
	// error code). Assert only that it is not the UNINSTALL_REFUSED overwrite guard.
	cmd.Force = true
	if ferr := cmd.Run(cc); errors.As(ferr, &ce) && ce.Code == "UNINSTALL_REFUSED" {
		t.Error("--force must not be refused by the overwrite guard")
	}
}
