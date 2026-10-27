---
name: bip-helper
description: Start, message, resume, and shut down a long-lived helper session — a `claude --bg` peer that survives the primary's compaction and restarts, can message any session, and that the user can attach to. Includes a sweep for leftover helpers. Use when side work (a long gate, a trial build, a cross-session review) would otherwise block the primary's attention.
---

# /bip-helper

A helper is a separate `claude --bg` session that you, the primary, start and later shut down.
Unlike an `Agent` subagent, it has its own context, outlives your compaction or restart, messages any session by name, and the user can attach to it.
Use a subagent when one call answers only you; use a helper when the work outlives your turn or talks to other sessions.

## Usage

```
/bip-helper start <role> "<brief>"   # role is a short word: review, gate, build
/bip-helper stop <role>
/bip-helper sweep
```

## Facts this skill depends on

- `claude --bg` sessions ignore the shell's `CLAUDE_CODE_OAUTH_TOKEN`; they authenticate from the `env` token in `~/.claude/settings.json`, written by the account switch in matsen/setup.
- A `--resume` keeps none of the original flags; pass them again.
- A message to a session in a different permission mode is held for its user to approve, and nobody watches a helper. Start it in your own mode with `--permission-mode` (`bypassPermissions` starts without a prompt).
- A stopped helper is missing from `ListAgents` and `SendMessage` cannot reach it; resume it first. Idle unattached sessions stop after about an hour, so a finished helper is usually stopped — expected, not a failure.
- A `notify_when_idle` notice arrives only when your turn ends, and a subscription made while the helper is idle reports that earlier idle. End your turn and act when the notice or report arrives.
- `claude --bg` prints `backgrounded · <id> · <name>`; `stop`, `rm`, `logs`, and `attach` take that short id. `--resume` takes the full session id from `claude agents --json --all`.
- A background session's scratchpad is `~/.claude/jobs/<id>/tmp/`, which `claude rm` deletes; nothing citable belongs there.

## Rules

- Every helper owns a home directory, `$HELPERS/<name>/`, holding `result.md`, every extra checkout, and every file it cites, so `stop` and `sweep` see all of it.
- The helper works in a directory of its own, never a checkout another live session uses (`/bip-issue-work` moves a co-tenant's branch work into a worktree; `reclaim_slot` stops with `NOT FREE` while any process is in the slot). A pooled clone is allowed only when available and held by you first (start, step 2); a `--bg` helper has no tmux window, so the hold is its only in-use mark.
- A worker in a pooled slot does not start helpers: `reclaim_slot` neither removes nor reports their checkouts, so they would outlive the slot. Ask the epic or conductor to start one.
- Delete with `find <path> -delete`, never `rm`: a `$` in the same command as `rm` trips Claude Code's destructive-removal guard, even in bypass mode.
- Kill only by the recorded short id, never `pgrep`/`pkill`, a name, or argv: a spawn prompt is argv, so a pattern can match your own shell.

## Records

```bash
HELPERS="${XDG_STATE_HOME:-$HOME/.local/state}/bip/helpers"
# $HELPERS/<short-id>.json   written by the primary at start:
#   {"id","session_id","name","primary","home","dir","dir_kind":"worktree|held-clone","hold"}
# $HELPERS/<name>/           the helper's home
# $HELPERS/<name>/result.md  written by the helper, one appended section per round
```

## start

1. **Name it and make its home.**
   Your name is the first line of `ListAgents`; the helper is `<primary>-<role>`.
   If `ListAgents` shows that name or `$HELPERS/<primary>-<role>` exists, pick another role word.
   ```bash
   HELPERS="${XDG_STATE_HOME:-$HOME/.local/state}/bip/helpers"
   HOME_DIR="$HELPERS/<primary>-<role>"
   mkdir -p "$HOME_DIR"
   ```

2. **Pick its working directory `DIR`.**
   - Research, review, messaging, or code changes: a detached worktree of your repo off `origin/main`, inside the home, `KIND=worktree`. Not the bare home: `claude --bg` refuses an untrusted directory ("Workspace not trusted"), but a worktree of a trusted repo starts. Tell a research-only helper the checkout is read-only; otherwise it branches there itself (for example via `/bip-issue-work`).
     ```bash
     git -C <repo> fetch -q origin
     DIR="$HOME_DIR/$(basename "$(git -C <repo> rev-parse --show-toplevel)")"
     git -C <repo> worktree add --detach "$DIR" origin/main
     ```
   - Gates or builds that need a built environment (a `.pixi` env, a vendored binary) that a fresh worktree would have to install from scratch: a conductor or epic session with a clone pool can lend an `available` pooled clone (as `/bip-conductor` Step 4 classifies it), held so spawn selection and `bip spawn` skip it, `KIND=held-clone`. Run from your own checkout, where `.epic-config.json` lives; `<this-skill's-base-directory>` is the base directory given at invocation; `lib/spawn-intent.sh` is its sibling.
     ```bash
     source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
     CLONE_ROOT=$(resolve_clone_root .epic-config.json)
     HOLD="$CLONE_ROOT/.holds/<clone>"
     mkdir -p "$CLONE_ROOT/.holds" && echo "helper <primary>-<role>" > "$HOLD"
     DIR="$CLONE_ROOT/<clone>"
     ```
     Write the hold before anything touches the clone, then confirm it is still `available`; if not, delete the hold and pick another.

3. **Write the brief.**
   The helper starts with no other context. Besides the task, give your name (it reports to you by `SendMessage`), its home and working directory as expanded paths, and these rules:
   - extra checkouts and anything it cites go under its home, never its scratchpad;
   - after each round, append a timed section with findings and evidence paths to `result.md`, then report to the primary; if that send fails, the primary was likely renamed on a restart — note it in `result.md` and don't resend; never guess the primary from a name prefix;
   - leave a held clone clean and on `main`;
   - no merging, no pushing to main, no other checkouts, no helpers of its own.

4. **Start it** from `$DIR`, in your own permission mode:
   ```bash
   ARGS=(-n "<primary>-<role>" --permission-mode <your-mode>)
   OUT=$(cd "$DIR" && claude --bg "${ARGS[@]}" "<brief>" 2>&1)
   ID=$(printf '%s\n' "$OUT" | sed -n 's/^backgrounded · \([0-9a-f]\{1,\}\) · .*/\1/p')
   SID=$(claude agents --json --all | jq -r --arg id "$ID" '.[] | select(.id==$id) | .sessionId')
   ```
   If `ID` or `SID` is empty, stop and show the user `$OUT`; do not guess an id from the listing.

5. **Write the record.** `HOLD` is empty unless `KIND=held-clone`.
   ```bash
   jq -n --arg id "$ID" --arg sid "$SID" --arg name "<primary>-<role>" --arg primary "<primary>" \
       --arg home "$HOME_DIR" --arg dir "$DIR" --arg kind "$KIND" --arg hold "${HOLD:-}" \
       '{id:$id, session_id:$sid, name:$name, primary:$primary, home:$home, dir:$dir, dir_kind:$kind, hold:$hold}' \
       > "$HELPERS/$ID.json"
   ```

6. **Confirm messages flow both ways.**
   Wait for its first report, then send it something (a real first task serves) and wait for the answer.
   If either is missing after a few minutes, read `claude logs <id>` and report to the user. A `failed` state in `claude agents --json --all` usually means the wrong account; a silent helper, a held message.

## Finding a helper's record

`stop`, `sweep`, and later turns start from the record, not from leftover shell variables:

```bash
HELPERS="${XDG_STATE_HOME:-$HOME/.local/state}/bip/helpers"
R=$(jq -r --arg n "<primary>-<role>" 'select(.name==$n) | input_filename' "$HELPERS"/*.json)
```

If `R` names no file or more than one, stop and show the user the records.
Read `id`, `session_id`, `home`, `dir`, `dir_kind`, and `hold` from `$R` with `jq -r`.

## Talking to it

- `SendMessage` to its name; replies arrive as cross-session messages.
- Read `result.md` in its home first. Resume it only for a new round or a question the file does not answer.
- If it is missing from `ListAgents`, resume it with the message as the prompt, rebuilding `ARGS` exactly as in start step 4 (without `--permission-mode` its messages are held; without `-n` it is renamed from the prompt):
  ```bash
  OUT=$(cd "$DIR" && claude --bg --resume "$SID" "${ARGS[@]}" "<message>" 2>&1)
  ```
  The resume is a new job: parse the new `ID` and `SID` from `$OUT` as in start step 4, write a new record (start step 5), and delete the old one.
  `jq .respawnFlags ~/.claude/jobs/<id>/state.json` shows the flags a job actually runs with.
- The user can watch or type to it with `claude attach <id>` (`←` returns to agent view, `Ctrl+Z` to the shell; the helper keeps running).

## stop

1. Ask it to commit and push anything it owns, append its last section to `result.md`, and report to you. If it is stopped, resume it for this.
2. `claude stop <id>`, then `claude rm <id>`.
   If `rm` reports unpushed commits or a worktree it could not remove, stop and show the user; do not pass `--discard-unpushed` or `--force-remove-worktree` without their go-ahead.
3. Remove every worktree under the home:
   ```bash
   find "$HOME_DIR" -mindepth 2 -maxdepth 3 -name .git -type f | while read -r g; do
       w=$(dirname "$g"); git -C "$w" worktree remove "$w" || echo "KEEP $w"
   done
   ```
   For each `KEEP` (uncommitted or untracked files), show the user rather than forcing.
4. Never delete a held clone.
   If `git -C "$DIR" status --porcelain` is empty and it is on `main`, lift the recorded hold with `find "$(dirname "$hold")" -maxdepth 1 -name "$(basename "$hold")" -delete`.
   Otherwise keep the hold and show the user what the helper left.
5. The home's evidence may be quoted on GitHub. Show the user the evidence paths `result.md` lists and ask before `find "$HOME_DIR" -delete`; if none are listed and step 3 kept nothing, delete it.
6. Delete the record with `find "$HELPERS" -maxdepth 1 -name "<id>.json" -delete`, and tell the user the helper is gone.

## sweep

For each record in `$HELPERS`, look up its `id` in `claude agents --json --all` and its `primary` in `ListAgents`, and report one line:

- **stale** — id not in the listing: offer to finish `stop` from step 3.
- **primary not found** — session present, no `ListAgents` row with the primary's name. Not proof it is gone: `ListAgents` renames a resumed session (`birch` becomes `birch-61`). Offer to message or stop the helper.
- **live** — both present: leave it.

Report and ask; stop or remove nothing without the user's go-ahead.
Match on the record's `id` only, never a name prefix.
