// Package pkgcli is the importable command surface of the v-pkg domain (the
// `v pkg` KIDS tool). It is exported so the `v` umbrella can mount it in-process
// as `v pkg <verb>` (static-pinned composition, v-cli-platform §3) while the
// standalone `v-pkg` binary embeds the same Commands for top-level verbs. The
// offline verbs (decompose / assemble / roundtrip / canonicalize / parse / lint)
// are the byte-identical port of py-kids-vc / XPDK2VC; the live KIDS lifecycle
// (build / install / verify / uninstall / snapshot / restore) runs over the
// m-driver-sdk engine seam.
package pkgcli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vista-cloud-dev/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// Commands is the v-pkg verb set. Embed it (anonymous) for top-level verbs in
// the standalone binary, or mount it as a named field (`Pkg Commands` with
// cmd:"" name:"pkg") under the `v` umbrella for `v pkg <verb>`.
type Commands struct {
	Parse        parseCmd        `cmd:"" group:"Inspect" help:"Parse a .KID file and summarize its builds and sections."`
	Decompose    decomposeCmd    `cmd:"" group:"Transform" help:"Split a .KID into a per-component KIDComponents/ tree."`
	Assemble     assembleCmd     `cmd:"" group:"Transform" help:"Reassemble a component tree back into a .KID."`
	Roundtrip    roundtripCmd    `cmd:"" group:"Transform" help:"Verify decompose→assemble reproduces the build (exit 3 on drift)."`
	Canonicalize canonicalizeCmd `cmd:"" group:"Transform" help:"Substitute install-time IENs with \"IEN\" in a tree (LOSSY; review-only)."`
	Classify     classifyCmd     `cmd:"" group:"Inspect" help:"Derive a .KID's reversibility class (pure-overwrite vs side-effecting) from its structure — no engine."`
	Diff         diffCmd         `cmd:"" group:"Inspect" help:"Compare a .KID to the live engine WITHOUT mutating it: per-artifact NEW/CHANGED/identical preview (read-only; exit 0)."`
	Lint         lintCmd         `cmd:"" group:"Inspect" help:"Run the PIKS data-class gate over a .KID (exit 3 on a blocked file)."`
	Build        buildCmd        `cmd:"" group:"Build & install" help:"Build a KIDS transport global from a declarative build spec (deterministic, normalized export)."`
	Install      installCmd      `cmd:"" group:"Build & install" help:"Install a built .KID on a live engine over the driver (non-interactive KIDS load+install)."`
	Snapshot     snapshotCmd     `cmd:"" group:"Back-out" help:"Capture the live pre-image of a patch's routines into a restorable .KID (class-1 reversal)."`
	Restore      restoreCmd      `cmd:"" group:"Back-out" help:"Re-apply a pre-image snapshot .KID to revert routines to stock (preview by default; --commit installs)."`
	Verify       verifyCmd       `cmd:"" group:"Build & install" help:"Verify a .KID's install on a live engine (#9.7 status + per-routine presence)."`
	Uninstall    uninstallCmd    `cmd:"" group:"Back-out" help:"Uninstall a .KID from a live engine (routine-only back-out: routines + #9.7/#9.6)."`
}

// --- parse -------------------------------------------------------------------

type parseCmd struct {
	KidFile string `arg:"" help:"Path to the .KID file."`
}

type buildSummary struct {
	Name       string         `json:"name"`
	Subscripts int            `json:"subscripts"`
	Sections   map[string]int `json:"sections"`
}

type parseResult struct {
	InstallNames []string       `json:"installNames"`
	Builds       []buildSummary `json:"builds"`
}

func (c *parseCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	res := parseResult{InstallNames: k.InstallNames}
	for _, name := range k.InstallNames {
		b := k.Builds[name]
		bs := buildSummary{Name: name, Subscripts: b.Len(), Sections: map[string]int{}}
		for _, p := range b.Pairs() {
			bs.Sections[p.Subs.Section()]++
		}
		res.Builds = append(res.Builds, bs)
	}
	return cc.Result(res, func() {
		cc.Title("pkg parse")
		fmt.Fprintf(cc.Stdout, "install_names: %s\n", strings.Join(res.InstallNames, ", "))
		for _, bs := range res.Builds {
			fmt.Fprintf(cc.Stdout, "  %s %s\n", cc.Accent(bs.Name), cc.Faint(fmt.Sprintf("(%d subscripts)", bs.Subscripts)))
			secs := make([]string, 0, len(bs.Sections))
			for s := range bs.Sections {
				secs = append(secs, s)
			}
			sort.Strings(secs)
			for _, s := range secs {
				fmt.Fprintf(cc.Stdout, "    %-8s %d\n", s, bs.Sections[s])
			}
		}
	})
}

// --- decompose ---------------------------------------------------------------

type decomposeCmd struct {
	KidFile   string `arg:"" help:"Path to the .KID file."`
	OutputDir string `arg:"" help:"Output directory for the component tree (replaced if it exists)."`
}

type decomposeResult struct {
	OutputDir string   `json:"outputDir"`
	Builds    []string `json:"builds"`
}

func (c *decomposeCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	if _, err := os.Stat(c.OutputDir); err == nil {
		if err := os.RemoveAll(c.OutputDir); err != nil {
			return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
		}
	}
	for _, name := range k.InstallNames {
		dir := filepath.Join(c.OutputDir, kids.PatchDescriptorToDir(name), "KIDComponents")
		if err := kids.DecomposeBuild(k.Builds[name], dir); err != nil {
			return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
		}
	}
	res := decomposeResult{OutputDir: c.OutputDir, Builds: k.InstallNames}
	return cc.Result(res, func() {
		cc.Title("pkg decompose")
		fmt.Fprintf(cc.Stdout, "%s decomposed %d build(s) to %s\n",
			cc.Success("ok"), len(res.Builds), cc.Accent(res.OutputDir))
	})
}

// --- assemble ----------------------------------------------------------------

type assembleCmd struct {
	InputDir  string `arg:"" help:"Component tree (a directory of <build>/KIDComponents/)."`
	OutputKid string `arg:"" help:"Output .KID path."`
}

type assembleResult struct {
	OutputKid    string   `json:"outputKid"`
	InstallNames []string `json:"installNames"`
}

func (c *assembleCmd) Run(cc *clikit.Context) error {
	entries, err := os.ReadDir(c.InputDir)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	var dirNames []string
	for _, e := range entries {
		if e.IsDir() {
			dirNames = append(dirNames, e.Name())
		}
	}
	sort.Strings(dirNames)

	var installNames []string
	buildsPairs := map[string][]kids.Pair{}
	for _, dn := range dirNames {
		comp := filepath.Join(c.InputDir, dn, "KIDComponents")
		if _, err := os.Stat(comp); err != nil {
			continue
		}
		installName := recoverInstallName(dn)
		pairs, err := kids.AssembleBuild(comp, installName)
		if err != nil {
			return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
		}
		installNames = append(installNames, installName)
		buildsPairs[installName] = pairs
	}
	if err := kids.WriteKID(installNames, buildsPairs, c.OutputKid); err != nil {
		return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
	}
	res := assembleResult{OutputKid: c.OutputKid, InstallNames: installNames}
	return cc.Result(res, func() {
		cc.Title("pkg assemble")
		fmt.Fprintf(cc.Stdout, "%s assembled %d build(s) → %s\n",
			cc.Success("ok"), len(installNames), cc.Accent(res.OutputKid))
	})
}

// recoverInstallName reverses PatchDescriptorToDir: VMTEST_1.0_1 → VMTEST*1.0*1.
// Port of the directory-name parsing in py-kids-vc's _cmd_assemble.
func recoverInstallName(dirName string) string {
	parts := strings.Split(dirName, "_")
	if len(parts) >= 3 {
		return parts[0] + "*" + strings.Join(parts[1:len(parts)-1], ".") + "*" + parts[len(parts)-1]
	}
	return dirName
}

// --- roundtrip ---------------------------------------------------------------

type roundtripCmd struct {
	KidFile string `arg:"" help:"Path to the .KID file."`
}

func (c *roundtripCmd) Run(cc *clikit.Context) error {
	res, err := kids.Roundtrip(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "ROUNDTRIP_ERROR", err.Error(), "")
	}
	if err := cc.Result(res, func() {
		cc.Title("pkg roundtrip")
		if res.OK {
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("roundtrip OK: %s", res.File)))
			fmt.Fprintf(cc.Stdout, "  builds: %d\n  pairs:  %d\n  canonicalized equality verified\n", res.Builds, res.Pairs)
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("roundtrip FAIL: %s", res.File)))
			for _, d := range res.Diff {
				fmt.Fprintf(cc.Stdout, "  build %s: %d → %d pairs\n    - %s\n    + %s\n",
					d.Build, d.PairsA, d.PairsB, d.FirstA, d.FirstB)
			}
		}
	}); err != nil {
		return err
	}
	if !res.OK {
		return clikit.Fail(clikit.ExitCheck, "ROUNDTRIP_FAILED",
			fmt.Sprintf("%s did not round-trip", res.File), "inspect the diff above")
	}
	return nil
}

// --- canonicalize ------------------------------------------------------------

type canonicalizeCmd struct {
	DecompDir string `arg:"" help:"A decomposed component tree to rewrite in place (LOSSY)."`
}

func (c *canonicalizeCmd) Run(cc *clikit.Context) error {
	stats, err := kids.CanonicalizeIENs(c.DecompDir)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "CANONICALIZE_FAILED", err.Error(), "")
	}
	return cc.Result(stats, func() {
		cc.Title("pkg canonicalize")
		fmt.Fprintf(cc.Stdout, "%s %d IEN substitution(s)\n", cc.Success("ok"), stats.Total())
		fmt.Fprintf(cc.Stdout, "  BLD: %d\n  KRN: %d\n", stats.BLD, stats.KRN)
	})
}

// --- lint (PIKS data-class gate) ---------------------------------------------

type lintCmd struct {
	KidFile string `arg:"" help:"Path to the .KID file."`
	Piks    string `name:"piks" help:"Path to an authoritative PIKS classification table (TSV: filenumber<TAB>class)."`
	Strict  bool   `help:"Treat unclassified data files as gate failures (fail-closed)."`
}

func (c *lintCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	classifier := kids.NewPIKSClassifier()
	if c.Piks != "" {
		if err := classifier.LoadPIKS(c.Piks); err != nil {
			return clikit.Fail(clikit.ExitUsage, "BAD_PIKS", err.Error(), "")
		}
	}
	res := kids.LintDataClass(k, classifier, c.Strict)

	diags := make([]clikit.Diagnostic, 0, len(res.Findings))
	for _, f := range res.Findings {
		diags = append(diags, clikit.Diagnostic{
			File:     "file " + f.File,
			Rule:     "KIDS-DATA-CLASS",
			Severity: f.Severity,
			Message:  fmt.Sprintf("%s [%s] %s", f.Section, f.Class, f.Message),
		})
	}
	summary := map[string]int{
		"dataFiles": res.DataFiles, "blocked": res.Blocked, "unclassified": res.Unclassified,
	}
	if err := cc.Diagnostics(summary, diags, func() {
		cc.Title("pkg lint — data-class gate")
		for _, d := range diags {
			fmt.Fprintf(cc.Stdout, "%s  %s  %s\n", cc.Severity(d.Severity), cc.Faint(d.File), d.Message)
		}
		if res.OK {
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("data-class gate clean (%d data file(s))", res.DataFiles)))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("data-class gate FAILED — %d blocked file(s)", res.Blocked)))
		}
	}); err != nil {
		return err
	}
	if !res.OK {
		refused := res.Blocked
		if c.Strict {
			refused += res.Unclassified
		}
		return clikit.Fail(clikit.ExitCheck, "DATA_CLASS_GATE",
			fmt.Sprintf("%d file(s) refused by the data-class gate", refused),
			"Patient/Institution-class operational data must not be versioned")
	}
	return nil
}
