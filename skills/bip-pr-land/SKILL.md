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

If `bip spawn` made the working tree as a linked worktree (`layout: { mode: worktree }` in `~/.config/bip/config.yml`), Step 7a detects it and pulls from the primary clone, and Step 8 removes the worktree before deleting the branch. In clone mode the detection is a no-op.

## Where to stand

This skill moves `HEAD` (Steps 7 and 8) and preserves the state files of its directory (Step 7a). Never run it in a clone another live session works in. To land a PR checked out in another session's clone, don't run this skill: run Step 6's guard from any checkout, against the PR head, so nothing moves:

```bash
git fetch -q origin "+pull/<N>/head:refs/pr/<N>" <base> \
  && sha=$(git rev-parse refs/pr/<N>) \
  && test "$sha" = "$(gh pr view <N> --json headRefOid -q .headRefOid)" \
  && git merge-base --is-ancestor origin/<base> "$sha" \
  && gh pr merge <N> --squash --match-head-commit "$sha" --body "closes #<issue>"
```

If the ancestor test fails, by default rebase `refs/pr/<N>` onto `origin/<base>` in a scratch worktree, `git push --force-with-lease origin HEAD:<headRefName>`, run the repo's gate on the rebased head (CI per Step 5.5; with no CI, the local gate its CLAUDE.md names), and re-run the guard. You may merge a stale head without the rebase only when you can say why it is safe (e.g. its files are disjoint from what landed since); say so in a PR comment.

Do this only for a worker that has ended (`stop_reason: awaiting-human-merge`, session idle). A live worker holding a landing delegation lands its own PR. If the clone is a fleet slot, reclaim it through `/bip-conductor`, which runs the issue-lead's terminal ceremony before preserving (Step 7a here would delete the status file the ceremony keys on).

Check for co-tenancy:

```sh
for p in $(pgrep -x claude); do readlink /proc/$p/cwd; done | /usr/bin/grep -cE "^$PWD(/|$)"
```

More than 1 means co-tenancy. The count is a floor: it misses a session that `cd`s into this clone inside its Bash calls. To inspect a PR read-only, prefer `git show <sha>:<path>` with `gh pr view <N> --json headRefOid`; `git fetch origin pull/N/head:<branch>` silently leaves an existing local branch at its old SHA (use `-f`).

## Workflow

### Step 1: Check for uncommitted work

```bash
git status --porcelain
git diff --stat
```

Resolve every dirty or untracked file; never `git stash`. Read enough of each to know what it is, then:
- Belongs to this PR: commit it.
- Unrelated: move it to `_ignore/$(date -I)-landing/` (mkdir -p first) and tell the user.
- Unclear: show the user the diff and ask.

### Step 2: Identify the PR

```bash
# Get current branch
BRANCH=$(git branch --show-current)

# Find the PR for this branch
gh pr view "$BRANCH" --json number,title,state,baseRefName
```

Abort if there is no PR or it is not open. Save `baseRefName` as `<base>`.

### Step 3: Log and proceed

Print the summary line and continue without waiting for confirmation:

```
Landing: #42 "Add feature X" (branch: my-feature → main)
```

### Step 4: Update base branch and rebase

```bash
git fetch origin
git rebase origin/<base>
```

On conflicts, stop and report. Do not force-push or auto-resolve.

### Step 5: Force-push rebased branch

```bash
git push --force-with-lease
```

### Step 5.5: Wait for CI to pass

```bash
gh pr checks "$BRANCH" --json name,state,conclusion
```

- Empty array and no `.github/workflows/`: proceed.
  Empty array in a repo with workflows means CI hasn't registered yet: wait (`until [ "$(gh pr checks "$BRANCH" --json name | jq length)" -gt 0 ]; do sleep 5; done`), then continue below.
- Checks present: block with `gh pr checks "$BRANCH" --watch --fail-fast` until all required checks conclude `SUCCESS`, `NEUTRAL` or `SKIPPED`.
  `SKIPPED` passes only if a check that builds or tests concluded `SUCCESS` at this head: a gate job can succeed while skipping the tests it gates. If so, rebase and wait for the tests.
- Any check fails: do not merge. Report the failing check (`gh pr view --web`) and stop.

### Step 6: Squash merge via gh

Ask GitHub which issues the PR will close:

```bash
gh pr view <N> --json closingIssuesReferences --jq '.closingIssuesReferences[].number'
```

Empty is a defect if the PR has an issue. The field is eventually consistent, so re-read before concluding a body edit did not take.

Keep the guard inside the merge command so the base cannot move between check and merge. Always pass `--body`: without it `gh` concatenates every commit body into the squash message, and GitHub closes any issue a closing keyword there names, negated or quoted or not.

```bash
test "$(git rev-parse HEAD)" = "$(gh pr view <N> --json headRefOid -q .headRefOid)" \
  && git fetch -q origin <base> \
  && git merge-base --is-ancestor origin/<base> HEAD \
  && gh pr merge --squash --match-head-commit "$(git rev-parse HEAD)" --body "closes #<issue>"   # or --body "" if it closes nothing
```

If the guard fails, go back to Step 4. A moved base voids an approval.

### Step 7a: Preserve EPIC state, then detect worktree mode

Run this as one Bash call (shell variables do not survive between calls, and the delete must depend on the verified preserve), before Step 8.

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

The destination is `<clone_root>/.preserved/`, never inside the clone. Resolve any STOP before continuing. The PR comment is how a conductor tells a preserved land from a bypassed one.

`bip worktree primary` prints the primary clone path only in a linked worktree; elsewhere it exits non-zero with no output. If `$PRIMARY` is set, Steps 7 and 7.5 run in the primary clone, else in `$LAND_DIR`.

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

Skip if Step 7a already `cd`'d to the primary.

If you landed from a scratch clone (EPIC worker, `bip spawn` in clone mode, or any copy that isn't the canonical one in `sources.yml`), pull the primary forward:

1. Resolve the primary path as `flow.ResolveRepoPath` does: `nexus_path` from `~/.config/bip/config.yml`, repo from `git remote get-url origin`, then `sources.yml` + `config.yml` paths. Skip if `$(pwd -P)` already equals it or the repo isn't listed.
2. `git -C "$PRIMARY" pull --ff-only`. On failure, warn and continue; never stash.

### Step 8: Remove worktree (if applicable) and delete branch

Remove a linked worktree before deleting its branch; git refuses to delete a branch checked out in a worktree:

```bash
if [ -n "$PRIMARY" ] && [ -n "$LAND_DIR" ] && [ "$LAND_DIR" != "$PRIMARY" ]; then
    bip worktree remove "$LAND_DIR"
fi
```

`bip worktree remove` defaults to `--force`, which the squash-merge's unreachable commits require. Then:

```bash
git branch -d <branch>
```

`gh pr merge` normally deletes the remote branch. If not: `git push origin --delete <branch>`

### Step 9: Ensure clean main

```bash
git status --porcelain
```

Move any leftover files to `_ignore/$(date -I)-landing/` and report them. End with a clean `git status` on main.

### Step 10: Confirm

Report: "Landed #42.
On `<base>`, up to date, worktree clean.
Branch `<branch>` deleted."
Also list, when they happened: files moved to `_ignore/`; "Primary clone `<path>` pulled." (Step 7.5); "Worktree `<path>` removed." (Step 8); "Worklog preserved to `<DEST>`, noted on the PR." (Step 7a).
