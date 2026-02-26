---
name: land
description: Land a PR branch — squash merge, clean up local and remote branches, return to main.
---

# /land

Squash-merge the current branch's PR and clean up.

## Usage

```
/land           # Land the current branch's PR
/land #42       # Land PR #42 (if not on that branch)
```

## Workflow

### Step 1: Identify the PR

```bash
# Get current branch
BRANCH=$(git branch --show-current)

# Find the PR for this branch
gh pr view "$BRANCH" --json number,title,state,baseRefName
```

If no PR found, abort: "No PR found for branch `$BRANCH`."
If PR is not open, abort: "PR is already `$STATE`."

Save the base branch name (usually `main` or `master`) from `baseRefName`.

### Step 2: Confirm with user

Show the PR title and number, and ask for confirmation before proceeding:

```
About to land: #42 "Add feature X" (branch: my-feature → main)
Proceed?
```

**STOP and wait for user confirmation.**

### Step 3: Update base branch and rebase

```bash
git fetch origin
git rebase origin/<base>
```

If rebase has conflicts, stop and report. Do not force-push or auto-resolve.

### Step 4: Force-push rebased branch

```bash
git push --force-with-lease
```

### Step 5: Squash merge via gh

```bash
# If PR closes an issue (check PR body for "closes #N" or "fixes #N"):
gh pr merge --squash --body "closes #N"

# Otherwise:
gh pr merge --squash --body ""
```

Follow the squash merge conventions from global CLAUDE.md — PR title becomes the commit message, body is minimal.

### Step 6: Return to base branch and pull

```bash
git checkout <base>
git pull
```

### Step 7: Delete local branch

```bash
git branch -d <branch>
```

The remote branch is already deleted by `gh pr merge` (GitHub default).
If not, also run: `git push origin --delete <branch>`

### Step 8: Confirm

Report: "Landed #42. On `<base>`, up to date. Branch `<branch>` deleted."
