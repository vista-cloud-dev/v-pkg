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
	InstallName string `json:"installName"`
	Out         string `json:"out"`
	Routines    int    `json:"routines"`
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

	pairs := kids.MakeBuildPairs(kids.BuildInput{
		InstallName: spec.InstallName(),
		Namespace:   spec.Package,
		Routines:    rtns,
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

	return cc.Result(buildResult{InstallName: spec.InstallName(), Out: out, Routines: len(rtns)}, func() {
		cc.Title("pkg build")
		fmt.Fprintf(cc.Stdout, "%s built %s (%d routine(s)) → %s\n",
			cc.Success("ok"), cc.Accent(spec.InstallName()), len(rtns), cc.Accent(out))
	})
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
