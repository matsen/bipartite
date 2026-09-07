---
name: bip-pr-land
description: Land a PR branch — squash merge, clean up local and remote branches, return to main.
---

# /bip-pr-land

Squash-merge the current branch's PR and clean up.

## Usage

```
/bip-pr-land           # Land the current branch's PR
/bip-pr-land #42       # Land PR #42 (if not on that branch)
```

## Worktree mode

If `bip spawn` created the working tree as a linked git worktree (because the user opted into the global `layout: { mode: worktree }` block in `~/.config/bip/config.yml`), this skill detects that in Step 7a and performs the base-branch pull from the primary clone, then removes the worktree in Step 8 before deleting the branch.
In the common clone-mode case (no `layout:` block) every step runs as it always has — the worktree detection is a no-op.

## Workflow

### Step 1: Check for uncommitted work

```bash
git status --porcelain
git diff --stat
```

If there are uncommitted changes or untracked files, you MUST resolve each one explicitly — **never stash and move on**:

1. **Identify every dirty file.**
   For each one, read enough of the diff or file content to understand what it is and why it exists.
2. **Categorize each file:**
   - **Belongs to this PR** (e.g. forgotten formatting fix, test update): stage and commit with a short message.
   - **Unclear**: show the user the file and diff, explain what you see, and ask whether to commit it with the PR or move it aside.
   - **Unrelated / stray**: move it to `_ignore/$(date -I)-landing/` so main stays clean.
     Create the directory if needed.
     Tell the user what you moved and why.
3. **Never use `git stash`.**
   Stashing hides work and risks losing it.
   Every file must be either committed or moved to `_ignore/`.
4. **Ask the user if unsure.**
   If you can't confidently categorize a file, ask.
   A quick question is always better than guessing wrong.

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

If rebase has conflicts, stop and report.
Do not force-push or auto-resolve.

### Step 5: Force-push rebased branch

```bash
git push --force-with-lease
```

### Step 5.5: Wait for CI to pass

Check whether the PR has any CI checks configured, and if so, block until they all pass:

```bash
gh pr checks "$BRANCH" --json name,state,conclusion
```

- **No checks configured** (empty array): proceed immediately.
  This repo has no CI for this PR.
- **Checks present**: wait until all required checks are `COMPLETED` with conclusion `SUCCESS` (or `NEUTRAL`/`SKIPPED`).
  Use `gh pr checks "$BRANCH" --watch --fail-fast` to block.
- **Any check fails**: abort with the failing check name and a link via `gh pr view --web`.
  Do **not** merge.
  Report to user and stop.

Never merge a PR with failing or pending required checks.
If checks are still queued/in progress, wait — do not assume they will pass.

### Step 6: Squash merge via gh

```bash
# If PR closes an issue (check PR body for "closes #N" or "fixes #N"):
gh pr merge --squash --body "closes #N"

# Otherwise:
gh pr merge --squash --body ""
```

Follow the squash merge conventions from global CLAUDE.md — PR title becomes the commit message, body is minimal.

### Step 7a: Detect worktree mode

Before pulling the base branch, check whether you are landing from a linked git worktree (created by `bip spawn` in worktree mode):

```bash
LAND_DIR=$(pwd -P)
if PRIMARY=$(bip worktree primary 2>/dev/null); then
    echo "Landing from worktree $LAND_DIR (primary: $PRIMARY)"
    cd "$PRIMARY"
fi
```

`bip worktree primary` exits 0 and prints the primary clone path **only** when the current directory is a linked worktree.
In every other case (primary clone, non-bip checkout, non-git directory) it exits non-zero with no stdout — `$PRIMARY` remains empty and the `cd` is skipped.

If `$PRIMARY` was set, Steps 7 and 7.5 below run in the primary clone; otherwise they run in `$LAND_DIR` exactly as today.

### Step 7: Return to base branch and pull

```bash
git checkout <base>
git pull
```

### Step 7.5: Sync the primary clone (clone-mode only)

**Skip this step if Step 7a already `cd`'d to the primary** — Step 7 has already pulled it.

If you landed from a scratch clone (EPIC worker, `bip spawn` without worktree mode, or any working copy that isn't the canonical one in `sources.yml`), the primary clone is now behind `origin/<base>`.
Pull it forward so the canonical checkout matches `main`.

1. Resolve the primary clone path the way `bip spawn` does (mirrors `flow.ResolveRepoPath`: `nexus_path` from `~/.config/bip/config.yml`, repo from `git remote get-url origin`, then `sources.yml` + `config.yml` paths).
   If `$(pwd -P)` already equals it, or the repo isn't listed, skip this step.

2. `git -C "$PRIMARY" pull --ff-only`.
   On failure, warn with the error and continue — the merge is already upstream, nothing is lost.
   Never stash.

3. Report what you did in Step 10.

### Step 8: Remove worktree (if applicable) and delete branch

If Step 7a found that you were landing from a linked worktree, remove the worktree **before** deleting its branch — git refuses to delete a branch that still has a worktree checked out on it:

```bash
if [ -n "$PRIMARY" ] && [ -n "$LAND_DIR" ] && [ "$LAND_DIR" != "$PRIMARY" ]; then
    bip worktree remove "$LAND_DIR"
fi
```

`bip worktree remove` defaults to `--force`, which is required because the squash-merge leaves the worktree carrying commits unreachable from the merged branch.
Then delete the local branch:

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

### Step 9.5: Preserve worklog, then clean up orchestration files

`.epic-status.json` and `.epic-worklog.md` are gitignored and stale after
landing, but **do not `rm` them without preserving `.epic-worklog.md`
first** (issue #2216: this step is the confirmed, unconditional deletion
site that destroyed `.epic-worklog.md` for matsengrp/phyz#2314 on PR
matsengrp/phyz#2316 — six source files, six re-pinned regression tests,
four phases of triage, two agent reviews, reasoning gone with no backup).

The premise that used to justify skipping this ("a slot that lands a PR
has its work in git, so the worklog is redundant") is false: `PROSE-DISCIPLINE.md`
has PR bodies rewritten to current state *by design*, so deliberation
deliberately does not live there, and the `issue-lead`'s periodic PR
comments are evaluation-stop summaries, not the worklog's continuous
narrative. On a landed PR the worklog is one of only two places the
reasoning behind the change survives (the other being the PR/issue
comment thread) — landing is exactly the moment this step must not
lose it.

A separately-attempted fix — preserving later, at conductor reclaim —
does not work, because reclaim happens an uncontrolled amount of time
after this step runs; several worklogs survived past landings only by
luck of that gap, not by design. Preserving right here, immediately
before the `rm`, closes the gap outright: there is no window in which
the files exist but nothing has copied them.

Only do this if `.epic-status.json` is present — a plain (non-EPIC)
`/bip-pr-land` run has nothing to preserve. Run the preservation block and
the final `rm` as **two separate commands**, not one pasted-together
script: the preservation block's `$CLONE_ROOT`/`$DEST`/`$ISSUE_N` sit in
the same command string as an `rm` if you don't split them, which
re-arms Claude Code's built-in destructive-removal guard (triggers on any
`$` anywhere in a command string containing `rm`) and blocks on an
approval prompt no permission rule can auto-allow — exactly the trap
`bip-conductor-spawn`'s Step 2 already documents and works around the
same way.

```bash
if [ -f .epic-status.json ]; then
    CLONE_ROOT=$(jq -r '.clone_root' .epic-config.json 2>/dev/null | sed "s|^~|$HOME|")
    ISSUE_N=$(jq -r '.issue // "unknown"' .epic-status.json 2>/dev/null)
    if [ -n "$CLONE_ROOT" ] && [ "$CLONE_ROOT" != "null" ]; then
        DEST="$CLONE_ROOT/.preserved/$ISSUE_N-$(date -I)"
        mkdir -p "$DEST"
        cp .epic-worklog.md .epic-status.json "$DEST/" 2>/dev/null
        printf 'Preserved from %s at land of PR #%s ("%s"), %s.\n' \
            "$(pwd -P)" "<PR number from Step 2>" "<PR title from Step 2>" "$(date -I)" \
            > "$DEST/README.md"
        echo "Preserved worklog+status to $DEST"
    fi
fi
```

Then, as a separate command with no `$` in it at all:

```bash
rm -f .epic-status.json .epic-worklog.md
```

If `CLONE_ROOT` can't be resolved (no `.epic-config.json`, or `jq` fails),
do not silently skip the preservation — report it and copy the worklog
somewhere durable (e.g. `_ignore/$(date -I)-landing/`) instead of losing it.

### Step 10: Confirm

Report: "Landed #42.
On `<base>`, up to date, worktree clean.
Branch `<branch>` deleted."
If any files were moved to `_ignore/`, list them.
If the primary clone was synced in Step 7.5, say so: "Primary clone `<path>` pulled."
If Step 8 removed a linked worktree, say so: "Worktree `<path>` removed."
If Step 9.5 preserved EPIC state, say so: "Worklog preserved to `<DEST>`."
