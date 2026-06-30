package pkgcli

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vista-cloud-dev/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/attest"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// Install attestation (verifiable-safety #4). Each engine-MUTATING op (install /
// uninstall / restore — never a read-only diff/verify) appends a tamper-EVIDENT
// record to a HOST-SIDE append-only ledger, so "it installed" becomes provable
// provenance an independent auditor can replay. The record core (hash chain +
// optional ed25519 signature) is the engine-neutral internal/attest package; this
// file is the pkgcli glue: it assembles a record from the op's report + the live
// before/after checksums, resolves the ledger path, loads the signing key, and
// appends. Records are emitted at the command chokepoint each op already funnels
// through (installCmd.Run / runMulti, uninstallCmd.Run, restoreCmd.Run).
//
// Policy (kickoff design calls):
//   - WHERE: a host-side JSON Lines ledger, default <kid>.attest.jsonl next to the
//     .KID, overridable with --attest <path>. Host-side keeps it outside the engine
//     the op mutates and offline-auditable.
//   - PROTECTION: a sha256 prevHash chain ALWAYS (tamper-evident); an OPT-IN
//     detached ed25519 signature (--sign, key from VPKG_ATTEST_KEY) is the real
//     tamper-resistance.
//   - WHEN: on by default for mutating ops (--no-attest suppresses); a record is
//     written only when the op actually MUTATED the engine (a refused / no-op /
//     failed-pre-write op writes nothing — there is no provenance to record).

// signKeyEnv is the env var holding the hex-encoded ed25519 SEED (32 bytes → 64 hex
// chars) used to sign attestation records under --sign. A seed, not a full private
// key, keeps the secret minimal; the public key is derived and recorded per record.
const signKeyEnv = "VPKG_ATTEST_KEY"

// verifyKeyEnv is the env var holding the hex-encoded ed25519 PUBLIC key (32 bytes)
// that `attest verify --trust` pins the ledger's signatures against when no explicit
// --pubkey is given.
const verifyKeyEnv = "VPKG_ATTEST_PUBKEY"

// attestFlags are the attestation knobs shared by install / uninstall / restore.
// They embed (anonymous) into each command so kong flattens --attest / --no-attest
// / --sign onto the verb.
type attestFlags struct {
	Attest   string `help:"Append the audit record to this ledger path (default: <kid>.attest.jsonl next to the .KID)." placeholder:"PATH"`
	NoAttest bool   `help:"Do NOT write an attestation audit record for this op (default: record every engine-mutating op)."`
	Sign     bool   `help:"Sign each record with a detached ed25519 signature (key from $VPKG_ATTEST_KEY, a hex 32-byte seed) — tamper-RESISTANT, not just tamper-evident."`
}

// defaultAttestPath is the conventional ledger next to a .KID: its path with the
// extension replaced by ".attest.jsonl".
func defaultAttestPath(kidFile string) string {
	ext := filepath.Ext(kidFile)
	return strings.TrimSuffix(kidFile, ext) + ".attest.jsonl"
}

// ledgerPath resolves the effective ledger: --attest wins, else the default sidecar.
func (f attestFlags) ledgerPath(kidFile string) string {
	if f.Attest != "" {
		return f.Attest
	}
	return defaultAttestPath(kidFile)
}

// loadSigningKey reads the ed25519 signing seed from $VPKG_ATTEST_KEY (hex, 32
// bytes) and returns the derived private key. An absent/short/malformed value is a
// loud error so --sign never silently produces an unsigned record.
func loadSigningKey() (ed25519.PrivateKey, error) {
	raw := strings.TrimSpace(os.Getenv(signKeyEnv))
	if raw == "" {
		return nil, fmt.Errorf("--sign requires a signing key in $%s (a hex-encoded ed25519 32-byte seed)", signKeyEnv)
	}
	seed, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("$%s is not valid hex: %w", signKeyEnv, err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("$%s must be a %d-byte hex seed, got %d bytes", signKeyEnv, ed25519.SeedSize, len(seed))
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// attestInput is the flattened change-set an op hands to newRecord. The interesting
// derivations (routine before/after checksums, component keys) are separate helpers;
// this is just the shape that becomes a Record.
type attestInput struct {
	Op, Action, Name, Class, Engine, Transport string
	Before, After                              map[string]string
	Components, RequiredBuilds                 []string
	Snapshot, SnapshotHash                     string
	Status                                     int
	Verify                                     string
	Exit                                       int
}

// newRecord maps an attestInput onto an unsealed attest.Record (chain/hash/sig are
// added by emitAttestation → attest.Seal). Pure — no engine, no clock.
func newRecord(in attestInput) attest.Record {
	return attest.Record{
		Schema: attest.Schema, Op: in.Op, Action: in.Action, Name: in.Name, Class: in.Class,
		Engine: in.Engine, Transport: in.Transport,
		Before: in.Before, After: in.After,
		Components: in.Components, RequiredBuilds: in.RequiredBuilds,
		Snapshot: in.Snapshot, SnapshotHash: in.SnapshotHash,
		Status: in.Status, Verify: in.Verify, Exit: in.Exit,
	}
}

// routineChecksumMaps builds the record's before/after routine-checksum maps from
// the install probe results. BEFORE is the live pre-image read off the engine
// (captured), or "absent" for a greenfield-added routine; AFTER is the checksum of
// the source this op FILED (the build's shipped routine source) — what an auditor's
// replay must still find on the live engine.
func routineChecksumMaps(b *kids.Build, captured []kids.RoutineSrc, greenfield []string) (before, after map[string]string) {
	before = map[string]string{}
	after = map[string]string{}
	for _, rs := range captured {
		before[rs.Name] = kids.BChecksum(rs.Lines)
	}
	for _, name := range greenfield {
		before[name] = "absent"
	}
	for _, name := range b.RoutineNames() {
		after[name] = kids.BChecksum(b.RoutineSource(name))
	}
	return before, after
}

// installAttestInput assembles the install op's attestInput from its report + build
// (the before/after checksum maps were computed during liveInstall and ride on the
// report). A completed install has exit 0.
func installAttestInput(res installReport, b *kids.Build, engine, transport string) attestInput {
	return attestInput{
		Op: "install", Action: res.Action, Name: res.Name, Class: res.Class,
		Engine: engine, Transport: transport,
		Before: res.Before, After: res.After,
		Components: componentKeys(b.Components()), RequiredBuilds: b.RequiredBuildNames(),
		Snapshot: res.Snapshot, SnapshotHash: res.SnapshotHash,
		Status: res.Status, Exit: 0,
	}
}

// uninstallVerifyVerdict folds an uninstall report's verify fields into the record's
// single Verify string: the verify-clean verdict, qualified by the stricter
// foreign-restore grade when one was computed. "" when --verify was not requested.
func uninstallVerifyVerdict(res uninstallReport) string {
	switch {
	case res.VerifyClean == "" && res.ForeignRestore == "":
		return ""
	case res.ForeignRestore != "":
		return strings.TrimSpace(res.VerifyClean + " (foreign:" + res.ForeignRestore + ")")
	default:
		return res.VerifyClean
	}
}

// installTimeOverwrites returns the routines this build OVERWROTE at install time —
// routines that pre-existed on the engine, read from the attestation ledger's latest
// install record (a Before entry that is neither "" nor "absent"). This is durable
// install-time truth uninstall folds into the foreign set, so a national routine an
// install clobbered (via --allow-overwrite) is protected from a delete-on-uninstall
// brick even when the build did NOT self-declare it foreign. Empty when there is no
// ledger or no matching install record — graceful degradation to the prior behavior.
func installTimeOverwrites(flags attestFlags, kidFile, name string) []string {
	recs, err := attest.Load(flags.ledgerPath(kidFile))
	if err != nil || len(recs) == 0 {
		return nil
	}
	var before map[string]string
	for i := range recs {
		if recs[i].Op == "install" && recs[i].Name == name && recs[i].Before != nil {
			before = recs[i].Before // the latest install record for this build wins
		}
	}
	var out []string
	for rt, b := range before {
		if b != "" && b != "absent" {
			out = append(out, rt)
		}
	}
	sort.Strings(out)
	return out
}

// mergeForeign unions the build's self-declared foreign routines with the install-time
// overwrite set (deduped, sorted) — the effective set uninstall must never delete/brick.
func mergeForeign(declared, overwrites []string) []string {
	set := make(map[string]bool, len(declared)+len(overwrites))
	for _, r := range declared {
		set[r] = true
	}
	for _, r := range overwrites {
		set[r] = true
	}
	out := make([]string, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// componentKeys renders a build's non-routine components as "<file>:<name>" keys —
// the same identity the verify/dry-run probes use — for the record's component list.
func componentKeys(comps []kids.Component) []string {
	var out []string
	for _, c := range comps {
		for _, n := range c.Names {
			out = append(out, c.FileStr+":"+n)
		}
	}
	return out
}

// emitAttestation seals a record onto the ledger's current tip and appends it,
// honoring --no-attest and --sign. It stamps the wall-clock timestamp here (the
// core stays clock-free for testability). A ledger I/O or signing failure is
// returned as an error: the op already mutated the engine, so a FAILED audit write
// must be surfaced loudly, never swallowed.
func emitAttestation(flags attestFlags, kidFile string, r attest.Record) error {
	if flags.NoAttest {
		return nil
	}
	var priv ed25519.PrivateKey
	if flags.Sign {
		p, err := loadSigningKey()
		if err != nil {
			return err
		}
		priv = p
	}
	path := flags.ledgerPath(kidFile)
	r.Timestamp = time.Now().UTC().Format(time.RFC3339)
	// Read-tip → seal → append must be atomic across concurrent v-pkg processes, or two
	// ops chain onto the same tip and fork the hash chain. Hold the cross-process ledger
	// lock for the whole sequence.
	return attest.WithLedgerLock(path, func() error {
		last, err := attest.LastHash(path)
		if err != nil {
			return fmt.Errorf("read attestation ledger %s: %w", path, err)
		}
		sealed, err := attest.Seal(r, last, priv)
		if err != nil {
			return fmt.Errorf("seal attestation record: %w", err)
		}
		if err := attest.Append(path, sealed); err != nil {
			return fmt.Errorf("append attestation record to %s: %w", path, err)
		}
		return nil
	})
}

// attestEmitError wraps an attestation-emit failure as a clikit error. The engine
// was already mutated, so this is a post-mutation failure: report it but it does not
// undo the op.
func attestEmitError(err error) *clikit.Error {
	return clikit.Fail(clikit.ExitRuntime, "ATTEST_FAILED", err.Error(),
		"the op succeeded but its audit record was NOT written — record provenance manually or re-run with a writable --attest path")
}
