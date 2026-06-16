# v-pkg — per-repo memory index

One line per memory file. Content lives in the files, not here.

- [streamed-install](streamed-install.md) — `v pkg install` now streams the transport global in size-bounded chunks → staging global `^XTMP("VPKGI")` → MERGE + `EN^XPDIJ` in one process, fixing a silent partial-install of large packages (the one-mega-routine staging truncated at ~3 routines). YDB live-proven on the full 15-routine MSL base (test-in-place 15/15 suites). IRIS validation of the chunked path owed.
- [krn-param-def-component](krn-param-def-component.md) — `v pkg` now builds + installs/verifies/uninstalls a **#8989.51 PARAMETER DEFINITION as a KIDS KRN component** (+ Required Builds #9.611), the VSL/MSL **T1.3 enabler**. KRN transport = BLD manifest + `"ORD"` (xref=1) + top-level `"KRN",8989.51,seq` record image MERGEd into `^XTV` by `KRN^XPDIK`. **KEY FIX**: the direct-populate install must seed `^XPD(9.7,XPDA,"KRN")` from the build manifest or `KRN^XPDIK` GVUNDEFs → status stuck at 2. SYS entity = #8989.518 IEN **4.2** (national). **Proven install→verify→uninstall→clean on BOTH engines** (vehu YDB + foia-t12 IRIS, `testdata/zzparam`).
