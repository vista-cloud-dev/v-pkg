package kids

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// readZWR reads a .zwr file into pairs, skipping blank lines (port of the inner
// _read_zwr). A missing file yields no pairs.
func readZWR(path string) ([]Pair, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var pairs []Pair
	for _, line := range splitKIDLines(data) {
		if strings.TrimSpace(line) == "" {
			continue
		}
		subs, value, ok := parseZWRLine(line)
		if !ok {
			continue
		}
		pairs = append(pairs, Pair{Subs: subs, Value: value})
	}
	return pairs, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// listDirsSorted returns the subdirectory names of dir, sorted (mirrors
// sorted(path.iterdir()) filtered to directories).
func listDirsSorted(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// globSorted returns paths in dir matching pattern, sorted by full path
// (mirrors sorted(dir.glob(pattern))).
func globSorted(dir, pattern string) []string {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil
	}
	sort.Strings(matches)
	return matches
}

// AssembleBuild reads a decomposed component tree under inDir and reconstructs
// the (subscript, value) pairs of one build. Port of assemble_build; installName
// is accepted for parity but unused (no IEN substitution in the MVP contract).
func AssembleBuild(inDir, installName string) ([]Pair, error) {
	_ = installName
	var pairs []Pair
	add := func(ps []Pair, err error) error {
		if err != nil {
			return err
		}
		pairs = append(pairs, ps...)
		return nil
	}

	// Simple sections.
	for _, ss := range simpleSections {
		path := filepath.Join(inDir, ss.filename)
		if exists(path) {
			if err := add(readZWR(path)); err != nil {
				return nil, err
			}
		}
	}

	// Routines: _index.zwr + per-routine .header + .m.
	rtnDir := filepath.Join(inDir, "Routines")
	if exists(rtnDir) {
		idxPath := filepath.Join(rtnDir, "_index.zwr")
		if exists(idxPath) {
			if err := add(readZWR(idxPath)); err != nil {
				return nil, err
			}
		}
		for _, mPath := range globSorted(rtnDir, "*.m") {
			name := strings.TrimSuffix(filepath.Base(mPath), ".m")
			headerPath := filepath.Join(rtnDir, name+".header")
			header := ""
			if exists(headerPath) {
				hb, err := os.ReadFile(headerPath)
				if err != nil {
					return nil, err
				}
				header = strings.TrimRight(string(hb), "\n")
			}
			pairs = append(pairs, Pair{Subs: rtnHeaderSubs(name), Value: header})
			mb, err := os.ReadFile(mPath)
			if err != nil {
				return nil, err
			}
			for idx, content := range splitKIDLines(mb) {
				pairs = append(pairs, Pair{Subs: rtnLineSubs(name, int64(idx+1)), Value: content})
			}
		}
	}

	// ORD (top-level single file).
	if ordPath := filepath.Join(inDir, "ORD.zwr"); exists(ordPath) {
		if err := add(readZWR(ordPath)); err != nil {
			return nil, err
		}
	}

	// KRN.
	krnRoot := filepath.Join(inDir, "KRN")
	if exists(krnRoot) {
		miscPath := filepath.Join(krnRoot, "_misc.zwr")
		if exists(miscPath) {
			if err := add(readZWR(miscPath)); err != nil {
				return nil, err
			}
		}
		for _, name := range listDirsSorted(krnRoot) {
			fileDir := filepath.Join(krnRoot, name)
			hdr := filepath.Join(fileDir, "FileHeader.zwr")
			if exists(hdr) {
				if err := add(readZWR(hdr)); err != nil {
					return nil, err
				}
			}
			for _, entryPath := range globSorted(fileDir, "*.zwr") {
				if filepath.Base(entryPath) == "FileHeader.zwr" {
					continue
				}
				if err := add(readZWR(entryPath)); err != nil {
					return nil, err
				}
			}
		}
	}

	// FIA (FileMan files): Files/_unclaimed.zwr + Files/<dir>/{DD,Data}.zwr.
	filesRoot := filepath.Join(inDir, "Files")
	if exists(filesRoot) {
		unclaimed := filepath.Join(filesRoot, "_unclaimed.zwr")
		if exists(unclaimed) {
			if err := add(readZWR(unclaimed)); err != nil {
				return nil, err
			}
		}
		for _, name := range listDirsSorted(filesRoot) {
			fileDir := filepath.Join(filesRoot, name)
			if dd := filepath.Join(fileDir, "DD.zwr"); exists(dd) {
				if err := add(readZWR(dd)); err != nil {
					return nil, err
				}
			}
			if data := filepath.Join(fileDir, "Data.zwr"); exists(data) {
				if err := add(readZWR(data)); err != nil {
					return nil, err
				}
			}
		}
	}

	// Catch-all misc.
	if miscPath := filepath.Join(inDir, "_misc.zwr"); exists(miscPath) {
		if err := add(readZWR(miscPath)); err != nil {
			return nil, err
		}
	}

	return pairs, nil
}

// rtnHeaderSubs builds the ("RTN", name) header subscript.
func rtnHeaderSubs(name string) Subs {
	return Subs{strSub("RTN"), strSub(name)}
}

// rtnLineSubs builds the ("RTN", name, line, 0) routine-line subscript.
func rtnLineSubs(name string, line int64) Subs {
	return Subs{strSub("RTN"), strSub(name), {kind: kindInt, intV: line}, {kind: kindInt, intV: 0}}
}

// WriteKID serializes builds back to KIDS distribution text. Port of write_kid.
func WriteKID(installNames []string, buildsPairs map[string][]Pair, outPath string) error {
	var lines []string
	lines = append(lines,
		"KIDS Distribution saved by v-pkg",
		"m-kids reassembled output",
		"**KIDS**:"+strings.Join(installNames, "^")+"^",
		"",
	)
	for _, name := range installNames {
		lines = append(lines, "**INSTALL NAME**", name)
		for _, p := range buildsPairs[name] {
			lines = append(lines, formatSubscript(p.Subs), p.Value)
		}
	}
	lines = append(lines, "**END**", "**END**")
	return os.WriteFile(outPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
