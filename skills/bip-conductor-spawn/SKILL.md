---
name: bip-conductor-spawn
description: Spawn a Claude session in a clone for an EPIC issue
---

# /bip-conductor-spawn

Spawn a Claude Code session in a tmux window to work on a GitHub issue.
The worker runs inside a **ralph-loop** with an **issue-lead subagent**
that evaluates progress at stopping points.

This is fleet-conductor machinery: it executes a spawn, annotating it
with live fleet facts the topic side (`/bip-epic`) cannot see. See
"Where the prompt comes from" below for the epic/conductor split.

## Usage

```
/bip-conductor-spawn <issue-number> [clone-name]
```

If clone-name is omitted, pick the best idle clone automatically.

## Configuration

Reads `.epic-config.json` from the repo root (see `/bip-conductor` for
format). **If the file does not exist**, stop and ask the user to
configure it via `/bip-conductor` first.

## Where the prompt comes from

Most of the time `/bip-epic` has already decided an issue is ready and
written the semantic brief — why it matters, scope, dependency/collision
warnings from its issue-body analysis — to `$CLONE_ROOT/.spawn-prompts/`.
Check there first, under **either** naming convention live in that
directory — `<N>.md` (current) or `spawn-<N>.txt` (older, still
written by some sessions):

```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
INTENT=$(ls "$CLONE_ROOT"/.spawn-prompts/{<N>.md,spawn-<N>.txt} 2>/dev/null | head -1)
[ -n "$INTENT" ] && cat "$INTENT"
```

Checking only `<N>.md` silently misses a real, live `spawn-<N>.txt` and
falls through to the escape hatch below with no warning — exactly the
failure this split exists to prevent, reintroduced by a filename. Don't
narrow this check to one pattern.

- **Intent file present**: it is the base for the `IMPORTANT CONTEXT`
  section of Step 4's prompt below — don't re-derive what it already
  says. Your job is to check it against live fleet state (Step 2b) and
  append fleet facts it structurally couldn't know: which host/clone is
  actually free, a concurrent worker editing an overlapping file, a
  build running on a target remote host. Delete the intent file after a
  successful launch (Step 5) — it has done its job.
- **No intent file** (conductor-initiated respawn, routine maintenance,
  quick fix with no upstream epic session): compose the prompt from the
  issue directly, same as before. This is the escape hatch, not the
  default path.

If the intent conflicts with current fleet state in a way that isn't a
mechanical fix (e.g. it asks for a clone/host that's genuinely
contended, not just occupied by a finished session) — don't resolve it
yourself. State the conflict to the user and let them decide; this
mirrors the user-confirmation gate every spawn already goes through.

## Workflow

### Prerequisite: Issue number required

Every spawn MUST target an existing GitHub issue. If the conductor wants
to spawn work that doesn't have an issue yet (reruns, follow-ups, quick
experiments), file the issue first:

1. Write a minimal issue body (title + 3-sentence motivation + success criteria)
2. `gh issue create --title "..." --body-file ISSUE-*.md`
3. Then proceed with the spawn using the new issue number

Never write a spawn prompt with `issue=0` or without `/bip-issue-work <N>`.
Issueless spawns break EPIC tracking, PR linking, and conductor polling.

### Step 1: Select or create slot

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

**Clone mode** (`local_worktrees` absent or false):

If clone-name not specified, find an idle clone. A good pick is on `main`,
clean, and has no live tmux pane in its directory — the pool is shared
across operators and finishing workers self-claim slots via
`/bip-conductor-handoff`, so filter these out to avoid choosing one that's
already taken:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
OCCUPIED=$(tmux list-panes -a -F '#{pane_current_path}' 2>/dev/null)
for name in $(jq -r '.clone_names[]' .epic-config.json); do
  dir="$CLONE_ROOT/$name"
  [ "$(git -C "$dir" branch --show-current 2>/dev/null)" = "main" ] || continue
  [ -z "$(git -C "$dir" status --porcelain 2>/dev/null)" ] || continue
  echo "$OCCUPIED" | grep -qxF "$dir" && continue   # live tmux pane here → owned
  [ -f "$dir/.epic-status.json" ] && continue         # a worker claimed it, unfinished
  echo "$name"
done
```

This selection is **best-effort** — it avoids clones that are obviously
taken. The actual guarantee is `bip spawn` itself: it refuses to launch
into a directory a live tmux pane already occupies (`--force` overrides), so
even if the pool churns between selection and spawn you can never land a
second agent in one checkout. Prefer clones with clean worktrees. If all
busy, offer to create a new clone using a name from `new_clone_names` in
the config.

**Worktree mode** (`local_worktrees: true`):

Slot name is always `issue-<N>`. Branch name is `<N>-<slug>` where `<slug>`
is the first 4 words of the issue title, lowercased and hyphenated.

Check if slot already exists:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
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
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
cd "$CLONE_ROOT/<clone>"
git checkout main && git pull --ff-only origin main
rm -f .epic-status.json .epic-worklog.md
```

**Fetch inside each clone, never once in the conductor.** Each clone has its
own `origin/main`, so a conductor-level fetch followed by
`git -C <clone> reset --hard origin/main` resets the clone to *its own stale*
ref and silently bases the worker on an old commit. The `cd` above is what
makes this correct — don't collapse it in a batch-spawn loop (bitten twice in
2026-08). `.epic-status.json` and `.epic-worklog.md` are gitignored, so
`reset --hard` preserves them.

**Worktree mode**: worktree was just created fresh from main — just clear
any stale status files from a previous run on this same issue:
```bash
rm -f "$SLOT/.epic-status.json" "$SLOT/.epic-worklog.md"
```

**State cleanup is mandatory** — stale files from a previous assignment
will confuse the worker and lead.

### Step 2b: Pre-launch staleness check

Mechanical, topic-agnostic, and easy to skip under pressure — don't.
For every blocker/dependency the issue body names (an issue number, a
PR number, "blocked on #N"), verify it's still true right now:

```bash
gh pr view <N> --json state,mergedAt   # merged already?
gh issue view <N> --json state          # closed already?
```

An issue that declares itself blocked on an already-merged PR is the
single most common staleness bug measured in practice (multiple
instances in three days). If a named blocker turns out to be resolved,
correct the composed prompt's `IMPORTANT CONTEXT` — don't pass the
stale claim through to the worker.

### Step 3: Read the issue

```bash
gh issue view <number> --json title,body
```

Extract key context: what the issue asks for, data locations, phasing,
dependencies. If a `.spawn-prompts/` intent file exists (see "Where
the prompt comes from" above, under either naming convention), this is
a cross-check against the live issue body, not a replacement for
reading it — the intent file may itself have gone stale since the
epic wrote it.

### Step 4: Compose the prompt

The prompt has two parts: (1) the work instructions passed as the
initial message to `claude` via `--prompt-file`, and (2) a ralph-loop
invocation that the worker runs as its first action. The ralph-loop
prompt is kept SHORT (no special characters) — just a reminder to
continue. The detailed instructions are already in the conversation
from the initial message.

The `IMPORTANT CONTEXT` section at the bottom is where the two sources
combine: start from the epic's intent file when one exists, correct it
per Step 2b, then append fleet facts only the conductor can see —
which host/clone is actually free right now, a concurrent worker
editing a file this issue also touches, a build in progress on a
target remote host. Without this annotation step those fleet warnings
never make it into the prompt at all.

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

HUMAN INTERRUPT — The AUTONOMY rule governs YOUR decisions, not the
human steering. If a human interrupts to ask a question, discuss, or
change direction, that supersedes the loop: PAUSE it FIRST
(/ralph-loop:cancel-ralph, or rm .claude/ralph-loop.local.md) so the
stop hook stops re-injecting "continue working", then engage. Do NOT
grind stale work between their messages just to satisfy the hook. When
the discussion resolves, either RESTART the loop with an updated prompt
reflecting the new direction, or — if the task is done or handed off —
confirm completion and wind down. Never silently drop autonomy or keep
running the old prompt against a changed plan.

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
   a. Run /bip-pr-check — fix everything it flags, commit and push
   b. Run /bip-pr-review — triage each finding (see REVIEW TRIAGE below),
      fix the ones you'll address, commit and push
   c. If either flagged issues that you fixed, go back to (a)
   Track quality gate iterations in .epic-status.json

   A PR that survives several review rounds accumulates one body section
   per round ("Reviewer follow-up: ...", a growing "History"). /bip-pr-check
   flags this and offers a rewrite — take it, unprompted. The body should
   read as the current state (premise, results, interpretation), not as a
   log of how it got there.
5. When both pass clean (or remaining findings are all deferred):
   - Land the PR yourself with /bip-pr-land, but only if the work
     obviously matches the issue and nothing needs the user's judgment
     (step 6's test). A clean gate alone is not enough: if a result
     reversed the issue's premise, leave the PR open and say so in the
     recap. Announce, then land without waiting.
   - Do NOT spawn the next slot. Handing off needs explicit permission.
   - Invoke the issue-lead one final time — it sets phase to completed
     and files any follow-ups from the PR body's DEFERRED section
   - Print a FINAL RECAP (see below), then the completion promise
     ISSUE WORK COMPLETE
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

If all three are not true, do the work now — a few minutes, an hour for a
clear win, a test gap you noticed, an unhandled edge case, something
mechanical but tedious, anything in files you're already editing, even a
50%-larger diff that stays coherent.

When in doubt, DO NOT defer. Bring it to the issue-lead with your reasoning.
The cost of an over-large PR is small (split it later if needed). The cost
of an under-finished PR is high (follow-up churn, broken windows, work
re-loaded into context cold weeks later, user prompts to merge what
should have been one coherent change).

FOLDING A FILED FOLLOW-UP BACK INTO ITS ORIGINATING PR — the reverse of
the DEFERRAL RULE above. A follow-up issue may already exist (filed by
issue-lead, or otherwise) that references an open PR as where it came
from. If the user asks to work on that follow-up without explicitly
asking for a new branch/PR/slot, don't assume the issue number implies
new work-unit machinery — a GitHub issue number in a request is not
automatically a new spawn target. Check for explicit continuation
language first ("same clone," "this PR," "in place," "as part of the
current work"). If present, treat it as scope-folding, not a new unit
of work:
  - Stay on the current branch. Don't create a new branch, clone, or
    worktree for the issue number.
  - Extend the current PR — a comment describing the plan (and any
    feasibility findings) is enough to make it resumable; no new PR.
  - Post a short redirect comment on the follow-up issue noting it will
    close via the current PR instead of a standalone one.
  - If this is an EPIC slot, update `.epic-status.json`/`.epic-worklog.md`
    in place to describe the extended scope — don't replace them with
    state for the follow-up issue as if it were a separate assignment.

Only spawn a new slot/branch when the user asks for genuinely separate
or parallel work — a different issue being *mentioned* is not that ask.

REVIEW TRIAGE — For each /bip-pr-review finding, apply the DEFERRAL RULE above:
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
Landed: <merged by me | left open — REASON>

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
/bip-issue-work N
```

### Common context additions

**Filesystem mode** — always include this block when the issue involves running jobs on remote compute nodes. Check `shared_filesystem` in `.epic-config.json`:

*When `shared_filesystem: false` (laptop — files must be synced):*
```
- Use make remote-sync + make remote-tmux for running on remote servers
- Use /bip-scout to find an available server before remote operations
- ALWAYS pass REMOTE_DIR=<remote_root>/<this-slot's-dir-name> on every
  make remote-sync / remote-tmux call (clone mode: the fruit name;
  worktree mode: issue-<N> -- create it remotely first if it doesn't
  exist yet). The config default is shared by every slot -- using it
  lets concurrent workers clobber each other's remote checkout.
- Always rebuild after sync: make remote-tmux REMOTE_HOST=... REMOTE_DIR=... CMD='zig build -Doptimize=ReleaseFast'
- Wrap the experiment in a Snakemake workflow
```

*When `shared_filesystem: true` (NFS — files already visible on all nodes):*
```
- Use /bip-scout to find an available server before remote operations
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

`bip spawn` refuses to launch into a directory a live tmux pane already
occupies (exits non-zero with `refusing: a tmux pane is already live in
<dir>`), so a clone claimed between Step 1 and here is caught at the spawn
itself — no second agent can land in one checkout. If you hit that refusal,
pick another idle clone. Pass `--force` only when you deliberately want a
second session in the same directory.

```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")

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

If launch succeeded and a `.spawn-prompts/` intent file (`$INTENT` from
Step 3) was consumed, delete it now — its job is done, and leaving it
behind is exactly the kind of finished-but-undeleted artifact that
accretes across a fleet.

Report to the user:
- Which clone was spawned
- Which issue it's working on
- Any phasing or gate criteria

If a persistent slot monitor is running (started by `/bip-conductor`), the
conductor will receive automatic notifications when this worker changes
phase. No additional monitoring setup is needed.

If no monitor is running, suggest starting one or using
`/loop 10m /bip-conductor-poll` to track progress.

## Creating new slots

**Clone mode** — create a new clone and register it:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
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
CLONE_ROOT=$(jq -r .clone_root .epic-config.json | sed "s|^~|$HOME|")
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
.epic-notifications.log
```

`.epic-status.json` and `.epic-worklog.md` live in each clone/worktree.
`.epic-notifications.log` lives in the conductor cwd and is written by
`bip epic watch`. None should be checked in.

## Conventions

Same as `/bip-epic`: `iN`/`pN` prefixes. Tmux windows named `NNN-YYY`
where NNN is the issue number and YYY is the clone/slot name
(e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).

## Layout config (issue #149)

`.epic-config.json` keeps working. The newer global `layout:` block in
`~/.config/bip/config.yml` configures worktree mode for non-EPIC `bip
spawn`; see `docs/guides/layout.md`.
