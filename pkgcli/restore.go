package pkgcli

import (
	"context"
	"fmt"

	"github.com/vista-cloud-dev/v-pkg/clikit"
)

// restore re-applies a pre-image snapshot .KID (produced by `v pkg snapshot`),
// putting the captured routines back as they were before a patch. Mechanically
// it IS install-of-the-pre-image (proposal: "restore = install-of-pre-image, a
// thin alias"), so it reuses the proven runInstall path. It PREVIEWS by default —
// the engine is not written without --commit — because the restore overwrites live
// (national) routines, the deliberately-gated step.
//
// restore reverses class-1 (pure-overwrite) patches completely. For a
// side-effecting patch the snapshot only carried routine pre-images, so restore
// puts the code back but cannot undo install-time data/side-effects — run the
// patch's authored back-out for that. (`v pkg snapshot` flags this at capture.)

type restoreResult struct {
	Name      string   `json:"name"`
	Routines  []string `json:"routines"`
	Committed bool     `json:"committed"` // false = preview only (engine untouched)
	Installed bool     `json:"installed,omitempty"`
	Status    int      `json:"status,omitempty"`
}

type restoreCmd struct {
	engineConn
	KidFile string `arg:"" help:"the pre-image snapshot .KID to re-apply (restore to its captured state)."`
	Commit  bool   `help:"DESTRUCTIVE: install the snapshot on the live engine (overwrites the live routines). Default off = preview."`
}

func (c *restoreCmd) Run(cc *clikit.Context) error {
	name, b, err := loadBuild(c.KidFile)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", err.Error(), "")
	}
	res := restoreResult{Name: name, Routines: b.RoutineNames()}

	if !c.Commit {
		return cc.Result(res, func() {
			cc.Title("pkg restore (preview) — " + c.Engine)
			fmt.Fprintf(cc.Stdout, "%s would restore %d routine(s) from %s: %s\n",
				cc.Success("ok"), len(res.Routines), cc.Accent(name), joinOr(res.Routines, "(none)"))
			fmt.Fprintln(cc.Stdout, cc.Faint("  engine NOT modified — pass --commit to re-apply the pre-image."))
		})
	}

	cl, derr := c.client()
	if derr != nil {
		return c.noDriver(derr)
	}
	ir, ierr := runInstall(context.Background(), cl, name, name+" via v pkg restore", b.Pairs())
	if ierr != nil {
		return clikit.Fail(clikit.ExitRuntime, "RESTORE_FAILED", ierr.Error(), "")
	}
	res.Committed = true
	res.Installed = ir.Installed
	res.Status = ir.Status

	if err := cc.Result(res, func() {
		cc.Title("pkg restore — " + c.Engine)
		if ir.Installed {
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("restored %d routine(s) from %s (#9.7 status %d)",
				len(res.Routines), name, ir.Status)))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("restore did not complete (#9.7 status %d): %s", ir.Status, ir.Error)))
		}
	}); err != nil {
		return err
	}
	if ir.Error == "already-installed" {
		return clikit.Fail(clikit.ExitRefused, "ALREADY_INSTALLED",
			name+": snapshot install identity already present",
			"uninstall the prior snapshot first, or snapshot with a fresh --name")
	}
	if !ir.Installed {
		return clikit.Fail(clikit.ExitRuntime, "NOT_RESTORED",
			fmt.Sprintf("restore did not reach Install Completed (status %d)", ir.Status), "")
	}
	return nil
}

// joinOr joins items with ", ", returning fallback when the list is empty.
func joinOr(items []string, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	out := items[0]
	for _, s := range items[1:] {
		out += ", " + s
	}
	return out
}
