package pkgcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vista-cloud-dev/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// defaultPreimagePath is the conventional sidecar location for a patch's
// pre-image snapshot: the .KID path with its extension replaced by
// ".preimage.kids", next to the patch. `install --auto-snapshot` writes here and
// `uninstall` (with no reversal flag) auto-detects it — so the install/uninstall
// pair needs no explicit path.
func defaultPreimagePath(kidPath string) string {
	ext := filepath.Ext(kidPath)
	return strings.TrimSuffix(kidPath, ext) + ".preimage.kids"
}

// resolveAutoRestore decides the effective pre-image to restore on uninstall. An
// explicit --restore/--backout always governs (no auto). With neither flag, if a
// conventional sidecar pre-image exists next to the patch, it is auto-detected so
// `uninstall <patch.kid>` reverses cleanly without re-specifying the snapshot.
func resolveAutoRestore(restore, backout, kidPath string, sidecarExists bool) (string, bool) {
	if restore == "" && backout == "" && sidecarExists {
		return defaultPreimagePath(kidPath), true
	}
	return restore, false
}

// fileExists reports whether path is a readable regular file.
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

// verifySidecarIntegrity refuses a tampered pre-image before it is restored (#3c). A
// snapshot v-pkg captured is stamped with a content hash (kids.StampHash); restore /
// uninstall auto-restore recompute it and a mismatch means the sidecar's routine
// source was altered after capture — restoring it would silently put back the WRONG
// source. A build with no stamp (an authored --backout, or a pre-#3c snapshot) has
// nothing to verify and passes. The path is named for the operator's diagnostics.
func verifySidecarIntegrity(b *kids.Build, path string) *clikit.Error {
	stored, ok, present := kids.VerifySidecarHash(b)
	if !present || ok {
		return nil
	}
	short := stored
	if len(short) > 12 {
		short = short[:12]
	}
	return clikit.Fail(clikit.ExitRefused, "SIDECAR_TAMPERED",
		fmt.Sprintf("refusing to restore from %s: pre-image integrity hash mismatch (stamped %s…) — the snapshot was altered after capture and would restore the wrong routine source", path, short),
		"re-capture the pre-image with install --auto-snapshot; never restore from an altered sidecar")
}
