---
name: routine-name-length-policy
description: v-pkg build spec gained allowLongNames — relaxes the routine-name cap from the legacy 8 to the M engine limit (31), keeping M-naming validation. Implements the org ADR.
metadata:
  type: project
---

`v-pkg build` enforces routine-name length. As of 2026-06-20 it implements the org ADR
`docs/background/routine-name-length-policy-adr.md` (modern routine-name length policy):

- **`Spec.AllowLongNames`** (`internal/buildspec/buildspec.go`, JSON `allowLongNames` in
  `kids/<pkg>.build.json`). Absent/false → legacy cap **`RoutineNameMaxStd = 8`**; true →
  **`RoutineNameMaxLong = 31`** (the M engine / YDB+IRIS significant-name limit).
- `isRoutineName(s, maxLen)` was parameterized (was hard-coded ≤8); the M-naming regex
  (`^%?[A-Z][A-Z0-9]*$`, no spaces) and the 31 ceiling still bind in both modes. `validateRoutines`
  and the `envCheck` check both honor the active cap.
- The `.kids` emit side (`internal/kids/`) never had a length limit, so no change there — the 8
  was purely this up-front validator.

Tests in `internal/buildspec/buildspec_test.go`: default-reject (9-char w/o flag), long-admit
(`STDASSERT`/`STDCOMPRESS`/`STDHTTPMSG` + long envCheck w/ flag), still-gated (>31 over ceiling,
lowercase, spaces). buildspec cov 98.7%. **MERGED to `main` 2026-06-20** (`bdd707b`).

Repo gotcha (now FIXED): `make test` uses `-race` (needs cgo) but the Makefile globally exports
`CGO_ENABLED := 0` for static builds — collided so the test gate failed unless run as
`make test CGO_ENABLED=1`. Fixed by setting `CGO_ENABLED=1` on the `test` recipe line only
(build/dist stay static); `make test` now passes with no override. Merged to `main` (same push).

Shared coordination note: [[routine-name-length-policy]] in the `docs` repo.
