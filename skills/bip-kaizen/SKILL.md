---
name: bip-kaizen
description: Reflect on what just happened and propose a concrete improvement to skills, CLAUDE.md, docs, or workflow.
---

# /bip-kaizen

Reflect on friction from the current session and propose one concrete improvement.
`$ARGUMENTS` is an optional hint about what to improve; if empty, infer it from the conversation.

## Step 1: Diagnose

Name the root cause, not the symptom: "the agent couldn't find the database" is a symptom; "the database path isn't in AGENTS.md" is the cause.

## Step 2: Pick the target

| Target | When |
|--------|------|
| The current project's AGENTS.md or CLAUDE.md (edit whichever exists; a CLAUDE.md anywhere between the working directory and an AGENTS.md stops that AGENTS.md loading, so mixed trees symlink `CLAUDE.md -> AGENTS.md`) | The agent lacked a project fact |
| Global `~/.claude/CLAUDE.md` | The agent lacked a cross-project fact or preference |
| A bipartite skill, or bip code | A `/skill` or `bip` command gave wrong or incomplete guidance |
| Code or docs in the current repo | Help text or docs don't match behavior |
| A new skill | A multi-step workflow is done by hand again and again |
| A GitHub issue | The fix needs design discussion |

## Step 3: Find the rule you would collide with

Before writing anything, grep for where guidance on this already lives:

```bash
grep -rn '<key term>' ~/re/bipartite/skills/ ~/re/bipartite/*.md ~/.claude/CLAUDE.md AGENTS.md CLAUDE.md 2>/dev/null
```

Then say which one holds:

- **Supersedes `<file>` `<section>`**: the existing rule is wrong or too narrow. Edit it in place; two rules on one topic have no tiebreak.
- **Amends `<file>` `<section>`**: the rule is right but incomplete. Extend that passage.
- **No existing rule**: you grepped and found nothing. Say what you grepped, so a reviewer can check.

List every other file that restates the rule, and update each one or say why not. A fix that lands in one of two copies leaves the other looking authoritative and stale.

Adding without deleting needs an answer to "what does this supersede?" (`CONSTITUTION.md` Article VII). State the size of your change in words, since this corpus writes one paragraph per line:

```bash
git diff HEAD -- '*.md' | grep '^+' | grep -v '^+++' | wc -w
```

A rule nobody can cite an instance of is a candidate for deletion, not for shortening.

## Step 4: Propose

```
## Kaizen: <short title>
**What happened**: <one or two sentences>
**Root cause**: <why>
**Fix**: <what changes where, with the diff or new text>
**Supersedes / Amends**: <file and section, or "no existing rule; grepped <term>">
**Other restatements**: <each, and whether updated, or "none found">
**Size**: <words added / removed>
```

One improvement per run; name any others in a line.

## Step 5: Ask, then act

Stop and ask the user which to do:

1. **Apply now**: edit a target outside bipartite directly.
2. **Send to the `bipartite` session**: for anything in the bipartite repo, when `ListAgents` shows a session by that name. Send the diagnosis and the proposed edit; it places, lands, and reports back.
   With no such session, open a PR from a worktree of your own, since `~/re/bipartite` is shared by every session on the machine:
   ```bash
   git -C ~/re/bipartite fetch -q origin
   git -C ~/re/bipartite worktree add ../bipartite-wt-kaizen-<x> -b kaizen/<x> origin/main
   ```
   Commit a pointer and its target together, push, open the PR with `gh pr create`, and run `/bip-pr-review` on it.
3. **Write an issue**: draft `ISSUE-kaizen-<topic>.md` in the current repo and submit it with `/bip-issue-check`.
4. **Skip**: the user disagrees with the diagnosis.

## Guidelines

- Propose the smallest change that fixes the problem; don't refactor nearby text.
- Spend emphasis (⭐ ⛔ ⚠, ALL-CAPS bold) only if the passage outranks every marker already in the file; a marker every few lines marks nothing.
