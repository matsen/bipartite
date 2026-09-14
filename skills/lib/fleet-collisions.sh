#!/usr/bin/env bash
# Conductor-only pre-spawn check. The epic side cannot implement this: clone
# branches are local (`shared_filesystem: false`), and remote refs are not a
# substitute -- under squash-merge every historical branch stays permanently
# ahead of `main` (measured on matsengrp/phyz: 847 remote branches, the first
# 400 all ahead), so "ahead of main" does not discriminate live from dead.
# The decisive case is UNCOMMITTED work, which exists only in the clone.
#
# Reports, and exits non-zero if ANY of the three fires:
#   1. every live branch across the pool and the files it touches
#   2. files touched by MORE THAN ONE live clone        <- permits a collision
#   3. a .epic-status.json whose clone has no live pane <- suppresses a spawn
#
# On (3) the usual action is DELETING a state file, so it uses prefix
# containment on pane cwd, not equality: a worker that `cd`s into a
# subdirectory is still in its clone and must not be reported stale.
#
# `git diff --name-only origin/main` already covers uncommitted TRACKED
# changes -- it diffs the working tree against that commit. The
# `status --porcelain -z` pass is there for UNTRACKED files only, and is
# NUL-delimited because git quotes paths containing spaces and a
# whitespace-split would mangle them into a silent non-match.
#
# Scope: written for and exercised only against matsengrp/phyz's ~/re/pz pool.
# The clone-root argument makes it portable in principle; that is untested.
#
# Usage: fleet-collisions.sh [clone-root]      (default ~/re/pz)
# Exit:  0 = nothing found, 1 = something found, 2 = could not check
set -uo pipefail
ROOT="${1:-$HOME/re/pz}"
[ -d "$ROOT" ] || { echo "no clone root at $ROOT" >&2; exit 2; }
TMP=$(mktemp) || exit 2; trap 'rm -f "$TMP"' EXIT
mapfile -t PANES < <(tmux list-panes -a -F '#{pane_current_path}' 2>/dev/null)
have_panes=1; [ "${#PANES[@]}" -eq 0 ] && have_panes=0
rc=0

echo "=== live branches and their touched files ==="
any_live=0
for d in "$ROOT"/*/; do
  n=$(basename "${d%/}")
  b=$(git -C "$d" branch --show-current 2>/dev/null) || continue
  { [ -z "$b" ] || [ "$b" = main ]; } && continue
  any_live=1
  git -C "$d" fetch -q origin main 2>/dev/null \
    || echo "  WARN $n: fetch failed; origin/main may be stale -> false collisions" >&2
  # Diff against the MERGE BASE, not origin/main: a branch even one commit
  # behind otherwise reports every file main changed since the branch point
  # as its own. Measured 2026-09-14: a slot 1 commit behind reported 15 files
  # when it had touched 6, and the 9 phantoms produced a false collision.
  base=$(git -C "$d" merge-base HEAD origin/main 2>/dev/null) || base=origin/main
  { git -C "$d" diff --name-only "$base" 2>/dev/null
    git -C "$d" status --porcelain -z 2>/dev/null \
      | while IFS= read -r -d '' e; do
          printf '%s\n' "${e:3}"
          # R/C entries emit a SECOND NUL field holding the OLD path, whole
          # (no 3-char status prefix). Both paths are real and either can
          # collide, so emit both rather than stripping the second.
          case "$e" in R*|C*) IFS= read -r -d '' old && printf '%s\n' "$old";; esac
        done
  } | sort -u | grep -v '^$' | sed "s|^|$n\t|" >> "$TMP"
  cnt=$(grep -c "^$n	" "$TMP" 2>/dev/null); cnt=${cnt:-0}
  printf "  %-10s %-34s %s files\n" "$n" "$b" "$cnt"
done
[ "$any_live" = 0 ] && echo "  none"

echo
echo "=== files touched by MORE THAN ONE live clone (the 4b gap) ==="
if awk -F'\t' '{c[$2]=c[$2]" "$1} END{f=0; for(k in c){n=split(c[k],a," ");
      if(n>1){print "  COLLISION "k" <-"c[k]; f=1}} exit !f}' "$TMP" | sort; then
  rc=1
else
  echo "  none"
fi

echo
echo "=== stale .epic-status.json (status file, no live pane in that clone) ==="
# Builtin glob, not `ls`: an external that fails for any reason would
# short-circuit this guard and let the script mark the whole pool STALE.
shopt -s nullglob; _statfiles=("$ROOT"/*/.epic-status.json); shopt -u nullglob
if [ "$have_panes" -eq 0 ] && [ "${#_statfiles[@]}" -gt 0 ]; then
  echo "  CANNOT CHECK: tmux returned no panes, and status files exist." >&2
  echo "  Refusing to report STALE for the whole pool -- the action on STALE is deletion." >&2
  exit 2
fi
found=0
for d in "$ROOT"/*/; do
  [ -f "$d/.epic-status.json" ] || continue
  p=$(realpath "${d%/}" 2>/dev/null) || continue
  live=0
  for pane in "${PANES[@]}"; do
    rp=$(realpath "$pane" 2>/dev/null) || continue
    case "$rp" in "$p"|"$p"/*) live=1; break;; esac
  done
  [ "$live" = 1 ] && continue
  echo "  STALE $(basename "$p")  issue=$(jq -r '.issue//"?"' "$d/.epic-status.json" 2>/dev/null) updated=$(jq -r '.updated_at//"?"' "$d/.epic-status.json" 2>/dev/null)"
  found=1
done
[ "$found" = 1 ] && rc=1 || echo "  none"

echo
echo "=== live pane, NO .epic-status.json (invisible to state-file sweeps) ==="
found2=0
RROOT=$(realpath "$ROOT" 2>/dev/null) || RROOT="$ROOT"
for pane in "${PANES[@]}"; do
  rp=$(realpath "$pane" 2>/dev/null) || continue
  case "$rp" in "$RROOT"/*) ;; *) continue;; esac
  rest=${rp#"$RROOT"/}; clone=${rest%%/*}
  [ -d "$ROOT/$clone" ] || continue
  [ -f "$ROOT/$clone/.epic-status.json" ] && continue
  echo "  NO-STATUS $clone"; found2=1
done
[ "$found2" = 1 ] && rc=1 || echo "  none"

exit "$rc"
