// Package vcontract is the v CLI domain command-surface contract
// (v-cli-platform.md §4): the generated, drift-gated manifest a domain emits to
// dist/v-contract.json, which the `v` umbrella aggregates into its registry
// (§5). It is built by reflecting the domain's kong command tree (via
// clikit.BuildSchema), so the manifest can never drift from the real surface.
package vcontract

import (
	"github.com/alecthomas/kong"

	"github.com/vista-cloud-dev/clikit"
)

// Manifest is one domain's contract (§4): its name, the tool SemVer, a
// contractVersion that bumps only on an incompatible command-surface change
// (independent of SemVer), the exit-code ladder the surface uses, and the full
// reflected command tree.
type Manifest struct {
	Domain          string                 `json:"domain"`
	Version         string                 `json:"version"`
	ContractVersion string                 `json:"contractVersion"`
	Exits           []ExitCode             `json:"exits"`
	Commands        []clikit.SchemaCommand `json:"commands"`
}

// ExitCode is one rung of the contract exit-code ladder.
type ExitCode struct {
	Code    int    `json:"code"`
	Meaning string `json:"meaning"`
}

// Ladder is the clikit exit-code ladder a v domain may return (driver-contract
// §2 / clikit). Carried in the contract so consumers know the codes to branch on.
func Ladder() []ExitCode {
	return []ExitCode{
		{clikit.ExitOK, "ok"},
		{clikit.ExitRuntime, "runtime error"},
		{clikit.ExitUsage, "usage error"},
		{clikit.ExitCheck, "check / gate failed"},
		{clikit.ExitRefused, "refused / substrate unavailable"},
	}
}

// Build reflects a parsed kong model into a domain Manifest. domain is the
// plain-language domain noun (e.g. "pkg"); version is the tool SemVer;
// contractVersion bumps on an incompatible surface change.
func Build(domain, version, contractVersion string, k *kong.Kong) Manifest {
	doc := clikit.BuildSchema(k, domain, version)
	return Manifest{
		Domain:          domain,
		Version:         version,
		ContractVersion: contractVersion,
		Exits:           Ladder(),
		Commands:        doc.Commands,
	}
}
