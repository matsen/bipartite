---
name: bip-epic
description: EPIC topic strategy — GitHub issue/PR triage, EPIC body composition, dependency-direction and collision detection between issues
---

# /bip-epic

Topic-scoped strategy for EPIC-based multi-clone orchestration: program state, issue/PR triage, EPIC body composition, and the semantic judgment calls a topic-agnostic fleet scanner structurally cannot make (a model-semantics change landing mid-sweep, a deletion in one issue that a sibling issue's patch depends on).
Fleet mechanics — clone/tmux inventory, staleness checks, spawning, pruning of clone-side artifacts — belong to `/bip-conductor`.
The two roles coordinate over `SendMessage` and a shared `.spawn-prompts/` directory (see "Handing spawn intent to the conductor" below); they do not need to be the same session, though they may run side-by-side in one tmux host.

Use this at **session start** to establish topic context.

## One EPIC at a time — ask if the invocation does not name one

**This skill is scoped to exactly one EPIC per session, and that EPIC is its
exclusive purview.** `/bip-epic 369` means EPIC #369 and nothing else. If the
invocation does not name an EPIC number, **ask which one before doing anything
else** — do not infer it from the repo, from the most recently updated EPIC, or
from what the fleet happens to be running, and do not survey every EPIC to
decide. There may be several open EPICs; only one is yours.

The failure this prevents is not a wrong answer, it is a slow drift that every
individual step justifies. Measured on `matsengrp/phyz` 2026-09-03: an unscoped
session scanned all eight open EPICs, then wrote spawn briefs for whatever
looked ready across them. **Eleven of twenty-three briefed issues belonged to
other EPICs, and ten of fifteen live slots ended up outside the session's
actual purview** before anyone noticed. Three of four EPIC bodies it reconciled
were not its own. No single brief was wrong on its merits; the boundary was
never stated, so nothing ever tripped.

Two corollaries, both learned the same day:

- **A file collision is not EPIC membership.** Two issues editing
  `src/core/preset.zig` can belong to different EPICs — one shipping a
  regime-specific preset for a named downstream consumer, the other building a
  general selection mechanism. Purpose determines membership; shared files
  determine only sequencing. Do not adopt an out-of-scope issue because it
  collides with an in-scope one; report the collision to the conductor and
  leave it.
- **Interesting is not in-scope.** A genuine defect found while working an
  in-scope issue, but belonging to another programme, gets **filed and parked**
  for a future epic session — not pursued because it is real and someone is
  already looking at it.

If work already in flight turns out to be outside the boundary, say so plainly
and hand it back rather than finishing it quietly; preserving WIP to a branch
beats both discarding it and completing it out of scope.
For fleet state (which clones are free, what's running where, tmux/host occupancy), ask `/bip-conductor` instead — this skill does not scan clones or tmux itself.
It does *consume* a conductor-supplied occupancy table when a conductor handshake succeeds (Step 2 below), but that table is read-only input to scoping, not something this skill goes and gets on its own.

## Role

The epic session does strategy, not fleet ops:
- Reads and interprets EPIC/issue/PR content on GitHub
- Composes and prunes EPIC bodies
- Flags dependency-direction conflicts and file-overlap collisions between open issues, before either gets spawned — by the time branches exist the cost of a collision is already sunk
- Decides an issue is ready and drafts the spawn brief, but does not spawn it — that goes to `/bip-conductor` as intent, not action
- Owns the user-facing topic channel: this is where scientific reasoning gets presented and where scientific decisions get made.
  `/bip-conductor` narrates fleet state only and points here for substance (see that skill's "The fleet/topic line"), so expect a semantic question raised in the conductor's window to arrive here rather than be answered there.
- Never writes code, creates branches, or spawns tmux windows for numbered issues itself

## Conventions

### Issue/PR naming
- `i281` = issue #281, `p275` = PR #275.
  Never bare `#N`.
- First mention in bullet lists: full URL inline.

Tmux window naming, reboot recovery, and the live-worker `SendMessage` mechanics are `/bip-conductor` concerns — see that skill's Conventions section.

### Message economy

Cross-session messages are nudges, not transcripts: lead with the decision or the ask, give the reasoning the receiver cannot reconstruct, and cite the EPIC body, the issue, or `.epic-decisions.md` for the rest instead of restating it. There is no word limit, but a message that recaps the thread, or reproduces what you just wrote to a durable artifact, spends context the fleet needs elsewhere. This never licenses a bare pointer where Step 7 requires substance: a drafted worker correction states the change itself, not "re-read the EPIC body." See `/bip-conductor`'s "Message economy" section for the full convention; it applies symmetrically here.

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
git pull --ff-only origin main
cat .epic-config.json
```

**Re-run that pull at the top of every fresh scan cycle, not just at cold start.**
An epic session only ever *reads* remote state — every `gh issue view` and `gh pr view` is correct regardless of how old the working tree is — so nothing in the normal loop ever forces a pull, and the tree rots silently while every answer stays right.
Measured 2026-09-01: both the epic and the conductor sat four merges behind on `main` for hours, each with a clean tree on the correct branch, and neither noticed until the other described it. `git fetch` does not help: it updates remote-tracking refs, not the working tree.
This matters here specifically because reading a stale file and writing it back is how an EPIC body or a skill acquires a reverted diff — see `/bip-kaizen` Step 5 on the shared working tree.

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
**Scope this fan-out to *your* EPIC** (see "One EPIC at a time" above). Confirm
the number you were given resolves to a real EPIC issue, and do not enumerate
the others:

```bash
gh issue view <your-epic-N> --json number,title,updatedAt
```

Listing every `EPIC in:title` issue is only appropriate when you have been
asked which EPIC to adopt and are presenting the options to the user — never as
the opening move of a scan, which is how a session ends up briefing other
EPICs' work.

**Group A: one subagent for your EPIC.** (Dispatch one per EPIC only in the
rare case the user has explicitly scoped you to more than one.)
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
> - `action_candidates`: open issues ready to spawn (unblocked, unassigned, dependencies satisfied), ordered by priority.
>   **Restrict these to issues this session's EPIC tracks** — the recency window is repo-wide and will surface other
>   EPICs' work, which is not yours to brief. List an out-of-EPIC item under `surprises` if it looks urgent, never here.
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

**Before writing any gate, dependency, or scope claim into a brief or an EPIC
body, read that claim in the issue's own text — its Dependencies block, its
Scope section, and its Out-of-scope section. A scanner's summary of a gate is
not the gate.**

Subagent scan reports are reliable about *whether* something is worth looking
at and unreliable about *why*. Measured on `matsengrp/phyz` 2026-09-03, three
times in one session, each time from relaying a summary instead of opening the
issue:

- A brief said an issue was unblocked because a landscape study had completed.
  Its own Dependencies block named three different items (all closed, so the
  conclusion held and the route was wrong).
- An issue was reported as "needs re-scoping before it is worth a slot." Its
  body had already *been* re-scoped the previous day — retitled, with the dead
  items struck. The summary described completed work as outstanding. That claim
  reached two EPIC bodies before being caught.
- A brief put "re-verify this label first" at the top as a precondition. The
  issue's **Out-of-scope** section excluded exactly that work, so a worker
  reading both would have hit a contradiction. Reading Dependencies alone would
  not have caught it — the conflict was only visible in Out-of-scope.

The third case is why the rule names three sections rather than one. The
resolution there is also the general one: when a needed precondition is
genuinely out of an issue's scope, **file it as its own issue and let both run**
rather than amending the scope to admit it or asking a worker to ignore the
boundary.

Note the failures all *survived* — the conclusions were right and the reasoning
was wrong — which is exactly why nothing tripped. A wrong route that reaches a
right answer teaches a worker to trust the wrong route.


Run this before proposing any issue as ready to spawn, not after — once a branch exists the cost of a collision is sunk.

1. For every currently open, unassigned (or about-to-be-spawned) issue, read its body against every other such issue's body and ask: can both land as currently written, in either order?
   Or does one delete, rename, or restructure something the other's patch depends on — a dependency-direction conflict that no ordering fixes, because one issue has to change shape?
   (Concrete shape: one issue deletes a function a sibling issue is patching; one issue's test assertion contradicts a value a sibling issue is about to change.)
2. Record findings where a future spawn will see them: a note in the EPIC body's dashboard, and — if either issue is about to be handed off as spawn intent (Step 6) — the warning goes directly into that issue's brief.

**Completion criterion**: every pair of currently-open, about-to-be-spawned issues has been read against each other for dependency direction, not just for whether one references the other's issue number.

### Step 4b: File-overlap collision detection

This is a distinct analysis from 4a, not a second pass over the same reading — it automates what has so far been a manual capability: extracting file paths from issue bodies and building an overlap matrix.

1. From every currently open, unassigned (or about-to-be-spawned) issue body, extract mentioned file/module paths — code blocks, a "Files:" section, or explicit paths in prose. **A path extracted this way is a candidate, not a location** — resolve it before recording it as anything else.
2. Build an overlap matrix: any two open issues naming the same file or module are a candidate collision.
3. For each overlap, this is not the same question as 4a's — file overlap alone (two issues touching the same file, in disjoint regions or compatible ways) is not itself a dependency-direction conflict, and clearing a pair on dependency grounds in 4a does not clear it here.
   Confirm whether the overlapping regions can coexist or whether one issue's edit invalidates the other's, even absent any direction conflict.
4. Record findings the same way as 4a: a dashboard note, and a brief warning for any issue about to be handed off (Step 6).

Resolve with `find`/`grep -r` against the actual repo before recording an overlap or its absence — a filename with no directory component is unresolved, and two issues naming `common.py` are not evidence of overlap by themselves.
This matters most in a repo with per-experiment `scripts/` copies, where the same filename can resolve to unrelated files, and the same symbol name can appear in several of them meaning different things.
Verification shell calls should carry their own working directory — `cd` inside the same command, or an absolute path — rather than relying on a `cd` from an earlier call persisting into this one.
When *naming* a file in a brief or message to another session (Step 6, or any epic→conductor relay), include the directory — a bare filename plus line number is not an address, and pushing resolution onto the receiver costs a round trip that including the directory up front would have avoided.

**Completion criterion**: every pair of currently-open, about-to-be-spawned issues naming an overlapping file/module has an explicit overlap verdict — "compatible" or "conflicts, because ___" — not merely "no dependency conflict found in 4a."
**A pair cleared in 4a is not thereby cleared here** — this is the check that "No collisions" wrongly skipped when two issues sharing a dependency-independent EPIC both edited `experiments/2026-08-27-2051-heavy-v2-stated-estimator/scripts/rescore_cell.py`.

**Report a 4b finding in 4b's vocabulary.**
A collision governs *sequencing*: "these two cannot be in flight at once, or the second silently re-baselines against something the first moved."
A dependency governs *eligibility*: "this one cannot start."
Those two strings are not interchangeable, and the conductor acts on them differently — the first is an ordering input it can schedule around, the second is a gate it must respect.
Never hand a 4b finding to the conductor, or write one into a brief, as "queues behind" or "blocked on"; that is 4a's phrasing and it reads as ineligibility.
The failure this prevents: a shared edit site in one function was written up as "#2123 queues behind #2101", which was never true — #2123's own body listed #2101 under *Out of scope* — and the false gate then propagated through a spawn brief and two logs unread.
Note the guard was already structural and both steps had in fact been run; it was the **write-up** that substituted the category, which is why this rule lives at the point of reporting rather than at the point of analysis.

### Step 5: Prune and update the EPIC body

Follow the **EPIC body update pattern** below.
Do this whenever findings come in, items complete, or new work starts — not on a fixed cadence.

**Prune on every write, not on a separate sweep.**
The observed failure mode is accretion, not staleness: a body cut from 524 to 146 lines had almost nothing stale in what was removed — it was cut because the work had *finished* (checked-off phases, a fully-ticked table) and kept getting appended around instead of trimmed.
Whenever you touch a body to add a finding or check a box, also look for sections whose work is now fully complete and cut them in the same edit, rather than letting them ride to the next dedicated cleanup.

**Before writing a correction, a retraction, or a unification of two findings**, apply the three checks in `PROSE-DISCIPLINE.md` ("Before writing a correction or a unification").

**Identified work needs an issue number, or it is not tracked.**
Every mechanism downstream of this skill keys on issue numbers: Group A's scanner resolves each dashboard item with `gh issue view <N>`, spawn intent files are named `<N>.md`, the conductor's dashboard maps slots to issues, and blocker sets are lists of numbers.
Work described only in body prose — "root-causing this is not yet filed", "a follow-up should re-examine X" — reads as durable to whoever writes it and resolves to nothing for whoever reads it next.
File it, and if that has to wait, put it in the Open work checklist as an explicit `**UNFILED**` bullet so the dashboard shows a gap rather than hiding one.
This is the same shape as an under-specified citation (Step 4b): it looks like an address and isn't.

**Fleet *policy* is not fleet state, and this rule does not cover it — ask for it and write it down.**
The prohibition above is about facts that *rot*: which clone holds which issue, what is running where.
It says nothing about how the fleet *works* — who lands PRs, whether CI exists, what a spawn prompt standardly instructs, what the reclaim rules are.
That second category is stable, and an epic session that conflates the two ends up refusing to learn it.
Measured 2026-09-02: the epic told its user "the bottleneck is the merge queue" and recommended they merge three PRs by hand, because it did not know workers land their own PRs via `/bip-pr-land` — a standing instruction in every spawn prompt, unchanged for the whole session.
The tell is the tense: *"cedar is running #2143"* rots by morning; *"workers land their own PRs"* does not.
Ask the conductor for policy once, keep it, and re-ask only when a spawn prompt's shape visibly changes.

**Type every collision claim as VERIFIED or FORECAST when you send it.**
A claim resolved against a landed diff and a claim about where a not-yet-written edit will land are different objects, and sending them in the same voice makes the receiver treat both as constraints.
Measured across one session: six collision claims, five of which dissolved on inspection — p2159/#2162 hunks were disjoint, #2157 never made the `src/` edit its Phase 2 predicted, #2160-R4 and #2162 turned out to exercise different code paths, and two `.gitignore` races were in different files.
The one that survived had a diff to check against.
Same-file and same-struct reasoning is a weak signal that reads as a strong one, and the conductor can resolve it in a single command at spawn time — so the useful division is that you raise the semantic question and mark your confidence, rather than spending a round-trip pre-verifying each one.
Note the inverse trap too: two modules can export the same type name (`SearchResult` in both `tree_search` and `search_dispatch`), so a name match is not a collision either.

**A number that crosses a session boundary loses its provenance unless you attach it.**
Re-derive anything load-bearing that arrived from a conductor, a worker, or your own earlier round, and quote the command.
Measured the same day: an inherited `12M` for a preserved artifact set was actually `6.0M` (a `find` list that pulled in a sibling directory), a subagent's per-topology timing was ~20% high, and an "it costs nothing to capture" assertion turned out to describe a column the harness was discarding before it reached disk.
None changed a conclusion; all three would have shipped into an issue body unchallenged.
See `PROSE-DISCIPLINE.md`'s "Before citing a measurement" for the issue-body form of this — the cross-session form is the same failure with a different surface.

**Fleet state is derived, never authored — don't let it back into the body.**
Which clone holds which issue, which slots are free, which tmux windows are live: recompute this from `git`/`tmux`/`gh` (that's `/bip-conductor`'s Step 5 dashboard) whenever you need it.
Never write it into an EPIC body or a continuation doc — it goes stale within a day and there is no mechanism to notice, which is exactly the failure a hand-maintained clone table caused when it went stale twice in one day.

**The rule is about the claim, not the artifact, and reading it as artifact-only is the way it gets obeyed and defeated at once.**
Keeping a clone table out of the EPIC body while telling the user "cedar is working on it" repeats the same stale fact in the place they will actually act on, and skips the recompute that would have caught it — the body is protected and the user is not.
Three instances in one session on `matsengrp/phyz`, all inherited from an earlier message rather than a command: a PR reported `DIRTY` after it had been rebased clean, a build target reported red after a merge turned it green, and a slot reported working after its issue had closed and merged.
The tell they share is grammatical: **a claim in the present continuous about what a slot, host, PR, or target is doing *right now* is fleet state**, whoever it is addressed to and whether or not it is being written down.
Ask the conductor or run the command; do not carry it forward from a message, including your own earlier one.
The asymmetry is the point — the conductor can re-derive all of this in a single command and this session structurally cannot see any of it, so the cost of asking is one message and the cost of guessing is a confident wrong statement to the person deciding what to do next.
Handoff artifacts (spawn intent, unfiled drafts) are the opposite category: authored deliberately, to a durable path outside git, precisely so they survive clone churn — pruning them (Step 6, below) is about cleaning up finished authored state, not about avoiding writing derived state down in the first place.

**A user decision may be written into the EPIC body only against a `FINAL` shape.**
When the session talking to the user is the conductor, not this one, the decision arrives here as a relay — see `/bip-conductor`'s "Decision relays: PROVISIONAL and FINAL" — and every relay is marked one or the other.
A `PROVISIONAL` relay is discussion in progress, not yet a settled decision; write nothing into the body from it beyond noting that a decision is pending, and check `.epic-decisions.md` in the conductor cwd (see `/bip-conductor`'s ".epic-decisions.md: the durable fleet-decision log") for the eventual `FINAL` entry before composing the body edit.
Only a `FINAL` relay — or a decision reached directly in this session — authorizes the write, and the body text should restate the decision's substance, not a paraphrase of the sentiment that led to it: "burn everything down and make it right" said in conversation is not "make parity diverges permanently" written into an EPIC body: the paraphrase is a second-hand summary as if it were the specification itself.
If the deciding session's shape is unclear, treat it as `PROVISIONAL` and ask before writing.

**An unmarked relay is not a decision, and not a cue to go ask the user yourself — reply asking the relaying session for the marked form.**
Sender-side discipline says nothing goes unmarked, but senders forget, and the thing they forget to mark is rarely a formal decision: it is an informal mention of what the user is *ready for* ("they're done thinking, they want to spawn the batch"), which reads enough like a decision to be tempting and enough unlike one to distrust.
The right response is neither of the two obvious ones.
Acting on it treats a peer's summary as the user's approval, which is exactly the substitution the FINAL rule exists to prevent.
Putting the question to the user yourself costs them a duplicate — on 2026-08-30 both sessions asked the same spawn-hold question within the same minute, because the epic could not distinguish "the user is ready for the batch" from "the user lifted these three specific holds," and reached for the user instead of for the sender.
Asking the sender costs one message, resolves the ambiguity at its source, and keeps the anti-substitution rule intact instead of trading it against the user's attention.
This is the rule regardless of whose column the decision sits in: role boundaries determine who *puts* a question to the user, and this determines what to do with an unmarked answer, which is a separate question that survives any reassignment of the first.

### Step 6: Hand spawn intent to the conductor

For each issue judged ready (unblocked per Step 3, no unresolved dependency-direction conflict per Step 4a, no unresolved file-overlap collision per Step 4b), draft the semantic brief — why it matters, scope, any dependency/collision warnings from Step 4a/4b — and write it to:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
mkdir -p "$CLONE_ROOT/.spawn-prompts"
# Use the Write tool to create "$CLONE_ROOT/.spawn-prompts/<N>.md" with the brief
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-epic`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory (see `skills/lib/spawn-intent.sh` in the `bipartite` repo).

**Every brief must open with an `EPIC: <N>` line**, naming the EPIC this
session owns. `/bip-conductor-spawn` requires it, and it is what lets the
conductor count live slots by EPIC on each poll — the one fleet-side view that
catches topic drift early. A brief arriving without it should be queried, not
guessed at, since a guess defeats the check. This is cheap to write and is the
only mechanical guard against the drift "One EPIC at a time" describes.

`clone_root` in `.epic-config.json` is tilde-form — resolve it the same way every `/bip-conductor*` skill does, or the brief silently lands in a literal `~` directory in the cwd instead of the shared intent directory, and the conductor never sees it.
`mkdir -p` because `.spawn-prompts/` won't exist yet on a fresh repo.

**Naming note**: an older, already-live convention in this same directory names files `spawn-<N>.txt` instead.
Both are valid intent — `/bip-conductor-spawn` checks for either (see that skill's "Where the prompt comes from").
Use `<N>.md` for new intent; don't rename existing `spawn-<N>.txt` files you find there, since one may be mid-flight.

This directory lives outside every clone's git (deliberately — it must survive clone churn) and is the handoff point to `/bip-conductor-spawn`, which reads it, checks it against live fleet state, appends the fleet facts only it can see, and executes after user confirmation.

**A brief states what is durable about *its own* issue. It must not restate another issue's gate, hold, or queue status.**
That is the fastest-rotting content a brief can carry, and the conductor supplies it at spawn anyway — the sentence above already assigns it live fleet facts, so a brief that duplicates them is both redundant and the only copy that can go stale.
The failure is quiet — a brief is read once, by a worker with no way to know which of its claims were true only at authoring time.
Write instead the collision constraint in 4b's vocabulary ("cannot run concurrently with `iN`, because both edit X"), which is a property of the two issues and stays true, and leave "is `iN` running right now, and is it approved" to the session that can see the answer.
The same goes for superlatives about queue position: "no 4a gate, no unresolved 4b collision" is durable; "the most spawnable item on the board" is a snapshot wearing a recommendation's clothes.

**The general form is: a brief may only assert what has stopped moving.** Another issue's gate status is the common case; **an unlanded result is the same failure with a longer fuse.** A brief whose motivation cites numbers still in review will not fail at launch — it fails hours in, when a worker discovers the finding it was built on was restructured during review, and the cost lands far from its cause (observed: a PR's skeptic round withdrew the exact result a held brief's redesign was anchored to). **Wait for the merge** — the hold costs hours, briefing costs a worker's day on a question that no longer exists.

**Name the host class for anything with real compute.** A worker reads its issue body and its brief; neither tells it whether the machine it landed on is a shared interactive workstation or a dedicated compute host, and there is no way for it to infer this. Say "run the sweep on an orca, not on `pax`" in as many words. Observed 2026-08-31: a brief omitted it and its worker launched a 391-job sweep locally, taking the shared workstation to load ~11 before killing it — behaving reasonably on the information it had.
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

# Before pushing: check if someone else edited since our pull.
# Both values MUST be non-empty -- see "a check that cannot read fails open" below.
CURRENT_AT=$(gh issue view <number> --json updatedAt -q .updatedAt)
if [ -z "$PULLED_AT" ] || [ -z "$CURRENT_AT" ]; then
  echo "ABORT: could not read updatedAt (network? auth?) — cannot establish safety, not pushing"
elif [ "$PULLED_AT" != "$CURRENT_AT" ]; then
  echo "CONFLICT: Issue was updated since pull ($PULLED_AT → $CURRENT_AT)"
  echo "Re-pull, merge changes, then try again."
  # Stop here — do NOT push
else
  gh issue edit <number> --body-file ISSUE-EPIC-<N>.md
fi
```

**A check that succeeds when it could not read the value it was checking is worse than no check, because it reports a safety it did not establish.**
The emptiness guard above is not defensive padding; without it this snippet **fails open on exactly the failures it exists to survive.**
When `gh` cannot reach the API it prints to stderr, exits non-zero, and leaves the variable empty — so `[ "$PULLED_AT" != "$CURRENT_AT" ]` compares `""` against `""`, concludes nobody else edited, and pushes whatever is in the local file. Observed live: an unreachable `api.github.com` produced two empty timestamps, the guard passed, and the push went ahead against a stale body.

The class is worth recognising beyond this snippet, because four instances turned up in one day on `matsengrp/phyz` and each looked like a working check:
- a harness's "the two arms differ" assertion that included a wall-time field, so it passed unconditionally;
- a sweep's `check_cmd`, which exited 0 whether or not the sweep had finished;
- this conflict guard;
- a fleet cleanup audit whose `dirty=$(git -C "$d" status --porcelain | wc -l)` returns **0** for an unreadable clone — indistinguishable from clean, and gating whether a window gets killed.

The shape is always the same: a *failure to read* is silently rendered as a *reassuring value* — empty string, zero count, exit 0.

**The sibling failure, and it is the more general one: the check works, reports the failure correctly, and the report is discarded by the control flow around it.** Here the guard is not fail-open — it is fail-reported-and-ignored, which looks even safer because you can see the warning in the transcript.

Measured 2026-09-01: a `python3` heredoc doing a string replacement in a shared skill file printed **`MISS`** when its anchor was not found — exactly right — and then exited **0**, because `print()` sets no exit status. The enclosing `... && git add && git commit && git push && echo PUSHED` therefore ran to completion and reported **PUSHED**. The commit contained only a reversion of 29 lines a peer had added, because the working tree had been read after `git fetch` without a pull. Two guards were present and both were defeated by plumbing: the staleness one was never written, and the anchor one wrote to stdout instead of `exit 1`.

So, whenever a check gates something destructive or outward-facing:
- **A diagnostic that does not set exit status is not a guard in a pipeline.** `print("MISS")` must be `sys.exit(1)`, or the `&&` chain is decorative.
- **Verify the mutation happened, not that the command ran.** After a scripted edit, `grep` for the new text before committing; after a commit, `git diff <base> HEAD -- <file>` should be exactly what you intended and nothing else.
- **`git fetch` updates remote-tracking refs and not your working tree.** Editing a shared file after `fetch` alone edits a stale copy, and the resulting commit silently reverts whatever landed upstream in between. Pull. Ask of any guard you write or inherit: **what does this do when the thing it queries is unavailable?** If the answer is "the same as when everything is fine", it is not a guard. Capture into a variable, check the command actually succeeded, and only then compare — and prefer a shape where the failure mode is a loud abort rather than a quiet pass, especially when what follows is destructive or outward-facing.

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
