---
name: half-install-heal
description: v-pkg `install --heal` purges a PROVEN-corrupt #9.7 half-install (B xref present, no usable 0-node / status≠3) by IEN so a clean reinstall proceeds; never touches a healthy install.
metadata:
  type: project
---

**`v pkg install --heal` (verifiable-safety #3a, 2026-06-30)** — closes the gap that a
prior aborted install leaves a `#9.7` entry with the `"B"` xref (and `"ASP"`/`"INI"`/
`"INIT"` subnodes) but **no usable 0-node** / a status that never reached 3. `EN^XPDIJ`
silently bails on it (`Q:'$D(^XPD(9.7,+$G(XPDA),0))`), yet the re-install guard
(`FinalInstallScript`: `I $D(^XPD(9.7,"B",name))`) FALSELY reports already-installed —
so a clean reinstall is impossible until the corpse is purged. Canon: the §7.1 gotcha
in `../design/kids-installation-automation.md`. Pairs with [[class-aware-install]],
[[streamed-install]].

**Model (pkgcli/heal.go + installspec HealDetectScript/HealPurgeScript):**
- `classifyHeal(ien, zeroPresent, status)` over a READ-ONLY probe → `healNotInstalled`
  (ien=0) / `healHealthy` (0-node present AND status 3) / `healCorrupt` (ien>0 but
  0-node absent OR status≠3).
- `decideHeal`: corrupt → purge then install; **healthy → REFUSE** (exit 4, "uninstall
  to remove a healthy install"); not-installed → proceed normally.
- Purge is by IEN — `K ^XPD(9.7,ien)` (entry subtree incl ASP/INI/INIT subnodes),
  `^XPD(9.7,"B",name,ien)`, `^XPD(9.7,"ASP",ien)`, `^XTMP("XPDI",ien)`. **DIK can't
  clean an entry whose 0-node is gone** (it reads the .01 from the 0-node to find the
  xref), which is why §7.1 mandates the manual global KILLs, not `^DIK`.

**Why/how to apply:** when a real install reports `already-installed` but the package
isn't actually there, the #9.7 is a corrupt half-install — `v pkg install <kid> --heal`.
Heal is a TARGETED purge of a proven-corrupt entry only: defense in depth, the purge
script RE-CONFIRMS corruption engine-side and emits `error=healthy-refused` if it finds
a status-3 entry, so a healthy install can never be purged even on a Go-side misgrade.
Live-proven on vehu (seed B xref + ASP, no 0-node → normal install falsely refuses →
--heal purges + reinstalls to status 3, one entry, stale ^XTMP gone; --heal on a
healthy ZZSKEL refused, untouched). Gate: `make stress` heal-refuses-healthy probe.
#9.7 alpha xrefs on vehu are `AKIDS`/`ASP`/`B`/`C` (AKIDS/C derive from the 0-node, so
a 0-node-less corpse never has them — B+ASP+subtree+XTMP is the complete purge).
