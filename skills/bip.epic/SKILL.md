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
Named by **clone name** (`cedar`, `oak`, etc.), not issue number.

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

Fields:
- **clone_root**: Parent directory containing all clones
- **clone_names**: Existing clone directory names
- **new_clone_names**: Names available for creating new clones
- **github_repo**: `org/repo` for `gh` commands
- **conductor**: Which clone is the orchestrator (stays on main)
- **max_lead_iterations**: Max issue-lead evaluations before escalating to `needs-human` (default: 8)

## Workflow

### Step 0: Load config and memory

```bash
cat .epic-config.json
```

Also read MEMORY.md from the auto-memory directory for orchestrator
context from previous sessions (decisions, patterns, what's next).

**If the file does not exist**, stop and ask the user:
1. Where are your clones? (e.g. `~/re/pz`)
2. What are the clone directory names?
3. What is the GitHub repo (`org/repo`)?
4. Which clone is the conductor?

Then create `.epic-config.json` with their answers and proceed.

All subsequent steps use values from this config — never hardcode
paths or clone names.

### Step 0.5: Pull main

```bash
git pull --ff-only origin main
```

If this fails, report the problem and continue with stale state.

### Step 1: Discover EPIC issues

```bash
gh issue list --label EPIC --json number,title,body
```

Fallback: `gh issue list --search "EPIC: in:title" --json number,title,body`

Parse the **Status dashboard** section from each EPIC body to extract
completed, active, and blocked items.

### Step 2: Scan clones

Use `clone_root` and `clone_names` from `.epic-config.json`.

For each clone:

```bash
git -C <clone> branch --show-current
git -C <clone> log --oneline -1
git -C <clone> status --porcelain | head -5
cat <clone>/.epic-status.json 2>/dev/null
```

Check tmux windows: `tmux list-windows -F "#W"`

Classify as:
- **active**: Has tmux window or fresh `.epic-status.json` (< 30 min)
- **assigned**: On non-main branch, no active session
- **idle**: On `main`, clean worktree

### Step 3: Check GitHub activity

```bash
# Recently merged PRs
gh pr list --search "is:pr is:merged sort:updated-desc" --limit 10 --json number,title,mergedAt

# Open PRs
gh pr list --json number,title,headRefName,state

# Recent issues
gh issue list --search "sort:updated-desc" --limit 10 --json number,title,state
```

Cross-reference with EPIC bodies — flag anything merged/closed that
the EPIC doesn't reflect yet.

### Step 4: Build status table

Display a single table showing the full picture:

| Clone | Branch | Status | Issue | Summary |
|-------|--------|--------|-------|---------|
| cedar | feat-x | active (tmux) | i281 | Implementing clamping |
| oak   | main   | idle   | —    | — |
| pine  | fix-y  | assigned | i295 | Blocked on upstream |

Then list **unassigned issues** ready for work (not blocked, not in progress):

**Ready issues** (not assigned to any clone):
- `i302` — Add retry logic to batch pipeline
- `i310` — Update benchmark thresholds

### Step 5: Propose next action

First, do housekeeping automatically (no need to ask):
- **Update EPIC bodies** if anything merged/closed since last update
- **Clean up stale clones** (no tmux window, status > 30 min): checkout main, pull, clear `.epic-status.json`

Then propose spawning work for ready issues on idle clones:

> "Shall I spawn `i302` and `i310`? `oak` and `birch` are idle."

Wait for user confirmation, then run `/bip.epic.spawn` (do NOT improvise tmux/claude commands).

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
  Workers loop `/pr-check` and `/pr-review` until both pass clean.
  The orchestrator can monitor progress via this field during polling.
- `scope` — set by the issue lead each iteration (one-line restatement of the issue goal)
- `stop_reason` — categorized reason from the lead's decision framework
- `lead_guidance` — actionable instruction for the worker's next iteration
- `lead_notes` — append-only log of lead evaluations (max 8 before escalation)
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
