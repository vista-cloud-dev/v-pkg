package kids

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const ddCodeReadme = `# DD-embedded MUMPS code annotations

These ` + "`.m`" + ` files are EXTRACTIONS of MUMPS code found inside the
FileMan ^DD nodes in ` + "`../DD.zwr`" + `. They are **informational** —
` + "`DD.zwr`" + ` is authoritative for KIDS round-trip.

Use these files to diff / blame / review the embedded code in a
human-readable form. Any change you want reflected in the KIDS
build must be made in the DD.zwr source (the assembler only
reads DD.zwr).

Filename convention: ` + "`<field>.<kind>.m`" + ` where kind ∈
{input-transform, computed, xref-set, xref-kill}.
`

// numEq reports whether a numeric subscript element equals n.
func numEq(s Sub, n float64) bool {
	v, ok := s.numVal()
	return ok && v == n
}

// pyFieldStr renders a subscript element the way Python's str() would, for
// composing DD annotation field names. Floats use the no-exponent formatter
// (file/field numbers like 9000010.11 must not become 9.00001011e+06).
func pyFieldStr(s Sub) string {
	switch s.kind {
	case kindStr:
		return s.str
	case kindInt:
		return strconv.FormatInt(s.intV, 10)
	default:
		return formatKIDSFloat(s.fltV)
	}
}

func joinFieldPretty(subs Subs) string {
	parts := make([]string, len(subs))
	for i, s := range subs {
		parts[i] = pyFieldStr(s)
	}
	return strings.Join(parts, ".")
}

// extractDDCode extracts MUMPS code embedded in ^DD nodes as per-field .m
// annotation files. Port of _extract_dd_code. The annotations are informational
// only — DD.zwr remains authoritative for round-trip, so these files are never
// read back by AssembleBuild.
func extractDDCode(ddPairs []Pair, outDir string) error {
	var emitted []string
	created := false

	isTrivial := func(code string) bool {
		c := strings.TrimSpace(code)
		switch c {
		case "", "Q", "K", "D", ";":
			return true
		}
		if len(c) <= 2 && !strings.ContainsAny(c, "=<>+-_") {
			return true
		}
		return false
	}

	emit := func(field, kind, code string) error {
		if isTrivial(code) {
			return nil
		}
		if !created {
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			created = true
		}
		safeField := sanitize(field)
		path := filepath.Join(outDir, safeField+"."+kind+".m")
		if err := os.WriteFile(path, []byte(code+"\n"), 0o644); err != nil {
			return err
		}
		emitted = append(emitted, path)
		return nil
	}

	isFieldZeroNode := func(subs Subs, value string) bool {
		if len(subs) < 4 || !numEq(subs[len(subs)-1], 0) {
			return false
		}
		pieces := strings.Split(value, "^")
		if len(pieces) < 5 {
			return false
		}
		typeCode := pieces[1]
		return len(typeCode) > 0 && typeCode[0] >= 'A' && typeCode[0] <= 'Z'
	}

	for _, p := range ddPairs {
		subs, value := p.Subs, p.Value
		if len(subs) < 2 || !subs[0].IsStr() || subs[0].str != "^DD" {
			continue
		}
		last := len(subs) - 1

		if isFieldZeroNode(subs, value) {
			fieldPretty := joinFieldPretty(subs[2:last])
			pieces := strings.Split(value, "^")
			fieldType := pieces[1]
			if strings.HasPrefix(fieldType, "C") {
				code := strings.Join(pieces[4:], "^")
				if err := emit(fieldPretty, "computed", code); err != nil {
					return err
				}
			} else if pieces[4] != "" {
				if err := emit(fieldPretty, "input-transform", pieces[4]); err != nil {
					return err
				}
			}
			continue
		}

		// Cross-reference code: (^DD, …, field, 1, xref_ien, {1,2}).
		if len(subs) >= 6 && (numEq(subs[last], 1) || numEq(subs[last], 2)) &&
			subs[last-1].IsInt() && subs[last-1].intV > 0 && numEq(subs[last-2], 1) {
			kind := "xref-set"
			if numEq(subs[last], 2) {
				kind = "xref-kill"
			}
			xrefIEN := subs[last-1].intV
			fieldPretty := joinFieldPretty(subs[2 : last-2])
			if err := emit(fieldPretty+".xref-"+strconv.FormatInt(xrefIEN, 10), kind, value); err != nil {
				return err
			}
			continue
		}

		// Computed-field word-processing code: (^DD, …, field, 9, N, 0).
		if len(subs) >= 6 && numEq(subs[last], 0) && subs[last-1].IsInt() && numEq(subs[last-2], 9) {
			fieldPretty := joinFieldPretty(subs[2 : last-2])
			if err := emit(fieldPretty, "computed-wp", value); err != nil {
				return err
			}
			continue
		}
	}

	if len(emitted) > 0 {
		if err := os.WriteFile(filepath.Join(outDir, "_README.md"), []byte(ddCodeReadme), 0o644); err != nil {
			return err
		}
	}
	return nil
}
