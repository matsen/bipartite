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
- A cross-session message to a session in a different permission mode is held for that session's user to approve.
  Nobody is watching a background helper, so a held message sits there silently.
  Start the helper in your own permission mode with `--permission-mode`; `bypassPermissions` starts without a confirmation prompt.
- A stopped helper disappears from `ListAgents`, so `SendMessage` cannot reach it; resume it first.
- Idle background sessions nobody is attached to are stopped automatically after roughly an hour.
  A helper that has finished and is waiting for you will therefore often be missing from `ListAgents`; that is the expected state, not a failure.
- A `notify_when_idle` notice reaches you only once your own turn ends, so it can arrive long after the helper went idle, and a subscription made while the helper is already idle reports that earlier idle.
  Don't hold a turn open waiting for one: end your turn and act on the notice, or on the helper's report message, when it arrives.
- `claude --bg` prints a short id (`backgrounded · <id> · <name>`); `stop`, `rm`, `logs`, and `attach` take that id.
  `--resume` takes the full session id from `claude agents --json --all`.
  The ref `ListAgents` shows in brackets is a third, different id.
- A background session's scratchpad is `~/.claude/jobs/<id>/tmp/`, and `claude rm` deletes it; nothing citable belongs there.

## Rules

- **Every helper owns a home directory, `$HELPERS/<name>/`.**
  Its `result.md`, every extra checkout it makes, and every file it cites as evidence live there, so `stop` and `sweep` see all of it.
- **The helper works in a directory of its own.**
  Never a checkout another live session works in: `/bip-issue-work` sends a co-tenant's branch work into a worktree, and `reclaim_slot` stops with `NOT FREE` while any process is still in the slot.
  A pooled clone is allowed only when it is available and you hold it first (start, step 2).
  A `--bg` helper has no tmux window, so the hold is the only thing that marks the clone as in use.
- **A worker in a pooled slot does not start helpers.**
  Its helper's checkouts would hang off the slot's repo, and `reclaim_slot` neither removes nor reports them, so they would outlive the slot.
  Ask the epic or the conductor to start one instead.
- **Delete with `find <path> -delete`, never `rm`.**
  A `$` in the same command as `rm` trips Claude Code's destructive-removal guard, which bypass mode does not suppress.
- **Kill only by the recorded short id.**
  Never `pgrep`/`pkill`, a session name, or argv — a spawn prompt is argv and quotes arbitrary text, so a pattern match can hit the wrong process, including your own shell.
- **Tell the user** when you start or stop a helper, with its name and short id.

## Records

```bash
HELPERS="${XDG_STATE_HOME:-$HOME/.local/state}/bip/helpers"
# $HELPERS/<short-id>.json   written by the primary at start:
#   {"id","session_id","name","primary","primary_cwd","home","dir","dir_kind":"home|worktree|held-clone","hold"}
# $HELPERS/<name>/           the helper's home
# $HELPERS/<name>/result.md  written by the helper, one appended section per round
```

## start

1. **Name it and make its home.**
   Your own name is the first line of `ListAgents` ("This session is `<name>` …").
   The helper is `<primary>-<role>`.
   If `ListAgents` shows that name or `$HELPERS/<primary>-<role>` already exists, pick another role word.
   ```bash
   HELPERS="${XDG_STATE_HOME:-$HOME/.local/state}/bip/helpers"
   HOME_DIR="$HELPERS/<primary>-<role>"
   mkdir -p "$HOME_DIR"
   ```

2. **Pick its working directory `DIR`.**
   - Research, review, or messaging only: the home itself, `DIR="$HOME_DIR"`, `KIND=home`.
   - Code changes: a detached worktree off `origin/main`, inside the home, `KIND=worktree`:
     ```bash
     git -C <repo> fetch -q origin
     DIR="$HOME_DIR/$(basename "$(git -C <repo> rev-parse --show-toplevel)")"
     git -C <repo> worktree add --detach "$DIR" origin/main
     ```
     It branches there itself (for example via `/bip-issue-work`), as the checkout's only session.
   - Gates or builds that need a built environment (a `.pixi` env, a vendored binary), where a fresh worktree would pay a full install first: a conductor or epic session that manages a clone pool can lend it an `available` pooled clone (as `/bip-conductor` Step 5 classifies it), held so spawn selection and `bip spawn` skip it, `KIND=held-clone`:
     ```bash
     source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
     CLONE_ROOT=$(resolve_clone_root .epic-config.json)
     HOLD="$CLONE_ROOT/.holds/<clone>"
     mkdir -p "$CLONE_ROOT/.holds" && echo "helper <primary>-<role>" > "$HOLD"
     DIR="$CLONE_ROOT/<clone>"
     ```
     Run it from your own checkout, where `.epic-config.json` lives.
     `<this-skill's-base-directory>` is this skill's base directory as given at invocation; `lib/spawn-intent.sh` is a sibling of every skill directory.
     Write the hold before anything else touches the clone, then confirm the clone is still `available`; if it is not, delete the hold and pick another.

3. **Write the brief.**
   The helper starts with no other context, so the brief must:
   - name you as the primary and say to report by `SendMessage` to you;
   - give its home and working directory as expanded paths;
   - say that any extra checkout it needs (a control, a second probe) goes under its home, as a `git worktree add` there;
   - say that anything it cites — logs, tables, outputs — is saved under its home, never in its session scratchpad;
   - say that after each round of work it appends a section to `result.md` in its home, headed with the time and the round's question, holding the round's findings and the paths of its evidence, before reporting to you;
   - for a held clone, say to leave the clone clean and on `main` when it finishes;
   - list what it must not do: merge, push to main, touch other checkouts, start helpers of its own.

4. **Start it** from `$DIR`, in your own permission mode, adding `--settings` only when the account file exists:
   ```bash
   SETTINGS="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/claude-account-settings.json"
   ARGS=(-n "<primary>-<role>" --permission-mode <your-mode>)
   [ -f "$SETTINGS" ] && ARGS=(--settings "$SETTINGS" "${ARGS[@]}")
   OUT=$(cd "$DIR" && claude --bg "${ARGS[@]}" "<brief>" 2>&1)
   ID=$(printf '%s\n' "$OUT" | sed -n 's/^backgrounded · \([0-9a-f]\{1,\}\) · .*/\1/p')
   SID=$(claude agents --json --all | jq -r --arg id "$ID" '.[] | select(.id==$id) | .sessionId')
   ```
   `<your-mode>` is one of `claude --help`'s `--permission-mode` choices; bypass mode is `bypassPermissions`.
   If `ID` or `SID` is empty, stop and show the user `$OUT`; do not guess an id from the listing.

5. **Write the record.**
   `HOLD` is empty unless `KIND=held-clone`.
   ```bash
   jq -n --arg id "$ID" --arg sid "$SID" --arg name "<primary>-<role>" --arg primary "<primary>" \
       --arg pcwd "$(pwd -P)" --arg home "$HOME_DIR" --arg dir "$DIR" --arg kind "$KIND" --arg hold "${HOLD:-}" \
       '{id:$id, session_id:$sid, name:$name, primary:$primary, primary_cwd:$pcwd, home:$home, dir:$dir, dir_kind:$kind, hold:$hold}' \
       > "$HELPERS/$ID.json"
   ```

6. **Confirm messages flow both ways.**
   Wait for its first report, then `SendMessage` it a one-line ping and wait for the answer.
   If either is missing after a few minutes, read `claude logs <id>` and report to the user.
   A `failed` state in `claude agents --json --all` most often means the wrong account; a silent helper, a held message.

## Finding a helper's record

`stop`, `sweep`, and a later turn start from the record, not from shell variables left over from `start`:

```bash
HELPERS="${XDG_STATE_HOME:-$HOME/.local/state}/bip/helpers"
R=$(jq -r --arg n "<primary>-<role>" 'select(.name==$n) | input_filename' "$HELPERS"/*.json)
```

If `R` names no file or more than one, stop and show the user the records.
Read `id`, `session_id`, `home`, `dir`, `dir_kind`, and `hold` from `$R` with `jq -r`.

## Talking to it

- `SendMessage` to its name; its replies arrive as cross-session messages.
- Read `result.md` in its home first; each round it finished has a section there.
  Resume it only for a new round or a question the file does not answer.
- If it is missing from `ListAgents`, it has stopped.
  Resume it with the message as the prompt:
  ```bash
  cd "$DIR" && claude --bg --resume "$SID" "<message>"
  ```
  The resume reuses its saved `--settings` path and permission mode; if the settings file is gone, the resume fails — recreate the file (a new login shell does) and retry.
- The user can watch or type to it with `claude attach <id>` (`←` returns to agent view, `Ctrl+Z` to the shell; the helper keeps running either way).

## stop

1. Ask it to wrap up: commit and push anything it owns, append its last section to `result.md`, then report to you.
   If it is stopped, resume it for this.
2. `claude stop <id>`, then `claude rm <id>`.
   If `rm` reports unpushed commits or a worktree it could not remove, stop and show the user; do not pass `--discard-unpushed` or `--force-remove-worktree` without their go-ahead.
3. Remove every worktree under the home:
   ```bash
   find "$HOME_DIR" -mindepth 2 -maxdepth 3 -name .git -type f | while read -r g; do
       w=$(dirname "$g"); git -C "$w" worktree remove "$w" || echo "KEEP $w"
   done
   ```
   `worktree remove` refuses on uncommitted or untracked files; for each `KEEP`, show the user rather than forcing.
4. For a held clone, never delete the clone.
   If `git -C "$DIR" status --porcelain` is empty and it is on `main`, lift the recorded hold with `find "$(dirname "$hold")" -maxdepth 1 -name "$(basename "$hold")" -delete`.
   Otherwise keep the hold and show the user what the helper left.
5. The home holds `result.md` and the evidence it cites, which may be quoted on GitHub.
   Show the user the evidence paths `result.md` lists and ask before `find "$HOME_DIR" -delete`; if nothing is listed and step 3 kept nothing, delete it.
6. Delete the record with `find "$HELPERS" -maxdepth 1 -name "<id>.json" -delete`, and tell the user the helper is gone.

## sweep

For each record in `$HELPERS`, look up its `id` in `claude agents --json --all` and its `primary` in `ListAgents`, then report one line per record:

- **stale** — id not in the listing: the session was removed; offer to finish `stop` from step 3.
- **primary not found** — session present but no `ListAgents` row carries the recorded primary name.
  `ListAgents` renames a session when it is resumed (`birch` becomes `birch-61`), so this is not proof the primary is gone.
  Say whether any session in `claude agents --json` has the recorded `primary_cwd`, and offer to message the helper or stop it.
- **live** — both present: leave it.

Also list any directory under `$HELPERS` that no record names, with its size.
Report and ask; do not stop or remove anything in a sweep without the user's go-ahead.
Match on the record's `id` only — never infer a helper from a name prefix in the listing.
