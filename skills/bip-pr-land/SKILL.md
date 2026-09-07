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

If `bip spawn` created the working tree as a linked git worktree (because the user opted into the global `layout: { mode: worktree }` block in `~/.config/bip/config.yml`), this skill detects that in Step 6a and performs the base-branch pull from the primary clone, then removes the worktree in Step 8 before deleting the branch.
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

### Step 6a: Detect worktree mode, and preserve EPIC state before anything can destroy it

Before pulling the base branch, check whether you are landing from a linked git worktree (created by `bip spawn` in worktree mode):

```bash
LAND_DIR=$(pwd -P)
if PRIMARY=$(bip worktree primary 2>/dev/null); then
    echo "Landing from worktree $LAND_DIR (primary: $PRIMARY)"
fi
```

`bip worktree primary` exits 0 and prints the primary clone path **only** when the current directory is a linked worktree.
In every other case (primary clone, non-bip checkout, non-git directory) it exits non-zero with no stdout — `$PRIMARY` remains empty.

**Preserve `.epic-status.json`/`.epic-worklog.md` from `$LAND_DIR` right here, before doing anything else** (issue #2216). This has to happen in this exact spot, not later: in worktree mode, Step 8 below runs `bip worktree remove --force "$LAND_DIR"`, which deletes the whole worktree directory — gitignored files included — before Step 9.5 (the old, clone-mode-only cleanup point) ever gets a chance to run. Preserving after that point is too late; the files are already gone. Doing it here, immediately after computing `$LAND_DIR` and before any `cd` or removal, covers both modes uniformly: in clone mode `$LAND_DIR` never disappears, so this is equivalent to preserving right before the eventual `rm`; in worktree mode it's the only point that isn't already destroyed.

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
if [ -f "$LAND_DIR/.epic-status.json" ] && [ -f "$LAND_DIR/.epic-config.json" ]; then
    if CLONE_ROOT=$(resolve_clone_root "$LAND_DIR/.epic-config.json"); then
        ISSUE_N=$(jq -r '.issue // "unknown"' "$LAND_DIR/.epic-status.json" 2>/dev/null)
        DEST="$CLONE_ROOT/.preserved/$ISSUE_N-$(date -I)"
        mkdir -p "$DEST"
        if cp "$LAND_DIR/.epic-worklog.md" "$LAND_DIR/.epic-status.json" "$DEST/"; then
            {
                printf 'Preserved from %s at land of PR #%s ("%s"), %s.\n' \
                    "$LAND_DIR" "<PR number from Step 2>" "<PR title from Step 2>" "$(date -I)"
                echo "Named <issue>-<date>, not this repo's usual <issue>-<slug>: a date is trivially derivable and collision-free at this step, at the cost of a visibly different naming convention here."
            } > "$DEST/README.md"
            PRESERVED_DEST="$DEST"
            echo "Preserved worklog+status to $DEST"
        else
            echo "PRESERVATION FAILED: cp into $DEST did not succeed -- stop and investigate before continuing, do not let Step 8/9.5 delete the originals" >&2
        fi
    else
        echo "PRESERVATION FAILED: $LAND_DIR/.epic-config.json exists but its clone_root could not be resolved -- stop and investigate, or copy $LAND_DIR/.epic-worklog.md somewhere durable by hand (e.g. _ignore/$(date -I)-landing/) before continuing" >&2
    fi
fi
```

**No `.epic-config.json` at all is not a failure — it means this land isn't part of a multi-clone EPIC fleet, and there is nothing to preserve into.** Every session on the machine picks up this skill the moment it merges (it's a symlink into `~/.claude/skills/bip-pr-land`), including plain non-EPIC repos that have never heard of `$CLONE_ROOT`. Those must see nothing here, not a scary-looking failure message — hence gating on `.epic-config.json`'s presence too, not just `.epic-status.json`'s. A repo that *has* `.epic-config.json` but still fails to resolve `clone_root` from it is a genuinely broken EPIC clone, and that case keeps the loud report.

`resolve_clone_root` (from `skills/lib/spawn-intent.sh`, already used by `bip-conductor-spawn` and `bip-conductor-poll`) fails loudly on stderr and returns non-zero if `.clone_root` is missing or unresolvable, rather than silently yielding an empty path — use it here instead of hand-rolling the same jq+tilde-expand logic a third time (that duplication is exactly what the helper was extracted to stop, per its own header comment). If either failure branch above fires, **do not proceed past it silently** — the whole point of this step is that a `cp` or `mkdir` failure must not read as success.

Now perform the primary-clone `cd`, if applicable — as its own command, since `bip worktree remove` in Step 8 needs `$LAND_DIR` to still resolve correctly relative to wherever Step 8 runs from:

```bash
[ -n "$PRIMARY" ] && cd "$PRIMARY"
```

If `$PRIMARY` was set, Steps 7 and 7.5 below run in the primary clone; otherwise they run in `$LAND_DIR` exactly as today.

### Step 7: Return to base branch and pull

```bash
git checkout <base>
git pull
```

### Step 7.5: Sync the primary clone (clone-mode only)

**Skip this step if Step 6a already `cd`'d to the primary** — Step 7 has already pulled it.

If you landed from a scratch clone (EPIC worker, `bip spawn` without worktree mode, or any working copy that isn't the canonical one in `sources.yml`), the primary clone is now behind `origin/<base>`.
Pull it forward so the canonical checkout matches `main`.

1. Resolve the primary clone path the way `bip spawn` does (mirrors `flow.ResolveRepoPath`: `nexus_path` from `~/.config/bip/config.yml`, repo from `git remote get-url origin`, then `sources.yml` + `config.yml` paths).
   If `$(pwd -P)` already equals it, or the repo isn't listed, skip this step.

2. `git -C "$PRIMARY" pull --ff-only`.
   On failure, warn with the error and continue — the merge is already upstream, nothing is lost.
   Never stash.

3. Report what you did in Step 10.

### Step 8: Remove worktree (if applicable) and delete branch

If Step 6a found that you were landing from a linked worktree, remove the worktree **before** deleting its branch — git refuses to delete a branch that still has a worktree checked out on it:

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

### Step 9.5: Clean up orchestration files, and record the preserved pointer

`.epic-status.json` and `.epic-worklog.md` were already preserved in Step 6a
(issue #2216 — see that step for why the preservation has to happen there,
before Step 8's worktree removal, rather than here). If Step 6a reported a
`PRESERVATION FAILED` line, **stop and resolve it before deleting anything**
— do not let this step destroy the only copy of a file Step 6a couldn't
back up.

If `$LAND_DIR/.epic-status.json` is present, remove the now-redundant
originals as a command with no `$` in it at all (avoids Claude Code's
`$`+`rm` destructive-removal guard — see `bip-conductor-spawn`'s Step 2 for
the verified trigger condition). In clone mode you're still standing in
`$LAND_DIR`, so this is a plain relative-path `rm`; in worktree mode
`$LAND_DIR` no longer exists (Step 8 already removed it), so there is
nothing left here to remove and this is a no-op:

```bash
rm -f .epic-status.json .epic-worklog.md
```

If Step 6a's preservation block set `$PRESERVED_DEST`, land the "committed
pointer" its own three-part rescue rule requires (see `bip-conductor-poll`'s
"before returning a clone to the pool" section) — a chat-only report is not
enough, since nothing reads this session's transcript later:

```bash
gh pr comment <PR number from Step 2> --body "🤖 EPIC worklog preserved to \`$PRESERVED_DEST\` (issue #2216)."
```

### Step 10: Confirm

Report: "Landed #42.
On `<base>`, up to date, worktree clean.
Branch `<branch>` deleted."
If any files were moved to `_ignore/`, list them.
If the primary clone was synced in Step 7.5, say so: "Primary clone `<path>` pulled."
If Step 8 removed a linked worktree, say so: "Worktree `<path>` removed."
If Step 6a preserved EPIC state, say so: "Worklog preserved to `<PRESERVED_DEST>`, noted on the PR."
