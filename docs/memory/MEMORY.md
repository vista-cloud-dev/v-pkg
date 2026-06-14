# v-pkg — per-repo memory index

One line per memory file. Content lives in the files, not here.

- [streamed-install](streamed-install.md) — `v pkg install` now streams the transport global in size-bounded chunks → staging global `^XTMP("VPKGI")` → MERGE + `EN^XPDIJ` in one process, fixing a silent partial-install of large packages (the one-mega-routine staging truncated at ~3 routines). YDB live-proven on the full 15-routine MSL base (test-in-place 15/15 suites). IRIS validation of the chunked path owed.
