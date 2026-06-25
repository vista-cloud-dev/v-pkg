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
// national broker routine CALLP^XWBBRK (the FU-5 / G-RPCHOOK wrap). The ONLY
// wrap-specific step is producing the patched routine: read stock XWBBRK off the
// live engine through the driver (readRoutineSource), splice host-side
// (internal/wrapsplice — content-anchored, re-validated per XWB patch = FU-21).
// Everything else — applying the patch, capturing the stock pre-image, and
// reversing it — goes STRICTLY through the generic v-pkg lifecycle, not a bespoke
// installer:
//
//   install --commit → liveInstall (the same class-aware path `v pkg install`
//                      uses): probe XWBBRK, snapshot its stock pre-image, ship the
//                      patch via runInstall. The wrap is class-1 pure-overwrite of
//                      one routine, so the snapshot is a COMPLETE undo.
//   backout --commit → liveRestore (the same path `v pkg restore` /
//                      `v pkg uninstall --restore` use): re-install the captured
//                      stock pre-image. No Unsplice, no second back-out KID — the
//                      reversal is install-of-the-pre-image, byte-exact stock.
//
// install/backout PREVIEW by default — they read the routine / the pre-image
// (read-only) and report WITHOUT touching the engine. The live step (which
// overwrites a national routine) only runs under --commit.

// The KIDS install identity for the wrap. Host-tool / install names are not under
// DBA namespace governance (that governs M routine/global names inside VistA);
// this names the #9.7 INSTALL entry the ship creates. The stock pre-image the
// install captures installs under "<name> PREIMAGE" (snapshotName), which backout
// restores — so there is no separate hand-rolled back-out identity.
const (
	wrapInstallName   = "VSLTAP RPC WRAP 1.0"
	wrapNamespace     = "VSLTAP"
	wrapDefaultRtn    = "XWBBRK"
	wrapInstallHeader = "VSLTAP RPC traffic-tap wrap (FU-5) via v pkg wrap-rpc"
	// wrapPreimageFile is the conventional sidecar `install --commit` writes the
	// captured stock pre-image to and `backout --commit` restores from, so the
	// install/backout pair needs no explicit path. Overridable with
	// --snapshot / --restore.
	wrapPreimageFile = "vsltap-rpc-wrap.preimage.kids"
)

type wrapRPCCmd struct {
	Status  wrapStatusCmd  `cmd:"" help:"Report whether the XWBBRK traffic-tap wrap is installed (read-only)."`
	Install wrapInstallCmd `cmd:"" help:"Install the traffic-tap wrap into CALLP^XWBBRK (preview by default; --commit ships it via the generic lifecycle, snapshotting stock first)."`
	Backout wrapBackoutCmd `cmd:"" help:"Remove the wrap by restoring the captured stock pre-image (preview by default; --commit restores via the generic lifecycle)."`
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
	Committed bool   `json:"committed"`          // false = preview only (engine untouched)
	Pairs     int    `json:"kidsPairs"`          // KIDS transport nodes built
	Out       string `json:"out,omitempty"`      // preview artifact prefix (install)
	Snapshot  string `json:"snapshot,omitempty"` // stock pre-image .KID captured (install --commit)
	Restore   string `json:"restore,omitempty"`  // stock pre-image .KID restored (backout)
	Installed bool   `json:"installed,omitempty"`
	Status    int    `json:"status,omitempty"`
}

type wrapInstallCmd struct {
	wrapBase
	Out            string `help:"write the patched routine (.m) and its KIDS (.kids) preview to this path prefix"`
	Snapshot       string `help:"capture the stock pre-image of the broker routine to this .KID before patching (default: the sidecar wrap-rpc backout auto-detects)."`
	AllowOverwrite bool   `help:"patch the broker WITHOUT capturing a stock pre-image (UNSAFE: wrap-rpc backout cannot then restore it)."`
	Commit         bool   `help:"DESTRUCTIVE: install the patched routine on the live engine via KIDS (overwrites the national routine). Default off = preview only."`
}

func (c *wrapInstallCmd) Run(cc *clikit.Context) error {
	ctx := context.Background()
	src, ferr := c.readStock(ctx)
	if ferr != nil {
		return ferr
	}
	next, err := wrapsplice.Splice(src)
	if err != nil {
		return clikit.Fail(clikit.ExitRefused, "SPLICE_REFUSED", err.Error(),
			"run `v pkg wrap-rpc status` to inspect the routine state")
	}
	pairs := kids.MakeBuildPairs(kids.BuildInput{
		InstallName: wrapInstallName,
		Namespace:   wrapNamespace,
		Routines:    []kids.RoutineSrc{{Name: c.routine(), Lines: next}},
	})
	res := wrapActionResult{Action: "install", Routine: c.routine(), Pairs: len(pairs)}

	if c.Out != "" {
		if werr := writeWrapPreview(c.Out, next, wrapInstallName, pairs); werr != nil {
			return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", werr.Error(), "")
		}
		res.Out = c.Out
	}

	if !c.Commit {
		// preview only — the engine was read but never written.
		return cc.Result(res, func() {
			cc.Title("pkg wrap-rpc install (preview) — " + c.Engine)
			fmt.Fprintf(cc.Stdout, "%s previewed %s on %s — %d KIDS node(s); engine NOT modified (pass --commit to install)\n",
				cc.Success("ok"), cc.Accent(c.routine()), cc.Accent(wrapInstallName), res.Pairs)
			if res.Out != "" {
				fmt.Fprintf(cc.Stdout, "  wrote preview → %s(.m/.kids)\n", res.Out)
			}
		})
	}

	// --- live install: STRICTLY the generic class-aware lifecycle (liveInstall),
	// which snapshots the stock pre-image (so backout = restore) and ships the
	// patch via the proven runInstall path. No bespoke wrap installer.
	cl, derr := c.client()
	if derr != nil {
		return c.noDriver(derr)
	}
	snapshot := c.Snapshot
	if snapshot == "" {
		snapshot = wrapPreimageFile
	}
	rep, ierr := liveInstall(ctx, cl, liveInstallInput{
		name: wrapInstallName, header: wrapInstallHeader, className: kids.ClassPureOverwrite.String(),
		routineNames: []string{c.routine()}, pairs: pairs,
		snapshotPath: snapshot, allowOverwrite: c.AllowOverwrite,
	})
	if ierr != nil {
		return ierr
	}
	res.Committed = true
	res.Installed = rep.Installed
	res.Status = rep.Status
	res.Snapshot = rep.Snapshot
	if err := cc.Result(res, func() {
		cc.Title("pkg wrap-rpc install — " + c.Engine)
		if res.Snapshot != "" {
			fmt.Fprintf(cc.Stdout, "  stock pre-image captured → %s\n", cc.Accent(res.Snapshot))
		}
		if rep.Installed {
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("install on %s (#9.7 status %d)", c.routine(), rep.Status)))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("install did not complete (#9.7 status %d): %s", rep.Status, rep.Error)))
		}
	}); err != nil {
		return err
	}
	if !rep.Installed {
		return clikit.Fail(clikit.ExitRuntime, "NOT_INSTALLED",
			fmt.Sprintf("install did not reach Install Completed (status %d)", rep.Status), "")
	}
	return nil
}

type wrapBackoutCmd struct {
	wrapBase
	Restore string `help:"the stock pre-image .KID to restore (default: the sidecar wrap-rpc install --commit captured)."`
	Commit  bool   `help:"DESTRUCTIVE: restore stock on the live engine via KIDS. Default off = preview only."`
}

func (c *wrapBackoutCmd) Run(cc *clikit.Context) error {
	ctx := context.Background()
	restore := c.Restore
	if restore == "" {
		restore = wrapPreimageFile
	}
	if !fileExists(restore) {
		return clikit.Fail(clikit.ExitRefused, "NO_PREIMAGE",
			"no stock pre-image found at "+restore,
			"pass --restore <preimage.kids>; wrap-rpc install --commit captures one automatically (or use `v pkg snapshot`)")
	}

	// Load the pre-image to report what we would restore (read-only).
	_, b, lerr := loadBuild(restore)
	if lerr != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", lerr.Error(), "")
	}
	res := wrapActionResult{Action: "backout", Routine: c.routine(), Pairs: len(b.Pairs()), Restore: restore}

	if !c.Commit {
		return cc.Result(res, func() {
			cc.Title("pkg wrap-rpc backout (preview) — " + c.Engine)
			fmt.Fprintf(cc.Stdout, "%s would restore stock %s from %s (%d routine(s)); engine NOT modified (pass --commit)\n",
				cc.Success("ok"), cc.Accent(c.routine()), cc.Accent(restore), len(b.RoutineNames()))
		})
	}

	// --- live backout: STRICTLY the generic reversal (liveRestore) — re-install
	// the captured stock pre-image. No Unsplice, no bespoke back-out KID.
	cl, derr := c.client()
	if derr != nil {
		return c.noDriver(derr)
	}
	_, done, status, _, rerr := liveRestore(ctx, cl, restore, "v pkg wrap-rpc backout", false)
	if rerr != nil {
		return rerr
	}
	res.Committed = true
	res.Installed = done
	res.Status = status
	if err := cc.Result(res, func() {
		cc.Title("pkg wrap-rpc backout — " + c.Engine)
		if done {
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("backout (restored stock %s) (#9.7 status %d)", c.routine(), status)))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("backout did not complete (#9.7 status %d)", status)))
		}
	}); err != nil {
		return err
	}
	if !done {
		return clikit.Fail(clikit.ExitRuntime, "NOT_RESTORED",
			fmt.Sprintf("backout did not reach Install Completed (status %d)", status), "")
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
