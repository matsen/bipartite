---
name: bip.epic
description: EPIC cold-start dashboard — full scan of clones, GitHub, and EPIC issues
---

# /bip.epic

Full cold-start dashboard for EPIC-based multi-clone orchestration.
Run from the **conductor clone** inside tmux.

Use this at **session start** to establish context. For mid-session
updates, use `/bip.epic.poll`. To spawn work, use `/bip.epic.spawn`.

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

**Numbered issues → spawn**: If work is tied to a GitHub issue (`iN`),
always use `/bip.epic.spawn` to assign it to a clone — even if the fix
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

### Step 3: Gather issue and PR state from GitHub

Start with the question: **what issues need work?**

```bash
# EPIC issues (project-level tracking)
gh issue list --search "EPIC in:title" --json number,title,body

# All open issues (the work backlog)
gh issue list --search "sort:updated-desc" --limit 20 --json number,title,state,labels

# Open PRs (in-flight work nearing completion)
gh pr list --json number,title,headRefName,state

# Recently merged PRs (what landed since last check)
gh pr list --search "is:pr is:merged sort:updated-desc" --limit 10 --json number,title,mergedAt
```

Parse the **Status dashboard** section from each EPIC body to extract
completed, active, and blocked items.

For every open issue and PR, verify its current state:
```bash
gh issue view <N> --json state,stateReason -q '.state + " " + (.stateReason // "")'
gh pr view <N> --json state,mergedAt -q '.state'
```

**Never ask the user a question about an issue/PR status that you
could answer with a `gh` query** — poll first, then present facts.

### Step 4: Check clone capacity

Scan clones/worktrees to answer "where can work run?"

Check tmux windows: `tmux list-windows -F "#W"`

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

**Clone mode** (`local_worktrees` absent or false): iterate `clone_names`:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
for name in $(jq -r '.clone_names[]' .epic-config.json); do
  git -C "$CLONE_ROOT/$name" branch --show-current
  git -C "$CLONE_ROOT/$name" log --oneline -1
  git -C "$CLONE_ROOT/$name" status --porcelain | head -5
  cat "$CLONE_ROOT/$name/.epic-status.json" 2>/dev/null
done
```

**Worktree mode** (`local_worktrees: true`): discover via `git worktree list`:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
find "$CLONE_ROOT" -maxdepth 1 -name 'issue-*' -type d | while read wt; do
  git -C "$wt" log --oneline -1 2>/dev/null
  git -C "$wt" status --porcelain 2>/dev/null | head -5
  cat "$wt/.epic-status.json" 2>/dev/null
done
```

For each clone/worktree, note which issue (if any) it's working on. Classify:
- **occupied**: Has tmux window (regardless of agent status — user may be doing follow-up work)
- **stale**: No tmux window, but has `.epic-status.json` or is on a non-main branch
- **available**: (clone mode) No tmux window, on `main`, clean worktree — (worktree mode) N/A, new worktrees are created on demand

**Key rule**: An open tmux window means the clone is in use. The user
closes the window when they are done. Never kill tmux windows.

### Step 5: Reconcile issues with clones

Cross-reference the two datasets:
- If an issue is **CLOSED on GitHub** but a clone is still assigned →
  if the clone has no tmux window, clean it up; if it has a tmux window,
  leave it alone (user may be doing follow-up work)
- If a PR is **MERGED** but the clone hasn't been reset → same rule
- Never present an issue as "ready to spawn" or "needs action" without
  confirming it's still OPEN on GitHub
- Flag anything merged/closed that the EPIC doesn't reflect yet

### Step 6: Build status display

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

### Step 7: Propose next action

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

Wait for user confirmation, then run `/bip.epic.spawn` (do NOT improvise tmux/claude commands).

### Step 8: Start slot monitor

After the dashboard is built and any spawns are launched, offer to start
a **persistent Monitor** that watches `.epic-status.json` across all
active slots. This replaces `/loop 10m /bip.epic.poll` for the most
time-sensitive signals (phase transitions), while `/bip.epic.poll`
remains available for full GitHub + slot reconciliation sweeps.

Use the Monitor tool with `persistent: true`:

```
description: "EPIC slot phase changes"
persistent: true
command: |
  CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
  LOCAL_WT=$(jq -r '.local_worktrees // false' .epic-config.json)
  touch /tmp/.epic-monitor-baseline

  if [ "$LOCAL_WT" = "true" ]; then
    WATCH_DIR="$CLONE_ROOT"
  else
    WATCH_DIR="$CLONE_ROOT"
  fi

  fswatch --batch-marker=EOF -r "$WATCH_DIR" --include '.epic-status.json' --exclude '.*' | \
    while read line; do
      if [ "$line" = "EOF" ]; then
        find "$WATCH_DIR" -name '.epic-status.json' -newer /tmp/.epic-monitor-baseline \
          -exec sh -c '
            SLOT=$(basename "$(dirname "$1")")
            PHASE=$(jq -r .phase "$1" 2>/dev/null)
            SUMMARY=$(jq -r .summary "$1" 2>/dev/null)
            ISSUE=$(jq -r .issue "$1" 2>/dev/null)
            echo "$SLOT (i$ISSUE): $PHASE — $SUMMARY"
          ' _ {} \;
        touch /tmp/.epic-monitor-baseline
      fi
    done
```

When a notification arrives showing `needs-human` or `completed`, the
conductor should react immediately — read the slot's status, check the
lead guidance, and either propose the next action or flag it for the user.

> "Slot monitor started — you'll see phase changes as they happen.
> Use `/bip.epic.poll` for a full reconciliation sweep when needed."

**Prerequisites**: Requires `fswatch` (`brew install fswatch` on macOS).
If not available, fall back to `/loop 10m /bip.epic.poll`.

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
  Workers loop `/bip.pr.check` and `/bip.pr.review` until both pass clean.
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
