# Continuation prompt

Every `/bip-*-tuckin` ends by writing a continuation prompt to one canonical path and echoing it.
This guide owns that path, the prompt's shape, and why the pattern exists; the tuckin skills reference it and supply only their own start directory, resume skill, and state file.

## Why this beats letting the window compact

Compaction summarizes the conversation: backward-looking, lossy, and still ephemeral — what matters ends up living only in a summary that degrades with each pass.

A tuckin does the opposite.
It **commits durable project knowledge where it belongs** — the onboarding doc, the manuscript's `%PROV`/`%TODO` markers, issue and PR bodies — and writes only a **short, forward-looking prompt** for the next session.
The next session then rebuilds from committed truth, not from a degraded transcript, and nothing that matters lives only in the window.

So the prompt is not a summary of what happened.
It is an orientation for what to do next, pointing at the durable state rather than restating it.

## Canonical location

`_ignore/CONTINUE.md`, in the working directory's repo root.
It is gitignored (`_ignore/` is; add `_ignore/` to `.gitignore` if a repo lacks it).
One fixed path, so the `SessionStart` hook below can load it and the next session always knows where to look.

## Shape

Under a page — a longer prompt does not get read.
It carries only:

- **Where to start**: `cd <dir>`.
- **The skill to run**: the cold-start or resume skill for this kind of session.
- **A pointer to the durable state**, one line — the file(s) the tuckin just committed, not their contents.
- **One to three things in flight**, each with its **falsifier inline**: the check that would show the item already done or moot.
  Not a general warning elsewhere in the prompt — a resuming session runs the top imperative before it reads a caveat below it, so an item that can expire must carry the check that kills it, on the same line.

```
Start in <dir>. Run /<skill>.
State: <durable file(s)>.
In flight:
- <item> — FIRST CHECK: <command or condition that proves it done/moot>
```

Never record open/merged/closed status as a fact: GitHub is the source of truth and a status line is wrong within hours.
Point at the check, not the answer.

## Auto-loading it

A `SessionStart` hook (matcher `clear|compact`) running `cat _ignore/CONTINUE.md 2>/dev/null || true` injects the prompt into the next session's context after a `/clear` or `/compact`, with no paste.
The hook lives in the user's `~/.claude/settings.json`.
