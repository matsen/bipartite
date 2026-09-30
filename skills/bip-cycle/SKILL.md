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

Skip the cycle, and say why, if any of these holds: the tuckin reported unpushed work, something pending with the user needs an answer before a reset, or the script exits non-zero (it does outside tmux).
