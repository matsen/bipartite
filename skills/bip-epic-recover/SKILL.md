---
name: bip-epic-recover
description: Recover bip-epic Claude sessions after a host reboot — replay a planned-reboot manifest if one exists, else find the killed sessions by scanning jsonl, label them, and resume each into a tmux window. Use when a box running a bip-epic fleet rebooted and the tmux sessions are gone.
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

## The bundled helper

This skill ships the `epic-recover` shell helper next to `SKILL.md` — no separate install. At the start of the run, set `HELPER` to its absolute path (this skill's base directory, given at invocation, + `/epic-recover`) and use `"$HELPER"` in every command below:

```bash
HELPER="<this-skill-dir>/epic-recover"
```

It needs GNU `date` (or `gdate`), `stat`, bash ≥ 4, and `jq`: it runs on Linux out of the box, and on macOS with `brew install coreutils bash`. It exits 2 with guidance if a prerequisite is missing.

Start tmux the usual way first (e.g. `eval $(keychain --eval id_rsa) && tmux`) so the server holds your ssh-agent; windows the helper creates inherit that env.

## Two recovery paths

If the reboot was **planned** and `/bip-epic-prepare-reboot` parked the host first, a host-wide manifest at `~/.epic-recover/manifest.json` records every session/window with its exact `session_id`. Replaying it is deterministic and lossless — prefer it. Otherwise fall back to the jsonl **scan** below, which infers the live-at-reboot set from file mtimes.

### Step 0: Check for a planned-reboot manifest

```bash
"$HELPER" --manifest-status
```

- **Exit 0 (`VALID …`)** — a manifest exists, was parked *before* the current boot, and is not yet consumed. Use the **manifest path** (Step M) and skip the scan entirely.
- **Exit 1 (`none` / `stale` / `invalid`)** — no usable manifest (none written, parked *after* this boot, already consumed, or unparseable). Fall through to the **scan path** (Step 1). This is project-specific, so `cd` to the project's main clone first.

### Step M: Replay the manifest

```bash
"$HELPER" --manifest-list      # TSV: session, index, window, cwd, session_id, method, confidence, issue, candidates
```

Present the plan grouped by session (real names, windows in order). Most windows resume straight to their recorded id. For any window with `confidence=ambiguous`, show its `candidates` and **ask the user which id to resume** (or to leave it a plain shell) — do not guess. Then replay:

```bash
"$HELPER" --manifest-resume \
  ["<session>:<index>=<chosen-id>" ...] \    # one per ambiguous window the user resolved
  ["<session>:<index>=skip" ...]             # leave that window a plain shell
```

This rebuilds **every** session by its real name with windows in order — Claude windows `--resume`'d to their ids, shell windows as bare shells — and reports "(from manifest)". Ambiguous windows not given an explicit pick are left as plain shells. On success it stamps the manifest consumed (renames it `manifest.<boot>.done`) so a later *unplanned* reboot does not replay a stale park. Then nudge resumed workers per **Step 5**.

## Scan path (no usable manifest)

### Step 1: Enumerate the killed sessions

From the project's main clone, run the helper's list mode:

```bash
cd <project-main-clone>
"$HELPER" --list
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

Show a compact labeled table (last-active, clone, label, short session-id) sorted newest first. Recommend the live-at-reboot set (newest per clone, plus the distinct main-clone sessions) but let the user choose. Always offer an explicit **Recover all** option alongside the per-session picks (and "Recommended set" / "Cancel"); when the user picks **Recover all**, pass *every* listed session-id to `--resume` in one call. **Do not spawn anything before the user confirms which sessions to resume.**

### Step 4: Resume the chosen sessions

Pass the selected session-ids to the helper. Run this from inside the project's tmux session so windows are added to it (the helper detects `$TMUX`):

```bash
"$HELPER" --resume <id1> <id2> ...
```

Each becomes a window named `<issue>-<clone>` (workers) or the clone basename (main-clone sessions), running `claude --dangerously-skip-permissions --resume <id>`.

### Step 5: Nudge resumed workers

A resumed worker reloads its context but sits idle — the ralph-loop wakeup that would have driven the next iteration died with the process. After each worker window finishes loading, send a one-line nudge so it resumes forward motion:

> Host rebooted and this session was interrupted. Re-read `.epic-status.json` and `.epic-worklog.md`, then continue from where you left off.

Auto-`send-keys` timing against claude's resume-load is unreliable, so prompt the user to paste it (or do it interactively once the window is ready). Conductor/planning/discussion sessions usually need no nudge.

## Notes

- `--list`, `--manifest-status`, and `--manifest-list` mutate nothing; only `--resume`, `--manifest-resume`, and the bare interactive mode create tmux windows. Re-running the read-only modes is safe.
- The manifest modes are **host-wide and project-agnostic** — they do not need `.epic-config.json` and rebuild every session in the manifest, not just one project's clones. The scan modes (`--list`, `--resume`, interactive) still require `.epic-config.json` in the cwd.
- For a planned reboot, write that manifest first with `/bip-epic-prepare-reboot` — it captures exact session ids from live processes, which is lossless where the mtime scan only infers.
- Driving a remote rebooted box from elsewhere: prefix the helper calls with `ssh <host> 'cd <main-clone> && bash <skill-dir>/epic-recover …'`, but resuming into tmux is cleanest run from a shell already inside that host's tmux.
- For a no-Claude recovery, run the helper directly with no args (`bash <skill-dir>/epic-recover`) for an interactive numbered picker over the same data — less smart labeling, same engine. Symlink it onto your `PATH` if you want it as a bare command.

## Manual verification

The manifest data paths are pure JSON logic and worth exercising after a change; the live-tmux/process parts of the partner skill are verified the same way (see `bip-epic-prepare-reboot`). Both helpers ship without a unit-test suite, matching this repo's shell-helper convention.

1. **Staleness** — a manifest whose `parked_at` is *after* the current boot → `--manifest-status` reports `stale` and exits 1 (recover falls back to the scan).
2. **Consumed** — after `--manifest-resume`, the manifest is renamed `manifest.<boot>.done`; a fresh `--manifest-status` reports `none`.
3. **Fallback** — with no manifest, `--manifest-status` exits 1 and the scan path behaves exactly as before.
4. **Ambiguity** — an `ambiguous` window is left a plain shell unless an explicit `<session>:<index>=<id>` pick resumes it.
