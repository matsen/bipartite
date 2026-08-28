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

Two things stay in the primary because they're cheap and structured:

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
> 3. `gh issue list --search "sort:created-desc" --limit 5 --json number,title,state,createdAt`.
> 4. For each active slot (per the tmux list and any `.epic-status.json` in `<clone_root>`), check the latest issue-lead comment: `gh api repos/<owner>/<repo>/issues/<N>/comments --jq '.[-1].body'`.
>    Look for the `🤖 **Issue Lead**` prefix.
> 5. Inventory slots from `.epic-config.json`.
>    Clone mode: iterate `clone_names`.
>    Worktree mode: `find <clone_root> -maxdepth 1 -name 'issue-*' -type d`.
>    For each slot, read `.epic-status.json` and surface: phase, summary, scope, stop_reason, lead_guidance.
>    Migrate legacy phases: `blocked → needs-human`, `pr-review → quality-gate`.
> 6. For active slots, `git -C <slot> log --oneline main..HEAD | head -5` to see recent commits.
> 7. For slots that look finished or blocked, capture the last 20 lines of tmux: `tmux capture-pane -t <window> -p | tail -20`.
> 8. Verify state with `gh pr view --json state` / `gh issue view --json state` for anything you plan to flag — never claim "open" or "merged" without a live confirmation.
>
> Return under 400 words, structured per `SUBAGENT-SCAN.md`:
> - `changes_since_baseline`: merged PRs, new issues, new issue-lead comments, slot phase changes
> - `active_items`: per active slot — clone, issue, phase, stop_reason, lead assessment (one line each)
> - `action_candidates`: pending spawn intent directly in `$CLONE_ROOT/.spawn-prompts/` (either `<N>.md` or `spawn-<N>.txt` — check both) waiting to be executed — ignore anything under `.spawn-prompts/consumed/`, that's already-launched intent, not pending; merged PRs that should trigger slot cleanup.
>   (Deciding *which* open issues are ready to spawn is `/bip-epic`'s call, not this poll's — report raw signal, not a readiness verdict.)
> - `surprises`: `needs-human`/`completed` slots, stale status files, contradictions, `RECOMMEND DEEPER LOOK` flags

If the report has zero `changes_since_baseline` and zero `surprises`, the poll output is one line: "All quiet."

## After polling

### Focus on what matters

**Lead with pending spawn intent** — `.spawn-prompts/` files (either naming pattern) the epic has already written and is waiting on the conductor to execute.
This is the most actionable information at this level; deciding which *other* open issues should be spawned next is `/bip-epic`'s call.

**Surface lead evaluations** — if a clone's status shows a recent lead evaluation (stop_reason set, lead_guidance present), mention the lead's assessment briefly.
This tells the conductor what the workers are doing without having to read full issue comments.

**Flag needs-human and completed** — if any clone has `phase: "needs-human"` (or legacy `blocked`) or `phase: "completed"`, highlight it prominently.
These require conductor attention.
Run `ListAgents` for an addressable epic session and, if one exists, `SendMessage` it the issue number and phase so `/bip-epic` can re-poll and update the EPIC body without waiting for its own cadence.
If no epic session is addressable, this is a no-op — `/bip-epic`'s independent `gh` polling remains the fallback path.

**Only report active clones** — clones with a tmux window that are actually doing something.
Don't list completed or idle clones; that's noise.
Completed clones can be mentioned briefly ("fir completed i374") but don't need a table row.

**Mention recent merges** only if they unblock something or change the plan.

**Correcting a live worker mid-flight**: `ListAgents`/`SendMessage` reach other local Claude tmux sessions and can push an immediate correction without waiting for the worker's next lead evaluation or killing its tmux window.
Send the correction directly — stating what changed in at least one line — rather than a bare pointer, which is a no-op for a worker that already reads the status file every step and strips the priority signal.
If a correction is long enough that pointing at a file seems preferable, its home is the spawn prompt or the issue body, not a nudge.
The durable/transient test is the deliverable: any nudge that changes what the worker will produce (scope, target, artifact, gate criterion) MUST also get a durable record, written by the **conductor** appending a timestamped, attributed entry to `.epic-worklog.md` in the same step — not `.epic-status.json`, since that file is the worker's own continuously-rewritten object and a second writer there is unsynchronized and gives no tiebreak against `lead_guidance`.
Facts that leave the deliverable unchanged (host load, a peer's timing, a dependency that just landed) may be message-only.
Delivery lands at the worker's next tool call, not instantly, and `SendMessage` only reaches addressable Claude sessions — run `ListAgents` first to confirm (the address is the session name it reports, not the tmux window name assigned at spawn — sending to the window name can fail with `No agent named '<window>' is reachable.`), and fall back to the file-only correction (a `conductor_guidance` field or a `lead_notes` entry tagged `source: conductor` — never `lead_guidance`, wait for the worker's own loop) when it isn't addressable.
Skip the nudge entirely for a worker in `awaiting-results` with a live `check_cmd`; use `notify_when_idle: true` instead of "tell me when this worker finishes" — main-conversation only, this-machine only, one-shot.

### Output structure

1. **Pending spawn intent**: `.spawn-prompts/` files (either naming pattern) waiting on execution, plus idle clones available to run them.

2. **Active work**: Clones with tmux windows that are mid-task.
   One line each: clone, issue, phase, stop_reason (if set), lead assessment.

3. **Needs human**: Clones in `needs-human` phase — show the lead's assessment and what decision is needed.

4. **Recently landed** (brief): PRs merged since last poll, only if noteworthy.

5. **Execute pending spawns**: If spawn intent files and idle clones both exist, propose running `/bip-conductor-spawn` for them.
   Wait for confirmation.

### Housekeeping (do silently, don't report unless problems)

This is the ongoing cleanup that keeps slots current between cold starts.
Do it every poll cycle — don't wait for `/bip-conductor`.
EPIC body content is not this skill's concern; `/bip-epic` keeps bodies current on its own cadence.

#### Slot cleanup for merged PRs

For each slot whose PR has merged (cross-reference merged PRs from check 1 with slot branches):

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
rm -f "$CLONE_ROOT/<clone>/.epic-status.json" "$CLONE_ROOT/<clone>/.epic-worklog.md"
```

Also clean up stale slots: no tmux window AND `.epic-status.json` older than 30 minutes.
Same cleanup as above.

If a merge closes an issue tracked in an EPIC, that's signal `/bip-epic` needs, not something to act on here — surface it under `changes_since_baseline` and move on.

#### Filter and route fleet-level findings

Before recording anything anywhere, run each candidate fleet-level finding through this filter (same as `/bip-conductor-tuckin` Step 3):

1. **Is it derived?**
   Recomputable from `git`, `tmux`, `gh`, or the filesystem — record nothing.
   Host quirks (which host had a warm cache, which was mid-build) are derived and go stale within hours; don't write them down anywhere, including MEMORY.md.
2. **Is it already recorded?**
   A finding that produced an issue, PR, test, or doc needs no second copy.

Only what survives both gates gets a destination: a durable clone-pool layout decision → a `CLAUDE.md`; a workflow rule → a skill; a topic-level decision → `/bip-epic`'s own memory, not here.
Nothing else fits → the poll report itself.

## Conventions

Same as `/bip-epic`: `iN`/`pN` prefixes, full URLs on first mention.
Tmux windows named `NNN-YYY` where NNN is the issue number and YYY is the clone/slot name (e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).

## Layout config (issue #149)

`.epic-config.json` keeps working.
The newer global `layout:` block in `~/.config/bip/config.yml` configures worktree mode for non-EPIC `bip spawn`; see `docs/guides/layout.md`.
