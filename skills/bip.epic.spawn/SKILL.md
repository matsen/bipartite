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

### Prerequisite: Issue number required

Every spawn MUST target an existing GitHub issue. If the conductor wants
to spawn work that doesn't have an issue yet (reruns, follow-ups, quick
experiments), file the issue first:

1. Write a minimal issue body (title + 3-sentence motivation + success criteria)
2. `gh issue create --title "..." --body-file ISSUE-*.md`
3. Then proceed with the spawn using the new issue number

Never write a spawn prompt with `issue=0` or without `/bip.issue.work <N>`.
Issueless spawns break EPIC tracking, PR linking, and conductor polling.

### Step 1: Select or create slot

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

**Clone mode** (`local_worktrees` absent or false):

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

**Worktree mode** (`local_worktrees: true`):

Slot name is always `issue-<N>`. Branch name is `<N>-<slug>` where `<slug>`
is the first 4 words of the issue title, lowercased and hyphenated.

Check if slot already exists:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
SLOT="$CLONE_ROOT/issue-<N>"
if [ -d "$SLOT" ]; then
  # Worktree exists — check for active tmux window
  if tmux list-windows -F "#W" | grep -q "^<N>-issue-<N>$"; then
    echo "Active session already running for <N>-issue-<N> — attach to it instead"
    exit 0
  else
    echo "Worktree exists, no active session — will resume"
  fi
else
  # Check for leftover branch from a previous failed attempt
  if git branch --list "<N>-*" | grep -q .; then
    git branch -D $(git branch --list "<N>-*" | tr -d ' ')
  fi
  # Create worktree from the main repo (conductor's working directory)
  git worktree add "$SLOT" -b <N>-<slug>
fi
```

### Step 2: Prepare slot and clean stale state

**Clone mode**:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
cd "$CLONE_ROOT/<clone>"
git checkout main && git pull --ff-only origin main
rm -f .epic-status.json .epic-worklog.md
```

**Worktree mode**: worktree was just created fresh from main — just clear
any stale status files from a previous run on this same issue:
```bash
rm -f "$SLOT/.epic-status.json" "$SLOT/.epic-worklog.md"
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

The prompt has two parts: (1) the work instructions passed as the
initial message to `claude` via `--prompt-file`, and (2) a ralph-loop
invocation that the worker runs as its first action. The ralph-loop
prompt is kept SHORT (no special characters) — just a reminder to
continue. The detailed instructions are already in the conversation
from the initial message.

**Prompt file** (written by conductor to /tmp/spawn-N.txt):
```
You are working on GitHub issue #N TITLE.

First, run this command to start the iteration loop:
/ralph-loop:ralph-loop --completion-promise 'ISSUE WORK COMPLETE' --max-iterations 20 Continue working on the task. Read .epic-status.json and .epic-worklog.md for context. Output ISSUE WORK COMPLETE in promise tags when done.

EPIC STATUS PROTOCOL — You MUST follow this:
1. At session start, write .epic-status.json (see format below)
2. Update it when you transition between phases
3. Update it when you finish or encounter a blocker
4. Maintain .epic-worklog.md as a narrative log (see format below)

.epic-status.json fields:
  issue — the issue number
  title — short title
  phase — one of: exploring, coding, testing, awaiting-results, quality-gate, needs-human, completed
  summary — human-readable one-liner
  updated_at — ISO 8601 timestamp
  blockers — list of blockers (empty list if none)
  scope — one-line restatement of issue goal (set by lead)
  stop_reason — category from lead decision framework (set by lead)
  lead_guidance — what the lead told you to do next (set by lead)
  lead_notes — list of lead evaluation entries (set by lead)
  completed_at — ISO 8601 timestamp set by the lead after the
    terminal completed ceremony (idempotency signal; do not set
    yourself). Record deferred work in the PR body DEFERRED section;
    the lead will file legitimate ones as follow-up issues.
  awaiting — set when waiting for experiment results (description, check_cmd, check_files, started_at, timeout_hours)

.epic-worklog.md format (append-only, never edit previous entries):
Timestamped markdown entries with phase header.
Brief description of what you did and why (3-5 sentences per entry).

RECOVERING CONTEXT (after compaction):
1. Read .epic-status.json — current phase and lead guidance
2. Read .epic-worklog.md — narrative of what happened
3. If lead_guidance is set → follow it
4. If lead_guidance is empty → read the last worklog entry and continue
5. If both are empty → read the issue and begin fresh

BRANCH: Create branch N-short-name from main.
AUTONOMY: Do the work. Do not ask the user whether to proceed with
implementation steps, run experiments, or set up tests — just do them.

EXPERIMENTS ARE MANDATORY: If the issue specifies running an experiment,
benchmark, or analysis, you MUST run it before considering the work done.
Writing code is not enough — the issue is not complete until every
experiment described in it has been executed and results collected.
Do not stop at "code is ready to run" — actually run it.

WORKLOG: Append entries to .epic-worklog.md when:
- Starting work or reading the issue
- Changing approach or strategy
- Hitting a blocker
- Completing a phase
- Receiving lead guidance (copy it to the worklog)

AWAITING RESULTS:
If you launch a long-running experiment:
1. Set phase to awaiting-results in .epic-status.json
2. Set the awaiting field with check_cmd and check_files
3. Each ralph-loop iteration: run check_cmd, if not ready end the turn
4. After 3 consecutive check failures, set stop_reason to
   mechanical-blocker and invoke the lead

PHONE NOTIFICATION — When you set phase to needs-human or completed,
ring the terminal bell and send a push notification so the user notices:
```bash
printf '\a'
NTFY_TOPIC=$(grep ntfy_topic ~/.config/bip/config.yml | awk '{print $2}')
[ -n "$NTFY_TOPIC" ] && curl -s -H "Title: bip epic" -d "#N <phase>: <one-line summary>" "ntfy.sh/$NTFY_TOPIC" > /dev/null
```
Do this EVERY time you write needs-human or completed to .epic-status.json.

STOPPING POINTS — When you reach a natural stopping point:
1. Append a worklog entry describing what you did and why you stopped
2. Update .epic-status.json with phase, summary, stop_reason
3. Spawn the issue-lead subagent for evaluation:

   Use the Agent tool with subagent_type issue-lead and prompt:
   Evaluate progress on issue #N in this clone. Follow your
   full evaluation protocol: read .epic-status.json,
   .epic-worklog.md, the issue body, commits, PR, and any
   experiment results. Write your assessment and guidance.

4. Read the lead response:
   - If it says PHASE: completed or PHASE: needs-human →
     output the completion promise ISSUE WORK COMPLETE
   - Otherwise → copy the lead guidance to .epic-worklog.md
     as a Lead guidance entry, then continue working

COMPLETION: When done (or when lead says completed):
1. Commit all work and push the branch
2. Create a PR with gh pr create, title matches issue, body says Closes #N
3. Update .epic-status.json phase to quality-gate
4. QUALITY GATE LOOP — repeat until both pass clean:
   a. Run /bip.pr.check — fix everything it flags, commit and push
   b. Run /bip.pr.review — triage each finding (see REVIEW TRIAGE below),
      fix the ones you'll address, commit and push
   c. If either flagged issues that you fixed, go back to (a)
   Track quality gate iterations in .epic-status.json
5. When both pass clean (or remaining findings are all deferred):
   - Invoke the issue-lead one final time (it will set phase to completed)
   - Print a FINAL RECAP (see below)
   - Output the completion promise ISSUE WORK COMPLETE
6. STOP only if a finding requires genuine user judgment (design
   questions, ambiguous requirements, architectural tradeoffs).
   For everything else — formatting, test gaps, docs, naming,
   lint, cruft — just fix it and move on.

DEFERRAL RULE — applies to ALL worker decisions, not just review findings.

Default: fold into this PR. The user prefers larger PRs that mix concerns
a little over narrow PRs that generate a trail of follow-up issues. Only
defer when ALL of the following hold:
  1. The work is NOT requested or implied by the current issue body.
  2. EITHER the issue body explicitly flags this work as a design decision
     for the user, OR the issue-lead has previously told you (in
     lead_guidance) that this specific work is out of scope.
  3. The work would more than double the PR diff AND requires distinct
     expertise, new infrastructure, multi-day experiments, or touches a
     clearly unrelated module. Size alone is not enough — a 400-line
     addition to files you're already editing should still be folded in.

If all three are not true, do the work in this session. Specifically:
  - "It would take a few minutes" → do it.
  - "It would take an hour but it's a clear win" → do it.
  - "I noticed a related test gap while editing" → do it.
  - "There's an unhandled edge case" → do it.
  - "The fix is mechanical but tedious" → do it.
  - "It touches the same files I'm already modifying" → do it.
  - "It would grow the PR by 50% but it's coherent with the current work" → do it.

When in doubt, DO NOT defer. Bring it to the issue-lead with your reasoning.
The cost of an over-large PR is small (split it later if needed). The cost
of an under-finished PR is high (follow-up churn, broken windows, work
re-loaded into context cold weeks later, user prompts to merge what
should have been one coherent change).

REVIEW TRIAGE — For each /bip.pr.review finding, apply the DEFERRAL RULE above:
  • FIX NOW (default) — sensible improvements you can complete
    (naming, docs, small refactors, test gaps, lint, edge cases,
    mechanical-but-tedious fixes). Just do them.
  • DEFER — only when all three DEFERRAL RULE conditions hold. For each
    deferred finding, add a line to the DEFERRED section of the PR body:
      > **Deferred**: <one-line description> — <why all three conditions hold>
    These become fodder for follow-up issues.

FINAL RECAP — Print this summary just before outputting the completion
promise so the conductor (and user) can see the full story at a glance.
By the time this runs, the final lead invocation has set phase to
`completed` and posted a PR comment listing any follow-ups it filed.

```
═══ COMPLETED: #N — TITLE ═══
PR: <full PR URL>

Summary:
  <2-5 sentence narrative — what changed and key decisions>

Pivots / surprises:
  <anything that deviated from the original plan, or "none">

Human-judgment items:
  <from your Step 8 summary — architectural tradeoffs, root-cause
   suspicions, perf findings — things the user should weigh in on
   that aren't simple follow-ups. Omit if none.>

Quality gate: passed
═══════════════════════════════
```

The lead's PR comment is the source of truth for filed follow-ups;
the recap doesn't duplicate it. Get the PR URL from
`gh pr view --json url -q .url`. This recap MUST appear in the
worker's output — it is the primary artifact the conductor reads
after the session ends.

**Do NOT invent follow-up ideas here.** The lead owns follow-up
filing. If you notice something during implementation that belongs in
a follow-up, record it in the PR body `DEFERRED` section (with
rationale per the DEFERRAL RULE) and the lead will classify it at its
final invocation.

IMPORTANT CONTEXT:
(Add issue-specific context here — data locations, phasing
instructions, remote execution notes, dependencies, key files)

Now read the issue and begin work:
/bip.issue.work N
```

### Common context additions

**Filesystem mode** — always include this block when the issue involves running jobs on remote compute nodes. Check `shared_filesystem` in `.epic-config.json`:

*When `shared_filesystem: false` (laptop — files must be synced):*
```
- Use make remote-sync + make remote-tmux for running on remote servers
- Use /bip.scout to find an available server before remote operations
- Always rebuild after sync: make remote-tmux REMOTE_HOST=... CMD='zig build -Doptimize=ReleaseFast'
- Wrap the experiment in a Snakemake workflow
```

*When `shared_filesystem: true` (NFS — files already visible on all nodes):*
```
- Use /bip.scout to find an available server before remote operations
- For short/medium jobs (< ~30 min): block on SSH
    ssh <host> "cd <absolute_clone_path> && <command>"
  Results appear on NFS immediately — no sync or polling needed.
- For long jobs (hours): background the SSH call, then poll local NFS paths
    ssh <host> "cd <absolute_clone_path> && nohup <command> > out.log 2>&1 &"
  Use the awaiting-results phase with check_files pointing to local NFS
  output paths — no SSH needed to poll, just test -f /nfs/path/output.
- Never use make remote-sync or make remote-tmux in NFS mode.
- SSH quoting tip: if a command has complex quoting or special characters,
  write it to a temp file (e.g. /tmp/run-<N>.sh), then:
    ssh <host> "bash /nfs/path/to/run-<N>.sh"
  Clean up the temp file when the command finishes.
- Use the absolute clone path in SSH commands (expand ~ from clone_root
  before embedding — remote shells resolve ~ relative to the SSH user's
  home, which may differ from the NFS path).
```

**For experiments (Snakemake workflows):**
```
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

Write the composed prompt to a temp file, then use `bip spawn` with
`--prompt-file` to pass it. This avoids shell expansion issues with
quotes, braces, and special characters in the prompt.

```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)

# Write prompt to temp file (conductor does this, NOT via shell expansion)
# Use the Write tool to create /tmp/spawn-<N>.txt with the full prompt

# Clone mode: --name is NNN-clone (e.g. "281-cedar")
bip spawn --prompt-file /tmp/spawn-<N>.txt \
  --dir "$CLONE_ROOT/<clone-name>" \
  --name "<N>-<clone-name>"

# Worktree mode: --name is NNN-issue-NNN (e.g. "281-issue-281")
bip spawn --prompt-file /tmp/spawn-<N>.txt \
  --dir "$CLONE_ROOT/issue-<N>" \
  --name "<N>-issue-<N>"
```

**IMPORTANT**: Always use `--prompt-file`, never `--prompt "$(cat file)"`.
The `$(cat)` pattern causes zsh shell expansion errors with complex prompts.

**Do NOT** use raw `tmux new-window` / `tmux send-keys` / `claude` commands.
Always go through `bip spawn` which handles the full lifecycle correctly.

### Step 6: Confirm

Report to the user:
- Which clone was spawned
- Which issue it's working on
- Any phasing or gate criteria

If a persistent slot monitor is running (started by `/bip.epic`), the
conductor will receive automatic notifications when this worker changes
phase. No additional monitoring setup is needed.

If no monitor is running, suggest starting one or using
`/loop 10m /bip.epic.poll` to track progress.

## Creating new slots

**Clone mode** — create a new clone and register it:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
REPO=$(jq -r .github_repo .epic-config.json)
cd "$CLONE_ROOT"
git clone "git@github.com:$REPO.git" <new-name>
```
After creating, add the new name to `clone_names` in `.epic-config.json`.

**Worktree mode** — no registration needed; worktrees are created on demand
in Step 1 and named `issue-<N>`. No config changes required.

## Cleaning up slots after work

**Clone mode** — if a clone is on a non-main branch:
1. Check if there's an open PR: `gh pr list --head <branch>`
2. If merged/closed: `git checkout main && git pull --ff-only`
3. If open: warn user — they may want to resume

**Worktree mode** — when an issue's PR is merged:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
git worktree remove "$CLONE_ROOT/issue-<N>"
git branch -d <N>-short-desc
```
If the worktree has uncommitted changes, use `--force`. Check for an open
PR first — don't remove a worktree with unmerged work.

## Gitignore reminder

Target project repos should gitignore these files (add to `.gitignore`):
```
.epic-status.json
.epic-worklog.md
```

These files live in clones, not in bipartite itself.

## Conventions

Same as `/bip.epic`: `iN`/`pN` prefixes. Tmux windows named `NNN-YYY`
where NNN is the issue number and YYY is the clone/slot name
(e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).
