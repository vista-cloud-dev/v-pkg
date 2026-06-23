package pkgcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vista-cloud-dev/v-pkg/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
	"github.com/vista-cloud-dev/v-pkg/internal/wrapsplice"
)

// wrap-rpc installs — and backs out — the RPC→S3 traffic-tap side-calls in the
// national broker routine CALLP^XWBBRK (the FU-5 / G-RPCHOOK wrap). The flow,
// per the owner's delivery decision (Option A): read stock XWBBRK off the live
// engine through the driver (readRoutineSource), splice/un-splice host-side
// (internal/wrapsplice — content-anchored, re-validated per XWB patch = FU-21),
// and ship the patched/stock routine back through the proven KIDS install path
// (runInstall). The two side-calls call VSLRPCWRAP (shipped by v-stdlib), which
// owns the fence + gate.
//
// install/backout PREVIEW by default — they read the routine (read-only), splice,
// build the KIDS pairs, and write the artifacts for inspection WITHOUT touching
// the engine. The live install (which overwrites a national routine) only runs
// under --commit, and is the deliberately-gated, hard-to-reverse step.

// The KIDS install identities for the wrap and its back-out. Host-tool / install
// names are not under DBA namespace governance (that governs M routine/global
// names inside VistA); these name the #9.7 INSTALL entries the ship creates.
const (
	wrapInstallName   = "VSLTAP RPC WRAP 1.0"
	wrapBackoutName   = "VSLTAP RPC WRAP BACKOUT 1.0"
	wrapNamespace     = "VSLTAP"
	wrapDefaultRtn    = "XWBBRK"
	wrapInstallHeader = "VSLTAP RPC traffic-tap wrap (FU-5) via v pkg wrap-rpc"
)

type wrapRPCCmd struct {
	Status  wrapStatusCmd  `cmd:"" help:"Report whether the XWBBRK traffic-tap wrap is installed (read-only)."`
	Install wrapInstallCmd `cmd:"" help:"Install the traffic-tap wrap into CALLP^XWBBRK (preview by default; --commit ships it via KIDS)."`
	Backout wrapBackoutCmd `cmd:"" help:"Remove the wrap, restoring stock CALLP^XWBBRK (preview by default; --commit ships it via KIDS)."`
}

// wrapBase carries the engine connection + the target routine, shared by the
// three verbs. Routine defaults to XWBBRK (the pinned seam) but is overridable so
// FU-21 re-validation can target a renamed/re-routed broker if a future XWB patch
// moves the dispatch.
type wrapBase struct {
	engineConn
	Routine string `help:"the broker routine to (un)patch" default:"XWBBRK"`
}

func (b wrapBase) routine() string {
	if b.Routine == "" {
		return wrapDefaultRtn
	}
	return b.Routine
}

// readStock reads the target routine off the engine over the driver.
func (b wrapBase) readStock(ctx context.Context) ([]string, *clikit.Error) {
	cl, err := b.client()
	if err != nil {
		return nil, b.noDriver(err)
	}
	src, err := readRoutineSource(ctx, cl, b.routine())
	if err != nil {
		return nil, clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(),
			"confirm the routine exists on the engine and the driver connection is configured")
	}
	return src, nil
}

// --- status -----------------------------------------------------------------

type wrapStatusResult struct {
	Routine   string `json:"routine"`
	Lines     int    `json:"lines"`
	Spliced   bool   `json:"spliced"`               // the wrap is currently installed
	AnchorsOK bool   `json:"anchorsOk"`             // the splice points are present + unique
	Anchor    string `json:"anchorError,omitempty"` // why the anchors are not OK (drift; FU-21)
}

type wrapStatusCmd struct{ wrapBase }

func (c *wrapStatusCmd) Run(cc *clikit.Context) error {
	src, ferr := c.readStock(context.Background())
	if ferr != nil {
		return ferr
	}
	res := wrapStatusResult{Routine: c.routine(), Lines: len(src), Spliced: wrapsplice.IsSpliced(src)}
	if _, _, err := wrapsplice.Validate(src); err != nil {
		res.Anchor = err.Error()
	} else {
		res.AnchorsOK = true
	}
	return cc.Result(res, func() {
		cc.Title("pkg wrap-rpc status — " + c.Engine)
		state := "stock (wrap NOT installed)"
		if res.Spliced {
			state = "wrap INSTALLED"
		}
		fmt.Fprintf(cc.Stdout, "%s %s: %s, %d lines\n", cc.Success("ok"), cc.Accent(res.Routine), state, res.Lines)
		if res.AnchorsOK {
			fmt.Fprintln(cc.Stdout, cc.Success("splice anchors present + unique (re-pin OK)"))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure("splice anchors NOT clean (FU-21 re-pin needed): "+res.Anchor))
		}
	})
}

// --- install / backout ------------------------------------------------------

type wrapActionResult struct {
	Action    string `json:"action"` // "install" | "backout"
	Routine   string `json:"routine"`
	Committed bool   `json:"committed"` // false = preview only (engine untouched)
	Pairs     int    `json:"kidsPairs"` // KIDS transport nodes built
	Out       string `json:"out,omitempty"`
	Installed bool   `json:"installed,omitempty"`
	Status    int    `json:"status,omitempty"`
}

type wrapInstallCmd struct {
	wrapBase
	Out    string `help:"write the patched routine (.m) and its KIDS (.kids) preview to this path prefix"`
	Commit bool   `help:"DESTRUCTIVE: install the patched routine on the live engine via KIDS (overwrites the national routine). Default off = preview only."`
}

func (c *wrapInstallCmd) Run(cc *clikit.Context) error {
	return runWrapAction(cc, c.wrapBase, "install", wrapInstallName, c.Out, c.Commit,
		func(src []string) ([]string, error) { return wrapsplice.Splice(src) })
}

type wrapBackoutCmd struct {
	wrapBase
	Out    string `help:"write the restored stock routine (.m) and its KIDS (.kids) preview to this path prefix"`
	Commit bool   `help:"DESTRUCTIVE: restore stock on the live engine via KIDS. Default off = preview only."`
}

func (c *wrapBackoutCmd) Run(cc *clikit.Context) error {
	return runWrapAction(cc, c.wrapBase, "backout", wrapBackoutName, c.Out, c.Commit,
		func(src []string) ([]string, error) { return wrapsplice.Unsplice(src) })
}

// runWrapAction is the shared install/backout body: read stock → transform
// (Splice/Unsplice) → build KIDS pairs → preview (default) or, under --commit,
// ship through the proven runInstall path. The engine READ always happens (it is
// read-only); the engine WRITE happens only with --commit.
func runWrapAction(cc *clikit.Context, b wrapBase, action, installName, out string, commit bool, transform func([]string) ([]string, error)) error {
	ctx := context.Background()
	src, ferr := b.readStock(ctx)
	if ferr != nil {
		return ferr
	}
	next, err := transform(src)
	if err != nil {
		return clikit.Fail(clikit.ExitRefused, "SPLICE_REFUSED", err.Error(),
			"run `v pkg wrap-rpc status` to inspect the routine state")
	}
	pairs := kids.MakeBuildPairs(kids.BuildInput{
		InstallName: installName,
		Namespace:   wrapNamespace,
		Routines:    []kids.RoutineSrc{{Name: b.routine(), Lines: next}},
	})
	res := wrapActionResult{Action: action, Routine: b.routine(), Pairs: len(pairs)}

	if out != "" {
		if werr := writeWrapPreview(out, next, installName, pairs); werr != nil {
			return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", werr.Error(), "")
		}
		res.Out = out
	}

	if !commit {
		// preview only — the engine was read but never written.
		return cc.Result(res, func() {
			cc.Title("pkg wrap-rpc " + action + " (preview) — " + b.Engine)
			fmt.Fprintf(cc.Stdout, "%s previewed %s on %s — %d KIDS node(s); engine NOT modified (pass --commit to install)\n",
				cc.Success("ok"), cc.Accent(b.routine()), cc.Accent(installName), res.Pairs)
			if res.Out != "" {
				fmt.Fprintf(cc.Stdout, "  wrote preview → %s(.m/.kids)\n", res.Out)
			}
		})
	}

	// --- live install (the gated, hard-to-reverse step) ---
	cl, derr := b.client()
	if derr != nil {
		return b.noDriver(derr)
	}
	ir, ierr := runInstall(ctx, cl, installName, wrapInstallHeader, pairs)
	if ierr != nil {
		return clikit.Fail(clikit.ExitRuntime, "INSTALL_FAILED", ierr.Error(), "")
	}
	res.Committed = true
	res.Installed = ir.Installed
	res.Status = ir.Status
	if err := cc.Result(res, func() {
		cc.Title("pkg wrap-rpc " + action + " — " + b.Engine)
		if ir.Installed {
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("%s on %s (#9.7 status %d)", action, b.routine(), ir.Status)))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("%s did not complete (#9.7 status %d): %s", action, ir.Status, ir.Error)))
		}
	}); err != nil {
		return err
	}
	if !ir.Installed {
		return clikit.Fail(clikit.ExitRuntime, "NOT_INSTALLED",
			fmt.Sprintf("%s did not reach Install Completed (status %d)", action, ir.Status), "")
	}
	return nil
}

// writeWrapPreview writes the transformed routine (.m) and its KIDS (.kids) under
// the given path prefix, for inspection before a --commit install.
func writeWrapPreview(prefix string, lines []string, installName string, pairs []kids.Pair) error {
	if dir := filepath.Dir(prefix); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	var body string
	for _, l := range lines {
		body += l + "\n"
	}
	if err := os.WriteFile(prefix+".m", []byte(body), 0o600); err != nil {
		return err
	}
	return kids.WriteKID([]string{installName}, map[string][]kids.Pair{installName: pairs}, prefix+".kids")
}
