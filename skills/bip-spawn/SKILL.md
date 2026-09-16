---
name: bip-spawn
description: "Open a tmux window with an agent session (Claude Code or agy) for a GitHub issue or PR."
---

# /bip-spawn

Open a tmux window for GitHub issue or PR review.

## Instructions

**Just run `bip spawn` directly.**
Do not attempt to construct agent runner or `tmux` commands yourself — `bip spawn` handles all of that.

```bash
bip spawn org/repo#123
bip spawn https://github.com/org/repo/pull/42
bip spawn org/repo#123 --agent agy
bip spawn org/repo#123 --prompt "Rebase and fix conflicts"
```

## What it does

1. Parses the GitHub reference (org/repo#number or URL)
2. Finds the local clone path from config
3. Creates a tmux window named `repo#123`
4. Launches the configured agent runner (`claude` or `agy`) with issue/PR context

## Common modes

There are two distinct ways to use `bip spawn`.
**Pick deliberately**:

- **Review** (default, no `--prompt`): opens an agent session with the issue or PR context loaded and waits at an empty prompt.
  Use this when *you* will drive the next steps in the spawned window.
- **Handoff-to-work**: pass `--prompt "/bip-issue-work <N>"` (for issues) or `--prompt "/bip-pr-review"` (for PRs).
  The spawned agent starts the workflow autonomously — the right choice when the parent session is delegating implementation and won't be steering.

Example handoff:

bip spawn matsengrp/phyz#1453 --prompt "/bip-issue-work 1453"
bip spawn matsengrp/phyz#1453 --agent agy --prompt "/bip-issue-work 1453"

If you're spawning from inside a `/bip-pr-land` or similar handoff context, the handoff form is almost always what you want.

## Requirements

- Must be running inside tmux
- Local repo clone must exist (shows clone command if not)

## Options

- `--prompt "..."` — Custom prompt instead of default review prompt
- `--agent "..."` — Agent runner for spawned session: `claude` (default) or `agy` (can also be configured globally via `spawn_agent: agy` in `~/.config/bip/config.yml` or `$BIP_SPAWN_AGENT`)
- `--model "..."` — Model to pass to the agent launcher (default: unspecified)

## Worktree mode (opt-in)

By default, `bip spawn` opens its tmux window in the canonical clone of the repo — every spawn shares one working tree.
To get one linked git worktree per issue instead (so concurrent spawns don't share a branch checkout), add a `layout:` block to `~/.config/bip/config.yml`:

```yaml
layout:
  mode: worktree
  worktree:
    root: "~/re/{repo}-workers"   # {repo} = matsen/bipartite -> "bipartite"
    slot: "issue-{issue}"          # supports {issue}, {pr}, {branch}, {slug}
```

Per-repo overrides live in `sources.yml` (see `docs/guides/layout.md`).
With no `layout:` block, behavior is identical to before issue #149.

When worktree mode fires, `bip spawn` runs `git worktree add` from the canonical clone on a fresh branch named `<N>-<slug>`, and the spawned tmux window opens in the new worktree.
`bip pr-land` later detects the worktree, lands the PR from the primary clone, and removes the worktree.
