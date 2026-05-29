package kids

import (
	"os"
	"sort"
	"strings"
)

// Pair is one (subscript, value) entry of a build.
type Pair struct {
	Subs  Subs
	Value string
}

// Build is one install's parsed subscript→value data. It is an insertion-
// ordered map (mirroring a Python dict): a repeated subscript overwrites the
// prior value but keeps its position, and iteration order is insertion order.
type Build struct {
	keys []Subs
	vals []string
	idx  map[string]int
}

func newBuild() *Build { return &Build{idx: map[string]int{}} }

// Set assigns value to subs, overwriting any existing entry for the same key.
func (b *Build) Set(subs Subs, value string) {
	k := subs.key()
	if i, ok := b.idx[k]; ok {
		b.vals[i] = value
		b.keys[i] = subs
		return
	}
	b.idx[k] = len(b.keys)
	b.keys = append(b.keys, subs)
	b.vals = append(b.vals, value)
}

// Get returns the value for subs.
func (b *Build) Get(subs Subs) (string, bool) {
	if i, ok := b.idx[subs.key()]; ok {
		return b.vals[i], true
	}
	return "", false
}

// Len is the number of distinct subscripts.
func (b *Build) Len() int { return len(b.keys) }

// Pairs returns the entries in insertion order.
func (b *Build) Pairs() []Pair {
	out := make([]Pair, len(b.keys))
	for i := range b.keys {
		out[i] = Pair{Subs: b.keys[i], Value: b.vals[i]}
	}
	return out
}

// Sorted returns the entries ordered by the _sort_key collation.
func (b *Build) Sorted() []Pair {
	out := b.Pairs()
	sort.SliceStable(out, func(i, j int) bool { return out[i].Subs.less(out[j].Subs) })
	return out
}

// KID is a parsed KIDS distribution: the ordered install names plus each
// install's build data.
type KID struct {
	InstallNames []string
	Builds       map[string]*Build
}

// build returns (creating if needed) the Build for an install name.
func (k *KID) build(name string) *Build {
	b, ok := k.Builds[name]
	if !ok {
		b = newBuild()
		k.Builds[name] = b
	}
	return b
}

// subscriptStart reports whether a content line begins a subscript line — the
// port of Python's SUBSCRIPT_RE `^"\^?[A-Z]`: a quote, an optional caret, then
// an uppercase letter.
func subscriptStart(line string) bool {
	if !strings.HasPrefix(line, `"`) {
		return false
	}
	rest := line[1:]
	rest = strings.TrimPrefix(rest, "^")
	return len(rest) > 0 && rest[0] >= 'A' && rest[0] <= 'Z'
}

// ParseKID parses a .KID file. It is the port of py-kids-vc's parse_kid state
// machine: BEGIN → KIDSSS → INSTLNM → CONTENT, looping back to INSTLNM on each
// new install and pairing subscript lines with their following value line.
func ParseKID(path string) (*KID, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseKIDBytes(data), nil
}

func parseKIDBytes(data []byte) *KID {
	// Match py-kids-vc's parse_kid, which opens the .KID with
	// errors="replace": invalid UTF-8 becomes U+FFFD via CPython's maximal-
	// subpart policy (see decodeUTF8Replace). Applied only here (the .KID read)
	// — the decomposed files we then write are valid UTF-8 and read back as-is,
	// just as the Python tool's strict reads of the component tree do.
	rawLines := splitKIDLines([]byte(decodeUTF8Replace(data)))
	result := &KID{Builds: map[string]*Build{}}
	state := "BEGIN"
	i := 0
	currentBuild := ""

	for i < len(rawLines) {
		line := rawLines[i]
		switch state {
		case "BEGIN":
			// Skip the two-line header preamble.
			i += 2
			state = "KIDSSS"
		case "KIDSSS":
			if !strings.HasPrefix(line, "**KIDS**:") {
				i++
				continue
			}
			payload := line[len("**KIDS**:"):]
			for _, name := range strings.Split(payload, "^") {
				if name != "" {
					result.InstallNames = append(result.InstallNames, name)
					result.build(name)
				}
			}
			i++
			if i < len(rawLines) && rawLines[i] == "" {
				i++
			}
			state = "INSTLNM"
		case "INSTLNM":
			if !strings.HasPrefix(line, "**INSTALL NAME**") {
				i++
				continue
			}
			i++
			if i >= len(rawLines) {
				return result
			}
			currentBuild = rawLines[i]
			result.build(currentBuild)
			i++
			state = "CONTENT"
		case "CONTENT":
			switch {
			case line == "**END**":
				i++
			case line == "**INSTALL NAME**":
				state = "INSTLNM"
			case strings.HasPrefix(line, "$END KID") || strings.HasPrefix(line, "$END TXT"):
				i++
			case strings.HasPrefix(line, "$KID") || strings.HasPrefix(line, "$TXT"):
				i++
			case !subscriptStart(line):
				i++
			default:
				if i+1 >= len(rawLines) {
					return result
				}
				subs := parseSubscriptLine(line)
				value := rawLines[i+1]
				result.build(currentBuild).Set(subs, value)
				i += 2
			}
		default:
			i++
		}
	}
	return result
}

// splitKIDLines splits raw file bytes into lines with trailing \r and \n
// stripped (Python's _strip_cr applied per readline). A trailing newline does
// not produce a final empty line, matching Python's file iteration.
func splitKIDLines(data []byte) []string {
	s := string(data)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	// A trailing "\n" yields a final "" element Python's iterator wouldn't
	// produce; drop it.
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	for i := range parts {
		parts[i] = strings.TrimRight(parts[i], "\r")
	}
	return parts
}
