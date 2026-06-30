package pkgcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// kidWith writes a minimal single-build .KID carrying one routine ZZT with the given
// stored RTN-node checksum and source lines, and returns its path. The source below
// has a ground-truthed SUMB of B10838 (see internal/kids/checksum_test.go).
func kidWith(t *testing.T, stored string, line5 string) string {
	t.Helper()
	body := strings.Join([]string{
		"KIDS Distribution saved by v-pkg",
		"v-pkg reassembled output",
		"**KIDS**:ZZT*1.0*1^",
		"",
		"**INSTALL NAME**",
		"ZZT*1.0*1",
		`"BLD",1,0)`,
		"ZZT*1.0*1^ZZT^0^0",
		`"RTN")`,
		"1",
		`"RTN","ZZT")`,
		"0^5^" + stored,
		`"RTN","ZZT",1,0)`,
		"ZZT ;test routine ;1.0",
		`"RTN","ZZT",2,0)`,
		" ;;1.0;ZZT;;",
		`"RTN","ZZT",3,0)`,
		" Q",
		`"RTN","ZZT",4,0)`,
		"PING() ;",
		`"RTN","ZZT",5,0)`,
		line5,
		`"VER")`,
		"8.0^22.2",
		"**END**",
		"**END**",
		"",
	}, "\n")
	p := filepath.Join(t.TempDir(), "ZZT.kids")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// checksumScan must FLAG a foreign .KID whose stored B-checksum does not match the
// shipped source (a byte-tamper), and find nothing wrong with a pristine one — all
// OFFLINE, no engine. checksumRefusal then turns a scan into the opt-in hard refusal.
func TestChecksumScanAndRefusal(t *testing.T) {
	// Pristine: stored B10838 matches the unmodified source → empty scan, no refusal.
	kp, err := kids.ParseKID(kidWith(t, "B10838", ` Q "pong"`))
	if err != nil {
		t.Fatal(err)
	}
	if scan := checksumScan(kp); len(scan) != 0 {
		t.Errorf("pristine .KID must produce an empty checksum scan, got %v", scan)
	}
	if ferr := checksumRefusal(checksumScan(kp)); ferr != nil {
		t.Errorf("pristine .KID must not be refused, got: %v", ferr)
	}

	// Tampered: same stored B10838, but line 5 was flipped — recomputed differs.
	kt, err := kids.ParseKID(kidWith(t, "B10838", ` Q "PONG"`))
	if err != nil {
		t.Fatal(err)
	}
	scan := checksumScan(kt)
	if len(scan["ZZT*1.0*1"]) != 1 {
		t.Fatalf("tampered .KID must flag exactly one routine, got %v", scan)
	}
	// Warn mode (default): a scan is informational — checksumRefusal still REFUSES
	// only when called (the strict --verify-checksums path); the caller decides.
	ferr := checksumRefusal(scan)
	if ferr == nil {
		t.Fatal("checksumRefusal on a mismatch scan must REFUSE (the --verify-checksums gate)")
	}
	if ferr.Code != "CHECKSUM_MISMATCH" {
		t.Errorf("refusal code = %q, want CHECKSUM_MISMATCH", ferr.Code)
	}
}

// A v-pkg build (stored "0", real checksums computed at install) has nothing to
// verify and must never be flagged — no false-fail, in warn or strict mode.
func TestChecksumScan_VpkgBuildClean(t *testing.T) {
	k, err := kids.ParseKID(kidWith(t, "0^0", ` Q "pong"`)) // 0^0 → piece3 "0"
	if err != nil {
		t.Fatal(err)
	}
	if scan := checksumScan(k); len(scan) != 0 {
		t.Errorf("a v-pkg (0/0) build must produce an empty checksum scan, got %v", scan)
	}
}
