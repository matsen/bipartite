---
name: bip-conductor-spawn
description: Spawn a Claude session in a clone for an EPIC issue
---

# /bip-conductor-spawn

Spawn a Claude Code session in a tmux window to work on a GitHub issue.
The worker runs inside a **ralph-loop** with an **issue-lead subagent** that evaluates progress at stopping points.

This is fleet-conductor machinery: it executes a spawn, annotating it with live fleet facts the topic side (`/bip-epic`) cannot see.
See "Where the prompt comes from" below for the epic/conductor split.

## Usage

```
/bip-conductor-spawn <issue-number> [clone-name]
```

If clone-name is omitted, pick the best idle clone automatically.

## Configuration

Reads `.epic-config.json` from the repo root (see `/bip-conductor` for format).
**If the file does not exist**, stop and ask the user to configure it via `/bip-conductor` first.

## Where the prompt comes from

Most of the time `/bip-epic` has already decided an issue is ready and written the semantic brief — why it matters, scope, dependency/collision warnings from its issue-body analysis — to `$CLONE_ROOT/.spawn-prompts/`.
Check there first, under **either** naming convention live in that directory — `<N>.md` (current) or `spawn-<N>.txt` (older):

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
INTENT=$(find_spawn_intent "$CLONE_ROOT" <N>)
[ -n "$INTENT" ] && cat "$INTENT"
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-conductor-spawn`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory. Don't narrow the lookup to one filename pattern.

- **Intent file present**: it is the base for the `IMPORTANT CONTEXT` section of Step 4's prompt — don't re-derive what it already says.
  Check it against live fleet state (Step 2b) and append fleet facts it structurally couldn't know: which host/clone is actually free, a concurrent worker editing an overlapping file, a build running on a target remote host.
  Mark it consumed after a successful launch (Step 6).
  An epic-written brief carries an `EPIC:` reference near the top; the conductor counts live slots by it. Match it tolerantly — `grep -iE '^\**EPIC\**:? *#?([0-9]+)'` — since both `EPIC: 369` and `**EPIC**: #369` are in use. If an epic brief lacks one, ask the epic rather than inferring it. A user-originated brief legitimately has none.
- **No intent file** — compose the prompt from the issue directly. This is a first-class path: a user-originated spawn, a conductor-initiated respawn, routine maintenance. A user-originated issue may belong to no EPIC; never reject or defer it for that. See `/bip-conductor`'s "Two intake paths".

If the intent conflicts with current fleet state, resolve it from measured state and say so in your report — a contended host or a taken clone is a placement decision. Escalate only if either resolution risks an actual problem (see `/bip-conductor`'s "Arbitration").

## Workflow

### Prerequisite: Issue number required

Every spawn MUST target an existing GitHub issue.
If the conductor wants to spawn work that doesn't have an issue yet (reruns, follow-ups, quick experiments), file the issue first:

1. Write a minimal issue body (title + 3-sentence motivation + success criteria)
2. `gh issue create --title "..." --body-file ISSUE-*.md`
3. Then proceed with the spawn using the new issue number

Never write a spawn prompt with `issue=0` or without `/bip-issue-work <N>`.
Issueless spawns break EPIC tracking, PR linking, and conductor tracking.

### Step 1: Select or create slot

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

**Clone mode** (`local_worktrees` absent or false):

If clone-name not specified, find an idle clone: on `main`, clean, no live tmux pane in its directory, no `.epic-status.json`:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
OCCUPIED=$(tmux list-panes -a -F '#{pane_current_path}' 2>/dev/null) || OCCUPIED=""
for name in $(jq -r '.clone_names[]' .epic-config.json); do
  dir="$CLONE_ROOT/$name"
  [ "$(git -C "$dir" branch --show-current 2>/dev/null)" = "main" ] || continue
  # Capture, then test: a bare `-z` on failed output reads "unreadable" as "clean".
  # `st`, not `status`: zsh makes `status` read-only.
  st=$(git -C "$dir" status --porcelain 2>/dev/null) || continue
  [ -z "$st" ] || continue
  echo "$OCCUPIED" | grep -qxF "$dir" && continue   # live tmux pane here → owned
  [ -f "$dir/.epic-status.json" ] && continue         # a worker claimed it, unfinished
  echo "$name"
done
```

If `tmux` fails, `OCCUPIED` is empty and every clone looks free. Selection is best-effort; `bip spawn`'s refusal to launch into a directory a live pane occupies (Step 5) is the invariant.

Don't rank idle clones by build-cache size; it records history, not whether the next build hits. If clones are otherwise equivalent, pick arbitrarily.
If all busy, offer to create a new clone using a name from `new_clone_names` in the config.

**Worktree mode** (`local_worktrees: true`):

Slot name is always `issue-<N>`.
Branch name is `<N>-<slug>` where `<slug>` is the first 4 words of the issue title, lowercased and hyphenated.

Check if slot already exists:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
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

Preserve the prior assignment's state, then clear all three stale-state files: `.epic-status.json`, `.epic-worklog.md`, and `.claude/ralph-loop.local.md`. Clear them only if the preserve did not fail. Deletes use `find <absolute-path> -delete`, never `rm`. A `$` in the same command as `rm` trips Claude Code's destructive-removal guard, which bypass mode does not suppress. And cwd does not persist between Bash calls in an agent thread.

**Clone mode**:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
cd "$CLONE_ROOT/<clone>"
git checkout main && git pull --ff-only origin main
DEST=$(preserve_epic_state "$(pwd -P)" "$CLONE_ROOT" "before reassigning this clone to a new issue.")
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "Preserved prior assignment's worklog+status to $DEST"
elif [ "$rc" -eq 2 ]; then
    echo "PRESERVATION FAILED: cp did not succeed -- stop and investigate before deleting anything" >&2
fi
if [ "$rc" -ne 2 ]; then
    find "$CLONE_ROOT/<clone>" -maxdepth 1 \( -name '.epic-status.json' -o -name '.epic-worklog.md' \) -delete
    find "$CLONE_ROOT/<clone>/.claude" -maxdepth 1 -name 'ralph-loop.local.md' -delete 2>/dev/null
fi
# Stale build artifact, independent of the preserve (see "Stale binaries" below).
find "$CLONE_ROOT/<clone>/zig-out/bin" -maxdepth 1 -name 'phyz' -delete 2>/dev/null
```

**Fetch inside each clone, never once in the conductor.** Each clone has its own `origin/main`, so a conductor-level fetch followed by `git -C <clone> reset --hard origin/main` bases the worker on that clone's stale ref. Don't collapse the `cd` in a batch-spawn loop.

**Worktree mode**: resolve `CLONE_ROOT` *before* `cd "$SLOT"`, in the same command — `.epic-config.json` is not inside a linked worktree, and resolving after the `cd` yields an empty root silently:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
cd "$SLOT"
DEST=$(preserve_epic_state "$(pwd -P)" "$CLONE_ROOT" "before a fresh restart on the same issue.")
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "Preserved prior attempt's worklog+status to $DEST"
elif [ "$rc" -eq 2 ]; then
    echo "PRESERVATION FAILED: cp did not succeed -- stop and investigate before deleting anything" >&2
fi
if [ "$rc" -ne 2 ]; then
    find "$SLOT" -maxdepth 1 \( -name '.epic-status.json' -o -name '.epic-worklog.md' \) -delete
    find "$SLOT/.claude" -maxdepth 1 -name 'ralph-loop.local.md' -delete 2>/dev/null
fi
find "$SLOT/zig-out/bin" -maxdepth 1 -name 'phyz' -delete 2>/dev/null
```

**Resuming a clone parked mid-issue** (branch and worklog deliberately kept): delete `.epic-status.json` anyway. It is instructions, not context — `lead_guidance` outranks the launch prompt, so a stale "stand down" makes the worker stand down on arrival. Keep the worklog. Do **not** delete `zig-out/bin/phyz` here: on a parked slot that binary can be the provenance of an artifact the branch already committed. Instead tell the slot to rebuild before re-verifying.

```bash
find "$SLOT" -maxdepth 1 -name '.epic-status.json' -delete   # keep .epic-worklog.md and zig-out/
```

**Stale binaries.** A pre-existing `zig-out/bin/phyz` reports its build-time commit, not the clone's `HEAD`, and a stale one returns confident wrong numbers. Two remedies cover each other's blind spots: the prep-step delete above (fresh assignments only), and the check in the brief (Step 4, "For code changes"). The delete is scoped to that one binary on purpose; add bench binaries on their own line if wanted.

### Step 2b: Pre-launch staleness check

For every blocker/dependency the issue body names (an issue number, a PR number, "blocked on #N"), verify it's still true right now:

```bash
gh pr view <N> --json state,mergedAt   # merged already?
gh issue view <N> --json state          # closed already?
```

If a named blocker turns out to be resolved, correct the composed prompt's `IMPORTANT CONTEXT` — don't pass the stale claim through to the worker.

That answers "is the blocker resolved", not "is the work already done". Nothing updates an issue body when a PR lands part of it, so also check the issue itself:

```bash
gh pr list --state merged --search "<N>" --json number,title,mergedAt   # read the titles
gh issue view <N> --json body -q .body | sed -n '/[Ff]iles to modify/,/^#/p'   # then check those paths on main
git show origin/main:<path>   # does the described content already exist?
```

Treat an issue's `Depends-on` / related-PR list as a thread to walk, not a list to state-check.

### Step 3: Read the issue

```bash
gh issue view <number> --json title,body
```

Extract key context: what the issue asks for, data locations, phasing, dependencies.
If an intent file exists, this is a cross-check against the live issue body, not a replacement for reading it — the intent file may have gone stale since the epic wrote it.

### Step 4: Compose the prompt

The prompt has two parts: (1) the work instructions passed as the initial message to `claude` via `--prompt-file`, and (2) a ralph-loop invocation that the worker runs as its first action.
The ralph-loop prompt is kept SHORT (no special characters) — just a reminder to continue.

The `IMPORTANT CONTEXT` section at the bottom is where the two sources combine: start from the epic's intent file when one exists, correct it per Step 2b, then append fleet facts only the conductor can see.
Spend the measurement here rather than in later corrections, which race the worker: hash the inputs the issue names, resolve every bare path against the real tree, locate the tools the work needs and report their versions.
Put a fact that invalidates an instruction *inside* that instruction (claim, why it's wrong, file:line), not in a separate warnings list.

Re-read every issue-specific *sentence* when reusing a previous prompt, not just every issue number — a substitution on the number cannot reach a sentence that describes the other issue's work without naming it.

Fill in two lines in COMPLETION every time:

- **`LANDING DELEGATION:`** — quote the recorded standing user delegation for this repo from the conductor's decisions log (file and date), or `NONE RECORDED`. `NONE RECORDED` is the default and means the worker stops at a clean gate and the user merges. Workers cannot read the decisions log, so it travels in the brief or not at all.
- **`JOINT LANDING GATE: YES | NO`** — YES only when landing is hard to reverse. Not for size, risk, or a shared file (that is sequencing). A hold stalls a finished slot invisibly, so YES needs a reason. For everything else, NOTIFY the owning session when the PR opens; that is detection, not prevention.

For an epic-reserved artifact, the gate is the shape of the edit: additive-only → land and notify; any removal or alteration of epic-recorded text → announce intent-to-land and wait for an ack. Check with a three-dot diff; a two-dot diff against `origin/main` reports everything landed since the fork as deletions:

```bash
git diff origin/main...HEAD -- <the owned files>
```

When you change a gating rule, sweep the slots already running under the old one; a prompt is frozen at launch.

**Prompt file** (written by conductor to /tmp/spawn-N.txt):
````
You are working on GitHub issue #N TITLE.

First, run this command to start the iteration loop:
/ralph-loop:ralph-loop --completion-promise 'ISSUE WORK COMPLETE' --max-iterations 20 Continue working on the task. Read .epic-status.json and .epic-worklog.md for context. Output ISSUE WORK COMPLETE in promise tags when done.

EPIC STATUS PROTOCOL — You MUST follow this:
1. At session start, write .epic-status.json (see format below)
2. Update it when you transition between phases, and refresh `summary` and
   `updated_at` immediately before and after each of these, phase change or
   not (there is no clock interrupt inside a turn, so tie it to events):
   - starting or finishing a build or test run you expect to exceed a few minutes
   - launching or reaping a remote job
   - spawning a subagent, and reading its result
   - any append to .epic-worklog.md
3. Update it when you finish or encounter a blocker
4. Maintain .epic-worklog.md as a narrative log (see format below)

.epic-status.json fields:
  issue — the issue number
  title — short title
  phase — one of: exploring, coding, testing, awaiting-results, quality-gate, needs-human, completed
  summary — human-readable one-liner
  updated_at — ISO 8601 UTC from `date -u +%Y-%m-%dT%H:%M:%SZ`. Never a
    placeholder, never local time with a `Z` appended.
  blockers — list of blockers (empty list if none)
  scope — one-line restatement of issue goal (set by lead)
  stop_reason — category from lead decision framework (set by lead)
  lead_guidance — what the lead told you to do next (set by lead)
  lead_notes — list of lead evaluation entries (set by lead)
  completed_at — ISO 8601 timestamp set by the lead after the terminal
    completed ceremony; do not set yourself.
  awaiting — set when waiting for experiment results (description,
    check_cmd, check_files, started_at, timeout_hours). `started_at` is
    real UTC from `date -u`: it is compared against `timeout_hours`.

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

PROMPT VS ISSUE — RAISE IT, DO NOT SETTLE IT. The issue binds what the work
is; this prompt may narrow it but not widen it. Ask whether obeying the
prompt would make you do MORE than the issue asks, or LESS:
- A prompt constraint that SUBTRACTS ("do not touch X", "this phase only",
  "defer Y until Z lands") wins; it rests on fleet state the issue cannot see.
- A prompt instruction that ADDS a deliverable the issue lacks is stale; the
  issue wins. Tell the conductor.
- The prompt's gate lines (test targets, SHA rules, approvals) win outright.
Do not quietly reconcile a contradiction in either direction.

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
2. Set the awaiting field with check_cmd and check_files. If the work runs
   on a host that does not share this filesystem, check_files cannot name
   a local path: put the real check in check_cmd and omit check_files or
   mark it remote.
3. Run check_cmd once while the work is definitely unfinished and confirm
   it EXITS NON-ZERO. The exit status is the whole interface; the loop never
   reads stdout. Use a command whose status is the answer (`test -s`, the
   tool's own status), keep `set -o pipefail` through any pipe, and never
   wrap the probe in `|| echo`. For a remote probe, check the status of the
   whole `ssh` call from outside it.
4. Each ralph-loop iteration: run check_cmd, if not ready end the turn
5. After 3 consecutive check failures, set stop_reason to
   mechanical-blocker and invoke the lead

WAITING: poll a PID you captured at launch (`$!`), never a pattern —
`pgrep -f` or `ps | grep` on any fragment of your own commands matches your
own session (this prompt is your argv) and never exits. Better: background
the command and let the harness re-invoke you. Any long foreground command
makes you unreachable to SendMessage; never hold the foreground for
something that is not work.

GIT: write any multi-line commit message to a file and use `git commit -F`;
a nested `"` in `-m "..."` truncates the message and breaks the `&&` chain.
After pushing, confirm the push landed (`git ls-remote origin <branch>`
matches `git rev-parse HEAD`), not just that the command ran.

rm AND $ — KEEP THEM OUT OF THE SAME COMMAND. Claude Code's destructive-
removal guard fires when a command contains both `$` (anywhere) and the
word rm/rmdir, bypass mode does not suppress it, and an unattended worker
stops at its prompt. Use literal absolute paths with no `$`, or
`find <dir> -maxdepth 1 -name 'pat' -delete` (never `-exec rm`). Split any
unrelated `$(...)`, `$?` or `$HOME` into a separate command.

REMOTE RUN OUTPUTS DO NOT SURVIVE YOUR OWN MERGE. Gitignored outputs on a
remote host live only in a pooled clone that is reset after your PR lands.
Before you request landing, copy anything a follow-up might need to
`$CLONE_ROOT/.preserved/<slug>/` with a README (what it is, source host and
path, which issue), and say in the PR body where it went — or commit it if
small and it belongs in the repo, or state explicitly that it is
regenerable and at what cost.

PUSH NOTIFICATION — on the EVENT, not a file write. When you finish, stand
down, hand off, or get blocked, `SendMessage` the conductor a one-line
notification (issue number, state, one-line summary), whether or not
.epic-status.json exists — /bip-pr-land deletes it. Read the address from
`$CLONE_ROOT/.conductor-session` (resolve CLONE_ROOT from .epic-config.json).
Do NOT `ListAgents` and pick a plausible row: workers sharing a clone root
share a name prefix. If the file is missing or the send fails, do not retry
and do not guess — record the failure in your FINAL RECAP. Send at the
point you decide to land, and again after if anything changed.

DO NOT EDIT THE EPIC ISSUE'S BODY. If you find it stale or contradicted by
your work, report that to the conductor and let the epic session fix it:
the epic's push path guards against concurrent edits and yours does not.

SUBAGENTS: do not delegate a permission-gated action (rm, force-push, a
write outside the clone, an unfamiliar binary) — a subagent's approval
prompt blocks your session with a command you did not write. Delegate
reading and analysis freely. A subagent report is a snapshot at an unknown
instant: re-derive any load-bearing claim yourself before acting on it.

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
   - If it says PHASE: quality-gate and `.epic-status.json` still
     carries `stop_reason: awaiting-human-merge` (COMPLETION step 5b)
     → the gate is clean and the merge is the user's; print the FINAL
     RECAP and output the completion promise
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

   A `make` exit of 124 (killed by `timeout`) or 144 (`make: *** wait: No
   child processes`) is not a verdict — the target never reached its
   summary line. Re-run the check directly, not through a timeout-capped or
   backgrounded wrapper, and report an unreached gate as UNMEASURED.

   A PR that survives several review rounds accumulates one body section
   per round ("Reviewer follow-up: ...", a growing "History"). /bip-pr-check
   flags this and offers a rewrite — take it, unprompted. The body should
   read as the current state (premise, results, interpretation), not as a
   log of how it got there.
5. When both pass clean (or remaining findings are all deferred), take
   the branch your LANDING DELEGATION line below selects. Either way,
   do NOT spawn the next slot: handing off needs explicit permission.

   a. A delegation is recorded (and, if JOINT LANDING GATE says YES,
      both approvals are in):
      - Land the PR yourself with /bip-pr-land, but only if the work
        obviously matches the issue and nothing needs the user's
        judgment (step 6's test). A clean gate alone is not enough: if
        a result reversed the issue's premise, leave the PR open and
        say so in the recap. Announce, then land without waiting.
      - Invoke the issue-lead one final time — it sets phase to
        completed and files any follow-ups from the PR body's DEFERRED
        section. It refuses `completed` while the PR is open, so this
        call comes after the land, never before.

   b. `NONE RECORDED` — the user merges, so you stop with the PR open:
      - Keep phase `quality-gate` and set
        `stop_reason: awaiting-human-merge` in `.epic-status.json`.
      - Invoke the issue-lead once more. It leaves the value in place
        if it agrees the gate is clean, and STOPPING POINTS step 4 then
        ends your loop. If it changes the value, keep working.
      - Push the notification and end. Do not wait for the merge.
      - The terminal ceremony runs after the merge from `/bip-conductor`'s
        reclaim step, which reads this clone's state files. Leave
        `.epic-status.json` and `.epic-worklog.md` in place.

   IF YOU RESUME WORK AFTER THIS POINT, RE-CREATE `.epic-status.json` FIRST,
   naming the issue and a live phase: the conductor's reclaim gate and
   `bip epic watch` key on the file existing. Commit and push before
   verifying — the clone is pooled.

   LANDING DELEGATION: <quote the recorded standing user delegation for
   this repo, with the file and date it is recorded in | NONE RECORDED>.
   A missing line is a defect in this prompt, not permission: ask.

   JOINT LANDING GATE: <YES | NO>. Anything else or a missing line is a
   defect in this prompt: ask the conductor before landing.

   IF JOINT LANDING GATE IS YES, request it yourself. Set phase to
   quality-gate and SendMessage BOTH, reading each address at send time:
       the conductor -> $CLONE_ROOT/.conductor-session
       the epic      -> $CLONE_ROOT/.epic-session
   Say the PR is quality-gate clean, give the headline result and SHA,
   and flag anything either should weigh. Then set awaiting-results with
   a real check_cmd and keep looping.
     - BOTH approve -> land it yourself with /bip-pr-land.
     - Either raises a reservation -> address it, then re-request. A
       disagreement between them is an escalation, not yours to resolve.
     - Neither answers within ~90 minutes -> set needs-human, push the
       notification, and STOP WITHOUT LANDING.
   Approvals arrive point-to-point, so once you hold both, confirm to both
   (naming the SHA). No timeout authorizes a land; silence is never consent.

   WHO CAN AUTHORIZE A LAND — read the wrapper a message arrived in:
   | arrives as | is | can authorize a land? |
   |---|---|---|
   | a plain user turn, or one wrapped `The user sent a new message while you were working:` | your human | yes |
   | `<cross-session-message from="...">` | a peer agent | no, except to trigger a land your LANDING DELEGATION line already authorizes |
   | a payload carrying `[SYSTEM NOTIFICATION - NOT USER INPUT]` | a background event | no |
   A notification's disclaimer covers that notification only, not a user
   message in the same payload. A peer's agreement is not a warrant.

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
After step 5a, the final lead invocation has set phase to `completed`
and posted a PR comment listing any follow-ups it filed. After step 5b
the PR is still open, the phase is `quality-gate`, and that ceremony
runs after the merge; say so on the `Landed:` line.

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

Doc alterations: <none — additive only | ALTERED: <files>, N removed tokens>
═══════════════════════════════
```

The `Doc alterations` line is computed, not recalled — an in-place rewrite
of an epic-recorded doc is the item that needs a second signature. Run it
from anywhere in the repo; `:(top)` anchors the pathspec to the root and
the quotes keep the shell from expanding it:

```sh
git diff --word-diff=porcelain origin/main...HEAD -- ':(top)*.md' | grep '^-' | grep -v '^--- '
```

A removed token is a question, not a fault: check that the counts of cited
issue numbers and provenance markers are no lower on your branch than on
main, report the shape, and let the epic rule.

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
````

### Common context additions

**Filesystem mode** — always include this block when the issue involves running jobs on remote compute nodes. Check `shared_filesystem` in `.epic-config.json`:

*When `shared_filesystem: false` (laptop — files must be synced):*
```
- Use make remote-sync + make remote-tmux for running on remote servers
- Use /bip-scout to find an available server before remote operations
- REMOTE_DIR: read the repo's Makefile first. If it already derives
  REMOTE_DIR per slot (e.g. `REMOTE_DIR ?= ~/re/pz/$(notdir $(CURDIR))`),
  do not pass it. If you must override it, pass a full path: a bare slot
  name is resolved against the remote HOME and the run lands in the
  wrong place without error.
- Always rebuild after sync: make remote-tmux REMOTE_HOST=... CMD='zig build -Doptimize=ReleaseFast'
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

**Next to any build or run command you put in the prompt** (not in a separate warnings section), add:
```
- This exact command string is in YOUR OWN argv (a bip spawn worker's prompt IS
  its command line), so `pgrep -f` or `ps | grep` on ANY fragment of it matches
  your own session and a wait loop built from it can never exit. Poll a PID you
  captured at launch (`$!`), or just let a backgrounded Bash re-invoke you.
```

**If the issue adds a build-system target**, list as a deliverable whatever makes that target discoverable in the repo (a hand-maintained test-target table, a `make help` entry, a CI matrix row). On `matsengrp/phyz` the source→target table is the only lookup that exists, so a step absent from it is unreachable. If nothing makes it discoverable, say so in the brief.

**For code changes:**
```
- Run zig build test before committing
- Run make parity if touching shared alignment code
- Check PRE-MERGE-CHECKLIST.md
- NEVER trust a pre-existing zig-out/bin/phyz. Before you measure ANYTHING
  with it, run this from the clone root; if it prints STALE or UNDETERMINED,
  rebuild first. A stale binary does not error -- it returns a confident
  wrong number under newer-looking provenance.
      # `--version` prints `phyz <version> (<commit>)`; older binaries print
      # the describe string alone, hence the `-g` fallback.
      vl=$(zig-out/bin/phyz --version)
      bc=$(printf '%s' "$vl" | sed -n 's/.*(\([0-9a-f]\{7,40\}\))[[:space:]]*$/\1/p')
      # Validate: an unresolvable SHA otherwise falls through to "ok".
      if [ -n "$bc" ]; then bc=$(git rev-parse --verify --quiet "${bc}^{commit}") || bc=""; fi
      if [ -z "$bc" ]; then
        vs=$(printf '%s' "$vl" | awk '{print $NF}'); vs=${vs%-dirty}
        case "$vs" in *-g*) ref="${vs##*-g}" ;; *) ref="$vs" ;; esac
        bc=$(git rev-parse --verify --quiet "${ref}^{commit}") || bc=""
      fi
      hc=$(git rev-parse --verify HEAD)
      if [ -z "$bc" ] || [ "$bc" = "unknown" ]; then
        echo "UNDETERMINED: no commit recoverable from '$vl' (tag not fetched, or built with no git) -- do NOT assume stale"
      elif [ "$bc" != "$hc" ] && [ -n "$(git diff --name-only "$bc" "$hc" -- src build.zig build.zig.zon)" ]; then
        echo "STALE: binary=$bc HEAD=$hc, and code differs -- rebuild before measuring"
      fi
      # A commit match does NOT establish content identity on a dirty tree:
      [ -n "$(git status --porcelain)" ] && echo "NOTE: tree dirty -- commit matches, content identity not established"
```

The check compares code between the two commits (`src build.zig build.zig.zon`, deliberately not `tests`) so a results-only commit after a build is not STALE. It resolves both sides to a commit rather than comparing `git describe` strings, because phyz tags every commit and a describe string changes when its tag lands. It parses `phyz --version`, so re-run it against a fresh binary whenever that line's format changes.

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
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)

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

`bip spawn`'s behaviour comes from the installed binary, not from the bipartite source. Before relying on a recent `bip` change, confirm the binary has it (`strings "$(command -v bip)" | grep -F '<new string>'`) and rebuild if not.

### Step 6: Confirm

If launch succeeded and an intent file (`$INTENT` from Step 3) was used, mark it consumed now — don't delete it, and verify the move, since an unmoved brief reads as pending forever:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
mark_spawn_intent_consumed "$INTENT"
ls "$CLONE_ROOT/.spawn-prompts/consumed/<N>.md"
```

The directory lives outside every clone's git, so deletion there is unrecoverable; moving it aside lets `/bip-conductor-tuckin` report it as consumed.

An epic can append to `.spawn-prompts/<N>.md` after you consumed it, recreating a fragment in the live queue. Tell a delta from a re-brief by content — a brief opens with an `EPIC:` header, a delta does not:

```bash
if [ -f "$CLONE_ROOT/.spawn-prompts/consumed/$N.md" ] \
   && ! grep -qiE '^\**EPIC\**:? *#?[0-9]+' "$CLONE_ROOT/.spawn-prompts/$N.md"; then
  echo "DELTA, not a brief"     # deliver it; do NOT spawn from it
else
  echo "BRIEF"
fi
```

Deliver a delta to a live slot by appending it to that slot's `.epic-worklog.md` before messaging. If no slot is live on the issue, post it as an issue comment instead.

A prompt-template change does not reach running workers; the prompt is frozen at launch and `RECOVERING CONTEXT` re-reads the same frozen text. Either message live workers to record the correction in their worklog, or accept that it starts at the next spawn — decide which.

**Verify the worker actually started before reporting it live.** `context used N%` proves the prompt was read, not that work began; a session blocked on the folder-trust dialog shows the same. Check for a created branch or a tool call in the pane.

Report to the user:
- Which clone was spawned
- Which issue it's working on
- Any phasing or gate criteria

If a persistent slot monitor is running (started by `/bip-conductor`), the conductor will receive automatic notifications when this worker changes phase.
If none is running, suggest starting one; workers also report back on their own.

## Creating new slots

**Clone mode** — create a new clone and register it:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json) || { echo "FATAL: cannot resolve clone_root" >&2; exit 1; }
[ -n "$CLONE_ROOT" ] && [ -d "$CLONE_ROOT" ] || { echo "FATAL: clone_root '$CLONE_ROOT' missing" >&2; exit 1; }
REPO=$(jq -r .github_repo .epic-config.json)
cd "$CLONE_ROOT"
git clone "git@github.com:$REPO.git" <new-name>
[ -d "$CLONE_ROOT/<new-name>" ] || { echo "FATAL: clone <new-name> not created" >&2; exit 1; }
```

Then all of:

1. **Add the name to `clone_names` in `.epic-config.json`.** `bip spawn` doesn't consult it, but Step 1's selection and the conductor's scans do; an unregistered clone is invisible.
2. **Trust the directory before spawning into it.** A fresh clone opens on Claude Code's folder-trust dialog, which queues the prompt. If you answer it from the conductor, read which option is highlighted first (`grep -nE 'No, exit|Yes, I trust'` the pane) and send `Down` before `Enter` when `No, exit` is highlighted — never a blind `Enter`.
3. **Restart `bip epic watch`.** It enumerates slots at startup only. Find it with `ps -eo pid,args | grep -E '^\s*[0-9]+ bip epic watch'`, not `pgrep -af` (which matches every worker's prompt).

**Worktree mode** — no registration needed; worktrees are created on demand in Step 1 and named `issue-<N>`.

## Cleaning up slots after work

**Clone mode** — if a clone is on a non-main branch:
1. Check if there's an open PR: `gh pr list --head <branch>`
2. If merged/closed: `git checkout main && git pull --ff-only`
3. If open: warn user — they may want to resume

**Worktree mode** — when an issue's PR is merged:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
git worktree remove "$CLONE_ROOT/issue-<N>"
git branch -d <N>-short-desc
```
If the worktree has uncommitted changes, use `--force`.
Check for an open PR first — don't remove a worktree with unmerged work.

## Gitignore reminder

Target project repos should gitignore these files (add to `.gitignore`):
```
.epic-status.json
.epic-worklog.md
.epic-notifications.log
.epic-decisions.md
```

`.epic-status.json` and `.epic-worklog.md` live in each clone/worktree.
`.epic-notifications.log` and `.epic-decisions.md` live in the conductor cwd (written by `bip epic watch` and by the conductor itself).
None should be checked in.

## Conventions

Same as `/bip-epic`: `iN`/`pN` prefixes.
Tmux windows named `NNN-YYY` where NNN is the issue number and YYY is the clone/slot name (e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).

## Layout config (issue #149)

`.epic-config.json` keeps working.
The newer global `layout:` block in `~/.config/bip/config.yml` configures worktree mode for non-EPIC `bip spawn`; see `docs/guides/layout.md`.
