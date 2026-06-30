---
name: transport-checksum
description: v-pkg recomputes each shipped routine's line-2-blind "B" checksum (SUMB) vs the .KID's stored RTN-node value; ~1.6% of real corpus KIDs are born self-inconsistent, so a mismatch WARNS by default (--verify-checksums to refuse).
metadata:
  type: project
---

**Transport-checksum verify (verifiable-safety #3b, 2026-06-30)** — the KIDS
convention's "Verify Checksums in Transport Global". `kids.RoutineChecksumB` ports
VistA's **new, line-2-blind "B" checksum** (`$$SUMB^XPDRSUM` ≡ `^%ZOSF("RSUM1")`:
`Y += $A(char)*(charpos+lineno)` over lines 1,3,4,… with single-`;` comment lines
contributing only their leading tag) and compares it to the value the .KID stores in
its 2-subscript RTN node `0^<numlines>^B<n>`. v-pkg's OWN builds store `0` (real
checksum computed at install, see buildkids.go) → skipped; only ingested/foreign
.KIDs carry a real `B<n>`. Pairs with [[verify-drift]], [[sidecar-integrity]].

**KEY DURABLE FINDING — warn, don't refuse.** The port is engine-correct (the live
`$$SUMB^XPDRSUM` agrees byte-for-byte over the transport source). But a full-corpus
sweep (TestChecksumCorpus, gated on VPKG_KIDS_CORPUS) shows **~1.6% of real national
patches (≈130 of 2056 checksummed files) are "born self-inconsistent"**: their stored
checksum was computed on a slightly different routine version than what shipped —
confirmed by setting the transport source into a global and calling the engine's own
`$$SUMB^XPDRSUM`, which matches this port, NOT the stored value. An offline
self-consistency check (source + checksum both in the same file, over a
NON-cryptographic sum) **cannot distinguish a born-inconsistent KID from a tamper**,
and a real tamperer who rewrites the forgeable checksum defeats it either way (true
tamper-resistance = signed attestation, #4). So:
- **Default = WARN + proceed** (mismatches ride on the install report as
  `checksumWarnings` and print; never false-fail a legitimate install).
- **`--verify-checksums` = HARD refuse** (exit 4 CHECKSUM_MISMATCH, pre-connect, no
  engine write) for operators who want it.
- Owner decided warn-by-default 2026-06-30 (the kickoff's refuse-by-default would
  block ~6% of foreign-patch *files*).

**Why/how to apply:** the gate is OFFLINE (`pkgcli/checksum.go` checksumScan/
checksumRefusal, run before `c.client()`); `--dry-run` skips it. The corpus guard
asserts the mismatch RATE stays < 5% — a SUMB port regression would flip thousands
OK→mismatch and trip it. Live-proven on vehu (greenfield foreign-style ZZT):
--verify-checksums refuses a byte-tampered .KID with no engine write; pristine passes
+ installs; warn mode installs but surfaces the mismatch. Ground-truth fixture:
the 5-line ZZT routine → SUMB B10838. Gate: `make stress` pristine-passes +
tampered-refused probes.
