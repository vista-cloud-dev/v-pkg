package pkgcli

import (
	"context"
	"fmt"
	"sort"

	"github.com/vista-cloud-dev/clikit"
	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// dry-run / diff PREVIEW a build's effect against the LIVE system BEFORE mutating
// it — the KIDS convention's "Verify Checksums in Transport Global" + "Compare
// Transport Global to Current System" (menu options 2–4). Unlike `classify` (static,
// no engine) and `verify` (asserts a COMPLETED install), this reads the resident
// state and reports, per shipped artifact, what an install WOULD do.
//
// It is built ENTIRELY from the existing read-only probes (waterline rule 3, all
// through mdriver.Client): checkDrift (routine source compare), verifyContent
// (0-node compare), and the VerifyScript "B"-index presence probe. It NEVER stages
// ^XTMP("VPKGI"/"XPDI") and NEVER reaches EN^XPDIJ, so it is a true no-op: the
// engine is read, never written. Exit is ALWAYS 0 — the plan is informational, and
// pairs with the install attestation (#4): the plan is the pre-image of the record.

// Dry-run plan verdicts. Routines are graded NEW/CHANGED/IDENTICAL (the spec's
// vocabulary, relabeled from the drift probe). Components and FILE DD fields are
// graded would-add/would-change/identical from the 0-node compare; a presence-only
// component (no validated content claim — see component-type-coverage.md) can only
// be graded `present` (exists, content unverifiable) or would-add (absent), never
// would-change.
const (
	planNew         = "NEW"          // routine: absent on the engine — install adds it
	planChanged     = "CHANGED"      // routine: resident differs from incoming — install overwrites
	planIdentical   = "identical"    // no-op: resident already matches what would be installed
	planWouldAdd    = "would-add"    // component/file: absent — install would add it
	planWouldChange = "would-change" // component/file: present but differs — install would change it
	planPresent     = "present"      // presence-only component: exists, content not verifiable
)

// classifyRoutine relabels a checkDrift verdict (absent/applied/drifted) into the
// pre-install plan vocabulary: absent→NEW, applied→IDENTICAL (the live routine
// already matches the shipped source, line-2-blind), drifted→CHANGED.
func classifyRoutine(driftState string) string {
	switch driftState {
	case "absent":
		return planNew
	case "applied":
		return planIdentical
	default: // "drifted"
		return planChanged
	}
}

// classifyContent relabels a verifyContent verdict (absent/mismatch/ok) into the
// plan vocabulary for a content-verifiable component or FILE DD field: absent→
// would-add, mismatch→would-change, ok→identical.
func classifyContent(contentState string) string {
	switch contentState {
	case "absent":
		return planWouldAdd
	case "mismatch":
		return planWouldChange
	default: // "ok"
		return planIdentical
	}
}

// dryRunSummary rolls the per-artifact verdicts into four buckets — a quick,
// machine-checkable shape for a gate (greenfield = all New; post-install = all
// Identical). New/Changed/Identical span routines + components + files; Present
// counts presence-only components whose content could not be verified.
type dryRunSummary struct {
	New       int `json:"new"`
	Changed   int `json:"changed"`
	Identical int `json:"identical"`
	Present   int `json:"present"`
}

// dryRunReport is the read-only, machine-checkable plan of what installing one
// build WOULD change on the live engine — the pre-image of the attestation (#4).
type dryRunReport struct {
	Name       string            `json:"name"`
	Class      string            `json:"class"`
	Routines   map[string]string `json:"routines,omitempty"`   // name -> NEW | CHANGED | identical
	Components map[string]string `json:"components,omitempty"` // "<file>:<name>" -> would-add | would-change | identical | present
	Files      map[string]string `json:"files,omitempty"`      // "<file>#<field>" -> would-add | would-change | identical
	Summary    dryRunSummary     `json:"summary"`
}

// multiDryRunReport is the per-build roll-up for a distribution (one or many).
type multiDryRunReport struct {
	Builds []dryRunReport `json:"builds"`
}

// assembleDryRun builds the plan from the canned engine verdicts (pure — no engine
// call, so it is unit-testable offline). Routines come from the drift probe; each
// shipped component is graded by its content verdict when content-verifiable, else
// by the "B"-index presence probe (present / would-add, never would-change); FILE
// DD fields are graded by their 0-node content verdict.
func assembleDryRun(name, class string, drift, content map[string]string, presence map[string]bool, comps []kids.Component, files []kids.FileContent) dryRunReport {
	rep := dryRunReport{
		Name: name, Class: class,
		Routines: map[string]string{}, Components: map[string]string{}, Files: map[string]string{},
	}
	for rt, state := range drift {
		rep.Routines[rt] = classifyRoutine(state)
	}
	for _, c := range comps {
		for _, n := range c.Names {
			key := c.FileStr + ":" + n
			switch {
			case content[key] != "": // a content-verifiable type — the 0-node compare wins
				rep.Components[key] = classifyContent(content[key])
			case presence[key]: // presence-only and present — content not verifiable
				rep.Components[key] = planPresent
			default: // absent
				rep.Components[key] = planWouldAdd
			}
		}
	}
	for _, f := range files {
		key := f.FileStr + "#" + f.Field
		rep.Files[key] = classifyContent(content[key])
	}
	rep.Summary = summarizeDryRun(rep)
	return rep
}

// summarizeDryRun rolls the per-artifact verdicts into the four-bucket summary.
func summarizeDryRun(rep dryRunReport) dryRunSummary {
	var s dryRunSummary
	tally := func(v string) {
		switch v {
		case planNew, planWouldAdd:
			s.New++
		case planChanged, planWouldChange:
			s.Changed++
		case planIdentical:
			s.Identical++
		case planPresent:
			s.Present++
		}
	}
	for _, v := range rep.Routines {
		tally(v)
	}
	for _, v := range rep.Components {
		tally(v)
	}
	for _, v := range rep.Files {
		tally(v)
	}
	return s
}

// dryRunPlan generates the plan for one build over the driver, using ONLY the
// read-only probes (checkDrift / verifyContent / the presence probe). Nothing is
// staged or filed — the engine is read, never written.
func dryRunPlan(ctx context.Context, cl *mdriver.Client, name, className string, b *kids.Build) (dryRunReport, error) {
	drift, err := checkDrift(ctx, cl, b)
	if err != nil {
		return dryRunReport{Name: name}, err
	}
	content, err := verifyContent(ctx, cl, b.EntryContents(), b.FileContents())
	if err != nil {
		return dryRunReport{Name: name}, err
	}
	// Presence for the presence-only component types (the content probe covers only
	// content-verifiable types). runVerify's "B"-index probe is read-only; the #9.7
	// status it also reads is unused here.
	pres, err := runVerify(ctx, cl, name, nil, b.Components(), nil)
	if err != nil {
		return dryRunReport{Name: name}, err
	}
	return assembleDryRun(name, className, drift, content, pres.Components, b.Components(), b.FileContents()), nil
}

// runDryRun generates and renders the plan for every build in a distribution and
// returns nil (exit 0 ALWAYS — read-only preview). Shared by `install --dry-run`
// and the thin `diff` alias.
func runDryRun(ctx context.Context, cc *clikit.Context, cl *mdriver.Client, engine string, k *kids.KID) error {
	var out multiDryRunReport
	for _, name := range k.InstallNames {
		b := k.Builds[name]
		plan, err := dryRunPlan(ctx, cl, name, kids.ClassifyBuild(name, b).ClassName, b)
		if err != nil {
			return clikit.Fail(clikit.ExitRuntime, "DRY_RUN_FAILED", err.Error(),
				"confirm the driver connection")
		}
		out.Builds = append(out.Builds, plan)
	}
	return cc.Result(out, func() {
		cc.Title("pkg diff (dry-run) — " + engine)
		for _, plan := range out.Builds {
			fmt.Fprintf(cc.Stdout, "%s [%s]\n", cc.Accent(plan.Name), plan.Class)
			renderDryRunSection(cc, "routine", plan.Routines)
			renderDryRunSection(cc, "component", plan.Components)
			renderDryRunSection(cc, "file", plan.Files)
			s := plan.Summary
			fmt.Fprintf(cc.Stdout, "  %s %d new, %d changed, %d identical, %d present (read-only — engine NOT modified)\n",
				cc.Faint("plan:"), s.New, s.Changed, s.Identical, s.Present)
		}
	})
}

// renderDryRunSection prints one verdict map in stable key order, coloring the
// verdict by whether it is a no-op (identical/present), an add (green) or a change.
func renderDryRunSection(cc *clikit.Context, kind string, m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		var mark string
		switch v {
		case planIdentical, planPresent:
			mark = cc.Faint(v)
		case planNew, planWouldAdd:
			mark = cc.Success(v)
		default: // CHANGED / would-change
			mark = cc.Failure(v)
		}
		fmt.Fprintf(cc.Stdout, "  %s %s %s\n", kind, k, mark)
	}
}

// --- diff -------------------------------------------------------------------

type diffCmd struct {
	engineConn
	KidFile string `arg:"" help:"Path to the .KID to compare against the live engine (read-only; reports NEW/CHANGED/identical per artifact)."`
}

func (c *diffCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	if len(k.InstallNames) == 0 {
		return clikit.Fail(clikit.ExitUsage, "NO_BUILD", "no build found in "+c.KidFile, "")
	}
	cl, err := c.client()
	if err != nil {
		return c.noDriver(err)
	}
	return runDryRun(context.Background(), cc, cl, c.Engine, k)
}
