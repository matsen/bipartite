# Continuation prompt

Every `/bip-*-tuckin` ends by writing a continuation prompt to a role-keyed path under `_ignore/` and echoing it.
This guide owns that path, how the role is determined, the prompt's shape, and why the pattern exists; the tuckins and `/bip-continue` reference it.

## Why this beats letting the window compact

Compaction summarizes the conversation: backward-looking, lossy, and still ephemeral — what matters ends up living only in a summary that degrades with each pass.

A tuckin does the opposite.
It **commits durable project knowledge where it belongs** — the onboarding doc, the EPIC body, the manuscript's `%PROV`/`%TODO` markers, issue and PR bodies — and writes only a **short, forward-looking prompt** for the next session.
The next session then rebuilds from committed truth, not from a degraded transcript, and nothing that matters lives only in the window.

So the prompt is not a summary of what happened.
It is an orientation for what to do next, pointing at the durable state rather than restating it.

## Canonical location

`_ignore/CONTINUE-<role>.md`, where `<role>` is one of `ms`, `spawn`, `epic`, `conductor`, or `generic`, in the directory the next session will start in.
The role suffix is not optional: an epic session and a conductor session routinely share one repo checkout, so a single `_ignore/CONTINUE.md` would let the second tuckin silently overwrite the first and the resume would pick up the wrong role.
For an epic, key it by EPIC number as well — `_ignore/CONTINUE-epic-<N>.md` — because one checkout can host several epic sessions for different EPICs, and a bare `CONTINUE-epic.md` recreates the collision between them.
It is gitignored (`_ignore/` is; add `_ignore/` to `.gitignore` if a repo lacks it).

## Determining the role

`/bip-tuckin` and `/bip-continue` pick the role the same way, and never from what the session has been discussing — an epic session talks about slots and spawns all day, so activity misclassifies it:

1. **This session's own name** — the "This session is `X`" row of `ListAgents` — against the self-registered fleet files `$CLONE_ROOT/.epic-session` and `$CLONE_ROOT/.conductor-session`, each of which holds the name of the session in that role and is rewritten every cycle. A match is the role (`epic` or `conductor`).
2. Else the **marker file** in the working directory: `.ms-config.json` → `ms`; `.epic-status.json` → `spawn` (a worker slot).
3. Else **ask**, or fall back to `generic`.

## Shape

Under a page — a longer prompt does not get read.
Stamp it with the commit SHA and the time it was written, so the resumer can see how stale it is.
It carries only:

- **Where to start**: `cd <dir>`.
- **The skill to run**: the cold-start or resume skill for this role.
- **A pointer to the durable state**, one line — the file(s) the tuckin committed (for an epic, the EPIC body), not their contents.
- **One to three things in flight**, each with its **falsifier inline**: a `FIRST CHECK:` command or condition that would show the item already done or moot.
  Not a general warning elsewhere — a resuming session runs the top imperative before it reads a caveat below it, so the check must sit on the item's own line.
- **Anything still pending with the user** — an open question the session was waiting on; a resume that drops these loses them silently.

```
# written at <sha> <iso-time>
Start in <dir>. Run /<skill>.
State: <durable file(s) / EPIC body>.
In flight:
- <item> — FIRST CHECK: <command or condition that proves it done/moot>
Pending with the user:
- <open question>
```

The prompt is stale by default.
Never record open/merged/closed status or a present-tense fleet claim ("X is running") as a fact: the durable sources — the EPIC body, issue and PR comments, `gh` — supersede the prompt wherever they disagree, and every number that will be cited is re-derived, not trusted (the resume side of `EVIDENCE-DISCIPLINE.md`).
Point at the check, not the answer.

## Resuming from it

In the next session, after `/clear`, run `/bip-continue`.
It determines the role (above), reads the matching `_ignore/CONTINUE-<role>.md`, reports how stale the file is (its stamp against `git log -1` and the newest PR/issue activity), runs each in-flight item's inline check before acting, and hands off to the cold-start or resume skill the prompt names — which owns the role-specific work (re-registering `.epic-session`/`.conductor-session`, live-run and pooled-slot handling, surfacing standing traps).

`/bip-tuckin` is the matching entry point on the way out: it delegates to the right `/bip-*-tuckin` (or a generic fallback), each of which writes this prompt.
