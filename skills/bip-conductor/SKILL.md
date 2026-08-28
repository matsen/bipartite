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

## Role

The conductor session owns the clone pool, tmux, and host state:
- Scans clones/worktrees, tmux windows, and (via `/bip-scout`) remote hosts
- Runs the pre-launch staleness check before every spawn (is the named blocker actually still open?)
- Executes spawns — composing the fleet-fact annotation on top of intent the epic already drafted, or from scratch for conductor-initiated work
- Cleans up finished slots and sweeps unfiled draft issues, with a filed-vs-unfiled guard that never deletes authored-but-unfiled work
- Never adjudicates a genuine resource conflict itself — states it and asks the user (see "Arbitration" below)
- Never does topic reasoning, and does not write code or create branches for numbered issues (light triage — reading files, checking CI output, running `gh` commands — is fine)

## Arbitration

When two things want the same clone, cache, or host, or when the epic's spawn intent conflicts with what the conductor observes live, this skill does not run a heuristic to pick a winner.
State the conflict plainly and ask the user.
The user-confirmation gate every spawn already goes through is the same gate that resolves this — no separate arbitration rule is needed on top of it.

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
  Any nudge that changes what the worker will produce — scope, target, artifact, gate criterion — is state that matters and MUST be recorded durably in the same step as the message, by the conductor appending a timestamped, attributed entry to `.epic-worklog.md`.
  Workers treat that file as append-only and never edit previous entries, so a conductor append cannot clobber worker history the way editing `lead_guidance` can: it is a field every spawn prompt attributes to the lead, so a second writer makes a conductor nudge indistinguishable from a lead verdict, with no tiebreak when they disagree.
  If the status file is written at all, use a field the lead does not own (`conductor_guidance`, or a `lead_notes` entry tagged `source: conductor`) — never merge into `lead_guidance`.
- Delivery is not instant: the message drains at the worker's *next tool call*, not mid-tool-call.
  It is not a substitute for `tmux capture-pane` when the conductor needs to see current state right now.
- `SendMessage` only reaches addressable Claude sessions (tmux Claude windows on this machine, or connected cloud/Remote Control sessions) — never a plain shell, a remote SSH job, or a non-Claude compute node.
  Run `ListAgents` first to confirm the target session is actually addressable; fall back to the file-only correction (a `conductor_guidance` field or a `lead_notes` entry tagged `source: conductor` — never `lead_guidance`) and wait for the worker's own loop when it isn't.
  **The address is the session name `ListAgents` reports, not the tmux window name the conductor assigned at spawn** — `SendMessage` to a window name can fail with `No agent named '<window>' is reachable.`
- Do not use `SendMessage` to route around the conductor's own restrictions (cross-session permission laundering) — the "should not write code or create branches for numbered issues" rule above applies equally to instructions phrased as a message to a worker.
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

When the conductor is the session talking to the user and a decision results, push it to `/bip-epic` rather than letting it sit only in this session's own conversation: read `$CLONE_ROOT/.epic-session` for the epic's self-registered address (same mechanics as "Completion pushes" above) and `SendMessage` it the decision, **prefixed `PROVISIONAL` or `FINAL`. Nothing goes unmarked.**

- **`PROVISIONAL`**: the decision is still being discussed, or the conductor is relaying a first read before the user has confirmed it. `/bip-epic` may note that a decision is pending but must not write it into an EPIC body — see that skill's "Fleet state is derived" section for the FINAL-only body rule this feeds.
- **`FINAL`**: the user has confirmed the decision's shape. This, and only this, authorizes `/bip-epic` to write the decision into an EPIC body.
- Append every `FINAL` relay (and, once it firms up, the `PROVISIONAL` relay it followed from) to `.epic-decisions.md` in the conductor cwd, timestamped and attributed — see ".epic-decisions.md: the durable fleet-decision log" below. A relay that lives only in the message is gone the moment either session compacts; the body-write authorization in `/bip-epic` depends on being able to re-derive that a `FINAL` marker was actually sent.

This is one-directional: epic→conductor relays (spawn intent, worker corrections) are unaffected by this rule, since the conductor writes no citable artifact from them.

### Forwarding worker findings

A worker's semantic finding — "this assumption doesn't hold," "this arm was already moved" — reaches the epic through the conductor, and passing it through invites the conductor to interpret it along the way.
Don't. Forward the finding **verbatim and attributed** to the worker (issue/slot) that produced it.
If the conductor has its own reading of the finding worth adding, add it as a separate, clearly marked line — never blended into the forwarded text so that the epic (or the user) cannot tell which parts are the worker's observation and which are the conductor's interpretation.
The failure this prevents: a worker's matrix-provenance finding, forwarded as if it undercut a sibling issue's premise, when the EPIC had already moved that arm for the same reason — the conductor's reading reached the user mid-decision as though it were the worker's own conclusion.
Append forwarded findings to `.epic-decisions.md` alongside decision relays (see ".epic-decisions.md: the durable fleet-decision log" above), same reasoning: message-only state does not survive compaction.

### `.epic-decisions.md`: the durable fleet-decision log

Every `FINAL` relay (above), every forwarded worker finding (above), and every negative-list entry (Step 5's dashboard) appends here — in the conductor cwd, gitignored the same way as `.epic-status.json` (see the gitignore note near that spec below).
Format is timestamped markdown, append-only, never edit a previous entry — the same convention as `.epic-worklog.md`, because this is attributed narrative for a human (or a fresh epic session) to read, not a machine-parsed event stream like the JSONL `.epic-notifications.log`.

```
## 2026-08-28T14:30:00Z — RELAY:FINAL (conductor→epic)
<the decision, restated in full, not a paraphrase of sentiment>

## 2026-08-28T14:32:00Z — FINDING (worker i2098, forwarded by conductor)
<the worker's finding, verbatim>
Conductor's own reading, if any, marked separately: <...>

## 2026-08-28T09:00:00Z — NEGATIVE (conductor)
<action not taken, and why — e.g. "i2080 not proposed: clone stood down 08-27 for <reason>, prerequisites merging today doesn't reopen it">
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

### Step 4: Fan out the clone/tmux scanner

Dispatch one `general-purpose` subagent — single call, following the dispatch pattern in `SUBAGENT-SCAN.md` (bipartite repo root).
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
> - `available`: (clone mode) no tmux window, on `main`, clean
>
> Return under 400 words:
> - `active_items`: per slot: name, phase, summary, scope, stop_reason, lead_guidance (from `.epic-status.json`), classification
> - `action_candidates`: stale slots ready for cleanup (clean up ONLY if no tmux window — never kill tmux windows); pending spawn-intent files (either naming pattern) with an idle clone to run them
> - `surprises`: phase migrations (`blocked`/`pr-review`), missing status files, contradictions

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

**Negative list**: also surface decisions already taken *against* an action, with reasons — sequencing already applied, an issue already stood down for a reason that would otherwise look resolved, and similar.
This is the one category of fleet fact the epic (or a fresh conductor) cannot re-derive from `git`/`tmux`/`gh`: it's the absence of work, which leaves no trace in any of those.
Read `.epic-decisions.md` in the conductor cwd (see Conventions, "Decision relays" and ".epic-decisions.md: the durable fleet-decision log") for prior entries and append any new one here, timestamped and attributed, in the same step you surface it — don't let it live only in this dashboard render.
Concrete shape from the run that motivated this: an issue whose stated prerequisites both merged the same day reads as unblocked, but the conductor had already stood a clone down for it for an unrelated reason — without the negative list, that reads as ready to the epic and gets proposed again.

### Step 6: Propose next action

First, do housekeeping automatically (no need to ask):
- **Clean up clones whose tmux window the user has already closed**:
  - An **open tmux window** means the user is still using that clone — even if the agent completed and the PR merged.
    The user often does follow-up work (filing next issues, inspecting results, ad-hoc commands).
    **Never kill a tmux window.
    Never clean up a clone that still has a tmux window open.**
  - Only clean up clones with **no tmux window** (the user closed it):
    - *Clone mode*: `git checkout main && git pull --ff-only`, clear `.epic-status.json`
    - *Worktree mode*: `git worktree remove --force $CLONE_ROOT/issue-N && git branch -d <branch>`

Then propose executing pending spawn intent:

> "Pending intent: `i302` (retry logic) and `i315` (scoring refactor), 2 clones available.
> Shall I run `/bip-conductor-spawn`?"

Wait for user confirmation, then run `/bip-conductor-spawn` (do NOT improvise tmux/claude commands).
Deciding *which other* open issues should be spawned next isn't this skill's call — that's `/bip-epic`.

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
To also receive events as Claude Code notifications when that pipeline is reliable, additionally start a Monitor with `command: tail -F .epic-notifications.log` and `persistent: true`.
The notifications log is the contract; Monitor is a latency optimization, not a correctness requirement.

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

- Must be `.gitignored` (along with `.epic-worklog.md` and `.epic-decisions.md` — see Conventions, "Decision relays" and ".epic-decisions.md: the durable fleet-decision log")
- Stale after 30 minutes with no tmux window
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
