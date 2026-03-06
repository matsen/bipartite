---
name: bip.epic.spawn
description: Spawn a Claude session in a clone for an EPIC issue
---

# /bip.epic.spawn

Spawn a Claude Code session in a tmux window to work on a GitHub issue.

## Usage

```
/bip.epic.spawn <issue-number> [clone-name]
```

If clone-name is omitted, pick the best idle clone automatically.

## Workflow

### Step 1: Select clone

If not specified, find an idle clone:
```bash
cd ~/re/pz
for d in */; do
  branch=$(git -C "$d" branch --show-current 2>/dev/null)
  [ "$branch" = "main" ] && echo "$d"
done
```

Prefer clones with clean worktrees. If all busy, offer to create a new
clone (wood-themed names: walnut, cherry, willow, juniper, etc.).

### Step 2: Update clone to latest main

```bash
cd ~/re/pz/<clone>
git checkout main && git pull --ff-only origin main
```

### Step 3: Read the issue

```bash
gh issue view <number> --json title,body
```

Extract key context: what the issue asks for, data locations, phasing,
dependencies.

### Step 4: Compose the prompt

Use the template below. Customize the IMPORTANT CONTEXT section based
on the issue — this is where the conductor adds value over a generic
spawn.

```
You are working on GitHub issue #<N> "<title>" in the phyz project.

EPIC STATUS PROTOCOL — You MUST follow this:
1. At session start, write .epic-status.json (see format below)
2. Update it when you transition between phases
3. Update it when you finish or encounter a blocker

.epic-status.json format:
{
  "issue": <N>,
  "title": "<title>",
  "phase": "exploring",
  "summary": "Reading issue and exploring codebase",
  "updated_at": "<ISO 8601 timestamp>",
  "blockers": []
}

Phases: exploring, coding, testing, blocked, completed

BRANCH: Create branch <N>-<short-name> from main.
COMPLETION: When done, update .epic-status.json phase to "completed",
commit your work, and stop. Do NOT push or create a PR — the
orchestrator session will review first.

IMPORTANT CONTEXT:
<Add issue-specific context here:>
<- Data locations (e.g. SSF143587 path, vialle benchmark path)>
<- Phasing instructions (e.g. "start with Phase 1 only")>
<- Remote execution notes (e.g. "use make remote-sync REMOTE_HOST=...")>
<- Dependencies or blockers>
<- Key files to read first>

Now read the issue and begin work:
/work-issue <N>
```

### Common context additions

**For experiments (Snakemake workflows):**
```
- Use make remote-sync + make remote-tmux for running on remote servers
- Use /bip.scout to find an available server before remote operations
- Always rebuild after sync: make remote-tmux REMOTE_HOST=... CMD='zig build -Doptimize=ReleaseFast'
- SSF143587 data is at ~/re/superfamily-pcp/results/SSF143587/
- Wrap the experiment in a Snakemake workflow
```

**For code changes:**
```
- Run zig build test before committing
- Run make parity if touching shared alignment code
- Check PRE-MERGE-CHECKLIST.md
```

**For phased work:**
```
- This issue has multiple phases. Start with Phase 1 only.
- Phase 1: <describe scope and gate criteria>
- Only proceed to Phase 2 if the gate passes.
```

### Step 5: Launch tmux window

Use `bip spawn` with `--dir`, `--name`, and `--prompt` flags. This
handles tmux window creation, temp file management, and launching
Claude Code with `--dangerously-skip-permissions` automatically.

```bash
bip spawn --prompt "<composed prompt>" \
  --dir "$HOME/re/pz/<clone-name>" \
  --name "<clone-name>"
```

**Do NOT** use raw `tmux new-window` / `tmux send-keys` / `claude` commands.
Always go through `bip spawn` which handles the full lifecycle correctly.

### Step 6: Confirm

Report to the user:
- Which clone was spawned
- Which issue it's working on
- Any phasing or gate criteria

## Creating new clones

```bash
cd ~/re/pz
git clone git@github.com:matsengrp/phyz.git <new-name>
```

## Cleaning up before reuse

If a clone is on a non-main branch:
1. Check if there's an open PR: `gh pr list --head <branch>`
2. If merged/closed: `git checkout main && git pull --ff-only`
3. If open: warn user — they may want to resume

## Conventions

Same as `/bip.epic`: `iN`/`pN` prefixes, clone-name tmux windows.
