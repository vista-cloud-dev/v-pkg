// Package wrapsplice host-side-patches the national RPC Broker routine XWBBRK to
// install — and exactly back out — the VSLRPCWRAP traffic-tap side-calls (the
// FU-5 / G-RPCHOOK wrap of the RPC→S3 tap workstream).
//
// It is pure text surgery over the routine's source lines, with NO engine
// contact: the caller reads stock XWBBRK off the engine through the driver, runs
// Splice here, and ships the patched routine back through the normal KIDS path
// (the transport monopoly). Keeping the splice logic off-engine and content-
// anchored is what makes the wrap re-validatable against each future XWB patch
// (FU-21): the anchors are matched by content and required to be unique, so a
// drifted XWBBRK is REFUSED rather than mis-patched.
//
// The seam (pinned on live foia(IRIS)+vehu(YDB), byte-identical — see the docs
// repo discoveries/fu-5b-callp-splice.md):
//
//	CALLP^XWBBRK:153  S:$L($G(XWBSEC)) ERR="-1^"_XWBSEC   <- req anchor (denial known)
//	         (+)      D req^VSLRPCWRAP                     <- inserted: one dir=req per RPC
//	         :155     IF '+ERR,(+S=0)!(+S>0) D
//	         :158     . D CAPI^XWBBRK2(.XWBP,...)          <- resp anchor (the dispatch)
//	         (+)      . D resp^VSLRPCWRAP                  <- inserted: dir=resp, success path
//
// The req side-call is top-level (fires for every RPC, incl. a CHKPRMIT denial,
// which is known only here); the resp side-call is inserted at the SAME dot level
// as the dispatch so it stays inside the `:155 IF…D` success block (success path
// only). Both call VSLRPCWRAP, which owns the FU-4 naked-indicator fence and the
// $$captureOn gate — so the side-calls need no guard, and the splice line stays a
// plain DO (a global-flag guard in the broker line would move the naked indicator
// before the fence runs). Every inserted line carries Marker, so Splice is
// idempotent-detectable and Unsplice is an exact inverse (restores stock).
package wrapsplice

import (
	"fmt"
	"strings"
)

// Marker tags every line this package inserts. It makes Splice idempotent-
// detectable (IsSpliced) and Unsplice an exact inverse — Unsplice removes only
// Marker lines, so a Splice→Unsplice round-trip restores stock byte-for-byte.
const Marker = ";VSLTAPW"

// The two side-calls. No args: VSLRPCWRAP.req/resp read the broker's process-
// scope vars directly and own the fence + gate.
const (
	reqCall  = "D req^VSLRPCWRAP"
	respCall = "D resp^VSLRPCWRAP"
)

// Anchors. The req anchor is matched by EXACT trimmed content (the post-CHKPRMIT
// denial line — XWBSEC also appears on nearby lines, so an exact match is what
// keeps it unique). The resp anchor is matched by the dispatch substring; the
// uniqueness requirement (exactly one) is the drift guard.
const (
	reqAnchorText    = `S:$L($G(XWBSEC)) ERR="-1^"_XWBSEC`
	respAnchorSubstr = `D CAPI^XWBBRK2(`
)

// Validate locates the two splice anchors in stock XWBBRK source and confirms
// each occurs exactly once. A count other than 1 means the routine has drifted
// from the pinned seam — Validate REFUSES (this is the FU-21 per-XWB-patch
// re-validation gate), returning an error rather than risk a mis-placed splice.
// It returns the 0-based indices of the req and resp anchor lines.
func Validate(src []string) (reqIdx, respIdx int, err error) {
	reqIdx, respIdx = -1, -1
	reqN, respN := 0, 0
	for i, line := range src {
		t := strings.TrimSpace(line)
		if t == reqAnchorText {
			reqN++
			reqIdx = i
		}
		if strings.Contains(t, respAnchorSubstr) {
			respN++
			respIdx = i
		}
	}
	if reqN != 1 {
		return -1, -1, fmt.Errorf("wrapsplice: req anchor %q found %d times (need exactly 1) — XWBBRK drifted from the pinned seam; re-validate the splice (FU-21)", reqAnchorText, reqN)
	}
	if respN != 1 {
		return -1, -1, fmt.Errorf("wrapsplice: resp anchor %q found %d times (need exactly 1) — XWBBRK drifted from the pinned seam; re-validate the splice (FU-21)", respAnchorSubstr, respN)
	}
	return reqIdx, respIdx, nil
}

// IsSpliced reports whether src already carries the wrap (any Marker line).
func IsSpliced(src []string) bool {
	for _, line := range src {
		if strings.Contains(line, Marker) {
			return true
		}
	}
	return false
}

// Splice inserts the two fenced side-calls into stock XWBBRK source and returns
// the patched lines. It refuses an already-spliced source (idempotency guard) and
// refuses a drifted one (via Validate). Each inserted line copies its anchor's
// leading whitespace+dot prefix, so the req call lands at top level after the
// denial line and the resp call lands at the dispatch's dot level (inside the
// success block).
func Splice(src []string) ([]string, error) {
	if IsSpliced(src) {
		return nil, fmt.Errorf("wrapsplice: source already spliced (Marker %q present) — Unsplice first", Marker)
	}
	reqIdx, respIdx, err := Validate(src)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(src)+2)
	for i, line := range src {
		out = append(out, line)
		if i == reqIdx {
			out = append(out, linePrefix(line)+reqCall+" "+Marker)
		}
		if i == respIdx {
			out = append(out, linePrefix(line)+respCall+" "+Marker)
		}
	}
	return out, nil
}

// Unsplice removes every Marker line, restoring stock XWBBRK source. Because
// Splice only ever ADDS Marker lines, this is its exact inverse. It refuses a
// source that is not spliced (nothing to back out).
func Unsplice(src []string) ([]string, error) {
	if !IsSpliced(src) {
		return nil, fmt.Errorf("wrapsplice: source is not spliced (no Marker %q) — nothing to back out", Marker)
	}
	out := make([]string, 0, len(src))
	for _, line := range src {
		if strings.Contains(line, Marker) {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// linePrefix returns the leading run of M whitespace and dots before the first
// command on the line (e.g. " " for a top-level line, " . " for a single-dot
// block line). The inserted side-call copies this so it sits at the anchor's
// exact statement level.
func linePrefix(line string) string {
	for i, r := range line {
		if r != ' ' && r != '\t' && r != '.' {
			return line[:i]
		}
	}
	return line
}
