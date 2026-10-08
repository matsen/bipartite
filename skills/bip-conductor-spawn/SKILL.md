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

`<this-skill's-base-directory>` is this skill's base directory as given at invocation; `lib/spawn-intent.sh` is a sibling of every skill directory.

- **Intent file present**: it is the base for the `IMPORTANT CONTEXT` section of Step 4's prompt — don't re-derive what it already says.
  Check it against live fleet state (Step 2b) and append fleet facts it structurally couldn't know: which host/clone is actually free, a concurrent worker editing an overlapping file, a build running on a target remote host.
  Mark it consumed after a successful launch (Step 6).
  An epic-written brief carries an `EPIC:` reference near the top; the conductor counts live slots by it. Match it tolerantly — `grep -iE '^\**EPIC\**:? *#?([0-9]+)'` — since both `EPIC: 369` and `**EPIC**: #369` are in use. If an epic brief lacks one, ask the epic rather than inferring it. A user-originated brief legitimately has none.
- **No intent file** — compose the prompt from the issue directly. This is a first-class path: a user-originated spawn, a conductor-initiated respawn, routine maintenance. A user-originated issue may belong to no EPIC; never reject or defer it for that. See `/bip-conductor`'s "Terms and intake".

If the intent conflicts with current fleet state, resolve it from measured state and say so in your report — a contended host or a taken clone is a placement decision. Escalate only if either resolution risks an actual problem (see `/bip-conductor`'s "Arbitration").

## Workflow

### Prerequisite: Issue number required

Every spawn targets an existing GitHub issue; issueless spawns break EPIC tracking, PR linking, and slot monitoring. For work with no issue yet (reruns, follow-ups, quick experiments), file a minimal one first (title, 3-sentence motivation, success criteria) through `/bip-issue-check`.

### Step 1: Select or create slot

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

**Clone mode** (`local_worktrees` absent or false):

If clone-name not specified, find an idle clone: on `main`, clean, no live tmux pane in its directory, no `.epic-status.json`, no hold:
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
  [ -f "$CLONE_ROOT/.holds/$name" ] && continue       # a conductor hold (bip-conductor Step 4)
  echo "$name"
done
```

If `tmux` fails, `OCCUPIED` is empty and every clone looks free. Selection is best-effort; `bip spawn`'s refusal to launch into a directory a live pane occupies (Step 5) is the invariant.

If all are busy, offer to create a new clone using a name from `new_clone_names` in the config.

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

**Stale binaries.** A stale `zig-out/bin/phyz` returns confident wrong numbers. The prep-step delete covers fresh assignments; the check in `context-additions.md` ("For code changes") covers the rest.

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

The `IMPORTANT CONTEXT` section at the bottom is where the two sources combine: start from the epic's intent file when one exists, correct it per Step 2b, then append fleet facts only the conductor can see.
Spend the measurement here rather than in later corrections, which race the worker: hash the inputs the issue names, resolve every bare path against the real tree, locate the tools the work needs and report their versions.
Put a fact that invalidates an instruction *inside* that instruction (claim, why it's wrong, file:line), not in a separate warnings list.

Re-read every issue-specific *sentence* when reusing a previous prompt, not just every issue number — a substitution on the number cannot reach a sentence that describes the other issue's work without naming it.

The `sed` below writes the literal absolute `CLONE_ROOT` into the brief. Workers' clones have no `.epic-config.json`, so a worker cannot resolve it.

Fill in four lines every time:

- **`EPIC:`** — `<owner/repo>#<N>` from the brief's `EPIC:` header, cross-repo included, or `none` for work under no EPIC. A follow-up takes its parent issue's. The worker reads the EPIC body's `Owner:` at send time, so freeze the EPIC here, never its owner.
- **`NOTIFY:`** — the exact `ListAgents` names of the sessions that requested or framed the issue, or `none`. With `EPIC: none`, these and the conductor are all the routing the slot has.

- **`LANDING DELEGATION:`** — the landing rule as it applies to this repo (`/bip-conductor`'s "Who may land"): the self-contained allowlist, which lands after the epic's claims check at the head SHA, and what shared code needs here (a named group member's review, or agent approval), plus any exception from the conductor's decisions log with its date. The worker classifies its own diff against the allowlist; a file outside it means shared code, and without that review the worker stops at a clean gate. Workers cannot read the decisions log, so it travels in the brief or not at all.
- **`JOINT LANDING GATE: YES | NO`** — YES only when landing is hard to reverse. Not for size, risk, or a shared file (that is sequencing). A hold stalls a finished slot invisibly, so YES needs a reason. For everything else, NOTIFY the owning session when the PR opens; that is detection, not prevention.

For an epic-reserved artifact, the gate is the shape of the edit: additive-only → land and notify; any removal or alteration of epic-recorded text → announce intent-to-land and wait for an ack. Check with a three-dot diff; a two-dot diff against `origin/main` reports everything landed since the fork as deletions:

```bash
git diff origin/main...HEAD -- <the owned files>
```

**Prompt file.** The brief is `worker-brief.txt` in this skill's directory; it ends at `IMPORTANT CONTEXT:`. Fill it with `sed` and append the context with a heredoc. Don't Read the template to fill it: `sed` fills it without loading its ~370 lines into your context. Escape `&`, `|` and `\` in the substituted values.

```bash
B="<this-skill's-base-directory>"
sed -e "s|{{N}}|<N>|g" -e "s|{{TITLE}}|<title>|g" -e "s|{{CLONE_ROOT}}|$CLONE_ROOT|g" \
  -e "s|{{EPIC}}|<owner/repo#N or none>|" -e "s|{{NOTIFY}}|<exact session names or none>|" \
  -e "s|{{LANDING_DELEGATION}}|<the landing rule for this repo, plus any dated exception>|" \
  -e "s|{{JOINT_LANDING_GATE}}|<YES or NO>|" \
  "$B/worker-brief.txt" > /tmp/spawn-<N>.txt
cat >> /tmp/spawn-<N>.txt <<'BRIEF'
<issue-specific context: the intent file's brief corrected per Step 2b, then fleet facts — data locations, phasing, remote execution, dependencies, key files>

Now read the issue and begin work:
/bip-issue-work <N>
BRIEF
grep -n '{{' /tmp/spawn-<N>.txt   # must print nothing
```

To check what the brief tells workers on one topic, grep it (`grep -n -A8 'DEFERRAL RULE' "$B/worker-brief.txt"`) rather than reading the whole file.

**Context additions.** Standard snippets for the context section are in `context-additions.md` in this skill's directory: filesystem mode (always, when the work runs jobs on remote nodes; chosen by `shared_filesystem`), experiments, the own-argv `pgrep` warning (next to any build or run command), build-system targets, code changes (including phyz's stale-binary check), and phased work. Read it when the issue calls for one of them.

### Step 5: Launch tmux window

`bip spawn` refuses a directory a live tmux pane already occupies (`refusing: a tmux pane is already live in <dir>`); pick another idle clone. Pass `--force` only when you deliberately want a second session there.

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)

# Clone mode: --name is NNN-clone (e.g. "281-cedar")
bip spawn --prompt-file /tmp/spawn-<N>.txt \
  --dir "$CLONE_ROOT/<clone-name>" \
  --name "<N>-<clone-name>"

# Worktree mode: --name is NNN-issue-NNN (e.g. "281-issue-281")
bip spawn --prompt-file /tmp/spawn-<N>.txt \
  --dir "$CLONE_ROOT/issue-<N>" \
  --name "<N>-issue-<N>"
```

Always use `--prompt-file`, never `--prompt "$(cat file)"`, which zsh expands. Never launch with raw `tmux new-window` / `send-keys` / `claude`.

`bip spawn`'s behaviour comes from the installed binary, not from the bipartite source. Before relying on a recent `bip` change, confirm the binary has it (`strings "$(command -v bip)" | grep -F '<new string>'`) and rebuild if not.

### Step 6: Confirm

If launch succeeded and an intent file (`$INTENT` from Step 3) was used, mark it consumed now — don't delete it, and verify the move, since an unmoved brief reads as pending forever:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
mark_spawn_intent_consumed "$INTENT"
ls "$CLONE_ROOT/.spawn-prompts/consumed/<N>.md"
```

Deletion there is unrecoverable (it is outside every clone's git); the move lets `/bip-conductor-tuckin` report it as consumed.

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

A brief or gating-rule change does not reach running workers; the prompt is frozen at launch and `RECOVERING CONTEXT` re-reads it. Either message live workers to record the correction in their worklog, or accept that it starts at the next spawn — decide which.

**Verify the worker actually started before reporting it live.** `context used N%` proves the prompt was read, not that work began; a session blocked on the folder-trust dialog shows the same. Check for a created branch or a tool call in the pane.

Report the clone, the issue, and any phasing or gate criteria. If no `bip fleet watch` is running, start one (`/bip-conductor` Step 7).

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
3. **Restart `bip fleet watch`.** It enumerates slots at startup only. From the conductor clone, `kill $(fleet_watchers)` (from `lib/spawn-intent.sh`), then start it as in `/bip-conductor` Step 7.

**Worktree mode** — no registration needed; worktrees are created on demand in Step 1 and named `issue-<N>`.

## Cleaning up slots after work

Reclaim per `/bip-conductor` Step 6 (`reclaim_slot` in clone mode), never by hand: it preserves the worklog first, and in clone mode runs the terminal ceremony.
