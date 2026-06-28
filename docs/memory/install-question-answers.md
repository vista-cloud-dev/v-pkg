---
name: install-question-answers
description: A.1.3 — v pkg install --answer NAME=VALUE pre-seeds #9.7 QUES so pre/post routines read it via $$ANSWER^XPDIQ; live-proven both engines
metadata:
  type: project
---

**2026-06-28: A.1.3 — `v pkg install --answer NAME=VALUE` pre-answers a build's
install questions.** The direct-populate install path skips the interactive
question phase (`EN^XPDIQ`, via `EN^XPDI`), so a pre/post-install routine that calls
`$$ANSWER^XPDIQ(name)` would get `""` (= KIDS default NO) for every question.
A.1.3 seeds the answers so the routine reads the intended value — completing the
install-fidelity work alongside [[install-fidelity-spike]] (A.1.1 pre/post, A.1.2
env-check) and [[install-hooks-authoring]] (B.3 authoring).

**Read contract (live-ground-truthed from `ANSWER^XPDIQ`):**
```
ANSWER(QUES) N IEN I '$D(XPDA)!($G(QUES)="") Q ""
 S IEN=$O(^XPD(9.7,XPDA,"QUES","B",QUES,0)) I IEN'>0 Q ""
 Q $G(^XPD(9.7,XPDA,"QUES",IEN,1))
```
So the seed is exactly three nodes per answer (IEN = 1..N):
```
^XPD(9.7,XPDA,"QUES",IEN,0)        = NAME      ; .01
^XPD(9.7,XPDA,"QUES",IEN,1)        = VALUE     ; what $$ANSWER returns (internal)
^XPD(9.7,XPDA,"QUES","B",NAME,IEN) = ""        ; name→IEN lookup xref
```
**The `"QUES"` subtree is internal install scratch — NOT a #9.7 FileMan field**
(no `*QUEST*` field exists in the #9.7 DD), so there is no multiple-header node to
emit; those three nodes are the whole faithful record. Verified live: seed them,
`$$ANSWER^XPDIQ("MYQ")` returns the value; an unseeded name returns `""`.

**Implementation:** `installspec.QuesAnswer{Name,Value}`; `FinalInstallScript` gained
a `ques []QuesAnswer` param and emits the seed AFTER the env-check (only a build
that passes is worth answering) and BEFORE `EN^XPDIJ` runs the pre/post routines.
`v pkg install` gained a repeatable `--answer NAME=VALUE` flag (`parseAnswers`
splits on the FIRST `=`, so a value may contain `=`; order preserved → deterministic
IENs). Adding the flag required `make contract` (golden surface drift).

**Live-proven on BOTH engines** (vehu YDB + foia-t12 IRIS), fixture
`testdata/zza4-ques/` (`ZZA4P` post-install does
`S ^ZZA4OUT("Q")=$$ANSWER^XPDIQ("ZZA4Q")`): `--answer ZZA4Q=HELLO` →
`^ZZA4OUT("Q")="HELLO"`; no `--answer` → `""` (default). Clean counterfactual — the
answer appears only when seeded. XPDA is in scope during `POST^XPDIJ1`, so
`$$ANSWER^XPDIQ` resolves at post-install time.

See [[install-fidelity-spike]], [[install-hooks-authoring]], [[kids-coverage-analysis]].
