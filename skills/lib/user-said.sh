#!/usr/bin/env bash
# Did the user type this phrase in the named session's window?
#
# Usage: user-said.sh <session-name> '<contiguous phrase>'
# Prints the timestamp of each user-typed turn containing the phrase; exits 1 if none.
#
# A window is the set of transcripts that carry a custom-title record with its name
# (each /rename and each cycle writes one); sessions sharing a cwd share a project
# dir, so the dir alone does not identify the window. A window never renamed fails.
#
# Only two record shapes are the user's own typing: a plain-string "user" turn that
# is not meta, and a mid-turn message queued with origin "human". Tool results,
# subagent and Monitor notifications, and command output are also "user" records,
# so after stripping pasted text and system reminders, a turn that opens with a tag
# counts only if the tag is one the user types through (<bash-input>, <command-...>).
set -euo pipefail
name=$1 phrase=$2

files=$(grep -lF -- "\"customTitle\":$(jq -Rn --arg n "$name" '$n')" ~/.claude/projects/*/*.jsonl 2>/dev/null || true)

found=0
for f in $files; do
  grep -qF -- "$phrase" "$f" || continue
  hits=$(jq -r --arg q "$phrase" '
    def typed: gsub("(?s)<pasted_content.*?</pasted_content[^>]*>"; "")
      | gsub("(?s)<system-reminder>.*?</system-reminder>"; "")
      | ltrimstr("\n") | sub("^\\s+"; "")
      | select((startswith("<") | not) or startswith("<bash-input>") or startswith("<command-"));
    select(
      (.type == "user" and .isMeta != true
        and (.message.content | type) == "string"
        and ([.message.content | typed | contains($q)] | any))
      or
      (.type == "attachment" and .attachment.type == "queued_command"
        and .attachment.origin.kind == "human"
        and ([.attachment.prompt | tostring | typed | contains($q)] | any))
    ) | .timestamp' "$f")
  [ -n "$hits" ] && { echo "$hits"; found=1; }
done

[ "$found" = 1 ]
