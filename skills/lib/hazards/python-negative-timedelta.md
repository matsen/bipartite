---
tags: [measurement, python]
measured: 2026-09-21
---

# A negative Python `timedelta` prints as `-1 day` plus a large seconds count

**Looks right:** an interval measured as `-1 day, 23:59:38` and read as *"about a day early."*

**What happens:** `timedelta` normalises to a non-negative `seconds` field, so every negative
interval under 24 hours renders with `days=-1`. The magnitude lives in the seconds, subtracted from
a day — and the printed form invites reading the days field as the magnitude.

```
>>> repr(timedelta(seconds=-22))
datetime.timedelta(days=-1, seconds=86378)
>>> str(timedelta(seconds=-22))
'-1 day, 23:59:38'
>>> timedelta(seconds=-22).total_seconds()
-22.0
```

**Measured 2026-09-21.** A conductor comparing `matsengrp/phyz` PR `createdAt` against the earliest
`committedDate` of its own commits read two intervals as *"PRs opened a day before their own first
commit"* and reported the magnitude onward. The real gap on #2886 was **22 seconds**
(`2026-09-20T23:08:02Z` versus `23:08:24Z`) — ordinary ordering around PR creation and push.

**Check:** `.total_seconds()`, always, for anything that can be negative. Never `.days`, and never
the string form.

**The expensive half was not the rendering.** A negative interval needs an explanation, the nearest
available one was *"rebase rewrote the commit dates"*, and that mechanism was reported as fact.
`git log --format='%ai|%ci'` over the branch found **0 of 24 commits with author ≠ committer** — no
evidence of rewriting at all. ⛔ **Verify the observation before the explanation**, because a
restated observation carries the premise of whatever story was reached for to produce it. The
conductor caught this itself while assembling evidence for a hazard file, and retracted the
mechanism rather than the number.

**What survives, narrowly:** `commits[].committedDate` cannot establish when work started, since a
PR's `createdAt` can precede it. Use a spawn log or another out-of-band record.
