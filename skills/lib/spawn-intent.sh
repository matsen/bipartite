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
