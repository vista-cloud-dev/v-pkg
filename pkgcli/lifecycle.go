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
	rtnHeal      = "ZVPKGHL"
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
	Name       string `json:"name"`
	Installed  bool   `json:"installed"`
	Status     int    `json:"status"`               // #9.7 STATUS piece 9 (3 = Install Completed)
	Error      string `json:"error,omitempty"`      // e.g. "already-installed"
	PackageIEN string `json:"packageIen,omitempty"` // #9.4 entry the A.3 footprint stamped, if any
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
func runInstall(ctx context.Context, cl *mdriver.Client, name, header string, pairs []kids.Pair, runEnvCheck bool, ques []installspec.QuesAnswer, reg *installspec.PkgReg) (installResult, error) {
	// Strip v-pkg's private metadata (the foreign-overwrite declaration) before it
	// can reach the engine transport global — it is read offline by uninstall, never
	// staged into KIDS filing (F1, waterline rule 3: only real KIDS content crosses
	// the seam). A declaration-free build is unaffected (EnginePairs is a no-op).
	pairs = kids.EnginePairs(pairs)
	chunks := installspec.StageChunks(pairs, stageChunkBytes)
	for i, body := range chunks {
		if _, _, err := runMScript(ctx, cl, rtnInstall, body); err != nil {
			return installResult{Name: name}, fmt.Errorf("stage chunk %d/%d: %w", i+1, len(chunks), err)
		}
	}
	markers, _, err := runMScript(ctx, cl, rtnInstall, installspec.FinalInstallScript(name, header, len(pairs), runEnvCheck, ques, reg))
	if err != nil {
		return installResult{Name: name}, err
	}
	r := installResult{Name: name}
	if e := markers["error"]; e != "" {
		if e == "already-installed" {
			r.Error = e
			return r, nil
		}
		// env-check-rejected^<rc>^<reqab>: the build's environment-check routine
		// or Required-Build (#9.611) enforcement rejected the install (A.1.2).
		// Refuse loudly — the aborted #9.7 entry was already purged engine-side.
		if strings.HasPrefix(e, "env-check-rejected") {
			return installResult{Name: name}, fmt.Errorf(
				"install refused: environment check / required-build enforcement rejected %s (%s) — "+
					"fix the environment, or pass --skip-env-check to bypass", name, e)
		}
		// e.g. stage-incomplete: a chunk was truncated — fail loudly, never
		// install a partial package.
		return installResult{Name: name}, fmt.Errorf("install refused: %s (staged %s of %d nodes)",
			e, strings.TrimSpace(markers["staged"]), len(pairs))
	}
	r.Status, _ = strconv.Atoi(strings.TrimSpace(markers["status"]))
	r.Installed = r.Status == 3
	r.PackageIEN = strings.TrimSpace(markers["pkg"])
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
	heal           bool                     // --heal: purge a corrupt half-install (§7.1) before installing
	runEnvCheck    bool                     // run the build's env-check + Required-Build enforcement before filing (A.1.2)
	quesAnswers    []installspec.QuesAnswer // pre-answered build install questions (A.1.3)
	pkgReg         *installspec.PkgReg      // optional PACKAGE #9.4 footprint to write after filing (A.3)
}

// liveInstall is the ONE class-aware install path (waterline rule: no caller
// hand-rolls a parallel installer). It probes which target routines already exist,
// decides the safe strategy (greenfield / snapshot+proceed / refuse) via
// decideInstall, captures the overwrite targets' pre-image when snapshotting, then
// ships via runInstall. Used by `v pkg install`. The returned *clikit.Error is the
// refuse / I/O failure; the caller renders the report and maps post-run install
// state.
func liveInstall(ctx context.Context, cl *mdriver.Client, in liveInstallInput) (installReport, *clikit.Error) {
	// --heal: if a corrupt half-install (§7.1) is blocking reinstall, purge that
	// PROVEN-corrupt entry first — never a healthy one. Runs before the overwrite
	// probe so a clean reinstall proceeds against a cleared #9.7 slot.
	var heal *healResult
	if in.heal {
		state, herr := probeHeal(ctx, cl, in.name)
		if herr != nil {
			return installReport{Name: in.name}, clikit.Fail(clikit.ExitRuntime, "HEAL_PROBE_FAILED", herr.Error(),
				"confirm the driver connection before healing")
		}
		action, reason := decideHeal(state)
		heal = &healResult{State: state.String(), Action: action.String(), Reason: reason}
		switch action {
		case healRefuse:
			return installReport{Name: in.name, Heal: heal}, clikit.Fail(clikit.ExitRefused, "HEAL_REFUSED",
				fmt.Sprintf("refusing to heal %s: %s", in.name, reason),
				"`v pkg uninstall` removes a healthy install; --heal only repairs a corrupt half-install")
		case healPurgeProceed:
			purged, perr := purgeHeal(ctx, cl, in.name)
			if perr != nil {
				return installReport{Name: in.name, Heal: heal}, clikit.Fail(clikit.ExitRuntime, "HEAL_FAILED", perr.Error(), "")
			}
			heal.Purged = purged
		}
	}

	captured, greenfield, err := captureRoutinePreimages(ctx, cl, in.routineNames)
	if err != nil {
		return installReport{Name: in.name}, clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(),
			"confirm the driver connection before installing")
	}
	action, reason := decideInstall(len(captured) > 0, in.snapshotPath, in.allowOverwrite)
	res := installReport{
		Name: in.name, Class: in.className, Action: action.String(), Reason: reason,
		Overwrites: routineNames(captured), Greenfield: greenfield, Heal: heal,
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
	ir, ierr := runInstall(ctx, cl, in.name, in.header, in.pairs, in.runEnvCheck, in.quesAnswers, in.pkgReg)
	if ierr != nil {
		return res, clikit.Fail(clikit.ExitRuntime, "INSTALL_FAILED", ierr.Error(), "")
	}
	res.Installed = ir.Installed
	res.Status = ir.Status
	res.Error = ir.Error
	res.PackageIEN = ir.PackageIEN
	return res, nil
}

type installCmd struct {
	engineConn
	KidFile         string   `arg:"" help:"Path to the built .KID transport file to install on the live engine."`
	DryRun          bool     `help:"Preview the install against the live engine WITHOUT mutating it: report NEW/CHANGED/identical per routine and would-add/would-change/identical per component (read-only; exit 0). Same plan as 'v pkg diff'."`
	Snapshot        string   `help:"Capture the pre-image of any routine this install overwrites to this .KID before installing (enables uninstall --restore)."`
	AutoSnapshot    bool     `help:"Like --snapshot, but to the conventional sidecar path (<kid>.preimage.kids) that uninstall auto-detects."`
	AllowOverwrite  bool     `help:"Overwrite existing routines WITHOUT capturing a pre-image (unsafe: uninstall cannot then restore them)."`
	Heal            bool     `help:"Before installing, purge a corrupt half-install (a #9.7 entry with no usable 0-node that falsely blocks reinstall as 'already-installed'); never touches a healthy install."`
	VerifyChecksums bool     `help:"Make the transport-checksum check a HARD gate: REFUSE a .KID whose stored routine checksum does not match its shipped source (default: warn and proceed — ~6% of real foreign patches carry a self-inconsistent checksum, indistinguishable from tampering offline)."`
	SkipEnvCheck    bool     `help:"Skip the build's environment-check routine + Required-Build (#9.611) enforcement before filing (default: run them, KIDS-faithful)."`
	Answer          []string `help:"Pre-answer a build install question as NAME=VALUE (the internal answer $$ANSWER^XPDIQ returns); repeatable." placeholder:"NAME=VALUE"`
	RegisterPackage string   `help:"Record the PACKAGE #9.4 footprint under this long NAME (find-or-create by the install-name prefix; stamps VERSION + PATCH APPLICATION HISTORY so $$VER/$$PATCH^XPDUTL see the install)." placeholder:"PACKAGE NAME"`
}

// packageReg derives the #9.4 registration from the install name (PREFIX*VERSION
// [*PATCH]) and the --register-package NAME. Returns nil when no name was given (no
// footprint) or when the install name is malformed.
func packageReg(installName, regName string) *installspec.PkgReg {
	if regName == "" {
		return nil
	}
	parts := strings.Split(installName, "*")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}
	reg := &installspec.PkgReg{Prefix: parts[0], Name: regName, Version: parts[1]}
	if len(parts) >= 3 {
		reg.Patch = parts[2]
	}
	return reg
}

// deregReg parses an install name (PREFIX*VERSION[*PATCH]) into the #9.4 footprint
// to clear on `uninstall --deregister`. Unlike packageReg it needs no long NAME —
// deregister FINDS the package by prefix and never creates it. nil if malformed.
func deregReg(installName string) *installspec.PkgReg {
	parts := strings.Split(installName, "*")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil
	}
	reg := &installspec.PkgReg{Prefix: parts[0], Version: parts[1]}
	if len(parts) >= 3 {
		reg.Patch = parts[2]
	}
	return reg
}

// runDeregister clears the PACKAGE #9.4 patch-history footprint for reg (the inverse
// of install --register-package). Returns whether a patch-history entry was removed
// (false, no engine call, for a nil/patchless reg).
func runDeregister(ctx context.Context, cl *mdriver.Client, reg *installspec.PkgReg) (bool, error) {
	script := installspec.DeregisterScript(reg)
	if script == "" {
		return false, nil
	}
	markers, _, err := runMScript(ctx, cl, rtnUninstall, script)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(markers["dereg"]) == "1", nil
}

// parseAnswers turns the repeatable --answer NAME=VALUE flags into ordered QUES
// answers. The name is everything before the first '=', the value everything after
// (so a value may itself contain '='); order is preserved for deterministic IENs.
func parseAnswers(flags []string) ([]installspec.QuesAnswer, error) {
	out := make([]installspec.QuesAnswer, 0, len(flags))
	for _, f := range flags {
		i := strings.IndexByte(f, '=')
		if i <= 0 {
			return nil, fmt.Errorf("--answer %q must be NAME=VALUE", f)
		}
		out = append(out, installspec.QuesAnswer{Name: f[:i], Value: f[i+1:]})
	}
	return out, nil
}

type installReport struct {
	Name             string                `json:"name"`
	Class            string                `json:"class"`
	Action           string                `json:"action"`
	Reason           string                `json:"reason"`
	Heal             *healResult           `json:"heal,omitempty"`             // --heal probe/purge of a corrupt half-install (§7.1)
	ChecksumWarnings []kids.ChecksumResult `json:"checksumWarnings,omitempty"` // #3b: routines whose stored checksum ≠ shipped source (warned, not refused)
	Overwrites       []string              `json:"overwrites,omitempty"`       // existing routines this install replaces
	Greenfield       []string              `json:"greenfield,omitempty"`       // routines this install newly adds
	Snapshot         string                `json:"snapshot,omitempty"`         // pre-image .KID written, if any
	Installed        bool                  `json:"installed"`
	Status           int                   `json:"status"`
	Error            string                `json:"error,omitempty"`
	PackageIEN       string                `json:"packageIen,omitempty"` // #9.4 entry the A.3 footprint stamped, if any
}

// installSequence installs each named build in order through the one class-aware
// install path, stopping at the first build that fails — an engine/refuse error or
// a build that does not reach #9.7 status 3. A multi-build distribution lists its
// constituents in dependency order in the **KIDS** header, so a failed prerequisite
// must abort the rest rather than install dependents against a missing base (A.4).
func installSequence(ctx context.Context, cl *mdriver.Client, names []string, mk func(name string) liveInstallInput) (multiInstallResult, *clikit.Error) {
	out := multiInstallResult{Installed: true}
	for _, name := range names {
		res, ferr := liveInstall(ctx, cl, mk(name))
		out.Builds = append(out.Builds, res)
		if ferr != nil {
			out.Installed = false
			return out, ferr
		}
		if res.Error != "" || !res.Installed {
			out.Installed = false
			return out, nil
		}
	}
	return out, nil
}

// multiInstallResult is the per-build roll-up for a multi-build distribution.
type multiInstallResult struct {
	Builds    []installReport `json:"builds"`
	Installed bool            `json:"installed"` // every constituent reached status 3
}

func (c *installCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	if len(k.InstallNames) == 0 {
		return clikit.Fail(clikit.ExitUsage, "NO_BUILD", "no build found in "+c.KidFile, "")
	}

	// Offline transport-checksum check (#3b), computed before reaching the engine.
	// Default = WARN: mismatches ride on the install report (and print) but do not
	// block — ~6% of real foreign patches are born self-inconsistent and the check
	// can't tell that from tampering offline. --verify-checksums makes it a HARD gate
	// that refuses (fail fast, no connection, no write). Skipped for --dry-run (a
	// read-only, exit-0 preview).
	var cksumScan map[string][]kids.ChecksumResult
	if !c.DryRun {
		cksumScan = checksumScan(k)
		if c.VerifyChecksums {
			if ferr := checksumRefusal(cksumScan); ferr != nil {
				return ferr
			}
		}
	}

	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	ctx := context.Background()

	// --dry-run: preview the install against the live engine and stop (read-only,
	// exit 0). The same plan `v pkg diff` produces; covers single- and multi-build.
	if c.DryRun {
		return runDryRun(ctx, cc, cl, c.Engine, k)
	}

	answers, aerr := parseAnswers(c.Answer)
	if aerr != nil {
		return clikit.Fail(clikit.ExitUsage, "BAD_ANSWER", aerr.Error(), "use --answer NAME=VALUE")
	}

	// A multi-build distribution (the **KIDS** header lists >1 build) installs each
	// constituent in header order (A.4). The per-build flags are inherently ambiguous
	// across constituents, so refuse the build-specific ones; --auto-snapshot still
	// pairs each build to its own sidecar path.
	if len(k.InstallNames) > 1 {
		return c.runMulti(ctx, cc, cl, k, answers)
	}

	name := k.InstallNames[0]
	b := k.Builds[name]
	rev := kids.Classify(k)

	// --auto-snapshot supplies the conventional sidecar path when no explicit
	// --snapshot is given, so install/uninstall pair without a path.
	snapshot := c.Snapshot
	if snapshot == "" && c.AutoSnapshot {
		snapshot = defaultPreimagePath(c.KidFile)
	}
	res, ferr := liveInstall(ctx, cl, liveInstallInput{
		name: name, header: name + " via v pkg install", className: rev.ClassName,
		routineNames: b.RoutineNames(), pairs: b.Pairs(),
		snapshotPath: snapshot, allowOverwrite: c.AllowOverwrite, heal: c.Heal,
		runEnvCheck: !c.SkipEnvCheck, quesAnswers: answers,
		pkgReg: packageReg(name, c.RegisterPackage),
	})
	if ferr != nil {
		return ferr
	}
	res.ChecksumWarnings = cksumScan[name] // #3b: warn-mode mismatches (none if --verify-checksums passed)

	if err := cc.Result(res, func() {
		cc.Title("pkg install — " + c.Engine)
		for _, w := range res.ChecksumWarnings {
			fmt.Fprintln(cc.Stdout, cc.Warning(fmt.Sprintf("checksum mismatch %s: stored %s, recomputed %s (installing anyway; --verify-checksums to refuse)", w.Name, w.Stored, w.Computed)))
		}
		if res.Heal != nil && res.Heal.Purged {
			fmt.Fprintf(cc.Stdout, "  %s purged a corrupt half-install of %s before reinstalling\n", cc.Accent("heal"), res.Name)
		}
		if len(res.Overwrites) > 0 {
			fmt.Fprintf(cc.Stdout, "  %s overwrites: %s\n", cc.Accent("patch"), strings.Join(res.Overwrites, ", "))
		}
		if res.Snapshot != "" {
			fmt.Fprintf(cc.Stdout, "  pre-image captured → %s\n", cc.Accent(res.Snapshot))
		}
		if res.PackageIEN != "" {
			fmt.Fprintf(cc.Stdout, "  PACKAGE #9.4 footprint → entry %s\n", cc.Accent(res.PackageIEN))
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

// runMulti installs a multi-build distribution: each constituent in **KIDS**-header
// order, stopping at the first failure (A.4). The build-specific flags
// (--register-package, --skip-env-check answers, --snapshot path) are ambiguous
// across constituents, so registration is refused and --auto-snapshot pairs each
// build to its own sidecar. --answer applies to every build (KIDS scopes a question
// per build, so a NAME matches whichever build defines it).
func (c *installCmd) runMulti(ctx context.Context, cc *clikit.Context, cl *mdriver.Client, k *kids.KID, answers []installspec.QuesAnswer) error {
	if c.RegisterPackage != "" {
		return clikit.Fail(clikit.ExitUsage, "MULTI_BUILD_REGISTER",
			"--register-package is ambiguous for a multi-build distribution", "install constituents one at a time to register a package footprint")
	}
	if c.Snapshot != "" {
		return clikit.Fail(clikit.ExitUsage, "MULTI_BUILD_SNAPSHOT",
			"--snapshot <path> is ambiguous for a multi-build distribution", "use --auto-snapshot (a per-build sidecar) instead")
	}
	mk := func(name string) liveInstallInput {
		b := k.Builds[name]
		snap := ""
		if c.AutoSnapshot {
			snap = defaultPreimagePath(c.KidFile) + "." + sanitizeName(name)
		}
		return liveInstallInput{
			name: name, header: name + " via v pkg install",
			className:    kids.ClassifyBuild(name, b).ClassName,
			routineNames: b.RoutineNames(), pairs: b.Pairs(),
			snapshotPath: snap, allowOverwrite: c.AllowOverwrite, heal: c.Heal,
			runEnvCheck: !c.SkipEnvCheck, quesAnswers: answers,
		}
	}
	res, ferr := installSequence(ctx, cl, k.InstallNames, mk)
	// #3b warn-mode: attach each build's checksum mismatches to its report (strict
	// --verify-checksums already refused upstream, so this only fires in warn mode).
	cksumScan := checksumScan(k)
	for i := range res.Builds {
		res.Builds[i].ChecksumWarnings = cksumScan[res.Builds[i].Name]
	}
	if rerr := cc.Result(res, func() {
		cc.Title("pkg install (multi-build) — " + c.Engine)
		for _, b := range res.Builds {
			for _, w := range b.ChecksumWarnings {
				fmt.Fprintln(cc.Stdout, cc.Warning(fmt.Sprintf("checksum mismatch %s:%s: stored %s, recomputed %s (installing anyway; --verify-checksums to refuse)", b.Name, w.Name, w.Stored, w.Computed)))
			}
			switch {
			case b.Error != "":
				fmt.Fprintln(cc.Stdout, cc.Failure(b.Name+": "+b.Error))
			case b.Installed:
				fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("installed %s (#9.7 status %d)", b.Name, b.Status)))
			default:
				fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("%s did not complete (#9.7 status %d)", b.Name, b.Status)))
			}
		}
		fmt.Fprintf(cc.Stdout, "  %d of %d build(s) installed\n", countInstalled(res.Builds), len(k.InstallNames))
	}); rerr != nil {
		return rerr
	}
	if ferr != nil {
		return ferr
	}
	if !res.Installed {
		return clikit.Fail(clikit.ExitRuntime, "NOT_INSTALLED",
			fmt.Sprintf("multi-build install stopped after %d of %d build(s)", countInstalled(res.Builds), len(k.InstallNames)), "")
	}
	return nil
}

// sanitizeName turns an install name (PKG*VER*PATCH) into a filename-safe token for
// per-build snapshot sidecars (the "*" becomes "_").
func sanitizeName(s string) string { return strings.ReplaceAll(s, "*", "_") }

// countInstalled counts how many of the gathered reports reached status 3.
func countInstalled(reports []installReport) int {
	n := 0
	for _, r := range reports {
		if r.Installed {
			n++
		}
	}
	return n
}

// --- verify -----------------------------------------------------------------

type verifyResult struct {
	Name      string          `json:"name"`
	Installed bool            `json:"installed"`
	Status    int             `json:"status"`
	Routines  map[string]bool `json:"routines"`
	// Components maps each shipped entry-component record ("<file>:<name>") ->
	// present, across every registered type (OPTION #19, INPUT TEMPLATE #.402, …) —
	// the registry-driven replacement for the per-type maps. Probed by the type's
	// storage-file "B" index (KRN^XPDIK builds it on install).
	Components map[string]bool `json:"components,omitempty"`
	Files      map[string]bool `json:"files,omitempty"` // FileMan FILE data dictionaries present
	// Drift maps each routine -> "applied" | "drifted" | "absent" when --drift is
	// requested: does the LIVE routine still match the source this patch shipped?
	// "drifted" = a later national patch overwrote our code (the FU-21 re-pin gate).
	Drift map[string]string `json:"drift,omitempty"`
	// Content maps each shipped entry record ("<file>:<name>") -> "ok" | "mismatch"
	// | "absent": does the LIVE 0-node match the image this build shipped (aside
	// from FileMan-transformed pieces)? This turns "a record by this name exists"
	// into "the record we shipped is the record that got filed."
	Content map[string]string `json:"content,omitempty"`
}

// ok reports a fully verified install: #9.7 present + completed, every routine
// loaded, every component present, every shipped entry record matching the image
// we filed (content, not just presence), and — when --drift was checked — every
// routine still carrying the shipped patch (none drifted/absent).
func (r verifyResult) ok() bool {
	if !r.Installed || r.Status != 3 {
		return false
	}
	for _, present := range r.Routines {
		if !present {
			return false
		}
	}
	for _, present := range r.Components {
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
	for _, state := range r.Content {
		if state != "ok" {
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

// foreignRestoreVerdict grades how faithfully the declared-foreign routines were
// restored from src (the re-installed pre-image), reading each live off the engine
// — the D4 national-overwrite verify, STRICTER than checkDrift's line-2-blind
// RoutineDriftMatch. It returns the WORST verdict across the foreign routines
// (kids.RestoreExact / RestoreProvenanceDrift / RestoreDrift), or "" when the
// build declares no foreign routines. A foreign routine that reads back absent is
// graded RestoreDrift (the restore did not reinstate it).
func foreignRestoreVerdict(ctx context.Context, cl *mdriver.Client, src *kids.Build, foreignNames []string) (string, error) {
	rank := map[string]int{"": 0, kids.RestoreExact: 1, kids.RestoreProvenanceDrift: 2, kids.RestoreDrift: 3}
	worst := ""
	for _, name := range foreignNames {
		live, present, err := readRoutinePreimage(ctx, cl, name)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", name, err)
		}
		v := kids.RestoreDrift
		if present {
			v = kids.RoutineRestoreVerdict(src.RoutineSource(name), live)
		}
		if rank[v] > rank[worst] {
			worst = v
		}
	}
	return worst, nil
}

// verifyContent reads the live 0-node of each shipped entry record off the engine
// and grades it against the shipped image: "ok" (matches, FileMan-transformed
// pieces aside), "mismatch" (a record by that name exists but differs), or
// "absent" (no record by that name). This is the content half of verify, run as a
// second staged script so the presence path is untouched.
func verifyContent(ctx context.Context, cl *mdriver.Client, contents []kids.EntryContent, files []kids.FileContent) (map[string]string, error) {
	if len(contents) == 0 && len(files) == 0 {
		return nil, nil
	}
	markers, _, err := runMScript(ctx, cl, rtnVerify, installspec.VerifyContentScript(contents, files))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(contents)+len(files))
	grade := func(key, expected, live string, volatile []int) {
		switch {
		case live == "":
			out[key] = "absent"
		case kids.ZeroMatch(expected, live, volatile):
			out[key] = "ok"
		default:
			out[key] = "mismatch"
		}
	}
	for _, c := range contents {
		key := c.FileStr + ":" + c.Name
		grade(key, c.Zero, strings.TrimSpace(markers["z:"+key]), c.Volatile)
	}
	// FILE DD fields: the live ^DD(file,fld,0) must match the shipped fieldDef.
	for _, f := range files {
		key := f.FileStr + "#" + f.Field
		grade(key, f.Zero, strings.TrimSpace(markers["dd:"+key]), nil)
	}
	return out, nil
}

func runVerify(ctx context.Context, cl *mdriver.Client, name string, routines []string, comps []kids.Component, files []string) (verifyResult, error) {
	markers, _, err := runMScript(ctx, cl, rtnVerify, installspec.VerifyScript(name, routines, comps, files))
	if err != nil {
		return verifyResult{Name: name}, err
	}
	r := verifyResult{Name: name, Routines: map[string]bool{}, Components: map[string]bool{}, Files: map[string]bool{}}
	r.Installed = strings.TrimSpace(markers["installed"]) == "1"
	r.Status, _ = strconv.Atoi(strings.TrimSpace(markers["status"]))
	for _, rt := range routines {
		r.Routines[rt] = strings.TrimSpace(markers["rtn:"+rt]) == "1"
	}
	for _, c := range comps {
		for _, n := range c.Names {
			key := c.FileStr + ":" + n
			r.Components[key] = strings.TrimSpace(markers["comp:"+key]) == "1"
		}
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
	res, err := runVerify(ctx, cl, name, b.RoutineNames(), b.Components(), fileNumStrings(b))
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "VERIFY_FAILED", err.Error(), "")
	}
	if c.Drift {
		res.Drift, err = checkDrift(ctx, cl, b)
		if err != nil {
			return clikit.Fail(clikit.ExitRuntime, "VERIFY_FAILED", err.Error(), "")
		}
	}
	res.Content, err = verifyContent(ctx, cl, b.EntryContents(), b.FileContents())
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
		for comp, present := range res.Components {
			mark := cc.Success("ok")
			if !present {
				mark = cc.Failure("missing")
			}
			fmt.Fprintf(cc.Stdout, "  component %s %s\n", comp, mark)
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
		for key, state := range res.Content {
			mark := cc.Success("match")
			if state != "ok" {
				mark = cc.Failure(state)
			}
			fmt.Fprintf(cc.Stdout, "  content %s %s\n", key, mark)
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

func runUninstall(ctx context.Context, cl *mdriver.Client, name string, routines []string, comps []kids.Component, files []string) (uninstallResult, error) {
	markers, _, err := runMScript(ctx, cl, rtnUninstall, installspec.UninstallScript(name, routines, comps, files))
	if err != nil {
		return uninstallResult{Name: name}, err
	}
	return uninstallResult{Name: name, Uninstalled: strings.TrimSpace(markers["uninstalled"]) == "1"}, nil
}

// clearBuildFootprint removes a build's #9.7/#9.6 registration BY INSTALL NAME
// without touching any routine or component (an all-empty UninstallScript deletes
// only the ^XPD(9.7/9.6,"B",name) index). The partitioned uninstall uses it to
// drop the snapshot's "<name> PREIMAGE" provenance entry — so the restore re-files
// idempotently across cycles and the uninstall leaves no ghost build registered.
func clearBuildFootprint(ctx context.Context, cl *mdriver.Client, name string) error {
	_, err := runUninstall(ctx, cl, name, nil, nil, nil)
	return err
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
	// actPartition: the BB1 reversal for a build that BOTH overwrote a foreign
	// routine AND added greenfield routines (the v-rpc-tap shape: it splices the
	// national XWBPRS *and* ships VSLRT*). A single restore would orphan the adds;
	// a single delete would brick the foreign national routine. So partition the
	// build's routines against the pre-image and run BOTH halves, ordered: restore
	// the overwritten foreign routine(s) from the pre-image FIRST, then delete the
	// greenfield-added routine(s) — F-I/R21, so a live caller never reaches a
	// callee that was deleted before its caller was un-spliced.
	actPartition
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
	case actPartition:
		return "partition"
	default:
		return "refuse"
	}
}

// partitionRoutines splits a build's routines into the set to RESTORE from a
// pre-image and the set to DELETE. The pre-image (captured at install time by
// `captureRoutinePreimages`) holds exactly the routines that PRE-EXISTED and were
// overwritten — so a build routine present in the pre-image is overwritten-foreign
// (restore it) and one absent from the pre-image is greenfield-added (delete it).
// This is the heart of the partitioned uninstall (BB1): without it a build that
// both overwrites a national routine and adds new ones has no safe single reversal.
// Build order is preserved within each partition; pre-image routines not in the
// build are ignored (only the build's own routines are ever touched).
func partitionRoutines(buildRoutines, preimageRoutines []string) (restore, del []string) {
	pre := make(map[string]bool, len(preimageRoutines))
	for _, r := range preimageRoutines {
		pre[r] = true
	}
	for _, r := range buildRoutines {
		if pre[r] {
			restore = append(restore, r)
		} else {
			del = append(del, r)
		}
	}
	return restore, del
}

// intersectRoutines returns the names in `want` that also appear in `have`, in
// `want` order. Used by the wrong-sidecar guard to find declared foreign routines
// that a pre-image failed to capture (i.e. landed in the delete set).
func intersectRoutines(want, have []string) []string {
	set := make(map[string]bool, len(have))
	for _, r := range have {
		set[r] = true
	}
	var out []string
	for _, r := range want {
		if set[r] {
			out = append(out, r)
		}
	}
	return out
}

// decideUninstall picks the back-out strategy from the patch's reversibility
// class and the operator's flags. The rule (patch-existing-routines proposal):
// NEVER silently delete a side-effecting patch (it orphans the data/side-effects
// its install code created); a class-1 patch that overwrote an existing routine
// needs its pre-image restored, not deleted. So a reversal artifact (--restore /
// --backout) always wins; absent one, a side-effecting patch is refused unless
// --force, and a class-1 patch falls back to the greenfield delete.
//
// hasGreenfieldAdds (BB1) is only meaningful WITH a pre-image: it reports whether
// the build adds routines the pre-image does not carry. When a --restore pre-image
// is in play AND the build also adds greenfield routines, a plain restore would
// orphan those adds, so we partition (restore the overwritten + delete the added).
//
// hasForeignOverwrites (F1) is the build's OWN declaration (read offline from the
// .KID) that it overwrote a routine owned by another package. With NO pre-image to
// restore that routine, a delete would BRICK it — so the build is REFUSED, never
// delete-all. This is checked BEFORE the side-effecting branch so a forced delete
// of even a side-effecting build still spares the declared-foreign routines.
func decideUninstall(class kids.ReversibilityClass, restoreKid, backoutKid string, force, hasGreenfieldAdds, hasForeignOverwrites bool) (uninstallAction, string) {
	if restoreKid != "" && backoutKid != "" {
		return actRefuse, "specify only one of --restore / --backout"
	}
	if restoreKid != "" {
		if hasGreenfieldAdds {
			return actPartition, "restore the overwritten routine(s) from the pre-image, then delete the greenfield-added routine(s) — ordered so the foreign routine is reinstated before its callees are removed"
		}
		return actRestore, "restore the provided pre-image snapshot"
	}
	if backoutKid != "" {
		return actBackout, "install the provided authored back-out"
	}
	if hasForeignOverwrites {
		if force {
			return actDelete, "FORCED — deleting ONLY the greenfield-added routine(s); the declared foreign overwrite(s) are NOT reversed (no pre-image) and are LEFT exactly as this patch left them. Re-install with --auto-snapshot to make the overwrite reversible"
		}
		return actRefuse, "this build declares a foreign-routine overwrite and no pre-image is available — deleting would BRICK the foreign/national routine it cannot restore. Re-install with --auto-snapshot (or pass --restore <pre-image>) to enable a clean reversal; --force deletes ONLY the greenfield routines and leaves the overwrite in place"
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
	KidFile    string `arg:"" help:"Path to the .KID whose install to reverse."`
	Restore    string `help:"Pre-image snapshot .KID to restore instead of deleting (class-1 reversal for a patched-over routine)."`
	Backout    string `help:"Authored back-out .KID to install instead of deleting (class-2 reversal of a side-effecting patch)."`
	Force      bool   `help:"Delete routines even for a side-effecting patch (UNSAFE: orphans install-time data/side-effects)."`
	Verify     bool   `help:"After a restore/back-out, confirm the live routines now match the re-applied artifact (verify-clean)."`
	Deregister bool   `help:"Also clear the PACKAGE #9.4 patch-history footprint a prior 'install --register-package' stamped, so $$PATCH^XPDUTL no longer reports this patch (symmetric to register)."`
}

type uninstallReport struct {
	Name           string   `json:"name"`
	Class          string   `json:"class"`
	Action         string   `json:"action"`
	Reason         string   `json:"reason"`
	AutoDetected   bool     `json:"autoDetected,omitempty"` // pre-image found via the sidecar convention
	Restored       []string `json:"restored,omitempty"`     // overwritten-foreign routines re-applied from the pre-image (actPartition)
	Deleted        []string `json:"deleted,omitempty"`      // greenfield-added routines deleted (actPartition)
	Done           bool     `json:"done"`
	Status         int      `json:"status,omitempty"`         // #9.7 status for restore/backout installs
	VerifyClean    string   `json:"verifyClean,omitempty"`    // "" | "clean" | "dirty" (when --verify)
	ForeignRestore string   `json:"foreignRestore,omitempty"` // D4: declared-foreign restore fidelity (exact | command-clean-provenance-drift | drift) when --verify
	Deregistered   bool     `json:"deregistered,omitempty"`   // #9.4 patch-history footprint cleared (when --deregister)
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

	// When a pre-image is in play, load it now and partition the build's routines
	// against it: routines the pre-image captured were overwritten (restore them),
	// routines the build adds that the pre-image lacks are greenfield (delete them).
	// A non-empty delete set means the build BOTH overwrote a foreign routine AND
	// added greenfield ones (the v-rpc-tap shape) — the actPartition case (BB1).
	var preBuild *kids.Build
	var preName string
	var restoreSet, deleteSet []string
	if restore != "" {
		var lerr error
		preName, preBuild, lerr = loadBuild(restore)
		if lerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", lerr.Error(), "")
		}
		// #3c: refuse a pre-image tampered after capture (auto-detected sidecar OR an
		// explicit --restore) before it is re-applied — never restore the wrong source.
		if ferr := verifySidecarIntegrity(preBuild, restore); ferr != nil {
			return ferr
		}
		restoreSet, deleteSet = partitionRoutines(b.RoutineNames(), preBuild.RoutineNames())
	}

	// F1: the build's OWN declaration (read offline from the .KID) of routines it
	// overwrote that belong to another package. This is the no-pre-image brick-path
	// signal — a declared-foreign routine must never be deleted without a pre-image
	// to restore it. The greenfield subset (build routines that are NOT declared
	// foreign) is what a forced delete may remove; for a declaration-free build it
	// is simply every routine (backward compatible).
	foreign := b.ForeignRoutines()
	_, greenfieldDelete := partitionRoutines(b.RoutineNames(), foreign)

	// Wrong/incomplete-sidecar guard: with a pre-image in play, every declared
	// foreign routine MUST be in it (so the partition restores, not deletes, it). A
	// foreign routine that falls into the delete set means the pre-image does not
	// carry it — restoring would not reinstate it and the partition would BRICK it.
	if restore != "" {
		if missing := intersectRoutines(foreign, deleteSet); len(missing) > 0 {
			return clikit.Fail(clikit.ExitRefused, "UNINSTALL_REFUSED",
				fmt.Sprintf("refusing to uninstall %s: the pre-image %s does not contain declared foreign routine(s) %s — "+
					"it is the wrong or an incomplete snapshot; restoring it would not reinstate them and the partition would delete (brick) them",
					name, restore, strings.Join(missing, ", ")),
				"capture a correct pre-image of the overwritten routine(s) with install --auto-snapshot")
		}
	}

	action, reason := decideUninstall(rev.Class, restore, c.Backout, c.Force, len(deleteSet) > 0, len(foreign) > 0)
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
		ir, ierr := runInstall(ctx, cl, rname, rname+" via v pkg uninstall --"+action.String(), rb.Pairs(), false, nil, nil)
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
			// D4: a pure-restore build that ALSO declared foreign overwrites (all of
			// them in the pre-image, so no greenfield to partition) gets the same
			// stricter national-overwrite grade against the re-applied pre-image.
			if action == actRestore && len(foreign) > 0 {
				fr, ferr := foreignRestoreVerdict(ctx, cl, rb, foreign)
				if ferr != nil {
					return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", ferr.Error(), "")
				}
				res.ForeignRestore = fr
			}
		}
	case actPartition:
		// Drop any stale snapshot provenance first: the restore re-installs the
		// pre-image build (install name "<name> PREIMAGE"), and a leftover #9.7 by
		// that name from a prior cycle makes install a no-op — the pre-image would
		// NOT be put back. Clearing the #9.7/#9.6 footprint (no routine touched)
		// makes the restore idempotent across repeated install/uninstall cycles.
		if cerr := clearBuildFootprint(ctx, cl, preName); cerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", cerr.Error(), "")
		}
		// (1) RESTORE the overwritten foreign routine(s) FIRST — re-install the
		// pre-image so the national routine (e.g. XWBPRS) is byte-restored and its
		// splice to the greenfield routines is gone BEFORE they are deleted.
		ir, ierr := runInstall(ctx, cl, preName, preName+" via v pkg uninstall --partition (restore)", preBuild.Pairs(), false, nil, nil)
		if ierr != nil {
			return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", ierr.Error(), "")
		}
		res.Status = ir.Status
		res.Restored = restoreSet
		if !ir.Installed {
			// Stop BEFORE the delete — never remove the greenfield callees while the
			// foreign caller may still be spliced to them (F-I/R21 ordering).
			return clikit.Fail(clikit.ExitRuntime, "NOT_UNINSTALLED",
				fmt.Sprintf("partitioned uninstall aborted: pre-image restore of %s did not complete (status %d) — greenfield routines left intact", preName, ir.Status), "")
		}
		// (2) DELETE the greenfield-added routine(s) + the build's non-routine
		// components + its #9.7/#9.6 entries. The overwritten foreign routine is NOT
		// in deleteSet, so it stays as just restored.
		ur, uerr := runUninstall(ctx, cl, name, deleteSet, b.Components(), fileNumStrings(b))
		if uerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", uerr.Error(), "")
		}
		res.Deleted = deleteSet
		res.Done = ur.Uninstalled
		// (3) Drop the snapshot's own #9.7/#9.6 provenance so the uninstall leaves no
		// "<name> PREIMAGE" ghost build registered on the engine.
		if res.Done {
			if cerr := clearBuildFootprint(ctx, cl, preName); cerr != nil {
				return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", cerr.Error(), "")
			}
		}
		// verify-clean: confirm the restored foreign routine(s) now match the pre-image.
		if c.Verify && res.Done {
			drift, derr := checkDrift(ctx, cl, preBuild)
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
			// D4: national-overwrite verify — grade the declared-foreign routine(s)
			// stricter than the line-2-blind drift: byte-equal on the checksum surface
			// (command lines) and on line-2 provenance.
			fr, ferr := foreignRestoreVerdict(ctx, cl, preBuild, foreign)
			if ferr != nil {
				return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", ferr.Error(), "")
			}
			res.ForeignRestore = fr
		}
	default: // actDelete
		// Delete the GREENFIELD subset only. For a declaration-free build that is
		// every routine (the original behavior); for a FORCED delete of a build with
		// declared foreign overwrites it EXCLUDES those — they are left in place, never
		// bricked, even under --force.
		ur, uerr := runUninstall(ctx, cl, name, greenfieldDelete, b.Components(), fileNumStrings(b))
		if uerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", uerr.Error(), "")
		}
		res.Done = ur.Uninstalled
		if len(foreign) > 0 {
			res.Deleted = greenfieldDelete // surface the partial-delete subset under --force
		}
	}

	// --deregister: clear the #9.4 patch-history footprint a prior
	// --register-package stamped (uninstall otherwise leaves it, so $$PATCH^XPDUTL
	// keeps reporting a ghost). Symmetric to install --register-package.
	if c.Deregister && res.Done {
		if reg := deregReg(name); reg != nil {
			removed, derr := runDeregister(ctx, cl, reg)
			if derr != nil {
				return clikit.Fail(clikit.ExitRuntime, "UNINSTALL_FAILED", derr.Error(), "")
			}
			res.Deregistered = removed
		}
	}

	if err := cc.Result(res, func() {
		cc.Title("pkg uninstall — " + c.Engine)
		fmt.Fprintf(cc.Stdout, "%s [%s] %s\n", cc.Accent(name), rev.ClassName, cc.Faint(reason))
		if res.AutoDetected {
			fmt.Fprintf(cc.Stdout, "  %s pre-image: %s\n", cc.Faint("auto-detected"), defaultPreimagePath(c.KidFile))
		}
		if action == actPartition {
			fmt.Fprintf(cc.Stdout, "  %s restore (foreign): %s\n", cc.Accent("1."), joinOr(res.Restored, "(none)"))
			fmt.Fprintf(cc.Stdout, "  %s delete (greenfield): %s\n", cc.Accent("2."), joinOr(res.Deleted, "(none)"))
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
		switch res.ForeignRestore {
		case kids.RestoreExact:
			fmt.Fprintln(cc.Stdout, cc.Success("foreign-restore: byte-identical (checksum surface + line-2 provenance)"))
		case kids.RestoreProvenanceDrift:
			fmt.Fprintln(cc.Stdout, cc.Faint("foreign-restore: checksum surface clean, but line-2 patch-history provenance differs"))
		case kids.RestoreDrift:
			fmt.Fprintln(cc.Stdout, cc.Failure("foreign-restore DRIFT: a command line differs — the foreign routine was NOT byte-restored"))
		}
		if c.Deregister {
			if res.Deregistered {
				fmt.Fprintln(cc.Stdout, cc.Success("deregistered: #9.4 patch-history footprint cleared"))
			} else {
				fmt.Fprintln(cc.Stdout, cc.Faint("deregister: no #9.4 patch-history footprint found"))
			}
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
