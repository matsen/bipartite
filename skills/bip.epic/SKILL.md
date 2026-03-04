---
name: bip.epic
description: EPIC dashboard — review progress, plan next steps, spawn work across clones
---

# /bip.epic

Orchestrate work across multiple clones for EPIC-tagged GitHub issues.
Run from the **reference clone** (e.g. `~/re/pz/balsa`) inside a tmux session.

## Overview

This skill scans EPIC issues, checks clone states, builds a status
dashboard, and lets the user dispatch work to idle clones via tmux.

## Conventions

### Issue/PR naming

Always prefix references with `i` for issues and `p` for PRs:
- `i281` = issue #281
- `p275` = PR #275

This applies everywhere: dashboard output, tmux window names, bullet
lists, options, status files. Never use bare `#N` — always `iN` or `pN`.

### Issue link format

The first time you mention an issue or PR in a bullet list, include the
full URL inline, e.g. `i281 (https://github.com/matsengrp/phyz/issues/281)`.
Subsequent mentions of the same item can use short `i281`. Derive the
repo base URL from `gh repo view --json url -q .url`.

### Tmux window naming

Name tmux windows by **clone name**: `cedar`, `oak`, etc. Clone names
are easier to remember than issue numbers and match the filesystem.

### Opening files for review

The user runs the orchestrator in Zed. When you write a file that needs
user review (issue files, reports, etc.), open it in Zed:

```bash
zed <file-path>
```

This applies to ISSUE-*.md files, status reports, or anything where user
feedback is wanted.

## Workflow

### Step 0: Pull main

```bash
git pull --ff-only origin main
```

If this fails (dirty worktree, diverged), report the problem and continue
with the stale state — don't force-reset.

### Step 1: Discover EPIC issues

```bash
gh issue list --label EPIC --json number,title,body
```

If no `EPIC` label exists, fall back to searching for issues whose title
starts with "EPIC:":

```bash
gh issue list --search "EPIC: in:title" --json number,title,body
```

For each EPIC, parse the **Status dashboard** section from the body to
extract:
- Completed items (checked boxes)
- Active items (unchecked boxes with issue/PR references)
- Blocked/future items

### Step 2: Scan clones

The clone root is the **parent directory** of the current working directory.
For example if CWD is `~/re/pz/balsa`, the clone root is `~/re/pz/`.

All clones are interchangeable — any clone can do feature work or run
experiments. (Legacy `run*` clones may still have active work; treat
them the same as any other clone.)

Known clones: alder, ash, balsa, birch, cedar, elm, maple, oak, pine,
teak, run1, run2.

For each clone directory in the clone root, collect:

1. **Git state**:
   ```bash
   git -C <clone> branch --show-current
   git -C <clone> log --oneline -1
   git -C <clone> status --porcelain | head -5
   ```

2. **Epic status file** (if present):
   ```bash
   cat <clone>/.epic-status.json 2>/dev/null
   ```

3. **Tmux window**: Check if a tmux window exists for this clone:
   ```bash
   tmux list-windows -F "#W"
   ```
   Windows are named by clone name (`cedar`, `oak`, etc.).

4. **Staleness**: If `.epic-status.json` exists but its `updated_at` is
   older than 30 minutes and no tmux window is active, mark as **stale**
   (the session likely ended without cleanup).

Classify each clone as:
- **active**: Has a tmux window or fresh `.epic-status.json` (< 30 min)
- **assigned**: On a non-main branch but no active session
- **idle**: On `main`, clean worktree, no active session

### Step 3: Build the dashboard

Present a report with three sections:

#### EPIC Progress

For each EPIC issue, show:
- Title and issue number (e.g. `i284`)
- Summary of what's done, what's active, what's next
- Which clones are working on related issues

#### Clone Status

A table like:

```
Clone   Branch                     Last Commit           Status
-----   ------                     -----------           ------
balsa   main                       5efaa47 (1h ago)      idle (this session)
cedar   276-baseball-iterate-nni   103def3 (2h ago)      assigned (i276), 1 dirty file
oak     198-iterate-benchmark      8056738 (5h ago)      assigned (i198)
teak    main                       1b07327 (8h ago)      idle, 3 dirty files
maple   main                       ba1ac9d (3d ago)      idle
birch   main                       5efaa47 (1h ago)      idle
elm     main                       5efaa47 (1h ago)      idle
ash     main                       5efaa47 (1h ago)      idle
pine    main                       5efaa47 (1h ago)      idle
alder   main                       5efaa47 (1h ago)      idle
run1    277-band-diagnostics       da9773b (3h ago)      assigned (i277)
run2    main                       1aa50fd (1d ago)      idle
```

#### Actionable Next Steps

Cross-reference EPIC active items with clone status. List concrete actions
the user could take, such as:
- "cedar has uncommitted work on i276 — resume or clean up?"
- "i281 (richer indel signals) is unstarted — assign to an idle clone?"
- "i278 (baseball × max-failures experiment) needs a run clone"

### Step 4: Offer options

Use `AskUserQuestion` to let the user pick what to do next. Options are
generated dynamically based on the dashboard. Common option types:

- **Start work on iN in clone X** — spawns a Claude session
- **Resume work on clone X (iN)** — opens a tmux window in an assigned clone
- **Clean up stale clone X** — switch back to main, delete branch
- **Run experiment iN on clone X** — syncs to remote and spawns

Present 3-4 of the most useful options plus "Other" for custom requests.

### Step 5: Spawn work

When the user picks an action that involves spawning a Claude session,
use tmux to create a new window and launch Claude Code with a carefully
composed prompt.

#### Tmux spawning pattern

```bash
# Write the prompt to a temp file (too long for shell args)
PROMPT_FILE=$(mktemp /tmp/epic-XXXXXX.txt)
cat > "$PROMPT_FILE" << 'PROMPT_EOF'
<composed prompt here>
PROMPT_EOF

# Create tmux window named by clone, in the clone directory
tmux new-window -n "<clone-name>" -c "<clone-path>"

# Launch Claude Code with the prompt
tmux send-keys -t "<clone-name>" \
  "claude --dangerously-skip-permissions \"\$(cat $PROMPT_FILE)\"; rm -f $PROMPT_FILE" Enter
```

#### Composed prompt structure

The prompt sent to the spawned session MUST include:

1. **The task**: Issue number, title, and relevant context from the EPIC
2. **Epic status protocol**: Instructions to maintain `.epic-status.json`
   (see below)
3. **`/work-issue <number>`**: Delegates the actual implementation to the
   work-issue skill, which handles branching, coding, pre-merge checks,
   and PR creation

Example spawned prompt:

```
You are working on GitHub issue i281 "Richer PRANK indel signals for iterate
NNI edge ranking" in the phyz project.

EPIC STATUS PROTOCOL — You MUST follow this:
1. At session start, write .epic-status.json (see format below)
2. Update it when you transition between phases (exploring → coding → testing)
3. Update it when you finish or encounter a blocker

.epic-status.json format:
{
  "issue": 281,
  "title": "Richer PRANK indel signals for iterate NNI edge ranking",
  "phase": "exploring",
  "summary": "Reading issue and exploring codebase",
  "updated_at": "<ISO 8601 timestamp>",
  "blockers": []
}

Phases: exploring, coding, testing, blocked, completed

Now run: /work-issue 281
```

The `/work-issue` skill handles the full lifecycle: reading the issue,
creating a branch, implementing, running `/pre-merge-check`, and creating
a PR. The spawned prompt only needs to add the epic status protocol on
top of that.

#### Creating new clones

If all dev clones are busy and the user wants to start new work:

```bash
cd ~/re/pz
git clone git@github.com:matsengrp/phyz.git <new-name>
```

Pick wood-themed names: `walnut`, `cherry`, `willow`, `spruce`, `juniper`,
`hemlock`, `poplar`, `rowan`. Check which names are already taken.

#### Cleaning up stale clones

Before reusing an assigned clone:

1. Check if there's an open PR for the branch
2. If merged/closed, safe to reset: `git checkout main && git pull`
3. If open, warn the user — they may want to resume instead

### Step 6: Monitor (optional)

After spawning, offer to check on running sessions:

```bash
# Read status files from all active clones
for d in ~/re/pz/*/; do
  [ -f "$d/.epic-status.json" ] && echo "=== $(basename $d) ===" && cat "$d/.epic-status.json"
done
```

## .epic-status.json specification

```json
{
  "issue": 281,
  "title": "Richer PRANK indel signals for iterate NNI edge ranking",
  "phase": "exploring | coding | testing | blocked | completed",
  "summary": "Human-readable one-liner of current activity",
  "updated_at": "2026-03-03T14:30:00Z",
  "blockers": ["waiting for i266 to merge", "need data from run1"]
}
```

- Written to `<clone-root>/.epic-status.json`
- **Must be .gitignored** — add to project .gitignore if not present
- Stale after 30 minutes with no tmux window = session ended uncleanly

## Error handling

- **Not in tmux**: Warn and exit — tmux is required for spawning
- **No EPIC issues found**: Report and offer to create one
- **Clone root detection fails**: Ask user for the clone root path
- **gh not authenticated**: Report error, suggest `gh auth login`

## Notes

- The conductor session (running this skill) stays in the reference clone
  on `main`. It does NOT do feature work itself.
- Multiple spawned sessions can run in parallel in different tmux windows.
- Run clones should use `make remote-sync` + `make remote-tmux` for
  experiment work on compute servers. The spawned prompt should include this.
- Always run `/bip.scout` before dispatching remote experiment work to pick
  an available server.
