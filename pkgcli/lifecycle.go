package pkgcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vista-cloud-dev/clikit"
	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
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
	rtnRead      = "ZVPKGRD"
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
	// A driver that could not stage (e.g. no routine source directory configured)
	// may report no fault yet load nothing; running EN^<rtn> would then fail with a
	// confusing link error. Refuse up front so the cause is the staging, not the run.
	if len(lr.Loaded) == 0 {
		return nil, "", fmt.Errorf("stage %s: driver loaded no routine (check the engine's routine source path / connection)", rtn)
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

// validRoutineName guards the routine name before it is interpolated into a
// generated M $TEXT script — only a real M routine name (optional leading %, then
// letters/digits, ≤31 chars — the YDB/IRIS significant-name limit, which
// allowLongNames packages like v-stdlib's VSLHL7TAP use) is allowed, so the name
// cannot inject M code.
func validRoutineName(s string) bool {
	if s == "" || len(s) > 31 {
		return false
	}
	for i, r := range s {
		switch {
		case r == '%':
			if i != 0 {
				return false
			}
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
			// a letter is valid anywhere
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// readRoutineBody is the M script body that streams a routine's source back as
// one `<<VPKG>>l<n>=<line>` marker per line (1-indexed, the $TEXT offset). It stops
// at the first line that is empty AND followed by an empty line (end of routine);
// VistA routine source carries no embedded blank lines, so a single sentinel is
// safe and the two-empty check is belt-and-suspenders. Lines carry no newline, so
// each survives the marker channel verbatim (leading whitespace and all).
func readRoutineBody(name string) string {
	return "N I,L,E S E=0 F I=1:1 D  Q:E\n" +
		". S L=$T(+I^" + name + ")\n" +
		". I L=\"\",$T(+(I+1)^" + name + ")=\"\" S E=1 Q\n" +
		". W \"" + installspec.ResultMarker + "l\",I,\"=\",L,!"
}

// parseRoutineLines reconstructs a routine's source, in order, from the
// `l<n>`-keyed markers readRoutineBody emitted. It walks contiguous 1-based keys
// and stops at the first gap (the end), so a value is never lost to map ordering.
func parseRoutineLines(markers map[string]string) []string {
	var out []string
	for i := 1; ; i++ {
		v, ok := markers["l"+strconv.Itoa(i)]
		if !ok {
			return out
		}
		out = append(out, v)
	}
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

// stageChunkBytes bounds each staging routine so the driver stages it reliably.
// A single routine large enough to carry a real package's transport global
// truncates silently when staged (T0b.2 discoveries P1); 40 KB is well under the
// observed limit while keeping the chunk count low.
const stageChunkBytes = 40000

// runInstall installs the build over the driver. The transport global is streamed
// into a staging global in size-bounded chunks (StageChunks), then a constant-size
// finalize routine verifies the staged count and runs INST → MERGE → EN^XPDIJ in
// one process. The outcome is read from the #9.7 status marker (or the
// already-installed guard). A staged-count mismatch is surfaced as an error.
func runInstall(ctx context.Context, cl *mdriver.Client, name, header string, pairs []kids.Pair) (installResult, error) {
	chunks := installspec.StageChunks(pairs, stageChunkBytes)
	for i, body := range chunks {
		if _, _, err := runMScript(ctx, cl, rtnInstall, body); err != nil {
			return installResult{Name: name}, fmt.Errorf("stage chunk %d/%d: %w", i+1, len(chunks), err)
		}
	}
	markers, _, err := runMScript(ctx, cl, rtnInstall, installspec.FinalInstallScript(name, header, len(pairs)))
	if err != nil {
		return installResult{Name: name}, err
	}
	r := installResult{Name: name}
	if e := markers["error"]; e != "" {
		if e == "already-installed" {
			r.Error = e
			return r, nil
		}
		// e.g. stage-incomplete: a chunk was truncated — fail loudly, never
		// install a partial package.
		return installResult{Name: name}, fmt.Errorf("install refused: %s (staged %s of %d nodes)",
			e, strings.TrimSpace(markers["staged"]), len(pairs))
	}
	r.Status, _ = strconv.Atoi(strings.TrimSpace(markers["status"]))
	r.Installed = r.Status == 3
	return r, nil
}

// installAction is the class-aware install strategy chosen after probing which
// of the build's routines already exist on the live engine.
type installAction int

const (
	// instProceed: install directly — either pure greenfield (nothing existing
	// is overwritten) or the operator accepted an unguarded overwrite.
	instProceed installAction = iota
	// instSnapshotProceed: capture the pre-image of the routines this install
	// overwrites, THEN install (so a later uninstall --restore can reverse it).
	instSnapshotProceed
	// instRefuse: the install would overwrite existing national routines with no
	// pre-image captured — refuse rather than silently clobber.
	instRefuse
)

func (a installAction) String() string {
	switch a {
	case instProceed:
		return "proceed"
	case instSnapshotProceed:
		return "snapshot+proceed"
	default:
		return "refuse"
	}
}

// decideInstall picks the install strategy. The rule (patch-existing-routines
// proposal): no SILENT clobber of national code. A pure-greenfield install
// proceeds (the existing behavior). An install that would OVERWRITE an existing
// routine must either capture a pre-image first (--snapshot, the safe path —
// enables uninstall --restore) or be an explicit unguarded overwrite
// (--allow-overwrite); absent both, it is refused.
func decideInstall(hasExisting bool, snapshot string, allowOverwrite bool) (installAction, string) {
	if !hasExisting {
		return instProceed, "all routines are new (greenfield) — nothing existing is overwritten"
	}
	if snapshot != "" {
		return instSnapshotProceed, "overwrites existing routine(s) — capturing their pre-image first (enables uninstall --restore)"
	}
	if allowOverwrite {
		return instProceed, "overwrites existing routine(s) WITHOUT a pre-image (--allow-overwrite) — uninstall cannot restore them"
	}
	return instRefuse, "would overwrite existing national routine(s) with no pre-image — pass --snapshot <out.kids> to capture one (enables uninstall --restore), or --allow-overwrite to clobber without one"
}

// liveInstallInput is the engine-write request for liveInstall — a named build's
// transport pairs plus the routines it lands on and the snapshot policy.
type liveInstallInput struct {
	name           string
	header         string
	className      string
	routineNames   []string // the routines this build lands on (overwrite probe targets)
	pairs          []kids.Pair
	snapshotPath   string // "" = no pre-image requested
	allowOverwrite bool
}

// liveInstall is the ONE class-aware install path (waterline rule: no caller
// hand-rolls a parallel installer). It probes which target routines already exist,
// decides the safe strategy (greenfield / snapshot+proceed / refuse) via
// decideInstall, captures the overwrite targets' pre-image when snapshotting, then
// ships via runInstall. Used by `v pkg install`. The returned *clikit.Error is the
// refuse / I/O failure; the caller renders the report and maps post-run install
// state.
func liveInstall(ctx context.Context, cl *mdriver.Client, in liveInstallInput) (installReport, *clikit.Error) {
	captured, greenfield, err := captureRoutinePreimages(ctx, cl, in.routineNames)
	if err != nil {
		return installReport{Name: in.name}, clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(),
			"confirm the driver connection before installing")
	}
	action, reason := decideInstall(len(captured) > 0, in.snapshotPath, in.allowOverwrite)
	res := installReport{
		Name: in.name, Class: in.className, Action: action.String(), Reason: reason,
		Overwrites: routineNames(captured), Greenfield: greenfield,
	}
	if action == instRefuse {
		return res, clikit.Fail(clikit.ExitRefused, "INSTALL_REFUSED",
			fmt.Sprintf("refusing to install %s: %s", in.name, reason),
			"pass --snapshot <out.kids> (or --auto-snapshot) to capture a pre-image first, or --allow-overwrite")
	}
	if action == instSnapshotProceed {
		snapName := snapshotName(in.name, "")
		pairs := buildSnapshotPairs(snapName, snapshotNamespace(in.name), captured)
		if werr := kids.WriteKID([]string{snapName}, map[string][]kids.Pair{snapName: pairs}, in.snapshotPath); werr != nil {
			return res, clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", werr.Error(), "")
		}
		res.Snapshot = in.snapshotPath
	}
	ir, ierr := runInstall(ctx, cl, in.name, in.header, in.pairs)
	if ierr != nil {
		return res, clikit.Fail(clikit.ExitRuntime, "INSTALL_FAILED", ierr.Error(), "")
	}
	res.Installed = ir.Installed
	res.Status = ir.Status
	res.Error = ir.Error
	return res, nil
}

type installCmd struct {
	engineConn
	KidFile        string `arg:"" help:"Path to the built .KID transport file to install on the live engine."`
	Snapshot       string `help:"Capture the pre-image of any routine this install overwrites to this .KID before installing (enables uninstall --restore)."`
	AutoSnapshot   bool   `help:"Like --snapshot, but to the conventional sidecar path (<kid>.preimage.kids) that uninstall auto-detects."`
	AllowOverwrite bool   `help:"Overwrite existing routines WITHOUT capturing a pre-image (unsafe: uninstall cannot then restore them)."`
}

type installReport struct {
	Name       string   `json:"name"`
	Class      string   `json:"class"`
	Action     string   `json:"action"`
	Reason     string   `json:"reason"`
	Overwrites []string `json:"overwrites,omitempty"` // existing routines this install replaces
	Greenfield []string `json:"greenfield,omitempty"` // routines this install newly adds
	Snapshot   string   `json:"snapshot,omitempty"`   // pre-image .KID written, if any
	Installed  bool     `json:"installed"`
	Status     int      `json:"status"`
	Error      string   `json:"error,omitempty"`
}

func (c *installCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	if len(k.InstallNames) != 1 {
		return clikit.Fail(clikit.ExitUsage, "MULTI_BUILD",
			fmt.Sprintf("install expects exactly one build, found %d", len(k.InstallNames)), "install one build at a time")
	}
	name := k.InstallNames[0]
	b := k.Builds[name]
	rev := kids.Classify(k)

	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	ctx := context.Background()

	// --auto-snapshot supplies the conventional sidecar path when no explicit
	// --snapshot is given, so install/uninstall pair without a path.
	snapshot := c.Snapshot
	if snapshot == "" && c.AutoSnapshot {
		snapshot = defaultPreimagePath(c.KidFile)
	}
	res, ferr := liveInstall(ctx, cl, liveInstallInput{
		name: name, header: name + " via v pkg install", className: rev.ClassName,
		routineNames: b.RoutineNames(), pairs: b.Pairs(),
		snapshotPath: snapshot, allowOverwrite: c.AllowOverwrite,
	})
	if ferr != nil {
		return ferr
	}

	if err := cc.Result(res, func() {
		cc.Title("pkg install — " + c.Engine)
		if len(res.Overwrites) > 0 {
			fmt.Fprintf(cc.Stdout, "  %s overwrites: %s\n", cc.Accent("patch"), strings.Join(res.Overwrites, ", "))
		}
		if res.Snapshot != "" {
			fmt.Fprintf(cc.Stdout, "  pre-image captured → %s\n", cc.Accent(res.Snapshot))
		}
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
	Params    map[string]bool `json:"params,omitempty"` // #8989.51 PARAMETER DEFINITIONs present
	Files     map[string]bool `json:"files,omitempty"`  // FileMan FILE data dictionaries present
	// Drift maps each routine -> "applied" | "drifted" | "absent" when --drift is
	// requested: does the LIVE routine still match the source this patch shipped?
	// "drifted" = a later national patch overwrote our code (the FU-21 re-pin gate).
	Drift map[string]string `json:"drift,omitempty"`
}

// ok reports a fully verified install: #9.7 present + completed, every routine
// loaded, every PARAMETER DEFINITION present, and — when --drift was checked —
// every routine still carrying the shipped patch (none drifted/absent).
func (r verifyResult) ok() bool {
	if !r.Installed || r.Status != 3 {
		return false
	}
	for _, present := range r.Routines {
		if !present {
			return false
		}
	}
	for _, present := range r.Params {
		if !present {
			return false
		}
	}
	for _, present := range r.Files {
		if !present {
			return false
		}
	}
	for _, state := range r.Drift {
		if state != "applied" {
			return false
		}
	}
	return true
}

// checkDrift reads each shipped routine off the live engine and compares it to
// the source the patch ships, returning "applied" (still our code), "drifted" (a
// later patch overwrote it), or "absent" (not on the engine). This is the FU-21
// re-pin gate: it answers "is my patch still applied to the live routine?"
func checkDrift(ctx context.Context, cl *mdriver.Client, b *kids.Build) (map[string]string, error) {
	drift := map[string]string{}
	for _, name := range b.RoutineNames() {
		live, present, err := readRoutinePreimage(ctx, cl, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		switch {
		case !present:
			drift[name] = "absent"
		case kids.RoutineDriftMatch(b.RoutineSource(name), live):
			drift[name] = "applied"
		default:
			drift[name] = "drifted"
		}
	}
	return drift, nil
}

func runVerify(ctx context.Context, cl *mdriver.Client, name string, routines, paramDefs, files []string) (verifyResult, error) {
	markers, _, err := runMScript(ctx, cl, rtnVerify, installspec.VerifyScript(name, routines, paramDefs, files))
	if err != nil {
		return verifyResult{Name: name}, err
	}
	r := verifyResult{Name: name, Routines: map[string]bool{}, Params: map[string]bool{}, Files: map[string]bool{}}
	r.Installed = strings.TrimSpace(markers["installed"]) == "1"
	r.Status, _ = strconv.Atoi(strings.TrimSpace(markers["status"]))
	for _, rt := range routines {
		r.Routines[rt] = strings.TrimSpace(markers["rtn:"+rt]) == "1"
	}
	for _, p := range paramDefs {
		r.Params[p] = strings.TrimSpace(markers["param:"+p]) == "1"
	}
	for _, f := range files {
		r.Files[f] = strings.TrimSpace(markers["file:"+f]) == "1"
	}
	return r, nil
}

type verifyCmd struct {
	engineConn
	KidFile string `arg:"" help:"Path to the .KID whose install to verify (its name + routines)."`
	Drift   bool   `help:"Also check whether each shipped routine still matches the patch on the live engine (detects a later national patch overwriting it)."`
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
	ctx := context.Background()
	res, err := runVerify(ctx, cl, name, b.RoutineNames(), b.ParamDefNames(), fileNumStrings(b))
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "VERIFY_FAILED", err.Error(), "")
	}
	if c.Drift {
		res.Drift, err = checkDrift(ctx, cl, b)
		if err != nil {
			return clikit.Fail(clikit.ExitRuntime, "VERIFY_FAILED", err.Error(), "")
		}
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
		for p, present := range res.Params {
			mark := cc.Success("ok")
			if !present {
				mark = cc.Failure("missing")
			}
			fmt.Fprintf(cc.Stdout, "  param %s %s\n", p, mark)
		}
		for f, present := range res.Files {
			mark := cc.Success("ok")
			if !present {
				mark = cc.Failure("missing")
			}
			fmt.Fprintf(cc.Stdout, "  file #%s %s\n", f, mark)
		}
		for rt, state := range res.Drift {
			mark := cc.Success("applied")
			switch state {
			case "drifted":
				mark = cc.Failure("DRIFTED (overwritten by a later patch)")
			case "absent":
				mark = cc.Failure("absent")
			}
			fmt.Fprintf(cc.Stdout, "  drift %s %s\n", rt, mark)
		}
	}); err != nil {
		return err
	}
	if !res.ok() {
		hint := "install it with `v pkg install`"
		for _, state := range res.Drift {
			if state == "drifted" {
				hint = "a later patch overwrote a routine — re-apply this patch (FU-21 re-pin)"
				break
			}
		}
		return clikit.Fail(clikit.ExitCheck, "NOT_VERIFIED",
			fmt.Sprintf("%s is not fully installed/applied", res.Name), hint)
	}
	return nil
}

// --- uninstall --------------------------------------------------------------

type uninstallResult struct {
	Name        string `json:"name"`
	Uninstalled bool   `json:"uninstalled"`
}

func runUninstall(ctx context.Context, cl *mdriver.Client, name string, routines, paramDefs, files []string) (uninstallResult, error) {
	markers, _, err := runMScript(ctx, cl, rtnUninstall, installspec.UninstallScript(name, routines, paramDefs, files))
	if err != nil {
		return uninstallResult{Name: name}, err
	}
	return uninstallResult{Name: name, Uninstalled: strings.TrimSpace(markers["uninstalled"]) == "1"}, nil
}

// fileNumStrings renders the build's FileMan file numbers as the strings the
// install scripts splice into M (an integer file number, e.g. "999000").
func fileNumStrings(b *kids.Build) []string {
	nums := b.FileNumbers()
	out := make([]string, 0, len(nums))
	for _, n := range nums {
		out = append(out, strconv.FormatInt(n, 10))
	}
	return out
}

// uninstallAction is the class-aware back-out strategy chosen for a .KID.
type uninstallAction int

const (
	// actDelete: routine-delete back-out (the original behavior) — correct ONLY
	// for a greenfield package whose components did not exist before install.
	actDelete uninstallAction = iota
	// actRestore: re-install a provided pre-image snapshot (the class-1 reversal
	// for a patch that OVERWROTE an existing routine — delete would brick it).
	actRestore
	// actBackout: install a provided authored back-out (the class-2 reversal —
	// run the patch's own inverse for its data/side-effects).
	actBackout
	// actRefuse: no safe back-out is possible from the inputs — refuse rather
	// than silently delete-and-orphan (the proposal's core safety fix).
	actRefuse
)

func (a uninstallAction) String() string {
	switch a {
	case actDelete:
		return "delete"
	case actRestore:
		return "restore"
	case actBackout:
		return "backout"
	default:
		return "refuse"
	}
}

// decideUninstall picks the back-out strategy from the patch's reversibility
// class and the operator's flags. The rule (patch-existing-routines proposal):
// NEVER silently delete a side-effecting patch (it orphans the data/side-effects
// its install code created); a class-1 patch that overwrote an existing routine
// needs its pre-image restored, not deleted. So a reversal artifact (--restore /
// --backout) always wins; absent one, a side-effecting patch is refused unless
// --force, and a class-1 patch falls back to the greenfield delete.
func decideUninstall(class kids.ReversibilityClass, restoreKid, backoutKid string, force bool) (uninstallAction, string) {
	if restoreKid != "" && backoutKid != "" {
		return actRefuse, "specify only one of --restore / --backout"
	}
	if restoreKid != "" {
		return actRestore, "restore the provided pre-image snapshot"
	}
	if backoutKid != "" {
		return actBackout, "install the provided authored back-out"
	}
	if class == kids.ClassSideEffecting {
		if force {
			return actDelete, "FORCED routine-delete — install-time data/side-effects are NOT reversed and will be orphaned"
		}
		return actRefuse, "side-effecting patch (class 2/3): deleting routines orphans the data/side-effects its install code created — provide --backout (authored inverse) or --restore (pre-image), or author a forward back-out patch (--force to delete routines anyway)"
	}
	return actDelete, "class-1: routine-delete back-out (if this patch OVERWROTE existing routines, use --restore <pre-image> instead — delete would brick them)"
}

type uninstallCmd struct {
	engineConn
	KidFile string `arg:"" help:"Path to the .KID whose install to reverse."`
	Restore string `help:"Pre-image snapshot .KID to restore instead of deleting (class-1 reversal for a patched-over routine)."`
	Backout string `help:"Authored back-out .KID to install instead of deleting (class-2 reversal of a side-effecting patch)."`
	Force   bool   `help:"Delete routines even for a side-effecting patch (UNSAFE: orphans install-time data/side-effects)."`
	Verify  bool   `help:"After a restore/back-out, confirm the live routines now match the re-applied artifact (verify-clean)."`
}

type uninstallReport struct {
	Name         string `json:"name"`
	Class        string `json:"class"`
	Action       string `json:"action"`
	Reason       string `json:"reason"`
	AutoDetected bool   `json:"autoDetected,omitempty"` // pre-image found via the sidecar convention
	Done         bool   `json:"done"`
	Status       int    `json:"status,omitempty"`      // #9.7 status for restore/backout installs
	VerifyClean  string `json:"verifyClean,omitempty"` // "" | "clean" | "dirty" (when --verify)
}

func (c *uninstallCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	if len(k.InstallNames) != 1 {
		return clikit.Fail(clikit.ExitUsage, "MULTI_BUILD",
			fmt.Sprintf("uninstall expects exactly one build, found %d", len(k.InstallNames)), "uninstall one build at a time")
	}
	name := k.InstallNames[0]
	b := k.Builds[name]
	rev := kids.Classify(k)

	// Auto-detect a paired pre-image at the conventional sidecar path so
	// `uninstall <patch.kid>` reverses cleanly without re-specifying --restore.
	restore, autoDetected := resolveAutoRestore(c.Restore, c.Backout, c.KidFile, fileExists(defaultPreimagePath(c.KidFile)))
	action, reason := decideUninstall(rev.Class, restore, c.Backout, c.Force)
	res := uninstallReport{Name: name, Class: rev.ClassName, Action: action.String(), Reason: reason, AutoDetected: autoDetected}

	if action == actRefuse {
		return clikit.Fail(clikit.ExitRefused, "UNINSTALL_REFUSED",
			fmt.Sprintf("refusing to uninstall %s [%s]: %s", name, rev.ClassName, reason),
			"provide --restore <pre-image> or --backout <authored back-out>, or --force to delete routines anyway")
	}

	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	ctx := context.Background()

	switch action {
	case actRestore, actBackout:
		// reversal = install the provided/auto-detected pre-image or back-out.
		src := restore
		if action == actBackout {
			src = c.Backout
		}
		rname, rb, lerr := loadBuild(src)
		if lerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", lerr.Error(), "")
		}
		ir, ierr := runInstall(ctx, cl, rname, rname+" via v pkg uninstall --"+action.String(), rb.Pairs())
		if ierr != nil {
			return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", ierr.Error(), "")
		}
		res.Done = ir.Installed
		res.Status = ir.Status
		// verify-clean: confirm the live routines now match the re-applied artifact.
		if c.Verify && res.Done {
			drift, derr := checkDrift(ctx, cl, rb)
			if derr != nil {
				return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", derr.Error(), "")
			}
			res.VerifyClean = "clean"
			for _, state := range drift {
				if state != "applied" {
					res.VerifyClean = "dirty"
					break
				}
			}
		}
	default: // actDelete
		ur, uerr := runUninstall(ctx, cl, name, b.RoutineNames(), b.ParamDefNames(), fileNumStrings(b))
		if uerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", uerr.Error(), "")
		}
		res.Done = ur.Uninstalled
	}

	if err := cc.Result(res, func() {
		cc.Title("pkg uninstall — " + c.Engine)
		fmt.Fprintf(cc.Stdout, "%s [%s] %s\n", cc.Accent(name), rev.ClassName, cc.Faint(reason))
		if res.AutoDetected {
			fmt.Fprintf(cc.Stdout, "  %s pre-image: %s\n", cc.Faint("auto-detected"), defaultPreimagePath(c.KidFile))
		}
		if res.Done {
			fmt.Fprintln(cc.Stdout, cc.Success(action.String()+" complete for "+name))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(action.String()+" not confirmed for "+name))
		}
		switch res.VerifyClean {
		case "clean":
			fmt.Fprintln(cc.Stdout, cc.Success("verify-clean: live routines match the re-applied artifact"))
		case "dirty":
			fmt.Fprintln(cc.Stdout, cc.Failure("verify-clean FAILED: live routines do not match the re-applied artifact"))
		}
	}); err != nil {
		return err
	}
	if !res.Done {
		return clikit.Fail(clikit.ExitRuntime, "NOT_UNINSTALLED",
			action.String()+" of "+name+" was not confirmed", "")
	}
	if res.VerifyClean == "dirty" {
		return clikit.Fail(clikit.ExitCheck, "VERIFY_CLEAN_FAILED",
			"reversal of "+name+" did not verify clean", "the live routines do not match the re-applied artifact")
	}
	return nil
}
