package pkgcli

import (
	"os"
	"path/filepath"
	"strings"
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
