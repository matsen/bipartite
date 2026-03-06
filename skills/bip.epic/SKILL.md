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
It orchestrates: scans, updates EPICs, spawns clones, reviews PRs.

## Configuration

The epic skill reads `.epic-config.json` from the repo root. This file
is gitignored and must exist before the skill can operate.

```json
{
  "clone_root": "~/re/myproject",
  "clone_names": ["alpha", "beta", "gamma"],
  "new_clone_names": ["delta", "epsilon", "zeta"],
  "github_repo": "org/repo",
  "conductor": "alpha"
}
```

Fields:
- **clone_root**: Parent directory containing all clones
- **clone_names**: Existing clone directory names
- **new_clone_names**: Names available for creating new clones
- **github_repo**: `org/repo` for `gh` commands
- **conductor**: Which clone is the orchestrator (stays on main)

## Workflow

### Step 0: Load config

```bash
cat .epic-config.json
```

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

### Step 4: Build dashboard

Three sections:

**EPIC Progress**: Per-EPIC summary of done/active/next, which clones
are working on what.

**Clone Status**: Table with clone, branch, last commit, status.

**Actionable Next Steps**: Cross-reference EPIC active items with clone
status. Concrete suggestions.

### Step 5: Offer options

Use `AskUserQuestion` with 3-4 dynamically generated options:
- Start work on iN in clone X
- Resume work on clone X
- Review/land PR pN
- Clean up stale clone X

### Step 6: Act on selection

- **Spawn work**: Run the `/bip.epic.spawn` skill (do NOT improvise tmux/claude commands)
- **Review PR**: Read PR body, check CI, summarize for user
- **Update EPICs**: Use the EPIC body update pattern (below)
- **Land PR**: Use `/land` skill

## EPIC body update pattern

EPIC issue bodies are the source of truth for project status. Update
them when findings come in, items complete, or new work starts.

**Local file convention**: Keep a persistent local copy as
`ISSUE-EPIC-<short-desc>.md` in the repo root (e.g.
`ISSUE-EPIC-indel-signals.md`, `ISSUE-EPIC-benchmark.md`).
These files are gitignored via the `ISSUE-*.md` pattern.

```bash
# Pull current body and record the timestamp
gh issue view <number> --json body,updatedAt > /tmp/epic-pull.json
jq -r .body /tmp/epic-pull.json > ISSUE-EPIC-<short-desc>.md
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
  gh issue edit <number> --body-file ISSUE-EPIC-<short-desc>.md
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
  "phase": "exploring | coding | testing | blocked | completed",
  "summary": "Human-readable one-liner",
  "updated_at": "2026-03-03T14:30:00Z",
  "blockers": [],
  "remote_run": null
}
```

- Must be `.gitignored`
- Stale after 30 minutes with no tmux window
- `remote_run` optional — set when work dispatched to remote server

## Error handling

- **Not in tmux**: Warn — tmux required for spawning
- **No EPIC issues found**: Report and offer to create one
- **gh not authenticated**: Suggest `gh auth login`
