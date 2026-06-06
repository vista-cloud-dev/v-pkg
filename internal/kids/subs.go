// Package kids parses, decomposes, assembles, and round-trip-verifies VistA
// KIDS (Kernel Installation & Distribution System) distribution files. It is a
// faithful Go port of py-kids-vc (github.com/rafael5/py-kids-vc, v0.1.0), which
// in turn ports Sam Habiel's XPDK2VC architecture. The decompose/assemble/
// round-trip contract and the KIDComponents/ layout are unchanged from the
// Python tool.
//
// The round-trip guarantee is semantic equality after line-2 canonicalization,
// NOT byte-for-byte: routine line 2's volatile patch-list/date/build-number
// pieces are stripped (see CanonicalizeRoutineLine2), exactly as XPDK2VC and
// py-kids-vc do. Roundtrip compares the re-parsed builds, not raw bytes.
package kids

import (
	"strconv"
	"strings"
)

// subKind distinguishes the three subscript element types KIDS uses: quoted
// strings, integers, and decimal file numbers. The distinction is load-bearing:
// the decomposer and CanonicalizeIENs branch on int-vs-float-vs-str (mirroring
// Python's isinstance checks).
type subKind uint8

const (
	kindStr subKind = iota
	kindInt
	kindFloat
)

// Sub is one subscript element: a quoted string, an integer, or a decimal file
// number. Numerics keep their parsed value (for sorting, equality keys, and the
// int/float/zero tests the decomposer makes); on output they are reformatted
// from the value the way Python's str() renders them — `.4` normalizes to
// `0.4`, matching py-kids-vc byte-for-byte. String elements are quoted/escaped.
type Sub struct {
	kind subKind
	str  string
	intV int64
	fltV float64
}

// formatKIDSFloat renders a float subscript the way Python's str(float) does for
// the decimal file-number range KIDS uses (no exponent; `.4`→`0.4`).
func formatKIDSFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// strSub builds a string-kind subscript element.
func strSub(s string) Sub { return Sub{kind: kindStr, str: s} }

// IsStr reports whether the element is a string subscript.
func (s Sub) IsStr() bool { return s.kind == kindStr }

// IsInt reports whether the element is an integer subscript.
func (s Sub) IsInt() bool { return s.kind == kindInt }

// IsFloat reports whether the element is a decimal (file-number) subscript.
func (s Sub) IsFloat() bool { return s.kind == kindFloat }

// IsNumeric reports whether the element is an int or float subscript.
func (s Sub) IsNumeric() bool { return s.kind != kindStr }

// Str returns the string value (empty for numerics).
func (s Sub) Str() string { return s.str }

// Int returns the integer value (0 for non-ints).
func (s Sub) Int() int64 { return s.intV }

// IsZeroInt reports whether the element is the integer 0 — the IEN=0 file-header
// marker the KRN decomposer keys on.
func (s Sub) IsZeroInt() bool { return s.kind == kindInt && s.intV == 0 }

// Subs is a parsed subscript tuple — the tail of a $NA reference such as
// `"KRN",19,12345,0)`.
type Subs []Sub

// coerceNum mirrors Python's int(token) if token.isdigit() else _coerce_num.
// A bare run of digits is an int; a token containing a '.' whose non-dot
// characters are all digits is a float; a leading-sign integer is an int;
// anything else falls back to a string element (which will be quoted on
// output, matching Python's str fallback).
func coerceNum(token string) Sub {
	if token != "" && isAllDigits(token) {
		if v, err := strconv.ParseInt(token, 10, 64); err == nil {
			return Sub{kind: kindInt, intV: v}
		}
	}
	if strings.Contains(token, ".") {
		if isAllDigits(strings.ReplaceAll(token, ".", "")) {
			if v, err := strconv.ParseFloat(token, 64); err == nil {
				return Sub{kind: kindFloat, fltV: v}
			}
		}
		return strSub(token)
	}
	if v, err := strconv.ParseInt(token, 10, 64); err == nil {
		return Sub{kind: kindInt, intV: v}
	}
	return strSub(token)
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseSubscriptLine parses a KIDS subscript line like `"KRN",19,12345,0)` into
// a Subs tuple. It is the port of Python's _parse_subscript_line: the trailing
// `)` is dropped, quoted strings handle the `""` escape, and unquoted tokens
// are coerced to int/float/string.
func parseSubscriptLine(line string) Subs {
	if !strings.HasSuffix(line, ")") {
		// Python asserts; in the Go port a malformed line yields no subs and
		// callers (decompose's tolerant path) skip it.
		return nil
	}
	inner := line[:len(line)-1]
	var parts Subs
	i, n := 0, len(inner)
	for i < n {
		c := inner[i]
		switch {
		case c == '"':
			j := i + 1
			var buf strings.Builder
			for j < n {
				if inner[j] == '"' {
					if j+1 < n && inner[j+1] == '"' {
						buf.WriteByte('"')
						j += 2
						continue
					}
					break
				}
				buf.WriteByte(inner[j])
				j++
			}
			parts = append(parts, strSub(buf.String()))
			i = j + 1
		case c == ',':
			i++
		default:
			j := i
			for j < n && inner[j] != ',' {
				j++
			}
			parts = append(parts, coerceNum(inner[i:j]))
			i = j
		}
	}
	return parts
}

// formatSubscript is the inverse of parseSubscriptLine: it renders a Subs back
// to `"KRN",19,12345,0)` form. Strings are quoted with `"`→`""` escaping;
// numerics emit their preserved source token (falling back to the parsed value
// for synthesized elements such as the "IEN" placeholder).
func formatSubscript(subs Subs) string {
	var b strings.Builder
	for i, s := range subs {
		if i > 0 {
			b.WriteByte(',')
		}
		switch s.kind {
		case kindStr:
			b.WriteByte('"')
			b.WriteString(strings.ReplaceAll(s.str, `"`, `""`))
			b.WriteByte('"')
		case kindInt:
			b.WriteString(strconv.FormatInt(s.intV, 10))
		case kindFloat:
			b.WriteString(formatKIDSFloat(s.fltV))
		}
	}
	b.WriteByte(')')
	return b.String()
}

// key is the canonical equality/identity string for a Subs, used as an
// ordered-map key. It encodes kind+value (NOT the source token) so that, like
// Python's dict, two tokens that denote the same value collide on one key.
// Strings are length-prefixed so the encoding is injective.
func (s Subs) key() string {
	var b strings.Builder
	for _, e := range s {
		switch e.kind {
		case kindStr:
			b.WriteByte('s')
			b.WriteString(strconv.Itoa(len(e.str)))
			b.WriteByte('|')
			b.WriteString(e.str)
		case kindInt:
			b.WriteByte('i')
			b.WriteString(strconv.FormatInt(e.intV, 10))
		case kindFloat:
			b.WriteByte('f')
			b.WriteString(strconv.FormatFloat(e.fltV, 'g', -1, 64))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// key is the canonical identity string for a single element, used to group by
// file number.
func (s Sub) key() string { return Subs{s}.key() }

// Section returns the top-level KIDS section name (the leading string element),
// or "UNKNOWN" when the tuple is empty or doesn't begin with a string.
func (s Subs) Section() string {
	if len(s) > 0 && s[0].IsStr() {
		return s[0].str
	}
	return "UNKNOWN"
}

// less orders two Subs the way Python's _sort_key does: element-by-element,
// first by type name ("float" < "int" < "str"), then by value within a type;
// a shorter tuple that is a prefix of a longer one sorts first.
func (s Subs) less(o Subs) bool {
	n := len(s)
	if len(o) < n {
		n = len(o)
	}
	for i := 0; i < n; i++ {
		a, b := s[i], o[i]
		if a.kind != b.kind {
			return typeName(a.kind) < typeName(b.kind)
		}
		switch a.kind {
		case kindStr:
			if a.str != b.str {
				return a.str < b.str
			}
		case kindInt:
			if a.intV != b.intV {
				return a.intV < b.intV
			}
		case kindFloat:
			if a.fltV != b.fltV {
				return a.fltV < b.fltV
			}
		}
	}
	return len(s) < len(o)
}

// typeName maps a kind to the Python type name it sorts as, so the ordering
// matches _sort_key's (type(s).__name__, s) tuples exactly.
func typeName(k subKind) string {
	switch k {
	case kindFloat:
		return "float"
	case kindInt:
		return "int"
	default:
		return "str"
	}
}

// zwrLine renders one subscript=value pair as a ZWR-format line, quoting and
// escaping the value (`"`→`""`).
func zwrLine(subs Subs, value string) string {
	return formatSubscript(subs) + `="` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// parseZWRLine parses a ZWR line like `"BLD",1,0)="..."` into (subs, value),
// the inverse of zwrLine. Like Python it splits on the first `)=`.
func parseZWRLine(line string) (Subs, string, bool) {
	idx := strings.Index(line, ")=")
	if idx < 0 {
		return nil, "", false
	}
	subsText := line[:idx+1]
	valueText := line[idx+2:]
	subs := parseSubscriptLine(subsText)
	if strings.HasPrefix(valueText, `"`) && strings.HasSuffix(valueText, `"`) && len(valueText) >= 2 {
		inner := strings.ReplaceAll(valueText[1:len(valueText)-1], `""`, `"`)
		return subs, inner, true
	}
	return subs, valueText, true
}
