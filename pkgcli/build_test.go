package pkgcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vista-cloud-dev/clikit"
)

// runBuild builds the ZZSKEL test package to out, discarding the JSON envelope.
func runBuild(t *testing.T, out string) {
	t.Helper()
	runBuildPkg(t, "zzskel", "ZZSKEL", out)
}

// runBuildPkg builds testdata/<dir>/kids/<pkg>.build.json to out.
func runBuildPkg(t *testing.T, dir, pkg, out string) {
	t.Helper()
	cc := clikit.NewContext(&clikit.Globals{Output: "json"}, "build")
	cc.Stdout = &bytes.Buffer{}
	cmd := &buildCmd{
		Spec: filepath.Join("..", "testdata", dir, "kids", pkg+".build.json"),
		Src:  filepath.Join("..", "testdata", dir, "src"),
		Out:  out,
	}
	if err := cmd.Run(cc); err != nil {
		t.Fatalf("v pkg build: %v", err)
	}
}

// TestBuild_ZZPARAM_Deterministic is the deterministic-build + golden gate for a
// package carrying a #8989.51 PARAMETER DEFINITION KRN component (the T1.3 shape):
// two builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZPARAM_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzparam", "ZZPARAM", a)
	runBuildPkg(t, "zzparam", "ZZPARAM", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (param-def) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzparam", "ZZPARAM.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZPARAM.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZVSLFS_Deterministic is the M3.T1 gate for a package shipping a
// brand-new FileMan FILE data dictionary (#999000 ZZVSLFS): two builds are
// byte-identical and match the committed golden .KID.
func TestBuild_ZZVSLFS_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzvslfs", "ZZVSLFS", a)
	runBuildPkg(t, "zzvslfs", "ZZVSLFS", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (file DD) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzvslfs", "ZZVSLFS.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZVSLFS.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZVSLAU_Deterministic is the B.2 gate for a package shipping a
// brand-new multi-field FileMan FILE (#999001 ZZVSL AUDIT — the five grounded
// field types: numeric, set-of-codes, date, pointer, free-text). It exercises the
// full path (buildspec → resolveFields → MakeBuildPairs): two builds are
// byte-identical and match the committed golden .KID.
func TestBuild_ZZVSLAU_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzvslaudit", "ZZVSLAU", a)
	runBuildPkg(t, "zzvslaudit", "ZZVSLAU", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (multi-field file DD) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzvslaudit", "ZZVSLAU.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZVSLAU.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZOPTION_Deterministic is the B.1 gate for a package shipping a #19
// OPTION as a KIDS KRN component (the generic entry-component emitter): two builds
// are byte-identical and match the committed golden .KID.
func TestBuild_ZZOPTION_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzoption", "ZZOPTION", a)
	runBuildPkg(t, "zzoption", "ZZOPTION", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (option) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzoption", "ZZOPTION.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZOPTION.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZKEY_Deterministic is the B.1 gate for a package shipping a SECURITY
// KEY (#19.1) as a KIDS KRN component (the third type on the generic emitter): two
// builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZKEY_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzkey", "ZZKEY", a)
	runBuildPkg(t, "zzkey", "ZZKEY", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (security key) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzkey", "ZZKEY.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZKEY.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZPROTO_Deterministic is the B.1 gate for a package shipping a
// PROTOCOL (#101) as a KIDS KRN component (the fourth type on the generic
// emitter): two builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZPROTO_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzproto", "ZZPROTO", a)
	runBuildPkg(t, "zzproto", "ZZPROTO", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (protocol) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzproto", "ZZPROTO.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZPROTO.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZRPC_Deterministic is the B.1 gate for a package shipping a REMOTE
// PROCEDURE (#8994) as a KIDS KRN component (the fifth type on the generic
// emitter): two builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZRPC_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzrpc", "ZZRPC", a)
	runBuildPkg(t, "zzrpc", "ZZRPC", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (rpc) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzrpc", "ZZRPC.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZRPC.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZMG_Deterministic is the B.1 gate for a package shipping a MAIL GROUP
// (#3.8) as a KIDS KRN component (the sixth type on the generic emitter): two
// builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZMG_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzmg", "ZZMG", a)
	runBuildPkg(t, "zzmg", "ZZMG", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (mail group) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzmg", "ZZMG.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZMG.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZLM_Deterministic is the B.1 gate for a package shipping a LIST
// TEMPLATE (#409.61) as a KIDS KRN component (the seventh type on the generic
// emitter): two builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZLM_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzlm", "ZZLM", a)
	runBuildPkg(t, "zzlm", "ZZLM", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (list template) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzlm", "ZZLM.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZLM.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZHF_Deterministic is the B.1 gate for a package shipping a HELP FRAME
// (#9.2) as a KIDS KRN component (the eighth type on the generic emitter): two
// builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZHF_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzhf", "ZZHF", a)
	runBuildPkg(t, "zzhf", "ZZHF", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (help frame) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzhf", "ZZHF.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZHF.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZHL_Deterministic is the B.1 gate for a package shipping an HL7
// APPLICATION PARAMETER (#771) as a KIDS KRN component (the ninth type on the
// generic emitter): two builds are byte-identical and match the committed golden.
func TestBuild_ZZHL_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzhl", "ZZHL", a)
	runBuildPkg(t, "zzhl", "ZZHL", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (hl7 app) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzhl", "ZZHL.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZHL.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZMIX_Deterministic is the B.1 multi-type gate: one build shipping
// BOTH an OPTION (#19) and a PARAMETER DEFINITION (#8989.51) — they share a single
// computed "BLD",1,"KRN",0) manifest header and take distinct ORD orders. Two
// builds are byte-identical and match the committed golden .KID.
func TestBuild_ZZMIX_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuildPkg(t, "zzmix", "ZZMIX", a)
	runBuildPkg(t, "zzmix", "ZZMIX", b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build (mixed entry types) is not deterministic — two builds differ")
	}

	golden := filepath.Join("..", "testdata", "zzmix", "ZZMIX.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZMIX.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}

// TestBuild_ZZSKEL_Deterministic is the T0a.2 gate: `v pkg build` of the ZZSKEL
// package yields a byte-identical normalized export across runs (deterministic
// build, coordination plan §7.2 #2), and matches the committed golden.
func TestBuild_ZZSKEL_Deterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.kids")
	b := filepath.Join(dir, "b.kids")
	runBuild(t, a)
	runBuild(t, b)

	gotA, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatal("v pkg build is not deterministic — two builds of the same spec differ")
	}

	golden := filepath.Join("..", "testdata", "zzskel", "ZZSKEL.kids")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, gotA, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(gotA, want) {
		t.Errorf("ZZSKEL.kids drift — run UPDATE_GOLDEN=1\n--- got ---\n%s", gotA)
	}
}
