---
name: bip-helper
description: Start, message, resume, and shut down a long-lived helper session — a `claude --bg` peer that survives the primary's compaction and restarts, can message any session, and that the user can attach to. Includes a sweep for leftover helpers. Use when side work (a long gate, a trial build, a cross-session review) would otherwise block the primary's attention.
---

# /bip-helper

A helper is a separate Claude session that you, the primary, start in the background and later shut down.
It is not an `Agent` subagent: a subagent lives inside your process, reports only to you, and dies with you.
A helper has its own context window, keeps running through your compaction or restart, messages any session by name, and the user can attach to it.
Reach for a subagent when the work fits in one call and only you need the answer; reach for a helper when it outlives your turn or talks to other sessions.

## Usage

```
/bip-helper start <role> "<brief>"   # role is a short word: review, gate, build
/bip-helper stop <role>
/bip-helper sweep
```

## Facts this skill depends on

- `claude --bg` sessions authenticate with the stored login, not the shell's `CLAUDE_CODE_OAUTH_TOKEN`.
  To run one on the shell's account, pass `--settings <file>` where the file holds `{"env":{"CLAUDE_CODE_OAUTH_TOKEN":"..."}}`.
  Claude Code stores only the path, and re-reads it whenever the session is resumed, so the file must still exist then.
- A stopped helper disappears from `ListAgents`, so `SendMessage` cannot reach it; resume it first.
- Idle background sessions nobody is attached to are stopped automatically after roughly an hour.
- `claude --bg` prints a short id (`backgrounded · <id> · <name>`); `stop`, `rm`, `logs`, and `attach` take that id.
  `--resume` takes the full session id from `claude agents --json --all`.
  The ref `ListAgents` shows in brackets is a third, different id.

## Rules

- **The helper gets a directory of its own.**
  Never a pooled worker slot, and never a checkout another live session works in: `reclaim_slot` holds on a second session in a slot, and `/bip-issue-work` sends a co-tenant's branch work into a worktree.
- **Kill only by the recorded short id.**
  Never `pgrep`/`pkill`, a session name, or argv — a spawn prompt is argv and quotes arbitrary text, so a pattern match can hit the wrong process, including your own shell.
- **Tell the user** when you start or stop a helper, with its name and short id.

## Records

Each helper has one record file, so a later session (yours after compaction, or anyone running `sweep`) can find it:

```bash
HELPERS="${XDG_STATE_HOME:-$HOME/.local/state}/bip/helpers"
mkdir -p "$HELPERS"
# $HELPERS/<short-id>.json
# {"id":"…","session_id":"…","name":"…","primary":"…","dir":"…","dir_kind":"worktree|scratch","repo":"…"}
```

## start

1. **Name it.**
   Your own name is the first line of `ListAgents` ("This session is `<name>` …").
   The helper is `<primary>-<role>`.
   If `ListAgents` already shows that name, pick another role word.

2. **Make its directory.**
   If it will change code, a detached worktree off `origin/main`, outside every slot:
   ```bash
   git -C <repo> fetch -q origin
   DIR=$(dirname "$(git -C <repo> rev-parse --show-toplevel)")/$(basename <repo>)-<primary>-<role>
   git -C <repo> worktree add --detach "$DIR" origin/main
   ```
   It branches there itself (for example via `/bip-issue-work`), as the checkout's only session.
   Otherwise, a scratch directory: `DIR=$(mktemp -d "${TMPDIR:-/tmp}/<primary>-<role>.XXXX")`.

3. **Start it** from `$DIR`, adding `--settings` only when the account file exists:
   ```bash
   SETTINGS="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/claude-account-settings.json"
   ARGS=(-n "<primary>-<role>")
   [ -f "$SETTINGS" ] && ARGS=(--settings "$SETTINGS" "${ARGS[@]}")
   OUT=$(cd "$DIR" && claude --bg "${ARGS[@]}" "<brief>" 2>&1)
   ID=$(printf '%s\n' "$OUT" | sed -n 's/^backgrounded · \([0-9a-f]*\) · .*/\1/p')
   SID=$(claude agents --json --all | jq -r --arg id "$ID" '.[] | select(.id==$id) | .sessionId')
   ```
   If `ID` or `SID` is empty, stop and show the user `$OUT`; do not guess an id from the listing.
   The brief must name you as the primary, say to report by `SendMessage` to you, name its directory, and state what it must not do (merge, push to main, touch other checkouts) — the helper starts with no other context.

4. **Write the record**, with `jq -n --arg …` into `$HELPERS/$ID.json`.

5. **Confirm it is up**: it should appear in `ListAgents` under its name within a minute.
   If `claude agents --json --all` shows it `failed`, read `claude logs $ID` and report — a failed first turn is most often the wrong account.

## Talking to it

- `SendMessage` to its name; its replies arrive as cross-session messages.
- To hear when it finishes a turn, send with `notify_when_idle: true` instead of polling.
- If it is missing from `ListAgents`, it has stopped. Resume it, then send:
  ```bash
  cd "$DIR" && claude --bg --resume "$SID" "<message>"
  ```
  The resume reuses its saved `--settings` path; if that file is gone, the resume fails — recreate the file (a new login shell does) and retry.
- The user can watch or type to it with `claude attach <id>` (`←` returns to agent view, `Ctrl+Z` to the shell; the helper keeps running either way).

## stop

1. Ask it to wrap up: commit and push anything it owns, then send you a final report.
   Wait for that report (`notify_when_idle: true`); if it is stopped, resume it for this.
2. `claude stop <id>`, then `claude rm <id>`.
   If `rm` reports unpushed commits or a worktree it could not remove, stop and show the user; do not pass `--discard-unpushed` or `--force-remove-worktree` without their go-ahead.
3. Remove the directory.
   Worktree: `git -C <repo> worktree remove "$DIR"` — it refuses on uncommitted changes; if so, show the user rather than forcing.
   Scratch: `rm -rf "$DIR"` after checking the path is the one in the record.
4. Delete the record, and tell the user the helper is gone.

## sweep

For each record in `$HELPERS`, look up its `id` in `claude agents --json --all` and its `primary` in `ListAgents`, then report one line per record:

- **stale** — id not in the listing: the session was removed; offer to remove the directory and record.
- **orphaned** — session present but its primary is not in `ListAgents`: offer to message the helper or stop it.
- **live** — both present: leave it.

Report and ask; do not stop or remove anything in a sweep without the user's go-ahead.
Match on the record's `id` only — never infer a helper from a name prefix in the listing.
