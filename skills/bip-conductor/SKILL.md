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
Workers report back on their own; re-run this skill for a full rescan.
To spawn work, use `/bip-conductor-spawn`.

## Terms and intake

- **An EPIC** is a GitHub tracking issue for a programme of work; a repo can have many open.
- **The epic agent** is a `/bip-epic` session scoped to exactly one EPIC. Currently one epic agent per conductor (what N would need is sketched in `matsen/bipartite#211`; don't build toward it here).
- **The slot protocol** — `.epic-status.json`, `.epic-worklog.md`, the issue-lead loop, `bip fleet watch` — is per-slot work tracking and applies to **every** spawn, whatever its source. The `.epic-` prefix is a legacy name, not "epic machinery".

Work reaches a slot two ways, and the source decides who owns scope:
- **Epic-originated**: the epic wrote a brief to `$CLONE_ROOT/.spawn-prompts/`. The epic has already judged the work worth a slot; the conductor checks only mechanics — gates, staleness, host and slot availability, file collisions.
- **User-originated**: the user asks directly, often from `/bip-ms`. There may be no EPIC at all. **Never reject, defer, or route to the epic a user-originated spawn for lacking an EPIC.**

## Role

The conductor session owns the clone pool, tmux, and host state:
- Scans clones/worktrees, tmux windows, and (via `/bip-scout`) remote hosts
- Runs the pre-launch checks before every spawn (is the named blocker still open? is the clone current? any file collisions?)
- Executes spawns — composing the fleet-fact annotation on top of intent the epic already drafted, or from scratch for conductor-initiated work
- Cleans up finished slots and sweeps unfiled draft issues, never deleting authored-but-unfiled work
- Resolves resource conflicts from measured fleet state and reports the call; escalates only when either resolution risks an actual problem (see "Arbitration")
- Never does topic reasoning, and does not write code or create branches for numbered issues (light triage — reading files, checking CI output, running `gh` commands — is fine)
- Holds no topic boundary of its own; it owns *mechanical* sequencing — file collisions between slots

### Who rules on what

**Epic owns science; conductor owns correctness and hygiene** (user, 2026-09-11, `matsengrp/phyz`). This records one EPIC's arrangement, not a general law.

- The conductor **rules** on hygiene decisions (a correctness PR's convention question, a doc call, a provenance rule) from measured state and informs the epic. It does not forward them to the user.
- Tell the worker a conductor ruling is the **terminus**, state the authority basis, and record it in the worker's `.epic-worklog.md` — never in `lead_guidance`.
- The split is by decision type, not by issue: the conductor can run a science issue's mechanical pipeline while the epic holds the gate on any published number it touches.
- The conductor still refuses to attribute a defect to a mechanism, rule whether a result holds, or interpret an arm's output — even when it found the defect.

**Skill changes** (user, 2026-09-11): *"I want you and the EPIC agent to agree on any changes before you actually push them to bipartite."* So: conductor drafts, epic reviews, both agree, then push. A skill edit encoding a scientific judgement is the epic's call.

### Mark what a message is

A worker cannot tell from tone whether a line is an instruction, a fleet fact, or background, and a worker that reads background as a gate waits for an approval nobody will give.
- Say when something is background rather than a gate.
- Announce that something is being withheld, with its release condition, and say the withholding carries no directional information.
- Timestamp fleet measurements — "(measured 23:52Z)" — and mark claims you did not re-run as "(recalled)".

### The fleet/topic line

The conductor's user-facing report contains nothing it did not derive from fleet state — `git`, `tmux`, `gh`, `ps`, `/bip-scout`, and the status/intent files.
Slots, windows, hosts, branches, PR *state* (open, draft, mergeable), what is running, what is dead, what is free, and who is blocked on whom **by name**.
Not why an issue matters, not whether a finding is right, not the substance of a blocker.
The user hears that once, from `/bip-epic`, in its window.

**This is a rule about narration, not about intake.**
Topic findings reach the conductor as *scheduling constraints* — "#2080 and #2088 share three files" is the conductor's to act on, because its whole consequence is an ordering decision.
Consume it as a constraint, log it, and do not re-verify, re-narrate, or re-litigate it. Re-derive a peer's number before *acting* on it, silently, and report only a delta.

**Process findings** (a check that answered an adjacent question, a convention two workers applied differently) do not go in the user-facing report:
- Measurement-discipline findings go to `EVIDENCE-DISCIPLINE.md` and the EPIC's record, where workers and reviewers read them.
- Fleet-mechanics findings go in this skill.
- Log it, then either land the edit or drop it deliberately. There is no "logged, pending" state.
- Surface one to the user only when it needs a decision only they can make, and lead with that decision.

## Arbitration

When two things want the same clone, cache, or host, or when the epic's spawn intent conflicts with what the conductor observes live, **resolve it from measured state and report the call with its reasoning.**

The test: **does the answer change what anyone does?** Escalate only when resolving it either way risks an actual problem — data loss, a clobbered remote checkout, two slots owning one deliverable, or whether work is worth doing. Merge and rebase friction is not that, and a question nothing operational reads is not either, even when a peer has framed it as the user's to rule.

### Paging the user

When a question clears that bar and is the user's to answer, ring once: `bip page --from <your ListAgents name> --link <issue/PR/comment URL> "<one-line ask>"`.
The push is a doorbell pointing at this window, not the message.
Ring even when the user typed here a moment ago: a recent turn is not presence, and an unrung ask sits unseen in the scroll.

- **The owner rings**: the session that will act on the answer. Any other session that spots the question sends it to the owner instead.
- **One ring per wait.** Hold every item for the user under it; if a question changes while you wait, update what you will show, and do not ring again.
- **If the question resolves before the user answers**, send `bip page --cancel --from <name> "<why>"`, so the ring doesn't bring them to a settled question.
- **The user's next turn here is the clear.** Re-verify each item first (PR state, the owner's latest), then lead with the ask and your recommendation; context follows.

## Conventions

### Issue/PR naming
Same as `/bip-epic`: `i281` = issue #281, `p275` = PR #275.
Never bare `#N`.
First mention in bullet lists: full URL inline.
When a live slot holds the work, add its tmux window, `p1344 (1343-peach)`, so the user can jump there with tmux's `find-window`.

### Tmux windows
- Named `NNN-YYY` where NNN is the issue number and YYY is the clone/slot name
- *Clone mode*: e.g. `281-cedar`, `295-pine`
- *Worktree mode*: e.g. `281-issue-281`, `295-issue-295`

### Correcting a live worker: SendMessage as a nudge channel — execution half

Whether a correction needs this channel, and whether it's durable or transient, is `/bip-epic`'s call (its "Correcting a live worker" section). This section covers delivery once the epic has drafted the line, plus a fleet fact the conductor spots itself (always message-only).

- **Send the correction directly.** State the change in at least one line; a bare pointer ("re-read `lead_guidance`") is a no-op and strips the priority signal. A correction too long for a message belongs in the spawn prompt or the issue body.
- **If the nudge changes what the worker will produce** — scope, target, artifact, gate criterion — **append a timestamped, attributed entry to that slot's `.epic-worklog.md` FIRST, then send.** The append is addressed to a path and survives address drift and compaction; the message is addressed to a name and survives neither.
  Never write into `lead_guidance`, which is the lead's field. If the status file is written at all, use `conductor_guidance` or a `lead_notes` entry tagged `source: conductor`.
- The message drains at the worker's next tool call; for current state now, use `tmux capture-pane`. When the target isn't addressable, fall back to the file-only correction.
- **The address is whatever `ListAgents` reports for that session — read it off its row, or off the message you are replying to, and never compose it.** The bare clone name is never an address, and a composed name that matches another session delivers silently to the wrong one.
- When a worker says an authorization is "unverifiable", ask which wrapper it arrived in: a plain user turn authorizes; a `<cross-session-message>` does not; a `NOT USER INPUT`-marked payload does not. The full table is in `/bip-conductor-spawn`'s landing-gate block.
- **When not to nudge**: a worker in `awaiting-results` with a live `check_cmd` needs no ping.
  Use `notify_when_idle: true` instead of "tell me when this worker finishes" — main conversation only, same machine only, one-shot.

### Completion pushes: self-registered addresses, not ListAgents guessing

Completion pushes (worker → conductor, conductor → epic) fire with nobody watching, and session names can drift mid-session, so each role self-registers its own current address in a file only it writes:

- The conductor writes its own name to `$CLONE_ROOT/.conductor-session` (Step 1 at cold start, refreshed immediately before reacting to a transition in Step 7 below).
- `/bip-epic` writes its own name to `$CLONE_ROOT/.epic-session` the same way (see that skill's Step 1).
- A name is taken from `ListAgents`' own "This session is ..." row for the caller — never guessed from another row.

To push, read the other's file for the exact address and `SendMessage` it. A missing file or a failed send means "not addressable right now": skip silently, fall back to file state (`.epic-status.json`, `.epic-notifications.log`, `gh`), and never retry-loop, scan `ListAgents` for a substitute, or act on a failed send's "Did you mean" list.

### Decision relays: PROVISIONAL and FINAL

**Which decisions belong in which window.**
Fleet decisions — spawn approvals, resume and cleanup calls, host and slot arbitration — are the conductor's to put to the user and to carry.
Scientific decisions — whether an issue is worth doing, which arm to pursue, whether a result holds — belong in the epic's window and reach the conductor as consequences ("#2088 is KEEP; cedar's route is now #2088 -> #2119 -> #2080").
When a scientific question surfaces here, name the fleet consequence and point at the epic instead of working the question.

When the conductor is the session talking to the user and a decision results, push it to `/bip-epic`: read `$CLONE_ROOT/.epic-session` and `SendMessage` it the decision, **prefixed `PROVISIONAL` or `FINAL`. Nothing goes unmarked** — including a passing line inside a message about something else ("the user says they're ready for the new batch"). If it is not yet a decision, say so and mark it `PROVISIONAL`.

**A `FINAL` relay carries the decision at the granularity the receiver acts on.** "Ready to spawn the batch" is not actionable against three issues, two with separate holds; itemize before relaying.

- **`PROVISIONAL`**: still being discussed, or a first read before the user confirmed it. `/bip-epic` may note it is pending but must not write it into an EPIC body.
- **`FINAL`**: the user has confirmed the decision's shape. This, and only this, authorizes `/bip-epic` to write the decision into an EPIC body.
- Append every `FINAL` relay (and the `PROVISIONAL` it followed from) to `.epic-decisions.md` in the conductor cwd, timestamped and attributed.

This is one-directional: epic→conductor relays (spawn intent, worker corrections) are unaffected.

### Forwarding worker findings

Forward a worker's semantic finding **verbatim and attributed** to the worker (issue/slot) that produced it.
If the conductor has its own reading, add it as a separate, clearly marked line — never blended into the forwarded text.
Append forwarded findings to `.epic-decisions.md` alongside decision relays.

This applies to *worker* findings only. A finding from the epic has its own voice and channel: log it as a one-line pointer plus its fleet consequence and forward nothing onward.

### Resolving a citation before acting on it

Resolve a cited file, line range or symbol against the repo before acting on it (scheduling around it, cleaning up a path, spawning against it), and never reconstruct a missing path component from context — ask the sender which copy they meant. Logging or relaying a finding is not acting: forward it attributed and unresolved.

**A "do not re-run X, it is settled" in a brief must quote the archive line that settles it and name the file** — a worker obeys a prohibition without testing it. When the epic strikes a RULED OUT entry, `grep -ril` every in-flight brief and `.spawn-prompts/` (including `consumed/`) for citations to it.

### Message economy: the log is the artifact, the message is the nudge

Lead with the decision, the ask, or the correction; give the reasoning the receiver cannot reconstruct; cite `.epic-decisions.md`, the issue, the PR, or the worklog entry for the rest. Succinct, not terse. This applies to user-facing reports too.
Economy governs how much you say, never whether you report a defect: a bad check is one line, not silence.
Two cases it never licenses: a forwarded worker finding still goes verbatim, and a live-worker correction still states the change in at least one line.
No ack to a relay's sender when the worker's own reply will reach them.

### `.epic-decisions.md`: the durable fleet-decision log

Every `FINAL` relay, every forwarded worker finding, and every negative-list entry (Step 5) appends here — in the conductor cwd, gitignored the same way as `.epic-status.json`.
Timestamped markdown, append-only, never edit a previous entry.

```
## 2026-08-28T09:00:00Z — NEGATIVE (conductor)
<action not taken, and why — e.g. "i2080 not proposed: clone stood down 08-27 for <reason>, prerequisites merging today doesn't reopen it">

## 2026-08-28T14:30:00Z — RELAY:FINAL (conductor→epic)
<the decision, restated in full, not a paraphrase of sentiment>

## 2026-08-28T14:32:00Z — FINDING (worker i2098, forwarded by conductor)
<the worker's finding, verbatim>
Conductor's own reading, if any, marked separately: <...>
```

**Not `.epic-worklog.md`.** That file is per-slot and is removed on slot cleanup.

## Workflow

### Step 1: Load config, or set it up

```bash
cat .epic-config.json
```

**If the file does not exist**, follow `setup.md` in this skill's directory, then proceed.

All subsequent steps use values from this config — never hardcode paths or clone names.

**Self-register for completion pushes**: resolve `CLONE_ROOT` and write this session's own `ListAgents` name (the "This session is ..." row) as the sole line of `$CLONE_ROOT/.conductor-session`.

**Read every tracked subdirectory `CLAUDE.md` or `AGENTS.md` in the project repo** — only the root one is auto-loaded, and the conductor reads artifacts by path from the root. Enumerate with `git ls-files | /usr/bin/grep -E '(CLAUDE|AGENTS)\.md$'`, not `find`.

### Step 2: Pull main

```bash
git pull --ff-only origin main
```

If this fails, report the problem and continue with stale state.

### Step 3: One-time unfiled-draft sweep

Cold start only.

`/bip-issue-file` moves a draft to `_ignore/` the moment it successfully files it, so anything still sitting loose as `ISSUE-*.md` directly in a clone root **is** unfiled (an `update` reuses the file in place, so a draft mid-revision on an existing issue is expected here too).

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
find "$CLONE_ROOT" -maxdepth 2 -name 'ISSUE-*.md' -not -path '*/_ignore/*'
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-conductor`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory.

**Never auto-delete or auto-file these.** Flag what you found (path, clone, first line) so the user can decide.

The conductor clone sits outside `clone_root`, so its own drafts (and the epic's) are not swept, deliberately: the epic owns its draft list and tells the conductor, and the conductor reports what it is told. Do not widen the `find`.

### Step 4: Scan clones and tmux

No subagent: this is a handful of shell calls, run from the conductor clone.

```bash
bip fleet currency     # per slot: branch, dirty, behind, head
bip fleet collisions   # live branches, missing or stale status files
tmux list-windows -a -F '#W'
find "$CLONE_ROOT/.spawn-prompts" -maxdepth 1 \( -name '*.md' -o -name 'spawn-*.txt' \)
find "$CLONE_ROOT" -mindepth 2 -maxdepth 2 -name .epic-status.json \
  -exec jq -c --arg f {} '{f: $f, issue, phase, summary, scope, stop_reason, lead_guidance}' {} \;
```

In worktree mode (`local_worktrees: true`), the slots are `find "$CLONE_ROOT" -maxdepth 1 -name 'issue-*' -type d`; `bip fleet currency` covers clone mode only.

Classify each slot:
- `occupied`: has a tmux window, whatever the agent's status (the user may be doing follow-up work).
- `held`: named in `$CLONE_ROOT/.holds/`. Don't spawn into it.
- `stale`: no tmux window, but has `.epic-status.json` or is on a non-main branch. Clean up only through `reclaim_slot` (Step 6), never by deleting the status file by hand.
- `available`: (clone mode) no tmux window, on `main`, clean, and current with `origin/main`.

Also note phase migrations (`blocked`/`pr-review`), missing status files, and contradictions.

**Hold a slot** when something outside it still depends on its contents: another live slot reads files inside the clone, or remote jobs run from it. Write the reason to `$CLONE_ROOT/.holds/<slot>` (`mkdir -p "$CLONE_ROOT/.holds" && echo "<reason>" > "$CLONE_ROOT/.holds/<slot>"`), and delete that file when the dependency ends. It lives outside the clone, so reclaim leaves it and a clean-tree check never sees it. While it exists, `bip spawn` refuses the slot (`--ignore-hold` overrides; `--force` does not), `bip fleet currency` reports it `HELD`, and spawn selection skips it.

**Clean is not current.** Run `bip fleet currency` from the conductor clone before every spawn, read its `scope:` line before the table, and fast-forward an idle slot that is behind.

A behind-count does not tell you whether a **finished result** is stale. For that, take the build commit recorded in the artifact and run the tree form: `git diff <build-commit> <tip> -- <source paths>`.

If work will run on remote hosts, also check `/bip-scout` for host occupancy (live builds, warm caches, other sessions' remote jobs) — the annotation step in `/bip-conductor-spawn` depends on it.

### Step 5: Build status display

The dashboard is **slot-centric** — the epic's dashboard is issue-centric; this one is about what's occupying the fleet.

| Slot | Classification | Issue | Phase | Notes |
|------|-----------------|-------|-------|-------|
| cedar | occupied (tmux `281-cedar`) | i281 | coding | — |
| pine | occupied (tmux `295-pine`) | i295 | awaiting-results | ~Tue |
| fir | stale (4d) | i589 | needs-human | check if experiment finished |
| hazel | available | — | — | — |

**Pending spawn intent**: list any `.spawn-prompts/` files found in Step 4 (either naming pattern), with the available slots that could run them.

**EPIC distribution**: one line counting live slots by the `EPIC:` header of the brief that spawned them, with user-originated slots as their own row. This is counting, not judging; it is what makes drift across EPICs visible.

```
EPIC #369: 5 slots    user/bip-ms: 2 slots    (no header): 1 slot
```

**Negative list**: surface decisions already taken *against* an action, with reasons — sequencing already applied, an issue already stood down for a reason that would otherwise look resolved.
This is the one category of fleet fact nobody can re-derive from `git`/`tmux`/`gh`.
Read only the prior `NEGATIVE` entries (`grep -n -A3 -- '— NEGATIVE' .epic-decisions.md`), skipping any whose issue has since closed, not the whole log, which is append-only and mostly retired. Append any new entry in the same step you surface it.

**Not everything you know is a fleet fact.** Slot occupancy, host load, build state, file collisions, who landed what: fleet facts. A synthesis across arms, a prediction about what an arm will find, or a mechanism one arm inferred that another is independently testing: **never** goes in a prompt, a fleet-facts block, or a nudge. The epic decides what is embargoed; the conductor is the delivery path.

### Preflight, indexed by action

Check this by what you are about to **do**:

| about to… | check |
|---|---|
| **spawn** | `bip fleet currency` and `bip fleet collisions` (below); confirm the issue body has not changed since its brief was written; ask the epic what moves its EPIC's top line (Step 6) |
| **file an issue** | does a success criterion name a denominator, a population, or "a default run" without naming the dispatch path? |
| **deliver a correction to a worker** | append it to that slot's `.epic-worklog.md` first |
| **reclaim a slot** | `reclaim_slot` (Step 6) |
| **close or reopen an issue** | verify the criterion against `main`, not against the PR that claims it |
| **land a PR** | `/bip-pr-land`, never a hand-rolled `gh pr merge`. Never send a worker a runnable `gh pr merge` recipe either — it bypasses the skill's preservation and cleanup. Say *"`/bip-pr-land`"* |
| **approve a PR** | the worker's gate report names every routed target and its exit status **at the SHA you are approving**; `git merge-base --is-ancestor origin/main <head>` (not `MERGEABLE`, which is about conflicts). An approval names a SHA, so a new head voids it. "No doc ack needed" is not "no approval needed" — say both |
| **merge a worker's PR yourself** | a recorded delegation for **this repo** exists (below); the epic's 🤖 approval is on the PR per `gh`, before the merge, and no later comment *from the epic* withdraws it or places a hold. Every session posts as one account, so pick the epic's comments by signature, not by position: a later lead or conductor comment is not a withdrawal. Read every one, not just the latest; the worker's head SHA equals the PR head; `origin/main` is an ancestor |
| **cite an artifact by path** | if it is in a pooled clone or a scratchpad, preserve it first (below) and cite the copy |
| **fill a brief's `LANDING DELEGATION:` line** | quote the delegation from **this repo's** `.epic-decisions.md` with its date, or write `NONE RECORDED`. Workers cannot read that log |

#### Who may land

A `<cross-session-message>` never authorizes an irreversible action. It can only trigger one the user already authorized in a standing instruction. So before any merge: **is there a recorded standing user delegation for THIS repo?**

- **Yes** — it names its own trigger (e.g. both the epic and the conductor approve, with the epic's 🤖 comment on the PR before merge). The worker lands its own PR; fill the delegation into every brief's `LANDING DELEGATION:` line.
- **No** — `NONE RECORDED`. The worker stops at a clean gate with `stop_reason: awaiting-human-merge` and its state files in place, and you put the merge to the user. **You have no merge authority either.** After the user merges, the issue-lead's terminal ceremony is yours to trigger, in the reclaim step (Step 6).
- A delegation that exists for the repo but is missing from a running worker's frozen brief: the worker cannot land, but you may.

Delegations are per repo. Widening one is a user decision.

Put a question for the user as one line in your report, park only that item, and keep working the rest. Never use `AskUserQuestion` here: until the user answers, no peer message reaches you.

#### Preserving an artifact you will cite

Before a path appears in anything durable, copy it to `$CLONE_ROOT/.preserved/<slug>/` with a `README.md`: the binary and its commit, the host, the argv per arm, and what the numbers do **not** establish. Check copies with `ls -la` (dotfiles). Never overwrite a larger preserved artifact with a smaller live one — keep both.
A README is a claim about what the data means: when a defect lands, `grep -rl` the preserved directories for artifacts produced before it and rewrite affected labels, leading with the current rule.

### Run `bip fleet collisions` before every spawn

```bash
bip fleet collisions   # from the conductor clone; exit 0 = clear, 1 = found, 2 = could not check
```

Both `bip fleet` checks take the pool from `.epic-config.json` in the cwd, and exit 2 on a worker's frozen copy of it or when `~/.claude/skills/lib` is missing. Read the `scope:` first line. Don't pipe it and read `${PIPESTATUS[0]}`: that is empty in zsh, and in both shells the next command overwrites it.

**Exit 2 means could not check, never clear.** A freshly spawned worker's unpushed branch shows as `UNCHECKABLE` in the against-landed section. Run it on the conductor's machine: clone branches and uncommitted work are local.

A collision naming N clones on one file means N-choose-2 pairs to trial-merge; derive the pairs from the report. When you hand a conflict to whoever resolves it, say what differs between the sides, not only that they conflict.

### Editing fleet tooling

`~/.claude/skills/*` are symlinks into `~/re/bipartite/skills/`, and `~/go/bin/bip` is a symlink to `~/re/bipartite/bip`. **Any edit, `pull`, `checkout`, `rebase`, or `stash` in that worktree changes every live session's instructions immediately, with no event.** The root-level doctrine files are loaded by path from the same tree.
- Do fleet-tool work in a separate clone of `bipartite`, and treat operations in the primary one as fleet-wide.
- Before committing in a clone another session can reach, run `git diff` (whose hunks), not just `git diff --stat` (which files). Don't leave edits staged in a clone you don't own.
- Run any long-lived shell script from an immutable copy (`cp watch.sh /tmp/watch.$$.sh && bash /tmp/watch.$$.sh`): bash reads scripts incrementally, so an in-place edit or a `git checkout` changes what a running script executes.

### Scheduled load no slot owns

Before spawning, and before telling a slot to run a full suite:

```bash
systemctl --user list-timers --all
uptime
```

Use `list-timers` (it shows LAST and NEXT), not `is-active` on the service. A timer with `RandomizedDelaySec` re-randomizes, so quote its window, not the NEXT timestamp. Tell affected slots that a build killed in that window is contention, not a defect in their branch.

### Step 6: Propose next action

First, do housekeeping automatically (no need to ask).

**Reclaim** a slot from outside its clone, passing what `ListAgents` reports for its session right before the call (`none` if it has none). Only `idle` and `none` proceed; `waiting`, `shell` and `busy` are not finished:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
reclaim_slot "$CLONE_ROOT/<slot>" <owner/repo> <PR number> <ListAgents state>
```

It holds unless the terminal ceremony has run, the PR closes at least one issue and all are closed, the tree is clean, the local branch has no commit the merged head lacks, and the composer is empty. It then preserves the worklog into the clone root's `.preserved/`, kills the worker's pane (never the window, which another session's pane can share — the same goes for a reclaim by hand), waits until no process has its cwd in the clone, checks out the base, and deletes the three state files and the merged branch. Its one output line:

- `RECLAIMED` → spawn pending intent (below).
- `HOLD` → nothing changed. Fix the cause, or leave the slot.
- `NOT FREE` (exit 2) → the window is gone but the clone is not reset. If processes are still in it, re-run later with `none`: a detached build or IQ-TREE run outlasts the 30 s poll. If the checkout or pull failed, fix that first. The check cannot see a process writing in from outside (`git -C`, `rsync`).
- `CEREMONY OWED #<pr> <slot-dir>` → spawn an `issue-lead` subagent: *"Post-merge terminal ceremony for <owner/repo>#<issue>, PR #<N>, which `gh` reports MERGED. The slot's clone is `<slot-dir>`. Read its `.epic-status.json` and `.epic-worklog.md` there, run git as `git -C <that path>`, and pass `-R <owner/repo>` to `gh`. Follow your full evaluation protocol; Step 8 applies."* Then re-run `reclaim_slot`.
- `ceremony UNRUN` → tell the user; the worklog is gone.
- `CEREMONY UNKNOWN` → leave the slot alone.

**A loop is live** when `.claude/ralph-loop.local.md` exists **and** its `session_id` belongs to a running session.

*Worktree mode* has no helper: preserve with `preserve_epic_state`, then `git worktree remove --force $CLONE_ROOT/issue-N && git branch -d <branch>`.

If a worklog seems lost, look in `<clone_root>/<clone>/.preserved/` too. A deleted worklog is usually recoverable from the slot's transcript, `~/.claude/projects/$(echo "$CLONE" | sed 's|/|-|g')/<session-id>.jsonl`: replay the seed `Write`'s `input.content`, then each `Edit`'s `old_string`→`new_string`, then any Bash heredoc appends, in timestamp order. Stop if an `old_string` doesn't match. Mark the result as a reconstruction in its README.

**After reclaiming**, spawn pending intent:

> "Pending intent: `i302` (retry logic) and `i315` (scoring refactor), 2 clones available. Spawning via `/bip-conductor-spawn`."

Keep it at that altitude: which intent is pending, which slots are free, any ordering constraint from the epic's Step 4b. If the user asks "why this one?", point at the brief.

Run `/bip-conductor-spawn` (do NOT improvise tmux/claude commands). Scout, place, spawn, and report after — the proposal is a report, not a gate.
Deciding *which other* open issues should be spawned isn't this skill's call — that's `/bip-epic`.

- **Spawn ready, in-scope, unblocked briefs freely while capacity exists.** The gate is topic and objective, not count.
- **When a slot completes, re-assess and spawn in the same step** — a slot finished, a PR merged, or a brief appeared is the trigger.
- Match worker count to the shape of the work: one issue whose result informs the others → spawn it alone; N independent ready issues → spawn N.
- **Before a spawn, and on every event that frees a slot, ask the epic: "what is the next thing that moves this EPIC's top line, and is this it?"** A brief's readiness was established when it was written. The epic answers; a stated hold is a complete answer.

If a live worker's scope needs correcting before its next stopping point: the epic decides whether it's durable and drafts the line, and the conductor delivers it (see "Correcting a live worker").

### Step 7: Start slot monitor

After the dashboard is built and any spawns are launched, start the **persistent slot monitor** — `bip fleet watch` — which observes every slot's `.epic-status.json` and writes phase-transition events to `.epic-notifications.log` (JSONL) in the conductor cwd.
The log survives watcher restarts and conductor compaction.

Keep exactly one running per conductor; `fleet_watchers` lists this conductor's. Add `--poll` (2 s stat loop) only when the clone root is on NFS or sshfs, where inotify misses remote writes:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
case "$(fleet_watchers | wc -l)" in
  0) case "$(stat -f -c %T "$CLONE_ROOT")" in nfs*|fuse*) POLL=--poll;; *) POLL=;; esac
     setsid nohup bip fleet watch $POLL </dev/null >/dev/null 2>&1 & ;;   # own session, so the Bash call's teardown can't reap it
  1) echo "watcher already running" ;;
  *) echo "DUPLICATE watchers: $(fleet_watchers | tr '\n' ' ')" ;;
esac
```
The watcher emits one event per phase transition (default filter: `needs-human`, `completed`, `awaiting-results`, `quality-gate`). A transition into a phase that is not one of the seven legal values always emits, so `--phases` only ever lists legal phases.

It is not liveness detection. It is silent when:
- a slot never transitions — only the checks in "Slot staleness" below catch that. The one exception it reports is a `STALLED` event: `awaiting-results`, no ralph loop, its pane neither mid-turn nor showing a shell or monitor still running, status and worklog quiet 45 minutes, so nothing will wake it when its jobs end;
- a slot was already in its phase when the watcher first read it. Restarting the watcher re-baselines the whole fleet, so re-run `/bip-conductor` after a restart; a clone added to `clone_names` after launch is never enumerated;
- **a slot lands**: `/bip-pr-land` deletes the status file, so `completed` is never observed. Run a second Monitor polling `gh pr list --state merged --search 'sort:updated-desc'` (the default order is by creation, so an old PR's merge falls off the list); for landings it is a correctness requirement.

To also receive events as notifications, start a Monitor with `command: tail -F .epic-notifications.log`. Monitors expire: set `timeout_ms: 1800000` (the cap) on both, and re-arm each on expiry.

When a `needs-human` or `completed` transition arrives, react immediately: read the slot's status and lead guidance, refresh `$CLONE_ROOT/.conductor-session`, then read `$CLONE_ROOT/.epic-session` and `SendMessage` that address the issue number and phase (skip silently if absent or the send fails), and propose the next action or flag it for the user.

**Process questions** (what runs in the pool, a slot reading `shell`): never match argv — a worker's whole prompt is its argv — and follow `process-checks.md` in this skill's directory.

**When several slots go quiet at once, ask what they share before investigating any one** — a host, a token or rate budget, a mount, a freshly landed commit. Never read push silence or a quiet log as "still running".

## Slot staleness

The status-file schema is `status-spec.md` in this skill's directory.

- *No tmux window, status file older than 30 minutes* → abandoned slot, a cleanup candidate (Step 6).
- *Window blocked on a permission modal* → frozen, cannot receive a `SendMessage`, and invisible to the watcher. Sweep on every reconciliation:

  ```bash
  for p in $(tmux list-panes -a -F '#{pane_id} #{pane_current_path}' | /usr/bin/grep -F "$CLONE_ROOT/" | awk '{print $1}'); do
    tmux capture-pane -p -t $p | /usr/bin/grep -q 'Do you want to proceed?' && echo "MODAL: $p"
  done
  ```

  Cancel with `Escape`; never send `Enter` or select "Yes" (the cursor sits on "1. Yes"). Then tell the worker what was blocked and that you declined. A subagent's modal surfaces in its parent's pane. For a slot "waiting on a subagent", ~20 minutes with no output is worth a sweep.
- *Window still open, status file older than ~45 minutes* → possibly **stalled**. Surface it in Step 5's dashboard for a human; never clean it up.
- Check file **mtimes** of both `.epic-status.json` and `.epic-worklog.md`, not the `updated_at` field, which workers sometimes hand-type.
- The `phase` field is not evidence that work is done: check `gh pr view`.
- A pane's scrollback is current; its statusline is re-rendered on Claude Code's own schedule and can be stale. Don't use it to verify anything.
