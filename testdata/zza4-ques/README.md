# zza4-ques — A.1.3 install-question (QUES) answer fixture

Throwaway package that proves **A.1.3**: `v pkg install --answer NAME=VALUE`
pre-answers a build's install questions so a pre/post-install routine reads them
via the real `$$ANSWER^XPDIQ` — the non-interactive equivalent of the KIDS
question phase (`EN^XPDIQ`) that the direct-populate path skips.

## Build

- **`kids/ZZA4.build.json`** — ships `ZZA4P`; declares `postInstall:"POST^ZZA4P"`
  (B.3 authoring). The post-install hook records the answer to question `ZZA4Q`.

## Routine (`src/ZZA4P.m`)

- `POST` sets `^ZZA4OUT("Q")=$$ANSWER^XPDIQ("ZZA4Q")` — the seeded internal answer,
  or `""` when no answer was seeded (the KIDS default = NO).

## How A.1.3 seeds the answer

`FinalInstallScript`, after `$$INST^XPDIL1` assigns `XPDA` and before `EN^XPDIJ`
runs the post-install routine, sets the three internal nodes `$$ANSWER^XPDIQ`
reads (live-ground-truthed; the `"QUES"` subtree is install scratch, not a #9.7
FileMan field):

```
^XPD(9.7,XPDA,"QUES",IEN,0)      = NAME
^XPD(9.7,XPDA,"QUES",IEN,1)      = VALUE   ; $$ANSWER returns this
^XPD(9.7,XPDA,"QUES","B",NAME,IEN) = ""    ; the name→IEN lookup xref
```

## Live-proven (2026-06-28) — both engines, via the driver stack

| Engine | `--answer ZZA4Q=HELLO` | control (no `--answer`) |
|---|---|---|
| YDB (vehu)      | `^ZZA4OUT("Q")="HELLO"`, #9.7 status 3 | `^ZZA4OUT("Q")=""` (default) |
| IRIS (foia-t12) | `^ZZA4OUT("Q")="HELLO"`, #9.7 status 3 | `^ZZA4OUT("Q")=""` (default) |

The counterfactual (answer appears only when seeded) proves the seed reached
`$$ANSWER^XPDIQ`, not some pre-existing state.
