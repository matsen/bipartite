---
name: bip-ms-tuckin
description: Persist manuscript session state before context reset
---

# /bip-ms-tuckin

Flush manuscript session state to durable storage before a context reset
or session end. Run this when context is getting long or before stopping.

## Usage

```
/bip-ms-tuckin
```

## Workflow

### Step 1: Check for uncommitted manuscript changes

```bash
cd <manuscript-root>
git diff --stat
git diff --cached --stat
```

If there are uncommitted changes to TeX files, ask the user whether to
commit them. Draft a commit message summarizing what changed (e.g.,
"Update implementation notes: Phase 1b done, branchScale normalization").
Do not commit without confirmation.

If there are only untracked `ISSUE-*.md` files, note them but do not
commit (these are intentionally untracked per CLAUDE.md).

### Step 2: Push EPIC body updates

For each tracked repo in `.ms-config.json`, check whether the EPIC
body was edited this session (compare the `updatedAt` from the start
of the session to the current `updatedAt`). If not pushed yet, push
any pending EPIC body updates:

```bash
gh issue view <epic-number> --repo <org/repo> --json updatedAt
```

Report which EPICs were pushed and which were already up to date.

### Step 3: Update `.ms-config.json`

Check whether `fetch_cmds` should be updated based on new experiments
or results that arrived this session. If new result paths were
discovered but not added to `fetch_cmds`, note them for the user.

### Step 4: Update memory

Update the auto-memory files to reflect the session's work:

1. **`project_dasmfit_status.md`** (or equivalent project memory):
   - Mark newly completed items
   - Update open issue list
   - Note the current bottleneck
   - Record any key decisions made this session

2. **`project_pending_decisions.md`** (if it exists):
   - Remove resolved decisions
   - Update status of pending decisions
   - Add any new unresolved items

3. **Feedback memories**: If the user corrected an approach or
   confirmed a non-obvious choice, save it.

Only update memories that changed. Do not rewrite unchanged files.

### Step 5: Verify build

```bash
make pdf
```

Ensure the manuscript builds cleanly. If it doesn't, fix the build
error before completing tuckin.

### Step 6: Report

Print a summary:

```
## Tuckin Complete

### Manuscript
- Committed: <yes/no, commit hash if yes>
- Build: <clean/broken>
- Uncommitted ISSUE files: <count>

### EPICs
- <repo> i<N>: <pushed/up-to-date/skipped>

### Issues filed this session
- <repo>#<N>: <title> — <status>

### Pending for next session
- <brief list of what's unfinished>

### Key decisions
- <any decisions the next session should know about>

Safe to reset context.
```
