package kids

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// IENStats reports how many IEN substitutions CanonicalizeIENs made per section.
type IENStats struct {
	BLD int `json:"BLD"`
	KRN int `json:"KRN"`
}

// Total is the sum of all substitutions.
func (s IENStats) Total() int { return s.BLD + s.KRN }

// CanonicalizeIENs rewrites every .zwr under decompDir to substitute integer
// IENs at known positions with the literal "IEN" — for cross-instance diffing.
// LOSSY: the original IENs are discarded (KIDS reassigns them on install).
// Port of canonicalize_iens.
//
//   - ("BLD", <int>, …)            → position 1 (build IEN)
//   - ("KRN", <numeric>, <int≠0>…) → position 2 (entry IEN per file)
func CanonicalizeIENs(decompDir string) (IENStats, error) {
	var stats IENStats
	err := filepath.WalkDir(decompDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".zwr") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		var newLines []string
		modified := false
		for _, line := range splitKIDLines(data) {
			if strings.TrimSpace(line) == "" {
				newLines = append(newLines, line)
				continue
			}
			subs, value, ok := parseZWRLine(line)
			if !ok || subs == nil {
				newLines = append(newLines, line)
				continue
			}
			newSubs := subs
			switch {
			case len(subs) >= 2 && subs[0].IsStr() && subs[0].str == "BLD" && subs[1].IsInt():
				newSubs = append(Subs{strSub("BLD"), strSub("IEN")}, subs[2:]...)
				stats.BLD++
				modified = true
			case len(subs) >= 3 && subs[0].IsStr() && subs[0].str == "KRN" &&
				subs[1].IsNumeric() && subs[2].IsInt() && subs[2].intV != 0:
				newSubs = append(Subs{strSub("KRN"), subs[1], strSub("IEN")}, subs[3:]...)
				stats.KRN++
				modified = true
			}
			newLines = append(newLines, zwrLine(newSubs, value))
		}
		if modified {
			out := strings.Join(newLines, "\n") + "\n"
			if werr := os.WriteFile(path, []byte(out), 0o644); werr != nil {
				return werr
			}
		}
		return nil
	})
	return stats, err
}
