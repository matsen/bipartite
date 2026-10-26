---
name: bip-cycle
description: Tuck in, then clear this session and resume it with /bip-continue, hands-free (in tmux, or in a `claude --bg` session). The one-command form of /bip-tuckin → /clear → /bip-continue.
allowed-tools: Skill, Bash
---

# /bip-cycle

Run `/bip-tuckin` with the Skill tool, passing `$ARGUMENTS` through (a role name, if given).

A role whose cold-start skill says it keeps no state between rounds (`/bip-librarian`) skips the tuckin: run `<this skill's base directory>/self-cycle /bip-<role>` instead, and it types that skill in place of `/bip-continue`.

When it has finished and the continuation prompt is written, make this the turn's last tool call:

```bash
<this skill's base directory>/self-cycle
```

It types `/clear` into this session's tmux pane, or, for a `claude --bg` session, into a `claude attach` client in a hidden tmux session. That input is queued until the turn ends. Once the fresh session's transcript appears, it types `/bip-continue`. Then end the turn without starting further work.

Skip the cycle, and say why, if any of these holds: the tuckin reported unpushed work, the script exits non-zero (it does when the session is neither in a tmux pane nor a background job), a background agent or task you are waiting on is still running (cycling strands its result), or you hold judgment a note can only point at, such as approved work not yet done whose reasoning lives in this conversation (say what; do it or brief a helper fully, then cycle — a `%TODO` or continuation line is a pointer, not a capture).
Waiting on the user is not on that list (user, 2026-10-17). A status report is already in the EPIC body or a PR, and the tuckin carries forward whatever is pending with them. If they must read or answer something, write it where it lasts (an issue or PR comment, the EPIC body), then send the `staff` session the one-line ask, that link, and who acts on the answer, and cycle. Staff turns away routine status.

`skills/lib/cycle-nudge.sh`, a Stop hook in `~/.claude/settings.json`, asks for a cycle, never while a ralph loop is active, once a session's context passes 300k tokens (500k in a worker slot, `BIP_CYCLE_TOKENS` / `BIP_CYCLE_WORKER_TOKENS`), and again per 100k after that; when to cycle is the agent's call. Set `"idleCompaction": false` in the same file: Claude Code otherwise compacts any 200k+ session idle about 54 minutes, below the nudge's threshold and with no tuckin first.
