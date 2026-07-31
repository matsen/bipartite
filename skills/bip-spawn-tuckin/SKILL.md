---
name: bip-spawn-tuckin
description: Persist a plain (non-EPIC) spawn session's state before context reset — commit and push work, sync the PR body — so /bip-spawn-resume can pick up cleanly.
---

# /bip-spawn-tuckin

Flush a `/bip-spawn` worker's state to durable storage before a context
reset or session end. Run this when context is getting long or before
stopping.

This is the write-side counterpart to `/bip-spawn-resume` (which reads
what this writes) and the single-slot, non-EPIC counterpart to
`/bip-epic-tuckin` (which persists an orchestrator's view across many
slots). EPIC workers already have their own persistence built into
`/bip-epic-spawn`'s STOPPING POINTS protocol (`.epic-status.json` +
`.epic-worklog.md` + issue-lead invocation) — if `.epic-status.json`
exists in this worktree/clone, this is an EPIC slot: stop and use that
protocol instead of this skill.

## Usage

```
/bip-spawn-tuckin
```

Run from inside the slot's worktree/clone, on its branch.

## Workflow

### Step 0: Confirm this isn't an EPIC slot

```bash
test -f .epic-status.json && echo "EPIC slot — use bip-epic-spawn's STOPPING POINTS protocol instead"
```

If it exists, stop here and point the user at that protocol instead.

### Step 1: Commit local work

```bash
git status --short
```

Plain spawn sessions have no worklog file — the commit log and PR are
the only durable record a fresh conversation can read. Don't leave
uncommitted changes behind:

- Coherent, working changes → commit them with a clear message
  describing what and why (this doubles as the "worklog entry" that
  `/bip-spawn-resume`'s `git log --oneline -10` will surface).
- Genuinely unfinished/broken state (mid-experiment, doesn't build) →
  either finish it enough to commit safely, or tell the user explicitly
  what's uncommitted and why, and let them decide. Don't silently
  commit broken code or silently discard it.

### Step 2: Push the branch

```bash
git push
```

(`git push -u origin <branch>` if this branch has never been pushed.)
A branch that only exists locally in a tmux pane doesn't survive the
pane closing or the host restarting — push it.

### Step 3: Sync the PR (if one exists)

```bash
gh pr view --json number,url,body,state 2>/dev/null
```

If a PR is open, make sure its body reflects where things actually
stand right now — apply `PROSE-DISCIPLINE.md` before writing anything.
Two cases:

- **Session reached a real stopping point (done, or blocked on the
  user)** — rewrite the body to read as current state (premise,
  results, interpretation), not as a log of how it got there. Same
  discipline as the quality-gate loop in `/bip-epic-spawn`: no growing
  "History" section.
- **Session is pausing mid-work, PR body would go stale if rewritten
  as "done"** — leave the body alone and instead add a PR comment
  describing what's done, what's left, and any open question, so
  `/bip-spawn-resume`'s "PR open" branch (which reads comments) picks
  it up. Don't describe unfinished work in body prose as if it were
  complete.

If no PR exists yet: only open one if the work is far enough along that
a draft PR is a better home for "where things stand" than a bare
branch. If it's a few commits into early exploration, leave it as a
branch — don't force a PR into existence just to have somewhere to
write status.

### Step 4: Report

```
## Tuckin Complete

- Branch: <name> (pushed)
- Uncommitted changes: none | committed as <sha>
- PR: #<N> body synced | #<N> comment added | none yet (branch only)

Safe to reset context.
```
