---
name: bip-conductor-tuckin
description: Persist fleet-side clone/slot state before context reset
---

# /bip-conductor-tuckin

Flush fleet state — clone/slot status, pending spawn intent — to
durable storage before a context reset or session end. Run this when
context is getting long or before stopping.

Topic-side state (EPIC bodies, findings, decisions) is
`/bip-epic-tuckin`'s job, not this skill's — run that separately if the
epic session is also resetting.

## Usage

```
/bip-conductor-tuckin
```

## Workflow

### Step 1: Update slot status files

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

Slot paths:
- *Clone mode*: `$CLONE_ROOT/<clone-name>`
- *Worktree mode*: `$CLONE_ROOT/issue-<N>`

For each slot the conductor interacted with this session:

1. Check if the slot's `.epic-status.json` is stale or missing
2. If the conductor has newer information (e.g., a slot finished,
   got blocked, or changed phase), update the file:
   ```bash
   CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
   cat > "$CLONE_ROOT/<slot>/.epic-status.json" << 'EOF'
   {
     "issue": <N>,
     "title": "<title>",
     "phase": "<current phase>",
     "summary": "<what happened>",
     "updated_at": "<ISO 8601 timestamp>",
     "blockers": []
   }
   EOF
   ```

Update slots the conductor has direct knowledge about. This includes:
- Slots whose PRs were merged this session (even if the slot's own
  session already exited)
- Slots the conductor observed finishing via tmux or poll

Do not guess status for slots with active sessions that may have
progressed beyond what the conductor last observed.

### Step 2: Note queued and consumed spawn intent

List `$CLONE_ROOT/.spawn-prompts/` and report two groups separately —
these are the epic's authored intent, not conductor state, so don't
touch their contents either way:

- **Queued** — regular files directly in `.spawn-prompts/`:
  `find "$CLONE_ROOT/.spawn-prompts" -maxdepth 1 -type f \( -name
  '*.md' -o -name 'spawn-*.txt' \)` (both naming conventions live
  there). `-type f` matters: a plain `ls` also lists the `consumed/`
  directory itself as an entry, which is not a queued intent file.
  Still waiting on execution.
- **Consumed** — files under `.spawn-prompts/consumed/`. Already
  launched by `/bip-conductor-spawn` Step 6.

Reporting these separately matters: a consumed file left in the top
level (or a queued one reported as consumed) invites either a
re-spawn of work that's already running or merged, or a queued brief
getting ignored as if it were stale. Report both groups so the next
conductor session (or the user) knows what's actually queued vs
already launched.

### Step 3: Filter and route fleet-level findings

Most state should already be in `.epic-status.json` per slot. Before
recording anything anywhere, run each candidate fleet-level finding
through this filter:

1. **Is it derived?** Recomputable from `git`, `tmux`, `gh`, or the
   filesystem — record nothing, whatever the destination would have
   been. Which remote host had a warm cache, which host was mid-build,
   local cache sizes, open-issue counts, which worker was blocked: all
   of these are re-measurable on demand and go stale within hours, so
   don't write them down anywhere, including MEMORY.md.
2. **Is it already recorded?** A finding that produced an issue, PR,
   test, or doc needs no second copy.

Only what survives both gates gets a destination:
- A durable repo-level fact → a `CLAUDE.md`.
- A workflow rule → a skill.
- A finding → the test, doc, or issue it came from.
- Nothing else fits → the Step 4 report below. This is where a user
  who has opted out of auto-memory files will actually see it — don't
  treat a MEMORY.md write as the only or required destination.

### Step 4: Report

Print a summary:

```
## Tuckin Complete

- Slot status updated: cedar, issue-295
- Queued spawn intent: i302 (retry logic)
- Consumed spawn intent: i298 (already launched, moved to consumed/)
- Fleet-level findings: <none survived the filter | routed to CLAUDE.md/skill/issue as listed>

Safe to reset context. Topic-side state (EPIC bodies) is unaffected by
this — run /bip-epic-tuckin if that session is resetting too.
```
