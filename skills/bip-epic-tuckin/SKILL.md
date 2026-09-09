---
name: bip-epic-tuckin
description: Persist topic-side EPIC state before context reset
---

# /bip-epic-tuckin

Flush topic state — EPIC bodies, findings, decisions — to durable storage before a context reset or session end.
Run this when context is getting long or before stopping.

Fleet-side state (clone/slot status, `.epic-status.json`, pending spawn intent) is `/bip-conductor-tuckin`'s job, not this skill's — run that separately if the conductor session is also resetting.

## Usage

```
/bip-epic-tuckin
```

## Workflow

### Step 1: Push EPIC body edits

For each `ISSUE-EPIC-*.md` file in the repo root, follow the **EPIC body update pattern** from `/bip-epic` (pull with `updatedAt` tracking → edit → conflict-check → push).
Extract the issue number from the filename (`ISSUE-EPIC-284.md` → 284).
Prune completed sections while you're in there — this is the same discipline as `/bip-epic`'s regular update pattern, not a separate cleanup pass.

Report which EPICs were pushed (and which were skipped due to conflicts).

### Step 2: Filter and route topic-level findings

Most state should already be in EPIC bodies (`ISSUE-EPIC-<N>.md`).
Before recording anything anywhere, run each candidate topic-level finding through this filter:

1. **Is it derived?**
   Recomputable from `git`, `gh`, or the EPIC bodies themselves — record nothing.
2. **Does it already have a home?**
   Dependency-direction or collision findings belong in the EPIC body they concern (Step 1), not as a separate copy.
   A finding that already produced an issue, PR, or EPIC body update needs no second copy either.

Only what survives both gates gets a destination:
- A durable repo-level fact → a `CLAUDE.md`.
- A workflow rule → a skill.
- A cross-EPIC pattern or key decision not yet captured anywhere → the EPIC body it's most relevant to, or the Step 3 report below if none fits.
- Nothing else fits → the Step 3 report.
  This is where a user who has opted out of auto-memory files will actually see it — don't treat a MEMORY.md write as the only or required destination.

### Step 2b: Write the continuation artifact — a long file plus a short prompt

**Two artifacts, and the split is the point.** A continuation prompt longer than a page does not get pasted; a file nobody is pointed at does not get read.

**The file** — `CONTINUATION-<N>.md` in the repo root (add to `.gitignore` if not already covered) — can be as long as it needs to be. Include:
- **How the programme works**: the slicing discipline, what is out of scope by default, when to reach for a skeptic agent.
- **The traps** that caused the most rework — naming collisions, artifacts everyone reads wrong, anything where each reasoning step was locally valid and the conclusion still came out wrong.
- **Current state with its numbers**, and the pointer to the EPIC section that supersedes everything else.
- **What has already been ruled out**, itemised. This is the highest-value section and the easiest to omit — without it a successor re-runs settled work.
- **What is in flight**, per worker, and what each one's next decision point is.
- **Housekeeping**: EPIC body headroom, known recurring conflicts, skill changes landed this session.

**The prompt** is what you print for the user, and it stays under roughly fifteen lines: the `/bip-epic <N>` invocation, a one-line pointer to the file, **the two or three traps that would cost the most if rediscovered**, one sentence on the working method, and the fleet's current state.

**Put the traps in the prompt itself, not only in the file.** They are precisely what a fresh session gets wrong *before* it has read anything.

### Step 3: Report

Print a summary:

```
## Tuckin Complete

- EPICs pushed: i281, i295
- EPICs skipped (conflict): i310
- Topic-level findings: <none survived the filter | routed to CLAUDE.md/skill/EPIC body as listed>
- Continuation artifact: CONTINUATION-<N>.md written; short prompt printed below

Safe to reset context. Fleet-side state (clones, slots) is unaffected
by this — run /bip-conductor-tuckin if that session is resetting too.
```
