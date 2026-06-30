//go:build unix

package attest

import (
	"os"
	"syscall"
)

// WithLedgerLock runs fn while holding an EXCLUSIVE advisory lock on <path>.lock, so a
// read-tip→seal→append sequence is atomic across concurrent v-pkg processes on the
// same host — without it, two ops read the same LastHash, seal with the same prevHash,
// and fork the tamper-evident chain (a false-positive "tampering" on the next verify).
// flock is advisory and per-open-file-description, so it also serializes goroutines in
// one process. The .lock file persists (empty, harmless).
func WithLedgerLock(path string, fn func() error) error {
	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()
	return fn()
}
