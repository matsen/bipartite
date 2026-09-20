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
individual step justifies. Measured 2026-09-03: an unscoped
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
It does *consume* a conductor-supplied occupancy table when a conductor handshake succeeds (Step 2 below) — read-only, never fetched by this skill itself. That table has **two** consumers downstream, and reading it as serving only the first is how briefs get written for occupied slots: Step 2 uses it to scope Group A's re-verification, and **Step 6 uses it as a readiness filter**.

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

**The epic/conductor split also has a diagnostic function, easily mistaken for overhead.**
The two roles hold different working sets, so what is invisible from inside one is ordinary from the other — across one measured day (2026-09-04): a shipped default resting on invalidated evidence, a void experiment plan, an incomplete census, a relocated document's missing pointers, a wrong causal story — **every one caught by the other session and none by the author re-reading its own text.**
**This is a property of running two sessions, not a habit to adopt, and it disappears silently if the roles are merged — worth knowing before anyone consolidates them on efficiency grounds.**

Two practices make it work and both are cheap: send raw measurements rather than conclusions, and re-derive a peer's number before acting on it.
**Neither licenses re-narrating the peer's analysis to the user** — that is exactly the duplication `/bip-conductor`'s fleet/topic rule forbids ("consume it as a constraint, log it, and do not re-verify, re-narrate, or re-litigate it").
Re-derive silently and report only the delta: a peer's five-item list that turns out to have nine is worth one line, not a second copy of their reasoning.

A third practice reads as tone and is actually cost: **keep corrections low-ceremony.**
"That framing is wrong, here is why" in one line, no preamble and no apology round, in either direction.
A correction that costs a diplomatic round trip does not get made at the margin, and the marginal ones are where the value was.

## Find the load-bearing conditional — it is usually the claim

**Before shipping a claim, locate the conditional clause its correctness depends on, and ask whether that conditional is actually the claim.** If the sentence is only true *while* / *unless* / *if nothing has* something, the something is the finding and the rest is framing.

⭐ **Why this needs its own check rather than more care: it is the one failure mode where nothing is false, so no check fires.** Its sibling — an instrument confidently answering about a universe that isn't the one asked about — gets caught the first time anyone re-derives from the source. A hedged understatement survives re-derivation, because re-derivation confirms it.

Measured on `matsengrp/superfamily-pcp`, 2026-09-15, both within an hour, both by sessions that had spent the afternoon cataloguing the sibling failure:

- **"`git -C <clone>/vendor/phyz rev-parse HEAD` recovers the run's phyz SHA."** True only until the next run in that clone. `bin/build-phyz.sh` fetches and `checkout --detach origin/<ref>` on **every** run with `cache false`, so the vendor HEAD advances *by construction*. The route was proposed as the better one; it was the more convenient one **answering a different question** — "what did the most recent build use", not "what did this run use".
- **The correction to it: "it reports the current HEAD, which is the run's binary only if nothing has rebuilt since."** Directionally right, materially too weak: it wrote as a *condition* what is the *default behaviour*. The hedge was carrying the entire finding.

➡ **The mechanical form: for each hedge in a claim you are about to ship, ask how often the hedged case actually obtains. If the answer is "always" or "on every run", rewrite the claim as the hedge.** ⚠ Note both examples were produced *inside a correction to the same class of error*. **Naming a failure class does not confer immunity to it**, so this check has to run on the sentence in front of you, not on your general level of caution.

## Write only what you're sure of

**Never write down something you aren't sure about. A gap someone has to dig for is cheaper than a confident wrong claim** — the gap costs one command, the claim costs a wrong action plus the round to undo it. This is also why brevity matters: every line in a durable artifact is a claim someone will act on. Five costly failures on `matsengrp/phyz`, 2026-09-10, all confidently written, none arithmetic:

- **Fleet state: omit it, or timestamp and attribute it.** *"spawned into `<clone>`"* was a peer's stated intent rather than a check — then corrected to "not spawned", then false again minutes later when the spawn happened. **A peer's intent is not a fact**: "I'm spawning X now" is not "X is spawned." The fix is no fact, not a fresher one — point at the conductor. And what makes the conductor's own fleet writes safe is **artifact type, not authority**: an append-only timestamped entry records an observation and cannot rot, while a state document read as *now* always can.
- **A correct verdict resting on a false premise still fails.** *"safe to clear"* was the right call about a registered worktree — but it came from a `test -d .git` false negative (in a worktree `.git` is a *file*, not a directory) and was then defended with *"the commits are on the remote"*, which rested on a **stale remote-tracking ref** that `git ls-remote` did not support. The verdict held; the premise would have let someone later delete the two local branches actually holding those commits.
- **A TODO or MUST-FIX carries its discharge condition, or is struck the moment it lands.** *"MUST FIX `<file:line>`"* — already fixed, six lines above that same body's warning that the risk had inverted toward fixing correct instances. It rots into a trap *because the work got done*: the one failure mode that worsens with good practice.
- **State each item's disposition** — done / correct-by-design / outstanding. *"left for #2455"* named three already-correct citations; *"is filed separately; do not fold it back here"* described an issue that was never filed. A bare pointer defaults to "outstanding" in the reader's mind.

## ⛔ The epic's own claims are the ones that never get checked — and its most expensive error is answering a proxy

⭐ **Every rule below is an error THIS ROLE committed, not one it caught in a worker.** That is the only reason to trust them over the general advice elsewhere in this file — **a reader who does not know it cannot tell which rules are hard-won.** Four, all from one day on `matsengrp/phyz` (2026-09-15).

### Never report a sub-question's answer as the programme's answer

**An EPIC's core question and the issue you just closed are different objects, and the second is far more available.** Measured: the epic told its user *"the core question is answered and closed"* on the strength of two landed issues that had, between them, identified a real defect and a real mechanism. **A skeptic refuted it in one pass.** The arithmetic: the two results closed **67.6%** of a *stage* gap at **one cluster**, and `0%` of the programme's headline gap — which was **78.5 nat**, not the **34.7** the epic had computed, because it compared against the other engine's *descent-only* figure rather than its search endpoint.

⚠ **The EPIC body itself had already scoped the result out**, calling it *"a user-facing quality defect independent of any [cross-engine] comparison."* **The epic read its own text and substituted anyway.**

➡ **Before writing that anything is answered: quote the EPIC's own question, then state which measured quantity closes it and by how much.** If the answer is a fraction of one stage at one fixture, say that instead. ⛔ **And check what the number you are comparing against actually IS** — a descent value and an endpoint value differ by the whole of the other engine's outer loop.

### Run a skeptic on your OWN artifacts, not only on workers'

**The "skeptic before filing" rule reads as being about other people's claims, and the epic exempted itself from it twice in one day.** It filed an issue with no skeptic pass — the pass, run afterwards, found the issue's central arithmetic rested on non-matched baselines and that the quantity **inverted** under a defensible change of one of them. And it claimed closure with no pass at all.

➡ **Two additions, and they are where the epic's own errors actually came from: before any claim that a question is ANSWERED, and before filing an issue the epic wrote itself.** ⭐ **Two fresh skeptics on overlapping material found disjoint defects two hours apart**, which is also the argument against a standing skeptic: the value is in having no stake and no inherited framing, and a persistent one accumulates the epic's priors until its agreement stops being evidence.

### ⛔ Attribution is the half of a claim nobody tests — and if every error favours you, that is a signal

**Five attribution inversions in one day, every one in the flattering direction.** The epic credited a worker with catching the epic's generalisation (the worker had written it, and caught its own); then claimed the same sentence as its own (it had not written it, only propagated it); then accepted credit for a conductor's correct prediction that the epic had in fact argued against and killed.

⚠ **The mechanism is that an attribution rides inside a message whose SUBSTANCE has already been verified, so it inherits a credibility nothing ever tested.** It is never the part anyone checks, because it is never the part in dispute.

⛔ **The set is asymmetric in a way that tells you where to look: of the five, the FOUR that INFLATED someone were each challenged by someone else. The ONE that ABSOLVED someone survived until its beneficiary objected to it.** ⚠ **That is structural, not luck.** An inflation is visible to the party who did the work and did not get the credit, so it has a natural challenger. **An absolution is visible to nobody but the person it lets off** — and they are the party least motivated to raise it. ➡ **So the correction that reflects badly on you is the only half of this problem the other party cannot fix, and it does not surface unless you volunteer it.**

➡ **The log answers it in one command** — pair each occurrence with its nearest preceding attribution header and read which session's entry it sits under. **Do that before repeating who found something.** ⭐ **And treat the DIRECTION as diagnostic: five for five favouring the same party is not chance.** Volunteering the correction that reflects badly on you is the only half of this problem the other party cannot fix.

### Before escalating: does the answer change what anyone DOES?

**A correctly-identified open question is not automatically an escalation.** The epic marked a taxonomy question *"the user's to rule"* and routed it up. The user declined and corrected the framing: *"the formalism may be getting away from the science here … I don't think this question is load-bearing?"* **They were right — nothing operational read the disputed count, and the relevant doc already published both values with reasons.**

➡ **Apply the test before spending the user's attention, and apply it to your own routing even when a peer has already accepted it.** ⚠ The failure is not asking a bad question; it is failing to notice that **both answers lead to the same action**, which makes the question decoration.

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
git pull --ff-only origin main || echo "PULL FAILED — you are on a stale tree; fix this before reading anything"
cat .epic-config.json
```

**Check that the pull succeeded, and never suppress its stderr.** In a pooled-clone fleet this aborts routinely rather than rarely: untracked artifacts left by earlier sessions collide with paths a later PR has since landed as *tracked*, and `git pull` refuses with *"untracked working tree files would be overwritten by merge"*. **Behind `2>/dev/null` that is silent, and every subsequent read is of a stale tree.** Move the blocker aside rather than deleting it — it may be unpreserved work — then pull again and confirm `git rev-list --count HEAD..origin/main` is `0`.

Measured 2026-09-09: four commits behind for hours, blocked by an untracked `experiments/2026-09-08-enumerate-n100-neighborhood-2396/` that a merged PR had since landed as tracked. The stale tree produced a confident **"the cited passage does not exist"** — repeated across four independent greps, which felt like corroboration and was one error — and that claim was escalated to a peer session, used to block a live worker's brief, and contributed to halting the fleet. **The passage existed, and so did the artifact.** Note this instruction sat 384 lines above a bullet that already says *"check the command actually succeeded"* and *"prefer a shape where the failure mode is a loud abort rather than a quiet pass"*; the rule was present and the adjacent instruction did not follow it. *(Sunset: if no epic session hits a failed pull in a month, cut this to one line.)*

**Re-run that pull at the top of every fresh scan cycle, not just at cold start.**
An epic session only ever *reads* remote state — every `gh issue view` and `gh pr view` is correct regardless of how old the working tree is — so nothing in the normal loop ever forces a pull, and the tree rots silently while every answer stays right.
Measured 2026-09-01: both the epic and the conductor sat four merges behind on `main` for hours, each with a clean tree on the correct branch, and neither noticed until the other described it. `git fetch` does not help: it updates remote-tracking refs, not the working tree.
This matters here specifically because reading a stale file and writing it back is how an EPIC body or a skill acquires a reverted diff — see `/bip-kaizen` Step 5 on the shared working tree.

**Read every TRACKED subdirectory `CLAUDE.md` in the project repo — they are NOT auto-loaded.** Only the repo-root `CLAUDE.md` and the user's global one land in context automatically. A subdirectory `CLAUDE.md` loads when a session *works in* that directory, and the epic role works from the repo root reading artifacts by path — so it systematically never loads them while reading files underneath them all day. Enumerate at session start:

```sh
git ls-files | grep 'CLAUDE\.md$'
```

**Use `git ls-files`, not `find`. Tracked-ness is the predicate for "the repo's guidance"; a file merely NAMED `CLAUDE.md` is not.** Measured on `matsengrp/phyz` 2026-09-11: `find` returns four hits totalling 137 KB of which one is real — a vendored copy inside a pixi env, and a stale nested clone under `_ignore/` whose 56 KB root `CLAUDE.md` opens with the identical "guidance for the phyz repository" header and is indistinguishable from the live file on inspection. `git ls-files` returns exactly the two real ones, because the index already filters to what the repo considers its own and keeps doing so as new files appear.

The rule's own evidence: an entire session read files under `experiments/` with `experiments/CLAUDE.md` never loaded, and rediscovered the expensive way a trap that file documents — that `results/` is gitignored *with per-file negations* to commit reference artifacts, whose misresolution produced a false "input absent" relayed onward as verified. **The root file's existing "see `experiments/CLAUDE.md`" pointer did not cause it to be read: a pointer is not a mechanism.**

**What this rule does NOT cover, stated because the same session assumed it did.** The costliest error that night — publishing a ratio between two incommensurable quantities — was not in any `CLAUDE.md`. The answer sat in `README.md` of a ~9000-line experiment write-up, and reading every tracked `CLAUDE.md` would not have caught it on its best day. **So: before claiming a measured number is new, grep the experiment's own README for the number itself.** Cheap, mechanical, and specific to a repo where results live in enormous prose READMEs rather than structured artifacts. The two rules are siblings, not one rule; do not let this one stand in for that one.

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
`sort:updated-desc --limit 20` reaches back only as far as the last 20 touched issues, which on an active repo is a couple of days: measured 2026-08-28, that window spanned 08-26 to 08-28 and saw 20 of 65 open issues.
The window compresses further as activity rises, so an issue that is neither on an EPIC dashboard nor recently touched is reachable by neither group.
Coverage is Group A's job plus a periodic audit for issues no EPIC references (see `matsengrp/phyz`#2093 for the audit script and the shape of that gap); Group B only answers "what moved lately".

**Never ask the user a question you could answer yourself, and never ask a sequencing or staging one at all.** Issue/PR status is a `gh` query — verify, then present facts. Order, placement, timing, and whether an adopted priority rule still binds are for this session and the conductor to settle between them; `/bip-conductor`'s "Arbitration" section states the conductor's half and applies symmetrically here — escalate only what it names (data loss, a clobbered checkout, two slots owning one deliverable) or a genuinely scientific question about whether work is worth doing. Measured 2026-09-03: asking whether an EPIC body's "run these after the default questions" meant *sequenced* or merely *lower priority when slots are scarce* returned "I have no idea. I'm guiding this at a high level, and I'd love it if you and the Epic agent would sort these things out." **That an adopted EPIC-body decision is being revisited does not make it the user's to re-decide, when what it governs is order.**

### Step 3: Reconcile

Compose the reconciliation from the two reports — do not paste subagent prose verbatim.

- Never present an issue as "ready to spawn" or "needs action" without confirming it's still OPEN on GitHub
- Flag anything merged/closed that an EPIC body doesn't reflect yet
- If either group's report has zero `surprises` and zero `changes_since_baseline`, send a follow-up to that subagent with a narrower question before concluding "nothing changed there."

This reconciliation does not scan clone/tmux occupancy itself — that's `/bip-conductor`'s dashboard — but Step 2's handshake table is in hand by now, and Step 6 requires it.

### Step 4a: Dependency-direction detection

**Before writing any gate, dependency, or scope claim into a brief or an EPIC
body, read that claim in the issue's own text — its Dependencies block, its
Scope section, its Out-of-scope section, and its Success criteria. A scanner's
summary of a gate is not the gate.**

**Those four are a FLOOR, not a ceiling: for an issue body under ~200 lines,
read it whole.** A section list fails the same way at four entries as at three,
just later — the underlying error is *a partial source read presented as a
complete one*, which from outside is indistinguishable from a correct one.
Measured 2026-09-08 on a 133-line issue: all three then-listed sections were
read, and the requirement that mattered was in **Success criteria** — first
producing an invented scope claim, then an over-correction denying the
requirement existed. **Falsification and sunset: if reading whole issues never
changes a briefing decision over a month, cut this back to the list.**

Subagent scan reports are reliable about *whether* something is worth looking
at and unreliable about *why*. Measured 2026-09-03, three
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

⛔ **4b COVERS FILED ISSUES AND STRUCTURALLY CANNOT COVER IN-FLIGHT WORK. Say so when you report it, because "no collisions" otherwise claims a coverage you do not have.** The overlap set that matters is *issues plus live branches*, and the second half is invisible from an epic session:

- **A worker's branch lives in its own clone and is not pushed until the PR opens.** With `shared_filesystem: false` each clone has its own working tree, so a branch under active edit exists nowhere this session can read. Measured on `matsengrp/phyz` 2026-09-14: 4b was run across six filed issues and reported one two-way collision on `tests/test_param_help_guard.zig`; the collision was **three-way**, because a live worker's unmerged local branch was restructuring the same file. Confirmed afterwards that `git ls-remote --heads origin` returns nothing for that branch — it was not reachable from here by any command.
- **And remote branches are not a usable substitute.** Under squash-merge every historical branch stays permanently ahead of `main`, so "ahead of main" does not discriminate live from long-dead: that repo's clone-side view is **847** remote-tracking refs (`git branch -r`, 2026-09-14), against **819** live heads on the remote (`git ls-remote --heads`) — the gap being stale refs to deleted branches, 23 of which `git remote prune --dry-run` would drop. **So the view an epic session actually has is less discriminating than the remote itself.** Sampled 400 of those refs: all 400 carry commits absent from `origin/main` (`git rev-list --count origin/main..<ref> > 0`). An independent later check by a different predicate — ancestry, `not an ancestor of origin/main` — found 40 of 40. Intersecting against them is noise, not coverage.

➡ **So report a 4b verdict as scoped — "collisions among the filed issues" — and hand live-branch intersection to `/bip-conductor`. This is not a courtesy hand-off; it is the only implementation.** There is no cheap remote-side approximation, because the 847-branch finding above removes the obvious one, and the early-stage case is worse still: a worker's collision often lives in **uncommitted working-tree changes**, visible only in `git status` on a machine this session cannot read. The conductor inventories clones and can see both; this session can see neither. The conductor is also the only party that can catch the inverse case, an issue whose file list collides with a branch nobody has filed an issue for.

⚠ **The same asymmetry applies to a stale slot reading as occupied.** Measured the same day: a preserved `.epic-status.json` survived its window's closure and made an idle clone read as occupied, and the occupancy loop skipped it **silently** — fail-open in the direction of "someone else has this." That is fleet state and the conductor's to sweep, but an epic session should know its occupancy input can be wrong in the direction that suppresses spawns, not only in the direction that permits a collision.

**4b is a check on PATHS, and there is a collision it is structurally blind to: one where an issue changes what an artifact another issue CITES means.** Two issues can write entirely disjoint files and still collide, because the collision is on *meaning* rather than on location.

Worked instance (`matsengrp/phyz`, 2026-09-06): a follow-up proposed fixing a harness script and re-running the sweep that script had produced. Its write targets intersected nothing. But the table it would regenerate was the **read-only baseline** three other issues were citing for a landed verdict, and re-running it would have moved 349 of 945 values under those citations. No path overlap; a real collision.

**There is no cheap detector for this and you should not pretend otherwise.** A file list cannot see it — resolving it needs someone who knows what each artifact is *for*, which is a topic-level read at brief time. Treat that as one more reason briefs go through the epic session rather than being generated from diffs. Two questions worth asking of any issue that regenerates or replaces a committed artifact:
- **Does anything cite this artifact as evidence?** If so, changing it re-baselines that evidence whether or not any path overlaps.
- **Is the artifact's value that it is CURRENT, or that it is the HISTORICAL RECORD of what a landed result was computed from?** If the second, "fix and regenerate" is the wrong remedy even when the fix is correct.

⭐ **But ONE sub-case of it DOES have a cheap detector, and the paragraph above should not be read as covering it: when the shared thing is a NAMED SYMBOL rather than an artifact's meaning, the import graph is greppable in one command.** An issue that edits a **shared definition** — a constant, a params list, a fixture path — has a footprint equal to that symbol's CONSUMER SET, not to the file it is defined in. ➡ **`git grep -ln <SYMBOL> -- <subtree>`, then read the hits.**

Measured `matsengrp/phyz` 2026-09-19/20, **three instances in one evening, and a path-overlap check reported all three as disjoint — correctly, which is the point.** An issue proposed adding one entry to `MATCHED_PARAMS`, a list defined in one experiment's `common.py` and declared a two-file footprint by its own body:
- A second experiment's `common2830.py` **imports** that list rather than copying it, then appends a value the edit would ALSO add — giving a silently duplicated key. ⚠ **Verified that the tool accepts the duplicate at exit 0 with no warning**, so nothing fails loudly; what breaks is that the key then has two definitions that can drift.
- A LIVE worker's unlanded branch imports it and additionally **partitions** it by string prefix, so an added entry lands in one partition or the other by its name, with no membership check.
- The grep returned **26 tracked consumers**: **17 code files across 11 directories plus 9 READMEs that state the value in prose.** ⭐ **Split the count by kind — the prose hits go stale on the edit exactly like the code hits, but they are different work, and "26 consumers" tells a worker one kind where "17 code plus 9 prose" tells it two.**

⚠ **The tell that generalises: an issue whose body names a file list, when what it is really editing is a NAME that other files read.** ⛔ **And a count taken this way is itself a forecast** — the consumer set grows with every new experiment, and a live branch adds hits the grep on `main` cannot see. **State the count with its date and its command, and tell the worker to re-run it at branch time.**

⛔ **AN ISSUE BODY IS A FORECAST OF A FOOTPRINT; A BRANCH DIFF IS THE FOOTPRINT. Never use the forecast when the fact is available.** Step 1 above tells you how to read an issue body, and says a path extracted that way is a candidate; it does not say what to do once the branch exists. Once it does, `git diff --name-only <main>...<branch>` **is** the answer and the body is superseded. ⚠ **Using the body anyway is Step 5's FORECAST delivered in VERIFIED's voice — inside a brief, where nothing types it.**

Measured `matsengrp/phyz` 2026-09-19, **six claims in one afternoon across four sessions — epic, conductor and two workers — all on the same basis**, and ⚠ **the count rose twice during the writing of this very rule, each time because someone went back to an artifact they had already cited**: two PRs each reported "zero overlap" with the other and shared **five** files (`build.zig`, `CLAUDE.md`, two `docs/ml/*.md`, `experiments/README.md`), one producing a real content conflict; and a brief recorded a live neighbour as *"`experiments/` plus `src/ml/`"* when that branch also edited **both** of the briefed issue's primary files. ⛔ **One did not even use the neighbour's issue body — it used a memory of what the issue was ABOUT, which is a forecast of a forecast.** ⚠ **And the two EARLIEST were the two parties whose job this is**: a spawn brief's bare *"Parallel-safe against #N, which is live. Checked at spawn time"*, and the conductor's own spawn prompt concluding *"✅ zero overlap with your two files"* — stated as a LIVE-BRANCH check, the one form nobody downstream re-derives. ⭐ **The worker repeated that verdict back in its own gate request rather than acting on the hedge printed directly beneath it**, which is this section's hedge-as-ceremony evidence and it sits in the same file as the verdict. ⭐ The generator is ordinary and invisible from inside: an issue named *"a CLV fix"* files under `src/ml/`, and **any behaviour change to a compared axis also updates the docs that record the comparison.** ➡ **Docs, `build.zig`, a shared README and a registry TSV are where two slots on one area actually collide — and where a conflict is least likely to be noticed, because nobody re-reads a doc merge.**

➡ **So a brief states its own issue's footprint and stops there** — *"this issue touches A, B, C; ask the conductor whether that intersects live work"* — with **no parallel-safety verdict, not even a hedged one**. A verdict is frozen at authoring time, about branches this session cannot read, in a document the worker treats as authoritative. ⚠ **A hedge underneath a confident verdict is a weak instrument: the worker reads the verdict and treats the hedge as ceremony.** In the instance above the hedge was present and correct, and it was the **conductor** who acted on it, not the worker it was written for. Removing the verdict is what makes the hedge do work.

⚠ **What the conductor's side of the hand-off actually IS, since a 4b reader cannot otherwise tell whether it was done**: the intersection itself — `git diff --name-only <main>...<branch>` for every live branch, intersected, plus `git merge-tree --write-tree` on each pair — not a restatement of either side's file list. ⛔ **And a clean auto-merge is not a correct merge.** Same day, same repo: two PRs re-pinned **different rows** of one `knob-correspondence.md`, so git merged them silently and cleanly, and whether the citations still resolved was decided by `make check-knob-citations` — which nothing about the merge prompts you to run. ⭐ **When two branches edit one machine-checked document, the checker is the arbiter and the merge result is not evidence.**

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

**Before writing a correction, a retraction, or a unification of two findings**, apply the three checks in `EVIDENCE-DISCIPLINE.md` ("Before writing a correction or a unification").

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
See `EVIDENCE-DISCIPLINE.md`'s "Before citing a measurement" for the issue-body form of this — the cross-session form is the same failure with a different surface.

**Fleet state is derived, never authored — don't let it back into the body.**
Which clone holds which issue, which slots are free, which tmux windows are live: recompute this from `git`/`tmux`/`gh` (that's `/bip-conductor`'s Step 5 dashboard) whenever you need it.
Never write it into an EPIC body or a continuation doc — it goes stale within a day and there is no mechanism to notice, which is exactly the failure a hand-maintained clone table caused when it went stale twice in one day.

**The rule is about the claim, not the artifact, and reading it as artifact-only is the way it gets obeyed and defeated at once.**
Keeping a clone table out of the EPIC body while telling the user "cedar is working on it" repeats the same stale fact in the place they will actually act on, and skips the recompute that would have caught it — the body is protected and the user is not.
Three instances in one session, all inherited from an earlier message rather than a command: a PR reported `DIRTY` after it had been rebased clean, a build target reported red after a merge turned it green, and a slot reported working after its issue had closed and merged.
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

For each issue judged ready — unblocked per Step 3, no unresolved dependency-direction conflict per Step 4a, no unresolved file-overlap collision per Step 4b, **and holding no live slot in the conductor's occupancy table** — draft the semantic brief — why it matters, scope, any dependency/collision warnings from Step 4a/4b — and write it to:

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

**Intersect the candidate set with the occupancy table as a discrete step, and state the check in the handoff message.** Group B's `action_candidates` are GitHub-derived and structurally fleet-blind — it can drop an issue with an open PR or a live branch, and cannot see a clone at all. Applying the filter is this session's job, and *having asked for the table early does not discharge it*: measured 2026-09-03, a session holding a GitHub-verified table from its own first message went on to write intent for two occupied issues and list a third as ready — 3 of 4 candidates. Write "checked against conductor table of HH:MM" in the handoff so the check is visible rather than assumed, and re-ask if that table is older than a poll cycle. Note that **no downstream guard catches this in clone mode**: `/bip-conductor-spawn`'s Step 1 check keys on worktree-mode names (`issue-<N>`, window `<N>-issue-<N>`), and `bip spawn`'s refusal keys on *directory* occupancy — so the same issue spawned into a second idle clone is refused by nothing.

`clone_root` in `.epic-config.json` is tilde-form — resolve it the same way every `/bip-conductor*` skill does, or the brief silently lands in a literal `~` directory in the cwd instead of the shared intent directory, and the conductor never sees it.
`mkdir -p` because `.spawn-prompts/` won't exist yet on a fresh repo.

**Naming note**: an older, already-live convention in this same directory names files `spawn-<N>.txt` instead.
Both are valid intent — `/bip-conductor-spawn` checks for either (see that skill's "Where the prompt comes from").
Use `<N>.md` for new intent; don't rename existing `spawn-<N>.txt` files you find there, since one may be mid-flight.

This directory lives outside every clone's git (deliberately — it must survive clone churn) and is the handoff point to `/bip-conductor-spawn`, which reads it, checks it against live fleet state, appends the fleet facts only it can see, and executes after user confirmation.

**A brief must not restate its own issue's SCOPE — point at the issue's scope section and spend the words on what the issue does not say.**
The worker has the issue in front of it; the brief is the only artifact that carries collisions, host class, framing that changed since filing, and traps. Everything *additive* is what a brief is for; everything that *restates* is a second copy that can drift from the first, and the reviewer — who often holds only the brief — is more exposed to that drift than the worker is.
Measured 2026-09-07: a brief compressed an issue's scope to "four shipped instances across five sites", taken from a *peer session's summary of an audit* rather than the issue, which says "Nine lines, in six files". The brief's Out-of-scope then forbade expanding past those five, which would have **foreclosed work the issue explicitly scoped in** — including one change point the issue states is invisible to the grep the brief recommended. The conductor then enforced the brief's number against a worker that had read the issue; the worker refused, quoted three issue lines back, and was right. Cost: one review round. Cost avoided only because the worker worked from the source.
Open the issue before writing a brief, every time, and prefer *"the change points #NNNN's 'Files to modify' lists"* over any count you have retyped.

**A brief also states what is durable about *its own* issue. It must not restate another issue's gate, hold, or queue status.**
That is the fastest-rotting content a brief can carry, and the conductor supplies it at spawn anyway — the sentence above already assigns it live fleet facts, so a brief that duplicates them is both redundant and the only copy that can go stale.
The failure is quiet — a brief is read once, by a worker with no way to know which of its claims were true only at authoring time.
Write instead the collision constraint in 4b's vocabulary ("cannot run concurrently with `iN`, because both edit X"), which is a property of the two issues and stays true, and leave "is `iN` running right now, and is it approved" to the session that can see the answer.
The same goes for superlatives about queue position: "no 4a gate, no unresolved 4b collision" is durable; "the most spawnable item on the board" is a snapshot wearing a recommendation's clothes.

**A subagent confirms what you asked it to check and silently re-emits the rest of your framing in its own voice — where it comes back reading as a finding.** Measured 2026-09-11: a wrong claim in a continuation file was relayed into a skeptic's brief *as the EPIC body's claim*. The skeptic verified the source half correctly — that half stands and was load-bearing — and repeated the attribution verbatim in its report. That was read as independent confirmation and written into a live worker's spawn prompt as established fact, plus two peer messages. Four artifacts, one unchecked attribution, **zero independent reads of the body**, which turned out never to have made the claim. The subagent never claimed to have checked the attribution and was never asked to: it was the one part of the brief with no reason to be tested. **Separate a claim from its attribution before briefing, and verify the attribution yourself — it is the half that never gets tested.** This is the two-reviewers-is-one-inference rule with a twist: the second reader was *your own framing*, laundered through a subagent and returned with a co-signature.

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

### Step 6b: The design question, and it must be asked of the NULL arm

The epic half of a landing gate is not *"does the artifact say what it claims"* — that is mechanical and belongs to the conductor. It is **"could this experiment have said otherwise?"**
Ask it of **every arm carrying a claim, including the one that returned the boring answer.** Nulls are where it gets skipped, because a null reads as the absence of a result rather than as a result requiring power.

Two distinct ways an arm cannot speak, with different tests and different remedies — do not merge them:
- **vacuous-positive** — the check *could not* have failed, because the arms were constructed to agree. Test: *could this check have failed, given how the arms were constructed?* Remedy: redesign the check.
- **no-power** — the check could have failed and the intervention was genuinely live, but **the pathway it acts through never activated**, so the comparison had nothing to resolve. Test: ***did the mechanism under test actually engage?*** Remedy: measure engagement *before* interpreting the null.

Measured 2026-09-07: an epic session ran the design question on a PR's *positive control*, cleared the PR, and published its *null arm* as a mechanism refutation. The null had no power — the perturbation phase never produced a strict improvement in either arm, so both received identical input. **The engagement counter was a column in the PR's own committed `cells.tsv`, and a durable finding in the EPIC body had already recorded that phase as inert at that cluster size.** Neither reviewer asked. The worker had labelled it correctly; two reviewers relabelled it, in opposite wrong directions, before it was restored.

### Step 6c: A pre-registration and a prior are the same text on opposite sides of a wall

Write your expectation down before the data exists; do not send it to the arm that will produce the data. **The words are identical and the function inverts.** On your side it is an auditable constraint on *you* — if you later reclassify a result to fit, the record catches it. Delivered to the arm, it is an anchor: the arm's classification stops being a measurement and becomes agreement with you, and the agreement proves nothing.

Scored, `matsengrp/phyz` 2026-09-11: the epic pre-registered which of nine test failures would prove to be genuine defects, **reasoning from test names — after having written the correct discriminator two paragraphs earlier** (whether the failing assertion compares a pinned literal or an invariant) **and substituting the name as a proxy for it.** Not a missing rule; a rule it wrote and then did not run. The worker classified two of those the other way on direct evidence and was right; the epic had over-predicted defects twice, and the two that stayed unresolved fell in the buckets it had marked uncertain and invariant. **That was measurable only because the priors never crossed** — had they, the worker would have read the same assertions already knowing what was expected, and both errors would have been invisible.

The corollary is what to send instead: **the scheme goes, the prior stays.** Categories, discriminators, decision rules and known-positive controls are instruments — they make the arm's answer better without deciding it.

**A third thing defaults to the withheld side: adjacent confounded evidence** — a neighbouring arm's result that bears on the question but cannot settle it. Neither scheme nor prior, and its only effect on the receiver is to tilt a judgement it was asked to make independently. Hold it against a release condition written *before* the data, and say the condition out loud: **that is what makes "hold this" different from "bury this."**

### Step 7: Correcting a live worker — the judgment half

When a live worker's scope needs correcting *before* its next natural stopping point, the epic — not the conductor — makes the call: is this change durable (it changes what the worker will produce: scope, target, artifact, gate criterion) or transient (host load, a peer's timing, a dependency that just landed)?

- **Durable**: draft the one-line correction and ask the conductor to deliver it — the conductor `SendMessage`s the worker directly and appends a timestamped, attributed entry to `.epic-worklog.md` in the same step (never edits `lead_guidance` — see `/bip-conductor`'s Conventions section for why the append-only path matters).
  The epic's job stops at drafting the line — route it through the conductor even though `SendMessage` would reach the worker directly from here too.
  A single delivery path is what keeps the message and the `.epic-worklog.md` append from diverging: two senders means a correction can land in the worker's context with no durable record, or a durable record with no delivery.
- **Transient**: message-only is fine; still routes through the conductor, for the same single-delivery-path reason.

A worker correction is often *more* time-sensitive than spawn intent — the worker is actively doing the wrong thing while it sits in a queue.
If the conductor session is addressable right now, `SendMessage` it directly to request immediate delivery, rather than leaving the correction for its next poll cycle — same mechanics and addressability caveats as Step 6's escape hatch.

### Step 7b: An issue body edited mid-flight reaches NOBODY — relay the delta

⛔ **The spawn prompt is frozen at launch. The issue body is not. A worker
holds a copy of the issue read once, at spawn, and never re-reads it.** So an
edit you make to an issue with a live slot on it is **invisible to the only
session that needs it**, and the two artifacts drift silently.

➡ **When you edit an issue that a live worker holds, the delta goes to the
worker as a Step 7 correction. The body edit is the record; the message is the
delivery.** Doing only the first is the same as not doing it.

⚠ **The failure is silent in BOTH directions, which is what makes it hard to
notice:**
- **Requirements added late never run.** Measured 2026-09-14: a
  `/bip-issue-check` pass added a "Required validations" section to an issue
  *after* the conductor had spawned it — **4 of 5 validations never ran**, and
  the gap surfaced only at the worker's own final gate.
- **Corrections landed late read as redundancy.** Same day, same EPIC: a wrong
  scope bullet was fixed in the body, and **two separate workers then flagged
  it as outstanding** — because neither had re-read the issue. ⚠ **A worker
  reporting something already done reads as duplicated effort, not as
  staleness**, so the drift is invisible until someone happens to check.

⭐ **Same class as a frozen spawn prompt carrying a command that has since been
corrected** — three instances in one evening on `matsengrp/phyz`. **The
general shape: any artifact frozen at handoff, whose source keeps moving, needs
its delta pushed rather than pulled.**

**Two cheap habits that close it:**
1. **Before editing an issue, ask whether a slot is live on it** (the conductor
   knows; you do not). If one is, the edit and the relay are one action.
2. ⚠ **If a worker reports something you have already fixed, that is not
   redundancy — it is evidence the drift happened.** Tell it the fix is
   landed, and check whether anything *else* you edited also failed to reach it.

### Step 8: Approving a merge or discharging a gate — post it to the PR

**When you approve a worker's merge, or discharge a gate you set, post a
`🤖`-prefixed comment on the PR as well as messaging the worker.** User
decision, 2026-09-14, `matsengrp/phyz`: PR comments only, `🤖`-prefixed.
**Not `gh pr review --approve`.**

⛔ **Why the message alone is not enough: an epic approval is point-to-point
and leaves no artifact.** Measured twice in one evening — a conductor could
not tell whether a PR carrying an explicit epic re-check gate had landed with
approval or on its own, and had to ask; and separately could not confirm a
ruling had reached the worker at all. **Both answers were benign because it
asked. The failure mode is the one where nobody asks**, and `reviews=0`
surviving as the only artifact is a bad record even when the approval was
real.

**Three conditions, and the first is what makes the practice worth anything:**

1. ⛔ **Post at approval time, not after the merge.** A comment timestamped
   after the merge cannot establish that the approval preceded it, which is
   the whole function. A retroactive one is a *record*, not *evidence* —
   label it as such if you post one.
2. **Say what was approved and on what grounds**, specifically enough that a
   conductor's question is answered without a round trip: the commit, what
   changed, and what you checked.
3. ⛔ **Say explicitly that it is not a code review.** You review *claims*;
   an approve-review asserts the other thing.

⭐ **Why `🤖` rather than an approve object, and it is not decoration.** It
matches `/bip-pr-land`'s existing `🤖 EPIC worklog preserved to …`
convention, and it retires the strongest objection to `--approve` at zero
cost: **a `🤖`-prefixed comment cannot be misread later as the maintainer
having reviewed code they never read.** ⚠ Be clear-eyed that this does **not**
make approvals machine-readable — `gh pr view --json reviews` still returns
`0`. What it fixes is the human-readable record, which is the more valuable
half wherever nothing machine-reads review state (CI disabled, squash-merge,
single maintainer): there, an approve object would gate nothing and function
as a label anyway. **The question is legibility, not enforcement.**

⭐ **The general rule this rests on, which generalizes past approvals: the
durable artifact is the one in the repo, not the one in a pooled clone.**
`/bip-pr-land` deletes `.epic-status.json` and `.epic-worklog.md` from the
clone at its Step 9.5, so **anything recorded only in a slot's own files is
durable exactly until that slot lands.** ➡ **For any ruling a future reader
would need, prefer the PR — body or comment — once a slot is near landing.**
The same reasoning fixed the `issue-lead` terminal ceremony's idempotency
guard the same day: it keyed on a field in a file the landing step removes,
so it could never fire. **A guard, and a record, should reference the thing
it is actually about — not a private flag that is supposed to correlate
with it.**

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

The class is worth recognising beyond this snippet, because four instances turned up in one day and each looked like a working check:
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

⛔ **THE CASE THE SNIPPET ABOVE DOES NOT COVER: WHEN YOUR PUSH SEQUENCE CONTAINS A WRITE OF ITS OWN, THAT WRITE INVALIDATES THE GUARD.** Replacing an EPIC body while archiving the old one to a comment is the common shape, and **posting the comment moves `updatedAt` by construction** — so a correctly-written conflict check fires on a delta you created yourself, with no concurrent editor anywhere in the window.

⚠ **Measured 2026-09-17 on `matsengrp/superfamily-pcp`, and the instructive part is that TWO defects cancelled.** The conductor mandated comment-first ordering — right, for the data-safety reason below — without noticing it guaranteed a false positive. The epic session's guard had dropped its `elif [ "$PULLED_AT" != "$CURRENT_AT" ]` branch, so it failed open and pushed anyway; correctly, as it happened, since the only thing that had moved the timestamp was its own comment 42 seconds earlier. **A broken guard under a broken ordering produced exactly the right outcome, and neither defect was visible in the result.**

⭐ **Hold on to the worse counterfactual: had the guard been written correctly it would have aborted spuriously, been overridden by hand, and plausibly been judged noise.** That is a durable loss. Today's was not.

⛔ **A SECOND WAY TO LOSE AN EDIT, AND IT IS THE OPPOSITE SHAPE: GITHUB REFUSES THE WRITE OUTRIGHT.** The truncation case below is a copy that silently goes short. **This one is loud and late** — the body is composed, the conflict guard passes, `gh issue edit` is invoked, and the API returns `GraphQL: Body is too long (updateIssue)`. ⚠ **A session that does not check the exit status has composed a correct edit, passed every guard it wrote, and lost the write.**

⚠ **Measured 2026-09-19 on `matsengrp/phyz`, and stated as a BRACKET because that is what was measured: EPIC #369's body was REJECTED at 262,341 BYTES and ACCEPTED at 261,932 BYTES.** ⭐ **2^18 = 262,144 sits exactly between them, so the bracket PINS the ceiling rather than merely being consistent with it.** ⛔ **BYTES, NOT CHARACTERS, and this entry shipped with the units wrong before anyone noticed.** A body dense in multi-byte glyphs (`⛔ ⚠ ⭐ ➡`) runs about **1.012 bytes per character**, so budgeting in characters overshoots by ~1.2% — **at this size roughly 3 KB, which is more than the available headroom and is exactly enough to cause the failed push this entry warns about.** ➡ **Check `wc -c`, not `${#body}`.** ⚠ **The trap was already named one block away**: the archive-comment material below says *"Characters, NOT bytes: `gh` reports `.body|length` in characters, and these files are full of multi-byte glyphs"* — and the confusion was committed anyway, adjacent to its own warning. ⭐ **The meta-lesson worth more than the number: the bracket form was chosen specifically to stop an inference being laundered into a citation, and its UNITS were still wrong. A bracket carries provenance; it does not carry dimension.** ➡ **State the unit in the same breath as the number, the way you state the population.**

➡ **Past roughly 250 KB, every addition needs a cut IN THE SAME EDIT.** A sweep-later plan does not survive the first session that does not know the limit exists. ➡ **Check the composed length against the bracket BEFORE composing the rest, not after** — one line, against discovering it at push time with the edit already written.

⭐ **What to cut, because the shape recurs: an EXPIRED caveat under a CHECKED-OFF item carrying prose counts for a quantity that has a live command.** The measured cut replaced stale figures with a pointer to the command plus the current value: **2,209 characters down to 373.** ⛔ **A body becomes a changelog in exactly those places, and that is where the space is.** This is `PROSE-DISCIPLINE.md`'s mutable-value rule paying rent — a number with a live command does not belong in prose, and the version that does is both stale and expensive.

⛔ **UNRESOLVED, AND IT MAY INVALIDATE THE ARCHIVE-AS-A-COMMENT REMEDY BELOW AT EXACTLY THE SCALE THAT NEEDS IT.** Whether a COMMENT has the same ceiling as a BODY is untested. ⚠ **The one recorded data point is suspicious: the archive comment in the measured case below was 64,734 characters and SUCCEEDED — within 802 of 2^16 = 65,536.** ➡ **If the comment ceiling is 2^16 while the body ceiling is 2^18, a 260 KB body CANNOT be archived into a comment, and the two blocks contradict each other.** **The test is one deliberate call whenever someone next needs it: post a ~70,000-character comment and see whether it is rejected. Do not guess it.** ⭐ **That is RELOCATION, not deferral — the test is free at the moment of actual need and costs a junk comment on a live repo now.** ➡ **And it can be narrowed at zero cost meanwhile: whenever anyone writes a large comment for ANY reason, record whether it succeeded and at what size.** ⚠ **Passive collection would have answered this months ago. The question is open because nobody was looking, not because looking was expensive** — a survey of six issues found a largest comment of **1,321 characters**, which does not say the ceiling is far away; it says the sample is uninformative by construction. ⚠ **Stated here unresolved on purpose, because a reader who has just lost an edit to the size ceiling will reach for that remedy FIRST, and finding out it does not fit is a second failed write on top of the first.**

⛔ **AND THE TIMESTAMP IS NOT THE GUARD THAT PROTECTS THE DATA. A CONFLICT CHECK DEFENDS AGAINST A CONCURRENT EDITOR AND DOES NOTHING FOR THE CONTENT.** In the measured case the body was 63,870 characters and the archive comment 64,734 — large enough that truncation is a real possibility rather than a theoretical one. **Had the comment truncated, every timestamp in the sequence would have been consistent and the findings would still be gone.** So the timestamps catch a concurrent editor *before* you write, and **the verified copy authorises the destructive step**:

```bash
T0=$(gh issue view <N> --json updatedAt -q .updatedAt)          # at pull
# ... edit locally ...
T1=$(gh issue view <N> --json updatedAt -q .updatedAt)          # BEFORE your own write
[ -n "$T0" ] && [ -n "$T1" ] || { echo "ABORT: could not read updatedAt"; exit 1; }
[ "$T0" = "$T1" ] || { echo "CONFLICT: a real concurrent editor"; exit 1; }

CID=$(gh issue comment <N> --body-file archive.md | grep -oE '[0-9]+$')
[ -n "$CID" ] || { echo "ABORT: the archive post failed"; exit 1; }

# `archive.md` is the SOURCE file you posted from, NOT a re-capture of the posted
# body: `gh api -q .body > file` appends a newline, so a re-capture measures one
# character longer. Measured both ways on the same comment: source 64,734 (exact
# match to .body|length), re-capture 64,735.
# Characters, NOT bytes: gh reports .body|length in characters, and these files are
# full of multi-byte glyphs. Measured: a 64,734-character archive is 65,199 bytes.
EXPECTED=$(jq -Rs 'length' < archive.md)
# NULL-CHECK THIS. Bash evaluates an empty string as 0 in arithmetic, so an
# unreadable archive.md makes the threshold 0 and the length gate pass
# UNCONDITIONALLY -- fail-open, on the destructive step. Verified.
[ -n "$EXPECTED" ] && [ "$EXPECTED" -gt 0 ] || { echo "ABORT: could not measure archive.md"; exit 1; }

LEN=$(gh api repos/<org>/<repo>/issues/comments/"$CID" -q '.body|length')
[ -n "$LEN" ] || { echo "ABORT: could not read the archive back"; exit 1; }
# Generous on purpose: this detects TRUNCATION, which is large. An exact test is
# brittle across encodings and trailing-newline handling -- measured in both
# directions, +1 and -31 depending on how the local file was produced.
[ "$LEN" -ge $((EXPECTED * 95 / 100)) ] || { echo "ABORT: archive short — $LEN of ~$EXPECTED"; exit 1; }

# Anchor the content check at the END. Truncation removes the tail, so a string
# from the opening passes on a truncated body. Pick a phrase from the LAST section.
gh api repos/<org>/<repo>/issues/comments/"$CID" -q .body | tail -40 \
  | grep -q '<a phrase from the archive'"'"'s final section>' \
  || { echo "ABORT: archive tail missing — truncated"; exit 1; }

gh issue edit <N> --body-file new-body.md   # authorised by the VERIFIED COPY, not by a timestamp
```

**Do not try to self-verify with a third timestamp.** The natural repair — re-read after your write and confirm it moved — fails because **`updatedAt` has been observed to lag a few seconds** (see the closing-keyword rules in this file), so a transient equality aborts a correct sequence. **Reading a value you never compare is worse still: an earlier draft of this passage read a `T2`, null-checked it, and left a comment claiming the edit was "gated on T2" when nothing compared it.**

⭐ **This passage took five review rounds to write, and every defect found was an instance of the rule it states** — an unused `T2`; a byte-vs-character comparison; a content grep anchored where truncation would not show; a dry run that validated a near-neighbour of the artifact rather than the artifact; and a fail-open on the destructive step. **All five were caught by review, none by use.** The failure mode is invisible to the person writing it and cheap for anyone else to see — which is the argument for a second reader on anything shaped like this, and the reason not to trim this block on the assumption it was written carefully the first time.

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
