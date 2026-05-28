---
name: bip-epic
description: EPIC cold-start dashboard — full scan of clones, GitHub, and EPIC issues
---

# /bip-epic

Full cold-start dashboard for EPIC-based multi-clone orchestration.
Run from the **conductor clone** inside tmux.

Use this at **session start** to establish context. For mid-session
updates, use `/bip-epic-poll`. To spawn work, use `/bip-epic-spawn`.

## Conventions

### Issue/PR naming
- `i281` = issue #281, `p275` = PR #275. Never bare `#N`.
- First mention in bullet lists: full URL inline.

### Tmux windows
- Named `NNN-YYY` where NNN is the issue number and YYY is the clone/slot name
- *Clone mode*: e.g. `281-cedar`, `295-pine`
- *Worktree mode*: e.g. `281-issue-281`, `295-issue-295`

### Conductor role
The conductor session stays on `main` and does NOT do feature work.
It orchestrates: scans, updates EPICs, spawns clones.

### Reboots: parking and recovery
For a **planned** reboot, run `/bip-epic-prepare-reboot` first (host-wide, while
tmux is alive): it resolves each Claude window's exact session id, optionally
checkpoints workers, and writes a manifest so the workspace returns
deterministically. For an **unplanned** reboot (or if no manifest was written),
use `/bip-epic-recover` from a project's main clone to find the killed Claude
sessions and resume each into a tmux window (`claude --resume`). Recover replays
the manifest when one exists and otherwise reads each session's own jsonl, so
workers that returned to `main` and concurrent main-clone sessions are all
recoverable.

**Numbered issues → spawn**: If work is tied to a GitHub issue (`iN`),
always use `/bip-epic-spawn` to assign it to a clone — even if the fix
seems trivial. The conductor can do light triage (reading files, checking
CI output, running `gh` commands) but should not write code or create
branches for numbered issues.

## Configuration

The epic skill reads `.epic-config.json` from the repo root. This file
is gitignored and must exist before the skill can operate.

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

**Validation**: If `local_worktrees: true` and `clone_names` are both present, **stop and report an error** — they are mutually exclusive. `clone_names` is meaningless in worktree mode because slots are created on demand and named after the issue.

Fields:
- **clone_root**: Parent directory containing all clones or worktrees
- **clone_names**: (clone mode only) Existing clone directory names
- **new_clone_names**: (clone mode only) Names available for creating new clones
- **local_worktrees**: (worktree mode) If `true`, use `git worktree` for local slots named `issue-N`
- **github_repo**: `org/repo` for `gh` commands
- **conductor**: (clone mode only) Which clone is the orchestrator (stays on main)
- **max_lead_iterations**: Max issue-lead evaluations before escalating to `needs-human` (default: 8)
- **shared_filesystem**: (optional, default `false`) Set to `true` when the conductor and all compute nodes share an NFS filesystem; the conductor composes direct SSH execution commands instead of `make remote-sync` calls, and experiment results are immediately visible on local NFS paths. Each machine sets this flag for itself — no central list of NFS nodes is needed.

## Workflow

### Step 1: Load config and memory

```bash
cat .epic-config.json
```

Also read MEMORY.md from the auto-memory directory for orchestrator
context from previous sessions (decisions, patterns, what's next).

**If the file does not exist**, stop and ask the user:
1. Are you using local git worktrees or separate clones for parallel work?
2. Where should slots live? (e.g. `~/re/pz-workers` for worktrees, or `~/re/pz` for clones)
3. (Clone mode only) What are the clone directory names? Which is the conductor?
4. What is the GitHub repo (`org/repo`)?
5. Are compute nodes on a shared NFS filesystem? (sets `shared_filesystem`)

**Note (worktree mode)**: The skill is run from the main repo itself, which
acts as the conductor. There is no separate conductor clone — `clone_root`
is just where worktrees are placed, not the main checkout.

Then create `.epic-config.json` with their answers and proceed.

All subsequent steps use values from this config — never hardcode
paths or clone names.

### Step 2: Pull main

```bash
git pull --ff-only origin main
```

If this fails, report the problem and continue with stale state.

### Step 3: Fan out scanners

First the cheap structured listing the primary can do directly:

```bash
gh issue list --search "EPIC in:title" --json number,title
```

Then dispatch three groups of `general-purpose` subagents in parallel
— single message, multiple `Agent` tool calls. Follow the dispatch
pattern in `SUBAGENT-SCAN.md` (bipartite repo root).

**Group A: one subagent per EPIC.** Brief:

> Read EPIC `i<N>` and report its current state. Tasks:
> 1. `gh issue view <N> --json title,body,updatedAt`.
> 2. Parse the Status dashboard, Key findings, and active clone
>    assignments table.
> 3. For each open item with a clone assignment, run `gh issue view
>    <child-N> --json state,stateReason` to confirm it is still open.
>
> Return under 400 words:
> - `changes_since_baseline`: completed items, new findings, items
>   newly opened
> - `active_items`: open work with clone assignments and brief status
> - `action_candidates`: items the EPIC marks ready but unassigned
> - `surprises`: contradictions, stale assignments, `RECOMMEND DEEPER
>   LOOK` flags

**Group B: one PR/issue triage subagent.** Brief:

> Triage the backlog for the conductor. Tasks:
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

**Group C: one slot-collector subagent.** Brief:

> Inventory clones/worktrees. Read `clone_root` and `local_worktrees`
> from `.epic-config.json`.
>
> Clone mode (`local_worktrees` absent or false): iterate
> `clone_names`; for each, capture branch, last commit, dirty files
> (max 5), and `.epic-status.json` contents.
>
> Worktree mode (`local_worktrees: true`): `find $CLONE_ROOT
> -maxdepth 1 -name 'issue-*' -type d`; for each, capture last
> commit, dirty files, and `.epic-status.json`.
>
> Also: `tmux list-windows -F "#W"`.
>
> Classify each slot:
> - `occupied`: has tmux window (regardless of agent status — user
>   may be doing follow-up work)
> - `stale`: no tmux window, but has `.epic-status.json` or is on
>   non-main branch
> - `available`: (clone mode) no tmux window, on `main`, clean
>
> Return under 400 words:
> - `active_items`: per slot: name, phase, summary, scope, stop_reason,
>   lead_guidance (from `.epic-status.json`), classification
> - `action_candidates`: stale slots ready for cleanup (clean up
>   ONLY if no tmux window — never kill tmux windows)
> - `surprises`: phase migrations (`blocked`/`pr-review`), missing
>   status files, contradictions

**Never ask the user a question about an issue/PR status that you
could answer with a `gh` query** — verify first, then present facts.

### Step 4: Reconcile issues with clones

After the Step 3 fan-out, the primary holds three structured reports.
Compose the reconciliation from them — do not paste subagent prose
verbatim. The primary sees all three reports; cross-reference them
yourself.

Reconcile across the three reports (Groups A, B, C):
- If an issue is **CLOSED on GitHub** but a clone is still assigned →
  if the clone has no tmux window, clean it up; if it has a tmux window,
  leave it alone (user may be doing follow-up work)
- If a PR is **MERGED** but the clone hasn't been reset → same rule
- Never present an issue as "ready to spawn" or "needs action" without
  confirming it's still OPEN on GitHub
- Flag anything merged/closed that the EPIC doesn't reflect yet
- If any group's report has zero `surprises` and zero
  `changes_since_baseline`, send a follow-up to that subagent with a
  narrower question before concluding "nothing changed there."

### Step 5: Build status display

The dashboard is **issue-centric**. The user cares about what work
needs attention and what's ready to start — clones are parenthetical.

**Recently merged** (last 48h): p705, p704, p703, p702, p647, p710, p711

**Section 1: Active issues** — every open issue that has work in
progress, sorted by status (active → awaiting → needs-human → stale):

| Issue | Status | Clone | Summary |
|-------|--------|-------|---------|
| i281 | active (tmux `281-cedar`) | cedar | Implementing clamping |
| i295 | awaiting (~Tue) | pine | 436/1800 ML jobs on orca02 |
| i310 | needs-human | fir | Architectural decision needed |
| i589 | stale (4d) | cedar | Check if experiment finished |

**Section 2: Ready to spawn** — open issues not assigned to any clone,
not blocked, not dependent on in-flight work, ordered by priority.
Check each candidate's `depends_on` field and any blocking context
before listing. If an issue depends on an unmerged PR or unfinished
experiment, it is NOT ready — omit it silently.

- `i302` — Add retry logic to batch pipeline
- `i315` — Refactor scoring module

*(N clones available)*

This two-section layout is the primary loop: what's running, what's
next. Keep it tight — the user should be able to scan in 10 seconds.

### Step 6: Propose next action

First, do housekeeping automatically (no need to ask):
- **Update EPIC bodies** if anything merged/closed since last update
- **Clean up clones whose tmux window the user has already closed**:
  - An **open tmux window** means the user is still using that clone —
    even if the agent completed and the PR merged. The user often does
    follow-up work (filing next issues, inspecting results, ad-hoc
    commands). **Never kill a tmux window. Never clean up a clone that
    still has a tmux window open.**
  - Only clean up clones with **no tmux window** (the user closed it):
    - *Clone mode*: `git checkout main && git pull --ff-only`, clear `.epic-status.json`
    - *Worktree mode*: `git worktree remove --force $CLONE_ROOT/issue-N && git branch -d <branch>`

Then propose spawning work for ready issues:

> "Ready to spawn: `i302` (retry logic) and `i315` (scoring refactor). 2 clones available. Shall I spawn them?"

Wait for user confirmation, then run `/bip-epic-spawn` (do NOT improvise tmux/claude commands).

### Step 7: Start slot monitor

After the dashboard is built and any spawns are launched, offer to start
the **persistent slot monitor** — `bip epic watch` — which observes
every slot's `.epic-status.json` and writes phase-transition events to
`.epic-notifications.log` (JSONL) in the conductor cwd. The log is the
canonical record; transitions survive watcher restarts and conductor
compaction. This replaces `/loop 10m /bip-epic-poll` for the most
time-sensitive signals (phase transitions), while `/bip-epic-poll`
remains available for full GitHub + slot reconciliation sweeps.

Start the watcher in the background:

```bash
nohup bip epic watch >/dev/null 2>&1 &
```

The watcher runs forever, exits cleanly on SIGTERM, and emits one event
per real phase transition (default filter: `needs-human`, `completed`,
`awaiting-results`, `quality-gate`). To also receive events as Claude
Code notifications when that pipeline is reliable, additionally start a
Monitor with `command: tail -F .epic-notifications.log` and
`persistent: true`. The notifications log is the contract; Monitor is a
latency optimization, not a correctness requirement.

When a transition arrives showing `needs-human` or `completed`, the
conductor should react immediately — read the slot's status, check the
lead guidance, and either propose the next action or flag it for the user.

> "Slot monitor started — phase transitions are streaming to
> `.epic-notifications.log`. Use `/bip-epic-poll` for a full
> reconciliation sweep when needed."

On NFS-mounted clone roots where inotify does not fire on remote writes,
pass `--poll` (defaults to a 2 s stat-loop) instead of fsnotify:

```bash
nohup bip epic watch --poll >/dev/null 2>&1 &
```

## EPIC body update pattern

EPIC issue bodies are the source of truth for project status. Update
them when findings come in, items complete, or new work starts.

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

# Edit the file (add findings, check boxes, update clone table)
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
- **Status dashboard**: Check/uncheck boxes, add new items
- **Key findings**: Numbered list, append new findings
- **Related experiments**: Add new experiment rows
- **Active clone assignments**: Update date and clone table

Always include the date in the clone assignments header.

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

- Must be `.gitignored` (along with `.epic-worklog.md`)
- Stale after 30 minutes with no tmux window
- `remote_run` optional — set when work dispatched to remote server
- `quality` optional — set during `quality-gate` phase:
  ```json
  {"pr_check": "pass|fail", "pr_review": "pass|fail", "iterations": 2}
  ```
  Workers loop `/bip-pr-check` and `/bip-pr-review` until both pass clean.
  The orchestrator can monitor progress via this field during polling.
- `scope` — set by the issue lead each iteration (one-line restatement of the issue goal)
- `stop_reason` — categorized reason from the lead's decision framework
- `lead_guidance` — actionable instruction for the worker's next iteration
- `lead_notes` — append-only log of lead evaluations (max 8 before escalation)
- `completed_at` — ISO 8601 timestamp set by the lead after it
  finishes the terminal `completed` ceremony (files any legitimate
  follow-ups, posts the final PR comment). Its presence is the
  idempotency signal: subsequent lead invocations at `completed` skip
  the ceremony.
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
- **No EPIC issues found**: Report and offer to create one
- **gh not authenticated**: Suggest `gh auth login`

## Layout config (issue #149)

`.epic-config.json`'s `clone_root` / `clone_names` / `local_worktrees`
keep working untouched. The newer way to configure worktree mode (for
non-EPIC `bip spawn` use) is the `layout:` block in
`~/.config/bip/config.yml` — see `docs/guides/layout.md`. EPIC
orchestration still reads `.epic-config.json` for now.
