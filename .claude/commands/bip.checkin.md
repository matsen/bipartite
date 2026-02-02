# /bip.checkin

Check in on recent activity across tracked repos. Shows issues, PRs, and comments that need your attention.

## Instructions

Run the check-in to fetch recent GitHub activity:

```bash
bip checkin
```

This will:
1. Read repos from `sources.json`
2. Fetch issues, PRs, and comments updated since last check-in
3. **Filter to items needing your action** (ball-in-my-court logic)
4. Display activity grouped by repo with GitHub refs (e.g., `matsengrp/repo#123`)
5. Check board sync status

## Ball-in-my-court filtering (default)

By default, checkin only shows items where you need to act:

| Scenario | Shown? | Reason |
|----------|--------|--------|
| Their issue/PR, no comments | Yes | Need to review |
| Their issue/PR, they commented last | Yes | They pinged again |
| Their issue/PR, you commented last | No | Waiting for their reply |
| Your issue/PR, no comments | No | Waiting for feedback |
| Your issue/PR, they commented last | Yes | They replied |
| Your issue/PR, you commented last | No | Waiting for their reply |

Use `--all` to see everything (original behavior).

## Options

- `bip checkin --all` — Show all activity (disable ball-in-my-court filtering)
- `bip checkin --since 2d` — Check activity from last 2 days instead of last check-in
- `bip checkin --since 12h` — Check activity from last 12 hours
- `bip checkin --repo matsengrp/dasm2-experiments` — Check single repo
- `bip checkin --category code` — Check only repos in the "code" category
- `bip checkin --summarize` — Add LLM-generated take-home summaries for each item (uses claude CLI)

## Review workflow

After checkin shows activity, spawn tmux windows for items that need review:

```bash
bip spawn matsengrp/repo#123              # By reference
bip spawn https://github.com/org/repo/pull/42   # By URL
bip spawn matsengrp/repo#123 --prompt "Rebase and fix conflicts"  # Custom prompt
```

Each window:
- Opens in the correct local repo clone
- Launches Claude Code with context about the issue/PR
- Named by repo and number (e.g., `repo#123`)

Tmux window existence = item under review. Close the window when done.

## Board triage (optional)

After presenting checkin results, ask the user:

> "Would you like to review board triage? I can check if any issues from this checkin should be added to project boards."

If they agree, then:

1. **Get current board state**:
   ```bash
   bip board list --json
   ```

2. **Compare with checkin issues**: Look for issues that:
   - Are newly opened (not just updated)
   - Have active discussion suggesting work is planned
   - Are assigned or have milestone set
   - Are NOT already on a board

3. **Propose additions**: For each candidate, suggest:
   ```
   bip board add <repo>#<number>    # <one-line reason>
   ```

Skip issues that are:
- Already on a board
- Stale (no recent activity beyond the update that triggered checkin)
- Pure questions/discussions without actionable work
- Already closed

## Output guidelines

When summarizing checkin results:
- Do NOT include tmux window-switching advice (users have custom prefix keys)
- Include full GitHub URLs for each item so they're clickable
