//go:build !unix

package attest

// WithLedgerLock is a no-op on non-unix platforms (the engine host is unix; the
// cross-process ledger lock uses flock there). The chain hash still detects any
// resulting fork on verify.
func WithLedgerLock(_ string, fn func() error) error { return fn() }
