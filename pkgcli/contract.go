package pkgcli

import (
	"github.com/alecthomas/kong"

	"github.com/vista-cloud-dev/clikit"
	"github.com/vista-cloud-dev/v-pkg/vcontract"
)

const (
	// Version is the declared SemVer of the v-pkg domain surface. It is a
	// committed constant (distinct from the link-time build version reported by
	// `version`) so the generated contract is drift-stable.
	Version = "0.4.0"

	// ContractVersion bumps only on an incompatible command-surface change
	// (v-cli-platform.md §4), independent of Version.
	ContractVersion = "1.0"
)

// Contract returns the v-pkg domain contract manifest, reflected from the actual
// pkgcli command tree. It is what the standalone `v-pkg` writes to
// dist/v-contract.json and what the `v` umbrella aggregates into its registry —
// one source, so the manifest can never drift from the real verbs.
func Contract() vcontract.Manifest {
	var grammar struct {
		clikit.Globals
		Commands
	}
	k, err := kong.New(&grammar)
	if err != nil {
		// The grammar is static; a failure here is a programming error.
		panic("pkgcli: build contract grammar: " + err.Error())
	}
	return vcontract.Build("pkg", Version, ContractVersion, k)
}
