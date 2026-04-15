---
name: bip.land
description: Land a PR branch — squash merge, clean up local and remote branches, return to main.
---

# /bip.land

Squash-merge the current branch's PR and clean up.

## Usage

```
/bip.land           # Land the current branch's PR
/bip.land #42       # Land PR #42 (if not on that branch)
```

## Workflow

### Step 1: Check for uncommitted work

```bash
git status --porcelain
git diff --stat
```

If there are uncommitted changes or untracked files, you MUST resolve each one explicitly — **never stash and move on**:

1. **Identify every dirty file.** For each one, read enough of the diff or file content to understand what it is and why it exists.
2. **Categorize each file:**
   - **Belongs to this PR** (e.g. forgotten formatting fix, test update): stage and commit with a short message.
   - **Unclear**: show the user the file and diff, explain what you see, and ask whether to commit it with the PR or move it aside.
   - **Unrelated / stray**: move it to `_ignore/$(date -I)-landing/` so main stays clean. Create the directory if needed. Tell the user what you moved and why.
3. **Never use `git stash`.** Stashing hides work and risks losing it. Every file must be either committed or moved to `_ignore/`.
4. **Ask the user if unsure.** If you can't confidently categorize a file, ask. A quick question is always better than guessing wrong.

### Step 2: Identify the PR

```bash
# Get current branch
BRANCH=$(git branch --show-current)

# Find the PR for this branch
gh pr view "$BRANCH" --json number,title,state,baseRefName
```

If no PR found, abort: "No PR found for branch `$BRANCH`."
If PR is not open, abort: "PR is already `$STATE`."

Save the base branch name (usually `main` or `master`) from `baseRefName`.

### Step 3: Log and proceed

Print the PR summary line, then continue without waiting for confirmation:

```
Landing: #42 "Add feature X" (branch: my-feature → main)
```

### Step 4: Update base branch and rebase

```bash
git fetch origin
git rebase origin/<base>
```

If rebase has conflicts, stop and report. Do not force-push or auto-resolve.

### Step 5: Force-push rebased branch

```bash
git push --force-with-lease
```

### Step 5.5: Wait for CI to pass

Check whether the PR has any CI checks configured, and if so, block until they all pass:

```bash
gh pr checks "$BRANCH" --json name,state,conclusion
```

- **No checks configured** (empty array): proceed immediately. This repo has no CI for this PR.
- **Checks present**: wait until all required checks are `COMPLETED` with conclusion `SUCCESS` (or `NEUTRAL`/`SKIPPED`). Use `gh pr checks "$BRANCH" --watch --fail-fast` to block.
- **Any check fails**: abort with the failing check name and a link via `gh pr view --web`. Do **not** merge. Report to user and stop.

Never merge a PR with failing or pending required checks. If checks are still queued/in progress, wait — do not assume they will pass.

### Step 6: Squash merge via gh

```bash
# If PR closes an issue (check PR body for "closes #N" or "fixes #N"):
gh pr merge --squash --body "closes #N"

# Otherwise:
gh pr merge --squash --body ""
```

Follow the squash merge conventions from global CLAUDE.md — PR title becomes the commit message, body is minimal.

### Step 7: Return to base branch and pull

```bash
git checkout <base>
git pull
```

### Step 7.5: Sync the primary clone

If you landed from a scratch clone (EPIC worker, `bip spawn`, or any
working copy that isn't the canonical one in `sources.yml`), the primary
clone is now behind `origin/<base>`. For compiled projects — we mostly
use Zig and C++ — this matters: binaries symlinked from the primary
clone onto `$PATH` go stale silently.

1. Resolve the primary clone path the way `bip spawn` does (mirrors
   `flow.GetRepoLocalPath`: `nexus_path` from `~/.config/bip/config.yml`,
   repo from `git remote get-url origin`, then `sources.yml` +
   `config.yml` paths). If `$(pwd -P)` already equals it, or the repo
   isn't listed, skip this step.

2. `git -C "$PRIMARY" pull --ff-only`. On failure, warn with the error
   and continue — the merge is already upstream, nothing is lost. Never
   stash.

3. If `$PRIMARY` is a compiled project, rebuild it using the project's
   documented build command (check `CLAUDE.md` / `README.md`). Skip
   interpreted projects. On build failure, warn and continue.

4. Report what you did in Step 10.

### Step 8: Delete local branch

```bash
git branch -d <branch>
```

The remote branch is already deleted by `gh pr merge` (GitHub default).
If not, also run: `git push origin --delete <branch>`

### Step 9: Ensure clean main

```bash
git status --porcelain
```

If any untracked or modified files remain on main:
- Move them to `_ignore/$(date -I)-landing/` (create the directory if needed)
- Report what was moved

The goal is a **totally clean `git status`** on main when landing is done.

### Step 9.5: Clean up orchestration files

Remove EPIC worker state files if present (these are gitignored and
stale after landing):

```bash
rm -f .epic-status.json .epic-worklog.md
```

### Step 10: Confirm

Report: "Landed #42. On `<base>`, up to date, worktree clean. Branch `<branch>` deleted."
If any files were moved to `_ignore/`, list them.
If the primary clone was synced in Step 7.5, say so:
"Primary clone `<path>` pulled" and, if rebuilt, "and rebuilt (`<command>`)."
