---
name: bip-epic-prepare-reboot
description: Quiesce the whole tmux host before a PLANNED reboot — walk every session, resolve each Claude window's exact session id, optionally checkpoint workers, and write a manifest so bip-epic-recover can rebuild the workspace deterministically. Use when you know a reboot is coming and want a clean, lossless restart.
allowed-tools: Bash, Read
---

# /bip-epic-prepare-reboot

`/bip-epic-recover` reconstructs a *killed* fleet after an unplanned reboot by scanning `~/.claude/projects/*/<id>.jsonl` and inferring which sessions were live from file mtimes. That inference is lossy: mtime is last *activity*, not "window was open", and it cannot tell which of several concurrent sessions in one cwd were on screen.

When the reboot is **planned**, we can do better: capture ground truth *while tmux is still alive*. This skill walks every pane on the host in one run, resolves each Claude window's exact `session_id` from the live process (its `--resume` cmdline, or start-time correlation for fresh sessions), optionally checkpoints epic workers, and writes a host-wide manifest. `bip-epic-recover` then replays that manifest after the reboot — rebuilding every session by its real name, windows in order, each Claude window resumed to the right id — instead of guessing.

The deterministic engine is the bundled `epic-prepare-reboot` shell helper; this skill is the brain that runs it, sanity-checks the resolved table with the user, and triggers the manifest write / shutdown. Run it **on the box about to reboot** — that is where tmux, the jsonl files, and the clones live.

## Usage

```
/bip-epic-prepare-reboot [--no-checkpoint] [--shutdown]
```

Run it once, host-wide, *before* the reboot. There is no project argument — the manifest spans the entire tmux host, not one epic project.

## The bundled helper

This skill ships the `epic-prepare-reboot` helper next to `SKILL.md` — no separate install. At the start of the run, set `HELPER` to its absolute path (this skill's base directory, given at invocation, + `/epic-prepare-reboot`) and use `"$HELPER"` in every command below:

```bash
HELPER="<this-skill-dir>/epic-prepare-reboot"
```

It needs `tmux`, `jq`, bash ≥ 4, and a GNU `date` (or `gdate`): it runs on Linux out of the box, and on macOS with `brew install coreutils bash`. Session-id resolution reads Linux `/proc` first and falls back to `ps` (so it is fully functional on macOS too, just second-resolution start times). It exits 2 with guidance if a prerequisite is missing.

## Workflow

### Step 1: Dry-run and review the resolved table

Always look before you write. `--dry-run` walks every pane, resolves each window, and prints the table **without** checkpointing, writing, or shutting anything down:

```bash
"$HELPER" --dry-run
```

Each row shows `SESSION  IDX  WINDOW  METHOD  CONF  SESSION_ID/CWD`. The resolution `method` per Claude window, in priority order:

- **cmdline** (`high`) — `--resume <id>` was in the Claude process args. Exact; covers any resumed session in any cwd.
- **starttime** (`medium`) — a *fresh* session (launched without `--resume`) carries no id in its cmdline, so the helper correlates the Claude process start time with the cwd's jsonl that *began* closest. Handles fresh windows, including two fresh windows in one cwd.
- **newest** (`low`) — last resort: the newest jsonl for the cwd.
- **shell** (`none`) — no Claude descendant; a bare shell or editor. Recorded with its cwd so the layout returns, recreated as a plain window.

When start-time correlation finds two jsonls that fit a fresh process nearly equally (within a few seconds), the window is flagged **`ambiguous`** and both candidate ids are recorded — recover will ask rather than guess.

Present a short summary to the user: how many sessions/windows, how many Claude windows resolved by each method, and call out any `ambiguous` rows by name. This is the moment to catch a misresolved window before it goes in the manifest.

### Step 2: Decide on checkpointing

By default the helper sends every epic **worker** (a Claude window whose cwd has `.epic-status.json`) a one-line checkpoint instruction — commit WIP, flush `.epic-status.json` + `.epic-worklog.md` — and waits briefly (`EPIC_PREPARE_CHECKPOINT_WAIT`, default 20s) for the flush. This **persists state; it does not wait for tasks to finish.**

- Default (checkpoint): use when workers are mid-task and you want their latest state captured.
- `--no-checkpoint`: skip it when the fleet is already quiet, or when interrupting in-flight tool calls would do more harm than good.

Confirm the choice with the user before the real run.

### Step 3: Write the manifest

Run the helper for real. Without `--shutdown` it stops after writing the manifest, leaving tmux running so you can verify:

```bash
"$HELPER"                 # checkpoint workers, then write the manifest
"$HELPER" --no-checkpoint # write the manifest without quiescing workers
```

The manifest lands at `~/.epic-recover/manifest.json` (override the dir with `EPIC_RECOVER_DIR`). It preserves your real session names, window names, order, and cwds, with a `session_id`, `method`, and `confidence` per Claude window — a **host-level** file so recover finds it without knowing the project list.

### Step 4: Shut down (optional)

When you are ready to reboot, either reboot the box yourself, or let the helper tear tmux down only **after** the manifest is on disk:

```bash
"$HELPER" --shutdown      # writes the manifest, then `tmux kill-server`
```

`--shutdown` never kills the server before the manifest write succeeds.

### Step 5: After the reboot

On the rebooted box, run `/bip-epic-recover`. It detects the manifest (parked before the current boot, not yet consumed), rebuilds **every** session by its real name with windows in order — Claude windows `--resume`'d to their exact ids, plain windows as bare shells — reports "(from manifest)", and asks about any `ambiguous` windows. After a successful replay it stamps the manifest consumed so a later *unplanned* reboot falls back to the jsonl scan instead of replaying a stale park.

## The manifest (what recover consumes)

```json
{
  "host": "pax",
  "parked_at": "2026-05-29T06:00:00Z",
  "sessions": [
    {"name": "phyz", "windows": [
      {"index": 0, "name": "claude",     "cwd": "/home/matsen/re/phyz",     "session_id": "30ebe63e-…", "method": "starttime", "confidence": "medium", "issue": null},
      {"index": 1, "name": "1483-alder", "cwd": "/home/matsen/re/pz/alder", "session_id": "ee906cc5-…", "method": "cmdline",   "confidence": "high",   "issue": 1483}
    ]}
  ]
}
```

`confidence` ∈ `high` (cmdline) · `medium` (starttime) · `low` (newest) · `ambiguous` · `none` (shell). An `ambiguous` window also carries a `candidates: [id, id]` array. `session_id` is `null` for shell windows.

## Manual verification

These helpers introspect live tmux and the process tree, so they are verified by exercising them, not by a unit-test suite (the same convention as `epic-recover`). To check a change, against any host with a few tmux windows open:

1. **Fidelity** — `"$HELPER" --dry-run`; confirm the table's sessions/windows match `tmux list-panes -a` exactly (names, order, cwds), with a resolved `session_id` per Claude window.
2. **Resolution** — a resumed window (started with `--resume`) shows `method=cmdline`; a fresh window shows `starttime`; two fresh windows in one cwd with near-identical start times show `ambiguous` (with two candidates in the manifest).
3. **Roundtrip** — run for real, then on the same host `epic-recover --manifest-status` reports VALID and `--manifest-resume` rebuilds the sessions by real name with windows in order. (See `bip-epic-recover`'s own verification notes for staleness/consumed/fallback.)
4. **Guards** — `--no-checkpoint` skips the worker nudges; `--shutdown` kills tmux only after the manifest exists.

## Notes

- `--dry-run` mutates nothing. Only a real run writes the manifest, sends checkpoints, or (with `--shutdown`) kills tmux.
- The skill is **host-wide and project-agnostic** — it does *not* require `.epic-config.json` and records every session on the host, not just epic clones.
- Naming pairs it with `bip-epic-recover`, though it operates beyond epic clones. Related: `bip-epic-tuckin` persists *orchestrator* state for a context reset — distinct from parking the *whole host* for a reboot.
- Driving a remote box from elsewhere: prefix with `ssh <host> 'bash <skill-dir>/epic-prepare-reboot …'`, but it is cleanest run from a shell on the box itself.
