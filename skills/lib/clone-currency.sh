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
# Usage:  ./clone-currency.sh                      # from your conductor clone
#         ./clone-currency.sh <CONDUCTOR> [CLONE_ROOT]
#         CONDUCTOR=~/re/phyz CLONE_ROOT=~/re/pz ./clone-currency.sh   # still works
set -u

# ⛔ NO HARDCODED FLEET PATH LIVES HERE, DELIBERATELY. This script used to default
# to CONDUCTOR=$HOME/re/phyz and CLONE_ROOT=$HOME/re/pz. Because it takes its
# scope from env vars while its sibling `fleet-collisions.sh` takes a POSITIONAL
# root, a caller who followed the sibling's shape -- `clone-currency.sh "$CLONE_ROOT"` --
# had the argument silently ignored and got a well-formed, entirely plausible
# report about the phyz pool. Measured 2026-09-18 on matsengrp/superfamily-pcp:
# a 17-slot report naming alder/ash/balsa, from a conductor whose pool is
# cobalt/copper/iron. Nothing errors, nothing is empty, and the wrong-fleet
# answer fails toward "go ahead" on the step whose output authorises a spawn.
#
# ⚠ AND THE THREE HELPERS NOW HAVE THREE DIFFERENT SCOPE SHAPES, so do not carry
# any one of them across (issue #252): `bip epic collide` takes NO scope argument
# and derives clone_root from .epic-config.json; `fleet-collisions.sh` REQUIRES a
# positional root and exits 2 on an empty one; this script takes positional, then
# env, then derives. All three print the scope they resolved as their first line
# -- read that line before reading the report, in every case.
#
# ⭐ THE ROOT CAUSE WAS AN INCONSISTENCY INSIDE THIS SCRIPT, not the caller's
# mistake: it already reads `clone_names` from .epic-config.json, but took the
# ROOT from a hardcoded default. Reading both from the same file makes a
# wrong-pool answer unreachable rather than merely discouraged.
#
# Precedence, most explicit first: positional, then environment (back-compat for
# existing callers), then derived from the conductor checkout's own config.
CONDUCTOR=${1:-${CONDUCTOR:-$PWD}}
CONDUCTOR=${CONDUCTOR/#\~/$HOME}
CONFIG="$CONDUCTOR/.epic-config.json"
[ -f "$CONFIG" ] || { echo "FATAL: no .epic-config.json at $CONDUCTOR -- run from a conductor clone, or pass one as \$1. NOT a clean sweep" >&2; exit 2; }

# Resolved in separate steps, NOT as one nested ${2:-${CLONE_ROOT:-$(...)}}: the
# command substitution contains parentheses and quotes, and nesting it inside two
# levels of parameter expansion made bash mis-parse the whole line (it reported
# `:-: command not found` and, under `set -u`, `CLONE_ROOT: unbound variable`).
# Found by running the four cases below, not by reading it.
# ⛔ REFUSE A FROZEN WORKER SNAPSHOT. Every worker clone carries its own
# .epic-config.json, taken at spawn time and never updated, and $PWD will happily
# read one -- so the $PWD fallback above reintroduced this file's own bug one level
# up. Measured 2026-09-18 on ~/re/sfpcp: zinc's copy is from 2026-05-12 and lists 7
# clone_names against the conductor's 9. Run no-arg from zinc and the UNGUARDED
# version returned exit 0, a well-formed 7-slot sweep of a 9-slot fleet, and
# reported `tin` -- the slot running that EPIC's top line -- as "(unmanaged)".
# Nothing errors, nothing is empty. Same fail-toward-"go ahead" shape as the
# hardcoded default this patch removed, with the pool under-reported rather than
# swapped. iron's and nickel's copies happen to be caught by the empty-clone_names
# guard below; zinc's is not. CLAUDE.md #338 already says never to trust a clone's
# own copy.
#
# REFUSE rather than silently re-resolve from main_checkout. Re-resolving works on
# every sample we have, but it makes the script follow a pointer inside the very
# file it just decided not to trust, and n=3 does not earn that. Refusing keeps the
# fail-closed shape the rest of this patch is about. An empty or absent
# main_checkout passes, so older configs without the key still work.
# The compare is literal, not path-normalised: a symlinked or trailing-slash
# CONDUCTOR yields a spurious FATAL naming the path to pass as $1, which is a loud
# and self-describing failure rather than a silent wrong-pool sweep.
MAIN=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get('main_checkout') or '')" "$CONFIG" 2>/dev/null) || MAIN=""
[ -z "$MAIN" ] || [ "$MAIN" = "$CONDUCTOR" ] || {
  echo "FATAL: $CONFIG is a frozen worker snapshot (main_checkout=$MAIN, reading from $CONDUCTOR); run from $MAIN or pass it as \$1 -- NOT a clean sweep" >&2; exit 2; }

CLONE_ROOT_ARG=${2:-}
CLONE_ROOT_ENV=${CLONE_ROOT:-}
if [ -n "$CLONE_ROOT_ARG" ]; then CLONE_ROOT="$CLONE_ROOT_ARG"
elif [ -n "$CLONE_ROOT_ENV" ]; then CLONE_ROOT="$CLONE_ROOT_ENV"
else CLONE_ROOT=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get('clone_root') or '')" "$CONFIG" 2>/dev/null) || CLONE_ROOT=""
fi
CLONE_ROOT=${CLONE_ROOT/#\~/$HOME}
# Same empty-denominator reasoning as the clone_names guard below: `:-` substitutes
# on unset OR EMPTY, so an unparseable config or a missing clone_root key would
# otherwise flow onward as "" and resolve every clone path against /.
[ -n "$CLONE_ROOT" ] && [ -d "$CLONE_ROOT" ] || { echo "FATAL: clone_root '$CLONE_ROOT' missing or not a directory (from $CONFIG) -- NOT a clean sweep" >&2; exit 2; }

echo "scope: conductor=$CONDUCTOR clone_root=$CLONE_ROOT"

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
  # Read status BEFORE counting. `git ... | wc -l` cannot tell "clean tree" from
  # "git failed": both give an empty pipe and 0, and 0 is what marks this clone
  # READY below -- so an unreadable clone would be handed to a new worker as
  # clean. Fail toward NOTREADY, the same direction the ancestry guard below
  # fails toward for the same reason.
  if dirty_out=$(git -C "$d" status --porcelain 2>/dev/null); then
    dirty=$(printf '%s' "$dirty_out" | grep -c . | tr -d ' ')
  else
    dirty="UNREADABLE"
  fi
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
