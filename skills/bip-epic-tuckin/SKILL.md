---
name: bip-epic-tuckin
description: Persist topic-side EPIC state before context reset
---

# /bip-epic-tuckin

Flush topic state — EPIC bodies, findings, decisions — to durable
storage before a context reset or session end. Run this when context
is getting long or before stopping.

Fleet-side state (clone/slot status, `.epic-status.json`, pending
spawn intent) is `/bip-conductor-tuckin`'s job, not this skill's — run
that separately if the conductor session is also resetting.

## Usage

```
/bip-epic-tuckin
```

## Workflow

### Step 1: Push EPIC body edits

For each `ISSUE-EPIC-*.md` file in the repo root, follow the **EPIC body
update pattern** from `/bip-epic` (pull with `updatedAt` tracking → edit →
conflict-check → push). Extract the issue number from the filename
(`ISSUE-EPIC-284.md` → 284). Prune completed sections while you're in
there — this is the same discipline as `/bip-epic`'s regular update
pattern, not a separate cleanup pass.

Report which EPICs were pushed (and which were skipped due to conflicts).

### Step 2: Update MEMORY.md (lightweight)

Most state should already be in EPIC bodies (`ISSUE-EPIC-<N>.md`).
Only update MEMORY.md for topic-level context that doesn't fit there:

- Key decisions and their rationale
- Cross-EPIC patterns or dependencies
- Dependency-direction or collision findings not yet folded into an
  EPIC body
- Things the next session should know that aren't obvious from the files

### Step 3: Report

Print a summary:

```
## Tuckin Complete

- EPICs pushed: i281, i295
- EPICs skipped (conflict): i310
- MEMORY.md updated: yes

Safe to reset context. Fleet-side state (clones, slots) is unaffected
by this — run /bip-conductor-tuckin if that session is resetting too.
```
