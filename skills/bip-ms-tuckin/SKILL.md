---
name: bip-ms-tuckin
description: Persist manuscript session state before context reset
---

# /bip-ms-tuckin

Flush manuscript session state to durable storage before a context reset or session end.
Run this when context is getting long or before stopping.

## Usage

```
/bip-ms-tuckin
```

## Workflow

### Step 1: Check for uncommitted manuscript changes

```bash
cd <manuscript-root>
git diff --stat
git diff --cached --stat
```

If there are uncommitted changes to TeX files, ask the user whether to commit them.
Draft a commit message summarizing what changed (e.g., "Update implementation notes: Phase 1b done, branchScale normalization").
Do not commit without confirmation.

If there are only untracked `ISSUE-*.md` files, note them but do not commit (these are intentionally untracked per CLAUDE.md).

### Step 2: Push EPIC body updates

For each tracked repo in `.ms-config.json`, check whether the EPIC body was edited this session (compare the `updatedAt` from the start of the session to the current `updatedAt`).
If not pushed yet, push any pending EPIC body updates:

```bash
gh issue view <epic-number> --repo <org/repo> --json updatedAt
```

Report which EPICs were pushed and which were already up to date.

### Step 3: Update `.ms-config.json`

Check whether `fetch_cmds` should be updated based on new experiments or results that arrived this session.
If new result paths were discovered but not added to `fetch_cmds`, note them for the user.

### Step 4: Update the durable session state

**Do not write memory files** under `~/.claude/projects/*/memory/` — they are keyed by
working directory, so they are invisible to other clones and to Erick.
Everything below lives in the repo instead.

1. **`misc/session-onboarding.md`** (create it if the repo has none — see the copies in
   `protein-dasm-tex` and `superfamily-pcp-tex` for the shape).
   This is what a fresh session reads on its first turn, so it carries only what is needed
   *before* acting: what the paper is and is not, the peer sessions and their remits, the
   working disciplines that cost a round to learn, environment gotchas, and the open items.
   Update it when any of those changed this session. Prefer pointing at the file that holds
   a fact over restating the fact, so it stays short and cannot go stale on its own.
   It is a standing document, not a log: **edit the lines that are now wrong rather than
   appending**, and delete an item when it closes.

   Its **live-threads section** carries what each open thread would *mean for the paper* —
   the reading that does not go stale — including analyses requested on PRs/issues this
   session, open scientific questions, and what each is waiting on, so the next session
   resumes the orchestration and not just the manuscript. (A thread that has *completed* is
   what triggers a paper update; one still open stays here.) **Never record
   open/merged/closed status**: GitHub is the source of truth and a status line is wrong
   within hours.

2. **Keep it to ONE file.** If the repo has both an onboarding doc and a separate notes
   file, merge them and delete the loser. Two documents with overlapping jobs is how state
   goes stale: the one you are looking at stays true and the other quietly lies, and
   whoever reads the wrong one has no way to tell. `protein-dasm-tex` had exactly this and
   merged into `misc/session-onboarding.md` on 2026-09-18.

3. **The `%PROV` / `%TODO` markers in the manuscript** are the primary memory for anything
   attached to a specific number or sentence, and they beat both files above because they
   sit where the claim is. If this session learned that a number is wrong, superseded, or
   not yet citable, that belongs in the marker next to it — not in a notes file a future
   session may not read. When marking a `%PROV` stale, **check its tail**: a prepended
   correction does not neutralize the original claim further down.

4. If Erick corrected an approach or confirmed a non-obvious choice, put it where it will
   be enforced: a durable project fact in `CLAUDE.md`, a workflow rule in a
   `~/re/bipartite/skills/` skill, a finding in the test, doc, or issue it concerns.

Only update what changed. Do not rewrite unchanged files.

### Step 5: Verify build

```bash
make pdf
```

Ensure the manuscript builds cleanly.
If it doesn't, fix the build error before completing tuckin.

### Step 6: Report

Print a summary:

```
## Tuckin Complete

### Manuscript
- Committed: <yes/no, commit hash if yes>
- Build: <clean/broken>
- Uncommitted ISSUE files: <count>
- Onboarding doc: <updated/unchanged — and what changed in it>

### EPICs
- <repo> i<N>: <pushed/up-to-date/skipped>

### Issues filed this session
- <repo>#<N>: <title> — <status>

### Pending for next session
- <brief list of what's unfinished>

### Key decisions
- <any decisions the next session should know about>
```

### Step 7: Continuation prompt

Write the continuation prompt to `_ignore/CONTINUE.md` and echo it, per `docs/guides/continuation-prompt.md`.
For this session: start in the tex repo, run `/bip-ms`, and the durable state is `misc/session-onboarding.md` plus the manuscript's `%PROV`/`%TODO` markers.
Then: safe to reset context.
