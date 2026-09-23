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

## Where to stand

This skill moves `HEAD` (Steps 7 and 8) and preserves the state files of the directory it runs in (Step 7a). Never run it in a clone another live session is working in. To land a PR checked out in another session's clone, don't run this skill: run Step 6's guard from any checkout of the repo, against the PR head instead of `HEAD`, so nothing moves:

```bash
git fetch -q origin "+pull/<N>/head:refs/pr/<N>" <base> \
  && test "$(git rev-parse refs/pr/<N>)" = "$(gh pr view <N> --json headRefOid -q .headRefOid)" \
  && git merge-base --is-ancestor origin/<base> refs/pr/<N> \
  && gh pr merge <N> --squash --body "closes #N"
```

If that clone is a fleet slot, reclaim it through `/bip-conductor`, which runs the issue-lead's terminal ceremony before preserving (Step 7a here would delete the status file the ceremony keys on).

Check for co-tenancy:

```sh
for p in $(pgrep -x claude); do readlink /proc/$p/cwd; done | /usr/bin/grep -cE "^$PWD(/|$)"
```

More than 1 means co-tenancy. The count is a floor: it cannot see a session that `cd`s into this clone inside its Bash calls. For read-only inspection of a PR, prefer `git show <sha>:<path>` with `gh pr view <N> --json headRefOid`; `git fetch origin pull/N/head:<branch>` silently leaves an existing local branch at its old SHA (use `-f`).

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

First ask GitHub which issues the PR will close — its own parse, which handles code spans and word boundaries:

```bash
gh pr view <N> --json closingIssuesReferences --jq '.closingIssuesReferences[].number'
```

Empty is a defect if the PR has an issue. The field is eventually consistent, so re-read before concluding an edit to the body did not take.

Merge with the guard **inside** the same command, so the base cannot move between check and merge, and always pass `--body` — without it `gh` concatenates every branch commit body into the squash message, and GitHub closes any issue a closing keyword in there names, negated or quoted or not:

```bash
test "$(git rev-parse HEAD)" = "$(gh pr view <N> --json headRefOid -q .headRefOid)" \
  && git fetch -q origin <base> \
  && git merge-base --is-ancestor origin/<base> HEAD \
  && gh pr merge --squash --body "closes #N"      # or --body "" if it closes nothing
```

If the guard fails, the base moved or HEAD differs from the pushed head: go back to Step 4. An approval names a SHA, so a moved base voids it.

### Step 7a: Preserve EPIC state, then detect worktree mode

Run this as **one** Bash invocation: shell variables do not survive between calls, and the delete of the originals must be conditional on the preserve having verified. It must run before Step 8, which in worktree mode removes the whole directory.

```bash
LAND_DIR=$(pwd -P)
SPAWN_INTENT=""
for cand in ~/.claude/skills/lib/spawn-intent.sh ~/re/bipartite/skills/lib/spawn-intent.sh; do
    [ -r "$cand" ] && { SPAWN_INTENT="$cand"; break; }
done
if [ -f "$LAND_DIR/.epic-status.json" ]; then
    [ -n "$SPAWN_INTENT" ] || { echo "STOP: spawn-intent.sh not found; nothing preserved" >&2; exit 1; }
    source "$SPAWN_INTENT"
    if [ -f "$LAND_DIR/.epic-config.json" ]; then
        CLONE_ROOT=$(resolve_clone_root "$LAND_DIR/.epic-config.json") || { echo "STOP: clone_root unresolvable" >&2; exit 1; }
    else
        CLONE_ROOT=$(dirname "$LAND_DIR"); echo "WARNING: no .epic-config.json; preserving into $CLONE_ROOT" >&2
    fi
    DEST=$(preserve_epic_state "$LAND_DIR" "$CLONE_ROOT" "at land of PR #<N> (\"<title>\").")
    rc=$?
    if [ "$rc" -eq 0 ] && [ -n "$(find "$DEST" -maxdepth 1 -type f -size +0c 2>/dev/null)" ]; then
        find "$LAND_DIR" -maxdepth 1 \( -name '.epic-status.json' -o -name '.epic-worklog.md' \) -delete
        find "$LAND_DIR/.claude" -maxdepth 1 -name 'ralph-loop.local.md' -delete 2>/dev/null
        gh pr comment <N> --body "🤖 EPIC worklog preserved to \`$DEST\`." \
          || echo "WARNING: preserved to $DEST but the PR comment failed; report the path" >&2
        echo "Preserved to $DEST"
    else
        echo "STOP: preservation did not verify (rc=$rc); originals left in place" >&2
    fi
fi
if PRIMARY=$(bip worktree primary 2>/dev/null); then
    echo "Landing from worktree $LAND_DIR (primary: $PRIMARY)"
    cd "$PRIMARY"
fi
```

The destination is always `<clone_root>/.preserved/`, never inside the clone. If this step printed STOP, resolve it before continuing. The PR comment is how a conductor tells a preserved land from a bypassed one.

`bip worktree primary` exits 0 and prints the primary clone path **only** when the current directory is a linked worktree.
In every other case (primary clone, non-bip checkout, non-git directory) it exits non-zero with no stdout — `$PRIMARY` remains empty and the `cd` is skipped.

If `$PRIMARY` was set, Steps 7 and 7.5 below run in the primary clone; otherwise they run in `$LAND_DIR` exactly as today.

### Step 7: Return to base branch and pull

```bash
git checkout <base>
git pull
```

Confirm the landed tree is the gated tree (valid because Step 6 required the head to be current with the base):

```bash
test "$(git rev-parse HEAD^{tree})" = "$(git rev-parse <gated-sha>^{tree})"
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

### Step 10: Confirm

Report: "Landed #42.
On `<base>`, up to date, worktree clean.
Branch `<branch>` deleted."
If any files were moved to `_ignore/`, list them.
If the primary clone was synced in Step 7.5, say so: "Primary clone `<path>` pulled."
If Step 8 removed a linked worktree, say so: "Worktree `<path>` removed."
If Step 7a preserved EPIC state, say so: "Worklog preserved to `<DEST>`, noted on the PR."
