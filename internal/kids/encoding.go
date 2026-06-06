package kids

import "strings"

// decodeUTF8Replace decodes bytes as UTF-8, replacing each ill-formed
// subsequence with U+FFFD — byte-for-byte identical to CPython's
// open(..., encoding="utf-8", errors="replace"), which py-kids-vc uses to read
// .KID files. It follows the Unicode "substitution of maximal subparts" policy:
// one U+FFFD per maximal invalid subpart (so two adjacent bad lead bytes yield
// two U+FFFD, while a truncated multi-byte sequence yields one). Go's
// strings.ToValidUTF8 collapses runs and so does NOT match CPython here.
func decodeUTF8Replace(data []byte) string {
	const repl = "�"
	var b strings.Builder
	b.Grow(len(data))
	n := len(data)
	for i := 0; i < n; {
		c := data[i]
		if c < 0x80 {
			b.WriteByte(c)
			i++
			continue
		}
		lo1, hi1, nc := utf8LeadSpec(c)
		if nc == 0 {
			// Invalid lead byte (stray continuation, 0xC0/0xC1, 0xF5..0xFF).
			b.WriteString(repl)
			i++
			continue
		}
		consumed := 1 // the lead byte
		ok := true
		for j := 1; j <= nc; j++ {
			lo, hi := byte(0x80), byte(0xBF)
			if j == 1 {
				lo, hi = lo1, hi1
			}
			if i+j >= n || data[i+j] < lo || data[i+j] > hi {
				ok = false
				break
			}
			consumed++
		}
		if ok {
			b.Write(data[i : i+nc+1]) // a well-formed sequence; preserve bytes
			i += nc + 1
		} else {
			b.WriteString(repl) // maximal subpart = the bytes validated so far
			i += consumed
		}
	}
	return b.String()
}

// utf8LeadSpec returns the valid byte range for the FIRST continuation byte and
// the number of continuation bytes a lead byte expects. nc==0 marks an invalid
// lead. The first-continuation ranges encode the over-long / surrogate / range
// constraints (E0/ED/F0/F4 specials); later continuations are always 0x80..0xBF.
func utf8LeadSpec(c byte) (lo1, hi1 byte, nc int) {
	switch {
	case c >= 0xC2 && c <= 0xDF:
		return 0x80, 0xBF, 1
	case c == 0xE0:
		return 0xA0, 0xBF, 2
	case c >= 0xE1 && c <= 0xEC:
		return 0x80, 0xBF, 2
	case c == 0xED:
		return 0x80, 0x9F, 2
	case c >= 0xEE && c <= 0xEF:
		return 0x80, 0xBF, 2
	case c == 0xF0:
		return 0x90, 0xBF, 3
	case c >= 0xF1 && c <= 0xF3:
		return 0x80, 0xBF, 3
	case c == 0xF4:
		return 0x80, 0x8F, 3
	default:
		return 0, 0, 0
	}
}
