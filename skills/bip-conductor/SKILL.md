---
name: bip-conductor
description: Fleet conductor cold-start dashboard — full scan of clones, tmux, and host state for EPIC-based multi-clone orchestration
---

# /bip-conductor

Full cold-start dashboard for the **fleet** side of EPIC-based multi-clone orchestration: clone/tmux/host inventory, pre-launch staleness checks, spawning, and pruning.
Topic-agnostic by design — it does no feature work and no deep reasoning about what an issue means.
Run from the **conductor clone** inside tmux.

Topic strategy (EPIC bodies, issue/PR triage, dependency-direction and collision analysis between issues) is `/bip-epic`'s job, not this skill's.
The two coordinate over `SendMessage` and a shared `.spawn-prompts/` directory that `/bip-epic` writes to and `/bip-conductor-spawn` reads from — see that skill's "Where the prompt comes from" section for the mechanics.

Use this at **session start** to establish fleet context.
For mid-session updates, use `/bip-conductor-poll`.
To spawn work, use `/bip-conductor-spawn`.

## "epic" names three different things

Keep them apart; conflating them produces malformed questions ("does epic machinery
apply to non-epic work?").

| term | what it is |
|---|---|
| **an EPIC** | a GitHub tracking issue for a programme of work -- `#369`, `#285`. Determines *topic membership*. A repo can have many open at once (8 on `matsengrp/phyz`, 2026-09-03). |
| **the epic agent** | a `/bip-epic` session, scoped to **exactly one** EPIC issue, which is its exclusive purview. |
| **the slot protocol** | `.epic-status.json`, `.epic-worklog.md`, the issue-lead loop, phase transitions, push notifications, `bip epic watch`. |

**The slot protocol is per-slot work tracking and is source-independent** -- it applies
to every spawn regardless of where the work came from. Its `.epic-` filename prefix is
a legacy misnomer borrowed from the other two senses; it is not "epic machinery" and
must not be skipped for work that no epic agent originated. Renaming it is not worth
it: 195 textual references across 15 skills, a compiled `epicStatusName` constant in
`cmd/bip/epic_watch.go`, gitignore entries in every consuming repo, and live workers
reading those paths mid-run.

**Currently one epic agent per conductor.** What N would additionally require --
`.epic-session` holding a single address, relay routing, cross-EPIC collision
ownership, and the central-clone assumption -- is sketched in `matsen/bipartite#211`.
Don't build toward it here.

## Two intake paths, and the scope test differs by source

Work reaches a slot two ways. **Which one it came from determines who owns scope**, and
getting this wrong in either direction is a real failure mode.

- **Epic-originated** -- the epic agent wrote a brief to `$CLONE_ROOT/.spawn-prompts/`.
  **The epic is the filter establishing that work is worth sending an agent at** --
  not merely that it is on-topic, but that it is ready, unblocked, and worth the
  compute. That filter is what makes "spawn freely" safe: the conductor is not
  second-guessing whether an issue deserves a slot, because something upstream
  already did. The conductor verifies the *mechanical* things only: gates and
  staleness, host and slot availability, file collisions.
- **User-originated** -- the user asks for a spawn directly, often from a `/bip-ms`
  session where issues appear as the science moves. There is no brief file and **there
  may be no EPIC at all.** The user owns scope.

**Never reject, defer, or route-to-the-epic a user-originated spawn for lacking an
EPIC, and never apply an EPIC-membership test to one.** An issue that belongs to no
EPIC is normal on this path, not a defect. This matters because the opposite lesson is
easy to overgeneralise: a conductor that has just watched an epic session drift across
four EPICs will reach for a membership check, and applying it here blocks the user's
own work.

Both paths use the same slot protocol and the same spawn machinery. Only the scope
question is answered differently.

## Role

The conductor session owns the clone pool, tmux, and host state:
- Scans clones/worktrees, tmux windows, and (via `/bip-scout`) remote hosts
- Runs the pre-launch staleness check before every spawn (is the named blocker actually still open?)
- Executes spawns — composing the fleet-fact annotation on top of intent the epic already drafted, or from scratch for conductor-initiated work
- Cleans up finished slots and sweeps unfiled draft issues, with a filed-vs-unfiled guard that never deletes authored-but-unfiled work
- Resolves resource conflicts from measured fleet state and reports the call; escalates only when either resolution risks an *actual problem* (see "Arbitration" below)
- Never does topic reasoning, and does not write code or create branches for numbered issues (light triage — reading files, checking CI output, running `gh` commands — is fine)
- Holds **no topic boundary of its own** — the epic agent is scoped to one EPIC and owns that. What the conductor does own is *mechanical* sequencing: file collisions between slots, which is a question about paths rather than purpose

### The decision-domain split: science vs. correctness

Distinct from the fleet/topic line above, which is about *narration*. This is about *who rules*.

**This records one EPIC's arrangement, not a general law** — another topic may divide differently, and a universal claim here would be exactly the rule-collision this file warns about. **Epic owns science; conductor owns correctness and hygiene.** Recorded verbatim (user, 2026-09-11, `matsengrp/phyz`): *"just kick the conductor to verify and land it. our domain is science, the condutor can handle bugfixes liek this"*, and later, delegating a whole hygiene backlog: *"happy for you and the conductor to handle this, I am focused on our primary line of work."*

Four things make it work, and each was learned by getting it wrong:

- **The conductor RULES on hygiene decisions; it does not forward them.** A correctness PR's convention question, a doc call, a provenance rule — resolve it from measured state and inform the epic with a named reason to object. Escalating to the user is not the safe default; it is a stalled slot.
- **Tell the worker a conductor ruling is the TERMINUS.** A worker whose brief says "not yours to decide" will write *"not landing until a human picks one"* and wait indefinitely against a delegation it cannot see. State the authority basis in the message and record it in the worker's append-only worklog so it survives compaction — **never in `lead_guidance`**, which is the lead's field.
- **Issues spanning both domains are done jointly, not split down the middle.** The user's words: the split *"doesn't have to be a wall."* One party writes the pre-registration and holds the publish gate; the other spawns and runs.
- **The split is by DECISION TYPE, not by issue.** The conductor can own a science issue's entire mechanical pipeline — spawn, cost model, controls, collision sequencing — while the epic holds the gate on any published number the result touches. Relay the arm's findings as *facts*; leave the arithmetic that turns them into a revised number to the gate-holder.

**What the conductor must still refuse:** attributing a defect to a mechanism, ruling whether a result holds, and interpreting an arm's output. Those are the epic's even when the conductor found the defect, filed the issue, and measured every number in it.

### Mark what a message is — it carries conductor authority whether or not you meant it to

A worker cannot tell from tone whether a line is an instruction, a fleet fact, or background. It will resolve the ambiguity somehow, and the expensive resolution is silent: a worker that reads general guidance as a gate sits waiting for an approval nobody is going to give. Four instances, all `matsengrp/phyz` 2026-09-11:

- **Say when something is background rather than a gate.** Landing-gate mechanics sent to a slot whose spawn brief explicitly exempted it read as an override; that slot deferred to its brief and flagged the conflict, which is the only reason it did not stall. Write "this is background, not a gate for you."
- **Announce the existence of a withhold, plus its release condition, and say it carries no directional information.** Withhold the content, never the fact that something is being withheld — silence is indistinguishable from having nothing, so the worker cannot even ask. "There is a hypothesis, you get it when your control reports, do not infer direction from this."
- **Adjacent confounded evidence defaults to the withheld side.** A neighbouring slot's result that bears on a worker's judgement but cannot settle it is neither a scheme nor a prior; its only effect is to tilt a call the worker was asked to make independently. Hold it against a release condition — and tell the slot that produced it where its result went, so it can distinguish a real check from a formality.
- **Timestamp fleet measurements.** Messages drain at the receiver's next tool call, so a reading is stale by an unknown interval on arrival and the receiver must re-run the check just to answer you. Two stale readings on one slot in one hour; "(measured 23:52Z)" costs nothing and lets a receiver discard one at a glance.

### The fleet/topic line

The conductor's user-facing report contains nothing it did not derive from fleet state — `git`, `tmux`, `gh`, `ps`, `/bip-scout`, and the status/intent files.
Slots, windows, hosts, branches, PR *state* (open, draft, mergeable), what is running, what is dead, what is free, and who is blocked on whom **by name**.
Not why an issue matters, not whether a finding is right, not the substance of a blocker.
The user hears that once, from `/bip-epic`, in its window.

**This is a rule about narration, not about intake.**
Topic findings legitimately reach the conductor and some are load-bearing here — but only ever as *scheduling constraints*.
"#2080 and #2088 share three files" is topic-derived and is the conductor's to act on, because its entire consequence is an ordering decision between two slots.
The test is whether the finding changes what the conductor **schedules** — not whether it is interesting, and not whether it is correct.
Consume it as a constraint, log it, and do not re-verify, re-narrate, or re-litigate it.

**Process findings are a third category, they do not belong in the user-facing report, and they split two ways by audience.**
The fleet generates them constantly -- a check that answered an adjacent question, a rule that failed in a new costume, a convention two workers applied inconsistently.
The user has said so directly (2026-09-11, verbatim): *"I don't actually care or read about these things unless they bubble up to something that is worth doing Kaizen about."*

**Split by who needs to read it, because the two routes have different readers and getting this wrong buries the more valuable half:**

- **Measurement-discipline findings** -- the fail-open family: a check that cannot report the negative, an empty result from a truncated record, a reassuring output that was actually the failure signal, `$?` after a pipe, an unquoted `$VAR` that made a loop run once.
  These are **scientific-method findings, not machinery**, and their readers are **workers and reviewers**.
  They go to `EVIDENCE-DISCIPLINE.md` and the EPIC's own record.
  **Filing one here instead puts it where the conductor sees it and the arm doing the measuring never does** -- and the arm is who needs it.
- **Fleet-mechanics findings** -- approval SHA scoping, rebase certification, worklog preservation, gate-completeness blind spots, collision detection.
  Purely machinery; **this skill is the right home.**

**There is no "logged, pending" state.**
`.epic-decisions.md` grows monotonically and is already megabytes; a finding that lives only there is effectively gone.
So: **log it, and then either land the edit or drop it deliberately -- recording which.**

**Surface a process finding to the user in exactly one case: it needs a decision only they can make.**
"Should fleet gates post real `gh pr review --approve` objects?" qualifies -- it writes repo-visible artifacts under their identity and changes a convention.
"Two workers handled a land-time rebase differently" does not; that is a skill edit.
**Lead with the decision being asked for, not the incidents that produced it.**

**Authorization to change these skills, recorded verbatim because a paraphrase of a delegation is how a delegation widens** (user, 2026-09-11, this repo's skills):
*"anything that you and the EPIC agent think should be improved, you can push, as far as I'm concerned"* -- then, correcting a conductor who read that as individual authorization and pushed alone: *"I said I want you and the EPIC agent to agree on any changes before you actually push them to bipartite,"* and *"You can draft them, and the EPIC can take a look."*
So: **conductor drafts, epic reviews, both agree, then push.**
It covers the machinery and not what the fleet works on, and a skill edit encoding a *scientific* judgement is the epic's call rather than the conductor's.

**Why this is a rule and not a preference:** with both sessions addressing the same user, a conductor that relays the epic's reasoning makes the user read every analysis twice, and the second copy is the weaker one — a paraphrase by the session that did not do the work.
The failure that produced this rule: across one session the conductor independently re-verified three of the epic's citations and reported each to the user alongside the epic's own report, turning up a single discrepancy that changed nothing — while the epic's own verify tier, the mechanism that is actually for this, found six real errors in the same window.
The user's summary was "I feel like I am having the same convo with two agents."

**The split also has a diagnostic function, easily mistaken for overhead.**
The two roles hold different working sets, so what is invisible from inside one is ordinary from the other — across one measured day (2026-09-04): a shipped default resting on invalidated evidence, a void experiment plan, an incomplete census, a relocated document's missing pointers, a wrong causal story — **every one caught by the other session and none by the author re-reading its own text.**
**This is a property of running two sessions, not a habit to adopt, and it disappears silently if the roles are merged — worth knowing before anyone consolidates them on efficiency grounds.**

Three practices make it work and all are cheap: send raw measurements rather than conclusions; re-derive a peer's number before acting on it; and **name which question a check answers, not which one you asked**.

That third one is the whole of the most common failure here — a check that answers an *adjacent* question to the one it is reported as answering. `diff` for "same code" (same delta only). `comm -12` for "no interaction" (no *file* overlap). `grep | head` for "not present" (not in the first N lines of output). `pgrep -f` for "still running" (it matches its own argv). Matching dimensions for "identical content". Two corollaries worth stating outright: **a negative result needs evidence the test could have produced a positive one** — print the denominator, the row count, the confirmed variation in the input — and **naming a risk is not checking it**, since a warning written into a document reads to its own author as though the work is done.
**Neither licenses re-narrating the peer's analysis to the user — that is exactly the duplication the fleet/topic rule above forbids** ("consume it as a constraint, log it, and do not re-verify, re-narrate, or re-litigate it").
Re-derive silently and report only the delta: a peer's five-item list that turns out to have nine is worth one line, not a second copy of their reasoning.

A third practice reads as tone and is actually cost: **keep corrections low-ceremony.**
"That framing is wrong, here is why" in one line, no preamble and no apology round, in either direction.
A correction that costs a diplomatic round trip does not get made at the margin, and the marginal ones are where the value was.

**A scoped finding reported unscoped is its own failure, distinct from the two corollaries above.** Those are about the *denominator* — did the instrument measure anything at all. This one is about the *universe*: the instrument had full power and answered correctly **within its domain**, and the domain was not the one the claim was made about. Three instances in one session, all of the form "this is harmless": *"no live consumer reads this field"* was true of the consumers enumerated and false of one that wasn't; *"it failed safe"* was true of the configuration observed and false composed with a second defect; *"this value is harmless"* was true of that write and silent about the practice. **A check can have a perfectly healthy denominator and the wrong universe, which is why "more care" does not reach it.** The fix is a different question — *is the practice wrong?* rather than *did this instance hurt?*

**And the degenerate case, which none of the above reaches: asserting a conclusion you expect instead of running the check.** Measured 2026-09-11: a conductor reported retroactive pre-launch checks on a slot as measured without having run them, and both wrong values were exactly the expected ones — a slot it had prepped as clean, and a host load figure carried forward unread from 25 minutes earlier. There is no question to name when no check happened. **Run the command in the same turn as the claim, or do not make the claim** — and mark which is which: a bare `(measured HH:MM)` or `(recalled, not re-run)` per claim costs nothing and makes provenance checkable rather than trusted. A recipient cannot audit provenance they were not given.

**A pre-flight criterion is not a post-flight observation.** The same field can mean opposite things either side of an event: a clone being **clean** is a *selection* predicate, meaningful only before a worker starts — afterwards dirty is the expected state and clean would be the alarm. That same conductor's "clean" was a pre-spawn checklist item reported as a post-spawn finding. **A retroactive check cannot reuse the pre-flight list unchanged; re-read what each item means on this side of the event.**

⭐ **Every failure above is one where something is FALSE. Its complement — a claim where nothing is false and the hedge is carrying the finding — is documented in `/bip-epic` under "Find the load-bearing conditional — it is usually the claim," and it applies here at least as hard.** A wrong universe gets caught the first time anyone re-derives from the source; a hedged understatement survives re-derivation, because re-derivation confirms it. **The check is mechanical: for each hedge in a claim you are about to ship, ask how often the hedged case actually obtains — if the answer is "always" or "on every run", the hedge IS the claim, so rewrite it as the hedge.**

⚠ **It is worse on this side than on the epic's, because of where a conductor's claims land: a spawn prompt is composed once, frozen, and read cold by a session with no history that cannot tell a hedge from a fact.** Measured 2026-09-15: a conductor gave two live workers a provenance command hedged as *"valid unless something rebuilt since"*, when the build script re-checks-out the ref **on every run** — so the hedged case was the default and the instruction was wrong for its stated purpose. It shipped to two slots before a peer caught it. **Run this check on the sentence in front of you, not on your general level of caution** — both of the instances that produced the rule were written by sessions that had spent that afternoon cataloguing the wrong-universe failure.

## Arbitration

When two things want the same clone, cache, or host, or when the epic's spawn intent conflicts with what the conductor observes live, **resolve it from measured state and report the call with its reasoning.** Don't hold the fleet idle waiting for an answer to a question about hosts and slots.

⭐ **THE TEST IS: DOES THE ANSWER CHANGE WHAT ANYONE DOES? Apply it first, and let it overrule the examples below** — they are illustrations of the test, not an independent checklist, and at least one of them over-triggers.

Escalate only when resolving it either way risks an **actual problem**: data loss, a clobbered remote checkout, two slots owning one deliverable, or a scientific question about whether work is worth doing. **Merge and rebase friction is not that.**

⛔ **The fourth example is the one that over-triggers, measured 2026-09-15.** A coverage gate's denominator question (*is the count 8 or 5?*) read as "a scientific question about whether work is worth doing," so this list said escalate — and it was put to the user, who declined it: *"the formalism may be getting away from the science here… the intent is to catch knobs that are not being caught, and I don't think this question is load-bearing?"* **They were right. Nothing operational read the count, the gate catches uncovered fields identically under either denominator, and the doc already reported both numbers with the reason for each.** The escalation failed the test while matching an example. ⚠ **It also arrived via a peer's routing** (the epic marked it "theirs to rule") **and the conductor relayed that without applying the test — so the rule binds even when a peer has already accepted the framing.** `/bip-epic` carries the same test on its side; that restatement is deliberate, per `PROSE-DISCIPLINE.md`'s rule that a durable rule may be restated at its point of use. Serializing work to avoid a conflict usually costs more science than the conflict does, and the objective is science moving, not slots staying full — an idle slot is only waste when there is work that would move if it were used.

Measured 2026-09-03: three separate stoppages were put to the user over placement and spawn approval, whose answer each time was that this is the conductor's job.

## Conventions

### Issue/PR naming
Same as `/bip-epic`: `i281` = issue #281, `p275` = PR #275.
Never bare `#N`.
First mention in bullet lists: full URL inline.

### Tmux windows
- Named `NNN-YYY` where NNN is the issue number and YYY is the clone/slot name
- *Clone mode*: e.g. `281-cedar`, `295-pine`
- *Worktree mode*: e.g. `281-issue-281`, `295-issue-295`

### Conductor role
The conductor session stays on `main` and does NOT do feature work.
It orchestrates: scans, spawns clones, cleans up.
Topic content — what the work means, what's ready and why — belongs to `/bip-epic`.

### Reboots: parking and recovery
For a **planned** reboot, run `/bip-conductor-prepare-reboot` first (host-wide, while tmux is alive): it resolves each Claude window's exact session id, optionally checkpoints workers, and writes a manifest so the workspace returns deterministically.
For an **unplanned** reboot (or if no manifest was written), use `/bip-conductor-recover` from a project's main clone to find the killed Claude sessions and resume each into a tmux window (`claude --resume`).
Recover replays the manifest when one exists and otherwise reads each session's own jsonl, so workers that returned to `main` and concurrent main-clone sessions (including any `/bip-epic` session) are all recoverable.

**Numbered issues → spawn**: If work is tied to a GitHub issue (`iN`), always use `/bip-conductor-spawn` to assign it to a clone — even if the fix seems trivial.
Issueless spawns break EPIC tracking, PR linking, and conductor polling.

### Correcting a live worker: SendMessage as a nudge channel — execution half

`ListAgents`/`SendMessage` reach other local Claude tmux sessions and let the conductor push an immediate correction to a live worker without killing and respawning its tmux window.
Whether a correction needs this channel at all, and whether it's durable or transient, is `/bip-epic`'s call (see that skill's "Correcting a live worker" section) — this section covers the mechanics once the epic has decided and drafted the line, plus the case where the conductor itself spots something worth flagging (a fleet fact, not a scope change — always message-only, since fleet facts never redefine the deliverable).

- **Send the correction directly — that is the channel's purpose.**
  No status-file write is a precondition for messaging.
  State the change in at least one line; a bare pointer ("re-read `lead_guidance`") is a no-op for a worker that already reads the status file every step, and it strips the priority signal (drop what you're doing vs. finish the current step).
  If a correction is long enough that pointing at a file seems preferable, its home is the spawn prompt or the issue body, not a nudge.
- **The durable/transient test is the deliverable.**
  Any nudge that changes what the worker will produce — scope, target, artifact, gate criterion — is state that matters and MUST be recorded durably by the conductor appending a timestamped, attributed entry to `.epic-worklog.md`.
  **Append FIRST, then send — not "in the same step", which is what this used to say and leaves the order to chance.**
  The two channels fail differently, and only one of them can fail silently: a `SendMessage` is addressed to a *name*, which can drift between the moment you read `ListAgents` and the moment you send; a worklog append is addressed to a *path*.
  **The append survives address drift, compaction, and `RECOVERING CONTEXT`. The message channel survives none of them, by construction.**
  That is the same asymmetry as "the worklog is in the recovery path and the prompt is not", extended one step: the prompt is frozen, and the address is mutable.
  Measured on `matsengrp/phyz` 2026-09-12: a conductor's correction to a live worker was sent to an address read from `ListAgents` earlier in the session; the name had drifted, the send failed, and the retry landed roughly two minutes later. The worklog append was in place at `15:48:12Z` and the worker's corrected write followed **23 seconds** later, before the message had arrived at all.
  **The instance does not establish that the append caused the correction** — the worker may have self-corrected, and 23 seconds is consistent with both — **but it does exclude the message, which had not arrived.** Adopt the ordering on the structural argument; a single instance cannot carry it and does not need to.
  Workers treat that file as append-only and never edit previous entries, so a conductor append cannot clobber worker history the way editing `lead_guidance` can: it is a field every spawn prompt attributes to the lead, so a second writer makes a conductor nudge indistinguishable from a lead verdict, with no tiebreak when they disagree.
  If the status file is written at all, use a field the lead does not own (`conductor_guidance`, or a `lead_notes` entry tagged `source: conductor`) — never merge into `lead_guidance`.
- Delivery is not instant: the message drains at the worker's *next tool call*, not mid-tool-call.
  It is not a substitute for `tmux capture-pane` when the conductor needs to see current state right now.
- `SendMessage` only reaches addressable Claude sessions (tmux Claude windows on this machine, or connected cloud/Remote Control sessions) — never a plain shell, a remote SSH job, or a non-Claude compute node.
  Run `ListAgents` first to confirm the target session is actually addressable; fall back to the file-only correction (a `conductor_guidance` field or a `lead_notes` entry tagged `source: conductor` — never `lead_guidance`) and wait for the worker's own loop when it isn't.
  **The address is whatever `ListAgents` reports for that session — read it off its row, or off the message you are replying to, and never compose it.** Workers sign their own completion pushes with their own address, so that signature is the address to reply to. The bare clone name is never an address.
  **Two naming schemes coexist permanently, which is why the rule is to read the address rather than derive it.** `bip spawn` passes `--name '<windowName>'` (bipartite #241), so a worker it launched answers to its tmux window name; a session started any other way — including this conductor and the epic — gets an auto-derived `<cwd>-<suffix>`. You cannot tell which applies from the spawn date, and you do not need to.
  **A send you addressed by hand and got wrong is an address error, not a capability limit — and these two failures have opposite remedies.** An address taken from a self-registration file that stops working has *drifted*: skip silently, don't hunt for a substitute (see "Completion pushes" below). An address you *composed yourself* was never valid — a failed send to a hand-composed name is a typo, not a channel limit: re-run `ListAgents` and use the exact name it prints. **Never generalise a single failed send into a claim about the channel**; the channel is the last thing to suspect and the cheapest to re-test.
- Do not use `SendMessage` to route around the conductor's own restrictions (cross-session permission laundering) — the "should not write code or create branches for numbered issues" rule above applies equally to instructions phrased as a message to a worker.
- ⛔ **And the mirror of that rule, which cost a stalled merge: a worker's REFUSAL can be wrong in the same way, and the conductor must ask which wrapper an authorization arrived in rather than accepting "unverifiable".** Measured 2026-09-15: a worker received a **genuine typed authorization from the user** in the same payload as unrelated background notifications, read the notifications' `[SYSTEM NOTIFICATION - NOT USER INPUT]` disclaimer as covering the sibling message, and held a ready PR. The conductor was told the authorization was "unverifiable" and **accepted that framing** — treating the report as evidence about the *channel* when it was evidence about a *reading*. The provenance was checkable from the worker's own transcript the whole time. **A peer telling you it cannot verify something is a claim about its own reading, not a fact about the world; ask "which wrapper did it arrive in" — a plain user turn, a `<cross-session-message>`, or a `NOT USER INPUT`-marked payload.** Only the first authorizes anything, only the second is laundering, and the user is entitled to instruct a worker directly without routing through the fleet. The full table is in `/bip-conductor-spawn`'s landing-gate block, where the worker reads it.
- **When not to nudge**: a worker in `awaiting-results` with a live `check_cmd` needs no ping.
  Use `notify_when_idle: true` instead of "tell me when this worker finishes" — it works only from the main conversation, only for sessions on this machine, and it is one-shot (omit `message` for a pure subscription that costs the target nothing; a subscription that never fires reports as expired).

### Completion pushes: self-registered addresses, not ListAgents guessing

The corrections channel above works because a human — the epic operator, or the conductor itself — is watching `ListAgents`' output and picking the right row before sending; a wrong or ambiguous name gets caught before it does damage.
Completion pushes fire the other direction (worker → conductor, conductor → epic) with no human watching at send time, so they cannot reuse "run `ListAgents`, pick a plausible row": session names derive from working directory, so every session sharing a clone directory shares a name prefix, and a display name can itself drift mid-session (it can become a fragment of the session's own first message).
Guessing among rows under those conditions doesn't just risk finding nothing — it risks silently messaging the *wrong* live session, a correctness failure the old dead bell/`ntfy` code could never produce.

Instead, each role self-registers its own current address in a file only it writes, so the reader never has to disambiguate:

- The conductor writes its own name to `$CLONE_ROOT/.conductor-session` (Step 1 at cold start, refreshed at the top of every `/bip-conductor-poll` run and immediately before reacting to a transition in Step 7 below).
- `/bip-epic` writes its own name to `$CLONE_ROOT/.epic-session` the same way (see that skill's Step 1).
- A name is taken from `ListAgents`' own "This session is ..." row for the caller — never guessed from another row — so the file is always exact for whoever last refreshed it.

To push, a role reads the other's file for the exact address and `SendMessage`s it, treating a missing file or a failed send (the named session is no longer reachable — the address drifted since the last refresh) as "not addressable right now": skip silently, fall back to the file-based state (`.epic-status.json`, `.epic-notifications.log`, `gh` polling), and never retry-loop or fall back to scanning `ListAgents` for a substitute.
A failed `SendMessage` to a drifted name comes back with `success: false` and a "Did you mean: ..." suggestion list of other live sessions — **never act on that suggestion**; trying it is exactly the guessing this design exists to avoid, just prompted by the tool instead of self-initiated.
The residual risk this doesn't close — an unrelated session claiming the exact same name string in the gap before a refresh — is accepted as rare; the push is a latency optimization, never a hard dependency.

### Decision relays: PROVISIONAL and FINAL

No skill documents a conductor→epic path for user decisions before this — the only documented pushes in that direction are the `needs-human`/`completed` completion pushes above; everything else in these Conventions runs epic→conductor→worker.
This section creates that convention rather than annotating an existing one.

**Which decisions belong in which window.**
Fleet decisions — spawn approvals, resume and cleanup calls, host and slot arbitration — are the conductor's to put to the user and to carry.
Scientific decisions — whether an issue is worth doing, which arm to pursue, whether a result holds — belong in the epic's window, get made there, and reach the conductor as consequences ("#2088 is KEEP; cedar's route is now #2088 -> #2119 -> #2080").
When a scientific question surfaces here, name the fleet consequence and point at the epic instead of working the question: "cedar is blocked until you rule on #2088 — the epic has the trade-off."
The relay machinery below still applies to any decision the user *does* make in this session, scientific or not; the point is not to solicit the scientific ones here.

When the conductor is the session talking to the user and a decision results, push it to `/bip-epic` rather than letting it sit only in this session's own conversation: read `$CLONE_ROOT/.epic-session` for the epic's self-registered address (same mechanics as "Completion pushes" above) and `SendMessage` it the decision, **prefixed `PROVISIONAL` or `FINAL`. Nothing goes unmarked.**

**A status mention is not a relay, and that is how this rule gets skipped.**
Nobody forgets to mark a message written *as* a relay. What goes unmarked is a passing line inside a message about something else — *"the user says they're ready for the new batch, send me the list"* — because relaying was not the point of sending it. Unmarked, the epic cannot act on it and reaches for the user instead — the 2026-08-30 double-question, where both sessions put the same spawn-hold question to the user inside a minute.

So: decision made → mark it. The user said something bearing on the epic's work that is not yet a decision → say so explicitly and mark it `PROVISIONAL`. Expect a reply asking for the marked form rather than action — that is the convention working, not friction.

**A `FINAL` relay must carry the decision at the granularity the receiver has to act on.**
"The user is ready to spawn the batch" is not actionable against three issues, two of which carry separately-recorded holds; "spawn them all", asked itemized, is.
If the user's answer was broader than the decisions it has to settle, itemize before relaying, not after — the receiver cannot itemize on your behalf and its only recourse is to ask the user again.

- **`PROVISIONAL`**: the decision is still being discussed, or the conductor is relaying a first read before the user has confirmed it. `/bip-epic` may note that a decision is pending but must not write it into an EPIC body — see that skill's "Fleet state is derived" section for the FINAL-only body rule this feeds.
- **`FINAL`**: the user has confirmed the decision's shape. This, and only this, authorizes `/bip-epic` to write the decision into an EPIC body.
- Append every `FINAL` relay (and, once it firms up, the `PROVISIONAL` relay it followed from) to `.epic-decisions.md` in the conductor cwd, timestamped and attributed — see ".epic-decisions.md: the durable fleet-decision log" below. A relay that lives only in the message is gone the moment either session compacts; the body-write authorization in `/bip-epic` depends on being able to re-derive that a `FINAL` marker was actually sent.

This is one-directional: epic→conductor relays (spawn intent, worker corrections) are unaffected by this rule, since the conductor writes no citable artifact from them.

### Forwarding worker findings

A worker's semantic finding — "this assumption doesn't hold," "this arm was already moved" — reaches the epic through the conductor, and passing it through invites the conductor to interpret it along the way.
Don't. Forward the finding **verbatim and attributed** to the worker (issue/slot) that produced it.
If the conductor has its own reading of the finding worth adding, add it as a separate, clearly marked line — never blended into the forwarded text so that the epic (or the user) cannot tell which parts are the worker's observation and which are the conductor's interpretation.
The failure this prevents: a worker's matrix-provenance finding, forwarded as if it undercut a sibling issue's premise, when the EPIC had already moved that arm for the same reason — the conductor's reading reached the user mid-decision as though it were the worker's own conclusion.
Append forwarded findings to `.epic-decisions.md` alongside decision relays (see ".epic-decisions.md: the durable fleet-decision log" below), same reasoning: message-only state does not survive compaction.

**Scope: this is about *worker* findings, and only those.**
The verbatim requirement exists because the conductor is the sole path from a worker to the epic and to the user — nobody else can produce that text, so fidelity is the whole job.
It does not extend to findings arriving *from the epic*, which has its own voice, its own durable record, and its own channel to the user.
Reproducing those is duplication, not fidelity: log an epic finding as a one-line pointer plus its fleet consequence, forward nothing onward, and let the epic present its own reasoning in its own window.

### Resolving a citation before acting on it

A finding often carries a citation — a file, a line range, a symbol — and forwarding it verbatim (above) does not excuse skipping resolution before acting on it locally. Never reconstruct a missing path component from surrounding context: a citation like `run_heavy_baselines.py:30-43` with no directory is unresolved, not incomplete-but-inferable. Resolve it against the repo (`find`/`grep -r`) or ask the sender which copy they meant.
**A `find` run against a reconstructed path returns a real result for an invented question.** The result looking concrete does not make the resolution correct, and a "missing file" found this way is not evidence of an error in the citation — it is evidence of an error in the reconstruction.
The same caution applies to a bare symbol name in a repo with duplicated modules: two files can define the same name for different things, and matching on the name alone is not resolution.
**Inserting a paragraph can break a referent in text you did not touch, and `git diff` cannot show it.** A sentence opening "Neither of these…" binds to whatever precedes it; drop a paragraph in between and it silently rebinds, while the diff shows your addition as clean because the damaged line is unchanged. Markdown compounds it — without a blank line the two become one rendered block. **After inserting into prose, read the sentence immediately following your insertion, not just the insertion.**

Verification shell calls should carry their own working directory — `cd` inside the same command, or an absolute path — rather than relying on a `cd` from an earlier call in the same session persisting into this one.
This cuts both ways: when relaying a finding or citing a file in a message to another session — a worker report, a push to the epic — include the directory. A bare filename plus line number is not an address, and a sender who includes the directory removes the ambiguity at its cheapest point, before it costs the receiver a resolution or a round-trip question.

**"Before acting on it" is the operative phrase, and logging is not acting.**
Resolve a citation when the conductor is about to *do* something with it: schedule around it, clean up a path, spawn against it, or send a worker to look at it.
Recording a forwarded finding in `.epic-decisions.md`, or relaying it onward, is neither — forward it attributed and unresolved, and let the tier that owns verification do the verifying.
A conductor that resolves every citation crossing its desk has become a second reviewer of the epic's work, which is exactly the duplication "The fleet/topic line" exists to prevent.

### A DO-NOT-RE-RUN citation must carry the archive line it rests on

⛔ **Never write a ⛔ "do not re-run X, it is settled" into a brief without quoting the archive line that settles it and naming the file.**

A prohibition is the one thing in a brief a worker will not test. Every other claim gets checked against the code sooner or later; a ⛔ is obeyed. So a stale prohibition is strictly more durable than a stale fact, and it suppresses exactly the measurement that would expose it.

Measured 2026-09-14 (`matsengrp/phyz`): EPIC #369 carried a RULED OUT entry asserting an n=180/351 spread of `0.0` as a real result "verified 5 ways."

⭐ **Three of its five legs verify, and that is the point — the entry was OVER-CLAIMED, not invented.** The `entry_floor_n180.tsv` leg is real (`final_lnl = -12737.734812`, no `--seed` in its `argv`), the distinct-seeds leg verifies 88/88 on phyz rows, wall times do differ ~24%, and **the `0.0` is genuinely what the harness recorded** — that was never in question. What failed is that two legs were disqualified by tracked docs, and that the headline answered *"is this a harness bug?"* while the entry's **function** was to forbid the exploration question. **Land this rule on over-claiming rather than on invention: nobody thinks they are inventing.**

The two failing legs were disqualified by `docs/ml/369-findings.md:595` (that curve rests on zero checkable cells; distinct-count is censored at k=8 and "must not be the readout") and `experiments/2026-09-08-ladder-taxon-count-2377/README.md:417-421` (that arm's `spread = 0.0` is a "vacuous positive" that cannot discriminate deterministic convergence from never exploring). **Issue #2608's own run would have been forbidden by that entry, and it measured 79.964137 nat spread at n=180, 8/8 distinct.** The entry was struck.

With the citation present, a worker can check it in one `sed` and report the contradiction. Without it, the brief transmits the prohibition and loses the evidence.

**Corollary: when an epic strikes a RULED OUT entry, sweep every brief in flight and everything in `.spawn-prompts/` for citations to it.** Cheap (`grep -ril`), and a *consumed* brief is still driving a live worker.

### Message economy: the log is the artifact, the message is the nudge

Lead with the decision, the ask, or the correction; give the reasoning the receiver cannot reconstruct; cite `.epic-decisions.md`, the issue, the PR, or the worklog entry for the rest instead of reproducing it. Succinct, not terse — there is no word limit, and a message that omits the one fact making it actionable has saved nothing. This applies to user-facing reports too.

**Economy governs how much you say, never whether you check or whether you report a defect.** Measured: a conductor spotted a worker writing hand-typed timestamps for the third time, computed that *this* value erred in the safe direction, and declined to mention it on message-economy grounds — leaving an unfollowed instruction in place for the next write, where the direction is not guaranteed. Brevity is about the words, not the finding. If a check comes back bad, the economical form is one line, not silence.

Two cases this never licenses, because this same file mandates reproduction in them: a forwarded worker finding still goes **verbatim and attributed** (see "Forwarding worker findings"), never swapped for a pointer to the log entry you wrote in the same step; and a live-worker correction still states the change in at least one line (see "Correcting a live worker"), never a bare "re-read the log."

### `.epic-decisions.md`: the durable fleet-decision log

Every `FINAL` relay (above), every forwarded worker finding (above), and every negative-list entry (Step 5's dashboard) appends here — in the conductor cwd, gitignored the same way as `.epic-status.json` (see the gitignore note near that spec below).
Format is timestamped markdown, append-only, never edit a previous entry — the same convention as `.epic-worklog.md`, because this is attributed narrative for a human (or a fresh epic session) to read, not a machine-parsed event stream like the JSONL `.epic-notifications.log`.

```
## 2026-08-28T09:00:00Z — NEGATIVE (conductor)
<action not taken, and why — e.g. "i2080 not proposed: clone stood down 08-27 for <reason>, prerequisites merging today doesn't reopen it">

## 2026-08-28T14:30:00Z — RELAY:FINAL (conductor→epic)
<the decision, restated in full, not a paraphrase of sentiment>

## 2026-08-28T14:32:00Z — FINDING (worker i2098, forwarded by conductor)
<the worker's finding, verbatim>
Conductor's own reading, if any, marked separately: <...>
```

**Not `.epic-worklog.md`.** That file is per-slot and gets `rm -f`'d on slot cleanup, so a fleet-level entry written there is destroyed by unrelated housekeeping the moment that slot is cleaned up.

## Configuration

The conductor skill reads `.epic-config.json` from the repo root — the same file `/bip-epic` reads.
This file is gitignored and must exist before either skill can operate; the conductor owns creating it.

**Clone mode** (remote compute or pre-existing clones):
```json
{
  "clone_root": "~/re/myproject",
  "clone_names": ["alpha", "beta", "gamma"],
  "new_clone_names": ["delta", "epsilon", "zeta"],
  "github_repo": "org/repo",
  "conductor": "alpha",
  "max_lead_iterations": 8
}
```

**Worktree mode** (local parallel work only):
```json
{
  "clone_root": "~/re/myproject-workers",
  "local_worktrees": true,
  "github_repo": "org/repo",
  "max_lead_iterations": 8
}
```

**Validation**: If `local_worktrees: true` and `clone_names` are both present, **stop and report an error** — they are mutually exclusive.
`clone_names` is meaningless in worktree mode because slots are created on demand and named after the issue.

Fields:
- **clone_root**: Parent directory containing all clones or worktrees.
  Also where the shared `.spawn-prompts/` intent directory and any unfiled `ISSUE-*.md` drafts live.
- **clone_names**: (clone mode only) Existing clone directory names
- **new_clone_names**: (clone mode only) Names available for creating new clones
- **local_worktrees**: (worktree mode) If `true`, use `git worktree` for local slots named `issue-N`
- **github_repo**: `org/repo` for `gh` commands
- **conductor**: (clone mode only) Which clone is the orchestrator (stays on main)
- **max_lead_iterations**: Max issue-lead evaluations before escalating to `needs-human` (default: 8)
- **shared_filesystem**: (optional, default `false`) Set to `true` when the conductor and all compute nodes share an NFS filesystem; the conductor composes direct SSH execution commands instead of `make remote-sync` calls, and experiment results are immediately visible on local NFS paths.
  Each machine sets this flag for itself — no central list of NFS nodes is needed.

## Workflow

### Step 1: Load config, or set it up

```bash
cat .epic-config.json
```

**If the file does not exist**, stop and ask the user:
1. Are you using local git worktrees or separate clones for parallel work?
2. Where should slots live?
   (e.g. `~/re/pz-workers` for worktrees, or `~/re/pz` for clones)
3. (Clone mode only) What are the clone directory names?
   Which is the conductor?
4. What is the GitHub repo (`org/repo`)?
5. Are compute nodes on a shared NFS filesystem?
   (sets `shared_filesystem`)

**Note (worktree mode)**: The skill is run from the main repo itself, which acts as the conductor.
There is no separate conductor clone — `clone_root` is just where worktrees are placed, not the main checkout.

Then create `.epic-config.json` with their answers and proceed.

All subsequent steps use values from this config — never hardcode paths or clone names.

**Self-register for completion pushes**: resolve `CLONE_ROOT` and write this session's own `ListAgents` name (the "This session is ..." row) as the sole line of `$CLONE_ROOT/.conductor-session` — see the Conventions section's "Completion pushes" for why this file exists and how it's kept fresh.

**Read every TRACKED subdirectory `CLAUDE.md` in the project repo — they are NOT auto-loaded.** Only the repo-root `CLAUDE.md` and the user's global one land in context automatically. A subdirectory one loads when a session *works in* that directory, and the conductor works from the repo root reading artifacts by path, so it never loads them. Enumerate with `git ls-files | grep 'CLAUDE\.md$'` — **not `find`**, which keys on filename rather than tracked-ness and returns vendored copies and stale nested clones indistinguishable from the live file.

**The conductor looks less exposed than the epic and isn't.** This skill tells you to re-derive a peer's number before acting on it — so reading a project's experiment artifacts is in-role, not a stray into the epic's lane. Measured 2026-09-11: doing exactly that meant reading a results `PROVENANCE.md`, an experiment README, committed TSVs and a fixture directory, all under `experiments/`, with `experiments/CLAUDE.md` never loaded. That file documents the `results/`-is-gitignored-with-per-file-negations pattern whose resolution then cost a wrong "input absent" that was relayed to the epic as verified.

### Step 2: Pull main

```bash
git pull --ff-only origin main
```

If this fails, report the problem and continue with stale state.

### Step 3: One-time unfiled-draft sweep

Do this only at cold start, not on every poll — it's a init-time check, not an ongoing one.

`/bip-issue-file` moves a draft to `_ignore/` the moment it successfully files it, so anything still sitting loose as `ISSUE-*.md` directly in a clone root **is** unfiled, by construction (an `update` operation reuses the file in place, so a draft that's mid-revision on an existing issue is expected here too — don't treat those as orphans).

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
find "$CLONE_ROOT" -maxdepth 2 -name 'ISSUE-*.md' -not -path '*/_ignore/*'
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-conductor`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory (see `skills/lib/spawn-intent.sh` in the `bipartite` repo).

**Never auto-delete or auto-file these.**
An unfiled draft is authored state — the only copy of someone's unfinished thinking — and at least one sampled case turned out to be the most valuable item in the batch after sitting unfiled for a day.
Just flag what you found (path, clone, first line) so the user can decide.

**The conductor clone is outside `clone_root`, so this sweep cannot see the conductor's own drafts — and that is deliberate, not a gap to close.**
`clone_root` is the worker pool; the conductor clone (and, on this layout, the epic session's cwd) sits beside it, so a draft authored there is invisible to the command above.
The tempting fix is to add that directory to the sweep. **Don't** — measured on `matsengrp/phyz`, 2026-09-12: the conductor clone holds **35** loose `ISSUE-*.md` at depth 1, of which the epic session considered **two** live.
Recency does not separate them (10 touched in the last 24 h, 23 in the last 7 days) because most are drafts edited in passing or superseded fragments from earlier in the same EPIC.
Extending the sweep would trade an under-report of 3 for an over-report of 33, and the larger number looks more authoritative precisely because a command produced it.

**The reason no sweep fixes this: a filename pattern is not a liveness signal.**
`_ignore/` marks *filed* — that is what `/bip-issue-file` uses it for, and the `-not -path` above already excludes it.
Nothing marks *abandoned*, and nothing can, because liveness is a fact about intent that lives only in the session that authored the draft.

**So the reporting contract, not the sweep, carries it: the epic session owns its own draft list and tells the conductor; the conductor reports what it is told and does not go looking.**
A conductor that discovers it under-reported the epic's drafts should route the fix there rather than widening `find`'s universe.

### Step 4: Fan out the clone/tmux scanner

Dispatch one `general-purpose` subagent — single call, following the dispatch pattern in `SUBAGENT-SCAN.md` (bipartite repo root).

**An agent's report is evidence about the question you asked it, and carries your own framing back unchanged on everything else.** So the place to catch a bad premise is the brief, not the report: separate a claim from its attribution before briefing, and verify the attribution yourself — it is the half that never gets tested. See `/bip-epic`'s briefing guidance for the worked instance.
Brief:

> Inventory clones/worktrees.
> Read `clone_root` and `local_worktrees` from `.epic-config.json`.
>
> Clone mode (`local_worktrees` absent or false): iterate `clone_names`; for each, capture branch, last commit, dirty files (max 5), and `.epic-status.json` contents.
>
> Worktree mode (`local_worktrees: true`): `find $CLONE_ROOT -maxdepth 1 -name 'issue-*' -type d`; for each, capture last commit, dirty files, and `.epic-status.json`.
>
> Also: `tmux list-windows -F "#W"`.
> And, for pending spawn intent from the epic: `find $CLONE_ROOT/.spawn-prompts -maxdepth 1 \( -name '*.md' -o -name 'spawn-*.txt' \)` — **both patterns**, since `<N>.md` and the older `spawn-<N>.txt` are both live conventions in that directory; a `-name '*.md'`-only find silently misses a real, current `spawn-<N>.txt` intent file.
> `-maxdepth 1` also matters here: `/bip-conductor-spawn` moves a launched intent file into a `.spawn-prompts/consumed/` subdirectory instead of deleting it, and this find must not descend into `consumed/` and report an already-launched file as still pending.
>
> Classify each slot:
> - `occupied`: has tmux window (regardless of agent status — user may be doing follow-up work)
> - `stale`: no tmux window, but has `.epic-status.json` or is on non-main branch
> - `available`: (clone mode) no tmux window, on `main`, clean, **and current with `origin/main`** -- see "Clean is not current" below
>
> Return under 400 words:
> - `active_items`: per slot: name, phase, summary, scope, stop_reason, lead_guidance (from `.epic-status.json`), classification
> - `action_candidates`: stale slots ready for cleanup (clean up ONLY if no tmux window — never kill tmux windows); pending spawn-intent files (either naming pattern) with an idle clone to run them
> - `surprises`: phase migrations (`blocked`/`pr-review`), missing status files, contradictions

### Clean is not current, and that failure goes the other way

⛔ **"On `main`, clean" is not a readiness predicate. A clone can satisfy it while being arbitrarily far behind, and nothing reports it.** Measured 2026-09-15 on `matsengrp/phyz`: **14 of 17 slots** were on `main` with zero dirty files and **behind by 2 to 189 commits** -- aspen and beech at 189 (head six days old). The conductor's own cold-start dashboard reported all 17 as spawn-ready.

⭐ **This is a DIFFERENT SHAPE from the probe failures catalogued elsewhere in this file, and worse. Those all fail toward "nothing to do"; this one fails toward "go ahead."** A wasted check costs a command. A worker spawned 189 commits stale costs a result.

Run `lib/clone-currency.sh` (sibling of `fleet-collisions.sh`) before every spawn. Three things it gets right that a hand-rolled loop does not:

- **It resolves the tip ONCE, in the conductor, after an explicit `git fetch`** -- never from each clone's own `origin/main`. A clone that has not fetched has a stale remote-tracking ref and reports **0 behind while being arbitrarily far behind**, which is the reassuring answer.
- **It counts in the CONDUCTOR's object DB**, because the clone may not have the objects; a per-clone `rev-list` returns empty and reads as "?" or as zero depending on how you wrote the loop.
- **It guards ancestry before counting.** `A..B` returns a meaningless number when `A` is not an ancestor of `B` -- the same range-vs-tree trap this repo keeps re-finding. It reports `DIVERGED` rather than a number.

⚠ **Its universe is `clone_names` from `.epic-config.json`, NOT every directory under `clone_root`, and that is deliberate.** The pool also holds CI clones -- on `matsengrp/phyz`, `nightly-ci` (30 behind) and `beagle-weekly-ci` (97 behind), both named directly in their systemd units' `ExecStart`. **Both are fine**: each script `git fetch && git reset --hard origin/main`s and re-execs itself at run time, so a stale HEAD at rest is the expected state there. A sweep over every directory flags them as NOT-READY every time -- a check at full power over the wrong population. They are listed as `(unmanaged)`, informational, never gated on.

⛔ **And the part that is NOT this check's job: a behind-count cannot report DIRECTION.** *"N behind"* after your own PR merges is the ordinary post-land state. *"N behind"* while a result is still in flight is `CLAUDE.md`'s repo-moved-under-a-finished-result trap. **Same number, opposite meanings**, and the count alone cannot tell you which. Measured the same night: three slots each read "2 behind", and all three were fine because both commits landed *after* their PRs merged -- establishing that took merge timestamps, not counts.

➡ **So for an idle slot, use the count and fast-forward. For a FINISHED RESULT, the question is different and so is the instrument: take the build commit from the artifact's own `phyz_build` field (`--summary-tsv`/`metadata.json`), and run the TREE form `git diff <build-commit> <tip> -- src/ build.zig build.zig.zon`.** ⭐ **The reference behaviour is a worker's, not this skill's**: `matsengrp/phyz` PR #2724 did exactly that unprompted, and replaced its hand-written build-currency prose with an automated `results/provenance_ok.txt` gate.

If work will run on remote hosts, also check `/bip-scout` for host occupancy (live builds, warm caches, other sessions' remote jobs) — this is exactly the kind of fleet fact `/bip-epic` cannot see and the annotation step in `/bip-conductor-spawn` depends on.

### Step 5: Build status display

The dashboard is **slot-centric** — the epic's dashboard is issue-centric; this one is about what's occupying the fleet.

| Slot | Classification | Issue | Phase | Notes |
|------|-----------------|-------|-------|-------|
| cedar | occupied (tmux `281-cedar`) | i281 | coding | — |
| pine | occupied (tmux `295-pine`) | i295 | awaiting-results | ~Tue |
| fir | stale (4d) | i589 | needs-human | check if experiment finished |
| hazel | available | — | — | — |

**Pending spawn intent**: list any `.spawn-prompts/` files found in Step 3/4 (either naming pattern), with the available slots that could run them.

**EPIC distribution**: one line counting live slots by the `EPIC:` header of the brief that spawned them, with user-originated slots as their own row:

```
EPIC #369: 5 slots    user/bip-ms: 2 slots    (no header): 1 slot
```

This is counting, not topic reasoning — the conductor holds no topic boundary (see "Two intake paths") and must not judge the distribution. But it is the one fleet-side signal that makes topic drift visible at all. Measured 2026-09-03: a single-epic session had **10 of 15 live slots outside its own EPIC**, spanning four EPICs, and nothing surfaced it for most of a day because no view counted this. The distribution row would have shown it on the first poll. `(no header)` is a real category, not an error — see the spawn skill's `EPIC:` requirement for why a brief might lack one.

**Negative list**: also surface decisions already taken *against* an action, with reasons — sequencing already applied, an issue already stood down for a reason that would otherwise look resolved, and similar.
This is the one category of fleet fact the epic (or a fresh conductor) cannot re-derive from `git`/`tmux`/`gh`: it's the absence of work, which leaves no trace in any of those.

### Not everything you know is a fleet fact — the embargo trap

**A cross-arm pattern feels exactly like a fleet fact, and the fleet-facts block of a spawn prompt is precisely where you write "things only the conductor can see." That is how an epic-session embargo gets spilled by the one session positioned to spill it.**

The epic decides what is embargoed; **you are the delivery path**, so the discriminating test has to live here. It is **not** *"do I know this?"* — you know both kinds. It is:

> **Is this mechanical state, or a conclusion another arm is supposed to reach on its own?**

Slot occupancy, host load, build state, file collisions, who landed what: the first. **A synthesis across arms, a prediction about what an arm will find, or a mechanism one arm inferred that another is independently testing: the second — and it does not go in a prompt, a fleet-facts block, or a nudge.**

The reason is not tidiness. Arms reaching verdicts independently is the *only* thing that makes their agreement mean anything; a pattern arriving as a premise converts a test into a confirmation and destroys the evidence it was supposed to produce.

**Worked instance (`matsengrp/phyz`, 2026-09-06), and it is the case that justifies the practice rather than the case where it looked clever:** an epic session withheld a prediction that four topologies would fail to survive a corrected protocol. The arm tested them independently and **falsified it** — all four survived. Had the prediction been relayed, it would have arrived in the one population where confirmation was cheapest, on a question carrying a rung at zero margin. **A withheld idea that turns out wrong is better evidence for withholding than one that turns out right.**
Read `.epic-decisions.md` in the conductor cwd (see Conventions, "Decision relays" and ".epic-decisions.md: the durable fleet-decision log") for prior entries and append any new one here, timestamped and attributed, in the same step you surface it — don't let it live only in this dashboard render.
Concrete shape from the run that motivated this: an issue whose stated prerequisites both merged the same day reads as unblocked, but the conductor had already stood a clone down for it for an unrelated reason — without the negative list, that reads as ready to the epic and gets proposed again.

### Preflight, indexed by ACTION rather than by topic

⛔ **Check this table by what you are about to DO, not by what the work is about.** Measured 2026-09-14: every rule that would have prevented that night's worst conductor error was loaded, correct, and indexed under *measurement* — none fired when the action was "file an issue."

| about to… | check | cost of skipping it, measured 2026-09-14 |
|---|---|---|
| **spawn** | run `lib/fleet-collisions.sh`; confirm the issue body has not changed since its brief was written | a duplicate slot spawned onto work already in progress; a three-way file collision missed |
| **file an issue** | does a success criterion name a denominator, a population, or "a default run" without naming the dispatch path? | #2627 shipped a stderr regression AND a wrong fraction — the worker followed the criterion correctly |
| **deliver a correction to a worker** | append it to that slot's `.epic-worklog.md` in the same step | a correction delivered only by message does not survive the worker's next compaction |
| **reclaim a slot** | all three state files, including `.claude/ralph-loop.local.md` | a preserved `.epic-status.json` survived a window closure and made an idle clone read as occupied — silently |
| **report a number you did not compute** | re-derive it, or relay the basis — *"it reports X"*, not *"X"* | a relayed "its suite passed" needed retracting when the session's background tasks died with it |
| **close or reopen an issue** | verify the criterion against `main`, not against the PR that claims it | #2636 sat open for two hours asserting a build was broken after it had been fixed |
| **land a PR** | run `/bip-pr-land`. Never a hand-rolled `gh pr merge` | a bare `gh pr merge --squash` omits `--body`, so `gh` concatenates every branch commit body into the merge message and GitHub parses it — #2620 auto-closed against the PR body, the author, and two reviewers, and the bypass also skips worklog preservation |

⭐ **The cost column is load-bearing, not decoration.** A tired conductor skips a rule; it does not skip a rule with last night's scar attached. When a row's incident is superseded by a worse one, replace it — an entry whose cost has gone stale is the first one to be ignored.

### Run `lib/fleet-collisions.sh` before every spawn

```bash
"$(dirname "<this-skill's-base-directory>")/lib/fleet-collisions.sh"   # exit 0 = clear, 1 = found, 2 = could not check
```

**This check cannot be done from the epic side and is not a courtesy hand-off.** Clone branches are local (`shared_filesystem: false`), and remote refs are not a substitute: under squash-merge every historical branch stays permanently ahead of `main` — measured on `matsengrp/phyz`, **847 remote branches, the first 400 all ahead** — so "ahead of main" does not discriminate live from long-dead. The decisive case is **uncommitted** work, which exists only in the clone. A three-way collision on one test file was visible on 2026-09-14 solely in `git status` output on the conductor's machine.

It reports four things: live branches and their touched files; **files touched by more than one live clone**; **a `.epic-status.json` whose clone has no live pane**; and **a live pane with no status file**.

⭐ **Those last two are one asymmetric check and the second direction is the worse one.** A stale file makes an idle clone read busy and costs a spawn. A *missing* file makes a busy clone invisible to every state-file sweep **including the phase monitor** — so a stall there produces silence rather than a stale timestamp, and a conductor can spawn a second worker into an occupied clone. Both were live on 2026-09-14, hours apart.

⚠ **And the general lesson, which is why both directions are in one script: fixing one direction of an asymmetric check is the moment you are least likely to examine the other.** The missing-file section exists only because the epic asked about the direction the conductor had just stopped looking at, and it fired on its first run.

⛔ **Exit 2 means "could not check" and must not be read as clear.** If `tmux` returns nothing while status files exist, the script refuses to report the pool stale and exits 2 — because the action on a STALE report is deleting a worker's state file.

### Unattended scheduled load is invisible to a worker and it will blame itself

Before spawning, and before telling any slot to run a full suite, check what the machine is already committed to that no slot owns:

```bash
systemctl --user list-timers phyz-nightly-test.timer --all
uptime
```

⛔ **Use the timer, not `is-active` on the service.** The service returns `inactive` the moment it finishes, so a worker whose build died at 02:50 and who checks at 03:15 reads `inactive`, concludes contention was not the cause, and goes back to blaming its own branch — the exact self-blame this rule exists to prevent, displaced by half an hour. `list-timers` answers both questions in one line:

```
NEXT                        LEFT  LAST                              PASSED
Tue 2026-09-15 02:31:05 PDT 22h   Mon 2026-09-14 02:36:20 PDT  1h 23min ago
```

⛔ **THE NEXT-FIRE TIMESTAMP IS NOT A QUOTABLE FIGURE — `RandomizedDelaySec=15min`, AND SYSTEMD RE-RANDOMIZES THE PENDING VALUE.** Measured 2026-09-15 on the same unit with **no run in between**: `NEXT Wed 02:34:06` at 16:26Z, `NEXT Wed 02:41:15` at 18:40Z, both against `LAST Tue 02:31:05`. `systemd-analyze calendar '*-*-* 02:30:00'` resolves flat to `02:30:00`, so the entire offset is the randomization.

➡ **Tell a worker the WINDOW, never the timestamp: fires 02:30–02:45, and a build killed anywhere in 02:30–04:00 is the nightly's ~90-minute ReleaseSafe suite rather than a defect in its branch.** A worker handed `02:34:06` who sees nothing at `02:35` concludes the timer is broken; a worker handed the window does not.

⛔ **The previous version of this line is worth keeping as a worked failure, because it is a class we had no name for: A DOC THAT GETS THE FACT RIGHT AND THE MECHANISM WRONG, THEN DERIVES A REMEDY FROM THE MECHANISM.** It read *"Scheduled 02:30, observed start 02:36:20 (`OnCalendar`, `Persistent=true`). Cite both."* ⭐ **The observation was true and checked out on inspection. `Persistent=true` was the wrong cause, and "cite both" is the instruction that follows only if that cause is right** — so it told a conductor to quote a value that will have moved by the time the worker reads it.

⚠ **This failure mode is invisible to the obvious audit, which is why it survived: anyone re-deriving the MEASUREMENT confirms it.** Only re-deriving the *mechanism* exposes the remedy. **When a doc pairs a number with a cause, check the cause separately — the number passing tells you nothing about it.**

On `matsengrp/phyz` the nightly is `phyz-nightly-test.timer`, the ReleaseSafe suite, running ~90 minutes on `pax` — the same workstation the fleet runs on. Measured 2026-09-14 at 09:43Z: **33 processes under `/tmp/phyz-nightly-ci`** plus one slot's full `zig build test` at 20, load average **48.77** on 32 cores, and three of another slot's `zig build` invocations killed.

**The failure is not the contention, it is the attribution.** A worker cannot see scheduled load it does not own, so it explains the symptom with whatever mechanism is to hand. The slot above reported its kills as "a low-memory watchdog" — with **106 Gi available**, `earlyoom` **inactive** (not even a unit), and no OOM line in 30 minutes of kernel log. It then lowered `-j`, which does nothing against a box oversubscribed by someone else.

This belongs to the conductor specifically: it is the only party positioned to look, the same reason the modal sweep and the host-load checks live here. Tell the affected slots the cause, and tell them **a build killed in that window is contention rather than a defect in their branch** — otherwise they re-diagnose their own changes over it.

### Step 6: Propose next action

First, do housekeeping automatically (no need to ask):

**Closing a finished worker's window is hygiene, not a decision to escalate.** The gate is not whether a window is open — it is whether anything in it is still wanted:
- **Reclaim** when the issue is closed (check `gh`, not `phase`), the clone is on `main` and clean, work is preserved (below), the prompt line holds nothing authored, the session itself reports exactly `idle` in `ListAgents`, **and no process has a working directory inside that clone.**

  ⛔ **THAT LAST CONDITION IS NOT REDUNDANT, AND THE OTHER FOUR CAN ALL HOLD WHILE THE WORKER IS STILL WORKING.** Measured 2026-09-15 on `matsengrp/phyz`: a conductor was one command from killing a finished-looking window when **every** file-and-prompt condition was satisfied — issue CLOSED per `gh`, clone on `main` and clean, work preserved, `cursor_x = 2` with only a dim autosuggest wrapper, no modal, **and both `.epic-status.json` and `.epic-worklog.md` already deleted.** The worker was `busy`: it had just spawned its final issue-lead check.

  ➡ **The trap is that "the state files are gone" reads as "finished" and is not.** `/bip-pr-land` removes them at its cleanup step, while the worker's own prompt tells it to invoke the lead **one final time after landing** — so there is a real window where the files are gone, the issue is closed, the clone is clean, and a subagent is mid-run. **The file conditions are necessary and not sufficient.**

  ⚠ **What you lose by killing into that window is the lead's terminal ceremony and any follow-ups it was about to file — and the symptom is silence**, indistinguishable from a lead that legitimately filed nothing, which is a real and common outcome. Nothing reports it. Same class as *"a landing emits no terminal event by construction"* elsewhere in this file: an absence of state read as completion, one step later in the lifecycle.

  ⛔ **`idle` IS AN ALLOWLIST. NOT-`busy` IS NOT THE TEST.** `ListAgents` has at least four states and all four were live on one box on 2026-09-15: `idle`, `busy`, `waiting`, `shell`. **`waiting` is the dangerous one** — a session blocked on a permission prompt or an approval is not busy and is emphatically not finished; killing into it destroys whatever it was about to ask for, with the same silent symptom described above. `shell` is likewise not "done". **Reclaim on `idle` and on nothing else.**

  ⚠ **It is a point-in-time read and the kill is a later command — re-check immediately before the kill, not once at the top of the sweep.** A worker can be re-invoked in between by a queued message or a loop, and the instance above is the proof the window is narrow: that slot went `busy` *because it spawned something*, which can happen at any moment.

  ⭐ **`ListAgents` is the check, and the pane is not.** The pane showed no `FINAL RECAP`, which reads identically to "already scrolled off." One `ListAgents` row said `busy`.

  ⛔ **AND `idle` IS STILL NOT ENOUGH, BECAUSE IT DESCRIBES THE CLAUDE SESSION AND NOT WHAT THAT SESSION LAUNCHED.** A `run_in_background` build is detached by construction and **outlives the window**. Measured 2026-09-15, eleven minutes after the `idle` clause above was pushed: a slot reclaimed at ~14:57Z — `idle` before the kill, on `main`, clean, no state files, nothing authored at the prompt — had a live `zig build --cache-dir .zig-cache-<clone> -j8` in it at 15:08Z, **elapsed 221 s, so started roughly eight minutes AFTER the window was killed**, still parented to a surviving shell-snapshot `zsh` with no `claude` process anywhere in the clone.

  ➡ **So the state you need is "nothing live in the clone", and neither the file conditions nor the session state reports it.** The check is the same `/proc` idiom this file already prescribes for fleet-scoping, pointed at reclaim:

  ⛔ **BUT THE CHECK HAS TO RUN AFTER THE KILL, NOT BEFORE — AND THIS IS WHERE AN EARLIER DRAFT OF THIS VERY CONDITION WAS WRONG.** Before the kill, the slot's **own** `claude` session and its shell have a cwd in the clone by construction, so a naive "is anything live here" check fails for **every open window** and the condition is unusable. Measured 2026-09-15: a finished, `idle`, fully-preserved slot reported **2 live processes** — its own session and its own `zsh`. Nothing was wrong.

  ➡ **So the order is: confirm the other conditions, kill the window, THEN verify quiescence before marking the clone free.** That is also the order that catches the real case — a detached build **survives** the kill, so only a post-kill check distinguishes it from the session you just ended.

  ```bash
  # AFTER tmux kill-window. Anything still here survived the kill.
  live=0
  for pid in $(pgrep -x zig; pgrep -x claude; pgrep -x zsh; pgrep -x rsync; pgrep -x python3); do
    [ "$(readlink /proc/$pid/cwd 2>/dev/null)" = "$CLONE" ] && { live=$((live+1)); echo "SURVIVED: $pid $(ps -o comm= -p $pid)"; }
  done
  [ "$live" -eq 0 ] && echo "free" || echo "NOT free: $live"
  ```

  ⚠ **And sample twice, because a `claude` process does not exit the instant its window closes.** Measured immediately after the kill above: the session's own pid was **still present**, which reads exactly like the orphaned-build case and is not — it was mid-teardown and gone shortly after. This file already records the general form (*"a state observed DURING a transition is not a state"*); **a window you just closed is mid-operation by construction.** Poll until it clears rather than concluding from the first sample:

  ```bash
  until ! [ -d /proc/$PID ] || [ "$(readlink /proc/$PID/cwd 2>/dev/null)" != "$CLONE" ]; do sleep 3; done
  ```

  ⚠ **What it costs to skip: the next spawn builds on the same `--cache-dir` concurrently**, which is the documented stale-futex hazard whose symptom is a build hanging at 0% CPU. And a "which cache is live" guess does not save you — the clone in that instance carried **three** cache directories.

  ⭐ **The invariant behind all three refinements — files, then `+idle`, then `+no live process`, all three found by USING the gate rather than reading it: every condition here describes AN ARTIFACT OR A SESSION, while the property you actually need is THAT NOTHING IS STILL ACTING ON THE CLONE.** Each condition is a proxy for that, which is why they accrete.

  ⚠ **The `cwd`-inside check is closer to the invariant than the other two and is still a proxy. Its hole, named rather than left open-ended: a process acting on the clone from OUTSIDE it.** `git -C <clone> …` (verified: runs from any cwd), an `rsync` or `make remote-sync` writing in, a script emitting results by absolute path — all pass a `cwd`-inside check while writing to the clone. ⭐ **It does NOT include a stray `zig build`, which is the tempting example and is wrong: `zig build` requires `build.zig` in its cwd** (verified — it errors with *"initialize build.zig template file"* otherwise), so a build acting on the clone necessarily has its cwd there and the check catches it. **Know which of those two your residual is; "something might be acting from outside" is bounded and checkable, "expect a fourth condition" is not.**

  ⛔ **And the structural fix, because proving ABSENCE is what makes this accrete: the OS cannot cheaply answer "is anything still acting on this directory", so every condition above is a proxy and always will be. Require PRESENCE instead — have the landing step write a completion token LAST, after everything else, and reclaim only on that token.** A token survives the session's death, cannot be faked by an idle pane, and is unaffected by detached children, because **whoever writes it writes it after they are done rather than an observer inferring doneness from outside.** Note this is the same shape as the `/bip-pr-land` state-file finding elsewhere in this file — the durable artifact belongs somewhere that is not a pooled clone whose state gets deleted mid-lifecycle. **The absence-reads-as-completion trap is structurally solved by a positive signal and only ever mitigated by more proxies.**
- **Hold** when there is genuinely typed, unsubmitted input at the prompt — but **text being present at the prompt is not evidence that anyone typed it.** Claude Code's autosuggest pre-fills the composer with a dim suggestion on an idle window, and it fires on *essentially every* idle window. `tmux capture-pane -p` **strips ANSI escapes**, so autosuggest and real input are byte-identical in its output: a check that only greps for text answers "yes" always, and every finished window becomes permanently unreclaimable.

  Two independent signals separate them. Verified in both directions, 2026-09-04:

  ```bash
  # cursor position — the primary check, because a human can audit it on screen
  tmux display-message -p -t <pane> '#{cursor_x} #{cursor_y}'
  # escape codes — more robust to wrapped/multi-line input
  tmux capture-pane -p -e -t <pane> | grep -a '❯' | tail -1 | cat -v
  ```

  | | prompt row | escapes | `cursor_x` |
  |---|---|---|---|
  | **autosuggest** | `❯ check progress` | `^[[2m`…`^[[0m` wrapper | **2** — start of input, text to its right |
  | **really typed** | `❯ this is some real text` | no wrapper | **2 + len** — end of the text |

  Column 2 is just past `❯ ` (glyph + `U+00A0`). Prefer the cursor test: the dim attribute may not render visibly at all depending on terminal theme, so a rule built on it is one a human cannot verify by looking. Check both and **hold on disagreement**. The old `U+00A0` caveat (bytes `302 240`, not stripped by `tr -d ' '`) still applies to whitespace-only *real* input — strip with Python `.strip()` — but it hardens the wrong layer if used as the primary check.

- **There is no such thing as a stop-hook leftover in the composer.** The ralph-loop `Stop` hook (`ralph-loop/hooks/stop-hook.sh`) continues a loop by printing `{"decision":"block","reason":<prompt>}`, which blocks session exit and feeds the prompt back as the next turn's input — a programmatic path that never touches the prompt line. When the loop ends it `rm`s `.claude/ralph-loop.local.md` and exits silently. Confirmed from source and independently by the user, who has never observed anything but autosuggest put text in the box. **Do not build a heuristic to distinguish hook artifacts from authored intent; the category is empty.** Read the cursor.

- **To test whether a loop is live, read its state file, not the pane** — a completion marker scrolls off, and `.claude/ralph-loop.local.md` does not. But existence alone is not liveness: the hook exits on a `session_id` mismatch, so a file from a dead session is inert while still looking live. The check is *file exists **and** its `session_id` belongs to a running session*.
- Reclaim: *clone mode* `git checkout main && git pull --ff-only`, remove `.epic-status.json`; *worktree mode* `git worktree remove --force $CLONE_ROOT/issue-N && git branch -d <branch>`.

**Preserve the worklog BEFORE reclaiming any slot, including one that landed a PR.** It used to be assumed that a slot finishing via PR "needs nothing" because the issue-lead posts every evaluation as a PR comment (`gh pr comment`) and `PROSE-DISCIPLINE.md` routes revision narrative there by design. That premise is false (issue #2216, falsified concretely by matsengrp/phyz#2314/PR#2316): PR bodies are deliberately rewritten to current state, so deliberation never lives there, and the issue-lead's PR comments are evaluation-stop summaries, not the worklog's continuous narrative — on a landed PR the worklog is one of only two places the reasoning behind the change survives (the other being the PR/issue comment thread, which lives on **GitHub**, not in git objects: a `git clone` alone does not carry it). `/bip-pr-land`'s own Step 6a now preserves it (Step 9.5 does the actual deletion) at land time, before Step 8 can destroy a worktree — the earliest point that's guaranteed to run — do not rely on reclaim to catch a landed slot; treat reclaim finding the files still present as a sign that step was skipped. A slot that stands down or escalates has no such upstream step at all and carries its **entire** value in `.epic-worklog.md`, which is gitignored, lives in a pooled clone, and is deleted by `/bip-conductor-spawn`'s own prep on the next assignment. Copy it to `$CLONE_ROOT/.preserved/<slug>/` with a provenance README first. Measured 2026-09-03: eleven such worklogs were rescued during one stand-down, the largest being 269 lines from a slot that wrote **no code at all** — **and reclaiming first would have destroyed it leaving no trace it had existed.**

**Re-preserve unconditionally at reclaim — don't test whether a `.preserved/` copy exists, don't diff it, just refresh it before deleting.** The tempting cheaper guard is a refusal keyed on "worklog never preserved," but that predicate is *absence*, which is not the thing you care about. Measured 2026-09-11: `cedar`'s worklog was preserved at 397 lines, the worker kept appending, and reclaim 40 minutes later found 427. A `.preserved/` directory existed and would have passed an absence check while 30 lines were missing — the guard reports clean on the exact variant it was written for. Don't substitute a content comparison either; that is just another predicate that can be wrong, where an unconditional refresh has nothing to be wrong about.

**A short gap is the dangerous case, not a reassuring one** — forty minutes reads as "we preserved it at the end," and cedar's missing entries carried back-dated headers, so the file looked internally consistent. What a stale snapshot preferentially destroys is **deliberation, not results**: results survive in artifacts by construction, while cedar's 30 lines were a contested-authorization episode that existed nowhere else.

**A worklog that IS already gone is usually recoverable from the slot's session transcript, byte-identically — check before writing it off.** Measured 2026-09-13: a slot's `.epic-worklog.md` and `.epic-status.json` were both deleted with no `.preserved/` entry, and the worklog was rebuilt from `~/.claude/projects/-home-matsen-re-pz-<clone>/<session-id>.jsonl` at **15,899 bytes, identical to the original** (which turned up afterwards in the wrong directory, and served as the check). The directory name is the clone's **absolute path with every `/` replaced by `-`, leading one included** — `~/re/pz/alder` becomes `-home-matsen-re-pz-alder`; derive it with `echo "$PWD" | sed 's|/|-|g'` rather than typing it, since a conductor doing this is already in a bad moment. Replay the tool calls that wrote it, in timestamp order: the seed `Write`'s `input.content`, then each `Edit`'s `old_string`→`new_string` applied to the accumulating document, then any `Bash` heredoc appending to the file. **Verify as you go — an `Edit` whose `old_string` does not match means the replay has diverged and the result is a plausible-looking fabrication, which is worse than a gap.** Mark the output as a reconstruction in its `README.md`, say which writes were replayed, and note that entry headers are the worker's own hand-written timestamps rather than file mtimes. **Two limits:** it cannot capture a write the transcript does not show, and `.epic-status.json` is usually *not* recoverable this way, since it is typically rewritten wholesale by paths that leave no replayable content.

**Two failure modes make this worth knowing in advance rather than after.** Preservation can write to the *wrong place* — `<clone_root>/<clone>/.preserved/` instead of `<clone_root>/.preserved/` — which reads as "never preserved" to any check looking in the right directory; before concluding loss, `find` the clone itself. And it can not run at all: the `bip-pr-land` defect fixed on 2026-09-13 left the helper undefined and every guard reporting a *handled error* rather than a missing helper.

**One clause an unconditional refresh needs, or it creates a third variant: never overwrite a larger preserved artifact with a smaller live one — write alongside it.** The same night, `maple`'s live worklog was a 41-line post-land summary while `.preserved/` held the 467-line original; a blind live-over-preserved copy would have destroyed it. The two clones diverged in opposite directions — cedar short at the preserved end, maple short at the live end. Give the smaller one its own filename and keep both.

#### A preserved-artifact README is not write-once

⛔ **The README you write when you preserve something is a claim about what the data MEANS, and the world keeps moving after you write it. Revisit it whenever something lands that changes what its contents mean.**

⚠ **Measured 2026-09-16 on `matsengrp/phyz`, and the shape is worth more than the instance: the file was WRONG TWICE while every sentence in it stayed ACCURATE.** A directory of superseded measurement outputs was preserved at 18:37Z with *"superseded by a scope narrowing, **not by a defect**"* — true and complete when written. Three hours later a worker in a *different slot* found a defect that had been live when those outputs were produced, and the label had become too favourable **without a word of it changing**. A later ruling then withdrew the one remaining use it had allowed. **Three rulings, two tightenings, and not one fact retracted.**

⭐ **That is a distinct failure mode from a stale number, and worse in one specific way: a stale number can be re-derived and caught, this cannot, because there is nothing to check it against.** The facts all verify. What went stale was the document's *stance* — and nothing in the file, and no sweep over it, can detect that. It is `PROSE-DISCIPLINE.md`'s *"the authoritative-looking copy is the one that goes stale"* one level up: not a value going stale inside a document, but a whole document's posture going stale while its values hold.

➡ **Two practices, both cheap:**
- **When a defect lands, `grep -rl` the preserved directories for artifacts produced before it.** A fix in one slot routinely invalidates a label written in another; nothing connects them automatically, and the slot that wrote the label is usually gone.
- **Put a dated "age of this label" line in every preservation README** — *this rule has been tightened N times since <date>; check whether anything has landed since.* It costs one line and it tells a reader in 2027 not to trust the label's age.

⚠ **This is NOT a licence to append revision narrative generally.** `PROSE-DISCIPLINE.md` bans it in issue and PR bodies and that ban holds. A preservation README is a **warning label on data**, not an argument, so its own credibility history is operational for the reader — which is the whole of the exception. And when a label is rewritten, **lead with the current rule rather than appending a third section**: the point is that a reader stops having to reconstruct the verdict from a stack of deferrals. Spot-check that the load-bearing strings survived the rewrite.

**After reclaiming**, propose executing pending spawn intent:

> "Pending intent: `i302` (retry logic) and `i315` (scoring refactor), 2 clones available.
> Shall I run `/bip-conductor-spawn`?"

Keep the proposal at that altitude: which intent is pending, which slots are free, and any ordering constraint from the epic's Step 4b.
If the user asks "why this one?", point at the brief rather than summarizing it — the brief is the artifact, and a conductor paraphrase of it is strictly worse than the thing itself.

Then run `/bip-conductor-spawn` (do NOT improvise tmux/claude commands). Scout, place, spawn, and report after — a ready brief sitting next to an idle slot is the failure this altitude exists to prevent, and the proposal above is a report, not a gate.

Deciding *which other* open issues should be spawned next isn't this skill's call — that's `/bip-epic`.

**A busy fleet is good as long as every slot is on topic. The gate is topic, not count.** So spawn ready, in-scope, unblocked briefs freely while capacity exists — do not throttle on volume, and do not treat a full fleet as a thing to apologise for. The failure to avoid is a slot working the wrong programme, not a fleet that is fully occupied with the right one.

**This is a trigger, not a disposition. When a slot completes, re-assess and spawn in the same step — do not report the completion and wait.** The paragraph above has been in this skill through sessions where the conductor still sat on ready briefs until the user asked. A disposition buried in a long document does not fire; an event does. The event is: a slot finished, a PR merged, a brief appeared.

**Match worker count to the shape of the work, not to caution:**
- One issue whose result informs the others → spawn that one, alone, and wait for it.
- N independent ready issues → spawn N.

Withholding ready on-topic work is its own failure, and it looks like prudence from the inside.

If a live worker's scope needs correcting *before* its next natural stopping point: the epic decides whether it's durable, drafts the line, and the conductor delivers it — see "Correcting a live worker" above for the mechanics and where the durable record goes.

### Step 7: Start slot monitor

After the dashboard is built and any spawns are launched, offer to start the **persistent slot monitor** — `bip epic watch` — which observes every slot's `.epic-status.json` and writes phase-transition events to `.epic-notifications.log` (JSONL) in the conductor cwd.
The log is the canonical record; transitions survive watcher restarts and conductor compaction.
This replaces `/loop 10m /bip-conductor-poll` for the most time-sensitive signals (phase transitions), while `/bip-conductor-poll` remains available for full slot reconciliation sweeps.

Start the watcher in the background:

```bash
nohup bip epic watch >/dev/null 2>&1 &
```

The watcher runs forever, exits cleanly on SIGTERM, and emits one event per real phase transition (default filter: `needs-human`, `completed`, `awaiting-results`, `quality-gate`).

**Three ways this watcher goes silent without failing. None is a bug in the watcher; all are reasons it is not liveness detection:**
- **An off-spec `phase` value falls outside `--phases` and emits nothing.** (One issue-lead wrote `phase: "premature-deferral"` — a `stop_reason` in the phase field — and that transition appears nowhere in the log.) Broadening the filter helps against *known* strings only: `--phases exploring,coding,testing,awaiting-results,quality-gate,needs-human,completed` **plus any off-spec value you have actually seen**. The flag takes an explicit list with no "all", so a newly invented value still slips.
- **A slot that never transitions produces no events at all**, however wide the filter. Only status-file mtime against the clock catches this — the two-check staleness rule in the `.epic-status.json` spec below, and `/bip-conductor-poll`'s sweep, are the actual liveness check.
- **First sight is a baseline, not an event.** The watcher emits on a *change between two observations*, so whatever phase a slot is in when it first reads that slot is never reported — **a slot that reaches `needs-human` before the watcher starts, and stays there, is invisible for as long as it holds.** So: **(a)** restarting the watcher silently re-baselines the whole fleet — a restart is a monitoring gap, not a refresh; reconcile with `/bip-conductor-poll` immediately after one. **(b)** A clone added to `clone_names` after launch is never enumerated at all (see `/bip-conductor-spawn`'s four registration steps). **(c)** The log records transitions *observed*, never states *held*.

⛔ **BEFORE ANY OF THAT: WHEN SEVERAL INDEPENDENT SLOTS GO QUIET AT ONCE, ASK WHAT THEY SHARE — NOT WHAT EACH WAS DOING.** Simultaneity across independent units is evidence of a **shared cause**. Reading N units failing together as N independent failures is a specific, nameable error, and its damage is that **it points you at the wrong LAYER** — every step after it is competent and misdirected.

➡ **The check is one question, asked BEFORE any per-slot investigation: what do these units have in common?** A host. A token or rate budget. A filesystem or mount. A binary or a freshly landed commit. A network path. **Each is one command; a six-slot pane sweep is not.**

⚠ **Measured 2026-09-15, and the conductor got it wrong in the expensive direction.** Six live slots produced one phase transition and no merges for roughly three hours. The conductor swept every pane by hand, confirmed all six `busy`, found their status files 181-236 minutes stale, and **drafted a skill change attributing the staleness to a missing refresh trigger.** The actual cause was a **fleet-wide token exhaustion** — the sessions could not act at all. ⛔ **The per-slot measurements were all correct; the conclusion built on them was not, because the units were never independent.**

⭐ **The tell was in the data the whole time: six independent slots do not stall within the same window by coincidence.** The conductor had the fleet's entire observable state and never asked the common-factor question.

⚠ **And note what this does NOT invalidate, because the boundary matters when you re-read your own findings afterwards: anything with its own elapsed-time clock survives.** A live process's `etimes`, a daemon's RSS, a file on disk — **a throttled session can still leave a process running**, so those readings stand. What does not survive is any inference about *why a session did or did not act*.

**Never read push silence, or a quiet notifications log, as "still running."** Measured 2026-09-04: eight issues closed in one burst and two never surfaced, found only by reconciling every clone's branch against `gh`. **Push notifications failed for both in the same burst** — the two channels are not independent redundancy; the reconciliation sweep is what actually caught them.

**That rule has been in this file since 2026-09-04 and the miss happened again on 2026-09-13, because the rule names a prohibition and hands the reader no instrument.** Here is the structural cause, and the instrument that makes the prohibition actionable. **A slot that LANDS emits no terminal event, ever.** `/bip-pr-land` deletes `.epic-status.json`, so the `quality-gate -> completed` transition is unobservable *by construction*: the watcher's last word on any slot that lands is structurally "PR opened", forever, and nothing in the log distinguishes that from a slot still grinding. **This fires on the success path, so it is the modal outcome rather than an edge case** — every slot that finishes cleanly produces it. Measured 2026-09-13 on `matsengrp/phyz`: a slot's last logged transition was `testing -> quality-gate` at 11:21:48Z ("PR #2583 opened. Running quality gate loop."); the PR merged seven minutes later at 11:28:36Z, and the conductor reported that slot as running for **65 minutes** afterwards across three user-facing reports. The completion push did not backstop it — that slot sent none, and why is not established; three sibling slots sent theirs from identical prompt text, so a push is not a guarantee. This is `EVIDENCE-DISCIPLINE.md`'s *"a detector's negative result is not evidence until the detector is shown to fire"* with a different detector. **The remedy is a different instrument, not a wider filter**: a landing is observable as a *PR state change* even though it is invisible as a status-file transition, so run a second Monitor polling `gh pr list --state merged` alongside the notifications tail. Neither alone is sufficient, and the two fail independently.

**Checking whether the watcher is running:** use `ps -eo pid,args | grep -E '^\s*[0-9]+ bip epic watch'` — anchored on the start of the command line — **not** `pgrep -af 'bip epic watch'`. The spawn prompt in `/bip-conductor-spawn` contains the literal string `bip epic watch` (in its PUSH NOTIFICATION block), so `pgrep -af` matches every live worker and returns tens of KB of prompt text.

**That is one instance of a class, and stating it as one command plus one string is why avoiding both does not help.** A `bip spawn`-launched worker's **entire spawn prompt IS its argv**, so *any* argv-matching search — `pgrep -af`, `ps | grep`, `ps aux | grep` — matches every live worker on almost any keyword a conductor would plausibly type: a clone name, the repo name, `zig`, an issue number. Measured 2026-09-14 on a 28-`claude`-process box: a conductor avoided `pgrep -af` exactly as the paragraph above says, then ran `ps -eo pid,etimes,args | grep -E 'zig|phyz' | grep -i cedar` to ask whether a build was running in one clone, and got **64.5 KB of an unrelated slot's spawn prompt** — matched on the word `cedar` appearing in that prompt's *prose*, nowhere near a process.

**The reliable form does not read argv at all.** Enumerate by exact process name, then attribute by working directory:

```bash
# Which clone is each live Claude session in? (no prompt text, one line)
for pid in $(pgrep -x claude); do
  printf '%s\t%s\n' "$pid" "$(readlink /proc/$pid/cwd)"
done
```

⚠ **`pgrep -x` is NOT fleet-scoped — the `cwd` filter is what does the scoping, so filter on it rather than merely printing it.** Measured the same day: of **28** live `claude` processes, only **6** had a cwd under this project's `$CLONE_ROOT`; the rest were other projects' pools (`~/re/sfpcp/*`, `~/re/d2/*`) and unrelated interactive sessions. A conductor that enumerates with `pgrep -x` and assumes fleet-only will reason about another project's slot as if it were its own. The same applies to `pgrep -x zig` when asking whether a build is live *in a particular clone*: resolve `/proc/<pid>/cwd` and match it against the clone path.

⛔ **NEVER GREP ARGV FOR A STRING YOU ALSO PUT IN A BRIEF — a compliance audit whose false positives are generated by the very text that mandates compliance.** Measured 2026-09-15: auditing whether a `-j8` build cap had been adopted, `ps -eo args | grep -oE '\-j[0-9]+'` returned **24 × `-j28`**, reading as "the cap was ignored fleet-wide." **16 of those matches were the conductor's own spawn prompts**, which contain `-j28` as instruction text and are the workers' argv. Real count via `pgrep -x zig` + `/proc/<pid>/cwd`: **two clones, both at `-j8`.** The prompt text guarantees a false positive for any flag, path, or issue number it mentions — **and the caller is the one who put it there**, so this fires hardest on exactly the checks a conductor most wants to run.

⭐ **THREE MECHANISMS, ONE PROPERTY: ALL THREE FAIL TOWARD "NO ACTION NEEDED."** That is the thing to remember; the mechanisms below are how it arrives.

| | what goes wrong | the tell |
|---|---|---|
| **self-match** | the pattern appears in **your own** command line | `pgrep -f '<pat>'` returns the asking shell's own `$$`; `ps aux \| grep` additionally matches every `claude` session's 28-39 KB spawn prompt |
| **wrong universe** | right pattern, **wrong population** | `pgrep` without `-u $(id -u)` on a shared host matches **other users'** jobs |
| **empty denominator** | a clean-looking zero from a probe that **never ran** | nothing distinguishes it from a correct zero unless you print the denominator |

⛔ **Each returns the reassuring answer and none returns an error.** Before trusting a negative, ask which of the three it could be: exclude `$$`, pass `-u`, and print the denominator — name the processes you *did* consider, the row count, the confirmed variation in the input.

⚠ **The wrong-universe case measured 2026-09-15, and it is the one with no local tell at all:** a conductor watching "has the remote sweep stopped" ran `pgrep -f snakemake` over ssh with no `-u`, matched **an unrelated user's** dasm2 snakemake on the same shared host, and **could never have reported STOPPED.** It was minutes from firing a false "the owner has not acted, intervene." Correctly scoped, the answer was available in one command.

⭐ **Keep these separate from a wrong *prescription*.** These three are *"my probe lied to me"*; a bad prescription is *"my reasoning outran my evidence."* Different remedies — the first wants a better instrument, the second wants re-deriving from the mechanism.

⛔ **AND THE GENERAL FORM, WHICH THE RULE ABOVE IS TOO NARROW TO COVER: ANY PROCESS-MATCHING PATTERN THAT APPEARS IN YOUR OWN COMMAND LINE MATCHES YOU.** The brief-text rule above needs a brief and needs a fleet. This one needs neither — **it deadlocks on an idle box with a single session**, and `pgrep -f` is the variant that looks safest.

⚠ **Measured 2026-09-15: three slots, two mechanisms, 33 to 70 minutes each, all three found only because the user asked about one slot by name.**

| slot | stuck | the loop |
|---|---|---|
| pine | 70 min | `until ! ps aux \| grep -q "[m]ake check"; do sleep 5; done` |
| teak | 60 min | `until ! pgrep -f 'zig-cache-teak -j8 test-ml-cli test-aln test-core test-parsimony' >/dev/null 2>&1; do sleep 5; done` |
| cedar | 33 min | `while pgrep -f 'check_knob_citations' > /dev/null 2>&1; do sleep 5; done` |

**pine's is the fleet-dependent kind the rule above covers**: `[m]ake check` matched 14 processes, of which 5 were real and **9 were `claude` sessions matched on spawn-prompt prose alone — including pine's own session** (pid 2259010, argv 27,914 B). It waited on a string its own parent guaranteed.

**teak's and cedar's matched the waiting shell ITSELF.** The pattern is a literal substring of the `zsh -c`'s own cmdline, carried inside `eval '...'`. Verified directly: `pgrep -f '<pattern>'` returned each waiter's own pid in its own match set.

⚠ **Neither the `[m]ake` bracket trick nor `>/dev/null 2>&1` helps.** The bracket only stops `grep` matching its *own* argv; it says nothing about other processes' argv and nothing about the waiter's own cmdline.

⛔ **AND THE SIBLING THIS BLOCK USED TO OMIT: `pgrep` WITHOUT `-u $(id -u)` ON A SHARED HOST MATCHES OTHER USERS' PROCESSES.** Not self-matching — *cross-user* matching — and it fails in the same direction, toward "still running." The orca hosts and `/fh/fast` boxes are shared; `pgrep -x phyz` there answers *"is anyone's phyz running"*, not yours. **Scope every remote process check with `-u $(id -u)`, and attribute by `/proc/<pid>/cwd` as above.** Measured 2026-09-15: a conductor's own watch for a worker's sweep matched a different user's snakemake and was structurally unable to report the stop.

⛔ **AND IT IS NOT CONFINED TO LOOPS. A ONE-SHOT `pgrep -f` ASKING "IS X STILL RUNNING" RETURNS AN OFF-BY-ONE THAT INFLATES THE COUNT, AND THERE IS NO SYMPTOM AT ALL.** The deadlock above is the LOUD version of this bug; **the silent count is the one a human reads and believes.** ⚠ **`pgrep -c` makes it strictly worse, because it discards the pids that would have shown you the problem.**

⭐ **Measured 2026-09-15 under control, with a pattern that existed nowhere else on the box:** `pgrep -c -f zzuniqpattern9173` returned **`1`**, and `pgrep -f` on the same pattern returned exactly one pid — **the asking shell's own `$$`**. Zero real matches, count of one, exit status 0. The `$$`-excluding form returned `0`.

⚠ **The instance that justifies writing this down came from the REVIEWER OF THIS VERY RULE, about two minutes after signing off on it**, running `pgrep -c -f check_knob` against its own gate: it returned `4`, one of which was the grep. It was caught only because that operator had pre-emptively written "may include this grep" into the command's own echo. **A rule freshly read did not prevent it.** That is the argument for the next paragraph being the default form rather than a remedy.

➡ **WHAT TO DO INSTEAD, which is the most useful line here: poll on a PID captured at launch (`$!`), never on a pattern — a pid cannot appear in your own cmdline.** Better still, **don't poll**: a backgrounded `Bash` re-invokes the session on exit, so the wait is the harness's job. **Without a correct loop to copy, a reader who accepts this entire diagnosis writes a fourth variant.**

**Where you must identify a process you did not launch, this is the default form — exact name, attribute by cwd, and exclude self unconditionally:**

```bash
n=0
for p in $(pgrep -x zig); do
  [ "$p" = "$$" ] && continue                                  # NOT optional, NOT a remedy
  [ "$(readlink /proc/$p/cwd 2>/dev/null)" = "$PWD" ] && n=$((n+1))
done
echo "$n"
```

⚠ **`pgrep -x` cannot match your shell on its own** — the shell is not named `zig` — **so the `$$` guard looks redundant here and is kept anyway**, because the line that gets edited later is the pattern, and the moment anyone switches `-x` to `-f` the guard is the only thing standing between them and the off-by-one above.

⛔ **LEAD WITH THE CONSEQUENCE, BECAUSE IT IS WORSE THAN THE WASTED TIME: A DEADLOCKED SLOT IS SIMULTANEOUSLY UNRECLAIMABLE AND UNREACHABLE.** The session reports `shell`, never `idle`, so the `idle`-only reclaim gate never fires — **and a session blocked on a foreground shell makes no tool call, so it cannot drain a `SendMessage`, including the correction telling it to stop.** ⭐ **Every push-based mechanism in this design assumes the receiver drains its inbox; this is a state where it provably does not.** Same shape as the permission-modal case below. Only the conductor can clear it, from outside, and nothing reports it: pine held that state for 70 minutes with its work already landed and its clone clean.

**The sweep. Run it whenever a slot reads `shell` rather than `idle`, and on any full reconciliation pass:**

```bash
for pid in $(pgrep -x zsh; pgrep -x bash; pgrep -x sh); do
  cwd=$(readlink /proc/$pid/cwd 2>/dev/null); case "$cwd" in $CLONE_ROOT/*) ;; *) continue;; esac
  ppid=$(ps -o ppid= -p $pid | tr -d ' '); et=$(ps -o etimes= -p $pid | tr -d ' ')
  [ "$(ps -o comm= -p $ppid | tr -d ' ')" = "claude" ] || continue
  [ "$pid" = "$$" ] && continue          # exclude self: this sweep matches its own cmdline
  [ "$et" -gt 600 ] || continue          # SEE THE WARNING BELOW BEFORE LOWERING THIS
  echo "$et|$pid|${cwd##*/}"; tr '\0' '\n' < /proc/$pid/cmdline | tail -1 | grep -o "eval '.*' < /dev/null"
done | sort -rn
```

⛔ **THIS SWEEP IS ITSELF AN INSTANCE OF THE CLASS IT DETECTS, and what excludes it is incidental rather than designed.** It greps for `eval '.*' < /dev/null` from a shell whose own cmdline contains that string, and it satisfies the ppid-is-`claude` filter. **Two things keep it out of its own results, and neither is a safeguard**: the `600`-second threshold, and — on a layout where the conductor clone sits *outside* `$CLONE_ROOT` — the cwd filter. ⚠ **Verified 2026-09-15: the sweep shell's own cmdline contained the pattern twice, and only the cwd filter excluded it on that layout.** So **a conductor that runs this from inside a pooled clone loses one of the two exclusions, and a future reader who lowers the threshold to catch faster deadlocks loses the other.** ⭐ **So exclude `$$` in the sweep as written, not conditionally on someone lowering the threshold** — per the default form above, the guard belongs in the line before it is needed, not after.

⚠ **Clear it before killing.** Confirm no real process of that clone's own is being waited on — `pgrep -x make`/`-x zig` plus a `/proc/<pid>/cwd` match. Measured the same day: teak had genuinely live `zig` pids from a later, separate build, and balsa's 105-minute shell was a real `run_speed_diag.sh` run. **A blind age-threshold kill would have taken both.**

⭐ **The argv SIZE is a clean structural discriminator between a spawned worker and a hand-started session, and it reads no prompt content.** `wc -c < /proc/<pid>/cmdline`, same 28-process measurement:

| | argv bytes |
|---|---|
| interactive sessions (conductor, epic, ad-hoc) | **7 – 51 B** |
| `bip spawn`-launched workers | **15,029 – 36,078 B** |

Nothing fell between 51 B and 15 KB — a ~295x gap. **Test against the gap, not against the band's low end**: a threshold pinned near the top of the worker range would have misclassified the smallest worker observed (15 KB). Anything over ~1 KB is a spawned worker; anything under ~100 B is not. This answers *"is this pid a worker or a human's session?"* structurally rather than by name-matching, which matters because display names drift and three naming schemes are live at once.
To also receive events as Claude Code notifications when that pipeline is reliable, additionally start a Monitor with `command: tail -F .epic-notifications.log` and `persistent: true`.
The notifications log is the contract **for transitions a worker emits**, and Monitor is a latency optimization for those; **a landing is not one of them** — `/bip-pr-land` removes the file that would carry it — so for landings the `gh pr list --state merged` Monitor above is a correctness requirement, not an optimization.

When a transition arrives showing `needs-human` or `completed`, the conductor should react immediately: read the slot's status, check the lead guidance, refresh `$CLONE_ROOT/.conductor-session` (see "Completion pushes" in Conventions — this is one of the moments that keeps it fresh), then read `$CLONE_ROOT/.epic-session` and `SendMessage` that address the issue number and phase if the file is present — skip silently if it's absent or the send fails — and either propose the next action or flag it for the user.

> "Slot monitor started — phase transitions are streaming to `.epic-notifications.log`.
> Use `/bip-conductor-poll` for a full reconciliation sweep when needed."

On NFS-mounted clone roots where inotify does not fire on remote writes, pass `--poll` (defaults to a 2 s stat-loop) instead of fsnotify:

```bash
nohup bip epic watch --poll >/dev/null 2>&1 &
```

## .epic-status.json specification

```json
{
  "issue": 281,
  "title": "Short title",
  "phase": "exploring | coding | testing | awaiting-results | quality-gate | needs-human | completed",
  "summary": "Human-readable one-liner",
  "updated_at": "2026-03-03T14:30:00Z",
  "blockers": [],
  "remote_run": null,
  "quality": null,
  "scope": "One-line restatement of issue goal from lead",
  "stop_reason": "phase-complete | needs-instrumentation | needs-deeper-investigation | awaiting-results | run-production | pr-ready | quality-gate | mechanical-blocker | scope-drift | needs-human | completed",
  "lead_guidance": "What the worker should do next",
  "lead_notes": [],
  "completed_at": null,
  "awaiting": null
}
```

- Must be `.gitignored` (along with `.epic-worklog.md`, `.epic-decisions.md`, and `.epic-notifications.log` — see Conventions, "Decision relays" and ".epic-decisions.md: the durable fleet-decision log")
- **The general rule, not just this list**: any file this skill writes to the **conductor cwd** needs a `.gitignore` entry in the consuming repo, because that cwd is inside a clone's git tree.
  It has bitten twice, independently: `.epic-decisions.md`, and `.epic-notifications.log` (written by `bip epic watch`) — both dirtied a worktree until someone noticed.
  Files at `$CLONE_ROOT` instead — `.epic-session`, `.conductor-session`, `.spawn-prompts/` — do **not** need it, since that path is deliberately outside every clone's git tree (see Step 6 in `/bip-epic` on why `.spawn-prompts/` lives there).
  When adding a fleet-state file, ask which of the two paths it's written to before deciding whether it needs a gitignore line.
- **Staleness is two checks, and the second one is what catches a slot that stops reporting.**
  - *No tmux window, status file older than 30 minutes* → abandoned slot, a cleanup candidate (Step 6).
  - ***Window blocked on a permission modal*** → **the conductor is the only party that can see this, and no other check in this file finds it.** A worker whose pane holds a `Do you want to proceed?` prompt is frozen: it cannot act, cannot read its worklog, and **cannot receive a `SendMessage`** — measured 2026-09-13, a conductor correction sat visibly queued behind such a modal for 64 minutes. The watcher is blind by construction (no phase transition ever occurs) and the worker cannot ask for help, so there is no "it will tell you" path. **Sweep for it directly, on every poll:**

    ```bash
    for p in $(tmux list-panes -a -F '#{pane_id} #{pane_current_path}' | grep '/re/pz/' | awk '{print $1}'); do
      tmux capture-pane -p -t $p | grep -q 'Do you want to proceed?' && echo "MODAL: $p"
    done
    ```

    **Cancel with `Escape`; never send `Enter` and never select "Yes".** The cursor sits on "1. Yes", so a blind `Enter` approves a destructive command on another session's behalf — which is not the conductor's call to make even when the intent looks benign, and is the same hazard that picked the wrong option in three of four windows during the trust-dialog incident above. Cancelling costs the worker one re-issue. Then tell it what was blocked and that you declined rather than approved.

    ⚠ **The command may not be the worker's.** A *subagent's* modal surfaces in the parent's pane, so the parent will not recognise it — one instance took a round trip to establish, with the worker correctly insisting it had never issued the command. Say which it was.

    **Time trigger, for a slot reporting "waiting on a subagent":** six substantial research subagents measured on this fleet ran **3.5–7.2 minutes**. Treat **~20 minutes** (≈3× the longest observed) with no output as worth a sweep. ⚠ Those were all read-only research agents — one dispatched to *compute* can legitimately run far longer, so scale the threshold to the task. Err low regardless: a needless check costs one command, a missed modal cost 64 minutes and a blocked correction.
  - ***Window still open***, *status file older than ~45 minutes* → possibly **stalled**. That needs a human, not cleanup: surface it in Step 5's dashboard and never clean it up. Every other staleness rule here requires *no tmux window* and the watcher is transition-based, so without this check a slot that stalls (or merely stops reporting) with its window open and its phase unchanged is invisible to the whole design (once, six hours on `phase: exploring`).
  - ⛔ **A PANE'S OUTPUT IS CURRENT; ITS STATUSLINE IS A WIDGET ON ITS OWN SCHEDULE. Do not confuse them — the conductor reads both from the same `capture-pane`.** Scrollback is append-only, so what a worker printed is what it printed. The bottom two lines are not: the statusline is regenerated by a hook Claude Code runs on *its* schedule (turn boundaries, `/compact` finishing, and similar), so **a render can be arbitrarily old while looking current, because its token counts drift on their own and there is no visible tell.** Measured 2026-09-14: a conductor edited the statusline script, read four live panes to verify the change, and got **pre-edit renders with plausible fresh numbers** — then wrote the wrong conclusion into a committed comment as a measured fact, and needed two more rounds to undo it. **The general form, which is the part that transfers: a display that refreshes on its own schedule cannot be used to verify a change to the thing that feeds it.** To test a statusline change, feed a payload to the script on stdin. To read a worker's real context usage, do not trust the pane's percentage at all — and note that every other frame error of this kind has *some* tell available (a SHA, a denominator, a two-dot diff); this one has none.
  - **Check file mtimes, not the `updated_at` field** — a placeholder timestamp defeats the field but not the mtime. **Two cheap tells that a worker is typing timestamps rather than running `date -u`:** the field sits minutes off the file's own mtime, and it is round to the minute; or two fields in the same file are byte-identical, which two separate `date -u` calls cannot be. Measured across three workers in one session — one wrote local time with a `Z` suffix (7 h off), one wrote a value 4 minutes in the future, and one reused a single literal for `updated_at` and `awaiting.started_at`. That last field is the one that matters, since `started_at` plus `timeout_hours` is arithmetic. Check both `.epic-status.json` and `.epic-worklog.md`; the pair separates a genuine stall from a worker alive but not reporting. `/bip-conductor-poll`'s liveness sweep is the executable version — run it there, not here.
  - Corollary, same root cause (a self-reported field is not evidence): **the `phase` field is not evidence that work is done.** A worker wrote `completed` while its issue-lead was still running the terminal ceremony and its PR was still open. Check `gh pr view`, not the phase.
- `remote_run` optional — set when work dispatched to remote server
- `quality` optional — set during `quality-gate` phase:
  ```json
  {"pr_check": "pass|fail", "pr_review": "pass|fail", "iterations": 2}
  ```
Workers loop `/bip-pr-check` and `/bip-pr-review` until both pass clean.
The conductor can monitor progress via this field during polling.
- `scope` — set by the issue lead each iteration (one-line restatement of the issue goal)
- `stop_reason` — categorized reason from the lead's decision framework
- `lead_guidance` — actionable instruction for the worker's next iteration
- `lead_notes` — append-only log of lead evaluations (max 8 before escalation)
- `completed_at` — ISO 8601 timestamp set by the lead after it finishes the terminal `completed` ceremony (files any legitimate follow-ups, posts the final PR comment).
  Its presence is the idempotency signal: subsequent lead invocations at `completed` skip the ceremony.
- `awaiting` — set during `awaiting-results` phase:
  ```json
  {
    "description": "What we're waiting for",
    "check_cmd": "command that exits 0 when done",
    "check_files": ["paths whose existence means done"],
    "started_at": "ISO 8601",
    "timeout_hours": 12
  }
  ```

### Phase migration

Legacy phases from older `.epic-status.json` files:
- `blocked` → treat as `needs-human`
- `pr-review` → treat as `quality-gate`

## Error handling

- **Not in tmux**: Warn — tmux required for spawning
- **gh not authenticated**: Suggest `gh auth login`

## Layout config (issue #149)

`.epic-config.json`'s `clone_root` / `clone_names` / `local_worktrees` keep working untouched.
The newer way to configure worktree mode (for non-EPIC `bip spawn` use) is the `layout:` block in `~/.config/bip/config.yml` — see `docs/guides/layout.md`.
EPIC orchestration still reads `.epic-config.json` for now.
