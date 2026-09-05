---
name: bip-conductor-poll
description: Quick poll of GitHub activity and clone status since last check
---

# /bip-conductor-poll

Lightweight mid-session update for the fleet conductor.
Checks what changed on GitHub (as it bears on slots/clones) and in active clones since last check.
Use this instead of `/bip-conductor` when you already have context established.

Fleet-scoped, not topic-scoped: this skill reconciles slots against GitHub state and does housekeeping.
It does not update EPIC bodies — that's `/bip-epic`'s job, on its own cadence, driven by findings and completions rather than a poll timer.

For continuous monitoring, prefer `bip epic watch` (started by `/bip-conductor` Step 7).
It writes phase transitions to `.epic-notifications.log` in the conductor cwd; this poll skill reads new entries from that log to catch transitions the conductor may have missed.
Use `/bip-conductor-poll` for:
- Full GitHub reconciliation (merged PRs, new issues, comments) as it bears on slot/clone state
- Slot cleanup
- Catching up on log entries written while the conductor was idle

For periodic auto-polling: `/loop 10m /bip-conductor-poll`

## What to check

Four things stay in the primary because they're cheap and structured:

**Pull the conductor's own clone** — `git pull --ff-only origin main` in the conductor cwd.
`/bip-conductor` Step 2 pulls once at cold start and nothing pulls it again, so across a long session this tree silently rots while every GitHub answer stays correct.
**A role that only ever *reads* remote state has no natural moment that forces a pull**, and `git fetch` — which a conductor runs constantly to check merge state — updates remote-tracking refs and not the working tree.
Measured 2026-09-01: a conductor sat nine hours and four merges behind on `main`, clean tree and correct branch, and nothing surfaced it, because `git merge-base --is-ancestor <sha> origin/main` reads the fetched ref rather than the tree. Every merge verification it performed was right. The staleness produced no wrong answer, which is exactly why it survived.
(Two sessions hit this the same day; `/bip-epic` Step 1 carries the same instruction for the same reason.)

**Refresh the completion-push address** — resolve `CLONE_ROOT` and rewrite `$CLONE_ROOT/.conductor-session` with this session's current `ListAgents` name.
This is the file workers read to push a `needs-human`/`completed` notification without guessing among `ListAgents` rows — see `/bip-conductor`'s Conventions section ("Completion pushes") for why.
Cheap (one `ListAgents` self-lookup, one file write) and bounds how stale the address can get between conductor cold starts.

**Notifications log tail** — if `bip epic watch` is running, it appends one JSONL line per phase transition to `.epic-notifications.log` in the conductor cwd.
The state file `/tmp/.epic-poll-last-read` records the previous poll time as Unix seconds; pass the elapsed window to `--since`:

```bash
LAST=$(cat /tmp/.epic-poll-last-read 2>/dev/null || echo $(($(date +%s) - 3600)))
NOW=$(date +%s)
bip epic watch --since "$((NOW - LAST))s" 2>/dev/null
echo "$NOW" > /tmp/.epic-poll-last-read
```

Surface any `needs-human` / `completed` transitions prominently per the section below.

**Tmux window list** — `tmux list-windows -F "#W"` is one line of output and tells the subagent which slots to inspect.

Everything else is delegated.
Dispatch one `general-purpose` subagent for the combined poll (the poll is lighter than the cold start, so a single combined scan keeps round-trips down).
Follow the dispatch pattern in `SUBAGENT-SCAN.md` (bipartite repo root).
Brief:

> Delta poll for the EPIC conductor since the previous poll.
> Tmux windows currently open: `<list from tmux list-windows>`.
> Tasks:
>
> 1. `gh pr list --search "is:pr is:merged sort:updated-desc" --limit 5 --json number,title,mergedAt,body`.
>    For each new merge: note key results and whether it closes an issue.
> 2. `gh pr list --json number,title,headRefName,state`.
>    Note new PRs or CI status changes.
> 3. `gh issue list --search "sort:created-desc" --limit 5 --json number,title,state,createdAt`.
> 4. For each active slot (per the tmux list and any `.epic-status.json` in `<clone_root>`), check the latest issue-lead comment: `gh api repos/<owner>/<repo>/issues/<N>/comments --jq '.[-1].body'`.
>    Look for the `🤖 **Issue Lead**` prefix.
> 5. Inventory slots from `.epic-config.json`.
>    Clone mode: iterate `clone_names`.
>    Worktree mode: `find <clone_root> -maxdepth 1 -name 'issue-*' -type d`.
>    For each slot, read `.epic-status.json` and surface: phase, summary, scope, stop_reason, lead_guidance.
>    Migrate legacy phases: `blocked → needs-human`, `pr-review → quality-gate`.
> 6. For active slots, `git -C <slot> log --oneline main..HEAD | head -5` to see recent commits.
> 7. For slots that look finished or blocked, capture the last 20 lines of tmux: `tmux capture-pane -t <window> -p | tail -20`. **Do not read text at the `❯` prompt in that output as pending user input** — `-p` strips escapes, and Claude Code's autosuggest puts dim placeholder text at the prompt of essentially every idle window. See `/bip-conductor` Step 6 for the cursor-position test that distinguishes them.
> 8. Verify state with `gh pr view --json state` / `gh issue view --json state` for anything you plan to flag — never claim "open" or "merged" without a live confirmation.
>
> Return under 400 words, structured per `SUBAGENT-SCAN.md`:
> - `changes_since_baseline`: merged PRs, new issues, new issue-lead comments, slot phase changes
> - `active_items`: per active slot — clone, issue, phase, stop_reason, lead assessment (one line each)
> - `action_candidates`: pending spawn intent directly in `$CLONE_ROOT/.spawn-prompts/` (either `<N>.md` or `spawn-<N>.txt` — check both) waiting to be executed — ignore anything under `.spawn-prompts/consumed/`, that's already-launched intent, not pending; merged PRs that should trigger slot cleanup.
>   (Deciding *which* open issues are ready to spawn is `/bip-epic`'s call, not this poll's — report raw signal, not a readiness verdict.)
> - `surprises`: `needs-human`/`completed` slots, stale status files, contradictions, `RECOMMEND DEEPER LOOK` flags. Include the liveness sweep's result: any slot whose `.epic-status.json` mtime is >45 min old, and whether its `.epic-worklog.md` is also stale (stalled) or fresh (alive but not reporting). Also flag any `phase` value outside the seven spec-valid ones — an off-spec string silently escapes `bip epic watch`'s `--phases` filter.

If the report has zero `changes_since_baseline` and zero `surprises`, the poll output is one line: "All quiet."

## After polling

### Focus on what matters

**Lead with pending spawn intent** — `.spawn-prompts/` files (either naming pattern) the epic has already written and is waiting on the conductor to execute.
This is the most actionable information at this level; deciding which *other* open issues should be spawned next is `/bip-epic`'s call.

**Surface lead evaluations** — if a clone's status shows a recent lead evaluation (stop_reason set, lead_guidance present), mention the lead's assessment briefly.
This tells the conductor what the workers are doing without having to read full issue comments.

**Flag needs-human and completed** — if any clone has `phase: "needs-human"` (or legacy `blocked`) or `phase: "completed"`, highlight it prominently.
These require conductor attention.
Read `$CLONE_ROOT/.epic-session` and, if present, `SendMessage` that address the issue number and phase so `/bip-epic` can re-poll and update the EPIC body without waiting for its own cadence.
If the file is absent or the send fails, this is a no-op — `/bip-epic`'s independent `gh` polling remains the fallback path.

**Only report active clones** — clones with a tmux window that are actually doing something.
Don't list completed or idle clones; that's noise.
Completed clones can be mentioned briefly ("fir completed i374") but don't need a table row.

**Mention recent merges** only if they unblock something or change the plan.

**Correcting a live worker mid-flight**: `ListAgents`/`SendMessage` reach other local Claude tmux sessions and can push an immediate correction without waiting for the worker's next lead evaluation or killing its tmux window.
Send the correction directly — stating what changed in at least one line — rather than a bare pointer, which is a no-op for a worker that already reads the status file every step and strips the priority signal.
If a correction is long enough that pointing at a file seems preferable, its home is the spawn prompt or the issue body, not a nudge.
The durable/transient test is the deliverable: any nudge that changes what the worker will produce (scope, target, artifact, gate criterion) MUST also get a durable record, written by the **conductor** appending a timestamped, attributed entry to `.epic-worklog.md` in the same step — not `.epic-status.json`, since that file is the worker's own continuously-rewritten object and a second writer there is unsynchronized and gives no tiebreak against `lead_guidance`.
Facts that leave the deliverable unchanged (host load, a peer's timing, a dependency that just landed) may be message-only.
Delivery lands at the worker's next tool call, not instantly. Fall back to the file-only correction (a `conductor_guidance` field or a `lead_notes` entry tagged `source: conductor` — never `lead_guidance`; wait for the worker's own loop) only once you have established the target is genuinely unaddressable, per the rule immediately below.

**Addressing — read this before any `SendMessage`.**
Read the address off `ListAgents`' own row, or off the message you are replying to. **Never compose it.** Worker names are `<clone>-<suffix>` (`fir-b3`, `teak-37`, `spruce-66`); the bare clone name and the tmux window name are both *not* addresses. **A failed send to a name you typed yourself is a typo, not a channel limit** — re-run `ListAgents` and use the exact string. Never conclude from one failed send that workers are unreachable. (An address that came from a self-registration file and stops working is the different case: it drifted, so skip silently and don't hunt a substitute.)
Skip the nudge entirely for a worker in `awaiting-results` with a live `check_cmd`; use `notify_when_idle: true` instead of "tell me when this worker finishes" — main-conversation only, this-machine only, one-shot.

### Output structure

1. **Pending spawn intent**: `.spawn-prompts/` files (either naming pattern) waiting on execution, plus idle clones available to run them.

2. **Active work**: Clones with tmux windows that are mid-task.
   One line each: clone, issue, phase, stop_reason (if set), lead assessment.

3. **Needs human**: Clones in `needs-human` phase — show the lead's assessment and what decision is needed.

4. **Decisions and negative list**: read new entries from `.epic-decisions.md` in the conductor cwd since the last poll (`FINAL` relays, forwarded worker findings, negative-list entries — see `/bip-conductor`'s Conventions, "Decision relays" and ".epic-decisions.md: the durable fleet-decision log") and report them here; this is the file's canonical output surface for a poll cycle, not a fresh derivation.

5. **Recently landed** (brief): PRs merged since last poll, only if noteworthy.

6. **Execute pending spawns**: If spawn intent files and idle clones both exist, run `/bip-conductor-spawn` for them and report after — placement and timing are the conductor's call, not a question to hold the fleet on (see `/bip-conductor`'s "Arbitration"). Escalate only a scientific question, or a risk of an actual problem: data loss, a clobbered checkout, two slots on one deliverable. Rebase friction is not that.
   **A busy fleet is fine as long as every slot is on topic — the gate is topic, not count.** Spawn ready, in-scope, unblocked briefs while capacity exists; don't throttle on volume. Whether an issue belongs to the programme at all is the originator's call (the epic agent for its own EPIC, the user for a direct or `/bip-ms` request) — but once it is in scope and ready, holding it back is a failure, not caution.

### Housekeeping (do silently, don't report unless problems)

This is the ongoing cleanup that keeps slots current between cold starts.
Do it every poll cycle — don't wait for `/bip-conductor`.
EPIC body content is not this skill's concern; `/bip-epic` keeps bodies current on its own cadence.

#### Slot cleanup for merged PRs

For each slot whose PR has merged (cross-reference merged PRs from check 1 with slot branches):

**Worktree mode**:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
# Confirm PR is merged before removing
gh pr list --head <branch> --state merged --json number | jq length
# If merged:
git worktree remove "$CLONE_ROOT/issue-<N>"
git branch -d <branch>
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation (e.g. `/home/user/.claude/skills/bip-conductor-poll`); the shared helper lives at `lib/spawn-intent.sh`, a sibling of every skill directory (see `skills/lib/spawn-intent.sh` in the `bipartite` repo).

**Clone mode**:
```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
git -C "$CLONE_ROOT/<clone>" checkout main
git -C "$CLONE_ROOT/<clone>" pull --ff-only origin main
rm -f "$CLONE_ROOT/<clone>/.epic-status.json" "$CLONE_ROOT/<clone>/.epic-worklog.md"
```

**Before returning a clone to the pool, check what untracked output it is carrying.**
Reclaiming is safe for tracked files and **silently destructive for untracked ones** — and untracked is where experiment output lives *by policy*. Nothing in the reclaim removes it, so it survives until the next spawn into that clone quietly destroys it. **A free clone is not an empty clone.**

**The same trap catches `.epic-worklog.md`, and it is gitignored rather than untracked — so a `git status` check does not show it.** A slot that lands a PR needs nothing; a slot that **stands down or escalates** carries its entire value there, and `/bip-conductor-spawn`'s prep deletes it on the next assignment. Copy it to `$CLONE_ROOT/.preserved/<slug>/` before reclaiming. Measured 2026-09-03: eleven were rescued in one stand-down, the largest 269 lines from a slot that wrote no code at all.

```bash
# gitignored OR untracked output under experiments/*/results
git -C "$CLONE_ROOT/<clone>" status --porcelain --ignored=matching -- 'experiments/*/results' \
  | awk '$1 == "??" || $1 == "!!" {print $2}'
```

**`--ignored` is not optional, and a plain `du` is the wrong tool — both mistakes were made writing this rule.** Experiment output is usually *gitignored*, not merely untracked, and **`git status --porcelain` does not show gitignored paths at all**: run against the clone that nearly lost its sweep, plain `status` on that directory returned **0 entries** while `--ignored=matching` returned 7. That silence is exactly how it went unnoticed. Conversely a bare `du` cannot tell tracked from ignored — measured on another clone, it reported 71M under `experiments/*/results` that was entirely *tracked reference data*, i.e. safe and already in git. Size is not the signal; reachability by `git` is.

If a directory turns up, the question is not whether it looks disposable — it always does — but **whether anything cited points into it**. Check the open issues and PRs for the experiment's name before reusing the clone. If something cites it, copy it to `$CLONE_ROOT/.preserved/<name>/` with a README saying what cites it and why it is not committed; that path is outside every clone's git tree, so no `git clean` can reach it.

**A rescue is not finished until something *in the repo* points at it.** `.preserved/` is a dotfile directory in a non-git clone root — exactly as invisible to a reader as the untracked output it rescued, and invisible for the same reason: the explanation sits somewhere the reader never goes. Measured 2026-09-02, hours after the first rescue: `grep -rn '\.preserved'` across the consuming repo returned **nothing**. So a rescue has three parts, not two — copy the data, write the local README, **and land a committed pointer** where someone hitting the gap would actually look (the experiment's own `README.md`, next to the numbers whose source went missing). Issue bodies are not sufficient: they reach whoever works those issues and nobody else. Without the third part `.preserved/` just accretes rescued directories that nothing references, which is the original failure relocated one level up.

Measured 2026-09-02: a clone was returned to the pool still holding the **only** copy of a 391-job sweep's output — including the exact file a live issue cited twice in its load-bearing argument, on a gitignored path absent from the repo. The next spawn into that clone would have destroyed it, and nothing in the reclaim would have warned anyone. The tell was never the clone's identity; it was that an open issue's citation resolved to a path only that clone had.

Also clean up stale slots: no tmux window AND `.epic-status.json` older than 30 minutes.
Same cleanup as above.

#### The liveness sweep — run this every poll, it is the only stall detector

`bip epic watch` reports transitions, so a slot that stops transitioning is invisible to it and nothing else detects that. A stale status file means two opposite things depending on whether a tmux window is open, so the window check belongs *inside* the sweep. Same 30-minute rule as above, extended to the window-open case rather than a second competing rule:

```bash
for f in "$CLONE_ROOT"/*/.epic-status.json; do
  [ -n "$(find "$f" -mmin +45 2>/dev/null)" ] || continue
  d=$(dirname "$f")
  if tmux list-panes -a -F '#{pane_current_path}' 2>/dev/null | grep -qxF "$d"; then
    echo "STALLED?  $d — window OPEN, never clean up; check the worklog next"
  else
    echo "ABANDONED $d — no window, cleanup candidate per the rule above"
  fi
done
```

For anything the sweep marks `STALLED?`, check its worklog mtime to tell a stall from a quiet worker:

```bash
find "$CLONE_ROOT/<clone>/.epic-worklog.md" -mmin +45   # empty output = worklog is fresh
```

- **status stale + worklog stale** → *candidate* stall only. **Run the `ps` check below before calling it one — live compute vetoes a stall verdict outright.** If nothing is running, escalate to the user. Never clean up either way: the window is open and may hold typed human input.
- **status stale + worklog fresh** → alive and working; the status file is simply lying. Not a stall — nudge it to resume writing status, and don't report it as dead.

**The `ps` leg is a veto on both branches, not a tiebreak on one.** Run it before reporting either verdict:

- **Compute running → not stalled, whatever the mtimes say.** A worker in a quality-gate loop runs multi-minute builds and full suites and writes no worklog entry for hours. Measured 2026-09-01: a slot **243 minutes stale on status and 234 on worklog** — both legs firmly "stalled" — was at that moment running `zig build-exe` at **99.9% CPU** under its own cache. Reporting it dead would have been exactly wrong, and it held the PR gating the whole queue.
- **No compute → now the mtimes decide**, per the two branches above.

**And on the other branch, a fresh worklog is necessary and not sufficient.** "Worklog fresh" bounds how long ago the worker last *thought*, not whether its background job is still alive — a worker can write a perfectly accurate "waiting on the sweep to finish" entry and then wait forever on a job that already died. Measured 2026-09-01: a slot passed the worklog check, and its pane said "3 shells still running", while `ps` showed no process at all under its clone.

```bash
ps -eo pid,args | grep -F "$CLONE_ROOT/<clone>" | grep -v grep   # any compute still alive?
```

If the worklog says it is waiting on something and no process matches, it is stalled regardless of the mtimes: tell it the job is gone, and where its output actually did or did not land. Do not trust a pane's "N shells still running" indicator — it goes stale when the session idles.

Use mtimes, not the `updated_at` field: a worker that writes a placeholder timestamp defeats the field but not the mtime (measured: a status file 6h old and frozen on `phase: exploring` while its worklog had been written 4 minutes earlier).

See `/bip-conductor`'s `.epic-status.json` spec for the `phase`-is-not-evidence corollary.

If a merge closes an issue tracked in an EPIC, that's signal `/bip-epic` needs, not something to act on here — surface it under `changes_since_baseline` and move on.

#### Filter and route fleet-level findings

Before recording anything anywhere, run each candidate fleet-level finding through this filter (same as `/bip-conductor-tuckin` Step 3):

1. **Is it derived?**
   Recomputable from `git`, `tmux`, `gh`, or the filesystem — record nothing.
   Host quirks (which host had a warm cache, which was mid-build) are derived and go stale within hours; don't write them down anywhere, including MEMORY.md.
2. **Is it already recorded?**
   A finding that produced an issue, PR, test, or doc needs no second copy.

Only what survives both gates gets a destination: a durable clone-pool layout decision → a `CLAUDE.md`; a workflow rule → a skill; a **fleet-level decision** (a `FINAL` relay, a forwarded worker finding, a negative-list entry) → `.epic-decisions.md` in the conductor cwd, not `/bip-epic`'s own memory — that memory is unavailable wherever the auto-memory directory is deliberately skipped (`bip-epic/SKILL.md`'s Step 1 explicitly allows this, and some setups do it), and `.epic-decisions.md` has no such gap since it's a plain file, not a memory-directory feature.
Nothing else fits → the poll report itself.

## Conventions

Same as `/bip-epic`: `iN`/`pN` prefixes, full URLs on first mention.
Tmux windows named `NNN-YYY` where NNN is the issue number and YYY is the clone/slot name (e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).

**Decision relays are marked `PROVISIONAL` or `FINAL`, nothing unmarked.**
When this conductor session is the one talking to the user mid-poll and a decision results, push it to `/bip-epic` via `$CLONE_ROOT/.epic-session` prefixed `PROVISIONAL` or `FINAL` and append it to `.epic-decisions.md` — see `/bip-conductor`'s Conventions, "Decision relays: PROVISIONAL and FINAL", for the full mechanics; this is the same rule, restated here because a conductor mid-cycle is reading this skill, not the cold-start one (the durable/transient nudge test above duplicates the same way).

**Resolve a citation before acting on it, here too.**
A worker's finding forwarded mid-poll carries the same citation risk as at cold start — a bare filename with no directory is unresolved, and reconstructing one from context produces a real `find` result for an invented question. See `/bip-conductor`'s Conventions, "Resolving a citation before acting on it", for the full mechanics; restated here for the same reason as the decision-relay rule above.

## Layout config (issue #149)

`.epic-config.json` keeps working.
The newer global `layout:` block in `~/.config/bip/config.yml` configures worktree mode for non-EPIC `bip spawn`; see `docs/guides/layout.md`.
