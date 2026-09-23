---
name: bip-conductor-poll
description: Quick poll of GitHub activity and clone status since last check
---

# /bip-conductor-poll

Lightweight mid-session update for the fleet conductor.
Checks what changed on GitHub (as it bears on slots/clones) and in active clones since last check.
Use this instead of `/bip-conductor` when you already have context established.

Fleet-scoped, not topic-scoped: this skill reconciles slots against GitHub state and does housekeeping.
It does not update EPIC bodies — that's `/bip-epic`'s job, on its own cadence, driven by findings and completions rather than a poll timer.

For continuous monitoring, prefer `bip epic watch` (started by `/bip-conductor` Step 7).
It writes phase transitions to `.epic-notifications.log` in the conductor cwd; this poll skill reads new entries from that log to catch transitions the conductor may have missed.
Use `/bip-conductor-poll` for:
- Full GitHub reconciliation (merged PRs, new issues, comments) as it bears on slot/clone state
- Slot cleanup
- Catching up on log entries written while the conductor was idle

For periodic auto-polling: `/loop 10m /bip-conductor-poll`

## What to check

Four things stay in the primary because they're cheap and structured:

**Pull the conductor's own clone** — `git pull --ff-only origin main` in the conductor cwd.
`/bip-conductor` Step 2 pulls once at cold start and nothing pulls it again, so across a long session this tree silently rots while every GitHub answer stays correct.
**A role that only ever *reads* remote state has no natural moment that forces a pull**, and `git fetch` — which a conductor runs constantly to check merge state — updates remote-tracking refs and not the working tree.
Measured 2026-09-01: a conductor sat nine hours and four merges behind on `main`, clean tree and correct branch, and nothing surfaced it, because `git merge-base --is-ancestor <sha> origin/main` reads the fetched ref rather than the tree. Every merge verification it performed was right. The staleness produced no wrong answer, which is exactly why it survived.
(Two sessions hit this the same day; `/bip-epic` Step 1 carries the same instruction for the same reason.)

**Refresh the completion-push address** — resolve `CLONE_ROOT` and rewrite `$CLONE_ROOT/.conductor-session` with this session's current `ListAgents` name.
This is the file workers read to push a `needs-human`/`completed` notification without guessing among `ListAgents` rows — see `/bip-conductor`'s Conventions section ("Completion pushes") for why.
Cheap (one `ListAgents` self-lookup, one file write) and bounds how stale the address can get between conductor cold starts.

**Notifications log tail** — if `bip epic watch` is running, it appends one JSONL line per phase transition to `.epic-notifications.log` in the conductor cwd.
The state file `/tmp/.epic-poll-last-read` records the previous poll time as Unix seconds; pass the elapsed window to `--since`:

```bash
LAST=$(cat /tmp/.epic-poll-last-read 2>/dev/null || echo $(($(date +%s) - 3600)))
NOW=$(date +%s)
bip epic watch --since "$((NOW - LAST))s" 2>/dev/null
echo "$NOW" > /tmp/.epic-poll-last-read
```

Surface any `needs-human` / `completed` transitions prominently per the section below.

**Tmux window list** — `tmux list-windows -F "#W"` is one line of output and tells the subagent which slots to inspect.

Everything else is delegated.
Dispatch one `general-purpose` subagent for the combined poll (the poll is lighter than the cold start, so a single combined scan keeps round-trips down).
Follow the dispatch pattern in `SUBAGENT-SCAN.md` (bipartite repo root).
Brief:

> Delta poll for the EPIC conductor since the previous poll.
> Tmux windows currently open: `<list from tmux list-windows>`.
> Tasks:
>
> 1. `gh pr list --search "is:pr is:merged sort:updated-desc" --limit 5 --json number,title,mergedAt,body`.
>    For each new merge: note key results and whether it closes an issue.
> 2. `gh pr list --json number,title,headRefName,state`.
>    Note new PRs or CI status changes.
>    **For each open PR, name who is acting on it.** An open PR is being worked or waiting on
>    somebody, and waiting has no owner: approval is granted by writing to GitHub, and nothing
>    tells the worker it happened. Measured 2026-09-08 — three slots idled on this in one day,
>    one of them already fully approved and idle for twenty minutes. Approved-and-unmerged, or
>    unreviewed with nobody asked, is one message. **If a month passes with no PR this catches,
>    delete it.**
> 3. `gh issue list --search "sort:created-desc" --limit 5 --json number,title,state,createdAt`.
> 4. For each active slot (per the tmux list and any `.epic-status.json` in `<clone_root>`), check the latest issue-lead comment: `gh api repos/<owner>/<repo>/issues/<N>/comments --jq '.[-1].body'`.
>    Look for the `🤖 **Issue Lead**` prefix.
> 5. Inventory slots from `.epic-config.json`.
>    Clone mode: iterate `clone_names`.
>    Worktree mode: `find <clone_root> -maxdepth 1 -name 'issue-*' -type d`.
>    For each slot, read `.epic-status.json` and surface: phase, summary, scope, stop_reason, lead_guidance.
>    Migrate legacy phases: `blocked → needs-human`, `pr-review → quality-gate`.
> 6. For active slots, `git -C <slot> log --oneline main..HEAD | head -5` to see recent commits.
> 7. For slots that look finished or blocked, capture the last 20 lines of tmux: `tmux capture-pane -t <window> -p | tail -20`. **Do not read text at the `❯` prompt in that output as pending user input** — `-p` strips escapes, and Claude Code's autosuggest puts dim placeholder text at the prompt of essentially every idle window. See `/bip-conductor` Step 6 for the cursor-position test that distinguishes them.
> 8. Verify state with `gh pr view --json state` / `gh issue view --json state` for anything you plan to flag — never claim "open" or "merged" without a live confirmation.
>
> Return under 400 words, structured per `SUBAGENT-SCAN.md`:
> - `changes_since_baseline`: merged PRs, new issues, new issue-lead comments, slot phase changes
> - `active_items`: per active slot — clone, issue, phase, stop_reason, lead assessment (one line each)
> - `action_candidates`: pending spawn intent directly in `$CLONE_ROOT/.spawn-prompts/` (either `<N>.md` or `spawn-<N>.txt` — check both) waiting to be executed — ignore anything under `.spawn-prompts/consumed/`, that's already-launched intent, not pending; merged PRs that should trigger slot cleanup.
>   (Deciding *which* open issues are ready to spawn is `/bip-epic`'s call, not this poll's — report raw signal, not a readiness verdict.)
> - `surprises`: `needs-human`/`completed` slots, stale status files, contradictions, `RECOMMEND DEEPER LOOK` flags. Include the liveness sweep's result: any slot whose `.epic-status.json` mtime is >45 min old, and whether its `.epic-worklog.md` is also stale (stalled) or fresh (alive but not reporting). Also flag any `phase` value outside the seven spec-valid ones — an off-spec string silently escapes `bip epic watch`'s `--phases` filter.

If the report has zero `changes_since_baseline` and zero `surprises`, the poll output is one line: "All quiet."

## After polling

### Focus on what matters

**Lead with pending spawn intent** — `.spawn-prompts/` files (either naming pattern) the epic has already written and is waiting on the conductor to execute.
This is the most actionable information at this level; deciding which *other* open issues should be spawned next is `/bip-epic`'s call.

**Surface lead evaluations** — if a clone's status shows a recent lead evaluation (stop_reason set, lead_guidance present), mention the lead's assessment briefly.
This tells the conductor what the workers are doing without having to read full issue comments.

**Flag needs-human and completed** — if any clone has `phase: "needs-human"` (or legacy `blocked`) or `phase: "completed"`, highlight it prominently.
These require conductor attention.
Read `$CLONE_ROOT/.epic-session` and, if present, `SendMessage` that address the issue number and phase so `/bip-epic` can re-poll and update the EPIC body without waiting for its own cadence.
If the file is absent or the send fails, this is a no-op — `/bip-epic`'s independent `gh` polling remains the fallback path.

**Only report active clones** — clones with a tmux window that are actually doing something.
Don't list completed or idle clones; that's noise.
Completed clones can be mentioned briefly ("fir completed i374") but don't need a table row.

**Mention recent merges** only if they unblock something or change the plan.

**Correcting a live worker mid-flight**: `ListAgents`/`SendMessage` reach other local Claude tmux sessions and can push an immediate correction without waiting for the worker's next lead evaluation or killing its tmux window.
Send the correction directly — stating what changed in at least one line — rather than a bare pointer, which is a no-op for a worker that already reads the status file every step and strips the priority signal.
If a correction is long enough that pointing at a file seems preferable, its home is the spawn prompt or the issue body, not a nudge.
The durable/transient test is the deliverable: any nudge that changes what the worker will produce (scope, target, artifact, gate criterion) MUST also get a durable record, written by the **conductor** appending a timestamped, attributed entry to `.epic-worklog.md` in the same step — not `.epic-status.json`, since that file is the worker's own continuously-rewritten object and a second writer there is unsynchronized and gives no tiebreak against `lead_guidance`.
Facts that leave the deliverable unchanged (host load, a peer's timing, a dependency that just landed) may be message-only.
Delivery lands at the worker's next tool call, not instantly. Fall back to the file-only correction (a `conductor_guidance` field or a `lead_notes` entry tagged `source: conductor` — never `lead_guidance`; wait for the worker's own loop) only once you have established the target is genuinely unaddressable, per the rule immediately below.

**Addressing — read this before any `SendMessage`.**
Read the address off `ListAgents`' own row, or off the message you are replying to. **Never compose it.** The bare clone name is never an address, and the scheme itself is not something to derive: `bip spawn` names a worker after its tmux window (bipartite #241), while a session started any other way gets an auto-derived `<cwd>-<suffix>`. **A failed send to a name you typed yourself is a typo, not a channel limit** — re-run `ListAgents` and use the exact string. Never conclude from one failed send that workers are unreachable. (An address that came from a self-registration file and stops working is the different case: it drifted, so skip silently and don't hunt a substitute.)
Skip the nudge entirely for a worker in `awaiting-results` with a live `check_cmd`; use `notify_when_idle: true` instead of "tell me when this worker finishes" — main-conversation only, this-machine only, one-shot.

### Output structure

1. **Pending spawn intent**: `.spawn-prompts/` files (either naming pattern) waiting on execution, plus idle clones available to run them.

2. **Active work**: Clones with tmux windows that are mid-task.
   One line each: clone, issue, phase, stop_reason (if set), lead assessment.

3. **Needs human**: Clones in `needs-human` phase — show the lead's assessment and what decision is needed.

4. **Decisions and negative list**: read new entries from `.epic-decisions.md` in the conductor cwd since the last poll (`FINAL` relays, forwarded worker findings, negative-list entries — see `/bip-conductor`'s Conventions, "Decision relays" and ".epic-decisions.md: the durable fleet-decision log") and report them here; this is the file's canonical output surface for a poll cycle, not a fresh derivation.

5. **Recently landed** (brief): PRs merged since last poll, only if noteworthy.

6. **Execute pending spawns**: If spawn intent files and idle clones both exist, run `/bip-conductor-spawn` for them and report after — placement and timing are the conductor's call, not a question to hold the fleet on (see `/bip-conductor`'s "Arbitration"). Escalate only a scientific question, or a risk of an actual problem: data loss, a clobbered checkout, two slots on one deliverable. Rebase friction is not that.
   **A busy fleet is fine as long as every slot is on topic — the gate is topic, not count.** Spawn ready, in-scope, unblocked briefs while capacity exists; don't throttle on volume. Whether an issue belongs to the programme at all is the originator's call (the epic agent for its own EPIC, the user for a direct or `/bip-ms` request) — but once it is in scope and ready, holding it back is a failure, not caution.

### Housekeeping (do silently, don't report unless problems)

This is the ongoing cleanup that keeps slots current between cold starts.
Do it every poll cycle — don't wait for `/bip-conductor`.
EPIC body content is not this skill's concern; `/bip-epic` keeps bodies current on its own cadence.

#### Mirror every live worklog (do this FIRST, before any cleanup)

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
mirror_worklogs "$CLONE_ROOT" || echo "MIRROR: at least one worklog could not be copied -- read the lines above" >&2
```

Silent on success. It prints only `MIRROR UNREADABLE`, `MIRROR FAILED`, or
`MIRROR SHRANK`, and returns non-zero for the first two — there is no count in
it anywhere, so it cannot report a reassuring zero for a clone it could not read.

**Why this runs every cycle rather than at land time.** A worklog lives only in
a pooled clone until its PR merges. `/bip-pr-land`'s Step 6a preserves it —
**but only for landings that go through `/bip-pr-land`.** A direct `gh pr merge`
bypasses Step 6a, Step 9.5, and the `🤖 EPIC worklog preserved` PR comment in one
move, and nothing anywhere notices. Measured 2026-09-14 on `matsengrp/phyz`: a
PR landed that way and left a **35,432-byte** worklog live in a pooled clone,
where the next spawn's prep would have deleted it.

⛔ **Do not replace this with detect-merge-then-preserve.** That races the next
spawn's prep, and when it loses, "files already gone" is indistinguishable from
"this landing was never bypassed" — a detector whose failure mode is its success
case. Mirroring has no such state: the copy exists before any merge, by any
path, and never needs to know *how* the PR merged, which is the one fact that is
unobservable from outside the slot.

⚠ **The mirror is NOT `.preserved/`.** Live, overwritten, best-effort — a floor
under data loss, not an archive. `.preserved/` is the final authoritative record
with provenance READMEs. **If they disagree, `.preserved/` wins.** A mirror
*larger* than a matching `.preserved/` entry means preservation ran **early** —
that is a `/bip-pr-land` timing question, so **report it rather than
reconciling it**. Never delete a `.preserved/` entry because a mirror exists.

`MIRROR SHRANK` is not an error and needs no action beyond noticing: the helper
keeps the longer copy alongside rather than overwriting it, and returns 0. It
fires on **same issue, same clone, smaller file** — truncation, a worker
compacting its own worklog mid-issue, a re-spawn onto the same issue, or a
worker resuming after landing and re-creating its state (which the spawn prompt
itself instructs). ⚠ **The ordinary reclaim path does not trip it**: the next
spawn carries a different issue number, so it writes to a different mirror path
and the old entry is simply left alone. Most of what does trip it is
legitimate, which is why it is not an error — a signal that fires on a normal
day stops being read.

**`.epic-status.json` is mirrored differently and deliberately: every distinct
version is kept, timestamped, never overwritten.** Size-keyed shrink protection
is the **wrong instrument** for it — a status file can shrink while *gaining*
the field you care about (a lead replacing a long `stop_reason` with a short one
while adding `completed_at`). Content matters and size does not track it, and
the files are 1–6 KB, so keeping everything costs nothing and removes the need
to guess which version mattered. A new copy is written only when the content
actually differs from the newest kept one, so an idle slot does not accumulate
an entry every cycle.

⚠ **Why status files are mirrored at all, which is not symmetry with the
worklog.** Measured 2026-09-14: an issue-lead's terminal assessment carried two
real findings — a wrong number in a landed doc, and a straddle result that
discharged an open hedge for 3 of 5 cells at zero compute — and **both lived in
the lead's output, not in the worklog.** A worklog-only mirror would have lost
them. The worklog carries narrative; lead output carries verdicts.

⚠ **`.mirror/` is neither an archive nor a view of current slots.** Entries are
issue-scoped and are never cleaned, so landed issues accumulate there
indefinitely — that is deliberate, it is the floor under data loss. **Do not
`ls .mirror/` to see what the fleet is working on**; it returns a superset of
everything that has ever run. Use the slot table for current state and
`.preserved/` for the archive.

#### Sweep for work that nothing would recover

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
audit_durability "$CLONE_ROOT" .epic-config.json || echo "DURABILITY: see the lines above" >&2
```

**Relay the LOUD lines (`DURABILITY UNCOMMITTED` / `UNPUSHED` / `NO-UPSTREAM` /
`DETACHED` / `UNCHECKABLE`) to the slot; do not commit or push on a worker's
behalf. Do NOT relay `DURABILITY note` lines** — those are state, not defects,
and only the loud class sets a non-zero return.

⛔ **Severity is gated on SLOT PHASE, not on commit count, and that distinction
is load-bearing.** Detection is unconditional — uncommitted work lives only in
the working tree whether or not the branch has commits — but **`dirty +
commits` on a working slot is a state, and reporting a state through a channel
meant for defects is what kills a guard.** Measured 2026-09-14: undifferentiated,
this fired on **four of five live slots every cycle**, all of them simply
working. The 4-of-5 rate is the argument for *classifying* the common case, not
for narrowing detection back to where it was silent on all four.

| condition | class |
|---|---|
| dirty + **zero commits**, any phase | **loud** — exists nowhere else |
| dirty + **no status file** | **loud** — post-land leftovers; `/bip-pr-land` Step 9.5 removes the status file and does **not** clean the tree, so the next spawn's prep takes these |
| dirty + phase `completed` / `needs-human` / **`quality-gate`** | **loud** — finished, stopped, or about to land |
| dirty + phase `exploring` / `coding` / `testing` / `awaiting-results` | note only |

⚠ **`quality-gate` is the non-obvious member of the loud set and belongs there:
a slot about to land with uncommitted files is exactly the case where those
files silently do not make the PR.** It reads identically to the routine case
unless phase distinguishes it. **First real run caught one** — a slot in
`quality-gate` holding an *untracked* new test alongside a `build.zig` change
that wires it in, which would have produced a PR referencing a file not in the
commit.

⭐ **Measured 2026-09-14, and the numbers are the argument**: a spot check of one
slot led to sweeping all six, and **four were holding unrecoverable work in
three different shapes**. **No single probe finds all three** — a sweep that
checks only "is the tree clean" reports two of them safe.

| line | shape |
|---|---|
| `DURABILITY UNCOMMITTED` | dirty tree, **zero commits**. One slot held all four of its issue's pipeline-defect fixes — the entire deliverable. `@{u}..HEAD` returns 0 for this; only `status --porcelain` sees it. |
| `DURABILITY DETACHED` | detached HEAD with a live status file. ⚠ **`git status` reads CLEAN here, which is why it is the worst of the three.** A bisect had narrowed a 30-commit window with every probe result in conversation context at 99.8% of the model's window — nothing on disk, nothing in the branch. |
| `DURABILITY UNCOMMITTED` (with commits) | dirty files on a branch that *does* have commits. ⚠ The first draft required zero commits as well, which made `dirty=2, ahead=3, up=0` fire **nothing at all** — silent, two files at risk. Uncommitted work lives only in the working tree whether or not the branch has commits; only the alarm differs. |
| `DURABILITY UNPUSHED` / `NO-UPSTREAM` | commits existing only in one clone. Lower risk (objects survive the prep's `git checkout main`) but a lost disk or forced reset takes them. |

⛔ **Slot-ness comes from `.epic-config.json`'s `clone_names`, not from
`.epic-status.json` existing** — and the difference is a real loss channel.
`/bip-pr-land`'s Step 9.5 removes the status file and **does not clean the
working tree**, so a slot that lands with uncommitted scratch has leftovers and
no status file, reads as idle to every mechanism, and loses them to the next
spawn's prep. A status-file gate conflates *"not a slot"* with *"a slot with no
status file"*; the clone list distinguishes them, and still keeps pinned
comparator clones in the same root out of the report. **A missing or unreadable
config is reported loudly rather than guessed around.**

⚠ **A clean `git status` has meant three different things on this fleet in one
day** — *"already preserved"*, *"preservation never ran"*, and *"nothing was
ever saved"*. **The discriminator is always something else** — `ahead`, an
upstream, a PR pointer comment — never the status output.

⛔ **This does NOT subsume the mirror and the mirror does not subsume this.**
`mirror_worklogs` covers `.epic-worklog.md` and `.epic-status.json` only.
Source, results and a detached HEAD have **no** mirror and should not get one —
mirroring source duplicates git badly. They have a git-native observable
instead, which is what this checks. **Closing one loss channel is the moment to
enumerate the others, not the moment to stop looking**: the mirror was built
first, and it took an unrelated observation to surface four exposed slots an
hour later.

#### Audit every status file for claims that are false

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
audit_status_files "$CLONE_ROOT" || echo "AUDIT: see the STATUS lines above" >&2
```

Silent on a clean pool. Every line it prints is a status file **asserting
something false**, which is invisible unless something compares the file against
reality:

- **`STATUS FUTURE-TIME`** — a timestamp ahead of the clock. ⚠ **Direction
  matters**: a *past*-dated `started_at` invents a timeout, which is loud and
  gets investigated; a *future*-dated one **hides a stall**, because the run
  keeps reading as having budget left and a hung job is never escalated.
  Measured 2026-09-14: `updated_at` 62 minutes ahead, round to the whole minute
  — which two `date -u` calls cannot both be. Clock skew and timezone were both
  ruled out (`System clock synchronized: yes`; a TZ error here is 7 hours, not
  62 minutes), so the fit is a worker writing an **ETA** into a field that means
  *last touched*.
- **`STATUS SUSPICIOUS-TIME`** — a timestamp ending in `:00` seconds. ⚠ **A
  heuristic, not a violation**: `date -u` distributes seconds uniformly, so
  ~1 in 60 legitimate values trip it. Treat it as *verify*, not *wrong*.
  ⭐ **It exists because the diagnosis moved.** Future-dating was first read as a
  worker writing an **ETA** into a last-touched field — until one of the bad
  values turned out to be written by an **issue-lead subagent**, which has no
  schedule to project. The surviving fit is that an agent **constructed** a
  plausible timestamp from its own sense of the time, rounded to the minute:
  that explains the `:00` seconds, the round offset and the forward direction
  together. Measured against 15 correctly-written files in the live pool, **zero**
  ended in `:00` (seconds seen: 05 06 08 10 13 15 16 19 24 34 42 44 58); both
  bad ones did.
  ⛔ **And it catches what `FUTURE-TIME` structurally cannot: a constructed
  timestamp landing in the PAST.** That one is the *loud* failure rather than
  the fatal one — it invents a timeout instead of hiding a stall — but it is
  equally fabricated.
- **`STATUS NO-AWAITING-BLOCK`** — `phase: awaiting-results` with no `awaiting`
  block, so the loop has nothing to check while the file reads as monitored.
  **Two of two slots that reached that phase on 2026-09-14 got it wrong, in
  different ways.** Two of two is a spec ambiguity, not two slips: the phase
  *name* reads as "I am waiting" while its *contract* is "I have a live
  readiness probe."
- **`STATUS OFF-SPEC-PHASE`** — a `phase` outside
  `exploring|coding|testing|awaiting-results|quality-gate|needs-human|completed`.
  ⛔ **Do not fix this by widening a filter.** The known instance
  (`phase: "premature-deferral"`, a `stop_reason` value in the `phase` field)
  surfaced *only* because `bip epic watch`'s `--phases` list had previously been
  widened to absorb that exact string. **That was the wrong direction**:
  widening a filter to accommodate an off-spec value converts a schema violation
  into a silent success, and the filter then does exactly what it was told
  against a value set that no longer means anything. **An off-spec phase must
  shout.** (This check found a *second* off-spec value, `implementing`, on its
  first run against a live pool.)
- **`STATUS UNREADABLE` / `STATUS UNPARSEABLE-TIME`** — the file or the field
  could not be parsed. Both are findings, not skips.

**The fix is always the worker's**, not yours: relay the line. These are claims
only the slot can correct, and a conductor editing `lead_guidance` or a phase is
how two writers end up indistinguishable in one file.

#### Slot cleanup for merged PRs

For each slot whose PR has merged (cross-reference merged PRs from check 1 with slot branches):

**First, the issue-lead's terminal ceremony, if nothing else will run it.** Where a human merges (`LANDING DELEGATION: NONE RECORDED`), the worker ended at a clean gate with `stop_reason: awaiting-human-merge`, and nothing after the merge calls the lead. You are that call's owner. It has to run **before** the preserve-and-checkout below, because that step removes the state files the lead reads.

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
post_merge_ceremony "$CLONE_ROOT/<slot>" <owner/repo> <PR number>
```

`<slot>` is the clone name, or `issue-<N>` in worktree mode. It prints one line:

- **`CEREMONY RAN #<pr>`** → a terminal lead comment is on the PR. Go on to the cleanup.
- **`CEREMONY OWED #<pr> <slot-dir>`** → no terminal comment, and the status file is still there. `/bip-pr-land` deletes that file, so it did not run, and no worker lead is going to run the ceremony. If the slot's session is `busy` in `ListAgents`, its own lead may be mid-run: skip this slot for this cycle. Otherwise spawn the lead:

  > Agent tool, `subagent_type: issue-lead`: *"Post-merge terminal ceremony for <owner/repo>#<issue>, PR #<N>, which `gh` reports MERGED. The slot's clone is `<slot-dir>`. Read its `.epic-status.json` and `.epic-worklog.md` there, run git as `git -C <that path>`, and pass `-R <owner/repo>` to `gh`. Follow your full evaluation protocol; Step 8 applies."*

  Leave this slot's cleanup until the lead returns. Then re-run `post_merge_ceremony` yourself, because a subagent's report is a snapshot. It must print `CEREMONY RAN`; then do the cleanup. The lead writes `phase: completed` into the status file before the cleanup preserves it, so the `.preserved/` copy records the ceremony.
- **`CEREMONY WORKER-OWNS #<pr>`** → `/bip-pr-land` ran (its `🤖 EPIC worklog preserved` comment is on the PR), so the worker's own final lead call owns the ceremony. Go on to the cleanup; the reclaim gate in `/bip-conductor` Step 6 is what keeps a mid-ceremony worker alive.
- **`ceremony UNRUN for #<issue> (PR #<pr>)`**, exit 1 → no terminal comment, and the state is already gone. Put the line in this poll's report; do not skip it silently. A lead that filed nothing and a lead that never ran look the same from outside. The user decides whether a lead run from the PR alone is worth it. Its guard and its follow-up source (the PR body's DEFERRED section) are both on the PR, but the worklog it would have read is gone.
- **`CEREMONY UNKNOWN #<pr>: …`**, exit 2 → the PR is not `MERGED`, or `gh` failed. Clean up nothing for this slot this cycle.

**Worktree mode**:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
# Confirm PR is merged before removing
gh pr list --head <branch> --state merged --json number | jq length
# If merged:
git worktree remove "$CLONE_ROOT/issue-<N>"
git branch -d <branch>
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-conductor-poll`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory (see `skills/lib/spawn-intent.sh` in the `bipartite` repo).

**Clone mode**:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
git -C "$CLONE_ROOT/<clone>" checkout main
git -C "$CLONE_ROOT/<clone>" pull --ff-only origin main
# After a worker's own land, .epic-status.json/.epic-worklog.md are already
# gone -- /bip-pr-land's Step 6a preserves them (and Step 9.5 deletes them)
# at land time (issue #2216). They are still here in two cases: a human
# merged (stop_reason awaiting-human-merge -- expected; the ceremony above
# has just run from them), or a land bypassed /bip-pr-land (a defect to
# report). Either way, preserve before deleting rather than assuming
# reclaim is a safe place to drop them silently. Uses the same
# preserve_epic_state() helper Step 6a does (issue #2216 follow-up) rather
# than a hand-rolled variant -- a backstop that behaves differently from
# what it backstops is its own silent gap.
if [ -f "$CLONE_ROOT/<clone>/.epic-status.json" ]; then
    ISSUE_N=$(jq -r '.issue // "unknown"' "$CLONE_ROOT/<clone>/.epic-status.json" 2>/dev/null)
    if [ "$(jq -r '.stop_reason // ""' "$CLONE_ROOT/<clone>/.epic-status.json" 2>/dev/null)" = "awaiting-human-merge" ]; then
        WHY="at reclaim, after a human merged the PR and the post-merge issue-lead ran."
        NOTE="after the human merge"
    else
        WHY="at reclaim. This clone landed a PR without /bip-pr-land preserving first -- investigate why."
        NOTE="the land that closed this issue skipped /bip-pr-land's preservation step"
    fi
    DEST=$(preserve_epic_state "$CLONE_ROOT/<clone>" "$CLONE_ROOT" "$WHY")
    rc=$?
    if [ "$rc" -eq 0 ]; then
        echo "Reclaim preserved EPIC state for issue $ISSUE_N to $DEST ($NOTE)"
        gh issue comment "$ISSUE_N" --body "🤖 EPIC worklog preserved to \`$DEST\` at reclaim ($NOTE)." 2>&1
    elif [ "$rc" -eq 2 ]; then
        echo "PRESERVATION FAILED at reclaim for issue $ISSUE_N -- stop, do not let the delete below run until this is resolved by hand" >&2
    fi
fi
cd "$CLONE_ROOT/<clone>"
```

Then, as a **separate command with no `$` in it at all** (a command
containing both `$` and `rm` re-arms Claude Code's built-in
destructive-removal guard even if `$CLONE_ROOT` was only expanded earlier
in the same string — see `bip-conductor-spawn`'s Step 2 for the verified
trigger condition; the `cd` above, already in its own command, is what
makes literal relative filenames possible here):

```bash
rm -f .epic-status.json .epic-worklog.md
```

**Before returning a clone to the pool, check what untracked output it is carrying.**
Reclaiming is safe for tracked files and **silently destructive for untracked ones** — and untracked is where experiment output lives *by policy*. Nothing in the reclaim removes it, so it survives until the next spawn into that clone quietly destroys it. **A free clone is not an empty clone.**

**The same trap used to catch `.epic-worklog.md` too, and it is gitignored rather than untracked — so a `git status` check does not show it.** The premise that used to excuse this ("a slot that lands a PR needs nothing, since the issue-lead posts every evaluation as a PR comment") is false (issue #2216, matsengrp/phyz#2314/PR#2316): `PROSE-DISCIPLINE.md` rewrites PR bodies to current state by design, so deliberation never lives there, and the issue-lead's PR comments are evaluation-stop summaries, not the worklog's full narrative — on a landed PR the worklog is one of only two places the reasoning survives. `/bip-pr-land`'s Step 6a is now the point that preserves it, at land time (before Step 8 can destroy a worktree, and before this reclaim step ever runs); the `if` block above is only a backstop for a land that skipped that skill. A slot that **stands down or escalates** (no PR at all) still needs the same care here, since nothing upstream of reclaim preserves it in that case: copy `.epic-worklog.md` to `$CLONE_ROOT/.preserved/<slug>/` before reclaiming. Measured 2026-09-03: eleven were rescued in one stand-down, the largest 269 lines from a slot that wrote no code at all.

```bash
# gitignored OR untracked output under experiments/*/results
git -C "$CLONE_ROOT/<clone>" status --porcelain --ignored=matching -- 'experiments/*/results' \
  | awk '$1 == "??" || $1 == "!!" {print $2}'
```

**`--ignored` is not optional, and a plain `du` is the wrong tool — both mistakes were made writing this rule.** Experiment output is usually *gitignored*, not merely untracked, and **`git status --porcelain` does not show gitignored paths at all**: run against the clone that nearly lost its sweep, plain `status` on that directory returned **0 entries** while `--ignored=matching` returned 7. That silence is exactly how it went unnoticed. Conversely a bare `du` cannot tell tracked from ignored — measured on another clone, it reported 71M under `experiments/*/results` that was entirely *tracked reference data*, i.e. safe and already in git. Size is not the signal; reachability by `git` is.

If a directory turns up, the question is not whether it looks disposable — it always does — but **whether anything cited points into it**. Check the open issues and PRs for the experiment's name before reusing the clone. If something cites it, copy it to `$CLONE_ROOT/.preserved/<name>/` with a README saying what cites it and why it is not committed; that path is outside every clone's git tree, so no `git clean` can reach it.

**A rescue is not finished until something *in the repo* points at it.** `.preserved/` is a dotfile directory in a non-git clone root — exactly as invisible to a reader as the untracked output it rescued, and invisible for the same reason: the explanation sits somewhere the reader never goes. Measured 2026-09-02, hours after the first rescue: `grep -rn '\.preserved'` across the consuming repo returned **nothing**. So a rescue has three parts, not two — copy the data, write the local README, **and land a committed pointer** where someone hitting the gap would actually look (the experiment's own `README.md`, next to the numbers whose source went missing). Issue bodies are not sufficient: they reach whoever works those issues and nobody else. Without the third part `.preserved/` just accretes rescued directories that nothing references, which is the original failure relocated one level up.

Measured 2026-09-02: a clone was returned to the pool still holding the **only** copy of a 391-job sweep's output — including the exact file a live issue cited twice in its load-bearing argument, on a gitignored path absent from the repo. The next spawn into that clone would have destroyed it, and nothing in the reclaim would have warned anyone. The tell was never the clone's identity; it was that an open issue's citation resolved to a path only that clone had.

Also clean up stale slots: no tmux window AND `.epic-status.json` older than 30 minutes.
Same cleanup as above.

#### The liveness sweep — run this every poll, it is the only stall detector

`bip epic watch` reports transitions, so a slot that stops transitioning is invisible to it and nothing else detects that. A stale status file means two opposite things depending on whether a tmux window is open, so the window check belongs *inside* the sweep. Same 30-minute rule as above, extended to the window-open case rather than a second competing rule:

```bash
for f in "$CLONE_ROOT"/*/.epic-status.json; do
  [ -n "$(find "$f" -mmin +45 2>/dev/null)" ] || continue
  d=$(dirname "$f")
  if [ "$(jq -r '.stop_reason // ""' "$f" 2>/dev/null)" = "awaiting-human-merge" ]; then
    echo "AWAITING-MERGE $d — clean gate, PR is the user's to merge; not a stall"
  elif tmux list-panes -a -F '#{pane_current_path}' 2>/dev/null | grep -qxF "$d"; then
    echo "STALLED?  $d — window OPEN, never clean up; check the worklog next"
  else
    echo "ABANDONED $d — no window, cleanup candidate per the rule above"
  fi
done
```

For anything the sweep marks `STALLED?`, check its worklog mtime to tell a stall from a quiet worker:

```bash
find "$CLONE_ROOT/<clone>/.epic-worklog.md" -mmin +45   # empty output = worklog is fresh
```

- **status stale + worklog stale** → *candidate* stall only. **Run the `ps` check below before calling it one — live compute vetoes a stall verdict outright.** If nothing is running, escalate to the user. Never clean up either way: the window is open and may hold typed human input.
- **status stale + worklog fresh** → alive and working; the status file is simply lying. Not a stall — nudge it to resume writing status, and don't report it as dead.

**The `ps` leg is a veto on both branches, not a tiebreak on one.** Run it before reporting either verdict:

- **Compute running → not stalled, whatever the mtimes say.** A worker in a quality-gate loop runs multi-minute builds and full suites and writes no worklog entry for hours. Measured 2026-09-01: a slot **243 minutes stale on status and 234 on worklog** — both legs firmly "stalled" — was at that moment running `zig build-exe` at **99.9% CPU** under its own cache. Reporting it dead would have been exactly wrong, and it held the PR gating the whole queue.
- **No compute → now the mtimes decide**, per the two branches above.

**And on the other branch, a fresh worklog is necessary and not sufficient.** "Worklog fresh" bounds how long ago the worker last *thought*, not whether its background job is still alive — a worker can write a perfectly accurate "waiting on the sweep to finish" entry and then wait forever on a job that already died. Measured 2026-09-01: a slot passed the worklog check, and its pane said "3 shells still running", while `ps` showed no process at all under its clone.

```bash
ps -eo pid,args | grep -F "$CLONE_ROOT/<clone>" | grep -v grep   # any compute still alive?
```

If the worklog says it is waiting on something and no process matches, it is stalled regardless of the mtimes: tell it the job is gone, and where its output actually did or did not land. Do not trust a pane's "N shells still running" indicator — it goes stale when the session idles.

Use mtimes, not the `updated_at` field: a worker that writes a placeholder timestamp defeats the field but not the mtime (measured: a status file 6h old and frozen on `phase: exploring` while its worklog had been written 4 minutes earlier).

See `/bip-conductor`'s `.epic-status.json` spec for the `phase`-is-not-evidence corollary.

If a merge closes an issue tracked in an EPIC, that's signal `/bip-epic` needs, not something to act on here — surface it under `changes_since_baseline` and move on.

#### Check the watcher itself — it can be ALIVE and not observing

⛔ **The sweep above assumes `bip epic watch` is working. It can be running, correct in `ps`, and silently emitting nothing.** Measured 2026-09-16 on `matsengrp/phyz`: the watcher had **19 hours of uptime**, a correct command line, and had written **nothing for 13 hours** while two live slots wrote status files and one transitioned `coding -> awaiting-results`. Restarting it fixed it instantly.

⚠ **Nothing surfaces this on its own, because a silent instrument is indistinguishable from the thing it reports.** A quiet `.epic-notifications.log` reads exactly like a quiet fleet, and a `tail -F` Monitor on it stays faithfully silent forever. It was caught only because a Monitor *expiry* notice — which fires on a timer, not on evidence — prompted someone to open the file.

⛔ **DO NOT compare the log's mtime to the newest status-file mtime. That check cries wolf and this skill's own instructions are why.** Workers are told to refresh `summary`/`updated_at` before and after every build *without changing phase*, so a healthy watcher shows an arbitrarily large gap whenever a slot sits in one phase. Measured the same day: a **34-minute** "stale" reading with the watcher perfectly healthy. That is the adjacent-question trap — status mtime answers *"is the worker alive"*, not *"did the watcher observe a phase change"*, and the two come apart exactly when a slot holds one phase for a long time, which is the normal state.

➡ **Compare PHASES, not mtimes:** each slot's on-disk `phase` against the watcher's last recorded `new_phase` for that slot.

```bash
python3 - <<'PY'
import json, os
cfg = json.load(open('.epic-config.json'))
last = {}
for line in open('.epic-notifications.log'):
    try:
        d = json.loads(line); last[d['slot']] = (d['ts'], d['new_phase'])
    except Exception: pass
for c in cfg['clone_names']:
    f = f"{os.path.expanduser(cfg['clone_root'])}/{c}/.epic-status.json"
    if not os.path.exists(f): continue
    cur = json.load(open(f)).get('phase')
    ts, wp = last.get(c, (None, None))
    flag = 'ok' if cur == wp else 'MISSED TRANSITION'
    print(f"{c:10} on-disk={cur:16} watcher={str(wp):16} {flag}  (watcher ts {ts})")
PY
```

Disagreement = a missed transition, so restart the watcher. **Agreement = nothing to see, however old the log is.**

⚠ **What it cannot see, stated so nobody reads it as complete: an `A -> B -> A` that resolved while the watcher was wedged reads as agreement.** It catches a wedge that is *still in effect*, which is the case you can act on.

**A restart silently re-baselines the whole fleet** (`/bip-conductor`'s Step 7), so reconcile every slot by hand **before** restarting, not after — the restart is itself a monitoring gap. The baseline survives a restart and survives a status file being deleted and rewritten with a different `issue`, so neither is a candidate explanation for silence; a wedged process is.

#### Processing something once registers as knowing it

⭐ **The generalization of the note above, and the one that fires far more often: you trust a stale internal state over a cheap external read — and the state you trust most is the one you personally put there.**

⛔ **A fact you relayed, acknowledged, or acted on is the one you are LEAST likely to re-check**, because handling it registers as knowing it. Three instances on `matsengrp/phyz` on 2026-09-16, in three different sessions, none of them careless:

| | what was trusted | what was true |
|---|---|---|
| **conductor** | reasoned from an issue body's pre-strike state — **having relayed the strike itself**, hours earlier | the criterion it cited had been struck |
| **conductor** | relayed a peer's proposal for a "free single-axis contrast", and **re-derived its motivating table**, which confirmed | the arm was disqualified by a line the conductor had forwarded an hour before: *byte-identical treefiles* |
| **epic** | reported a PR as "acked pending its one cut" | the cut had been made and the PR had merged 15 minutes earlier |

⚠ **The second is the instructive one: re-deriving a NUMBER is not testing the PREMISE that makes it relevant.** The table was right. The inference it invited was already dead. A confirmation on the wrong axis is what makes you stop looking.

➡ **The remedy is one line and it is cheap: re-read the artifact, not your memory of acting on it.** Before citing an issue criterion by number, `gh issue view` it. Before reporting a PR's state, `gh pr view` it. Before relaying a proposal, grep your own outbound log for the artifact it depends on.

⭐ **An acknowledgement is the highest-risk act of all**, because it feels terminal to the actor while the work continues elsewhere. **Nothing you acked is in a state you know.**

#### Filter and route fleet-level findings

Before recording anything anywhere, run each candidate fleet-level finding through this filter (same as `/bip-conductor-tuckin` Step 3):

1. **Is it derived?**
   Recomputable from `git`, `tmux`, `gh`, or the filesystem — record nothing.
   Host quirks (which host had a warm cache, which was mid-build) are derived and go stale within hours; don't write them down anywhere, including MEMORY.md.
2. **Is it already recorded?**
   A finding that produced an issue, PR, test, or doc needs no second copy.

Only what survives both gates gets a destination: a durable clone-pool layout decision → a `CLAUDE.md`; a workflow rule → a skill; a **fleet-level decision** (a `FINAL` relay, a forwarded worker finding, a negative-list entry) → `.epic-decisions.md` in the conductor cwd, not `/bip-epic`'s own memory — that memory is unavailable wherever the auto-memory directory is deliberately skipped (`bip-epic/SKILL.md`'s Step 1 explicitly allows this, and some setups do it), and `.epic-decisions.md` has no such gap since it's a plain file, not a memory-directory feature.
Nothing else fits → the poll report itself.

## Conventions

Same as `/bip-epic`: `iN`/`pN` prefixes, full URLs on first mention.
Tmux windows named `NNN-YYY` where NNN is the issue number and YYY is the clone/slot name (e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).

**Decision relays are marked `PROVISIONAL` or `FINAL`, nothing unmarked.**
When this conductor session is the one talking to the user mid-poll and a decision results, push it to `/bip-epic` via `$CLONE_ROOT/.epic-session` prefixed `PROVISIONAL` or `FINAL` and append it to `.epic-decisions.md` — see `/bip-conductor`'s Conventions, "Decision relays: PROVISIONAL and FINAL", for the full mechanics; this is the same rule, restated here because a conductor mid-cycle is reading this skill, not the cold-start one (the durable/transient nudge test above duplicates the same way).

**Resolve a citation before acting on it, here too.**
A worker's finding forwarded mid-poll carries the same citation risk as at cold start — a bare filename with no directory is unresolved, and reconstructing one from context produces a real `find` result for an invented question. See `/bip-conductor`'s Conventions, "Resolving a citation before acting on it", for the full mechanics; restated here for the same reason as the decision-relay rule above.

## Layout config (issue #149)

`.epic-config.json` keeps working.
The newer global `layout:` block in `~/.config/bip/config.yml` configures worktree mode for non-EPIC `bip spawn`; see `docs/guides/layout.md`.
