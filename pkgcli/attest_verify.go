package pkgcli

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/vista-cloud-dev/clikit"
	mdriver "github.com/vista-cloud-dev/m-driver-sdk"
	"github.com/vista-cloud-dev/v-pkg/internal/attest"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// `v pkg attest` is the read-only audit surface over the attestation ledger that
// install / uninstall / restore write (#4). It mutates nothing — it validates the
// tamper-evidence (hash chain), optionally the tamper-resistance (pinned ed25519
// signatures), and optionally REPLAYS each record against the live engine to confirm
// the recorded AFTER checksums still hold.
type attestCmd struct {
	Verify attestVerifyCmd `cmd:"" help:"Validate an attestation ledger's hash chain (+ signatures), optionally replaying it against the live engine."`
}

type attestVerifyCmd struct {
	Ledger    string `arg:"" help:"Path to the attestation ledger (JSON Lines) to validate."`
	Trust     string `help:"Pin every record's signature to this hex ed25519 public key (default $VPKG_ATTEST_PUBKEY). Fails if any record is unsigned or signed by a different key." placeholder:"HEXKEY"`
	Replay    bool   `help:"Also read each recorded routine off the live engine and confirm it still matches the record's AFTER checksum (needs --engine)."`
	Engine    string `help:"Engine for --replay: ydb or iris." enum:"ydb,iris," default:""`
	Transport string `help:"Driver transport for --replay." enum:"local,docker,remote" default:"remote"`
}

// replaySummary reports the live-engine replay of a ledger's recorded AFTER state.
type replaySummary struct {
	Checked    int      `json:"checked"`              // routine assertions replayed
	Mismatches []string `json:"mismatches,omitempty"` // "<record#>:<routine> recorded <x>, live <y>"
}

type attestVerifyReport struct {
	Ledger   string         `json:"ledger"`
	Records  int            `json:"records"`
	ChainOK  bool           `json:"chainOk"`
	Trusted  bool           `json:"trusted,omitempty"` // pinned-key signature check passed
	Replayed *replaySummary `json:"replay,omitempty"`
	Problem  string         `json:"problem,omitempty"`
}

// trustKey resolves the pinned public key: --trust wins, else $VPKG_ATTEST_PUBKEY,
// else none (nil). A malformed key is a loud usage error.
func (c *attestVerifyCmd) trustKey() (ed25519.PublicKey, error) {
	raw := strings.TrimSpace(c.Trust)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(verifyKeyEnv))
	}
	if raw == "" {
		return nil, nil
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("trust key is not valid hex: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("trust key must be a %d-byte hex ed25519 public key, got %d bytes", ed25519.PublicKeySize, len(b))
	}
	return ed25519.PublicKey(b), nil
}

func (c *attestVerifyCmd) Run(cc *clikit.Context) error {
	records, err := attest.Load(c.Ledger)
	if err != nil {
		return clikit.Fail(clikit.ExitRuntime, "LEDGER_READ_FAILED", err.Error(), "")
	}
	rep := attestVerifyReport{Ledger: c.Ledger, Records: len(records)}

	if _, cerr := attest.VerifyChain(records); cerr != nil {
		rep.ChainOK = false
		rep.Problem = cerr.Error()
		return c.fail(cc, rep)
	}
	rep.ChainOK = true

	trust, terr := c.trustKey()
	if terr != nil {
		return clikit.Fail(clikit.ExitUsage, "BAD_TRUST_KEY", terr.Error(), "")
	}
	if trust != nil {
		if verr := attest.VerifyChainTrusted(records, trust); verr != nil {
			rep.Problem = verr.Error()
			return c.fail(cc, rep)
		}
		rep.Trusted = true
	}

	if c.Replay {
		if c.Engine == "" {
			return clikit.Fail(clikit.ExitUsage, "REPLAY_NO_ENGINE", "--replay needs --engine ydb|iris", "")
		}
		summary, rerr := c.replay(cc, records)
		if rerr != nil {
			return rerr
		}
		rep.Replayed = summary
		if len(summary.Mismatches) > 0 {
			rep.Problem = fmt.Sprintf("%d replay mismatch(es): the live engine no longer matches what the ledger recorded", len(summary.Mismatches))
			return c.fail(cc, rep)
		}
	}

	return cc.Result(rep, func() { c.render(cc, rep) })
}

// replay confirms the live engine still matches the ledger's NET recorded state. A
// routine touched by several ops (install adds it → uninstall removes it) has its
// expected state defined by the LAST record that recorded an AFTER for it — so a
// full-lifecycle ledger replays clean against the resulting engine, while a record
// whose effect was NOT later superseded must still hold. This makes replay the honest
// "does the engine match what the ledger says it should be now" check.
func (c *attestVerifyCmd) replay(_ *clikit.Context, records []attest.Record) (*replaySummary, error) {
	bin, err := mdriver.Locate(c.Engine, mdriver.DefaultLocateDeps())
	if err != nil {
		return nil, clikit.Fail(clikit.ExitRefused, "NO_DRIVER", err.Error(),
			"build the m-"+c.Engine+" driver (make build) or set M_"+strings.ToUpper(c.Engine)+"_BIN")
	}
	cl := mdriver.NewClient(bin, c.Engine, c.Transport, nil, nil)
	ctx := context.Background()

	// Net expected state per routine: the AFTER from the highest-indexed record that
	// recorded one (later ops supersede earlier ones).
	type expect struct {
		idx  int
		want string
	}
	net := map[string]expect{}
	for i, r := range records {
		for name, want := range r.After {
			net[name] = expect{idx: i, want: want}
		}
	}

	var sum replaySummary
	names := make([]string, 0, len(net))
	for name := range net {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		e := net[name]
		sum.Checked++
		live, present, rerr := readRoutinePreimage(ctx, cl, name)
		if rerr != nil {
			return nil, clikit.Fail(clikit.ExitRuntime, "REPLAY_READ_FAILED", rerr.Error(), "")
		}
		switch {
		case e.want == "absent":
			if present {
				sum.Mismatches = append(sum.Mismatches, fmt.Sprintf("%d:%s recorded absent, live present", e.idx, name))
			}
		case !present:
			sum.Mismatches = append(sum.Mismatches, fmt.Sprintf("%d:%s recorded %s, live absent", e.idx, name, e.want))
		default:
			if got := kids.BChecksum(live); got != e.want {
				sum.Mismatches = append(sum.Mismatches, fmt.Sprintf("%d:%s recorded %s, live %s", e.idx, name, e.want, got))
			}
		}
	}
	return &sum, nil
}

func (c *attestVerifyCmd) fail(cc *clikit.Context, rep attestVerifyReport) error {
	if err := cc.Result(rep, func() { c.render(cc, rep) }); err != nil {
		return err
	}
	return clikit.Fail(clikit.ExitCheck, "ATTEST_VERIFY_FAILED", rep.Problem,
		"the ledger was altered or the live engine drifted from what it recorded")
}

func (c *attestVerifyCmd) render(cc *clikit.Context, rep attestVerifyReport) {
	cc.Title("pkg attest verify")
	cc.KV([2]string{"ledger", rep.Ledger}, [2]string{"records", fmt.Sprint(rep.Records)})
	if rep.ChainOK {
		fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("hash chain intact (%d record(s))", rep.Records)))
	} else {
		fmt.Fprintln(cc.Stdout, cc.Failure("hash chain BROKEN: "+rep.Problem))
		return
	}
	if rep.Trusted {
		fmt.Fprintln(cc.Stdout, cc.Success("signatures verified against the pinned key"))
	}
	if rep.Replayed != nil {
		if len(rep.Replayed.Mismatches) == 0 {
			fmt.Fprintln(cc.Stdout, cc.Success(fmt.Sprintf("replay clean: %d routine assertion(s) still match the live engine", rep.Replayed.Checked)))
		} else {
			fmt.Fprintln(cc.Stdout, cc.Failure(fmt.Sprintf("replay FAILED: %d of %d assertion(s) drifted", len(rep.Replayed.Mismatches), rep.Replayed.Checked)))
			for _, m := range rep.Replayed.Mismatches {
				fmt.Fprintf(cc.Stdout, "  %s %s\n", cc.Failure("drift"), m)
			}
		}
	}
}
