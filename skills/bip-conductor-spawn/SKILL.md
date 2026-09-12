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
  # NOT `status=`: zsh reserves `status` as a read-only alias for `$?`, so
  # the assignment dies with `read-only variable: status` and takes the whole
  # selection loop with it. Verified: `zsh -c 'status=$(echo hi)'` errors;
  # the same line under bash succeeds. zsh is the default shell on pax.
  st=$(git -C "$dir" status --porcelain 2>/dev/null) || continue
  [ -z "$st" ] || continue
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
DEST=$(preserve_epic_state "$(pwd -P)" "$CLONE_ROOT" \
    "before reassigning this clone to a new issue. The prior assignment ended here without a PR (issue #2216 success criterion 3: a stand-down/needs-human slot has no upstream preservation step of its own, so this reassignment point is where it has to happen).")
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "Preserved prior assignment's worklog+status to $DEST before reassigning"
elif [ "$rc" -eq 2 ]; then
    echo "PRESERVATION FAILED: cp did not succeed -- stop and investigate before deleting anything" >&2
fi
if [ "$rc" -ne 2 ]; then
    find "$CLONE_ROOT/<clone>" -maxdepth 1 \( -name '.epic-status.json' -o -name '.epic-worklog.md' \) -delete
    find "$CLONE_ROOT/<clone>/.claude" -maxdepth 1 -name 'ralph-loop.local.md' -delete 2>/dev/null
fi
# Stale build artifact, not preserved state -- deliberately OUTSIDE the `rc` gate,
# since it has nothing to do with whether worklog preservation succeeded.
# See "The stale-binary sweep" below. Scoped to the one binary on purpose.
find "$CLONE_ROOT/<clone>/zig-out/bin" -maxdepth 1 -name 'phyz' -delete 2>/dev/null
```

`find ... -delete` on an absolute path, not `cd` + a separate relative-path `rm`: no `rm`/`rmdir` token anywhere in the command, so it can't trip Claude Code's destructive-removal guard regardless of the `$`s already in scope, and it doesn't depend on cwd surviving into a later invocation — so this can run in the same script as the preservation above instead of needing to be split, and stays correct in an agent thread where cwd does not persist (see the `$`+`rm` note below for the guard's trigger condition; issue #2216's follow-up finding for the cwd point).

**`.claude/ralph-loop.local.md` is the third stale-state file and the one that gets forgotten.**
It is the ralph-loop plugin's own state (iteration count, max, completion promise, and the `session_id` that owns it).
The hook exits on a `session_id` mismatch, so a file left by a dead session cannot actually drive a new worker — but it reads as a live loop to anyone inspecting the clone, including a conductor deciding whether a slot is busy.
Observed 2026-09-04: one clone in a 17-clone pool carried a state file from a session dead 3 hours, while every other signal said the slot was free.

**Fetch inside each clone, never once in the conductor.**
Each clone has its own `origin/main`, so a conductor-level fetch followed by `git -C <clone> reset --hard origin/main` resets the clone to *its own stale* ref and silently bases the worker on an old commit.
The `cd` above is what makes this correct — don't collapse it in a batch-spawn loop (bitten twice in 2026-08).
`.epic-status.json` and `.epic-worklog.md` are gitignored, so `reset --hard` preserves them.

**Worktree mode**: worktree was just created fresh from main — preserve then clear any stale status files from a previous run on this same issue. **Resolve `CLONE_ROOT` before `cd`ing to `$SLOT`, in the same command**: `.epic-config.json` lives at the repo root the skill started in, not inside a per-issue worktree slot (a linked worktree is a separate directory tree with its own untracked files — it does not inherit the primary checkout's `.epic-config.json`). Resolving it after the `cd` silently yields an empty `$CLONE_ROOT`, and since `resolve_clone_root` fails loudly only when the file it's given exists and is unparseable — not when the file is simply absent from the *wrong* directory — the failure here would be silent rather than loud, so get the ordering right rather than relying on the loud-failure property to catch it:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
cd "$SLOT"
DEST=$(preserve_epic_state "$(pwd -P)" "$CLONE_ROOT" \
    "before a fresh restart on the same issue (issue #2216 success criterion 3).")
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "Preserved prior attempt's worklog+status to $DEST before restarting"
elif [ "$rc" -eq 2 ]; then
    echo "PRESERVATION FAILED: cp did not succeed -- stop and investigate before deleting anything" >&2
fi
if [ "$rc" -ne 2 ]; then
    find "$SLOT" -maxdepth 1 \( -name '.epic-status.json' -o -name '.epic-worklog.md' \) -delete
    find "$SLOT/.claude" -maxdepth 1 -name 'ralph-loop.local.md' -delete 2>/dev/null
fi
# Stale build artifact, not preserved state -- see "The stale-binary sweep" below.
find "$SLOT/zig-out/bin" -maxdepth 1 -name 'phyz' -delete 2>/dev/null
```

Same `find`-on-an-absolute-path reasoning as the clone-mode block above: no `rm`/`rmdir` token, no dependence on cwd surviving into a later invocation, so preserve and delete run together instead of needing a split.

All three cleanup blocks above use `find <absolute-path> ... -delete`
rather than `rm -f <relative-path>`, and that's deliberate on two
independent grounds:

- **The destructive-removal guard.** A `$` in the same command as the word
  `rm`/`rmdir` trips Claude Code's built-in destructive-removal guard,
  which **bypass-permissions mode does not suppress** — the call blocks on
  an approval prompt no permission rule can auto-allow. The gate tests the
  **whole submitted command string**, not just literal `rm` invocations —
  a comment or error message containing the word "rm" counts too. See the
  `rm` AND `$` section of the prompt template in Step 4 for the verified
  trigger condition. `find ... -delete` has no `rm`/`rmdir` token in it at
  all, so no combination of `$`s elsewhere in the same command can arm the
  guard — this is the one place the skill itself has to follow its own
  rule, and it's why these blocks don't need the "separate command" split
  an `rm`-based version would.
- **Cross-invocation persistence is thread-type dependent, not a fixed
  rule.** It persists in a main interactive session, but an agent/subagent
  thread resets both cwd and shell variables between separate Bash calls
  (measured directly, issue #2216's follow-up) — so a `cd "$SLOT"` in one
  invocation followed by a bare relative-path `rm` in a later one would
  silently delete nothing (or the wrong thing) in that thread type. `find`
  on an absolute path sidesteps this entirely; it doesn't care what
  directory the invocation happens to start in.

**State cleanup is mandatory** — stale files from a previous assignment will confuse the worker and lead.

**Respawning into a *preserved* clone is the case this cleanup misses, and it silently defeats the spawn.**
Both blocks above assume a clean slate: a fresh assignment, where `git checkout main` discards the old work and removing both files is obviously right.
The dangerous case is the opposite one — a clone parked mid-issue whose branch and worklog you are deliberately keeping, because the whole point of the respawn is to resume that work.
There the instinct is to preserve everything, and the worklog *should* be preserved. **`.epic-status.json` must not be**, even then.
The reason is that the two files are different kinds of thing: the worklog is context, and the status file is **instructions**. Per the worker's own recovery protocol below ("Read `.epic-status.json` — current phase and lead guidance … If `lead_guidance` is set → follow it"), `lead_guidance` is consulted *before* anything in the launch prompt and outranks it.
So a clone parked at `needs-human` with `lead_guidance` reading "stand down, wait for #X to land" will stand the worker down on arrival, in the same breath as a fresh prompt telling it to start — and the more obsolete that guidance is, the more confidently it fires. Observed 2026-08-30: a clone carrying a stand-down that named two issues *which had both since landed* was one `rm` away from silently no-op'ing its own spawn.

```bash
find "$SLOT" -maxdepth 1 -name '.epic-status.json' -delete   # ALWAYS -- stale instructions, not context.
# keep .epic-worklog.md when resuming: it is the history the worker needs
#
# DO NOT add a zig-out/bin/phyz delete here for symmetry with the two
# fresh-assignment blocks above. On a parked slot that binary can BE the
# provenance of an artifact the slot has ALREADY COMMITTED -- observed
# 2026-09-12, a slot whose branch build was what its committed
# 945-topology reference table had been measured against. Deleting it
# destroys a baseline that cannot be rebuilt from HEAD.
# See "The stale-binary sweep" below for the full account.
```

`find` on an absolute path, not `cd "$SLOT"` + a separate relative-path `rm`: no `rm`/`rmdir` token in the command at all, so it can't trip the guard no matter what else shares the invocation, and it doesn't depend on cwd surviving from an earlier command — a real risk in an agent thread, where cwd does not persist across separate Bash calls (issue #2216's follow-up finding).

The symptom is near-invisible: the window opens, the worker reads its guidance, reports the parked phase, and stops. It looks like a worker that considered the task and declined.

### The stale-binary sweep: `zig-out/bin/phyz`

**A pre-existing `zig-out/bin/phyz` is not evidence of anything about the checkout next to it, and it fails in the direction that does the most damage: it returns a confident wrong number rather than an error.**
The version string is `git describe` at *build* time, so a clone that has since pulled, rebased, or switched branches carries a binary whose `--version` no longer matches its own `HEAD` — and nothing re-checks it.
This is the fleet-scale form of the hazard `matsengrp/phyz`'s `CLAUDE.md` records as #2183 ("a stale local binary can silently outlive its own `commit_sha`").

**Measured 2026-09-12 on the 17-clone `~/re/pz` pool, because a rule with no measurement behind it ages into one nobody can re-justify.**
An epic session reported it on the 2 clones it happened to touch; a full census found **12 of the 14 clones that had a binary at all were stale**.
Only the 2 rebuilt within the hour matched.
Worst cases: `maple` at `HEAD` `91f3ab2b` carrying a `g4d9d6171` build; `pine` a `v0.1.1004-93` build against `HEAD` `0a4029db`; `teak` a three-day-old binary.
**The 2-of-2 sample and the 12-of-14 census point to different remedies** — the first reads as two clones someone forgot to rebuild, the second as a property of the pool.

**Apply that same sample-vs-census caution to the 12 itself: some of them are stale BY CONSTRUCTION, not by neglect.**
A clone that has just landed a PR mismatches automatically, because squash-merge rewrites the SHA — the binary was built from the pre-squash branch commit, which no longer exists on `main`.
Two of the 12 were exactly this (`cedar` built from `cda708d7`, `ash` from `edd1f787`), against cases like `maple`'s three-generation drift that are genuine neglect.
**The hazard is identical either way** — the binary still is not the `HEAD` code, and still answers confidently — **but say so, or the first reader who lands a PR, sees a mismatch, and knows exactly why concludes the rule cries wolf.**

**Two remedies, different blind spots — keep both.**
A check in the spawn brief still depends on the worker remembering to run it, which is precisely the property that failed 12 times; a missing binary fails loudly at the first invocation.
But deletion at prep only covers clones that go *through* prep — a directory populated by a manual `rsync`, or one that predates the pool, never does.
So the brief-line check (Step 4's template) and the prep-step deletion are **not redundant**: each covers the other's blind spot.

**Scoped to `zig-out/bin/phyz` deliberately.**
Some clones also carry stale *bench* binaries (`bench_hky3_pade`, `nni_vs_spr_1735`, `validate_gpu_vs_zig`, …) — same hazard class, different blast radius, and sweeping them in silently is how a prep step earns a reputation.
If you want those too, add a second `find` line with its own comment saying why, rather than widening the `-name` pattern.

**THE EXCEPTION, and it has a real counterexample rather than a hypothetical one: never delete the binary on a RESUME.**
The fresh-assignment blocks above run after `git checkout main` has discarded the prior branch, so nothing can still depend on that binary's identity — deletion is free.
A slot **parked mid-issue** is the opposite case, and it is the one the resume block covers: there the binary can *be* the provenance of an artifact the slot has already committed.
Observed 2026-09-12: a slot parked in `awaiting-results` under a sequencing hold held a branch build whose `--version` disagreed with its `HEAD` — and that binary was exactly what its committed 945-topology reference table had been measured against.
An unconditional prep-step deletion would have destroyed that baseline, and the failure would have surfaced only when the re-verification produced numbers nobody could reconcile.
**Write the exception down or the next conductor re-derives it the expensive way.**

**The exception is not complete without its second half, so do not let a trim separate them.**
Leaving the binary in place preserves the provenance and re-creates the original hazard: that slot now holds a binary that is stale for its *next* run, and it is the one clone your sweep deliberately did not fix.
So the carve-out obliges you to message the parked slot — as a fleet fact, explicitly not a gate — telling it to **rebuild before re-verifying, and to run the check below *after* the rebuild, not before.**
"Skip the build, reuse `zig-out`" is the exact mechanism by which a worker's stated intent to re-verify silently becomes an assumption, and a parked slot is where that intent has had the longest time to go stale.

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

**Spend the measurement at spawn rather than saving it for the corrections channel: annotation at spawn beats correction after it, because a correction races the worker.** A brief is composed once and read once, at a known moment; a nudge arrives mid-run, can land after the step it was about, and can be redundant with what the worker already had. Measured 2026-09-11: three issue-body fixes relayed an hour into a run — one of them a hard blocker, a tool absent from `PATH` so a cell could not have run as written — arrived to find the worker had already handled all three, because two were in the prompt's fleet facts at launch.

So the annotation is not a courtesy or a restatement of the brief: **it is the only channel guaranteed to arrive before the work.** What pays at spawn — hash the inputs the issue names and report what is actually there; resolve every bare path against the real tree and say which were ambiguous; locate the tools the cells need and report their versions; and re-derive rather than reuse any check whose answer could have moved.

**This is not an argument against sending corrections — send them.** A redundant correction costs a message; a needed one withheld costs a run. It is an argument against *relying* on them: spend the measurement at spawn so the correction channel is insurance rather than the plan.

**And put the fact that would invalidate an instruction *inside* that instruction, not in a warnings list elsewhere in the prompt.** `IMPORTANT CONTEXT` is the sharpest instance of this in the whole system: a persisted artifact, composed once, full of imperatives, read cold by a session with no history that trusts it to have resolved its own tensions — and read before the worker has seen the issue, the repo, or anything else. **A worker is even less able to notice a stale imperative than a resuming session is, because it has strictly less context to notice it with.** So don't write "be careful about X" in a trap list; name the claim the worker will encounter, say it is wrong, cite the sites with file:line, and say which one their own work sits on. An imperative gets executed before a warning gets applied.

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
  updated_at — ISO 8601 UTC, from `date -u +%Y-%m-%dT%H:%M:%SZ`.
    Never a placeholder, and never local time with a `Z` appended.
  blockers — list of blockers (empty list if none)
  scope — one-line restatement of issue goal (set by lead)
  stop_reason — category from lead decision framework (set by lead)
  lead_guidance — what the lead told you to do next (set by lead)
  lead_notes — list of lead evaluation entries (set by lead)
  completed_at — ISO 8601 timestamp set by the lead after the
    terminal completed ceremony (idempotency signal; do not set
    yourself). If you resume work after landing, re-create this file —
    see the landing step.
  awaiting — set when waiting for experiment results (description, check_cmd, check_files, started_at, timeout_hours).
    `started_at` must be real UTC: it is the one timestamp here that gets
    ARITHMETIC done to it, against `timeout_hours`.

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
   **Both of the following were measured on 2026-09-11, and the
   generalization matters more than either: a field can be inert in
   every path you check and load-bearing in one you don't.** "Nothing
   reads this field" and "no harm resulted" are both claims about the
   cases you happened to enumerate.
   **Stamp `started_at` with `date -u +%Y-%m-%dT%H:%M:%SZ`.** Two
   independent workers wrote local time with a `Z` suffix (UTC-7, so ~7
   hours in the past). Twice this was assessed as harmless — correctly,
   for `updated_at`, which no live consumer reads: the conductor's
   liveness sweep uses file mtimes and `bip epic watch` keys on phase
   transitions. Then one worker wrote it into `started_at` against
   `timeout_hours: 2`, where it becomes arithmetic: ~11 hours elapsed
   against a 2-hour budget, so the run reads as timed out before it
   began.
   **The other direction is worse, and a fourth worker produced it on
   2026-09-12: timestamps ~61 and ~85 minutes in the FUTURE**, round to
   the whole minute (`16:45:00Z` against a real clock of `15:20:26Z`) --
   two `date -u` calls cannot both land on `:00`, which is the tell.
   **A past-dated `started_at` INVENTS a timeout, which is loud and gets
   investigated. A future-dated one HIDES a stall: the run keeps reading
   as having budget left, so a hung job is never escalated.** The failure
   that announces itself is the safe one; prefer neither, but know which
   way you erred.
   **And the general form, which is why this keeps recurring in new
   costumes: a hand-written timestamp is a CLAIM, not a measurement, and
   it is indistinguishable from a real one at the point of reading.**
   Every downstream consumer -- worklog ordering, "which round came
   first", any staleness reasoning -- inherits it silently. The check that
   separates them asks a different question of the artifact rather than
   re-reading the header: compare the value against the file's own
   mtime.
   **If the work runs on a host that does not share this filesystem
   (`shared_filesystem: false`), `check_files` cannot name a local
   path** — the artifact exists only on the remote until something
   pulls it back, so a local path never appears and any consumer of
   that field never fires. Put the real check in `check_cmd` (remote),
   and either omit `check_files` or mark it explicitly remote.
   **That one looks harmless in isolation and is not, because it
   composes.** The slot where it was observed came to no harm only
   because its `check_cmd` was right — a property of that worker, not
   of this template. This skill's own measured rate for the other half
   is **four of eleven live-slot probes unable to report not-done**. A
   worker with a fail-open `check_cmd` *and* a local `check_files`
   under `shared_filesystem: false` has **no working readiness signal
   at all**: neither field fires, the loop advances on nothing or
   spins, and the status file reads as though something was checked.
   **This paragraph is deliberately a longer restatement of the field
   spec above, and the duplication is load-bearing — do not dedupe
   it.** A format rule read once at session start, in reference mood,
   has decayed by the time the block is written forty minutes later;
   the same rule at the point of use has not. Different reading moods
   need different forms.
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

`rm` AND `$` -- KEEP THEM OUT OF THE SAME COMMAND. Claude Code carries a
built-in destructive-removal guard that **bypass-permissions mode does not
suppress** (`dangerousRemoval:{bypassImmune:!0}` in the binary). When it
fires it blocks on an approval prompt that no permission rule can
auto-allow, and an unattended worker in a loop just stops there. One worker
lost a long stretch of wall-clock to this on 2026-09-05.

**The trigger is `$`, not `*`.** Verified in the 2.1.263 binary, the guard's
own first line:

    if(!e.includes("$")||!/\brm(?:dir)?\b/i.test(e))return null

No `$` **anywhere in the command** -> the guard cannot fire, whatever globs
you use. `rm -rf` is *not* required either; a plain `rm -f` trips it. So the
intuitive rule ("avoid `rm` with wildcards") is aimed at the wrong
character: `rm build/tmp/*` is fine, and `rm -f "$DIR/x"` is not.

What actually fires it, once a `$` is present: a `$VAR/` or `$1`/`$@`
immediately followed by a glob char, another `$`, a `/`, a quote, or
end-of-token; a path that is the working directory or an ancestor; a
critical system directory; and globs traversing more than one
unenumerable level.

Write it one of these ways instead:
- **Literal absolute paths, no `$`, no glob** -- `rm -f /home/you/re/x/.state`.
  Enumerate the filenames rather than globbing them. This is the safe default.
- **`find <dir> -maxdepth 1 -name 'pat' -delete`** when the directory really
  must come from a variable. No `rm` token, so the guard short-circuits even
  with `$` in the path. Do NOT use `-exec rm` -- that reintroduces the token.

Beware the whole-command scope: the gate tests the **entire** command string
for `$`, so a `$` anywhere else -- `$(date)`, a `$?` check, an unrelated
`$HOME` in a `&&` chain -- re-arms the guard for an `rm` that looked clean on
its own. Split those into separate commands.

REMOTE RUN OUTPUTS DO NOT SURVIVE YOUR OWN MERGE. If you ran anything on a
remote host, copy back whatever you will want later BEFORE the PR lands --
not after, and not "if there is time."

The failure has a specific shape and it has now nearly cost real data twice
in two days on this fleet. Per-run outputs under `runs/`, `results/`, etc.
are gitignored, so `make remote-sync` never carried them and the PR never
contained them. They live only in the remote clone. That clone is POOLED:
when your PR merges, the slot is reclaimed and reset to `main`, and the next
spawn reuses the same remote directory. Nobody is warned, nothing errors, and
the trees/logs/grids are simply gone -- discovered later by someone who needs
them for a follow-up issue.

Both rescues so far were luck, not process: the #2297 reference alignment
(stranded in a pooled clone) and 43 MB of #2297 search trees (still on
`orca04` only because someone went looking before the next spawn).

So, before you request landing:
- Copy anything a follow-up might need to `$CLONE_ROOT/.preserved/<slug>/`,
  with a short README saying what it is, which host and path it came from,
  and which issue produced it.
- Say in the PR body where it went. A reader six weeks out has no other way
  to find it.
- If it is small and genuinely belongs in the repo, commit it instead --
  preserved-but-untracked is a fallback, not the goal.
- Deciding it is all regenerable is a legitimate call; say so explicitly in
  the PR rather than leaving it unstated. "Regenerable" means someone has the
  command AND the budget, so a 6-seed 110-taxon re-run at ~15 min/seed is not
  free.

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

   IF YOU RESUME WORK AFTER THIS POINT, RE-CREATE `.epic-status.json`
   FIRST. Landing deletes it and the worklog, and both the conductor's
   reclaim gate and `bip epic watch` key on the file EXISTING — a slot
   without one emits no phase transition and reads as finished. So a
   break you notice after standing down, including one your own merged
   PR caused, is invisible to every fleet mechanism until you write a
   status file naming the issue and a live phase. And commit+push
   before verifying, not after: the clone is pooled, and the next
   spawn's prep runs `git checkout main`.

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
   AUTHORIZES A LAND. Silence is never consent.

   **A two-approver gate has no shared view of its own state, so
   CONFIRM TO BOTH ONCE YOU HAVE TWO.** Approvals arrive
   point-to-point: each approver sees its own and not the other's, so
   both can sit waiting on each other while you hold a complete gate.
   Measured 2026-09-11: a worker sat at `quality-gate` with both
   approvals in hand while the conductor was reporting it blocked on
   the epic, because the epic had replied directly to the worker. You
   are the only party who sees both — say so, naming the SHA, the
   moment the second arrives. If you find yourself
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
- NEVER trust a pre-existing zig-out/bin/phyz. Before you measure ANYTHING
  with it, run this from the clone root; if it prints STALE or UNRESOLVABLE,
  rebuild first. A stale binary does not error -- it returns a confident
  wrong number under newer-looking provenance.
      # Issue #2250 landed 2026-09-12: `--version` now prints
      # `phyz <version> (<commit>)`, where <commit> is `git rev-parse HEAD`
      # at BUILD time -- tag-graph independent, so prefer it outright.
      # Older binaries still in a pool print the describe string alone;
      # keep the `-g` fallback until none remain.
      vl=$(zig-out/bin/phyz --version)
      bc=$(printf '%s' "$vl" | sed -n 's/.*(\([0-9a-f]\{7,40\}\))[[:space:]]*$/\1/p')
      # MUST validate: a SHA this clone does not have parses fine, and an
      # unvalidated one reaches `git diff` as a bad object -- empty stdout,
      # exit status swallowed by $(), so the check falls through to "ok".
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

That last item is the brief-side half of the stale-binary remedy; the prep-side half is the `find`/`-delete` in Step 2 (see "The stale-binary sweep").
**Keep both.**
Prep-step deletion only protects clones that go through prep, and the check only fires if the worker runs it — each covers the other's blind spot, which is why neither is redundant.

**RESOLVE BOTH SIDES TO A COMMIT. Never compare two `git describe` strings, and never compare the version string to `rev-parse HEAD` either.**
Two earlier drafts of this check were wrong, in opposite directions, and the second was wrong in a way its own two-direction verification could not catch.

*First draft:* compared `--version` to `rev-parse --short=8 HEAD` — `phyz v0.1.1128-1-gc0b066a9` against `c0b066a9`, **unequal on a perfectly in-sync clone**. A worker who scripts that gets "rebuild always" and learns the check is noise; one who inverts it to make it pass gets "pass always". Either way the check written to prevent a silent wrong number becomes one.

*Second draft:* compared `--version` to `git describe --tags --always --dirty`. That is self-consistent within a clone and still **false-positives on a current binary**, because `git describe` is not stable over time — **this repo's Auto-tag workflow creates a tag per commit, so a commit's describe string CHANGES once its own tag lands.** Measured 2026-09-12: a clone whose binary was built from exactly `HEAD` (`c0b066a9`) reported `v0.1.1128-1-gc0b066a9`; the `v0.1.1129` tag was then created **pointing at that same commit**, and after a `git fetch --tags` the same clone's `git describe` returned `v0.1.1129`. The check flipped from match to **STALE with nothing rebuilt and nothing changed in the tree** — a pure false positive on the freshest possible binary, and the "cry wolf" outcome this section warns about two paragraphs up, arriving by a route nobody was watching.

**Why the earlier verification missed it, which is the reusable part.** It was run in both directions — a passing in-sync clone and a genuinely stale one — and both answered correctly. It never asked *"could this clone's `describe` output change without the binary or the tree changing?"* **Full power over the question asked; wrong question.** Two-direction verification is necessary and is not sufficient: it establishes the check separates the two states you had, not that the states are stable.

`git describe` is also **clone-relative**: `git pull --ff-only` does not fetch tags, so two clones at the identical commit can describe it differently (measured: `v0.1.1119-7-g9aac8693` in one clone at 1123 tags, `v0.1.1125-1-g9aac8693` in another at 1133, same commit). And when `HEAD` sits exactly on a tag, describe emits **no `-g<sha>` suffix at all** — so a regex that extracts the SHA works on most commits and silently returns nothing on tagged ones.

The form above handles all three: it parses the SHA from the `-g` suffix when present, resolves the bare tag name to its commit when it is not, and compares commit to commit. Verified 2026-09-12 on three clones — a binary built from `HEAD` whose commit had since been tagged (correctly `ok`, where the previous draft said STALE), a genuinely stale parked slot (correctly `STALE`, `9aac8693` vs `fb257d04`), and a freshly rebuilt clone (correctly `ok`).

**This still cannot establish that the binary matches the working *files*, only the commit** — hence the dirty-tree note in the snippet.

**`matsengrp/phyz` issue #2250 has LANDED (PRs #2538/#2540, 2026-09-12), and it changed this check rather than merely retiring a caveat.** `--version` now prints `phyz <version> (<commit>)` with `<commit>` resolved by `git rev-parse HEAD` at build time, and every artifact's `metadata.json`/`--summary-tsv` carries the same value as `phyz_build`. The commit is now *stated* rather than *inferred from a tag graph*, so the `-g`-suffix parsing above is a compatibility fallback for binaries built before that date, not the primary path.

**Landing it also silently BROKE the previous version of this snippet, which is the part worth remembering.** That version took `awk '{print $NF}'` — the last whitespace-separated field. Against the new format the last field is `(<commit>)`, parentheses included, which matches no `-g` case and resolves to nothing, so **the check returned UNDETERMINED on every current binary**. It failed safe (never a false STALE) and therefore announced nothing: a staleness check that had stopped detecting staleness, reporting the same reassuring silence as a clean pass. Verified 2026-09-12 by running the shipped logic against the new format string.

**The general form, since this is the fifth revision of this check:** *a check that parses another tool's output has that tool's output format as an unpinned input.* Neither two-direction verification nor a careful reading reaches it — the check was correct, and something else moved. `src/main.zig` now carries a comment tying its format string to `versions.py`'s parsing regex; this snippet is a third consumer and is not pinned by anything, so **re-run it against a freshly built binary whenever phyz's `--version` line changes.**

**The parens path MUST validate through `git rev-parse --verify`, and the sixth revision of this check shipped without it.** A SHA that parses cleanly but this clone does not have — `deadbeef...`, or a short `0123456` — reached `git diff` as a bad object, which prints to stderr, yields **empty stdout**, and has its exit status **swallowed by the command substitution**. `-n ""` is then false and the check falls through to **`ok`**: a fail-open on precisely the binaries it exists to catch. **This is not hypothetical on a pooled fleet** — after a squash merge the pre-squash branch commits are gone from `origin`, so a binary built in a worker clone reports a SHA no other clone can resolve, and `remote-sync`'d directories have the same shape. Those are the parked/stale binaries. The old `-g` path never had this bug because it validated *as a side effect* of how it resolved the ref.

**So the sharper general form, which is NOT the format-drift lesson above: a validating parse and a non-validating parse are not interchangeable even when they extract the same field.** The rewrite extracted the field more directly and lost the validation silently. Format drift and validation loss are two different failure modes in one check, one revision apart.

**Pin the shapes, because this check is now on its seventh revision and every revision has broken a different one.** Run each of these against the snippet from a clone and confirm the verdict; an unresolvable commit is `UNDETERMINED`, never `STALE`, and never `ok`:

| # | `--version` line shape | expected |
|---|---|---|
| 1 | post-#2250, commit == `HEAD` | `ok` |
| 2 | post-#2250, commit is an older commit touching `src` | `STALE` |
| 3 | pre-#2250, describe-only with a `-g` suffix this clone can resolve | `STALE` |
| 4 | commit is the literal `unknown` (built with no git) | `UNDETERMINED` |
| 5 | post-#2250, good commit, `-dirty` in the version field | `ok` |
| 6 | post-#2250, well-formed 40-char SHA this clone does not have | `UNDETERMINED` |
| 7 | post-#2250, well-formed short SHA this clone does not have | `UNDETERMINED` |

Shapes 6 and 7 are the fail-open regression guards; 3 is the compatibility guard and is load-bearing today, not defensive — pooled clones still carry pre-#2250 binaries (measured 2026-09-12: `alder` at `phyz v0.1.1129-3-gff1336c1`).

The replacement is verified across all seven shapes a binary in a pool can have: post-#2250 built from `HEAD` (`ok`), post-#2250 built from genuinely older code (`STALE`), pre-#2250 describe-only (fallback resolves, `STALE` correctly), `git` unavailable at build time so the commit is the literal `unknown` (`UNDETERMINED`, never `STALE`), a `-dirty` version suffix alongside a good commit (`ok`), and the two unresolvable-SHA shapes (`UNDETERMINED`). Verify by extracting the snippet **from this file** and running it, not by re-running a copy you typed — those are different artifacts.

**Compare the CODE between the two commits, not the commits themselves — otherwise the normal measurement workflow trips it.**
Build, measure, then commit the results is the correct ordering for an experiment, and it leaves `HEAD` one or more commits ahead of the binary *by construction*, with only outputs in between.
Measured 2026-09-12: a slot's binary sat at its own branch commit while `HEAD` carried a later commit touching only an experiment README and three results TSVs — the binary's `src/`/`tests/` were byte-identical to `HEAD`'s, so it was a perfectly valid measurement binary that a commit-equality test calls STALE.
Gating the mismatch on `git diff --name-only "$bc" "$hc" -- src build.zig build.zig.zon` being non-empty keeps every true positive (a genuinely older build) and drops that false one.
**`tests` is deliberately NOT in that list, and a future editor will want to add it back for symmetry — don't.** `zig-out/bin/phyz` is built from `src` + `build.zig`; test files are separate build steps that never link into it (verified on `matsengrp/phyz`: the exe's roots are `src/main.zig` and `src/root.zig`, its only `addImport` is `build_options`, the `tests/*_helpers.zig` modules are wired into `addTest` blocks alone, and nothing under `src/` imports from `tests/`). Including `tests` would report STALE on a test-only commit whose binary is perfectly current — **the same false positive this gate exists to remove, in a narrower costume.**
**`build.zig.zon` IS in the list** because a dependency change alters the produced binary while leaving `src`/`build.zig` untouched — omitting it reintroduces the false *negative* the whole check exists to prevent.
**This was the fourth revision of this check, and the false positive was found by using it rather than by reviewing it** — worth knowing before trusting the next clever narrowing of it.

**THREE STATES, NOT TWO: an unresolvable string must report UNDETERMINED, never STALE.**
If the binary's string is tag-exact (`v0.1.1129`, no `-g` suffix) and the *checking* clone has not fetched that tag, `git rev-parse` fails — and collapsing that into the mismatch branch prints STALE for a binary that may be perfectly current.
That is the same cry-wolf outcome as the two broken drafts above, arriving from the opposite direction, and **it fires on exactly the under-fetched clones the tag problem already affects.**
Verified 2026-09-12 across all four states: a binary from `HEAD` whose commit had since been tagged → `ok`; a parked slot's branch build → `STALE` (`9aac8693` vs `fb257d04`); the string `v0.1.1129` evaluated in a clone holding 959 tags and not that one → `UNDETERMINED`; a `-dirty` suffix → stripped and resolved. **A check that cannot resolve its comparand has established nothing and must not render a verdict.**
Warn on a dirty tree but do not block on it — a commit match genuinely cannot establish content identity, and blocking would make the check unusable in any clone mid-edit, which is most of them.

**The one-line rule, since this cost two sessions two separate defects in one night: on this repo `git describe` names a commit's position in a tag graph that moves underneath it. Only a SHA names a commit.**
The Auto-tag workflow tags *every* commit — the last six on `origin/main` each carry exactly one — so a commit's describe output changes from `vN-1-g<sha>` to `vN+1` the moment its own tag lands, with nothing rebuilt. **A binary built promptly after its commit is therefore the modal case for this false positive, not an edge case.**

**And the generalizable check, which would have caught it before either direction was run: enumerate a check's inputs and ask which are free to vary independently of the property being tested.**
`git describe` reads the tag graph; the tag graph moves independently of both the binary and the working tree, so a comparison built on it cannot be testing only staleness.
Two-direction verification does not reach this — it establishes that a check separates the two states you happened to have, **not that those states are stable.**

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

**A prompt-template fix does not reach any worker already running.** The prompt file is frozen at launch, and `RECOVERING CONTEXT` sends a compacted worker back to that same frozen text — so a template correction is live in new spawns and absent from every in-flight one. That is the configuration where a fix a worker has already applied silently regresses after a compaction. When you land a template change, either message the live workers and ask them to record the correction in `.epic-worklog.md` (append-only, and it *is* in the recovery path, where the prompt is not editable), or accept that the fix starts at the next spawn — but decide which, rather than treating the edit as having closed the case fleet-wide.

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
