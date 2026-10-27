---
name: bip-conductor-spawn
description: Spawn a Claude session in a clone for an EPIC issue
---

# /bip-conductor-spawn

Spawn a Claude Code worker in a tmux window for a GitHub issue. The worker runs in a ralph-loop with an issue-lead subagent at stopping points.
The conductor runs this skill and adds the live fleet facts that `/bip-epic` cannot see.

## Usage

```
/bip-conductor-spawn <issue-number> [clone-name]
```

If clone-name is omitted, pick the best idle clone.

## Configuration

Reads `.epic-config.json` from the repo root (format: `/bip-conductor`). If it is missing, stop and ask the user to set it up via `/bip-conductor`.

## Where the prompt comes from

Usually `/bip-epic` has written the brief to `$CLONE_ROOT/.spawn-prompts/`, as `<N>.md` (current) or `spawn-<N>.txt` (older). Check there first:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
INTENT=$(find_spawn_intent "$CLONE_ROOT" <N>)
[ -n "$INTENT" ] && cat "$INTENT"
```

`<this-skill's-base-directory>` is this skill's base directory as given at invocation; `lib/spawn-intent.sh` is a sibling of every skill directory.

- **Intent file present**: it is the base of Step 4's `IMPORTANT CONTEXT`. Check it against live state (Step 2b), append fleet facts it couldn't know (free host/clone, a worker on an overlapping file, a build on the target host), and mark it consumed after launch (Step 6).
  An epic brief has an `EPIC:` line near the top; the conductor counts live slots by it. Three forms are in use (`EPIC: matsengrp/phyz#369`, `EPIC: 369`, `**EPIC**: #369`), so match with `grep -iE '^\**EPIC\**:? *([[:alnum:]_.-]+/[[:alnum:]_.-]+)?#?([0-9]+)'`. If an epic brief lacks one, ask the epic; a user-originated brief has none.
- **No intent file**: compose the prompt from the issue; this is normal (user spawns, respawns, maintenance), so don't wait for a brief. An issue may belong to no EPIC; never reject or defer it for that (`/bip-conductor` "Terms and intake").

Resolve a conflict between intent and fleet state from measured state, and report it. Escalate only per `/bip-conductor` "Arbitration".

## Workflow

### Prerequisite: Issue number required

Every spawn targets an existing issue (EPIC tracking, PR linking and slot monitoring need one). If none exists, file a minimal one (title, 3-sentence motivation, success criteria) through `/bip-issue-check`.

### Step 1: Select or create slot

Read `clone_root` and `local_worktrees` from `.epic-config.json`.

**Clone mode** (`local_worktrees` absent or false). If clone-name is not given, find an idle clone:
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

If `tmux` fails, every clone looks free; `bip spawn`'s refusal of an occupied directory (Step 5) is the real guard.
If all are busy, offer to create a clone named from `new_clone_names` (see "Creating new slots").

**Worktree mode** (`local_worktrees: true`). Slot is `issue-<N>`; branch is `<N>-<slug>`, slug = first 4 words of the issue title, lowercased and hyphenated.

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

Run each block below as one command; the working directory does not persist between Bash calls.
Preserve the prior state; only if that did not fail, clear `.epic-status.json`, `.epic-worklog.md` and `.claude/ralph-loop.local.md`. Delete with `find <absolute-path> -delete`, not `rm`: a `$` in an `rm` command trips Claude Code's destructive-removal guard, even in bypass mode.

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

Fetch inside each clone, not once in the conductor: each clone has its own `origin/main`, so `git -C <clone> reset --hard origin/main` after a conductor fetch uses a stale ref. Keep the `cd` in a batch-spawn loop.

**Worktree mode**: resolve `CLONE_ROOT` before `cd "$SLOT"`, in the same command; a linked worktree has no `.epic-config.json`, so resolving after the `cd` silently gives an empty root.

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

**Resuming a clone parked mid-issue**: delete only `.epic-status.json`, since its `lead_guidance` outranks the launch prompt (a stale "stand down" stops the worker). Keep the worklog and `zig-out/bin/phyz` (it may be the provenance of a committed artifact); tell the slot to rebuild before re-verifying.

```bash
find "$SLOT" -maxdepth 1 -name '.epic-status.json' -delete   # keep .epic-worklog.md and zig-out/
```

**Stale binaries.** A stale `zig-out/bin/phyz` gives wrong numbers; the delete above covers fresh assignments, `context-additions.md` ("For code changes") the rest.

### Step 2b: Pre-launch staleness check

Verify every blocker or dependency the issue body names is still open:

```bash
gh pr view <N> --json state,mergedAt   # merged already?
gh issue view <N> --json state          # closed already?
```

If one is resolved, correct `IMPORTANT CONTEXT`. Also check whether the work is already done; nothing updates an issue body when a PR lands part of it:

```bash
gh pr list --state merged --search "<N>" --json number,title,mergedAt   # read the titles
gh issue view <N> --json body -q .body | sed -n '/[Ff]iles to modify/,/^#/p'   # then check those paths on main
git show origin/main:<path>   # does the described content already exist?
```

Walk the `Depends-on` / related-PR list as a thread, not only a state check.

### Step 3: Read the issue

```bash
gh issue view <number> --json title,body
```

Read it even with an intent file; the brief may be stale.

### Step 4: Compose the prompt

`IMPORTANT CONTEXT` is the intent file (if any) corrected per Step 2b, plus conductor-only fleet facts.
Measure now, since later corrections race the worker: hash the inputs the issue names, resolve every bare path against the real tree, locate the needed tools and give their versions.
Put a fact that invalidates an instruction inside that instruction (claim, why wrong, file:line).
When reusing a previous prompt, re-read every issue-specific sentence, not just the numbers.
The `sed` writes the literal `CLONE_ROOT`, because worker clones have no `.epic-config.json`.

Fill in every time:

- **`EPIC:`** — `<owner/repo>#<N>` from the brief's `EPIC:` line, or `none`; a follow-up takes its parent's. Freeze the EPIC, not its owner: the worker reads `Owner:` at send time.
- **`OWNER_CMD`** — the literal lookup for the worker to copy: `gh issue view <EPIC N> -R <owner/repo> --json body -q .body | grep -m1 '^Owner: '`, or `true # EPIC: none`.
- **`NOTIFY:`** — exact `ListAgents` names of the sessions that requested or framed the issue, or `none`. With `EPIC: none`, these and the conductor are the slot's only routing.
- **`LANDING DELEGATION:`** — `/bip-conductor` "Who may land" as it applies to this repo, plus any dated exception from the decisions log (workers can't read it). With `EPIC: none`, name the signer (a `NOTIFY:` session, or the conductor for a mechanical-only PR); with none, a self-contained PR stops at `awaiting-human-merge`.
- **`JOINT LANDING GATE: YES | NO`** — YES only when landing is hard to reverse, with a reason; a hold stalls a finished slot invisibly. Size, risk or a shared file is sequencing, not a gate; otherwise NO, and the owning session hears of it when the PR opens.

For an epic-reserved artifact, an additive-only edit lands and notifies; removing or altering epic-recorded text needs an announced intent-to-land and an ack. Use a three-dot diff; a two-dot diff shows everything landed since the fork as deletions:

```bash
git diff origin/main...HEAD -- <the owned files>
```

**Prompt file.** Fill `worker-brief.txt` (this skill's directory; ends at `IMPORTANT CONTEXT:`) with `sed`, without Reading it, then append context with a heredoc. Escape `&`, `|` and `\` in values, `OWNER_CMD`'s pipe included.

```bash
B="<this-skill's-base-directory>"
sed -e "s|{{N}}|<N>|g" -e "s|{{TITLE}}|<title>|g" -e "s|{{CLONE_ROOT}}|$CLONE_ROOT|g" \
  -e "s|{{EPIC}}|<owner/repo#N or none>|" -e "s|{{OWNER_CMD}}|<the lookup above>|" -e "s|{{NOTIFY}}|<exact session names or none>|" \
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

To check what the brief says on one topic, grep it rather than Reading it: `grep -n -A8 'DEFERRAL RULE' "$B/worker-brief.txt"`.

**Context additions.** `context-additions.md` (this skill's directory) has snippets for filesystem mode (always, when work runs remote jobs; chosen by `shared_filesystem`), experiments, build-system targets, code changes (incl. phyz's stale-binary check), and phased work. Read it when the issue calls for one.

### Step 5: Launch tmux window

`bip spawn` refuses a directory with a live tmux pane (`refusing: a tmux pane is already live in <dir>`); pick another clone, or pass `--force` for a deliberate second session.

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

Use `--prompt-file`, not `--prompt "$(cat file)"`, which zsh expands. Never launch with raw `tmux new-window` / `send-keys` / `claude`.
Before relying on a recent `bip` change, confirm the installed binary has it (`strings "$(command -v bip)" | grep -F '<new string>'`); rebuild if not.

### Step 6: Confirm

After a launch from an intent file (`$INTENT`), mark it consumed: a move, not a delete (it is outside git, and `/bip-conductor-tuckin` reports it). Verify the move; an unmoved brief reads as pending forever:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
mark_spawn_intent_consumed "$INTENT"
ls "$CLONE_ROOT/.spawn-prompts/consumed/<N>.md"
```

An epic can append to `.spawn-prompts/<N>.md` after consumption. A brief opens with an `EPIC:` line; a delta does not:

```bash
if [ -f "$CLONE_ROOT/.spawn-prompts/consumed/$N.md" ] \
   && ! grep -qiE '^\**EPIC\**:? *([[:alnum:]_.-]+/[[:alnum:]_.-]+)?#?[0-9]+' "$CLONE_ROOT/.spawn-prompts/$N.md"; then
  echo "DELTA, not a brief"     # deliver it; do NOT spawn from it
else
  echo "BRIEF"
fi
```

Deliver a delta by appending it to the live slot's `.epic-worklog.md`, then messaging the slot; with no live slot, post it as an issue comment.

The prompt is frozen at launch (`RECOVERING CONTEXT` re-reads it). For a brief or gating-rule change, decide: message live workers to record it in their worklog, or let it start at the next spawn.

Before reporting a worker live, look for a created branch or a tool call in the pane; `context used N%` also shows for a session stuck on the folder-trust dialog.

Report the clone, the issue, and any phasing or gate criteria. If no `bip fleet watch` is running, start one (`/bip-conductor` Step 7).

## Creating new slots

**Clone mode**:
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

1. Add the name to `clone_names` in `.epic-config.json`; Step 1 and the conductor's scans use it (`bip spawn` does not).
2. Trust the directory before spawning; a fresh clone opens on the folder-trust dialog, which queues the prompt. From the conductor, read the highlighted option first (`grep -nE 'No, exit|Yes, I trust'` the pane) and send `Down` before `Enter` when `No, exit` is highlighted; never a blind `Enter`.
3. Restart `bip fleet watch` (it enumerates slots only at startup): from the conductor clone, `kill $(fleet_watchers)` (from `lib/spawn-intent.sh`), then start it as in `/bip-conductor` Step 7.


## Cleaning up slots after work

Reclaim per `/bip-conductor` Step 6 (`reclaim_slot` in clone mode), never by hand: it preserves the worklog first, and in clone mode runs the terminal ceremony.
