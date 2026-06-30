package pkgcli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
	"github.com/vista-cloud-dev/v-pkg/internal/installspec"
)

// heal repairs the half-install corruption documented in
// kids-installation-automation.md §7.1: a prior aborted install can leave a #9.7
// entry with the "B" xref (and "ASP"/"INI"/"INIT" subnodes) but no usable 0-node /
// a status that never reached 3. EN^XPDIJ silently bails on it, yet the re-install
// guard (FinalInstallScript: `I $D(^XPD(9.7,"B",name))`) FALSELY reports
// already-installed — so a clean reinstall is impossible until the corpse is purged.
//
// `install --heal` purges that PROVEN-corrupt entry (and only that), then installs
// normally. It NEVER touches a healthy install — that is uninstall's job. The state
// is graded from a read-only probe (classifyHeal) before any purge, and the purge
// script re-confirms corruption engine-side as defense in depth.

// healState grades a #9.7 INSTALL entry from the read-only heal-detect probe.
type healState int

const (
	// healNotInstalled: no #9.7 entry — nothing to heal, install proceeds normally.
	healNotInstalled healState = iota
	// healHealthy: a complete install (0-node present AND status 3) — must NOT be
	// healed (heal is for corruption; removing a healthy install is uninstall's job).
	healHealthy
	// healCorrupt: an entry exists ("B" xref present) but is unusable — no 0-node, or
	// a status that never reached 3 (Install Completed). The purge target.
	healCorrupt
)

func (s healState) String() string {
	switch s {
	case healHealthy:
		return "healthy"
	case healCorrupt:
		return "corrupt"
	default:
		return "not-installed"
	}
}

// classifyHeal grades the entry from the probe markers. A healthy install (0-node
// present AND status 3) is graded healthy; an entry with a "B" xref (ien>0) but no
// 0-node, or any status other than 3, is corrupt; no entry (ien==0) is not-installed.
func classifyHeal(ien int, zeroPresent bool, status int) healState {
	if ien == 0 {
		return healNotInstalled
	}
	if zeroPresent && status == 3 {
		return healHealthy
	}
	return healCorrupt
}

// healAction is the strategy `install --heal` takes for a graded entry.
type healAction int

const (
	// healProceed: no prior entry — nothing to heal; install normally.
	healProceed healAction = iota
	// healPurgeProceed: a corrupt entry blocks reinstall — purge it, then install.
	healPurgeProceed
	// healRefuse: the install is healthy — refuse (nothing to heal; uninstall first).
	healRefuse
)

func (a healAction) String() string {
	switch a {
	case healPurgeProceed:
		return "purge+proceed"
	case healRefuse:
		return "refuse"
	default:
		return "proceed"
	}
}

// decideHeal maps a graded state to the heal action plus a human reason.
func decideHeal(state healState) (healAction, string) {
	switch state {
	case healCorrupt:
		return healPurgeProceed, "corrupt half-install detected (#9.7 entry present but no usable 0-node / status not Install Completed) — purging it so a clean reinstall can proceed"
	case healHealthy:
		return healRefuse, "already installed and healthy (#9.7 status 3) — heal only repairs a corrupt half-install; `v pkg uninstall` to remove a healthy install first"
	default: // healNotInstalled
		return healProceed, "no prior #9.7 entry — nothing to heal; installing normally"
	}
}

// healResult records what heal observed and did, surfaced on the install report.
type healResult struct {
	State  string `json:"state"`            // not-installed | healthy | corrupt
	Action string `json:"action"`           // proceed | purge+proceed | refuse
	Purged bool   `json:"purged,omitempty"` // a corrupt entry was actually purged
	Reason string `json:"reason,omitempty"`
}

// probeHeal runs the read-only heal-detect script and grades the entry.
func probeHeal(ctx context.Context, cl *mdriver.Client, name string) (healState, error) {
	markers, _, err := runMScript(ctx, cl, rtnHeal, installspec.HealDetectScript(name))
	if err != nil {
		return healNotInstalled, err
	}
	ien, _ := strconv.Atoi(strings.TrimSpace(markers["ien"]))
	zero := strings.TrimSpace(markers["zero"]) == "1"
	status, _ := strconv.Atoi(strings.TrimSpace(markers["status"]))
	return classifyHeal(ien, zero, status), nil
}

// purgeHeal runs the guarded purge script and reports whether the corrupt entry was
// removed. A healthy-refused marker (the engine-side defense-in-depth guard fired)
// is surfaced as an error so the caller never proceeds to clobber a healthy install.
func purgeHeal(ctx context.Context, cl *mdriver.Client, name string) (bool, error) {
	markers, _, err := runMScript(ctx, cl, rtnHeal, installspec.HealPurgeScript(name))
	if err != nil {
		return false, err
	}
	if markers["error"] == "healthy-refused" {
		return false, fmt.Errorf("heal refused: %s is a healthy install (#9.7 status 3), not a corrupt half-install", name)
	}
	return strings.TrimSpace(markers["healed"]) == "1", nil
}
