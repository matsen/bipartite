# Sourceable helpers for resolving .epic-config.json's clone_root and
# locating spawn-intent files under either live naming convention.
#
# Shared by skills/bip-epic (write side, Step 6) and
# skills/bip-conductor-spawn (read side, "Where the prompt comes from") —
# extracted per issue #195 after the same two bugs (missing tilde
# expansion, single-pattern glob) were independently fixed three times
# across the split skills.
#
# Usage:
#   source "<path-to-this-file>"
#   CLONE_ROOT=$(resolve_clone_root .epic-config.json)
#   INTENT=$(find_spawn_intent "$CLONE_ROOT" 302)

# resolve_clone_root <config-file>
# Reads .clone_root from the given JSON config and tilde-expands it.
# Defaults to .epic-config.json in the current directory. Fails with a
# message on stderr if the file is unreadable or .clone_root is
# missing/null, rather than silently resolving to an empty string.
resolve_clone_root() {
    local config_file="${1:-.epic-config.json}"
    local root
    root=$(jq -r '.clone_root' "$config_file") || return 1
    if [ -z "$root" ] || [ "$root" = "null" ]; then
        echo "resolve_clone_root: no .clone_root in $config_file" >&2
        return 1
    fi
    echo "$root" | sed "s|^~|$HOME|"
}

# find_spawn_intent <clone_root> <issue-number>
# Locates a spawn-intent file for the given issue number under either
# naming convention live in .spawn-prompts/: "<N>.md" (current) or
# "spawn-<N>.txt" (older, still written by some sessions). Prefers
# "<N>.md" when both exist — relying on `ls`'s alphabetical ordering of
# its (unglobbed) arguments, where a leading digit sorts before "s", not
# on argument order. Prints the path, or nothing if neither exists.
find_spawn_intent() {
    local clone_root="$1"
    local issue_number="$2"
    ls "$clone_root/.spawn-prompts/$issue_number.md" \
       "$clone_root/.spawn-prompts/spawn-$issue_number.txt" 2>/dev/null | head -1
}

# preserve_epic_state <source_dir> <clone_root> [reason]
# Copies <source_dir>/.epic-status.json and .epic-worklog.md, if
# present, to <clone_root>/.preserved/<issue>-<date>/ with un-dotted,
# clone-namespaced filenames (i<issue>-<clone>.worklog.md/.status.json,
# matching this repo's own .preserved/ convention) and a provenance
# README recording <reason>. Extracted per issue #2216 after the same
# preserve-then-delete pipeline was independently written -- and
# independently mis-written (unchecked cp, wrong-order
# resolve_clone_root, guard-tripping wording, an &&-chained cp pair
# that dropped the status file entirely whenever the worklog was
# merely absent, and a same-day same-issue collision that silently
# overwrote an earlier preserved copy -- all caught before this landed,
# not after) -- four times across bip-pr-land, bip-conductor-spawn (x2),
# and bip-conductor-poll.
#
# The two copies are attempted independently rather than &&-chained:
# .epic-status.json is required (its presence is the whole precondition
# for calling this function), but .epic-worklog.md may legitimately not
# exist yet (an early-crash slot, or the window between the session-start
# status write and the first worklog append), and that must not cost the
# status file too. Removes the destination directory again on total
# failure so a failed preserve doesn't litter .preserved/ with an empty
# directory a later reader could mistake for a successful rescue.
#
# Prints the destination directory to stdout on success. Returns:
#   0 - .epic-status.json preserved (stdout has the path); the README
#       notes if .epic-worklog.md wasn't found to preserve alongside it
#   1 - no .epic-status.json in <source_dir>; nothing to do, not an error
#   2 - .epic-status.json present but could not be copied; caller must
#       NOT proceed to delete the originals
#
# Does not resolve clone_root itself and does not post any "committed
# pointer" comment -- callers differ on where clone_root comes from
# (an absolute .epic-config.json path vs. the default cwd one) and on
# gh pr comment vs. gh issue comment, so those stay call-site-specific.
preserve_epic_state() {
    local source_dir="$1"
    local clone_root="$2"
    local reason="${3:-}"
    if [ ! -f "$source_dir/.epic-status.json" ]; then
        return 1
    fi
    local issue_n clone_name today dest status_ok=0 worklog_ok=0
    issue_n=$(jq -r '.issue // "unknown"' "$source_dir/.epic-status.json" 2>/dev/null)
    clone_name=$(basename "$source_dir")
    today=$(date -I)
    dest="$clone_root/.preserved/$issue_n-$today"
    mkdir -p "$dest"

    # Never silently overwrite an existing preserved copy. Same issue,
    # same clone, same day can legitimately happen twice -- e.g. this
    # function's own issue #2216 landed as PR #224 then, the same day,
    # PR #225 -- and Step 9.5 deletes the source worklog after every
    # land, so a second call's source is a NEW file, not a superset of
    # the first. A bare filename collision would cp right over the
    # earlier one with no warning: measured, rc=0 both times, the first
    # worklog unrecoverable after the second call. Pick the first unused
    # numeric suffix instead of clobbering.
    local status_name="i$issue_n-$clone_name.status.json"
    local worklog_name="i$issue_n-$clone_name.worklog.md"
    if [ -e "$dest/$status_name" ] || [ -e "$dest/$worklog_name" ]; then
        local n=2
        while [ -e "$dest/i$issue_n-$clone_name.$n.status.json" ] || [ -e "$dest/i$issue_n-$clone_name.$n.worklog.md" ]; do
            n=$((n + 1))
        done
        status_name="i$issue_n-$clone_name.$n.status.json"
        worklog_name="i$issue_n-$clone_name.$n.worklog.md"
    fi

    cp "$source_dir/.epic-status.json" "$dest/$status_name" && status_ok=1
    if [ -f "$source_dir/.epic-worklog.md" ]; then
        cp "$source_dir/.epic-worklog.md" "$dest/$worklog_name" && worklog_ok=1
    fi

    if [ "$status_ok" -eq 0 ]; then
        rmdir "$dest" 2>/dev/null
        return 2
    fi
    {
        printf 'Preserved from %s, %s.%s\n' "$source_dir" "$today" "${reason:+ $reason}"
        printf 'Directory named %s-%s: the usual .preserved/ convention in this repo is <issue>-<slug>, but a date is trivially derivable and collision-free at this exact step, at the cost of a visibly different naming scheme here.\n' \
            "$issue_n" "$today"
        [ "$worklog_ok" -eq 1 ] || printf 'No .epic-worklog.md was found in %s to preserve alongside it.\n' "$source_dir"
    } > "$dest/README.md"
    echo "$dest"
    return 0
}

# mirror_worklogs <clone_root>
# Copies every live slot's .epic-worklog.md to <clone_root>/.mirror/ as
# i<issue>-<clone>.worklog.md. Idempotent; run it on every poll cycle.
#
# WHY THIS EXISTS, AND WHY IT IS NOT preserve_epic_state's JOB.
# A worklog lives ONLY in a pooled clone until its PR lands. /bip-pr-land's
# Step 6a preserves it at land time -- but only for landings that GO THROUGH
# /bip-pr-land. A direct `gh pr merge` bypasses Step 6a, Step 9.5, and the
# `EPIC worklog preserved` PR comment in one move, and nothing notices.
# Measured 2026-09-14 (matsengrp/phyz): PR #2648 landed that way and left a
# 35,432-byte worklog live in a pooled clone, where the next spawn's prep
# would have deleted it. The decisive tell was a contrast -- the PR that used
# the skill had 1 preservation-pointer comment, the one that did not had 0.
#
# WHY MIRRORING RATHER THAN DETECT-MERGE-THEN-PRESERVE. The reactive design
# races the next spawn's prep, and when it loses, "files already gone" is
# indistinguishable from "this landing was never bypassed". A detector whose
# failure mode is its success case is not a detector. Mirroring has no such
# state: the copy exists before any merge, by any path, and never needs to
# know how the PR merged -- which is the fact that is unobservable from
# outside the slot.
#
# THIS IS NOT .preserved/. The mirror is live, overwritten and best-effort --
# a floor under data loss. .preserved/ is the final, authoritative record with
# provenance READMEs. Never cite a mirror entry as the archived version, and
# never delete a .preserved/ entry because a mirror exists. If the two
# disagree, .preserved/ wins; a mirror LARGER than a matching .preserved/
# entry means preservation ran EARLY, which is a /bip-pr-land timing question
# and not a mirror fault -- report it rather than reconciling it.
#
# NEVER SHRINKS. If a live worklog is smaller than its mirror, the old copy is
# kept as i<issue>-<clone>.worklog.SHRANK-<ts>.md and the event reported.
# Mirroring is MORE exposed to truncation than .preserved/ is, precisely
# because it overwrites every cycle rather than once. Both directions occurred
# on 2026-09-14: one slot live-smaller (2135 B vs 15950 preserved), another
# live-larger (4472 -> 5859 after its terminal ceremony). A rule keyed on
# either direction alone would have been wrong once that day, so this keys on
# SHRINKAGE specifically rather than on "changed".
#
# SHRANK RETURNS 0, NOT 1, AND HERE IS THE ACTUAL SET IT FIRES ON -- read this
# before flipping it. `dst` is scoped by ISSUE and CLONE, and the comparison
# only runs `if [ -f "$dst" ]`, so the ordinary reclaim path CANNOT trip it:
# after a land, Step 9.5 deletes both state files, the clone is reclaimed, and
# the next spawn carries a DIFFERENT `.issue`, hence a different `dst`, hence
# a fresh copy with the old entry untouched. What trips it is same issue,
# same clone, smaller file:
#   1. genuine truncation                                     <- want to know
#   2. a worker rewriting or compacting its own worklog mid-issue  <- routine
#   3. a re-spawn onto the SAME issue after a reset                <- routine
#   4. a worker resuming after landing and re-creating its state for the same
#      issue -- which `bip-conductor-spawn`'s own prompt INSTRUCTS ("IF YOU
#      RESUME WORK AFTER THIS POINT, RE-CREATE `.epic-status.json` FIRST").
#      Observed 2026-09-14: a slot's re-created worklog was 2135 B against a
#      15950 B predecessor.
# Cases 2-4 are legitimate and at least one of them is protocol-mandated, so a
# non-zero return would make the poll step report failure on a normal day, and
# a signal that fires on the normal case stops being read. The data is never
# lost either way -- the longer copy is kept alongside. That, not "a post-land
# re-creation is shorter" (which only reaches here via case 4, not via the
# reclaim path), is the reason for 0.
#
# SEAM, worth knowing before an incident rather than during one: if a clone's
# status file is deleted while its worklog lives, `iss` falls back to
# `unknown` and the mirror writes to a DIFFERENT path, so the shrink guard
# does not apply across that transition. The outcome is still safe -- the old
# copy is retained under the old name -- but the guard is silently inactive
# there.
#
# Prints nothing on success. Returns 0 if every readable worklog was mirrored,
# 1 if any could not be -- there is no count anywhere in it, so it cannot
# report a reassuring zero for an unreadable clone.
mirror_worklogs() {
    local clone_root="$1"
    local m="$clone_root/.mirror"
    mkdir -p "$m" || { echo "MIRROR FATAL: cannot create $m"; return 1; }
    local rc=0 d c src dst iss s_new s_old ts
    for d in "$clone_root"/*/; do
        c=$(basename "${d%/}")
        case "$c" in .mirror|.preserved|.spawn-prompts) continue ;; esac
        src="$d.epic-worklog.md"
        [ -e "$src" ] || continue
        if [ ! -r "$src" ]; then
            echo "MIRROR UNREADABLE $c: $src exists but cannot be read"
            rc=1; continue
        fi
        iss=unknown
        if [ -r "$d.epic-status.json" ]; then
            iss=$(jq -r '.issue // "unknown"' "$d.epic-status.json" 2>/dev/null) || iss=unknown
            [ -n "$iss" ] || iss=unknown
        fi
        dst="$m/i$iss-$c.worklog.md"
        if [ -f "$dst" ]; then
            s_new=$(wc -c < "$src"); s_old=$(wc -c < "$dst")
            if [ "$s_new" -lt "$s_old" ]; then
                ts=$(date -u +%Y%m%dT%H%M%SZ)
                cp "$dst" "$m/i$iss-$c.worklog.SHRANK-$ts.md" || rc=1
                echo "MIRROR SHRANK $c issue=$iss live=${s_new}B was=${s_old}B -- kept old copy as i$iss-$c.worklog.SHRANK-$ts.md"
            fi
        fi
        cp "$src" "$dst" || { echo "MIRROR FAILED $c: cp $src -> $dst"; rc=1; }
    done
    return $rc
}

# mark_spawn_intent_consumed <intent-file-path>
# Moves a spawn-intent file into a "consumed/" subdirectory sibling to
# it, so a launched intent is distinguishable on disk from one still
# queued, without reading file contents. A plain `mv`, not a delete —
# `.spawn-prompts/` lives outside every clone's git, so deletion there
# is unrecoverable, and `/bip-conductor-tuckin` needs to report queued
# vs consumed separately (issue #198). A repeat consumption of the same
# issue number (e.g. a re-spawn after the epic writes a fresh intent
# file under the same name) overwrites the previous consumed file —
# consumed/ is a queued-vs-consumed marker, not a full history, so only
# the most recent consumption needs to be preserved.
mark_spawn_intent_consumed() {
    local intent_path="$1"
    local consumed_dir
    consumed_dir="$(dirname "$intent_path")/consumed"
    mkdir -p "$consumed_dir"
    mv "$intent_path" "$consumed_dir/"
}
