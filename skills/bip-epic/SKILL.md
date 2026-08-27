---
name: bip-epic
description: EPIC topic strategy — GitHub issue/PR triage, EPIC body composition, dependency-direction and collision detection between issues
---

# /bip-epic

Topic-scoped strategy for EPIC-based multi-clone orchestration: program
state, issue/PR triage, EPIC body composition, and the semantic
judgment calls a topic-agnostic fleet scanner structurally cannot make
(a model-semantics change landing mid-sweep, a deletion in one issue
that a sibling issue's patch depends on). Fleet mechanics — clone/tmux
inventory, staleness checks, spawning, pruning of clone-side artifacts
— belong to `/bip-conductor`. The two roles coordinate over
`SendMessage` and a shared `.spawn-prompts/` directory (see "Handing
spawn intent to the conductor" below); they do not need to be the same
session, though they may run side-by-side in one tmux host.

Use this at **session start** to establish topic context. For fleet
state (which clones are free, what's running where, tmux/host
occupancy), ask `/bip-conductor` instead — this skill does not scan
clones or tmux, and has no visibility into them.

## Role

The epic session does strategy, not fleet ops:
- Reads and interprets EPIC/issue/PR content on GitHub
- Composes and prunes EPIC bodies
- Flags dependency-direction conflicts and file-overlap collisions
  between open issues, before either gets spawned — by the time
  branches exist the cost of a collision is already sunk
- Decides an issue is ready and drafts the spawn brief, but does not
  spawn it — that goes to `/bip-conductor` as intent, not action
- Never writes code, creates branches, or spawns tmux windows for
  numbered issues itself

## Conventions

### Issue/PR naming
- `i281` = issue #281, `p275` = PR #275. Never bare `#N`.
- First mention in bullet lists: full URL inline.

Tmux window naming, reboot recovery, and the live-worker `SendMessage`
mechanics are `/bip-conductor` concerns — see that skill's Conventions
section.

## Configuration

Reads `.epic-config.json` from the repo root — the same file
`/bip-conductor` uses, so the two roles never disagree about where
things live. The epic role only needs two fields out of it:
- **github_repo**: `org/repo` for `gh` commands
- **clone_root**: to locate the shared `$CLONE_ROOT/.spawn-prompts/`
  directory (see below)

**If the file does not exist**, don't create it here — run
`/bip-conductor` first. It owns the clone/worktree layout questions
(clone mode vs. worktree mode, clone names, shared filesystem) that
the rest of the file's fields are about.

## Workflow

### Step 1: Load config and memory

```bash
cat .epic-config.json
```

Also read MEMORY.md from the auto-memory directory for topic-level
context from previous sessions (decisions, findings, what's next).

### Step 2: Fan out scanners

Dispatch two groups of `general-purpose` subagents in parallel —
single message, multiple `Agent` tool calls. Follow the dispatch
pattern in `SUBAGENT-SCAN.md` (bipartite repo root). Start with the
cheap structured listing the primary can do directly:

```bash
gh issue list --search "EPIC in:title" --json number,title
```

**Group A: one subagent per EPIC.** Brief:

> Read EPIC `i<N>` and report its current state. Tasks:
> 1. `gh issue view <N> --json title,body,updatedAt`.
> 2. Parse the Status dashboard and Key findings. (Older EPIC bodies
>    may still carry a legacy clone-assignment table from before the
>    fleet/topic split — that's fleet state that shouldn't have been
>    authored here; note it as a `surprise` for cleanup, don't treat
>    it as current truth.)
> 3. For each open item the dashboard lists, run `gh issue view
>    <child-N> --json state,stateReason` to confirm it is still open.
>
> Return under 400 words:
> - `changes_since_baseline`: completed items, new findings, items
>   newly opened
> - `active_items`: open work with brief status (for which clone is
>   running it, if any, that's `/bip-conductor`'s dashboard, not this
>   report)
> - `action_candidates`: items the EPIC marks ready but unassigned
> - `surprises`: contradictions, legacy clone-assignment tables found
>   in the body, `RECOMMEND DEEPER LOOK` flags

**Group B: one PR/issue triage subagent.** Brief:

> Triage the backlog for the epic. Tasks:
> 1. `gh issue list --search "sort:updated-desc" --limit 20 --json
>    number,title,state,labels,body`.
> 2. `gh pr list --json number,title,headRefName,state`.
> 3. `gh pr list --search "is:pr is:merged sort:updated-desc"
>    --limit 10 --json number,title,mergedAt`.
> 4. For each open issue, check its `depends_on` field and any
>    blocking context. An issue that depends on an unmerged PR or
>    unfinished experiment is NOT ready — omit silently.
> 5. Verify state with `gh issue view <N> --json state` for any
>    issue you plan to flag — never claim "open" without confirmation.
>
> Return under 400 words:
> - `changes_since_baseline`: PRs merged since last session
> - `active_items`: open PRs with state/CI status
> - `action_candidates`: open issues ready to spawn (unblocked,
>   unassigned, dependencies satisfied), ordered by priority
> - `surprises`: closed/merged items the EPICs don't reflect yet,
>   issues with unclear blocker state, `RECOMMEND DEEPER LOOK` flags

**Never ask the user a question about an issue/PR status that you
could answer with a `gh` query** — verify first, then present facts.

### Step 3: Reconcile

Compose the reconciliation from the two reports — do not paste
subagent prose verbatim.

- Never present an issue as "ready to spawn" or "needs action" without
  confirming it's still OPEN on GitHub
- Flag anything merged/closed that an EPIC body doesn't reflect yet
- If either group's report has zero `surprises` and zero
  `changes_since_baseline`, send a follow-up to that subagent with a
  narrower question before concluding "nothing changed there."

Clone/tmux occupancy is not part of this reconciliation — it's
`/bip-conductor`'s dashboard, not this one.

### Step 4: Dependency-direction and collision detection

Run this before proposing any issue as ready to spawn, not after —
once a branch exists the cost of a collision is sunk. This automates
what has so far been a manual capability: extracting file paths from
issue bodies and building an overlap matrix.

1. From every currently open, unassigned (or about-to-be-spawned)
   issue body, extract mentioned file/module paths — code blocks,
   a "Files:" section, or explicit paths in prose.
2. Build an overlap matrix: any two open issues naming the same file
   or module are a candidate collision.
3. For each overlap, read both issue bodies against each other and
   ask the harder question, not just "who goes first": can both land
   as currently written, in either order? Or does one delete,
   rename, or restructure something the other's patch depends on —
   a dependency-direction conflict that no ordering fixes, because one
   issue has to change shape? (Concrete shape: one issue deletes a
   function a sibling issue is patching; one issue's test assertion
   contradicts a value a sibling issue is about to change.)
4. Record findings where a future spawn will see them: a note in the
   EPIC body's dashboard, and — if either issue is about to be handed
   off as spawn intent (Step 6) — the warning goes directly into that
   issue's brief.

### Step 5: Prune and update the EPIC body

Follow the **EPIC body update pattern** below. Do this whenever
findings come in, items complete, or new work starts — not on a fixed
cadence.

**Prune on every write, not on a separate sweep.** The observed
failure mode is accretion, not staleness: a body cut from 524 to 146
lines had almost nothing stale in what was removed — it was cut
because the work had *finished* (checked-off phases, a fully-ticked
table) and kept getting appended around instead of trimmed. Whenever
you touch a body to add a finding or check a box, also look for
sections whose work is now fully complete and cut them in the same
edit, rather than letting them ride to the next dedicated cleanup.

**Fleet state is derived, never authored — don't let it back into the
body.** Which clone holds which issue, which slots are free, which
tmux windows are live: recompute this from `git`/`tmux`/`gh` (that's
`/bip-conductor`'s Step 5 dashboard) whenever you need it. Never write
it into an EPIC body or a continuation doc — it goes stale within a
day and there is no mechanism to notice, which is exactly the failure
a hand-maintained clone table caused when it went stale twice in one
day. Handoff artifacts (spawn intent, unfiled drafts) are the opposite
category: authored deliberately, to a durable path outside git,
precisely so they survive clone churn — pruning them (Step 6, below)
is about cleaning up finished authored state, not about avoiding
writing derived state down in the first place.

### Step 6: Hand spawn intent to the conductor

For each issue judged ready (unblocked per Step 3, no unresolved
collision or dependency-direction conflict per Step 4), draft the
semantic brief — why it matters, scope, any dependency/collision
warnings from Step 4 — and write it to:

```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
mkdir -p "$CLONE_ROOT/.spawn-prompts"
# Use the Write tool to create "$CLONE_ROOT/.spawn-prompts/<N>.md" with the brief
```

`clone_root` in `.epic-config.json` is tilde-form — resolve it the same
way every `/bip-conductor*` skill does, or the brief silently lands in
a literal `~` directory in the cwd instead of the shared intent
directory, and the conductor never sees it. `mkdir -p` because
`.spawn-prompts/` won't exist yet on a fresh repo.

**Naming note**: an older, already-live convention in this same
directory names files `spawn-<N>.txt` instead. Both are valid intent —
`/bip-conductor-spawn` checks for either (see that skill's "Where the
prompt comes from"). Use `<N>.md` for new intent; don't rename existing
`spawn-<N>.txt` files you find there, since one may be mid-flight.

This directory lives outside every clone's git (deliberately — it
must survive clone churn) and is the handoff point to
`/bip-conductor-spawn`, which reads it, checks it against live fleet
state, appends the fleet facts only it can see, and executes after
user confirmation. **Do not spawn or touch tmux/clones yourself** —
that's the conductor's job, and mixing the two roles back together is
exactly the failure mode this split exists to avoid.

**Check whether a conductor session exists before announcing a handoff
into a void** — `ListAgents` for one. Writing intent and telling the
user "the conductor will pick these up" is only true if a conductor is
or will be running; on a small job with no separate conductor session,
that message describes nothing and the intent file just accretes,
unpicked-up, forever.

- **Conductor addressable**: tell the user which issues now have
  pending intent —
  > "Spawn intent written for `i302` (retry logic) and `i315` (scoring
  > refactor) — the conductor will pick these up."
  If it's urgent, `ListAgents`/`SendMessage` it directly rather than
  waiting for its next poll cycle — see `/bip-conductor`'s Conventions
  section for the mechanics and addressability caveats.
- **No conductor addressable**: say so, and offer both real options —
  start one now (`/bip-conductor` in a separate tmux window, the normal
  setup for a multi-issue fleet), or, for a single small job where
  standing up a second session is overhead, run `/bip-conductor-spawn`
  yourself from this session for just this intent file. The second
  option is a deliberate, user-confirmed exception to "don't spawn
  yourself" above, not a silent default — say plainly that it mixes
  the two roles for this one spawn, and let the user pick.

### Step 7: Correcting a live worker — the judgment half

When a live worker's scope needs correcting *before* its next natural
stopping point, the epic — not the conductor — makes the call: is this
change durable (it changes what the worker will produce: scope,
target, artifact, gate criterion) or transient (host load, a peer's
timing, a dependency that just landed)?

- **Durable**: draft the one-line correction and ask the conductor to
  deliver it — the conductor `SendMessage`s the worker directly and
  appends a timestamped, attributed entry to `.epic-worklog.md` in the
  same step (never edits `lead_guidance` — see `/bip-conductor`'s
  Conventions section for why the append-only path matters). The
  epic's job stops at drafting the line — route it through the
  conductor even though `SendMessage` would reach the worker directly
  from here too. A single delivery path is what keeps the message and
  the `.epic-worklog.md` append from diverging: two senders means a
  correction can land in the worker's context with no durable record,
  or a durable record with no delivery.
- **Transient**: message-only is fine; still routes through the
  conductor, for the same single-delivery-path reason.

A worker correction is often *more* time-sensitive than spawn intent —
the worker is actively doing the wrong thing while it sits in a queue.
If the conductor session is addressable right now, `SendMessage` it
directly to request immediate delivery, rather than leaving the
correction for its next poll cycle — same mechanics and addressability
caveats as Step 6's escape hatch.

## EPIC body update pattern

EPIC issue bodies are the source of truth for project status. Update
them when findings come in, items complete, or new work starts — and
prune completed sections in the same edit (Step 5 above).

**Local file convention**: Keep a persistent local copy as
`ISSUE-EPIC-<N>.md` in the repo root (e.g. `ISSUE-EPIC-281.md`,
`ISSUE-EPIC-295.md`). These files are gitignored via the `ISSUE-*.md`
pattern.

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

**Conflict check**: Record `updatedAt` when pulling. Before pushing,
re-fetch `updatedAt` — if it changed, someone else edited. Re-pull,
merge their changes, and retry. When in doubt, ask the user.

Key sections to maintain:
- **Status dashboard**: Check/uncheck boxes, add new items, cut
  finished ones
- **Key findings**: Numbered list, append new findings
- **Related experiments**: Add new experiment rows

**Do not maintain a clone/slot assignment table in the body** — which
clone is working which issue is fleet state, derived from `git`/`tmux`,
not authored here (see the "Fleet state is derived" note above). If an
item's status dashboard entry needs a pointer to who's on it, name the
issue, not the clone — `/bip-conductor`'s dashboard is the place to
ask "which clone."

## Error handling

- **No EPIC issues found**: Report and offer to create one
- **gh not authenticated**: Suggest `gh auth login`
- **`.epic-config.json` missing**: Run `/bip-conductor` first — it owns setup

## Layout config (issue #149)

`.epic-config.json`'s `clone_root` / `clone_names` / `local_worktrees`
keep working untouched. The newer way to configure worktree mode (for
non-EPIC `bip spawn` use) is the `layout:` block in
`~/.config/bip/config.yml` — see `docs/guides/layout.md`. EPIC
orchestration still reads `.epic-config.json` for now.
