---
name: bip.epic.spawn
description: Spawn a Claude session in a clone for an EPIC issue
---

# /bip.epic.spawn

Spawn a Claude Code session in a tmux window to work on a GitHub issue.
The worker runs inside a **ralph-loop** with an **issue-lead subagent**
that evaluates progress at stopping points.

## Usage

```
/bip.epic.spawn <issue-number> [clone-name]
```

If clone-name is omitted, pick the best idle clone automatically.

## Configuration

Reads `.epic-config.json` from the repo root (see `/bip.epic` for format).
**If the file does not exist**, stop and ask the user to configure it
via `/bip.epic` first.

## Workflow

### Step 1: Select clone

Read `clone_root` and `clone_names` from `.epic-config.json`.

If clone-name not specified, find an idle clone:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
for name in $(jq -r '.clone_names[]' .epic-config.json); do
  branch=$(git -C "$CLONE_ROOT/$name" branch --show-current 2>/dev/null)
  [ "$branch" = "main" ] && echo "$name"
done
```

Prefer clones with clean worktrees. If all busy, offer to create a new
clone using a name from `new_clone_names` in the config.

### Step 2: Update clone and clean stale state

```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
cd "$CLONE_ROOT/<clone>"
git checkout main && git pull --ff-only origin main
rm -f .epic-status.json .epic-worklog.md
```

**State cleanup is mandatory** — stale files from a previous assignment
will confuse the worker and lead.

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
/ralph-loop --completion-promise "ISSUE WORK COMPLETE" \
You are working on GitHub issue #<N> "<title>".

EPIC STATUS PROTOCOL — You MUST follow this:
1. At session start, write .epic-status.json (see format below)
2. Update it when you transition between phases
3. Update it when you finish or encounter a blocker
4. Maintain .epic-worklog.md as a narrative log (see format below)

.epic-status.json format:
{
  "issue": <N>,
  "title": "<title>",
  "phase": "exploring",
  "summary": "Reading issue and exploring codebase",
  "updated_at": "<ISO 8601 timestamp>",
  "blockers": [],
  "remote_run": null,
  "quality": null,
  "scope": null,
  "stop_reason": null,
  "lead_guidance": null,
  "lead_notes": [],
  "awaiting": null
}

Phases: exploring, coding, testing, awaiting-results, quality-gate, needs-human, completed

.epic-worklog.md format (append-only, never edit previous entries):
## <ISO 8601 timestamp> — <Phase>

Brief description of what you did and why (3-5 sentences).

RECOVERING CONTEXT (after compaction):
1. Read .epic-status.json — current phase and lead guidance
2. Read .epic-worklog.md — narrative of what happened
3. If lead_guidance is set → follow it
4. If lead_guidance is empty → read the last worklog entry and continue
5. If both are empty → read the issue and begin fresh

BRANCH: Create branch <N>-<short-name> from main.
AUTONOMY: Do the work. Do not ask the user whether to proceed with
implementation steps, run experiments, or set up tests — just do them.

WORKLOG: Append entries to .epic-worklog.md when:
- Starting work or reading the issue
- Changing approach or strategy
- Hitting a blocker
- Completing a phase
- Receiving lead guidance (copy it to the worklog)

AWAITING RESULTS:
If you launch a long-running experiment:
1. Set phase to "awaiting-results" in .epic-status.json
2. Set the "awaiting" field with check_cmd and check_files
3. Each ralph-loop iteration: run check_cmd, if not ready end the turn
4. After 3 consecutive check failures, set stop_reason to
   "mechanical-blocker" and invoke the lead

STOPPING POINTS — When you reach a natural stopping point:
1. Append a worklog entry describing what you did and why you stopped
2. Update .epic-status.json with phase, summary, stop_reason
3. Spawn the issue-lead subagent for evaluation:

   Use the Agent tool with subagent_type "issue-lead" and prompt:
   "Evaluate progress on issue #<N> in this clone. Follow your
    full evaluation protocol: read .epic-status.json,
    .epic-worklog.md, the issue body, commits, PR, and any
    experiment results. Write your assessment and guidance."

4. Read the lead's response:
   - If it says PHASE: completed or PHASE: needs-human →
     output <promise>ISSUE WORK COMPLETE</promise>
   - Otherwise → copy the lead's guidance to .epic-worklog.md
     as a "Lead guidance" entry, then continue working

COMPLETION: When done (or when lead says completed):
1. Commit all work and push the branch
2. Create a PR: gh pr create --title "<title>" --body "Closes #<N>"
3. Update .epic-status.json phase to "quality-gate"
4. QUALITY GATE LOOP — repeat until both pass clean:
   a. Run /pr-check — fix everything it flags, commit and push
   b. Run /pr-review — fix ALL issues (even minor/advisory), commit and push
   c. If either flagged issues, go back to (a)
   Track iterations in .epic-status.json:
   {"phase":"quality-gate","quality":{"pr_check":"pass|fail","pr_review":"pass|fail","iterations":N}}
5. When both pass with zero issues:
   - Invoke the issue-lead one final time (it will set phase to "completed")
   - Output <promise>ISSUE WORK COMPLETE</promise>
6. STOP only if a finding requires genuine user judgment (design
   questions, ambiguous requirements, architectural tradeoffs).
   For everything else — formatting, test gaps, docs, naming,
   lint, cruft — just fix it and move on.

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
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
bip spawn --prompt "<composed prompt>" \
  --dir "$CLONE_ROOT/<clone-name>" \
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
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
REPO=$(jq -r .github_repo .epic-config.json)
cd "$CLONE_ROOT"
git clone "git@github.com:$REPO.git" <new-name>
```

After creating, add the new name to `clone_names` in `.epic-config.json`.

## Cleaning up before reuse

If a clone is on a non-main branch:
1. Check if there's an open PR: `gh pr list --head <branch>`
2. If merged/closed: `git checkout main && git pull --ff-only`
3. If open: warn user — they may want to resume

## Gitignore reminder

Target project repos should gitignore these files (add to `.gitignore`):
```
.epic-status.json
.epic-worklog.md
```

These files live in clones, not in bipartite itself.

## Conventions

Same as `/bip.epic`: `iN`/`pN` prefixes, clone-name tmux windows.
