package pkgcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/vista-cloud-dev/clikit"
	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// snapshot / restore make v-pkg pre-image aware (the patch-existing-routines
// proposal). Before a patch overwrites a national routine, `snapshot` captures
// the routine's LIVE pre-image off the engine into a restorable .KID; `restore`
// re-applies that pre-image. Together they are the class-1 (pure-overwrite)
// reversal path — the only class where this is a COMPLETE undo (per
// docs/kids-corpus-findings.md, ~36% of real patches). For side-effecting
// patches the snapshot is honest provenance, NOT a reversal guarantee, and
// snapshot says so (it cannot capture the effects of install code / data).
//
// Engine access is the driver stack only (waterline rule 3): the pre-image READ
// is readRoutinePreimage (read-only, through mdriver.Client); the restore WRITE
// reuses the proven runInstall path and is gated behind --commit.

// snapshotName derives the install name for the pre-image build. The default
// suffixes " PREIMAGE" onto the original install name (a distinct #9.7 identity
// so the snapshot install never collides with the patch it backs up); an
// explicit --name overrides.
func snapshotName(orig, override string) string {
	if override != "" {
		return override
	}
	return orig + " PREIMAGE"
}

// snapshotNamespace extracts the leading namespace token from an install name
// (the run of alphanumerics before the first separator — "*" or space), used
// only for the snapshot build's cosmetic BLD namespace field. Falls back to
// "VPKG" when the name has no leading namespace.
func snapshotNamespace(orig string) string {
	i := strings.IndexFunc(orig, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	switch {
	case i < 0:
		return orig
	case i == 0:
		return "VPKG"
	default:
		return orig[:i]
	}
}

// buildSnapshotPairs assembles an installable routine-only KIDS transport from
// captured pre-image source — the inverse target a class-1 restore re-applies.
func buildSnapshotPairs(name, namespace string, captured []kids.RoutineSrc) []kids.Pair {
	return kids.MakeBuildPairs(kids.BuildInput{
		InstallName: name,
		Namespace:   namespace,
		Routines:    captured,
	})
}

// readRoutinePreimage reads a routine's live source off the engine and reports
// whether it is present. An ABSENT routine (no source) is the greenfield case —
// it returns present=false with NO error (the patch adds it; reversal = delete,
// which has no pre-image). A driver/engine fault propagates as an error.
func readRoutinePreimage(ctx context.Context, cl *mdriver.Client, name string) ([]string, bool, error) {
	if !validRoutineName(name) {
		return nil, false, fmt.Errorf("invalid routine name %q", name)
	}
	markers, _, err := runMScript(ctx, cl, rtnRead, readRoutineBody(name))
	if err != nil {
		return nil, false, err
	}
	lines := parseRoutineLines(markers)
	return lines, len(lines) > 0, nil
}

// captureRoutinePreimages reads every named routine, splitting them into the
// captured (present, with source) and absent (greenfield) sets.
func captureRoutinePreimages(ctx context.Context, cl *mdriver.Client, names []string) (captured []kids.RoutineSrc, absent []string, err error) {
	for _, n := range names {
		lines, present, rerr := readRoutinePreimage(ctx, cl, n)
		if rerr != nil {
			return nil, nil, fmt.Errorf("read %s: %w", n, rerr)
		}
		if !present {
			absent = append(absent, n)
			continue
		}
		captured = append(captured, kids.RoutineSrc{Name: n, Lines: lines})
	}
	return captured, absent, nil
}

// --- snapshot ---------------------------------------------------------------

type snapshotResult struct {
	Name         string   `json:"name"`                 // the patch being snapshotted
	SnapshotName string   `json:"snapshotName"`         // the pre-image build's install name
	Class        string   `json:"class"`                // reversibility class of the patch
	Captured     []string `json:"captured"`             // routines whose pre-image was saved
	Absent       []string `json:"absent"`               // routines the patch ADDS (no pre-image; greenfield)
	NonRoutine   bool     `json:"nonRoutine"`           // the patch also ships non-routine components
	Uncaptured   []string `json:"uncaptured,omitempty"` // non-routine components this snapshot does NOT capture
	CompleteUndo bool     `json:"completeUndo"`         // true iff restoring the snapshot fully reverses the patch
	Out          string   `json:"out"`                  // path the snapshot .KID was written to
}

// uncapturedComponents itemizes the NON-routine components a routine-only
// snapshot cannot capture, so the operator knows exactly what an authored
// back-out (uninstall --backout) must cover — nothing is silently dropped. These
// are reversed by the patch's authored inverse, NOT by a generic pre-image (their
// install code has no generic inverse; capturing a value would over-claim
// reversibility — see the proposal). params are listed by name; the 8989.51 file
// (which holds them) is therefore skipped in the bare file-number list.
func uncapturedComponents(params, fileManFiles, fileDDFiles []string) []string {
	var out []string
	for _, p := range params {
		out = append(out, "param: "+p)
	}
	for _, f := range fileDDFiles {
		out = append(out, "FileMan FILE (DD/data) #"+f)
	}
	for _, f := range fileManFiles {
		if f == "8989.51" { // already listed by param name above
			continue
		}
		out = append(out, "FileMan entries in file #"+f)
	}
	return out
}

type snapshotCmd struct {
	engineConn
	KidFile string `arg:"" help:"the .KID about to be installed — the live pre-image of its routines is captured."`
	OutKid  string `arg:"" help:"output path for the pre-image snapshot .KID (what 'v pkg restore' re-applies)."`
	Name    string `help:"install name for the snapshot build (default: \"<orig> PREIMAGE\")."`
}

func (c *snapshotCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	if len(k.InstallNames) != 1 {
		return clikit.Fail(clikit.ExitUsage, "MULTI_BUILD",
			fmt.Sprintf("snapshot expects exactly one build, found %d", len(k.InstallNames)),
			"snapshot one build at a time")
	}
	name := k.InstallNames[0]
	b := k.Builds[name]
	rev := kids.Classify(k)
	bc := rev.Builds[0]

	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	captured, absent, err := captureRoutinePreimages(context.Background(), cl, b.RoutineNames())
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(),
			"confirm the driver connection and that the target routines are readable")
	}

	snapName := snapshotName(name, c.Name)
	pairs := buildSnapshotPairs(snapName, snapshotNamespace(name), captured)
	if err := kids.WriteKID([]string{snapName}, map[string][]kids.Pair{snapName: pairs}, c.OutKid); err != nil {
		return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
	}

	// The snapshot fully reverses the patch ONLY when it is pure-overwrite AND
	// every component it overwrites is a routine we captured (no greenfield adds,
	// no non-routine components). Otherwise it is provenance, not a guarantee.
	nonRoutine := bc.ShipsFileManEntries || bc.ShipsFileDD
	uncaptured := uncapturedComponents(b.ParamDefNames(), bc.FileManFiles, bc.FileDDFiles)
	res := snapshotResult{
		Name:         name,
		SnapshotName: snapName,
		Class:        bc.ClassName,
		Captured:     routineNames(captured),
		Absent:       absent,
		NonRoutine:   nonRoutine,
		Uncaptured:   uncaptured,
		CompleteUndo: bc.Class == kids.ClassPureOverwrite && len(absent) == 0 && !nonRoutine,
		Out:          c.OutKid,
	}
	return cc.Result(res, func() {
		cc.Title("pkg snapshot — " + c.Engine)
		fmt.Fprintf(cc.Stdout, "%s captured %d routine pre-image(s) → %s\n",
			cc.Success("ok"), len(res.Captured), cc.Accent(res.Out))
		if len(res.Absent) > 0 {
			fmt.Fprintf(cc.Stdout, "  %s adds (no pre-image, greenfield): %s\n",
				cc.Faint("absent"), strings.Join(res.Absent, ", "))
		}
		if res.CompleteUndo {
			fmt.Fprintln(cc.Stdout, cc.Success("class-1 pure-overwrite: `v pkg restore` fully reverses this patch."))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Warning("NOT a complete undo ("+res.Class+"): this snapshot is provenance, not a reversal guarantee."))
			for _, u := range res.Uncaptured {
				fmt.Fprintf(cc.Stdout, "  %s %s\n", cc.Failure("uncaptured"), u)
			}
			fmt.Fprintln(cc.Stdout, cc.Faint("  these reverse via the patch's authored back-out (uninstall --backout), not a generic pre-image."))
		}
	})
}

// routineNames returns just the names from a captured set.
func routineNames(rs []kids.RoutineSrc) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}
