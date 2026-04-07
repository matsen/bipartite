---
name: bip.pr.check
description: Quick PR readiness check — clean worktree, good description, squashed body. Run before /bip.pr.review.
---

# /bip.pr.check

Quick sanity check before running the heavier `/bip.pr.review`. Catches common issues that waste review cycles.

## Usage

```
/bip.pr.check
```

## Workflow

### Step 1: Verify branch state

Check that we're on a feature branch (not main/master):

```bash
git branch --show-current
```

If on main/master, warn and stop.

### Step 2: Check for uncommitted changes

```bash
git status --short
```

If there are uncommitted changes:
- List them clearly
- Ask the user: "There are uncommitted changes. Commit these before proceeding?"
- **Do not proceed** until the worktree is clean or the user explicitly says to continue

### Step 3: Fetch and rebase on base branch

```bash
git fetch origin
BASE=$(gh pr view --json baseRefName -q .baseRefName 2>/dev/null || echo "main")
git rebase "origin/$BASE"
```

If rebase has conflicts, stop and report — do not force-resolve.

If the rebase moved commits, push immediately:
```bash
git push --force-with-lease
```

### Step 4: Check PR exists

```bash
gh pr view --json number,title,body,state,isDraft 2>/dev/null
```

If no PR exists:
- Tell the user and ask if they want to create one now
- Stop here if no PR

### Step 5: Evaluate PR title

Check the PR title for quality:
- Is it descriptive (not just "WIP" or a branch name)?
- Is it under ~70 characters?
- Does it describe the *what* not the *how*?

Flag any issues and suggest improvements.

### Step 6: Evaluate PR body (the critical check)

Fetch the PR body and evaluate whether it reads as a **clean summary** or as **historical commit noise**.

**Signs of a bad (historical/unsquashed) body:**
- Starts with `* commit message` or `- commit message` bullet lists
- Contains a sequence of past-tense actions that read like a git log
- Has "Co-Authored-By" lines scattered through the body
- Contains multiple "fix typo", "address review", "WIP" entries
- Is auto-generated from concatenated commit messages (GitHub's default for squash merge)
- Is empty or just whitespace

**Signs of a good (squashed/summary) body:**
- Has a `## Summary` or similar section with 1-3 bullet points explaining *what* and *why*
- Reads as a coherent description a reviewer can understand
- Has a test plan or notes section if appropriate
- Is concise but informative

If the body looks like historical commit noise or is empty:
1. Read the actual diff to understand the changes: `git diff origin/$(gh pr view --json baseRefName -q .baseRefName)...HEAD`
2. Draft a clean replacement body in this format:
   ```
   ## Summary
   - [1-3 bullet points describing what changed and why]

   ## Test plan
   - [How to verify the changes work]

   🤖 Generated with [Claude Code](https://claude.com/claude-code)
   ```
3. Update the PR body directly:
```bash
gh pr edit <number> --body "$(cat <<'EOF'
<new body>
EOF
)"
```

### Step 7: Check draft status

If the PR is marked as draft:
- Ask: "PR is currently a draft. Ready to mark as ready for review?"
- If yes: `gh pr ready`

### Step 8: Summary

Print a compact checklist:

```
## PR Check

- [x] On feature branch: `branch-name`
- [x] Worktree clean
- [x] PR exists: #123
- [x] Title: descriptive and concise
- [x] Body: clean summary (not commit history)
- [x] All commits pushed
- [ ] Draft status: still draft — mark ready when done

→ Ready for /bip.pr.review
```

Or flag what needs attention before proceeding.

## Notes

- This is intentionally lightweight — no code review, no tests, no linting
- Run `/bip.pr.review` after this passes for the full quality sweep
- The body check is the most valuable part: it prevents the common mistake of merging with GitHub's default concatenated-commits body
