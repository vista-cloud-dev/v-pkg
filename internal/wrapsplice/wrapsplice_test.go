package wrapsplice

import (
	"slices"
	"strings"
	"testing"
)

// stockCALLP is the verbatim CALLP^XWBBRK section read from live foia(IRIS) —
// byte-identical on vehu(YDB) — lines 140-162 (one leading space on body lines;
// " . " on the single-dot block lines). It is the patch target. Note XWBSEC
// appears on three lines (149/152/153) and "CAPI" on several, so the anchors
// must be surgical.
var stockCALLP = []string{
	`CALLP(XWBP,P,DEBUG) ;make API call using Protocol string`,
	` ;ERR will be 0 or "-1^text"`,
	` N ERR,S`,
	` S ERR=0`,
	` IF '$D(DEBUG) S DEBUG=0`,
	` ;IF 'DEBUG D:$D(XRTL) T0^%ZOSV ;start RTL`,
	` S ERR=$$PRSP(P)`,
	` IF '+ERR S ERR=$$PRSM(XWB(0,"MESG"))`,
	` IF '+ERR S ERR=$$PRSA(XWB(1,"TEXT")) I $G(XWB(2,"CAPI"))="XUS SET SHARED" S XWBSHARE=1 Q`,
	` I +ERR S XWBSEC=$P(ERR,U,2) ;P10 -- dpc`,
	` IF '+ERR S S=$$PRSB(XWB(2,"PARM"))`,
	` ;Check OK`,
	` I '+ERR D CHKPRMIT^XWBSEC(XWB(2,"CAPI")) ;checks if RPC allowed to run`,
	` S:$L($G(XWBSEC)) ERR="-1^"_XWBSEC`,
	` ;IF 'DEBUG S:$D(XRT0) XRTN="RPC BROKER READ/PARSE" D:$D(XRT0) T1^%ZOSV ;stop RTL`,
	` IF '+ERR,(+S=0)!(+S>0) D`,
	` . ;Logging`,
	` . I $G(XWBDEBUG)>1 D LOG^XWBDLOG("RPC: "_XWB(2,"CAPI"))`,
	` . D CAPI^XWBBRK2(.XWBP,XWB(2,"RTAG"),XWB(2,"RNAM"),S)`,
	` E  D CLRBUF ;p10`,
	` IF 'DEBUG K XWB`,
	` IF $D(XWBARY) K @XWBARY,XWBARY`,
	` Q`,
}

const (
	reqLine  = ` D req^VSLRPCWRAP ;VSLTAPW`    // top level (one leading space)
	respLine = ` . D resp^VSLRPCWRAP ;VSLTAPW` // dot level of the dispatch
)

func TestValidate_FindsBothAnchorsUniquely(t *testing.T) {
	reqIdx, respIdx, err := Validate(stockCALLP)
	if err != nil {
		t.Fatalf("Validate stock: unexpected error: %v", err)
	}
	if got := strings.TrimSpace(stockCALLP[reqIdx]); got != reqAnchorText {
		t.Errorf("req anchor index %d = %q, want the denial line", reqIdx, got)
	}
	if !strings.Contains(stockCALLP[respIdx], respAnchorSubstr) {
		t.Errorf("resp anchor index %d = %q, want the dispatch line", respIdx, stockCALLP[respIdx])
	}
	if respIdx <= reqIdx {
		t.Errorf("resp anchor (%d) must come after the req anchor (%d)", respIdx, reqIdx)
	}
}

func TestSplice_InsertsBothSideCallsAtTheRightPlaces(t *testing.T) {
	out, err := Splice(stockCALLP)
	if err != nil {
		t.Fatalf("Splice: %v", err)
	}
	if len(out) != len(stockCALLP)+2 {
		t.Fatalf("Splice added %d lines, want exactly 2", len(out)-len(stockCALLP))
	}
	// the req side-call lands immediately after the denial line, top level.
	denialAt := slices.IndexFunc(out, func(s string) bool { return strings.TrimSpace(s) == reqAnchorText })
	if denialAt < 0 || out[denialAt+1] != reqLine {
		t.Errorf("req side-call not directly after the denial line; got next = %q, want %q", out[denialAt+1], reqLine)
	}
	// the resp side-call lands immediately after the dispatch, at its dot level.
	dispatchAt := slices.IndexFunc(out, func(s string) bool { return strings.Contains(s, respAnchorSubstr) })
	if dispatchAt < 0 || out[dispatchAt+1] != respLine {
		t.Errorf("resp side-call not directly after the dispatch; got next = %q, want %q", out[dispatchAt+1], respLine)
	}
}

func TestSplice_Surgical_OnlyTheTwoNeighborsChange(t *testing.T) {
	out, err := Splice(stockCALLP)
	if err != nil {
		t.Fatalf("Splice: %v", err)
	}
	// Every stock line must survive verbatim and in order; the only additions are
	// the two Marker lines. (Guards against touching the XWBSEC mentions on
	// 149/152 or the "CAPI" mentions on 148/152/157.)
	withoutMarkers := make([]string, 0, len(out))
	markerCount := 0
	for _, l := range out {
		if strings.Contains(l, Marker) {
			markerCount++
			continue
		}
		withoutMarkers = append(withoutMarkers, l)
	}
	if markerCount != 2 {
		t.Errorf("got %d Marker lines, want 2", markerCount)
	}
	if !slices.Equal(withoutMarkers, stockCALLP) {
		t.Errorf("removing the Marker lines did not reproduce stock — the splice was not surgical")
	}
}

func TestUnsplice_RestoresStockByteForByte(t *testing.T) {
	spliced, err := Splice(stockCALLP)
	if err != nil {
		t.Fatalf("Splice: %v", err)
	}
	restored, err := Unsplice(spliced)
	if err != nil {
		t.Fatalf("Unsplice: %v", err)
	}
	if !slices.Equal(restored, stockCALLP) {
		t.Errorf("Splice→Unsplice did not restore stock byte-for-byte\n got: %#v\nwant: %#v", restored, stockCALLP)
	}
}

func TestSplice_RefusesAlreadySpliced(t *testing.T) {
	spliced, err := Splice(stockCALLP)
	if err != nil {
		t.Fatalf("Splice: %v", err)
	}
	if _, err := Splice(spliced); err == nil {
		t.Error("Splice of an already-spliced source should error (idempotency guard), got nil")
	}
	if !IsSpliced(spliced) {
		t.Error("IsSpliced should report true for a spliced source")
	}
	if IsSpliced(stockCALLP) {
		t.Error("IsSpliced should report false for stock source")
	}
}

func TestUnsplice_RefusesUnsplicedSource(t *testing.T) {
	if _, err := Unsplice(stockCALLP); err == nil {
		t.Error("Unsplice of a non-spliced source should error, got nil")
	}
}

func TestValidate_RefusesDrift(t *testing.T) {
	// FU-21: a missing or duplicated anchor means XWBBRK drifted — refuse.
	t.Run("missing req anchor", func(t *testing.T) {
		drift := slices.Clone(stockCALLP)
		drift = slices.DeleteFunc(drift, func(s string) bool { return strings.TrimSpace(s) == reqAnchorText })
		if _, _, err := Validate(drift); err == nil {
			t.Error("Validate should refuse a source missing the req anchor")
		}
		if _, err := Splice(drift); err == nil {
			t.Error("Splice should refuse a drifted source")
		}
	})
	t.Run("duplicated dispatch anchor", func(t *testing.T) {
		drift := slices.Clone(stockCALLP)
		drift = append(drift, ` . D CAPI^XWBBRK2(.XWBP,"X","Y",S)`) // a second dispatch -> ambiguous
		if _, _, err := Validate(drift); err == nil {
			t.Error("Validate should refuse a source with two dispatch anchors")
		}
	})
}
