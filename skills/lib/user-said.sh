#!/usr/bin/env bash
# Did the user type this phrase in the named session's window?
#
# Usage: user-said.sh <session-name> '<contiguous phrase>'
# Prints the timestamp of each user-typed turn containing the phrase; exits 1 if none.
#
# Only two record shapes are the user's own typing: a plain-string "user" turn
# that is not meta and not a cross-session message, and a mid-turn message queued
# with origin "human". Tool results are also "user" records, so a session that
# merely read a file quoting the user would otherwise match. Text the user pasted
# (<pasted_content> blocks) is someone else's words and is stripped first.
set -euo pipefail
name=$1 phrase=$2

cwds=$(jq -r --arg n "$name" 'select(.name == $n) | .cwd' ~/.claude/sessions/*.json 2>/dev/null | sort -u)
[ -n "$cwds" ] || { echo "no live session named $name" >&2; exit 2; }

found=0
while IFS= read -r cwd; do
  dir=~/.claude/projects/$(printf '%s' "$cwd" | sed 's/[^a-zA-Z0-9]/-/g')
  for f in "$dir"/*.jsonl; do
    [ -e "$f" ] || continue
    grep -qF -- "$phrase" "$f" || continue
    hits=$(jq -r --arg q "$phrase" '
      select(
        (.type == "user" and .isMeta != true
          and (.message.content | type) == "string"
          and (.message.content | gsub("(?s)<pasted_content.*?</pasted_content[^>]*>"; "") | contains($q))
          and (.message.content | contains("<cross-session-message") | not))
        or
        (.type == "attachment" and .attachment.type == "queued_command"
          and .attachment.origin.kind == "human"
          and (.attachment.prompt | tostring | gsub("(?s)<pasted_content.*?</pasted_content[^>]*>"; "") | contains($q)))
      ) | .timestamp' "$f")
    [ -n "$hits" ] && { echo "$hits"; found=1; }
  done
done <<< "$cwds"

[ "$found" = 1 ]
