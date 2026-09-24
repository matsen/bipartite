---
name: bip-continue
description: Resume from the continuation prompt a tuckin left — pick the role-keyed _ignore/CONTINUE-<role>.md, report its staleness, run each in-flight item's inline check before acting, and hand off to the cold-start skill it names. Run in a fresh session after /clear; pairs with /bip-tuckin.
allowed-tools: Skill, Bash, Read
---

# /bip-continue

Pick up where the last session's `/bip-tuckin` left off, from the role-keyed prompt it wrote (`docs/guides/continuation-prompt.md`).

## Steps

1. **Determine the role** (per the guide: `ListAgents` name against `$CLONE_ROOT/.epic-session` / `.conductor-session`, then the marker file) and read `_ignore/CONTINUE-<role>.md` — for an epic, `_ignore/CONTINUE-epic-<N>.md`, keyed by EPIC number.
   If several `CONTINUE-*.md` exist and the role or EPIC is unknown, list them and ask.
   If none exists, say so and stop; if a marker file makes the type obvious, offer the matching cold-start instead (`/bip-ms`, `/bip-spawn-resume`, `/bip-conductor`, or `/bip-epic`).
2. **Report staleness**: the file's written-at stamp against `git log -1` and the newest PR/issue activity. If the world moved since it was written, the checks below matter more.
3. **Follow the prompt**: `cd` where it says, run the cold-start or resume skill it names — that skill re-registers `.epic-session`/`.conductor-session` and handles the role-specific setup — and read the durable state it points at.
4. **Before acting on any in-flight item, run its inline `FIRST CHECK`.** Treat every present-tense claim and every number in the file as stale until a command re-derives it; the EPIC body and issue/PR comments supersede the file where they disagree. Drop what the checks retire.
5. **Carry forward anything pending with the user.** Then state where things stand in a sentence or two, and continue.

Leave `_ignore/CONTINUE-<role>.md` in place; the next `/bip-tuckin` overwrites it.
