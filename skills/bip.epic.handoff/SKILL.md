---
name: bip.epic.handoff
description: Worker self-spawns a follow-up issue and hands off to a new slot
---

# /bip.epic.handoff

Hand off work from a finishing worker session to a new slot. Run this
from a **worker** (not the conductor) when your current issue is done
and you know what should happen next.

## Prerequisites — direction must be decided FIRST

**Do not run this skill until you have a clear next direction.**

If you have just finished an issue and are unsure what comes next:
1. Think through what you learned, what's unresolved, and what the
   logical next step is
2. Write a **prose proposal** (2-4 sentences) describing the direction
   and why it's the right next move
3. Present it to the user and **wait for confirmation**
4. Only after the user agrees, proceed with the handoff

This skill is for execution, not ideation. The thinking happens before
you invoke it.

## Usage

```
/bip.epic.handoff [issue-number]
```

- With an issue number: spawn work for an existing issue
- Without: run `/issue-next` first to create the follow-up issue

## Workflow

### Step 1: Ensure current work is complete

Before handing off, verify:
- Current branch is pushed
- PR is created (or work was exploratory with no PR needed)
- `.epic-status.json` phase is `completed` or `quality-gate`

If not, finish the current work first.

### Step 2: Create follow-up issue (if needed)

If no issue number was provided:
```
/issue-next
```

This drafts and creates the follow-up issue. Record the new issue
number as `<N>`.

### Step 3: Update the EPIC body

Add the new issue to the EPIC dashboard so the conductor sees it.
Follow the **EPIC body update pattern** from `/bip.epic`:

1. Pull the current EPIC body with conflict tracking (`updatedAt`)
2. Add the new issue to the status dashboard (unchecked)
3. If the current issue's PR is ready, check its box
4. Conflict-check and push

If the conflict check fails, report it and continue — the conductor
will reconcile on next poll.

### Step 4: Select or create a slot

Read `.epic-config.json` from the repo root.

**Finding the config**: In clone mode, `.epic-config.json` is in each
clone's repo root. In worktree mode, it's in the **main checkout**
(not the worktree). If you can't find it:
```bash
# Worktree mode: find the main checkout
git worktree list | head -1 | awk '{print $1}'
# Then read .epic-config.json from there
```

Then follow the slot selection logic from `/bip.epic.spawn` Step 1:
- **Clone mode**: find an idle clone (on `main`, clean worktree)
- **Worktree mode**: create `issue-<N>` worktree

### Step 5: Compose prompt and spawn

Follow `/bip.epic.spawn` Steps 3-5 exactly:
1. Read the new issue (`gh issue view <N>`)
2. Compose the prompt (work instructions + ralph-loop + epic status
   protocol) — use the full template from `/bip.epic.spawn` Step 4
3. Write to `/tmp/spawn-<N>.txt`
4. Launch via `bip spawn --prompt-file /tmp/spawn-<N>.txt`

### Step 6: Clean up current slot (optional)

If the current issue's PR is merged:
- **Worktree mode**: the conductor will clean up on next poll
- **Clone mode**: `git checkout main && git pull --ff-only`

Don't force-remove your own worktree while you're in it — just let
the conductor handle cleanup.

### Step 7: Report

```
## Handoff Complete

- Finished: i<current> — <title>
- Spawned: i<N> — <title> → <slot-name>
- EPIC updated: i<epic-number> ✓ (or ✗ conflict, conductor will reconcile)
```

## Conventions

Same as `/bip.epic`: `iN`/`pN` prefixes, full URLs on first mention.
Tmux windows named by clone name (clone mode) or `i<N>` (worktree mode).
