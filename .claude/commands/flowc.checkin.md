# /flowc.checkin

Check in on recent activity across tracked repos. Shows issues, PRs, and comments that need your attention.

## Instructions

Run the check-in to fetch recent GitHub activity:

```bash
flowc checkin
```

This will:
1. Read repos from `sources.json`
2. Fetch issues, PRs, and comments updated since last check-in (stored in `.last-checkin.json`)
3. **Filter to items needing your action** (ball-in-my-court logic)
4. Display activity grouped by repo with GitHub refs (e.g., `matsengrp/repo#123`)
5. Check board sync status
6. Update the last check-in timestamp

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

- `flowc checkin --all` — Show all activity (disable ball-in-my-court filtering)
- `flowc checkin --since 2d` — Check activity from last 2 days instead of last check-in
- `flowc checkin --since 12h` — Check activity from last 12 hours
- `flowc checkin --repo matsengrp/dasm2-experiments` — Check single repo
- `flowc checkin --category code` — Check only repos in the "code" category
- `flowc checkin --summarize` — Add LLM-generated take-home summaries for each item (uses claude CLI)

## Review workflow

After checkin shows activity, spawn tmux windows for items that need review:

```bash
flowc spawn matsengrp/repo#123              # By reference
flowc spawn https://github.com/org/repo/pull/42   # By URL
flowc spawn matsengrp/repo#123 --prompt "Rebase and fix conflicts"  # Custom prompt
```

Each window:
- Opens in the correct local repo clone
- Launches Claude Code with context about the issue/PR
- Named by repo and number (e.g., `repo#123`)

Tmux window existence = item under review. Close the window when done.
