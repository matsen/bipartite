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
Check there first, under **either** naming convention live in that directory — `<N>.md` (current) or `spawn-<N>.txt` (older, still written by some sessions):

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
INTENT=$(find_spawn_intent "$CLONE_ROOT" <N>)
[ -n "$INTENT" ] && cat "$INTENT"
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-conductor-spawn`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory (see `skills/lib/spawn-intent.sh` in the `bipartite` repo).

Checking only `<N>.md` silently misses a real, live `spawn-<N>.txt` and falls through to the escape hatch below with no warning — exactly the failure this split exists to prevent, reintroduced by a filename.
Don't narrow this check to one pattern.

- **Intent file present**: it is the base for the `IMPORTANT CONTEXT` section of Step 4's prompt below — don't re-derive what it already says.
  Your job is to check it against live fleet state (Step 2b) and append fleet facts it structurally couldn't know: which host/clone is actually free, a concurrent worker editing an overlapping file, a build running on a target remote host.
  Mark the intent file consumed after a successful launch (Step 6) — it has done its job, but the directory lives outside git, so this is a move to `consumed/`, not a delete (see Step 6).
  **An epic-written intent file must carry an EPIC reference near the top.** It is what lets the conductor count live slots by EPIC (`/bip-conductor` Step 5) — the only fleet-side view that makes topic drift visible, since the conductor holds no topic boundary itself. If a brief arrives without one, ask the epic rather than inferring it from the issue; a guess defeats the check. Absence is also legitimate for a user-originated spawn, which is why the count reports `(no header)` as its own row rather than an error.

  **Match it tolerantly — `grep -iE '^\**EPIC\**:? *#?([0-9]+)'`, not a fixed literal.** Both `EPIC: 369` and `**EPIC**: #369` are in live use, and a strict pattern reports a header that is present as missing. Observed 2026-09-03: a `grep 'EPIC: *369'` returned no match against a body whose first line was `**EPIC**: #369`, and the conductor briefly reported the epic had skipped the convention it had just adopted. **A false negative here is worse than no check**, because it accuses the other side of a lapse.
- **No intent file** — compose the prompt from the issue directly. **This is a first-class path, not an escape hatch.** It covers a user-originated spawn (the user asks directly, often from a `/bip-ms` session where issues appear as the science moves), a conductor-initiated respawn, and routine maintenance. A user-originated issue **may belong to no EPIC at all, and that is normal** — never reject it, defer it, or route it to the epic for a membership ruling. See `/bip-conductor`'s "Two intake paths" for who owns scope on each.

If the intent conflicts with current fleet state, resolve it from measured state and say so in your report — a contended host or a taken clone is a placement decision, not a question for the user. Escalate only if either resolution risks an actual problem (see `/bip-conductor`'s "Arbitration").

## Workflow

### Prerequisite: Issue number required

Every spawn MUST target an existing GitHub issue.
If the conductor wants to spawn work that doesn't have an issue yet (reruns, follow-ups, quick experiments), file the issue first:

1. Write a minimal issue body (title + 3-sentence motivation + success criteria)
2. `gh issue create --title "..." --body-file ISSUE-*.md`
3. Then proceed with the spawn using the new issue number

Never write a spawn prompt with `issue=0` or without `/bip-issue-work <N>`.
Issueless spawns break EPIC tracking, PR linking, and conductor polling.

### Step 1: Select or create slot

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

**Clone mode** (`local_worktrees` absent or false):

If clone-name not specified, find an idle clone.
A good pick is on `main`, clean, and has no live tmux pane in its directory — the pool is shared across operators and finishing workers self-claim slots via `/bip-conductor-handoff`, so filter these out to avoid choosing one that's already taken:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
OCCUPIED=$(tmux list-panes -a -F '#{pane_current_path}' 2>/dev/null) || OCCUPIED=""
for name in $(jq -r '.clone_names[]' .epic-config.json); do
  dir="$CLONE_ROOT/$name"
  [ "$(git -C "$dir" branch --show-current 2>/dev/null)" = "main" ] || continue
  # Capture, then test readability separately: `git status | wc -l` or a bare
  # `-z` on failed output reports "clean" and "could not read" identically.
  status=$(git -C "$dir" status --porcelain 2>/dev/null) || continue
  [ -z "$status" ] || continue
  echo "$OCCUPIED" | grep -qxF "$dir" && continue   # live tmux pane here → owned
  [ -f "$dir/.epic-status.json" ] && continue         # a worker claimed it, unfinished
  echo "$name"
done
```

**Two fail-open hazards in that snippet** (the class is recorded in `/bip-epic`; see `18cc049`):

- **The `git status` test**: an unreadable clone yields empty output, exactly what a clean clone yields, so a bare `-z` selects a broken checkout as idle — hence the capture-then-test form above.
- **`OCCUPIED` is empty if `tmux` fails**, and an empty `OCCUPIED` makes every clone look unoccupied, including ones with a live worker. Selection is best-effort; `bip spawn`'s refusal to launch into a directory a live pane already occupies (Step 5; `--force` overrides) is the invariant. Do not invert that reliance by "improving" selection into a gate.

**Do not rank idle clones by build-cache size.** Cache size records *what has historically been built in that clone*, not whether the next build's hashes hit — and a worker's first act is usually to edit source, which invalidates everything downstream. Ranking by size is also a feedback loop: the biggest gets picked, grows biggest, gets picked again. (The cold-vs-warm gap on this repo is genuinely minutes against seconds — but no clone in a working pool is ever cold, so the choice is among degrees of warm that do not predict the next build.)

Measured 2026-09-01: clone caches ran 2.1G to 114G, **585G across 13 clones** on a disk at 50% with no routine pruning step anywhere, and the smallest cache (2.1G) shipped a merged PR that same day. A manual sweep hours later reclaimed **349G** (51% -> 31%) without touching anything a worker needed — the bulk of what the ranking treated as an asset was reclaimable garbage. `remote-gc` exists but its scope is shared-NFS compute hosts; a local workstation clone root is covered by nothing.

If clones are otherwise equivalent, pick arbitrarily; the tiebreak that *does* pay is avoiding a clone whose cache is pathologically large, since that is unpruned history rather than readiness. **Watch for cache-directory proliferation too** — subagent, lead and review runs create their own dirs (`.zig-cache-<clone>-bench`, `-lead`, `-lead5`, `-review`, `-safe2` all observed), and nothing removes them when the run ends.

**Workers do not reliably honour the `--cache-dir .zig-cache-<clone>` convention** — measured 2026-09-01, a live worker was building into its clone's *bare* `.zig-cache`.
So "a bare `.zig-cache` is a leftover" is false in general: a cache-pruning sweep must check liveness per directory rather than reasoning from the name.
When you do check, note that **`find -newermt -type f` cannot see a directory mtime**, and a directory's mtime changes when an entry is created *or unlinked* — so a transient temp file leaves a fresh `tmp/` with no fresh file anywhere. Two readers disagreed for exactly this reason on 2026-09-01: `-type f` said "nothing written today", `stat tmp/` said 02:57 today, and both were correct. Check files *and* directory mtimes, and treat "where a process ran" as distinct from "what it wrote" — a `/proc` cwd match is not evidence of a write.

Prefer clones with clean worktrees.
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

**Clone mode**:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
cd "$CLONE_ROOT/<clone>"
git checkout main && git pull --ff-only origin main
rm -f .epic-status.json .epic-worklog.md .claude/ralph-loop.local.md
```

**`.claude/ralph-loop.local.md` is the third stale-state file and the one that gets forgotten.**
It is the ralph-loop plugin's own state (iteration count, max, completion promise, and the `session_id` that owns it).
The hook exits on a `session_id` mismatch, so a file left by a dead session cannot actually drive a new worker — but it reads as a live loop to anyone inspecting the clone, including a conductor deciding whether a slot is busy.
Observed 2026-09-04: one clone in a 17-clone pool carried a state file from a session dead 3 hours, while every other signal said the slot was free.

**Fetch inside each clone, never once in the conductor.**
Each clone has its own `origin/main`, so a conductor-level fetch followed by `git -C <clone> reset --hard origin/main` resets the clone to *its own stale* ref and silently bases the worker on an old commit.
The `cd` above is what makes this correct — don't collapse it in a batch-spawn loop (bitten twice in 2026-08).
`.epic-status.json` and `.epic-worklog.md` are gitignored, so `reset --hard` preserves them.

**Worktree mode**: worktree was just created fresh from main — just clear any stale status files from a previous run on this same issue:
```bash
rm -f "$SLOT/.epic-status.json" "$SLOT/.epic-worklog.md" "$SLOT/.claude/ralph-loop.local.md"
```

**State cleanup is mandatory** — stale files from a previous assignment will confuse the worker and lead.

**Respawning into a *preserved* clone is the case this cleanup misses, and it silently defeats the spawn.**
Both blocks above assume a clean slate: a fresh assignment, where `git checkout main` discards the old work and removing both files is obviously right.
The dangerous case is the opposite one — a clone parked mid-issue whose branch and worklog you are deliberately keeping, because the whole point of the respawn is to resume that work.
There the instinct is to preserve everything, and the worklog *should* be preserved. **`.epic-status.json` must not be**, even then.
The reason is that the two files are different kinds of thing: the worklog is context, and the status file is **instructions**. Per the worker's own recovery protocol below ("Read `.epic-status.json` — current phase and lead guidance … If `lead_guidance` is set → follow it"), `lead_guidance` is consulted *before* anything in the launch prompt and outranks it.
So a clone parked at `needs-human` with `lead_guidance` reading "stand down, wait for #X to land" will stand the worker down on arrival, in the same breath as a fresh prompt telling it to start — and the more obsolete that guidance is, the more confidently it fires. Observed 2026-08-30: a clone carrying a stand-down that named two issues *which had both since landed* was one `rm` away from silently no-op'ing its own spawn.

```bash
rm -f "$SLOT/.epic-status.json"      # ALWAYS -- it is stale instructions, not context
# keep .epic-worklog.md when resuming: it is the history the worker needs
```

The symptom is near-invisible: the window opens, the worker reads its guidance, reports the parked phase, and stops. It looks like a worker that considered the task and declined.

### Step 2b: Pre-launch staleness check

Mechanical, topic-agnostic, and easy to skip under pressure — don't.
For every blocker/dependency the issue body names (an issue number, a PR number, "blocked on #N"), verify it's still true right now:

```bash
gh pr view <N> --json state,mergedAt   # merged already?
gh issue view <N> --json state          # closed already?
```

An issue that declares itself blocked on an already-merged PR is the single most common staleness bug measured in practice (multiple instances in three days).
If a named blocker turns out to be resolved, correct the composed prompt's `IMPORTANT CONTEXT` — don't pass the stale claim through to the worker.

### Step 3: Read the issue

```bash
gh issue view <number> --json title,body
```

Extract key context: what the issue asks for, data locations, phasing, dependencies.
If a `.spawn-prompts/` intent file exists (see "Where the prompt comes from" above, under either naming convention), this is a cross-check against the live issue body, not a replacement for reading it — the intent file may itself have gone stale since the epic wrote it.

### Step 4: Compose the prompt

The prompt has two parts: (1) the work instructions passed as the initial message to `claude` via `--prompt-file`, and (2) a ralph-loop invocation that the worker runs as its first action.
The ralph-loop prompt is kept SHORT (no special characters) — just a reminder to continue.
The detailed instructions are already in the conversation from the initial message.

The `IMPORTANT CONTEXT` section at the bottom is where the two sources combine: start from the epic's intent file when one exists, correct it per Step 2b, then append fleet facts only the conductor can see — which host/clone is actually free right now, a concurrent worker editing a file this issue also touches, a build in progress on a target remote host.
Without this annotation step those fleet warnings never make it into the prompt at all.

**Prompt file** (written by conductor to /tmp/spawn-N.txt):
````
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
3. **Run check_cmd once while the work is definitely unfinished and confirm
   it says not-done.** A probe you have never seen fail is not a probe.
4. Each ralph-loop iteration: run check_cmd, if not ready end the turn
5. After 3 consecutive check failures, set stop_reason to
   mechanical-blocker and invoke the lead

**`check_cmd` must be able to report not-done, and the ordinary idioms
silently prevent it.** A probe that exits 0 regardless either reports "done"
immediately or "still running" forever; the loop then advances on nothing or
spins, and in both cases the status file reads as though someone checked.

**This is a standing defect, not one bad day.** On `matsengrp/phyz`
2026-09-03 a completed grid (i2197) sat unseen because the worker's
`check_cmd` was fail-open; the next day a finished sweep sat unseen for 35
minutes, for the same reason.

Measured 2026-09-04 across two sweeps: **four of eleven live-slot probes
could not report not-done**, in two idioms -- `check_cmd: "true"`, and the
`... || echo NOT_READY` family (including `grep -c ... || echo 0`, where the
`grep -c` was **correctly** exit-coded and the `|| echo 0` had been added to
suppress noise). That last is the instructive case: a defensive habit
converting a working probe into a broken one, which is why "don't write
sloppy checks" does not reach it. **Two of the four appeared after the same
idiom had already been corrected in another slot, by message, hours earlier
the same day.** That is why this rule lives in the spawn prompt: the fix has
to be where the prompt is written, not in something each worker must
remember.

Prefer a command whose exit status *is* the answer -- `test -f`/`test -s` on
the artifact, or the tool's own status. If you must post-process, keep the
status (`set -o pipefail` -- a trailing `| tail`/`| head` otherwise
steals it; a bare `print()`/`echo` sets none). **Never wrap
the whole probe in `|| echo`.**

PUSH NOTIFICATION — When you set phase to needs-human or completed, read
`$CLONE_ROOT/.conductor-session` (resolve `CLONE_ROOT` from
`.epic-config.json` the usual way). If it exists, `SendMessage` that exact
address a one-line notification (issue number, phase, one-line summary)
so the conductor doesn't have to wait for its next poll cycle. Do NOT
`ListAgents` and pick a plausible-looking row yourself — session names
derive from working directory, so every other worker sharing your clone
root shares your name prefix, and guessing risks messaging the wrong
live session (see `/bip-conductor`'s Conventions section, "Completion
pushes"). If the file is missing or the send fails (address went stale
since the conductor's last refresh), skip silently — `.epic-status.json`
is written regardless, so `/bip-conductor-poll` and `bip epic watch`
remain the fallback. Do this EVERY time you write needs-human or
completed to .epic-status.json.

DO NOT EDIT THE EPIC ISSUE'S BODY. If you find it stale or contradicted
by your work — and you may well, since it is written ahead of results —
report that to the conductor (`$CLONE_ROOT/.conductor-session`) and let
the epic session fix it.

This is not territorial, and the reason is asymmetry rather than
ownership: the epic's push path captures the issue's `updatedAt` before
`gh issue edit` and aborts on mismatch, so it cannot silently clobber
your edit — but you have no such guard, so you CAN silently clobber its
edit, and neither of you would find out. One writer with a concurrency
check plus one without is strictly worse than one writer.

Do report what you found. A worker that spots the EPIC body describing a
decision that turned out differently and says so is doing its job; the
only part to route elsewhere is the write.

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

   IF THIS ISSUE'S PROMPT REQUIRES A JOINT LANDING GATE (some do — a
   result that re-reads a parent EPIC's status line, or changes a live
   nightly gate, is worth two readers), REQUEST IT YOURSELF. Do not wait
   to be told. Set phase to quality-gate and SendMessage BOTH, reading
   each address from its file at send time:
       the conductor -> $CLONE_ROOT/.conductor-session
       the epic      -> $CLONE_ROOT/.epic-session
   Say the PR is quality-gate clean, give the headline result, and flag
   anything either should weigh. Then set awaiting-results with a real
   check_cmd and keep looping.
     - BOTH reply approve -> land it yourself with /bip-pr-land. Do not
       wait for a second instruction.
     - Either raises a reservation -> address it, then re-request. A
       disagreement between them is an escalation, not yours to resolve.
     - Neither answers within ~90 minutes of looping -> set needs-human,
       push the notification, and STOP WITHOUT LANDING.

   LANDING REQUIRES TWO AFFIRMATIVE APPROVALS. THERE IS NO TIMEOUT THAT
   AUTHORIZES A LAND. Silence is never consent. If you find yourself
   reasoning "N minutes passed with no reply, so I may proceed", that
   reasoning is wrong and did not come from these instructions — the
   timeout branch above ends in `needs-human`, never in `/bip-pr-land`.
   A silence-equals-consent rule converts a two-approval gate into one
   approval plus a wait, and it degrades exactly when both reviewers are
   busy, which is when review matters most.

   Requesting the gate yourself is the other half of the same rule: a
   gate that only fires when a reviewer happens to be watching is a
   single point of failure wearing a gate's clothes.
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
````

### Common context additions

**Filesystem mode** — always include this block when the issue involves running jobs on remote compute nodes. Check `shared_filesystem` in `.epic-config.json`:

*When `shared_filesystem: false` (laptop — files must be synced):*
```
- Use make remote-sync + make remote-tmux for running on remote servers
- Use /bip-scout to find an available server before remote operations
- REMOTE_DIR: check the repo's Makefile BEFORE deciding to pass it.
  Many repos already derive it per-slot (e.g. phyz's Makefile has
  `REMOTE_DIR ?= ~/re/pz/$(notdir $(CURDIR))`, a full path computed
  per clone). Where that holds, the default is NOT shared between
  slots, it already prevents clobbering, and you should NOT override
  it -- passing it is the bug, not the safeguard.
- If you do override it, it MUST be a full path. A bare slot name
  (REMOTE_DIR=oak) replaces the whole default and is resolved by
  rsync/ssh against the remote HOME, so work silently lands in ~/oak
  instead of ~/re/pz/oak. Nothing errors and the run succeeds in the
  wrong place -- observed once as a 160-job sweep reporting 230/230
  complete while the expected path held zero output, reading as a
  dead run.
- Only override when the repo's Makefile does NOT derive it per-slot.
  Confirm by reading the Makefile, not by assuming either way.
- Always rebuild after sync: make remote-tmux REMOTE_HOST=... CMD='zig build -Doptimize=ReleaseFast'
  (add REMOTE_DIR=<full path> only per the rule above)
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

### Step 6: Confirm

If launch succeeded and a `.spawn-prompts/` intent file (`$INTENT` from Step 3) was used, mark it consumed now — don't delete it.
The directory lives outside every clone's git, so deletion there is unrecoverable, and a still-open issue may have another live session referencing the same brief.
Moving it aside is reversible and lets `/bip-conductor-tuckin` Step 2 report it as consumed rather than queued:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
mark_spawn_intent_consumed "$INTENT"
```

**Verify the worker actually started before reporting it live.** A prompt-ingested session and a working one are indistinguishable by context usage: **`context used N%` proves the prompt was read, not that work began.** Check for a created branch, or a tool call in the pane. Measured 2026-09-03: four spawns were reported as live for ~15 minutes while all four sat at a folder-trust dialog with their prompts queued, every one showing a plausible ~34% context.

Report to the user:
- Which clone was spawned
- Which issue it's working on
- Any phasing or gate criteria

If a persistent slot monitor is running (started by `/bip-conductor`), the conductor will receive automatic notifications when this worker changes phase.
No additional monitoring setup is needed.

If no monitor is running, suggest starting one or using `/loop 10m /bip-conductor-poll` to track progress.

## Creating new slots

**Clone mode** — create a new clone and register it:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
REPO=$(jq -r .github_repo .epic-config.json)
cd "$CLONE_ROOT"
git clone "git@github.com:$REPO.git" <new-name>
```
After creating, **four registration steps, and skipping any one produces a slot that fails silently**:

1. **Add the name to `clone_names` in `.epic-config.json`.** `bip spawn` does not consult the registry, so an unregistered clone spawns fine — but Step 1's idle-clone *selection* iterates `clone_names`, so the slot is free and **invisible to the conductor**. Measured 2026-09-03: three clones were created and only two registered; the third read as "no free slots" at the exact moment headroom was needed.
2. **Trust the directory before spawning into it.** A fresh clone has no `hasTrustDialogAccepted` entry in `~/.claude.json`, so `claude` opens on *"Quick safety check: Is this a project you created or one you trust?"* and **queues the prompt behind a modal dialog.** The window exists, the session is up, and no work starts. Either spawn once and answer the dialog, or confirm the entry exists first.
3. **Restart `bip epic watch`.** The watcher enumerates slots when it starts and does not rediscover the pool afterwards, so a clone added to `clone_names` after the watcher launched emits **no phase transitions at all** — the slot works, does real work, and is silently unmonitored. Kill and relaunch it (`ps -eo pid,args | grep -E '^\s*[0-9]+ bip epic watch'` to find it — **never** `pgrep -af`, which matches the literal string inside every worker's spawn prompt and dumps tens of KB). Measured 2026-09-03: four slots ran blind for most of a day because this step did not exist.
4. **Verify `.epic-config.json` actually resolves from the new clone, and fail loudly if it does not.** Every helper here calls `resolve_clone_root .epic-config.json` against the *current* directory, so the file must be findable from wherever the caller runs. A missing or unparseable file makes `jq -r .github_repo` print `null` and `resolve_clone_root` return empty — after which `cd "$CLONE_ROOT/<clone>"` resolves against `$HOME` and the work lands in the wrong place with **no error at any step**. Gate it explicitly rather than letting an empty value flow onward:

   ```bash
   CLONE_ROOT=$(resolve_clone_root .epic-config.json) || { echo "FATAL: cannot resolve clone_root" >&2; exit 1; }
   [ -n "$CLONE_ROOT" ] && [ -d "$CLONE_ROOT" ] || { echo "FATAL: clone_root '$CLONE_ROOT' missing" >&2; exit 1; }
   [ -d "$CLONE_ROOT/<new-name>" ] || { echo "FATAL: clone <new-name> not created" >&2; exit 1; }
   ```

   This is the same failure class as the `REMOTE_DIR` note below: a path component that silently resolves to something plausible is worse than one that errors, because the run succeeds in the wrong place and only a later `ls` of the expected path reveals it.

**If you must answer that dialog from the conductor, read which option is highlighted first — never send a blind `Enter`.** The default is not stable across windows: `grep -nE 'No, exit|Yes, I trust' ` the pane, then send `Down` before `Enter` when `No, exit` is the highlighted row. Measured 2026-09-03: a blind `Enter` intended to rescue four blocked workers selected `No, exit` in three of them and quit the sessions it was rescuing.

**Worktree mode** — no registration needed; worktrees are created on demand in Step 1 and named `issue-<N>`.
No config changes required.

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
