# Where to Start

Bipartite can be adopted in three steps. Each step adds sessions and tooling to the one before. Start at the first, and move on when the test at the end of each tier says you are ready.

| Tier | What you run | Needs |
|---|---|---|
| 1. One session | Claude Code, one working session at a time | [skills-starter](https://github.com/matsengrp/skills-starter) |
| 2. Named sessions | Several named sessions in terminal tabs or VS Code terminals, which message each other | bipartite (`make install`) |
| 3. Fleet | A pool of repo clones, with a worker session spawned into a tmux window for each issue, coordinated by long-lived sessions | bipartite and tmux |

## Tier 1: one session

If you don't have Claude Code yet, [install it and log in](https://code.claude.com/docs/en/setup) first. Then use the [skills-starter](https://github.com/matsengrp/skills-starter) template. Its install script adds three skills and turns off Claude Code's auto memory, which otherwise saves notes nobody reviews:

- `/kaizen` turns friction into a committed fix: a line in a project's `AGENTS.md`, a line in your `CLAUDE.md`, or a new skill.
- `/tuckin` and `/continue` carry a session across `/clear`.

The idea to take away from this tier: Claude learns by changing files you can read and delete, not by accumulating memories.

The starter's README also suggests a second session, named `skills`, that lands the fixes `/kaizen` sends it. It is recommended: it keeps your skills repo in one session's hands, so fixes from different sessions don't collide. Without it, `/kaizen` edits your skills repo directly.

You do not need bipartite for tier 1. Move on when you find yourself with two or more Claude sessions open on one project and copying text between them.

## Tier 2: named sessions

Install bipartite ([Getting Started](getting-started.md)). Then run a few sessions side by side, each in its own terminal, and give each a name for its job: `claude -n <name>` at start, or `/rename <name>` later. A session lists the others by name and sends them messages with Claude Code's `SendMessage` tool. This works in any terminal, including VS Code's. Start every session in the same permission mode: a session holds a message from a session in a different mode until you approve it there, so the message looks lost.

A typical set for one project:

- A manuscript session (`/bip-ms`) in the paper repo. It holds the project's knowledge, reads new results as they arrive, and drafts an issue when you ask for one.
- One session per issue you are working on (`/bip-issue-work`). Give each its own copy of the code repo (a separate clone, or a git worktree) so their branches don't collide. It finishes with `/bip-pr-review` and `/bip-pr-land`.
- `/bip-lit` for the paper library, from any session.

Use `/bip-tuckin` and `/bip-continue` in place of the starter's pair; they recognize the manuscript and issue sessions and save each one's state in the right place. `/bip-helper` starts a background session (`claude --bg`) for side work, such as a long test run, without tying up a terminal.

Move on when starting, tracking and unsticking issue sessions by hand takes more of your time than reading their results.

## Tier 3: fleet

The fleet adds long-lived sessions that run the work for you. [Agent Roles](roles.md) describes each:

- An epic session (`/bip-epic`) for each open research question. Its EPIC is a GitHub issue that tracks that question and links the issues working on it. The epic session keeps the EPIC current and drafts the next issues.
- A conductor (`/bip-conductor`). It owns a pool of repo clones, and for each issue it starts a worker session in a tmux window in a free clone.
- Staff (`/bip-staff`). Sessions send it the questions they can't settle themselves. It answers what it can and notifies you only for decisions that are yours.

Tier 3 needs tmux: spawned workers, the review windows `/bip-inbox` opens for PRs awaiting your review, and the skills that rebuild the fleet after a host reboot all run through it. The [Issue Lifecycle](issue-lifecycle.md) guide gives the order of the issue and PR skills.
