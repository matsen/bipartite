---
name: bip.epic.tuckin
description: Persist orchestrator state before context reset
---

# /bip.epic.tuckin

Flush orchestrator state to durable storage before a context reset or
session end. Run this when context is getting long or before stopping.

## Usage

```
/bip.epic.tuckin
```

## Workflow

### Step 1: Push EPIC body edits

EPIC bodies live as `ISSUE-EPIC-<N>.md` files in the repo root (gitignored).
These are the working copies — edit them locally, then push to GitHub.

For each `ISSUE-EPIC-*.md` file in the repo root:

1. Extract the issue number from the filename (`ISSUE-EPIC-284.md` → 284)
2. Conflict check before pushing:
   ```bash
   CURRENT_AT=$(gh issue view <number> --json updatedAt -q .updatedAt)
   ```
   Compare against the `updatedAt` recorded when the file was last pulled.
   If it changed, someone else edited — re-pull, merge, and retry.
3. Push to GitHub:
   ```bash
   gh issue edit <number> --body-file ISSUE-EPIC-<N>.md
   ```

During the session, always edit the local file first, then push. This
ensures the local file is the source of truth and survives context resets.

If editing EPIC bodies mid-session (not just at tuckin), follow the same
pattern: edit the local `ISSUE-EPIC-<N>.md`, then push with `gh issue edit`.

Report which EPICs were pushed (and which were skipped due to conflicts).

### Step 2: Update slot status files

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

Slot paths:
- *Clone mode*: `$CLONE_ROOT/<clone-name>`
- *Worktree mode*: `$CLONE_ROOT/issue-<N>`

For each slot the orchestrator interacted with this session:

1. Check if the slot's `.epic-status.json` is stale or missing
2. If the orchestrator has newer information (e.g., a slot finished,
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

Update slots the orchestrator has direct knowledge about. This includes:
- Slots whose PRs were merged this session (even if the slot's own
  session already exited)
- Slots the orchestrator observed finishing via tmux or poll

Do not guess status for slots with active sessions that may have
progressed beyond what the orchestrator last observed.

### Step 3: Update MEMORY.md (lightweight)

Most state should already be in EPIC bodies (`ISSUE-EPIC-<N>.md`) and
clone status files (`.epic-status.json`). Only update MEMORY.md for
orchestrator-level context that doesn't fit in those files:

- Key decisions and their rationale
- Cross-EPIC patterns or dependencies
- Things the next session should know that aren't obvious from the files

### Step 4: Report

Print a summary:

```
## Tuckin Complete

- EPICs pushed: i281, i295
- EPICs skipped (conflict): i310
- Slot status updated: cedar, issue-295
- MEMORY.md updated: yes

Safe to reset context.
```
