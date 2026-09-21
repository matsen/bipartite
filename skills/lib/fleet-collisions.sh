#!/usr/bin/env bash
# Conductor-only pre-spawn check: the PANE AND PROCESS half.
#
# ⛔ THE FILE-OVERLAP HALF OF THIS SCRIPT IS NOW `bip epic collide`. Sections
# 1, 2 and 2b (live branches and their touched files; files touched by more
# than one live clone; a live branch versus what LANDED since it forked) were
# ported to Go in issue #252 and removed from here. Run BOTH:
#
#   bip epic collide                 # 0 = clear, 10 = found, 11 = could not check
#   fleet-collisions.sh "$ROOT"      # 0 = clear, 1 = found,  2 = could not check
#
# The port was not a translation. The command derives its clone-pool root from
# `.epic-config.json` instead of defaulting to one, which is the defect this
# script had at its `ROOT="${1:-$HOME/re/pz}"` line -- measured 2026-09-17 on
# matsengrp/superfamily-pcp, an argument-less run returned a well-formed,
# entirely plausible report about ANOTHER repository's clone pool and missed a
# real three-way collision. This file's own root is now REQUIRED for the same
# reason; see below.
#
# WHY THESE THREE SECTIONS STAYED IN SHELL: they need a tmux socket, a process
# subtree walk, and `jq`. They are about this MACHINE's live state, not about
# the EPIC's branches, so they do not share the command's scope model.
#
# Reports, and exits non-zero if ANY of the three fires:
#   3. a .epic-status.json whose clone has no live pane <- suppresses a spawn
#   4. a live pane with no status file  <- invisible to state-file sweeps
#   5. a pane whose agent session is DEAD <- both artifacts present, healthy-
#      looking, and the work inside may be uncommitted. Measured 2026-09-14:
#      a slot exited cleanly mid-issue with 36 KB uncommitted across 4 files;
#      sections 3 and 4 both reported clean because the pane AND the status
#      file were present. A tmux pane outlives the session inside it.
#
# THE REBOOT CASE IS WHY (3) HAS A GUARD, AND IT IS NOT A DEFENSIVE EDGE.
# After a reboot tmux is gone and every .epic-status.json persists on disk.
# That is the one state with a purpose-built recovery path -- see
# `bip-conductor-recover`, scoped to "a box running a bip-conductor fleet
# rebooted and the tmux sessions are gone" -- and those files are what it
# reads to rebuild the workspace. Without the guard this check declares the
# whole pool stale and the documented action deletes the input to its own
# recovery skill. Rarity makes it worse: nobody is watching for it.
#
# VERIFYING THIS SCRIPT'S OWN EXIT CODE: do not read `$?` after a pipeline.
# Anyone checking these exit codes will reach for `... | sed -n '/foo/,/bar/p'`
# to read the output, and `$?` then reports SED's status, not this script's.
# Measured 2026-09-14: read as 0 when the real status was 1. REDIRECT TO A FILE
# AND RUN IT UN-PIPED. The `"${PIPESTATUS[0]}"` form this header used to
# prescribe is itself fail-open in the fleet's login shell -- zsh spells it
# `$pipestatus[1]`, and evaluates `${PIPESTATUS[0]}` to the EMPTY STRING rather
# than erroring -- and both spellings are rebuilt by every command, so the read
# is only valid as the very next thing after the pipeline. A redirect has no
# window between producing the status and reading it. See
# `skills/bip-conductor/SKILL.md`.
#
# Run it from a conductor's own pane. A non-interactive context (a systemd
# timer, say) has no tmux socket, so PANES is empty every time and this
# exits 2 on every run -- correct, but a timer would swallow it.
#
# On (3) the usual action is DELETING a state file, so it uses prefix
# containment on pane cwd, not equality: a worker that `cd`s into a
# subdirectory is still in its clone and must not be reported stale.
#
# Usage: fleet-collisions.sh <clone-root>      (REQUIRED)
# Exit:  0 = nothing found, 1 = something found, 2 = could not check
set -uo pipefail
# ⛔ NO DEFAULT, DELIBERATELY. This used to be `ROOT="${1:-$HOME/re/pz}"`, and
# `${x:-y}` substitutes on unset OR EMPTY -- so an unparseable config upstream
# made an explicitly-passed root evaporate into another repository's clone pool
# and returned the reassuring answer about the wrong fleet. An empty root is not
# a root. `bip epic collide` derives its scope from `.epic-config.json`; this
# script takes it as an argument because it is also run against pools with no
# config, but it will not invent one.
ROOT="${1:-}"
[ -n "$ROOT" ] || { echo "FATAL: no clone root given. Usage: fleet-collisions.sh <clone-root> -- there is no default, and an empty argument is not a root. NOT a clean check" >&2; exit 2; }
[ -d "$ROOT" ] || { echo "no clone root at $ROOT" >&2; exit 2; }
# A clone root that EXISTS but is EMPTY has to be handled before the
# `for d in "$ROOT"/*/` loops below, because that idiom behaves differently and
# badly in both shells when nothing matches. Measured 2026-09-14 on an empty
# directory:
#   zsh  -> `no matches found: <root>/*/`; the statement ERRORS and the loop
#           body never runs
#   bash -> passes the literal unexpanded `<root>/*/` in as `$d`; ONE SILENT
#           ITERATION on a path that does not exist
# Reachable exactly when someone runs the conductor for the first time. Each
# loop below happens to degrade safely in bash today (a failing `git`/`rev-parse`
# sends it to `continue`), but that is ACCIDENTAL, it is absent in zsh, and an
# empty pool is a legitimate state rather than an error -- so say so once and
# exit clean instead of letting the loops discover it several different ways.
# `find`, not a glob, for the same reason the loops are the problem. Tested
# against `bfs 4.1.1`, this fleet's `find`, not GNU findutils.
# `! -name '.*'` matters: `*/` does not match dot-directories but `find -type d`
# does, and a real pool root holds `.spawn-prompts/`, `.preserved/`, `.mirror/`.
# Without it the guard passes on a pool with no clones but some machinery dirs,
# and the loops error anyway. The guard and the glob must share a universe.
if [ -z "$(find "$ROOT" -maxdepth 1 -mindepth 1 -type d ! -name '.*' -print -quit 2>/dev/null)" ]; then
  echo "no clones under $ROOT yet -- nothing to check"
  exit 0
fi
mapfile -t PANES < <(tmux list-panes -a -F '#{pane_current_path}' 2>/dev/null)
mapfile -t PANE_PP < <(tmux list-panes -a -F '#{pane_current_path} #{pane_pid}' 2>/dev/null)

# Is there a live agent runner (claude or agy) anywhere in this pane's process subtree?
# `#{pane_current_command}` is NOT usable for this: a busy session shows the
# shell it is running a command through (measured: a session reporting BUSY
# showed `bash`), and an exited session leaves the pane at `zsh` -- the two
# are indistinguishable. A tmux pane outlives the agent session inside it,
# so pane-exists is not session-alive.
# 0 = a live agent is in the subtree, 1 = none found, 2 = COULD NOT DETERMINE.
# The 2 case matters: the action on a DEAD-SESSION report is resuming or
# reclaiming a clone, so an unreadable subtree (a pane owned by another user,
# a /proc race as a process exits) must not be reported as dead. Same
# asymmetry as the tmux guard's exit 2.
pane_has_agent() {
  local queue=("$1") pid kids seen=0
  while [ "${#queue[@]}" -gt 0 ]; do
    pid="${queue[0]}"; queue=("${queue[@]:1}")
    local comm; comm=$(ps -o comm= -p "$pid" 2>/dev/null)
    if [ -z "$comm" ]; then
      # the root pane pid itself unreadable -> cannot determine
      [ "$seen" -eq 0 ] && return 2
      continue
    fi
    seen=1
    case "${comm##*/}" in *claude*|agy) return 0;; esac
    mapfile -t kids < <(pgrep -P "$pid" 2>/dev/null)
    [ "${#kids[@]}" -gt 0 ] && queue+=("${kids[@]}")
  done
  return 1
}
have_panes=1; [ "${#PANES[@]}" -eq 0 ] && have_panes=0
# EXIT STATUS: two independent flags, resolved ONCE at the bottom.
#
# This used to be a single `rc` that every section assigned directly, and a
# plain `found_any=1` in ANY LATER SECTION silently overwrote an earlier `uncheckable=1`.
# "I could not check" became "I checked and found a problem": the caller saw
# 1, concluded the script ran fine, and never learned a clone was never
# examined. The script's own header says 2 means cannot-establish, so the
# downgrade destroyed the more important of the two signals.
#
# Never write `rc=` in a new section. Set `uncheckable=1` for "a clone could
# not be examined" and `found_any=1` for "a real problem was found". The
# resolution below makes 2 dominate 1 structurally, so the next section
# someone adds cannot get the precedence wrong. `bip epic collide` carries the
# same contract with the same reasoning, at 11 and 10.
uncheckable=0
found_any=0

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
[ "$found" = 1 ] && found_any=1 || echo "  none"

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
[ "$found2" = 1 ] && found_any=1 || echo "  none"

echo
echo "=== pane alive but agent session is DEAD (work may be uncommitted) ==="
found3=0
if [ "$have_panes" -eq 1 ]; then
  for row in "${PANE_PP[@]}"; do
    ppath="${row% *}"; ppid="${row##* }"
    case "$(realpath "$ppath" 2>/dev/null)" in "$RROOT"/*) ;; *) continue;; esac
    pane_has_agent "$ppid"; pha=$?
    [ "$pha" -eq 0 ] && continue
    if [ "$pha" -eq 2 ]; then
      rest=$(realpath "$ppath" 2>/dev/null); rest="${rest#"$RROOT"/}"
      echo "  CANNOT DETERMINE ${rest%%/*}  pane_pid=$ppid (subtree unreadable) -- NOT reporting dead" >&2
      uncheckable=1; continue
    fi
    rest=$(realpath "$ppath" 2>/dev/null); rest="${rest#"$RROOT"/}"; clone="${rest%%/*}"
    # Read status BEFORE counting. Piping git straight into `wc -l` cannot tell
    # "clean tree" from "git failed" -- both give an empty pipe and 0 -- and here
    # 0 prints as `uncommitted=0`, which reads as "nothing to preserve" on the
    # one line that exists to say otherwise. Report UNKNOWN and set uncheckable,
    # per this file's own "fail toward checking, never toward skipping".
    # `skills/lib/hazards/counting-a-failed-command.md`.
    if dirty_out=$(git -C "$RROOT/$clone" status --porcelain 2>/dev/null); then
      dirty=$(printf '%s' "$dirty_out" | grep -c . || true)
    else
      dirty="UNKNOWN (git status failed -- assume work is present)"
      uncheckable=1
    fi
    echo "  DEAD-SESSION $clone  pane_pid=$ppid  uncommitted=$dirty"
    echo "    -> preserve worklog/status/diff BEFORE anything else; the pane's"
    echo "       last line usually carries resume command ('claude --resume <id>' or 'agy --conversation <id>')."
    found3=1
  done
fi
[ "$found3" = 1 ] && found_any=1 || echo "  none"

# 2 (could not establish) DOMINATES 1 (found something), because a caller
# acting on 1 believes the pool was fully examined.
if [ "$uncheckable" = 1 ]; then exit 2
elif [ "$found_any" = 1 ]; then exit 1
else exit 0
fi
