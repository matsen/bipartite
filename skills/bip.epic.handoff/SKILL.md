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
- Without: run `/bip.issue.next` first to create the follow-up issue

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
/bip.issue.next
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
(the conductor's working directory, not the worktree). To find it:
```bash
# Worktree mode: find the main checkout
MAIN_CHECKOUT=$(git worktree list | head -1 | awk '{print $1}')
cat "$MAIN_CHECKOUT/.epic-config.json"
```

Then select or create a slot:

**Clone mode** (`local_worktrees` absent or false):
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
# Find an idle clone (on main, clean worktree)
for name in $(jq -r '.clone_names[]' .epic-config.json); do
  branch=$(git -C "$CLONE_ROOT/$name" branch --show-current 2>/dev/null)
  [ "$branch" = "main" ] && echo "$name"
done
```
Pick the first idle clone. If all are busy, offer to create a new
clone using a name from `new_clone_names` in the config.

**Worktree mode** (`local_worktrees: true`):
```bash
CLONE_ROOT=$(jq -r .clone_root "$MAIN_CHECKOUT/.epic-config.json")
SLOT="$CLONE_ROOT/issue-<N>"
SLUG=$(gh issue view <N> --json title -q '.title' | tr '[:upper:]' '[:lower:]' | awk '{for(i=1;i<=4&&i<=NF;i++) printf "%s%s",$i,(i<4&&i<NF?"-":"")}')

if [ -d "$SLOT" ]; then
  echo "Worktree exists — will resume"
else
  # Clean up leftover branches from previous attempts
  git -C "$MAIN_CHECKOUT" branch --list "<N>-*" | tr -d ' ' | xargs -r git -C "$MAIN_CHECKOUT" branch -D
  # Create worktree from the main checkout
  git -C "$MAIN_CHECKOUT" worktree add "$SLOT" -b "<N>-$SLUG"
fi
```

### Step 5: Prepare the slot

**Clone mode**:
```bash
cd "$CLONE_ROOT/<clone-name>"
git checkout main && git pull --ff-only origin main
rm -f .epic-status.json .epic-worklog.md
```

**Worktree mode** — just clear stale status files:
```bash
rm -f "$SLOT/.epic-status.json" "$SLOT/.epic-worklog.md"
```

### Step 6: Compose prompt and spawn

Follow `/bip.epic.spawn` Steps 3-4 to compose the prompt:
1. Read the new issue: `gh issue view <N>`
2. Compose the full prompt including:
   - `/bip.issue.work <N>` as the core instruction
   - Ralph-loop invocation block
   - EPIC status protocol (`.epic-status.json` fields, worklog format)
   - Any phasing or gate criteria from the issue
3. Use the **Write tool** to create `/tmp/spawn-<N>.txt` with the
   full prompt (do NOT use shell redirection — complex prompts break
   zsh expansion)

Then launch with the correct flags for your mode:

```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)

# Clone mode: --name is NNN-clone (e.g. "281-cedar")
bip spawn --prompt-file /tmp/spawn-<N>.txt \
  --dir "$CLONE_ROOT/<clone-name>" \
  --name "<N>-<clone-name>"

# Worktree mode: --name is NNN-issue-NNN (e.g. "281-issue-281")
bip spawn --prompt-file /tmp/spawn-<N>.txt \
  --dir "$CLONE_ROOT/issue-<N>" \
  --name "<N>-issue-<N>"
```

**IMPORTANT**: Always use `--prompt-file`, never `--prompt "$(cat ...)"`.
Always use `--dir` and `--name` — without them the tmux window gets
a wrong name and the session runs in the wrong directory.

**Do NOT** use raw `tmux new-window` / `tmux send-keys` / `claude`.
Always go through `bip spawn`.

### Step 7: Clean up current slot (optional)

If the current issue's PR is merged:
- **Worktree mode**: the conductor will clean up on next poll
- **Clone mode**: `git checkout main && git pull --ff-only`

Don't force-remove your own worktree while you're in it — just let
the conductor handle cleanup.

### Step 8: Report

```
## Handoff Complete

- Finished: i<current> — <title>
- Spawned: i<N> — <title> → <slot-name>
- EPIC updated: i<epic-number> ✓ (or ✗ conflict, conductor will reconcile)
```

## Conventions

Same as `/bip.epic`: `iN`/`pN` prefixes, full URLs on first mention.
Tmux windows named `NNN-YYY` where NNN is the issue number and YYY is the
clone/slot name (e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).
