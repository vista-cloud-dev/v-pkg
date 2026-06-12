package kids

import "strings"

// This file exposes the M-source rendering primitives the install-script
// generator (installspec) needs: a build's (subscript, value) pairs become
// `SET ^XTMP("XPDI",XPDA,<subs>)=<value>` statements that populate the KIDS
// transport global directly (the non-interactive load path, T0a.3). They live in
// package kids because formatSubscript is unexported here.

// MString renders v as an M string literal, doubling embedded quotes:
// `a"b` → `"a""b"`. It does NOT chunk for the 255-char M line limit — callers
// that may emit long routine lines must guard or split (see MaxMLine).
func MString(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

// MRef returns the M global reference for this subscript tail, opened by
// rootOpen. rootOpen must end at an open paren plus any fixed leading
// subscripts — e.g. `^XTMP("XPDI",XPDA,` — and MRef supplies the closing paren.
// So Subs{"RTN","ZZSKEL",5,0}.MRef(`^XTMP("XPDI",XPDA,`) yields
// `^XTMP("XPDI",XPDA,"RTN","ZZSKEL",5,0)`.
func (s Subs) MRef(rootOpen string) string {
	return rootOpen + formatSubscript(s)
}
