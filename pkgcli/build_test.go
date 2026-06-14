package pkgcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/clikit"
)

// runBuild builds the ZZSKEL test package to out, discarding the JSON envelope.
func runBuild(t *testing.T, out string) {
	t.Helper()
	cc := clikit.NewContext(&clikit.Globals{Output: "json"}, "build")
	cc.Stdout = &bytes.Buffer{}
	cmd := &buildCmd{
		Spec: filepath.Join("..", "testdata", "zzskel", "kids", "ZZSKEL.build.json"),
		Src:  filepath.Join("..", "testdata", "zzskel", "src"),
		Out:  out,
	}
	if err := cmd.Run(cc); err != nil {
		t.Fatalf("v pkg build: %v", err)
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
