---
name: bip-tuckin
description: Universal tuckin dispatcher — persist session state before a context reset by delegating to the right /bip-*-tuckin (ms, spawn, epic, conductor), or a generic fallback when the session is none of those. Run before /clear; pairs with /bip-continue.
allowed-tools: Skill, Bash, Read, Write, Edit
---

# /bip-tuckin

One entry point for persisting session state before `/clear`.
It determines the session's role, runs the matching tuckin, and that skill writes the role-keyed continuation prompt (`docs/guides/continuation-prompt.md`) which `/bip-continue` reads in the next session.

## Choosing the tuckin

If `$ARGUMENTS` names a role — `ms`, `spawn`, `epic`, `conductor`, or `generic` — use it.

Otherwise determine the role per `docs/guides/continuation-prompt.md` ("Determining the role"): this session's `ListAgents` name against `$CLONE_ROOT/.epic-session` / `.conductor-session`, then the marker file (`.ms-config.json` → `ms`, `.epic-status.json` → `spawn`), then ask.
Never infer the role from what the session has been discussing.

Map the role to its skill and invoke it with the Skill tool — do not reimplement it:
`ms` → `/bip-ms-tuckin`, `spawn` → `/bip-spawn-tuckin`, `epic` → `/bip-epic-tuckin`, `conductor` → `/bip-conductor-tuckin`.

## Generic fallback

For a session that is none of the four — an ad-hoc or adjudicator session — there is no specialised tuckin, so do the minimum that makes a reset safe:

1. Commit and push anything this session owns; branch first if on `main`, and leave the pooled-slot branches a conductor manages alone. Report what is unpushed rather than forcing it.
2. Write `_ignore/CONTINUE-generic.md` per the guide.
3. Report what was committed and that the prompt is written; then it is safe to reset.
