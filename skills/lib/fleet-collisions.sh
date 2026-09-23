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
#   2b. a live branch touching a file that LANDED on main since that branch
#      forked. Section 2's universe is live-vs-live; this is live-vs-landed,
#      a different frame. Measured 2026-09-14: a PR went CONFLICTING against
#      one that merged two hours earlier and section 2 reported clean.
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
# Measured 2026-09-14: read as 0 when the real status was 1. Use
# `"${PIPESTATUS[0]}"`, or redirect to a file and run it un-piped.
#
# Run it from a conductor's own pane. A non-interactive context (a systemd
# timer, say) has no tmux socket, so PANES is empty every time and this
# exits 2 on every run -- correct, but a timer would swallow it.
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
# Usage: fleet-collisions.sh <clone-root>      (required; no default)
# Exit:  0 = nothing found, 1 = something found, 2 = could not check
set -uo pipefail
# No default: an omitted or empty root used to fall back to ~/re/pz, which
# silently reported on another repository's pool.
ROOT="${1:-}"
[ -n "$ROOT" ] || { echo "usage: fleet-collisions.sh <clone-root> -- NOT a clean check" >&2; exit 2; }
[ -d "$ROOT" ] || { echo "no clone root at $ROOT" >&2; exit 2; }
# A clone root that EXISTS but is EMPTY has to be handled before the three
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
# exit clean instead of letting three loops discover it three different ways.
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
TMP=$(mktemp) || exit 2; trap 'rm -f "$TMP"' EXIT
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
# someone adds cannot get the precedence wrong -- the same argument as the
# denominator comment further down.
# THE POOL'S OWN REMOTE. A clone root can legitimately hold clones of OTHER
# repositories -- matsengrp/phyz's pool holds `ash-iqtree-a00094e0`, a pinned
# checkout of matsen/iqtree2 that experiments reference by absolute path. Those
# are not slots and must not be reported as anything.
#
# Invisible until the detached-HEAD path started reporting weird states loudly:
# that IQ-TREE clone sits at a detached tag commit permanently, so it emitted
# `UNCHECKABLE ... detached HEAD` on every run, setting uncheckable=1 and making
# EVERY run exit 2 -- destroying the signal the stickiness fix had been added to
# protect, one commit after adding it.
#
# Modal origin across the pool rather than the conductor's own: this script
# takes the root as an argument and may be run from anywhere.
POOL_ORIGIN=$(for _d in "$ROOT"/*/; do git -C "$_d" remote get-url origin 2>/dev/null; done \
              | sort | uniq -c | sort -rn | head -1 | sed 's/^ *[0-9]* //')

# is_pool_clone <dir> -- true when <dir>'s origin matches the pool's modal one.
# Returns TRUE when POOL_ORIGIN could not be determined: an unknown pool must
# not silently exclude every clone. That is not hypothetical -- an earlier draft
# of this call site ran with the function undefined, `|| continue` fired on the
# "command not found" status, and the whole section skipped every slot while
# printing nothing but errors. Fail toward checking, never toward skipping.
is_pool_clone() {
  [ -n "$POOL_ORIGIN" ] || return 0
  [ "$(git -C "$1" remote get-url origin 2>/dev/null)" = "$POOL_ORIGIN" ]
}

uncheckable=0
found_any=0

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
  } | sort -u | while IFS= read -r f; do [ -n "$f" ] && printf '%s\t%s\n' "$n" "$f"; done >> "$TMP"
  cnt=0; pfx="$n$(printf '\t')"
  while IFS= read -r l; do case "$l" in "$pfx"*) cnt=$((cnt+1));; esac; done < "$TMP"
  printf "  %-10s %-34s %s files\n" "$n" "$b" "$cnt"
done
[ "$any_live" = 0 ] && echo "  none"

echo
echo "=== files touched by MORE THAN ONE live clone (the 4b gap) ==="
if awk -F'\t' '{c[$2]=c[$2]" "$1} END{f=0; for(k in c){n=split(c[k],a," ");
      if(n>1){print "  COLLISION "k" <-"c[k]; f=1}} exit !f}' "$TMP" | sort; then
  found_any=1
else
  echo "  none"
fi

echo
echo "=== live branch vs. what LANDED since it forked (the 4b gap's OTHER half) ==="
# Section 2 compares live clones against EACH OTHER. It is silent about a
# branch that conflicts with a commit ALREADY ON main -- which is a different
# universe, not a weaker version of the same one.
#
# Measured 2026-09-14 (matsengrp/phyz): PR #2653 went CONFLICTING against
# PR #2648, which had merged two hours earlier and edited the same
# `docs/ml/knob-correspondence.md` row the live branch was marking. Section 2
# reported clean and was CORRECT about what it measures. Recently-landed
# commits were outside its frame, so it was SILENT rather than wrong -- and
# silence reads as safety.
#
# The generalizable question a future editor should ask before adding a
# section here is not "is my check correct?" but "what is this check's
# DENOMINATOR, and does it contain the thing that actually fires?"
#
# TWO THINGS THIS SECTION DOES THAT THE OBVIOUS VERSION DOES NOT:
#
# 1. It handles a clone that is MID-REBASE. Section 1 skips any clone whose
#    `git branch --show-current` is empty -- which is exactly the state a
#    conflicted clone is in, i.e. the one you most want checked. The branch
#    name is recovered from `.git/rebase-merge/head-name` and the comparison
#    runs against the REMOTE ref, which is stable while a local rebase churns.
#    A local HEAD is not a stable referent while another session is rebasing.
#
# 2. It PRINTS THE DENOMINATORS on a clear result. `clear (mine=8 landed=0)`
#    says nothing has landed since that branch forked; `clear (mine=36
#    landed=44)` says the check had real data on both sides and found no
#    overlap. Without them, a trivially-clear result, a genuinely-clear
#    result, and a BROKEN check all print the same word. The first draft of
#    this section derived branch names from `branch --show-current`, returned
#    empty for the mid-rebase clone, produced two empty diffs, and reported
#    `clear` for the one slot whose PR was CONFLICTING at that moment -- a
#    command error laundered into a reassuring result.
found_landed=0
any_checked=0
for d in "$ROOT"/*/; do
  n=$(basename "${d%/}")
  # Skip clones of OTHER repositories living in the same root -- see
  # POOL_ORIGIN above. Without this, the pool's pinned IQ-TREE checkout sits on
  # a permanent detached HEAD, trips the detached-HEAD report below, and sets
  # uncheckable=1 on EVERY run -- destroying the "could not check" signal one
  # commit after the stickiness fix was added to protect it.
  is_pool_clone "$d" || continue
  b=$(git -C "$d" branch --show-current 2>/dev/null)
  # ASK GIT FOR THE GIT DIR; DO NOT ASSUME THE LAYOUT. In a clone `.git` is a
  # directory, but in a WORKTREE it is a FILE containing `gitdir: ...`, so
  # `$d/.git/<rebase-dir>` does not resolve and `[ -d "$d/.git" ]` is FALSE.
  # Both of the weird-state paths below would then silently skip a worktree --
  # reinstating, in a supported mode, the exact gap they exist to close.
  # `skills/bip-epic/SKILL.md` already records this trap ("in a worktree
  # `.git` is a file, not a directory"), where it produced a correct verdict
  # on a false premise and nearly authorised deleting two branches.
  # `--absolute-git-dir` is the one form that works in both layouts.
  gd=$(git -C "$d" rev-parse --absolute-git-dir 2>/dev/null)
  if [ -z "$b" ] && [ -n "$gd" ]; then
    # Git has TWO rebase backends and they store head-name in different
    # directories: rebase-merge/ (the merge backend) and rebase-apply/
    # (--apply / am-based). Covering only one leaves the other as a silent
    # skip, which is the exact failure this section exists to stop.
    for rd in rebase-merge rebase-apply; do
      if [ -r "$gd/$rd/head-name" ]; then
        b=$(sed 's#^refs/heads/##' "$gd/$rd/head-name" 2>/dev/null)
        [ -n "$b" ] && echo "  NOTE $n is mid-rebase; using branch '$b' from $rd/head-name"
        break
      fi
    done
  fi
  if [ -z "$b" ]; then
    # Detached and NOT rebasing -- e.g. a bisect, or a hand checkout of a
    # candidate commit. Silently skipping it would repeat this section's own
    # mistake one notch along: the clone in the weird state is the one you
    # most want checked.
    if [ -n "$gd" ]; then
      echo "  UNCHECKABLE $n: detached HEAD, no rebase in progress -- cannot determine its branch" >&2
      uncheckable=1
    fi
    continue
  fi
  [ "$b" = main ] && continue
  # Compare the REMOTE ref: a local rebase rewrites HEAD under us.
  if ! git -C "$d" rev-parse --verify --quiet "origin/$b" >/dev/null 2>&1; then
    echo "  UNCHECKABLE $n: no origin/$b (branch never pushed) -- cannot compare against landed work" >&2
    uncheckable=1; continue
  fi
  base=$(git -C "$d" merge-base "origin/$b" origin/main 2>/dev/null)
  if [ -z "$base" ]; then
    echo "  UNCHECKABLE $n: no merge-base for origin/$b and origin/main" >&2
    uncheckable=1; continue
  fi
  any_checked=1
  mine=$(git -C "$d" diff --name-only "$base" "origin/$b" 2>/dev/null | sort -u)
  landed=$(git -C "$d" diff --name-only "$base" origin/main 2>/dev/null | sort -u)
  ov=$(comm -12 <(printf '%s\n' "$mine") <(printf '%s\n' "$landed") | grep -v '^$')
  nm=$(printf '%s\n' "$mine" | grep -c .)
  nl=$(printf '%s\n' "$landed" | grep -c .)
  if [ -n "$ov" ]; then
    echo "  OVERLAPS-LANDED $n: $(echo $ov)"
    echo "    -> rebase and take main's content for those files; your copy predates the merge."
    echo "       Verify with: git -C $d diff origin/main -- <file>   (should show ONLY your additions)"
    found_landed=1
  elif [ "$nm" = 0 ]; then
    # A live branch that has changed NOTHING relative to its fork point is
    # either not started or not measurable, and both mean a clear result here
    # is worthless. The denominators alone made this visible to a human who
    # reads and thinks; they did not make the SCRIPT say so, which is the
    # fail-open one level in.
    echo "  NO-COMMITS $n (mine=0, landed=$nl) -- nothing to compare; do NOT read this as clear" >&2
    uncheckable=1
  else
    echo "  $n clear (mine=$nm landed=$nl)"
  fi
done
[ "$any_checked" = 0 ] && echo "  (no pushed live branches to check)"
[ "$found_landed" = 1 ] && found_any=1

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
