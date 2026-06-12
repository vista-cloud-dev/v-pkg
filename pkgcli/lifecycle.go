package pkgcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
	"github.com/vista-cloud-dev/v-pkg/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/installspec"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// This file mounts the live KIDS lifecycle verbs — `v pkg install/verify/
// uninstall` (VSL M0a, tasks T0a.3/T0a.4) — on top of the install-script
// generators in internal/installspec. The waterline split (org CLAUDE.md): the
// KIDS knowledge (the generated M, the #9.7/#9.6/^XTMP shapes) lives HERE, above
// the line; reaching a live engine is engine-neutral `m` work and is consumed
// through the shared m-driver-sdk reference Client (mdriver.Client) — v-pkg never
// hand-rolls transport or runs a driver binary directly.
//
// Each verb generates a one-shot M script, stages it as a scratch routine over
// the driver `exec load`, runs its EN entry over `exec run` (one process → one
// symbol table, so XPDA survives the SETs the install needs), and reads the
// machine-readable `<<VPKG>>key=value` markers back off the captured device
// output. Live-proven end-to-end on the YDB FOIA engine (ZZSKEL), see
// docs/kids-installation-automation.md §7.1.

// Scratch routine names for the staged lifecycle scripts (ZV* keeps them in a
// local/throwaway namespace, clear of real VistA routines). They persist on the
// target after the run — harmless, and overwritten on the next invocation.
const (
	rtnInstall   = "ZVPKGINS"
	rtnVerify    = "ZVPKGVFY"
	rtnUninstall = "ZVPKGUNI"
)

// engineConn selects which engine driver to drive and over which transport — the
// same neutral knobs as `m vista` (vista_cmd.go). The connection itself
// (container/base-url, credentials) is read by the driver from its M_<ENGINE>_*
// environment, so it never appears here.
type engineConn struct {
	Engine    string `help:"Engine to reach: ydb or iris." enum:"ydb,iris" required:""`
	Transport string `help:"Driver transport: local | docker | remote." enum:"local,docker,remote" default:"remote"`
}

// client resolves the m-<engine> driver binary (driver-contract §4) and returns
// the shared reference Client — the seam's single transport (waterline rule 3).
func (e engineConn) client() (*mdriver.Client, error) {
	bin, err := mdriver.Locate(e.Engine, mdriver.DefaultLocateDeps())
	if err != nil {
		return nil, err
	}
	return mdriver.NewClient(bin, e.Engine, e.Transport, nil, nil), nil
}

func (e engineConn) noDriver(err error) *clikit.Error {
	return clikit.Fail(clikit.ExitRefused, "NO_DRIVER", err.Error(),
		"build the m-"+e.Engine+" driver (make build) or set M_"+strings.ToUpper(e.Engine)+"_BIN")
}

// wrapRoutine turns a direct-mode M script body into a loadable routine: a
// column-1 header + an EN label, every body line indented one space, and a
// trailing quit. The generated scripts assume a single persistent symbol table,
// which running EN^<rtn> in one driver process provides.
func wrapRoutine(name, body string) string {
	var b strings.Builder
	b.WriteString(name + " ;v-pkg generated lifecycle routine — safe to delete\n")
	b.WriteString("EN ;\n")
	for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		b.WriteString(" " + line + "\n")
	}
	b.WriteString(" Q\n")
	return b.String()
}

// parseMarkers extracts the `<<VPKG>>key=value` result lines from captured device
// output. A marker may appear mid-line (after KIDS' own device writes), so it is
// scanned anywhere in the stream and read to the next line break.
func parseMarkers(out string) map[string]string {
	m := map[string]string{}
	for _, seg := range strings.Split(out, installspec.ResultMarker)[1:] {
		line := seg
		if i := strings.IndexAny(seg, "\r\n"); i >= 0 {
			line = seg[:i]
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			m[k] = v
		}
	}
	return m
}

// runMScript stages body as the scratch routine rtn over the driver, runs its EN
// entry, and returns the parsed markers (plus the raw device output for logging).
// A staging/compile fault or a run-time engine fault is surfaced as a Go error;
// the markers reflect whatever the script managed to write first.
func runMScript(ctx context.Context, cl *mdriver.Client, rtn, body string) (map[string]string, string, error) {
	dir, err := os.MkdirTemp("", "vpkg-m-")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, rtn+".m")
	if err := os.WriteFile(path, []byte(wrapRoutine(rtn, body)), 0o600); err != nil {
		return nil, "", err
	}
	lr, err := cl.Load(ctx, []string{path})
	if err != nil {
		return nil, "", err
	}
	if lr.EngineError != nil {
		return nil, "", fmt.Errorf("stage %s: %s %s", rtn, lr.EngineError.Mnemonic, lr.EngineError.Text)
	}
	res, err := cl.ExecRun(ctx, "EN^"+rtn, nil)
	if err != nil {
		return nil, "", err
	}
	markers := parseMarkers(res.Stdout)
	if res.EngineError != nil {
		return markers, res.Stdout, fmt.Errorf("run EN^%s: %s %s", rtn, res.EngineError.Mnemonic, res.EngineError.Text)
	}
	return markers, res.Stdout, nil
}

// loadBuild parses a .KID and returns the single build's install name and data.
func loadBuild(kidFile string) (string, *kids.Build, error) {
	k, err := kids.ParseKID(kidFile)
	if err != nil {
		return "", nil, err
	}
	if len(k.InstallNames) != 1 {
		return "", nil, fmt.Errorf("expected exactly one build in %s, found %d", kidFile, len(k.InstallNames))
	}
	name := k.InstallNames[0]
	return name, k.Builds[name], nil
}

// --- install ----------------------------------------------------------------

type installResult struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Status    int    `json:"status"`          // #9.7 STATUS piece 9 (3 = Install Completed)
	Error     string `json:"error,omitempty"` // e.g. "already-installed"
}

// runInstall generates and runs the non-interactive install script, returning the
// outcome read from the #9.7 status marker (or the already-installed guard).
func runInstall(ctx context.Context, cl *mdriver.Client, name, header string, pairs []kids.Pair) (installResult, error) {
	markers, _, err := runMScript(ctx, cl, rtnInstall, installspec.InstallScript(name, header, pairs))
	if err != nil {
		return installResult{Name: name}, err
	}
	r := installResult{Name: name}
	if e := markers["error"]; e != "" {
		r.Error = e
		return r, nil
	}
	r.Status, _ = strconv.Atoi(strings.TrimSpace(markers["status"]))
	r.Installed = r.Status == 3
	return r, nil
}

type installCmd struct {
	engineConn
	KidFile string `arg:"" help:"Path to the built .KID transport file to install on the live engine."`
}

func (c *installCmd) Run(cc *clikit.Context) error {
	name, b, err := loadBuild(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	res, err := runInstall(context.Background(), cl, name, name+" via v pkg install", b.Pairs())
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "INSTALL_FAILED", err.Error(), "")
	}
	if err := cc.Result(res, func() {
		cc.Title("pkg install — " + c.Engine)
		switch {
		case res.Error != "":
			fmt.Fprintln(cc.Stdout, cc.Failure(res.Name+": "+res.Error))
		case res.Installed:
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("installed %s (#9.7 status %d)", res.Name, res.Status)))
		default:
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("%s did not complete (#9.7 status %d)", res.Name, res.Status)))
		}
	}); err != nil {
		return err
	}
	if res.Error != "" {
		return clikit.Fail(clikit.ExitRefused, "ALREADY_INSTALLED", res.Name+": "+res.Error,
			"uninstall it first, or bump the patch")
	}
	if !res.Installed {
		return clikit.Fail(clikit.ExitRuntime, "NOT_INSTALLED",
			fmt.Sprintf("%s did not reach Install Completed (status %d)", res.Name, res.Status), "")
	}
	return nil
}

// --- verify -----------------------------------------------------------------

type verifyResult struct {
	Name      string          `json:"name"`
	Installed bool            `json:"installed"`
	Status    int             `json:"status"`
	Routines  map[string]bool `json:"routines"`
}

// ok reports a fully verified install: #9.7 present + completed and every routine
// loaded.
func (r verifyResult) ok() bool {
	if !r.Installed || r.Status != 3 {
		return false
	}
	for _, present := range r.Routines {
		if !present {
			return false
		}
	}
	return true
}

func runVerify(ctx context.Context, cl *mdriver.Client, name string, routines []string) (verifyResult, error) {
	markers, _, err := runMScript(ctx, cl, rtnVerify, installspec.VerifyScript(name, routines))
	if err != nil {
		return verifyResult{Name: name}, err
	}
	r := verifyResult{Name: name, Routines: map[string]bool{}}
	r.Installed = strings.TrimSpace(markers["installed"]) == "1"
	r.Status, _ = strconv.Atoi(strings.TrimSpace(markers["status"]))
	for _, rt := range routines {
		r.Routines[rt] = strings.TrimSpace(markers["rtn:"+rt]) == "1"
	}
	return r, nil
}

type verifyCmd struct {
	engineConn
	KidFile string `arg:"" help:"Path to the .KID whose install to verify (its name + routines)."`
}

func (c *verifyCmd) Run(cc *clikit.Context) error {
	name, b, err := loadBuild(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	res, err := runVerify(context.Background(), cl, name, b.RoutineNames())
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "VERIFY_FAILED", err.Error(), "")
	}
	if err := cc.Result(res, func() {
		cc.Title("pkg verify — " + c.Engine)
		cc.KV(
			[2]string{"name", res.Name},
			[2]string{"installed", fmt.Sprint(res.Installed)},
			[2]string{"status", fmt.Sprint(res.Status)},
		)
		for rt, present := range res.Routines {
			mark := cc.Success("ok")
			if !present {
				mark = cc.Failure("missing")
			}
			fmt.Fprintf(cc.Stdout, "  %s %s\n", rt, mark)
		}
	}); err != nil {
		return err
	}
	if !res.ok() {
		return clikit.Fail(clikit.ExitCheck, "NOT_VERIFIED",
			fmt.Sprintf("%s is not fully installed", res.Name), "install it with `v pkg install`")
	}
	return nil
}

// --- uninstall --------------------------------------------------------------

type uninstallResult struct {
	Name        string `json:"name"`
	Uninstalled bool   `json:"uninstalled"`
}

func runUninstall(ctx context.Context, cl *mdriver.Client, name string, routines []string) (uninstallResult, error) {
	markers, _, err := runMScript(ctx, cl, rtnUninstall, installspec.UninstallScript(name, routines))
	if err != nil {
		return uninstallResult{Name: name}, err
	}
	return uninstallResult{Name: name, Uninstalled: strings.TrimSpace(markers["uninstalled"]) == "1"}, nil
}

type uninstallCmd struct {
	engineConn
	KidFile string `arg:"" help:"Path to the .KID whose install to reverse (routine-only back-out)."`
}

func (c *uninstallCmd) Run(cc *clikit.Context) error {
	name, b, err := loadBuild(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	res, err := runUninstall(context.Background(), cl, name, b.RoutineNames())
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", err.Error(), "")
	}
	if err := cc.Result(res, func() {
		cc.Title("pkg uninstall — " + c.Engine)
		if res.Uninstalled {
			fmt.Fprintln(cc.Stdout, cc.Success("uninstalled "+res.Name))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure("uninstall not confirmed for "+res.Name))
		}
	}); err != nil {
		return err
	}
	if !res.Uninstalled {
		return clikit.Fail(clikit.ExitRuntime, "NOT_UNINSTALLED",
			"uninstall of "+res.Name+" was not confirmed", "")
	}
	return nil
}
