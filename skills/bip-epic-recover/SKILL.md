---
name: bip-epic-recover
description: Recover bip-epic Claude sessions after a host reboot — find the killed sessions, label them, and resume each into a tmux window. Use when a box running a bip-epic fleet rebooted and the tmux sessions are gone.
allowed-tools: Bash, Read
---

# /bip-epic-recover

A reboot of a host running a bip-epic fleet kills the tmux server and every `claude` process at once. The clones, their `.epic-status.json`/`.epic-worklog.md`, and the full per-session `~/.claude/projects/*/<id>.jsonl` conversations all survive on disk, so the sessions are resumable via `claude --dangerously-skip-permissions --resume <id>`. This skill finds those sessions, labels them in human terms, and resumes the ones you pick into tmux windows.

The deterministic engine is the bundled `epic-recover` shell helper; this skill is the brain that turns its raw output into labeled choices. Run it **on the box that rebooted** — that is where the tmux, jsonl, and clones live.

## Usage

```
/bip-epic-recover [project]
```

`project` is a path to a bip-epic project's **main clone** (the dir holding `.epic-config.json`), or its basename. Defaults to the current directory.

## Prerequisite: install the helper on the recovery host

The helper lives at `skills/bip-epic-recover/epic-recover` (Linux-targeted; uses GNU `date` and `jq`). On the host that runs the fleet:

```bash
install -m755 skills/bip-epic-recover/epic-recover ~/bin/epic-recover   # ~/bin on PATH
```

Start tmux the usual way first (e.g. `eval $(keychain --eval id_rsa) && tmux`) so the server holds your ssh-agent; windows the helper creates inherit that env.

## Workflow

### Step 1: Enumerate the killed sessions

From the project's main clone, run the helper's list mode:

```bash
cd <project-main-clone>
epic-recover --list
```

This emits one TSV row per Claude session whose cwd is the main clone **or any worker clone** and whose jsonl was active in the window before the last reboot (default 36h; `EPIC_RECOVER_SINCE_HOURS` to widen). Columns: `last_active  clone  branch  issue  phase  session  first_prompt`. There is intentionally **no git-branch filter** — a worker clone often returns to `main` while its conversation continues, and the main clone hosts several concurrent sessions (conductor, planning, topic coordination).

### Step 2: Label each session

`first_prompt` is often spawn-prompt boilerplate ("IMPORTANT: Before doing any work…", "Caveat: The messages below…", a ralph-loop preamble) rather than the topic. When it is, peek deeper to name the session — read the first human-authored message past the preamble:

```bash
f=~/.claude/projects/$(printf '%s' "<clone-cwd>" | sed 's#/#-#g')/<session-id>*.jsonl
jq -rs '[.[]|select(.type=="user")|.message.content
          | if type=="string" then . elif type=="array" then (.[]|select(.type=="text")|.text) else empty end]
        | map(select(test("tool_result|tool_use_id|^Caveat:|^IMPORTANT: Before")|not))
        | .[0:3] | .[]' "$f" | head -c 600
```

Turn each row into a short human label, e.g. `#1481 → clade-matched ASR parity`, `pcp-pipeline #52 follow-ups`, `coordinate-pca`, `DASM planning`. Note duplicates: a clone may have several sessions across the day; the newest is usually the one that was live at reboot.

### Step 3: Present the choices and confirm

Show a compact labeled table (last-active, clone, label, short session-id) sorted newest first. Recommend the live-at-reboot set (newest per clone, plus the distinct main-clone sessions) but let the user choose. **Do not spawn anything before the user confirms which sessions to resume.**

### Step 4: Resume the chosen sessions

Pass the selected session-ids to the helper. Run this from inside the project's tmux session so windows are added to it (the helper detects `$TMUX`):

```bash
epic-recover --resume <id1> <id2> ...
```

Each becomes a window named `<issue>-<clone>` (workers) or the clone basename (main-clone sessions), running `claude --dangerously-skip-permissions --resume <id>`.

### Step 5: Nudge resumed workers

A resumed worker reloads its context but sits idle — the ralph-loop wakeup that would have driven the next iteration died with the process. After each worker window finishes loading, send a one-line nudge so it resumes forward motion:

> Host rebooted and this session was interrupted. Re-read `.epic-status.json` and `.epic-worklog.md`, then continue from where you left off.

Auto-`send-keys` timing against claude's resume-load is unreliable, so prompt the user to paste it (or do it interactively once the window is ready). Conductor/planning/discussion sessions usually need no nudge.

## Notes

- `--list` mutates nothing; only `--resume` (and the helper's bare interactive mode) create tmux windows. Re-running is safe.
- Driving a remote rebooted box from elsewhere: prefix the helper calls with `ssh <host> 'cd <main-clone> && epic-recover …'`, but resuming into tmux is cleanest run from a shell already inside that host's tmux.
- For a no-Claude recovery, `epic-recover` with no args gives an interactive numbered picker over the same data — less smart labeling, same engine.
