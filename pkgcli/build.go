package pkgcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vista-cloud-dev/v-pkg/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/buildspec"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// buildCmd is `v pkg build` (VSL T0a.2): assemble a KIDS transport global from a
// declarative build spec (kids/<pkg>.build.json) + the routine source, producing
// a NORMALIZED, byte-identical export — the deterministic-build invariant
// (coordination plan §7.2 #2). Unlike `assemble` (which reassembles an existing
// decomposed .KID tree), `build` constructs the package from its git source of
// truth.
type buildCmd struct {
	Spec string `arg:"" help:"Path to the kids/<pkg>.build.json build spec."`
	Src  string `help:"Directory holding the routine source (<routine>.m)." default:"src" placeholder:"DIR"`
	Out  string `help:"Output .KID path (default: dist/kids/<pkg>.kids)." placeholder:"PATH"`
}

type buildResult struct {
	InstallName    string `json:"installName"`
	Out            string `json:"out"`
	Routines       int    `json:"routines"`
	ParamDefs      int    `json:"paramDefs,omitempty"`
	RequiredBuilds int    `json:"requiredBuilds,omitempty"`
}

func (c *buildCmd) Run(cc *clikit.Context) error {
	spec, err := buildspec.Load(c.Spec)
	if err != nil {
		return clikit.Fail(clikit.ExitUsage, "BAD_SPEC", err.Error(), "fix the build spec")
	}

	rtns := make([]kids.RoutineSrc, 0, len(spec.Components.Routines))
	for _, name := range spec.Components.Routines {
		p := filepath.Join(c.Src, name+".m")
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", rerr.Error(),
				"stage the routine source under --src (e.g. "+filepath.Join(c.Src, name+".m")+")")
		}
		rtns = append(rtns, kids.RoutineSrc{Name: name, Lines: routineLines(data)})
	}

	paramDefs, perr := resolveParamDefs(spec.Components.ParameterDefinitions)
	if perr != nil {
		return clikit.Fail(clikit.ExitUsage, "BAD_SPEC", perr.Error(), "fix the parameterDefinitions in the build spec")
	}
	reqBuilds := resolveRequiredBuilds(spec.RequiredBuilds)

	pairs := kids.MakeBuildPairs(kids.BuildInput{
		InstallName:    spec.InstallName(),
		Namespace:      spec.Package,
		Routines:       rtns,
		ParamDefs:      paramDefs,
		RequiredBuilds: reqBuilds,
	})

	out := c.Out
	if out == "" {
		out = filepath.Join("dist", "kids", spec.Package+".kids")
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
	}
	if err := kids.WriteKID([]string{spec.InstallName()},
		map[string][]kids.Pair{spec.InstallName(): pairs}, out); err != nil {
		return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
	}

	return cc.Result(buildResult{
		InstallName: spec.InstallName(), Out: out, Routines: len(rtns),
		ParamDefs: len(paramDefs), RequiredBuilds: len(reqBuilds),
	}, func() {
		cc.Title("pkg build")
		fmt.Fprintf(cc.Stdout, "%s built %s (%d routine(s), %d param def(s), %d required build(s)) → %s\n",
			cc.Success("ok"), cc.Accent(spec.InstallName()), len(rtns), len(paramDefs), len(reqBuilds), cc.Accent(out))
	})
}

// resolveParamDefs maps the human PARAMETER DEFINITION spec onto the kids emit
// shape — value-data-type name → #8989.51 code, entity abbreviation → #8989.518
// IEN. The spec is already validated, so the lookups are guaranteed present;
// the error guards a future spec that slips an unknown key past validation.
func resolveParamDefs(defs []buildspec.ParamDef) ([]kids.ParamDef, error) {
	out := make([]kids.ParamDef, 0, len(defs))
	for _, d := range defs {
		dtName := d.DataType
		if dtName == "" {
			dtName = "free text"
		}
		dt, ok := buildspec.ParamDataTypeCode[dtName]
		if !ok {
			return nil, fmt.Errorf("parameter %s: unknown data type %q", d.Name, d.DataType)
		}
		ents := make([]kids.ParamEntity, 0, len(d.Entities))
		for _, e := range d.Entities {
			ien, ok := buildspec.ParamEntityIEN[e.Entity]
			if !ok {
				return nil, fmt.Errorf("parameter %s: unknown entity %q", d.Name, e.Entity)
			}
			ents = append(ents, kids.ParamEntity{EntityIEN: ien, Precedence: e.Precedence})
		}
		out = append(out, kids.ParamDef{
			Name: d.Name, DisplayText: d.DisplayText, DataTypeCode: dt, Entities: ents,
		})
	}
	return out, nil
}

// resolveRequiredBuilds maps the spec's Required Builds onto the kids emit shape,
// turning the action phrase into its #9.611 ACTION code.
func resolveRequiredBuilds(reqs []buildspec.RequiredBuild) []kids.ReqBuild {
	out := make([]kids.ReqBuild, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, kids.ReqBuild{Name: r.Name, Action: buildspec.RequiredBuildActionCode[r.Action]})
	}
	return out
}

// routineLines splits routine source into lines, dropping a single trailing
// newline (so a normal text file does not yield a spurious empty final line).
func routineLines(data []byte) []string {
	s := strings.TrimRight(string(data), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
