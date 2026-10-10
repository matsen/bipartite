---
name: bip-cycle
description: Tuck in, then clear this session and resume it with /bip-continue, hands-free (tmux only). The one-command form of /bip-tuckin → /clear → /bip-continue.
allowed-tools: Skill, Bash
---

# /bip-cycle

Run `/bip-tuckin` with the Skill tool, passing `$ARGUMENTS` through (a role name, if given).

When it has finished and the continuation prompt is written, make this the turn's last tool call:

```bash
<this skill's base directory>/self-cycle
```

It types `/clear` into this session's tmux pane. That input is queued until the turn ends. Once the fresh session's transcript appears, it types `/bip-continue`. Then end the turn without starting further work.

Skip the cycle, and say why, if any of these holds: the tuckin reported unpushed work, something pending with the user needs an answer before a reset, the script exits non-zero (it does outside tmux), or you hold key context a tuckin can't capture yet (say what, and cycle once it is written down or resolved).

`skills/lib/cycle-nudge.sh`, a Stop hook in `~/.claude/settings.json`, asks for a cycle once a session's context passes 300k tokens (500k in a worker slot, `BIP_CYCLE_TOKENS` / `BIP_CYCLE_WORKER_TOKENS`), and again per 100k after that; when to cycle is the agent's call.
