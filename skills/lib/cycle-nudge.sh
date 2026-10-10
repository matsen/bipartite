#!/usr/bin/env bash
# Claude Code Stop hook: nudge a long session to /bip-cycle.
# Reads the hook's JSON on stdin, takes the context size from the last
# main-thread turn's usage, and past a threshold blocks the stop once per
# 100k tokens of growth so the agent decides whether to cycle now.
set -u
input=$(cat)
[ -n "${TMUX:-}" ] || exit 0  # /bip-cycle needs tmux
[ "$(jq -r '.stop_hook_active // false' <<<"$input")" = true ] && exit 0
transcript=$(jq -r '.transcript_path // empty' <<<"$input")
session=$(jq -r '.session_id // empty' <<<"$input")
cwd=$(jq -r '.cwd // empty' <<<"$input")
[ -f "$transcript" ] && [ -n "$session" ] || exit 0

tokens=$(tail -n 400 "$transcript" | jq -r 'select(.type=="assistant" and (.isSidechain|not) and .message.usage) | .message.usage | (.input_tokens + (.cache_read_input_tokens // 0) + (.cache_creation_input_tokens // 0))' 2>/dev/null | tail -1)
[ -n "$tokens" ] || exit 0

root=$(git -C "$cwd" rev-parse --show-toplevel 2>/dev/null || echo "$cwd")
# A /clear gives the session a new id, which orphans a ralph loop keyed to the old one.
[ -f "$root/.claude/ralph-loop.local.md" ] || [ -f "$cwd/.claude/ralph-loop.local.md" ] && exit 0
# A worker slot (its clone holds .epic-status.json) runs short, so it gets more room.
if [ -f "$root/.epic-status.json" ]; then threshold=${BIP_CYCLE_WORKER_TOKENS:-500000}; else threshold=${BIP_CYCLE_TOKENS:-300000}; fi
state_dir=${XDG_STATE_HOME:-$HOME/.local/state}/bip/cycle-nudge
mkdir -p "$state_dir"
# Below threshold (a compaction can shrink a session in place): forget past nudges.
[ "$tokens" -ge "$threshold" ] || { rm -f "$state_dir/$session"; exit 0; }
bucket=$((tokens / 100000))
last=$(cat "$state_dir/$session" 2>/dev/null || echo -1)
[ "$bucket" -gt "$last" ] || exit 0
echo "$bucket" > "$state_dir/$session"

k=$((tokens / 1000))
jq -n --arg r "This session's context is ${k}k tokens, past the ${threshold%000}k cycle threshold. At a natural boundary, run /bip-cycle: a fresh session costs less per turn and attends better. Wait instead if a background agent or task you are waiting on is still running (cycling strands its result), or if you hold key context a tuckin can't capture yet; say in one line which, write down what you can, and cycle once it is resolved. Either way, knowledge that must outlive the next session belongs in its durable home (EPIC body, paper, issue), not only in the continuation note." '{decision: "block", reason: $r}'
