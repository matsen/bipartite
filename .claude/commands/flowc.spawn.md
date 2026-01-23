# /spawn

Open a tmux window for GitHub issue or PR review.

## Instructions

```bash
flowc spawn org/repo#123
flowc spawn https://github.com/org/repo/pull/42
flowc spawn org/repo#123 --prompt "Rebase and fix conflicts"
```

## What it does

1. Parses the GitHub reference (org/repo#number or URL)
2. Finds the local clone path from config
3. Creates a tmux window named `repo#123`
4. Launches Claude Code with issue/PR context

## Requirements

- Must be running inside tmux
- Local repo clone must exist (shows clone command if not)

## Options

- `--prompt "..."` — Custom prompt instead of default review prompt
