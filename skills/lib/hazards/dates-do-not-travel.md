---
tags: [measurement, records]
measured: 2026-09-21
---

# Dates do not travel with content

Two failures, opposite ends of the same defect. Both were hit on 2026-09-21 by a session that had
spent the day correcting other people's numbers.

## Reading: a line carries no date

**Looks right:** `grep -rn 'X' FILE` to decide whether a finding is already recorded.

**What happens:** you learn `X` is in the file. You conclude `X` was known before today. The file
does not say when the line arrived.

Measured in one file, `matsen/bipartite` `EVIDENCE-DISCIPLINE.md` — one bullet, two ages:

| text | entered | age when cited |
|---|---|---|
| the `grep`-is-a-shell-function bullet | `e696b0a`, 2026-09-12 | 9 days |
| `--ignore-files`, in the same bullet | `3216d7c`, 2026-09-21 04:45 | ~6 hours |

A conductor read the merged bullet, reported a colleague's few-hours-old finding as nine days old
and already-known, and was one message from filing it as a process failure.

**Check:** `git log --oneline -S'<the exact text>' -- <file>`. Run it on the clause you are citing,
not on the section it lives in.

## Writing: a dated entry is evidence about its date

⛔ **The mirror, and it is worse because the timestamp is right there.** An entry written
`2026-09-10` records what was true then. Nothing in it says *as of today*, so a later reader —
usually its own author — cites it as a standing fact.

Measured in one session's own append-only fleet log:

```
11:42:32Z   "names rotate on resume; look them up at send time"       <- the standing rule
12:25:30Z   "the address is the session name, never the tmux
             window name I assigned at spawn"                          <- a same-day observation
```

Eleven days later the author quoted the 12:25 line as a general rule about the transport, and
asserted it while successfully sending ten messages to a session addressed by exactly the tmux
window name it says cannot work. The 11:42 note — undated in its phrasing, and the actual rule —
was not retrieved.

The two entries are **43 minutes apart in one session.** This is not decay. The correct fact was
present; the wrong one was shorter and more recent, so this is a retrieval failure rather than a
memory one.

**Convention, since there is no command for this half:** put the scope in the sentence, not only in
the heading. *"As of 2026-09-10, X"* forces a citer to notice the date; a heading above a paragraph
does not travel when the paragraph is quoted.

## Why the two compound

The reading side makes a dated observation citable as current. The writing side makes an undated
standing rule invisible as a rule. Hit together, you cite the wrong entry *over* the right one, and
the record contains both. Counted as instance 5 in `#242`'s tally, where the argument about what
this implies for a fact store lives.
