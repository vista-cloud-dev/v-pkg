// Command v-pkg is the VistA KIDS package tool — the standalone form of the
// `v pkg` domain. It decomposes monolithic .KID distribution files into a
// per-component tree suitable for git and reassembles that tree back to an
// installable .KID, on the shared clikit conventions (--output text|json,
// schema, deterministic errors, the exit-code ladder). The verb set lives in
// the importable pkgcli package so the `v` umbrella mounts the same commands as
// `v pkg <verb>` (static-pinned composition).
//
// The round-trip guarantee is semantic equality after routine line-2
// canonicalization — NOT byte-identity — exactly as py-kids-vc / XPDK2VC define
// it (volatile patch-list/date/build-number pieces are stripped).
//
// Try:
//
//	v-pkg parse OR_3.0_484.KID
//	v-pkg decompose OR_3.0_484.KID ./patches/
//	v-pkg assemble ./patches/ rebuilt.KID
//	v-pkg roundtrip OR_3.0_484.KID        # exit 3 on drift
//	v-pkg canonicalize ./patches/         # LOSSY IEN substitution
//	v-pkg lint OR_3.0_484.KID             # PIKS data-class gate (K2)
//	v-pkg schema | jq .
package main

import (
	"os"

	"github.com/willabides/kongplete"

	"github.com/vista-cloud-dev/v-pkg/clikit"
	"github.com/vista-cloud-dev/v-pkg/pkgcli"
)

// CLI is the standalone v-pkg grammar: the pkgcli verbs at the top level, plus
// the shared clikit meta commands.
type CLI struct {
	clikit.Globals
	pkgcli.Commands

	Schema  clikit.SchemaCmd  `cmd:"" help:"Emit the command/flag/enum tree as JSON (agent discovery)."`
	Version clikit.VersionCmd `cmd:"" help:"Show version and build info."`

	InstallCompletions kongplete.InstallCompletions `cmd:"" help:"Install shell tab-completions."`
}

func main() {
	cli := &CLI{}
	os.Exit(clikit.Run(
		"v-pkg",
		"VistA KIDS package tool — decompose / assemble / roundtrip / canonicalize / lint.",
		cli, &cli.Globals,
	))
}
