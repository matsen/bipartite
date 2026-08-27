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
