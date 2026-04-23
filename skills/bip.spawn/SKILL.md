---
name: bip.spawn
description: "Open a tmux window with a Claude Code session for a GitHub issue or PR."
---

# /bip.spawn

Open a tmux window for GitHub issue or PR review.

## Instructions

**Just run `bip spawn` directly.** Do not attempt to construct `claude` or `tmux` commands yourself — `bip spawn` handles all of that.

```bash
bip spawn org/repo#123
bip spawn https://github.com/org/repo/pull/42
bip spawn org/repo#123 --prompt "Rebase and fix conflicts"
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
