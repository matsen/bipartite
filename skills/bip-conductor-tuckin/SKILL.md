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
   CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
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

### Step 2: Note pending spawn intent

List any `$CLONE_ROOT/.spawn-prompts/*.md` files still waiting on
execution — these are the epic's authored intent, not conductor state,
so don't touch their contents, just confirm they're still there and
report them so the next conductor session (or the user) knows what's
queued.

### Step 3: Update MEMORY.md (lightweight)

Most state should already be in `.epic-status.json` per slot. Only
update MEMORY.md for fleet-level context that doesn't fit there:

- Host/cache quirks (which remote host had a warm cache for what,
  which host was mid-build)
- Clone-pool layout decisions
- Things the next conductor session should know that aren't obvious
  from the slot files

### Step 4: Report

Print a summary:

```
## Tuckin Complete

- Slot status updated: cedar, issue-295
- Pending spawn intent: i302 (retry logic)
- MEMORY.md updated: yes

Safe to reset context. Topic-side state (EPIC bodies) is unaffected by
this — run /bip-epic-tuckin if that session is resetting too.
```
