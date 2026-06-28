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
