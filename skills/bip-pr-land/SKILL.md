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

⛔ **`--body` IS NOT OPTIONAL AND IS NOT COSMETIC. Omitting it silently changes which issues the merge closes.** Without `--body`, `gh` defaults the squash commit message to **every branch commit body concatenated**, and GitHub parses that text for closing keywords. Measured on `matsengrp/phyz` 2026-09-15, three merge commits:

| commit | how merged | body lines |
|---|---|---|
| `5dd7e2e5` | this skill (`--body "closes #2671"`) | **3** |
| `99e09aa5` | this skill (`--body "closes #2650"`) | **3** |
| `3af8d743` | a bare `gh pr merge --squash`, no `--body` | **103** |

Inside those 103 lines sat a branch commit whose entire purpose was *"stop closing the issue"* — and which therefore **quoted the line it had removed**, wrapped across a newline:

```
The issue-lead's evaluation found two defects: the PR body said "Closes
#2620" although criteria 1, 3, 4, 5, 7, and 9 remain unmet
```

`Closes` + newline + `#2620` parses. Quotation marks do nothing; there is no escaping syntax. **#2620 closed on merge against the explicit intent of the PR body, the author, and two reviewers.**

⭐ **THE CHECK EVERYONE RAN WAS THE PR BODY, AND IT PASSED.** Two independent readers ran `gh pr view --json body | grep -i closes` and both correctly got nothing, because the PR body genuinely had no `Closes` line — it had been deliberately removed. **The right question is not "does the PR body close an issue" but "what text will GitHub parse at merge time," and under a default squash those are different artifacts.**

➡ **So: always pass `--body` explicitly, and additionally enumerate every issue number the merge would close.** `--body` alone does not cover a wrong or extra number inside the body you wrote, nor anyone landing outside this skill. Run this **before** Step 6 and compare its output against the issues you intend to close:

```bash
BASE=$(gh pr view --json baseRefName -q .baseRefName)
git log --format='%B' "origin/$BASE"..HEAD \
  | tr '\n' ' ' \
  | grep -oiE '\b(close[sd]?|fix(e[sd])?|resolve[sd]?)\b[[:space:]]*#[0-9]+' \
  | grep -oE '[0-9]+' | sort -u
```

⛔ **THE `tr '\n' ' '` IS THE ENTIRE POINT AND IT IS NOT DECORATION. `grep` IS LINE-BASED, SO NO PATTERN — INCLUDING `[[:space:]]`, `\s`, OR `[[:space:]]*` — CAN MATCH ACROSS A NEWLINE IN ORDINARY `grep`.** The wrap is the mechanism, so a grep without the `tr` (or without `-Pz`) reads the two halves as unrelated lines and **reports clean on the real defect.**

⚠ **Verified, because the first draft of this very gate had that bug.** Run against `3af8d743`, the actual offending merge commit, a `grep -inE '…[[:space:]]*#?[0-9]+'` with no `tr` exits **1 with no output** — a clean bill of health on the commit the rule exists to catch. **A gate written from a correct diagnosis, by someone who had just finished writing up that diagnosis, still shipped unable to detect the instance.** Test a gate against the artifact that motivated it; a plausible pattern is not evidence.

**Pinned behaviour — re-verify these if you touch the command:**

| input | gate reports | intended | verdict |
|---|---|---|---|
| `3af8d743` (the defect) | `2620` | *nothing* | ⛔ fires correctly |
| `5dd7e2e5` (clean land) | `2671` | `2671` | ok |
| `99e09aa5` (clean land) | `2650` | `2650` | ok |
| `Closes #99` on one line | `99` | — | positive control |
| `"Closes` / `#77"` wrapped | `77` | — | positive control, the real shape |
| `see #123 for context` | *nothing* | — | negative control, no keyword |

Every number in the output that you do **not** intend to close is a defect. Fix it by rewording the commit body — break the keyword token, or move the number away from it — **not** by narrowing the gate. There is no escaping syntax and quotation marks do nothing.

⛔ **And do not hand-roll the merge to skip this.** A bare `gh pr merge` run outside this skill also skips Step 6a (EPIC worklog preservation), Step 9.5 (state cleanup), and the `🤖 EPIC worklog preserved` PR comment that `bip-conductor-poll`'s rescue rule reads. The 2026-09-15 instance lost nothing only because that slot's worklog had been preserved the previous night for an unrelated reason.

⭐ **The generalisation, which transfers well past `gh`: a defect's own documentation is executable context, so quoting a trigger re-arms it — prose is not inert.** Same class as the ⛔-citation rule in `bip-conductor` (a stale prohibition is more durable than a stale fact, because it suppresses the measurement that would expose it). Whenever a commit message, issue body, or skill file needs to *discuss* a syntax that something downstream acts on, break the token rather than quoting it.

### Step 6a: Detect worktree mode, and preserve EPIC state before anything can destroy it

**Run this whole step as a single Bash invocation, start to finish — do not split it across separate tool calls.** Cross-invocation persistence is **thread-type dependent**, not a fixed rule: in an agent/subagent thread, both shell variables *and* the working directory reset between separate Bash calls (measured directly this session); in a main interactive session, cwd persists but shell variables still do not. Either way, `$PRIMARY`, `$LAND_DIR`, and `$DEST` below must be set and consumed within this one script to be safe across both thread types. An earlier version of this step split the `cd "$PRIMARY"` out into its own later command, which silently broke worktree-mode landing (`git checkout` in Step 7 would run inside the about-to-be-removed worktree instead of the primary clone) — see issue #2216's follow-up finding.

Before pulling the base branch, check whether you are landing from a linked git worktree (created by `bip spawn` in worktree mode), and — before doing anything else, including the `cd` below — preserve `.epic-status.json`/`.epic-worklog.md` from `$LAND_DIR` if this is an EPIC-managed clone (issue #2216). This has to happen here, not at the old Step 9.5: in worktree mode, Step 8 below runs `bip worktree remove --force "$LAND_DIR"`, which deletes the whole worktree directory — gitignored files included — before Step 9.5 ever gets a chance to run. Doing it here, before any `cd` or removal, covers both modes uniformly: in clone mode `$LAND_DIR` never disappears, so this is equivalent to preserving right before the eventual `rm`; in worktree mode it's the only point that isn't already destroyed.

```bash
LAND_DIR=$(pwd -P)
# Resolve the helper WITHOUT a placeholder you have to fill in, and fail
# loudly if it is missing -- a bare `source` of a nonexistent file does not
# stop the script, it just leaves the functions undefined, and every guard
# below then silently takes its failure branch. See "If you cannot source it".
SPAWN_INTENT=""
for cand in ~/.claude/skills/lib/spawn-intent.sh \
            ~/re/bipartite/skills/lib/spawn-intent.sh; do
    [ -r "$cand" ] && { SPAWN_INTENT="$cand"; break; }
done
if [ -n "$SPAWN_INTENT" ]; then
    source "$SPAWN_INTENT"
else
    echo "PRESERVATION HELPER NOT FOUND -- read 'If you cannot source it' below before continuing. Do NOT search from / for it." >&2
fi
if PRIMARY=$(bip worktree primary 2>/dev/null); then
    echo "Landing from worktree $LAND_DIR (primary: $PRIMARY)"
fi
if [ -f "$LAND_DIR/.epic-status.json" ] && [ -f "$LAND_DIR/.epic-config.json" ]; then
    if CLONE_ROOT=$(resolve_clone_root "$LAND_DIR/.epic-config.json"); then
        DEST=$(preserve_epic_state "$LAND_DIR" "$CLONE_ROOT" \
            "at land of PR #<PR number from Step 2> (\"<PR title from Step 2>\").")
        rc=$?
        if [ "$rc" -eq 0 ]; then
            echo "Preserved worklog+status to $DEST"
            if gh pr comment <PR number from Step 2> --body "🤖 EPIC worklog preserved to \`$DEST\` (issue #2216)."; then
                echo "Posted preservation pointer to PR #<PR number from Step 2>"
            else
                echo "WARNING: preservation succeeded ($DEST) but the gh pr comment pointer failed to post -- note the path in Step 10's report so it isn't lost" >&2
            fi
        elif [ "$rc" -eq 2 ]; then
            echo "PRESERVATION FAILED: cp into .preserved did not succeed -- stop and investigate before continuing, do not let Step 8/9.5 delete the originals" >&2
        fi
    else
        echo "PRESERVATION FAILED: $LAND_DIR/.epic-config.json exists but its clone_root could not be resolved -- stop and investigate. If you hand-copy, the destination is the CLONE ROOT's .preserved/ (the parent of this clone), never a path inside this clone -- see 'If you cannot source it' below" >&2
    fi
fi
if [ -n "$PRIMARY" ]; then cd "$PRIMARY"; fi
```

**If you cannot source it: the destination is `<clone_root>/.preserved/`, NOT anywhere inside this clone — and do not go looking for the helper with an unbounded `find`.**

Both halves of that were measured on `matsengrp/phyz` 2026-09-12/13, on the same clone, twice. A worker could not resolve the old `"$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"` placeholder, ran **`find / -iname "spawn-intent.sh"`** — which on that host descends into a read-only sshfs mount with a documented 25-hour-hang incident behind it — then hand-rolled `mkdir -p` + `cp` and wrote to **`<clone_root>/<clone>/.preserved/`**. Three PRs landed that way. The files survived only because nothing had reused the clone yet; a `bip spawn` prep or a fresh clone would have taken them, and nothing looking in `.preserved/` would ever have found them. Two sibling clones had **43 and 10 files** sitting in the same wrong place from earlier instances of exactly this.

The failure is silent by construction: `source` of a missing file does not abort the script, so `resolve_clone_root` and `preserve_epic_state` are simply undefined, every `if` below takes its failure branch, and the step reports a handled error rather than a missing helper.

So if the loop above found nothing:

```bash
CLONE_ROOT=$(jq -r '.clone_root' "$LAND_DIR/.epic-config.json" | sed "s|^~|$HOME|")
ISSUE=$(jq -r '.issue' "$LAND_DIR/.epic-status.json")
CLONE=$(basename "$LAND_DIR")
DEST="$CLONE_ROOT/.preserved/$ISSUE-$(date -I)"
mkdir -p "$DEST"
cp "$LAND_DIR/.epic-status.json" "$DEST/i$ISSUE-$CLONE.status.json"
cp "$LAND_DIR/.epic-worklog.md"  "$DEST/i$ISSUE-$CLONE.worklog.md"
```

Note `$CLONE_ROOT/.preserved`, not `$LAND_DIR/.preserved`. If a sibling clone appears to have its own `.preserved/`, that is a previous instance of this same bug — do not copy it.

`preserve_epic_state` (from `skills/lib/spawn-intent.sh`) is the shared preserve-then-report pipeline all four preserve sites in this repo now use — extracted per issue #2216 after unchecked `cp`, wrong-order `resolve_clone_root`, and guard-tripping wording each independently recurred across the sites that used to hand-roll it. It produces the un-dotted, clone-namespaced filenames (`i<issue>-<clone>.worklog.md`/`.status.json`) matching this repo's existing `.preserved/` convention (23 of 25 prior entries use this form) rather than the dotted originals, which a plain `ls` or `*` glob would otherwise show as an empty directory. Its return code distinguishes "nothing to preserve" (1, silent) from "preservation failed" (2, must not proceed to delete) — check `$rc` immediately after the assignment, in the same command, since a later command would not see it.

The `gh pr comment` runs immediately, inside this same step, rather than being deferred to Step 9.5 — the PR was already merged in Step 6 above, so the PR number is available here, and posting now means the "committed pointer" `bip-conductor-poll`'s three-part rescue rule requires doesn't depend on a variable surviving through Steps 7/7.5/8/9 to reach Step 9.5 (it wouldn't: same cross-invocation problem as `$PRIMARY`, just further away).

`bip worktree primary` exits 0 and prints the primary clone path **only** when the current directory is a linked worktree; in every other case (primary clone, non-bip checkout, non-git directory) it exits non-zero with no stdout and `$PRIMARY` remains empty. **No `.epic-config.json` at all is not a failure** — it means this land isn't part of a multi-clone EPIC fleet, and there is nothing to preserve into. Every session on the machine picks up this skill the moment it merges (it's a symlink into `~/.claude/skills/bip-pr-land`), including plain non-EPIC repos that have never heard of `$CLONE_ROOT`. Those must see nothing here, not a scary-looking failure message — hence gating on `.epic-config.json`'s presence too, not just `.epic-status.json`'s. A repo that *has* `.epic-config.json` but still fails to resolve `clone_root` from it is a genuinely broken EPIC clone, and that case keeps the loud report.

`resolve_clone_root` (from `skills/lib/spawn-intent.sh`, already used by `bip-conductor-spawn` and `bip-conductor-poll`) fails loudly on stderr and returns non-zero if `.clone_root` is missing or unresolvable, rather than silently yielding an empty path — use it here instead of hand-rolling the same jq+tilde-expand logic a third time (that duplication is exactly what the helper was extracted to stop, per its own header comment). If either failure branch above fires, **do not proceed past it silently** — the whole point of this step is that a `cp` or `mkdir` failure must not read as success.

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

### Step 9.5: Clean up orchestration files

`.epic-status.json` and `.epic-worklog.md` were already preserved — and, if
preservation succeeded, the "committed pointer" `bip-conductor-poll`'s
three-part rescue rule requires was already posted via `gh pr comment` —
inside Step 6a (issue #2216; see that step for why both have to happen
there, before Step 8's worktree removal, rather than here). If Step 6a
reported a `PRESERVATION FAILED` line, **stop and resolve it before
deleting anything** — do not let this step destroy the only copy of a
file Step 6a couldn't back up.

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

### Step 10: Confirm

Report: "Landed #42.
On `<base>`, up to date, worktree clean.
Branch `<branch>` deleted."
If any files were moved to `_ignore/`, list them.
If the primary clone was synced in Step 7.5, say so: "Primary clone `<path>` pulled."
If Step 8 removed a linked worktree, say so: "Worktree `<path>` removed."
⛔ **A LANDING THAT BYPASSES THIS SKILL SILENTLY SKIPS PRESERVATION, AND THE
OBSERVABLE IS THE PR COMMENT.** `gh pr merge` run outside `/bip-pr-land` skips
Step 6a (preserve), Step 9.5 (delete), and the `🤖 EPIC worklog preserved`
comment in one move — **no error, no warning, no trace.** Measured 2026-09-14 on
`matsengrp/phyz`: a PR landed that way and left a **35,432-byte** worklog live
in a pooled clone, where the next spawn's prep would have deleted it with
nothing to show it had existed.

➡ **The check is the pointer comment, not the files.** After any merge:
`gh pr view <N> --json comments --jq '.comments[].body' | grep -c 'worklog preserved'`
— **1 means this skill ran, 0 means it did not.** That contrast is what
diagnosed the instance: the PR that used the skill had 1, the one that did not
had 0, and the second's state files were still sitting in the clone (which
Step 9.5 would have removed). ⚠ **Do not diagnose it from the presence or
absence of `.preserved/<issue>-*`** — absent is also what a preservation that
ran *to the wrong directory* looks like, and that failure mode is real
(`<clone_root>/<clone>/.preserved/` instead of `<clone_root>/.preserved/`; see
"If you cannot source it" above). The comment distinguishes *did not run* from
*ran and failed*; the directory does not.

⚠ A conductor-side worklog mirror (`/bip-conductor-poll`'s
`mirror_worklogs`) now runs every poll cycle as a floor under this, so a
bypassed landing no longer loses the worklog outright. **That is a backstop, not
a licence** — it covers `.epic-worklog.md` and `.epic-status.json` only, is
overwritten rather than archived, and does not post the pointer a reader six
weeks out needs.

If Step 6a preserved EPIC state, say so: "Worklog preserved to `<DEST>`, noted on the PR." (recall the exact path from what Step 6a printed earlier in this same session — it was not carried forward as a shell variable)
