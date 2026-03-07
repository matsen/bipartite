---
name: bip.epic.poll
description: Quick poll of GitHub activity and clone status since last check
---

# /bip.epic.poll

Lightweight mid-session update. Checks what changed on GitHub and in
active clones since last check. Use this instead of `/bip.epic` when
you already have context established.

## What to check

### 1. Recently merged PRs

```bash
gh pr list --search "is:pr is:merged sort:updated-desc" --limit 5 --json number,title,mergedAt
```

For each new merge: read the PR body (`gh pr view <N> --json body`),
note key results, check if it closes an issue.

### 2. Open PRs

```bash
gh pr list --json number,title,headRefName,state
```

Note any new PRs or CI status changes.

### 3. New issues

```bash
gh issue list --search "sort:created-desc" --limit 5 --json number,title,state,createdAt
```

### 4. Issue comments

Check comments on active issues (especially ones with running clones):

```bash
gh api repos/{owner}/{repo}/issues/{number}/comments --jq '.[-1].body' | head -40
```

### 5. Clone status

Read `clone_root` and `clone_names` from `.epic-config.json`:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
for name in $(jq -r '.clone_names[]' .epic-config.json); do
  [ -f "$CLONE_ROOT/$name/.epic-status.json" ] && echo "=== $name ===" && cat "$CLONE_ROOT/$name/.epic-status.json"
done
```

Also check tmux: `tmux list-windows -F "#W"`

For active clones, check recent commits:
```bash
git -C <clone> log --oneline main..HEAD | head -5
```

### 6. Tmux output (if interesting)

For clones that seem to have finished or are blocked:
```bash
tmux capture-pane -t <clone-name> -p | tail -20
```

## After polling

### Focus on what matters

**Lead with unblocked issues** — issues that are ready to work on but
not assigned to any clone. This is the most actionable information.

**Only report active clones** — clones with a tmux window that are
actually doing something. Don't list completed or idle clones; that's
noise. Completed clones can be mentioned briefly ("fir completed i374")
but don't need a table row.

**Mention recent merges** only if they unblock something or change
the plan.

### Output structure

1. **Unblocked issues**: Issues ready for work, not assigned to a clone.
   Cross-reference with EPIC dashboards to find next items.

2. **Active work**: Clones with tmux windows that are mid-task. One line
   each: clone, issue, what they're doing.

3. **Recently landed** (brief): PRs merged since last poll, only if
   noteworthy.

4. **Propose spawns**: If unblocked issues and idle clones exist, propose
   which to spawn. Wait for confirmation.

### Housekeeping (do silently, don't report unless problems)

- Update EPIC bodies if merges changed status
- Update MEMORY.md only for orchestrator-level decisions/patterns

## Conventions

Same as `/bip.epic`: `iN`/`pN` prefixes, full URLs on first mention,
clone-name tmux windows.
