package pkgcli

import (
	"fmt"

	"github.com/vista-cloud-dev/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// classifyCmd is `v pkg classify <kid>`: statically derive the reversibility
// class of a KIDS distribution (pure-overwrite vs side-effecting) from the
// transport global alone — no live engine. It is the keystone the
// patch-existing-routines proposal requires before snapshot/restore and
// class-aware uninstall: it answers "can this patch be reversed by restoring a
// pre-image, or does it run install code / file data that has no generic
// inverse?" (~64% of real patches are the latter — see docs/kids-corpus-findings.md).
type classifyCmd struct {
	KidFile string `arg:"" help:"Path to the .KID file."`
}

func (c *classifyCmd) Run(cc *clikit.Context) error {
	k, err := kids.ParseKID(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	rev := kids.Classify(k)
	return cc.Result(rev, func() {
		cc.Title("pkg classify")
		fmt.Fprintf(cc.Stdout, "reversibility: %s\n", reversibilityLabel(cc, rev.Class))
		for _, b := range rev.Builds {
			fmt.Fprintf(cc.Stdout, "  %s %s\n", cc.Accent(b.Build), cc.Faint("["+b.ClassName+"]"))
			fmt.Fprintf(cc.Stdout, "    routines:      %d\n", b.RoutineCount)
			if b.HasInstallCode {
				for role, entry := range b.InstallCode {
					fmt.Fprintf(cc.Stdout, "    install code:  %s = %s\n", role, entry)
				}
			}
			if b.ShipsFileManEntries {
				fmt.Fprintf(cc.Stdout, "    FileMan entries in file(s): %v\n", b.FileManFiles)
			}
			if b.ShipsFileDD {
				fmt.Fprintf(cc.Stdout, "    ships FILE (DD/data): %v\n", b.FileDDFiles)
			}
		}
		if rev.Class == kids.ClassPureOverwrite {
			fmt.Fprintln(cc.Stdout, cc.Faint("  → snapshot/restore can fully reverse this (uninstall = restore pre-image)."))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Faint("  → NOT restore-reversible: needs an authored back-out or a forward back-out patch."))
		}
	})
}

func reversibilityLabel(cc *clikit.Context, c kids.ReversibilityClass) string {
	if c == kids.ClassPureOverwrite {
		return cc.Success(c.String())
	}
	return cc.Accent(c.String())
}
