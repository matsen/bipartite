---
name: bip-epic
description: EPIC topic strategy — GitHub issue/PR triage, EPIC body composition, dependency-direction and collision detection between issues
---

# /bip-epic

Topic-scoped strategy for EPIC-based multi-clone orchestration: program state, issue/PR triage, EPIC body composition, and the semantic judgment calls a topic-agnostic fleet scanner structurally cannot make (a model-semantics change landing mid-sweep, a deletion in one issue that a sibling issue's patch depends on).
Fleet mechanics — clone/tmux inventory, staleness checks, spawning, pruning of clone-side artifacts — belong to `/bip-conductor`.
The two roles coordinate over `SendMessage` and a shared `.spawn-prompts/` directory (see "Handing spawn intent to the conductor" below); they do not need to be the same session, though they may run side-by-side in one tmux host.

Use this at **session start** to establish topic context.
For fleet state (which clones are free, what's running where, tmux/host occupancy), ask `/bip-conductor` instead — this skill does not scan clones or tmux itself.
It does *consume* a conductor-supplied occupancy table when a conductor handshake succeeds (Step 2 below), but that table is read-only input to scoping, not something this skill goes and gets on its own.

## Role

The epic session does strategy, not fleet ops:
- Reads and interprets EPIC/issue/PR content on GitHub
- Composes and prunes EPIC bodies
- Flags dependency-direction conflicts and file-overlap collisions between open issues, before either gets spawned — by the time branches exist the cost of a collision is already sunk
- Decides an issue is ready and drafts the spawn brief, but does not spawn it — that goes to `/bip-conductor` as intent, not action
- Never writes code, creates branches, or spawns tmux windows for numbered issues itself

## Conventions

### Issue/PR naming
- `i281` = issue #281, `p275` = PR #275.
  Never bare `#N`.
- First mention in bullet lists: full URL inline.

Tmux window naming, reboot recovery, and the live-worker `SendMessage` mechanics are `/bip-conductor` concerns — see that skill's Conventions section.

## Configuration

Reads `.epic-config.json` from the repo root — the same file `/bip-conductor` uses, so the two roles never disagree about where things live.
The epic role only needs two fields out of it:
- **github_repo**: `org/repo` for `gh` commands
- **clone_root**: to locate the shared `$CLONE_ROOT/.spawn-prompts/` directory (see below)

**If the file does not exist**, don't create it here — run `/bip-conductor` first.
It owns the clone/worktree layout questions (clone mode vs. worktree mode, clone names, shared filesystem) that the rest of the file's fields are about.

## Workflow

### Step 1: Load config and memory

```bash
cat .epic-config.json
```

If this project uses the auto-memory directory, also read its MEMORY.md for topic-level context from previous sessions (decisions, findings, what's next) — some setups deliberately don't use it (e.g. because the directory is keyed by working directory and invisible to other clones), in which case skip this and rely on EPIC bodies and issue history instead.

**Self-register for completion pushes**: resolve `CLONE_ROOT` and write this session's own `ListAgents` name (the "This session is ..." row) as the sole line of `$CLONE_ROOT/.epic-session` — this is how the conductor finds the epic to push a `needs-human`/`completed` notification without guessing among `ListAgents` rows.
See `/bip-conductor`'s Conventions section ("Completion pushes") for the full mechanics and why guessing isn't safe here.
Re-run this write every time `/bip-epic` starts a fresh poll cycle, since the address can drift mid-session.

### Step 2: Fan out scanners

**Handshake with the conductor before dispatching Group A.**
Group A's step 3 re-verifies every open child item's state one `gh issue view` at a time; most of that is exactly what `/bip-conductor`'s own Step 5 dashboard already computed 90 seconds earlier from `git`/`tmux`.
In the run that motivated this rule, Group A spent ~58k tokens re-verifying 10 issues this way, 9 of which the conductor's own report covered a minute and a half later.

- Resolve `CLONE_ROOT`, read `$CLONE_ROOT/.conductor-session` for the conductor's self-registered address (see `/bip-conductor`'s Conventions section, "Completion pushes"), and `SendMessage` it a request for its current slot→issue occupancy table (Step 5's dashboard) and its negative list (decisions already taken against an action — see `/bip-conductor`'s Step 5).
- **Payload**: the slot→issue occupancy table (which issue, and which **phase**, each slot holds) plus the negative list.
- **Scope**: "what the conductor does not already own" means issues with no live slot in that table, **and only for slots in an in-progress phase** (`coding`, `exploring`, `testing`, `awaiting-results`, `quality-gate`) — occupancy is evidence about the fleet, not about GitHub, and a slot's issue can close while the slot itself still shows occupied.
  Never skip the confirmation for `completed` or `needs-human` phases: a slot that just finished is *more* likely to have a closed issue than one still mid-flight, since finishing is exactly what triggers the issue closing, and slot cleanup (which would otherwise clear the slot) is a separate housekeeping step that can lag behind.
  Pass the table (with phase) down to each Group A subagent and have it skip step 3's `gh issue view` confirmation only for child items on an in-progress-phase slot. Everything else about the subagent's job (reading the EPIC body, parsing findings, flagging surprises) is unaffected; only the redundant per-item open/closed check is narrowed.
- **Fallback**: if `$CLONE_ROOT/.conductor-session` is absent, or the send fails (no conductor addressable), proceed with the full Group A dispatch below, unscoped — run step 3's confirmation for every item, and do not block waiting for a conductor that may never exist.
  A solo `/bip-epic` run with no separate conductor session must not deadlock here (see Step 6's conductor-absent handling for the same fallback shape).

This narrows Group A's redundant re-verification; it does not narrow Group A's judgment.
In the run that motivated this rule, Group A independently found an open PR and three dependency corrections the conductor's dashboard had no way to see — occupancy and dependency reasoning are different questions, and the conductor's table answers only the first.

Dispatch two groups of `general-purpose` subagents in parallel — single message, multiple `Agent` tool calls.
Follow the dispatch pattern in `SUBAGENT-SCAN.md` (bipartite repo root).
Start with the cheap structured listing the primary can do directly:

```bash
gh issue list --search "EPIC in:title" --json number,title
```

**Group A: one subagent per EPIC.**
Brief:

> Read EPIC `i<N>` and report its current state.
> Tasks:
> 1. `gh issue view <N> --json title,body,updatedAt`.
> 2. Parse the Status dashboard and Key findings.
>    (Older EPIC bodies may still carry a legacy clone-assignment table from before the fleet/topic split — that's fleet state that shouldn't have been authored here; note it as a `surprise` for cleanup, don't treat it as current truth.)
> 3. For each open item the dashboard lists, run `gh issue view <child-N> --json state,stateReason` to confirm it is still open — **except** items the conductor's occupancy table (handshake above) shows holding a slot in an **in-progress** phase (`coding`, `exploring`, `testing`, `awaiting-results`, `quality-gate`); take "open" as given only for those and skip the query. Still confirm for `completed`/`needs-human` slots, and for anything the table doesn't cover — a slot can finish and its issue close before the slot itself is cleaned up, so occupancy alone never implies "still open."
>
> Return under 400 words:
> - `changes_since_baseline`: completed items, new findings, items newly opened
> - `active_items`: open work with brief status (for which clone is running it, if any, that's `/bip-conductor`'s dashboard, not this report)
> - `action_candidates`: items the EPIC marks ready but unassigned
> - `surprises`: contradictions, legacy clone-assignment tables found in the body, `RECOMMEND DEEPER LOOK` flags

**Group B: one recent-activity subagent.**
Brief:

> Report recent issue/PR activity for the epic.
> This is a recency feed, not backlog coverage — see the note below.
> Tasks:
> 1. `gh issue list --search "sort:updated-desc" --limit 20 --json number,title,state,labels,body`.
> 2. `gh pr list --json number,title,headRefName,state`.
> 3. `gh pr list --search "is:pr is:merged sort:updated-desc" --limit 10 --json number,title,mergedAt`.
> 4. For each open issue, check its `depends_on` field and any blocking context.
>    An issue that depends on an unmerged PR or unfinished experiment is NOT ready — omit silently.
> 5. Verify state with `gh issue view <N> --json state` for any issue you plan to flag — never claim "open" without confirmation.
>
> Return under 400 words:
> - `changes_since_baseline`: PRs merged since last session
> - `active_items`: open PRs with state/CI status
> - `action_candidates`: open issues ready to spawn (unblocked, unassigned, dependencies satisfied), ordered by priority
> - `surprises`: closed/merged items the EPICs don't reflect yet, issues with unclear blocker state, `RECOMMEND DEEPER LOOK` flags

**Group B does not cover the backlog, and must not be relied on as if it did.**
`sort:updated-desc --limit 20` reaches back only as far as the last 20 touched issues, which on an active repo is a couple of days: measured on `matsengrp/phyz` 2026-08-28, that window spanned 08-26 to 08-28 and saw 20 of 65 open issues.
The window compresses further as activity rises, so an issue that is neither on an EPIC dashboard nor recently touched is reachable by neither group.
Coverage is Group A's job plus a periodic audit for issues no EPIC references (see `matsengrp/phyz`#2093 for the audit script and the shape of that gap); Group B only answers "what moved lately".

**Never ask the user a question about an issue/PR status that you could answer with a `gh` query** — verify first, then present facts.

### Step 3: Reconcile

Compose the reconciliation from the two reports — do not paste subagent prose verbatim.

- Never present an issue as "ready to spawn" or "needs action" without confirming it's still OPEN on GitHub
- Flag anything merged/closed that an EPIC body doesn't reflect yet
- If either group's report has zero `surprises` and zero `changes_since_baseline`, send a follow-up to that subagent with a narrower question before concluding "nothing changed there."

This reconciliation itself does not scan clone/tmux occupancy — that's `/bip-conductor`'s dashboard, not this one — though Step 2's handshake may already have pulled the conductor's occupancy table in for scoping purposes by the time you reach this step.

### Step 4a: Dependency-direction detection

Run this before proposing any issue as ready to spawn, not after — once a branch exists the cost of a collision is sunk.

1. For every currently open, unassigned (or about-to-be-spawned) issue, read its body against every other such issue's body and ask: can both land as currently written, in either order?
   Or does one delete, rename, or restructure something the other's patch depends on — a dependency-direction conflict that no ordering fixes, because one issue has to change shape?
   (Concrete shape: one issue deletes a function a sibling issue is patching; one issue's test assertion contradicts a value a sibling issue is about to change.)
2. Record findings where a future spawn will see them: a note in the EPIC body's dashboard, and — if either issue is about to be handed off as spawn intent (Step 6) — the warning goes directly into that issue's brief.

**Completion criterion**: every pair of currently-open, about-to-be-spawned issues has been read against each other for dependency direction, not just for whether one references the other's issue number.

### Step 4b: File-overlap collision detection

This is a distinct analysis from 4a, not a second pass over the same reading — it automates what has so far been a manual capability: extracting file paths from issue bodies and building an overlap matrix.

1. From every currently open, unassigned (or about-to-be-spawned) issue body, extract mentioned file/module paths — code blocks, a "Files:" section, or explicit paths in prose.
2. Build an overlap matrix: any two open issues naming the same file or module are a candidate collision.
3. For each overlap, this is not the same question as 4a's — file overlap alone (two issues touching the same file, in disjoint regions or compatible ways) is not itself a dependency-direction conflict, and clearing a pair on dependency grounds in 4a does not clear it here.
   Confirm whether the overlapping regions can coexist or whether one issue's edit invalidates the other's, even absent any direction conflict.
4. Record findings the same way as 4a: a dashboard note, and a brief warning for any issue about to be handed off (Step 6).

**Completion criterion**: every pair of currently-open, about-to-be-spawned issues naming an overlapping file/module has an explicit overlap verdict — "compatible" or "conflicts, because ___" — not merely "no dependency conflict found in 4a."
**A pair cleared in 4a is not thereby cleared here** — this is the check that "No collisions" wrongly skipped when two issues sharing a dependency-independent EPIC both edited `experiments/2026-08-27-2051-heavy-v2-stated-estimator/scripts/rescore_cell.py`.

### Step 5: Prune and update the EPIC body

Follow the **EPIC body update pattern** below.
Do this whenever findings come in, items complete, or new work starts — not on a fixed cadence.

**Prune on every write, not on a separate sweep.**
The observed failure mode is accretion, not staleness: a body cut from 524 to 146 lines had almost nothing stale in what was removed — it was cut because the work had *finished* (checked-off phases, a fully-ticked table) and kept getting appended around instead of trimmed.
Whenever you touch a body to add a finding or check a box, also look for sections whose work is now fully complete and cut them in the same edit, rather than letting them ride to the next dedicated cleanup.

**Fleet state is derived, never authored — don't let it back into the body.**
Which clone holds which issue, which slots are free, which tmux windows are live: recompute this from `git`/`tmux`/`gh` (that's `/bip-conductor`'s Step 5 dashboard) whenever you need it.
Never write it into an EPIC body or a continuation doc — it goes stale within a day and there is no mechanism to notice, which is exactly the failure a hand-maintained clone table caused when it went stale twice in one day.
Handoff artifacts (spawn intent, unfiled drafts) are the opposite category: authored deliberately, to a durable path outside git, precisely so they survive clone churn — pruning them (Step 6, below) is about cleaning up finished authored state, not about avoiding writing derived state down in the first place.

**A user decision may be written into the EPIC body only against a `FINAL` shape.**
When the session talking to the user is the conductor, not this one, the decision arrives here as a relay — see `/bip-conductor`'s "Decision relays: PROVISIONAL and FINAL" — and every relay is marked one or the other.
A `PROVISIONAL` relay is discussion in progress, not yet a settled decision; write nothing into the body from it beyond noting that a decision is pending, and check `.epic-decisions.md` in the conductor cwd (see `/bip-conductor`'s ".epic-decisions.md: the durable fleet-decision log") for the eventual `FINAL` entry before composing the body edit.
Only a `FINAL` relay — or a decision reached directly in this session — authorizes the write, and the body text should restate the decision's substance, not a paraphrase of the sentiment that led to it: "burn everything down and make it right" said in conversation is not "make parity diverges permanently" written into an EPIC body: the paraphrase is a second-hand summary as if it were the specification itself.
If the deciding session's shape is unclear, treat it as `PROVISIONAL` and ask before writing.

### Step 6: Hand spawn intent to the conductor

For each issue judged ready (unblocked per Step 3, no unresolved dependency-direction conflict per Step 4a, no unresolved file-overlap collision per Step 4b), draft the semantic brief — why it matters, scope, any dependency/collision warnings from Step 4a/4b — and write it to:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
mkdir -p "$CLONE_ROOT/.spawn-prompts"
# Use the Write tool to create "$CLONE_ROOT/.spawn-prompts/<N>.md" with the brief
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-epic`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory (see `skills/lib/spawn-intent.sh` in the `bipartite` repo).

`clone_root` in `.epic-config.json` is tilde-form — resolve it the same way every `/bip-conductor*` skill does, or the brief silently lands in a literal `~` directory in the cwd instead of the shared intent directory, and the conductor never sees it.
`mkdir -p` because `.spawn-prompts/` won't exist yet on a fresh repo.

**Naming note**: an older, already-live convention in this same directory names files `spawn-<N>.txt` instead.
Both are valid intent — `/bip-conductor-spawn` checks for either (see that skill's "Where the prompt comes from").
Use `<N>.md` for new intent; don't rename existing `spawn-<N>.txt` files you find there, since one may be mid-flight.

This directory lives outside every clone's git (deliberately — it must survive clone churn) and is the handoff point to `/bip-conductor-spawn`, which reads it, checks it against live fleet state, appends the fleet facts only it can see, and executes after user confirmation.
**Do not spawn or touch tmux/clones yourself** — that's the conductor's job, and mixing the two roles back together is exactly the failure mode this split exists to avoid.

**Check whether a conductor session exists before announcing a handoff into a void** — `ListAgents` for one.
Writing intent and telling the user "the conductor will pick these up" is only true if a conductor is or will be running; on a small job with no separate conductor session, that message describes nothing and the intent file just accretes, unpicked-up, forever.

- **Conductor addressable**: tell the user which issues now have pending intent —
> "Spawn intent written for `i302` (retry logic) and `i315` (scoring refactor) — the conductor will pick these up."
If it's urgent, `ListAgents`/`SendMessage` it directly rather than waiting for its next poll cycle — see `/bip-conductor`'s Conventions section for the mechanics and addressability caveats.
- **No conductor addressable**: say so, and offer both real options — start one now (`/bip-conductor` in a separate tmux window, the normal setup for a multi-issue fleet), or, for a single small job where standing up a second session is overhead, run `/bip-conductor-spawn` yourself from this session for just this intent file.
  The second option is a deliberate, user-confirmed exception to "don't spawn yourself" above, not a silent default — say plainly that it mixes the two roles for this one spawn, and let the user pick.

### Step 7: Correcting a live worker — the judgment half

When a live worker's scope needs correcting *before* its next natural stopping point, the epic — not the conductor — makes the call: is this change durable (it changes what the worker will produce: scope, target, artifact, gate criterion) or transient (host load, a peer's timing, a dependency that just landed)?

- **Durable**: draft the one-line correction and ask the conductor to deliver it — the conductor `SendMessage`s the worker directly and appends a timestamped, attributed entry to `.epic-worklog.md` in the same step (never edits `lead_guidance` — see `/bip-conductor`'s Conventions section for why the append-only path matters).
  The epic's job stops at drafting the line — route it through the conductor even though `SendMessage` would reach the worker directly from here too.
  A single delivery path is what keeps the message and the `.epic-worklog.md` append from diverging: two senders means a correction can land in the worker's context with no durable record, or a durable record with no delivery.
- **Transient**: message-only is fine; still routes through the conductor, for the same single-delivery-path reason.

A worker correction is often *more* time-sensitive than spawn intent — the worker is actively doing the wrong thing while it sits in a queue.
If the conductor session is addressable right now, `SendMessage` it directly to request immediate delivery, rather than leaving the correction for its next poll cycle — same mechanics and addressability caveats as Step 6's escape hatch.

## EPIC body update pattern

EPIC issue bodies are the source of truth for project status.
Update them when findings come in, items complete, or new work starts — and prune completed sections in the same edit (Step 5 above).

**Local file convention**: Keep a persistent local copy as `ISSUE-EPIC-<N>.md` in the repo root (e.g. `ISSUE-EPIC-281.md`, `ISSUE-EPIC-295.md`).
These files are gitignored via the `ISSUE-*.md` pattern.

```bash
# Pull current body and record the timestamp
gh issue view <number> --json body,updatedAt > /tmp/epic-pull.json
jq -r .body /tmp/epic-pull.json > ISSUE-EPIC-<N>.md
PULLED_AT=$(jq -r .updatedAt /tmp/epic-pull.json)
rm -f /tmp/epic-pull.json

# Edit the file (add findings, check boxes, cut finished sections)
# ...

# Before pushing: check if someone else edited since our pull
CURRENT_AT=$(gh issue view <number> --json updatedAt -q .updatedAt)
if [ "$PULLED_AT" != "$CURRENT_AT" ]; then
  echo "CONFLICT: Issue was updated since pull ($PULLED_AT → $CURRENT_AT)"
  echo "Re-pull, merge changes, then try again."
  # Stop here — do NOT push
else
  gh issue edit <number> --body-file ISSUE-EPIC-<N>.md
fi
```

**Conflict check**: Record `updatedAt` when pulling.
Before pushing, re-fetch `updatedAt` — if it changed, someone else edited.
Re-pull, merge their changes, and retry.
When in doubt, ask the user.

Key sections to maintain:
- **Status dashboard**: Check/uncheck boxes, add new items, cut finished ones
- **Key findings**: Numbered list, append new findings
- **Related experiments**: Add new experiment rows

**Do not maintain a clone/slot assignment table in the body** — which clone is working which issue is fleet state, derived from `git`/`tmux`, not authored here (see the "Fleet state is derived" note above).
If an item's status dashboard entry needs a pointer to who's on it, name the issue, not the clone — `/bip-conductor`'s dashboard is the place to ask "which clone."

## Error handling

- **No EPIC issues found**: Report and offer to create one
- **gh not authenticated**: Suggest `gh auth login`
- **`.epic-config.json` missing**: Run `/bip-conductor` first — it owns setup

## Layout config (issue #149)

`.epic-config.json`'s `clone_root` / `clone_names` / `local_worktrees` keep working untouched.
The newer way to configure worktree mode (for non-EPIC `bip spawn` use) is the `layout:` block in `~/.config/bip/config.yml` — see `docs/guides/layout.md`.
EPIC orchestration still reads `.epic-config.json` for now.
