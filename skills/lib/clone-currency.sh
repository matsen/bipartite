#!/bin/bash
# Pre-spawn currency sweep for the conductor's clone pool.
#
# QUESTION ANSWERED: "is this idle SLOT's tree current with main, so a worker
# spawned into it starts from current code?"
#
# QUESTION *NOT* ANSWERED: "did the repo move under a finished result?" That one
# needs the artifact's own `phyz_build` commit (CLAUDE.md, issue #2250), not a
# clone HEAD, and it is a TREE DIFF -- `git diff <build-commit> <tip> -- src/
# build.zig build.zig.zon` -- not a count. A behind-count cannot report
# DIRECTION: "N behind" after your own PR merges is the ordinary post-land
# state; "N behind" while a result is in flight is the repo-moved-under-a-result
# trap. Same number, opposite meanings.
#
# Usage:  CONDUCTOR=~/re/phyz CLONE_ROOT=~/re/pz ./clone-currency.sh
set -u
CONDUCTOR=${CONDUCTOR:-$HOME/re/phyz}
CLONE_ROOT=${CLONE_ROOT:-$HOME/re/pz}
CONFIG="$CONDUCTOR/.epic-config.json"

# The universe is `clone_names` from .epic-config.json -- NOT every directory
# under CLONE_ROOT. The pool also holds CI clones (nightly-ci,
# beagle-weekly-ci) whose timers `git fetch && git reset --hard origin/main` at
# run time and re-exec themselves, so a stale HEAD at rest is EXPECTED there and
# reporting it as NOT-READY is a false alarm against the wrong population.
mapfile -t SLOTS < <(python3 -c "import json;print('\n'.join(json.load(open('$CONFIG'))['clone_names']))")

# EMPTY DENOMINATOR GUARD. Without this, a missing or unparseable $CONFIG makes
# `python3` fail, `mapfile` yield an empty array, the loop body never execute,
# and the script print `ready=0 not-ready=0` and exit 0 -- which reads as
# "nothing wrong" when the probe never ran. That fails toward "go ahead", the
# exact direction this whole check exists to close. A probe that cannot report
# a positive has established nothing; say so loudly instead of exiting clean.
[ "${#SLOTS[@]}" -gt 0 ] || { echo "FATAL: no clone_names resolved from $CONFIG (missing, unparseable, or empty) -- NOT a clean sweep" >&2; exit 2; }

git -C "$CONDUCTOR" fetch -q origin main || { echo "FATAL: fetch failed"; exit 2; }
TIP=$(git -C "$CONDUCTOR" rev-parse FETCH_HEAD)
echo "tip $(echo "$TIP" | cut -c1-8)  ($(git -C "$CONDUCTOR" show -s --format=%cI "$TIP"))"
echo "slots: ${#SLOTS[@]}"

ready=0; notready=0
for name in "${SLOTS[@]}"; do
  d="$CLONE_ROOT/$name"
  [ -e "$d/.git" ] || { printf '  %-12s MISSING\n' "$name"; notready=$((notready+1)); continue; }
  br=$(git -C "$d" rev-parse --abbrev-ref HEAD)
  head=$(git -C "$d" rev-parse HEAD)
  dirty=$(git -C "$d" status --porcelain | wc -l | tr -d ' ')
  # Count in the CONDUCTOR's object DB: the clone may not have the objects, and
  # its own `origin/main` is stale exactly when it matters. Guard ancestry
  # first -- `A..B` returns a meaningless number when A is not an ancestor of B.
  if git -C "$CONDUCTOR" merge-base --is-ancestor "$head" "$TIP" 2>/dev/null; then
    behind=$(git -C "$CONDUCTOR" rev-list --count "$head..$TIP")
  else
    behind="DIVERGED"
  fi
  if [ "$br" = "main" ] && [ "$dirty" = "0" ] && [ "$behind" = "0" ]; then
    ready=$((ready+1))
  else
    notready=$((notready+1))
    printf '  %-12s NOT-READY  branch=%s dirty=%s behind=%s head=%s\n' \
      "$name" "$br" "$dirty" "$behind" "$(echo "$head" | cut -c1-8)"
  fi
done
echo "ready=$ready not-ready=$notready"

# Informational only: directories in the pool that are not slots. Never gated on.
for d in "$CLONE_ROOT"/*/; do
  d=${d%/}; n=${d##*/}; [ -e "$d/.git" ] || continue
  printf '%s\n' "${SLOTS[@]}" | grep -qx "$n" || printf '  (unmanaged) %s\n' "$n"
done
