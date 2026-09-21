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

# clone_root_has_clones <clone_root>
# True when <clone_root> contains at least one subdirectory.
#
# WHY THIS EXISTS: `for d in "$ROOT"/*/` is the idiom every sweep here uses,
# and it behaves DIFFERENTLY AND BADLY IN BOTH SHELLS when the root is empty.
# Measured 2026-09-14 on an empty directory:
#   zsh   -> `no matches found: <root>/*/`, the statement ERRORS and the loop
#            body never runs
#   bash  -> passes the literal unexpanded `<root>/*/` in as `$d`, ONE SILENT
#            ITERATION on a path that does not exist
# Reachable exactly when someone runs the conductor for the first time: the
# clone root exists (every caller guards for that) and has no clones in it yet.
# Today every such loop happens to degrade safely in bash -- `[ -e "$src" ]`
# or a failing `rev-parse` sends it to `continue` -- but that safety is
# ACCIDENTAL, it is absent in zsh, and neither behaviour was written down.
#
# `find`, not a glob, for the same reason as `mirror_worklogs`'s newest-copy
# lookup: this file is sourced into whatever shell the caller runs. Tested
# against `bfs 4.1.1` (this fleet's `find`), not GNU findutils.
clone_root_has_clones() {
    # `! -name '.*'` is load-bearing and was added after this guard FAILED its
    # own test: `*/` does not match dot-directories, but `find -type d` does --
    # and `mirror_worklogs` calls `mkdir -p "$clone_root/.mirror"` immediately
    # BEFORE this check, so on an empty pool the guard saw the directory the
    # function had just created, passed, and let the glob error anyway. The
    # guard and the glob it protects must enumerate the SAME universe.
    [ -n "$(find "$1" -maxdepth 1 -mindepth 1 -type d ! -name '.*' -print -quit 2>/dev/null)" ]
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
# Copies every live slot's .epic-worklog.md AND .epic-status.json to
# <clone_root>/.mirror/. Idempotent; run it on every poll cycle.
#
# WHY .epic-status.json IS MIRRORED TOO, AND IT IS NOT SYMMETRY.
# Measured 2026-09-14: an issue-lead's terminal assessment carried two real
# findings -- a wrong number in a landed doc, and a straddle result that
# discharged an open hedge for 3 of 5 cells at zero compute -- and BOTH lived
# in the lead's assessment, not in the worklog. A worklog-only mirror would
# have lost them; they survived because /bip-pr-land happened to preserve the
# assessment alongside. The worklog carries narrative; lead output carries
# verdicts, and they are different files.
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
#      *** AND THIS GUARD HAS NOW FIRED ON IT IN PRODUCTION. *** Same day,
#      within an hour of this comment being written and while the case was
#      still hypothetical here: a slot landed its PR, was asked to resume for
#      post-land work, re-created its worklog, and the mirror caught
#      32922 B -> 3948 B, keeping the long copy. 29 KB that a plain `cp` loop
#      would have destroyed. The guard was argued into existence from reasoning
#      with no instance in hand; it has TWO instances now, both 2026-09-14.
#      Second: cedar/#2649 post-land, where `.preserved/` held a 32,922 B
#      worklog from 13:14 and the post-land re-creation was 13,468 B at 14:57
#      -- both kept, so a naive overwrite would have lost 19 KB. Do not delete
#      it on the grounds that the case looks theoretical; it fires on the
#      ordinary landing path, not on an edge case.
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
    clone_root_has_clones "$clone_root" || return 0
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
        # SHRANK applies PER FILE, not per slot: a lead assessment is rewritten
        # on every lead invocation and so shrinks routinely, and letting that
        # drag a slot's whole return code would make the signal unreadable.
        if [ -f "$dst" ]; then
            s_new=$(wc -c < "$src"); s_old=$(wc -c < "$dst")
            if [ "$s_new" -lt "$s_old" ]; then
                ts=$(date -u +%Y%m%dT%H%M%SZ)
                cp "$dst" "$m/i$iss-$c.worklog.SHRANK-$ts.md" || rc=1
                echo "MIRROR SHRANK $c issue=$iss live=${s_new}B was=${s_old}B -- kept old copy as i$iss-$c.worklog.SHRANK-$ts.md"
            fi
        fi
        cp "$src" "$dst" || { echo "MIRROR FAILED $c: cp $src -> $dst"; rc=1; }

        # .epic-status.json: KEEP EVERY VERSION, timestamped, never overwrite.
        # Size-keyed shrink protection is the WRONG instrument here -- a status
        # file can shrink while GAINING the field you care about (a lead can
        # replace a long stop_reason with a short one while adding
        # completed_at, or rewrite lead_notes wholesale). Content matters and
        # size does not track it. The files are ~1-6 KB, so keeping every
        # distinct version costs nothing and removes the need to guess which
        # one mattered. Only writes when the content actually differs from the
        # newest kept copy, so an unchanged slot does not accumulate entries
        # every poll cycle.
        sstat="$d.epic-status.json"
        if [ -e "$sstat" ]; then
            if [ ! -r "$sstat" ]; then
                echo "MIRROR UNREADABLE $c: $sstat exists but cannot be read"
                rc=1
            else
                # `find`, not a glob: this file is sourced into whatever
                # shell the caller runs, and under zsh an unmatched glob in
                # command position is a SHELL error that `2>/dev/null` on the
                # command does not suppress -- zsh fails the expansion before
                # `ls` is ever invoked. Measured: every first run of a fresh
                # slot printed `no matches found: .../i<N>-<clone>.status.*.json`
                # while otherwise working correctly. `find -newer`-free and
                # glob-free, so it behaves identically in bash and zsh.
                # `-printf` is a GNU extension and `find` on this fleet is
                # `bfs 4.1.1`, not GNU findutils -- tested there: `-printf`
                # works and returns exit 0 with no output on no match. A
                # reader will assume GNU; it is not.
                local newest
                newest=$(find "$m" -maxdepth 1 -name "i$iss-$c.status.*.json" \
                         -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -1 | cut -d' ' -f2-)
                if [ -z "$newest" ] || ! cmp -s "$sstat" "$newest"; then
                    cp "$sstat" "$m/i$iss-$c.status.$(date -u +%Y%m%dT%H%M%SZ).json" \
                        || { echo "MIRROR FAILED $c: cp $sstat"; rc=1; }
                fi
            fi
        fi
    done
    return $rc
}

# audit_status_files <clone_root>
# Reports .epic-status.json fields that ASSERT SOMETHING FALSE. Silent on a
# clean pool; one line per problem; returns non-zero if any fired.
#
# All three checks exist because the same failure keeps recurring in different
# fields: the status file reads as monitored while monitoring nothing, and the
# discrepancy is invisible unless something compares the file against reality.
#
# ⭐ WHY THESE ARE CHECKS AND NOT BETTER DOCUMENTATION. Both fields audited
# here ALREADY HAVE an explicit stated rule in the worker-facing spawn prompt,
# and both rules were broken anyway on 2026-09-14:
#   - `bip-conductor-spawn/SKILL.md` lists the seven phases in the
#     second-person `.epic-status.json fields:` block, and THREE distinct
#     off-spec values still appeared in one day.
#   - Two lines below it, `updated_at` says "Never a placeholder, and never
#     local time with a `Z` appended" -- and a future-dated value appeared.
# ⚠ Incidentally that same line rules out the obvious diagnosis: pax is UTC-7,
# so local-time-with-Z reads 7 hours BEHIND, not 62 minutes ahead. The
# documented failure mode has the wrong sign for what was observed.
#
# 1. FUTURE-DATED TIMESTAMPS. Measured 2026-09-14: a slot wrote
#    `updated_at: 2026-09-14T20:35:00Z` against a real clock of 19:33:32Z --
#    62 minutes ahead, and round to the whole minute, which two `date -u` calls
#    cannot both be. The epic ruled out clock skew (`System clock
#    synchronized: yes`) and timezone (a TZ error here is 7 hours, not 62
#    minutes); the fit is a worker writing an ETA into a field that means LAST
#    TOUCHED.
#    ⚠ Direction matters: a PAST-dated `started_at` invents a timeout, which is
#    loud and gets investigated. A FUTURE-dated one HIDES A STALL -- the run
#    keeps reading as having budget, so a hung job is never escalated. The
#    quiet failure is the one to catch.
#
# 2. `awaiting-results` WITH NO `awaiting` BLOCK. Two of two slots that reached
#    that phase on 2026-09-14 got it wrong, in different ways -- one had no
#    block at all while a 16-search remote run was in flight, the other was at
#    a stopping point and not waiting on anything. Two of two is a spec
#    ambiguity, not two slips: the phase NAME reads as "I am waiting" while its
#    CONTRACT is "I have a live readiness probe".
#
# 3. A `phase` OUTSIDE THE DOCUMENTED SET. A lead wrote
#    `phase: "premature-deferral"` -- a `stop_reason` value in the `phase`
#    field.
#    ⛔ It surfaced ONLY because `bip epic watch`'s `--phases` filter had
#    previously been WIDENED to absorb that exact string. That was the wrong
#    remedy and this comment exists so nobody repeats it: widening a filter to
#    accommodate an off-spec value converts a schema violation into a silent
#    success. The filter then does exactly what it was told, against a value
#    set that no longer means anything. An off-spec phase must SHOUT.
#
# Uses python3 rather than jq: this parses timestamps and compares them to the
# clock, and it must distinguish "field absent" from "field unparseable" --
# both are findings, and a jq default would collapse them into each other.
audit_status_files() {
    local clone_root="$1" rc=0 out
    clone_root_has_clones "$clone_root" || return 0
    out=$(python3 - "$clone_root" <<'PY'
import json, os, sys, datetime, glob
root = sys.argv[1]
OK = {"exploring","coding","testing","awaiting-results",
      "quality-gate","needs-human","completed"}
now = datetime.datetime.now(datetime.timezone.utc)
bad = False
for p in sorted(glob.glob(os.path.join(root, "*", ".epic-status.json"))):
    clone = os.path.basename(os.path.dirname(p))
    try:
        d = json.load(open(p))
    except Exception as e:
        print(f"STATUS UNREADABLE {clone}: {e}"); bad = True; continue
    mtime = datetime.datetime.fromtimestamp(os.path.getmtime(p), datetime.timezone.utc)

    ph = d.get("phase")
    if ph not in OK:
        print(f"STATUS OFF-SPEC-PHASE {clone}: phase={ph!r} is not one of {sorted(OK)}")
        bad = True

    if ph == "awaiting-results" and not d.get("awaiting"):
        print(f"STATUS NO-AWAITING-BLOCK {clone}: phase=awaiting-results with no awaiting block "
              f"-- nothing to check, and the file reads as monitored")
        bad = True

    # ABSENT is handled DIFFERENTLY per field, and getting this wrong was the
    # audit's own first defect: both branches originally `continue`d on a
    # missing value, which silently accepted a status file with NO
    # `updated_at` at all -- the WORSE failure, since a future timestamp hides
    # a stall in one direction while a missing one leaves staleness
    # unassessable in any direction. `python3` was chosen over `jq` precisely
    # so absent and unparseable would not collapse into each other; routing
    # absent to `continue` collapsed them anyway.
    #   updated_at          -> required; absent must SHOUT
    #   awaiting.started_at -> required ONLY WHEN AN `awaiting` BLOCK EXISTS.
    #
    # ⚠ That last clause is a correction to this code's own first fix. The
    # reasoning offered for skipping an absent `started_at` was that
    # NO-AWAITING-BLOCK already reports it -- true when the WHOLE BLOCK is
    # missing, and false when the block exists WITHOUT that field. In that
    # second case NO-AWAITING-BLOCK does not fire (a block is present) and the
    # skip swallows it, so a slot with a `check_cmd` and no `started_at` passes
    # clean while `started_at + timeout_hours` -- the only thing that catches a
    # dead remote run -- cannot be evaluated at all. A justification that holds
    # for one case, applied to a broader condition: the same shape this whole
    # audit exists to catch, twice now, inside the audit.
    _aw = d.get("awaiting") or {}
    for field, val, required in (("updated_at", d.get("updated_at"), True),
                                 ("awaiting.started_at", _aw.get("started_at"),
                                  bool(_aw))):
        if not val:
            if required:
                print(f"STATUS MISSING-TIME {clone}: {field} is absent -- staleness "
                      f"cannot be assessed in either direction")
                bad = True
            continue
        try:
            t = datetime.datetime.fromisoformat(str(val).replace("Z", "+00:00"))
            if t.tzinfo is None:
                t = t.replace(tzinfo=datetime.timezone.utc)
        except Exception:
            print(f"STATUS UNPARSEABLE-TIME {clone}: {field}={val!r}"); bad = True; continue
        # CONSTRUCTED-RATHER-THAN-MEASURED, reported as a HEURISTIC not a
        # violation. Both bad values found on 2026-09-14 ended in `:00`
        # seconds. `date -u +%H:%M:%SZ` distributes seconds uniformly, so a
        # legitimate reading lands on `:00` about once in sixty; measured
        # against 15 correctly-written files in the live pool, ZERO did
        # (seconds seen: 05 06 08 10 13 15 16 19 24 34 42 44 58).
        #
        # ⭐ WHY THIS EXISTS SEPARATELY FROM FUTURE-TIME: the diagnosis moved.
        # It was first read as a worker writing an ETA into a last-touched
        # field -- then one of the bad values turned out to be written by an
        # issue-LEAD subagent, which has no schedule to project. The fit that
        # survives is that an agent CONSTRUCTED a plausible timestamp from its
        # own sense of the current time, rounded to the minute. That explains
        # the `:00` seconds, the round offset, and the forward direction at
        # once.
        # ⛔ And it catches a case FUTURE-TIME cannot see: a constructed
        # timestamp landing in the PAST. By the direction analysis above that
        # one is the loud failure rather than the fatal one -- it invents a
        # timeout instead of hiding a stall -- but it is equally fabricated.
        #
        # ⚠ Heuristic wording is deliberate. One in sixty legitimate values
        # will trip this, so it must read "verify" and must NOT carry the same
        # severity as a future timestamp.
        if str(val).endswith(":00Z") or str(val).endswith(":00+00:00"):
            print(f"STATUS SUSPICIOUS-TIME {clone}: {field}={val} ends in :00 seconds "
                  f"-- looks CONSTRUCTED rather than measured; verify it came from "
                  f"`date -u`. (~1 in 60 legitimate values trip this)")
            bad = True
        if t > now + datetime.timedelta(seconds=90):
            print(f"STATUS FUTURE-TIME {clone}: {field}={val} is "
                  f"{int((t-now).total_seconds()//60)} min AHEAD of the clock "
                  f"(file mtime {mtime.strftime('%Y-%m-%dT%H:%M:%SZ')}) "
                  f"-- a future timestamp HIDES a stall")
            bad = True
sys.exit(1 if bad else 0)
PY
    ) || rc=1
    [ -n "$out" ] && printf '%s\n' "$out"
    return $rc
}

# audit_durability <clone_root>
# Reports slots holding work that NOTHING WOULD RECOVER. Silent when every
# slot's work exists somewhere other than one pooled clone's working tree.
#
# WHY A SEPARATE SWEEP FROM `mirror_worklogs`. The mirror closed ONE loss
# channel -- the worklog -- and having closed it, it is tempting to treat the
# problem as solved. It is not: the mirror covers `.epic-worklog.md` and
# `.epic-status.json` and NOTHING ELSE. Source, results, and a detached HEAD
# have no mirror and should not get one (mirroring source duplicates git
# badly). They have a git-native observable instead, and this checks it.
#
# ⭐ MEASURED 2026-09-14, and the numbers are the argument for the sweep
# existing: a spot check of one slot led to sweeping all six, and FOUR were
# holding unrecoverable work in THREE DIFFERENT SHAPES. No single probe finds
# all three -- a sweep that checks only "is the tree clean" reports two of them
# safe:
#
#   1. UNCOMMITTED WITH ZERO COMMITS -- dirty tree, `ahead=0`. Three slots.
#      One held all four of its issue's pipeline-defect fixes, i.e. the entire
#      deliverable. `@{u}..HEAD` returns 0 for this; only `status --porcelain`
#      sees it.
#   2. DETACHED HEAD WITH AN EMPTY BRANCH -- and this one's `git status` reads
#      CLEAN, which is why it is the worst. A bisect had narrowed a 30-commit
#      window and every probe result existed only in conversation context, at
#      99.8% of the model's window. Nothing on disk, nothing in the branch,
#      nothing to recover from.
#   3. COMMITTED BUT UNPUSHED -- three commits across 13 files existing only in
#      one clone. Lower risk than (1) since objects survive the prep's
#      `git checkout main`, but a lost disk or a forced reset takes it.
#
# ⚠ A CLEAN `git status` HAS MEANT THREE DIFFERENT THINGS on this fleet in one
# day: "already preserved", "preservation never ran", and "nothing was ever
# saved". The discriminator is always something else -- `ahead`, an upstream,
# a PR pointer comment -- never the status output.
#
# Slot-ness is gated on `.epic-status.json` existing, deliberately: a clone
# root legitimately holds clones of OTHER repositories (a pinned comparator
# checkout, say), and those have no status file. A landed slot whose status
# file was deleted is also correctly skipped -- it is on `main` and clean, with
# nothing at risk.
#
# `rev-parse`, never `test -d .git`: in a WORKTREE `.git` is a FILE, and that
# exact false negative is recorded in `skills/bip-epic/SKILL.md`.
audit_durability() {
    local clone_root="$1" cfg="${2:-.epic-config.json}" rc=0 d c b ahead dirty up gd names
    clone_root_has_clones "$clone_root" || return 0
    # SLOT-NESS COMES FROM THE CONFIG'S CLONE LIST, NOT FROM A STATUS FILE.
    #
    # The obvious gate -- "has .epic-status.json" -- conflates two different
    # things and skips a real loss channel. `/bip-pr-land`'s Step 9.5 does
    # `rm -f .epic-status.json .epic-worklog.md` and DOES NOT clean the working
    # tree, so a slot that lands with uncommitted scratch (an un-added test, an
    # experiment output, a half-finished file) ends up with leftovers and NO
    # status file. It then reads as idle to every mechanism, and the next
    # spawn's prep deletes them -- precisely the shape this sweep exists to
    # catch, sitting behind the gate.
    #
    # But the gate is doing real work and must not simply be dropped: a clone
    # root legitimately holds clones of OTHER repositories (a pinned comparator
    # checkout), which have no status file either. `clone_names` distinguishes
    # "not a slot" from "a slot with no status file"; status-file presence
    # cannot.
    names=$(python3 -c '
import json,sys
try:
    d=json.load(open(sys.argv[1]))
except Exception:
    sys.exit(1)
print("\n".join((d.get("clone_names") or []) + (d.get("new_clone_names") or [])))
' "$cfg" 2>/dev/null) || {
        echo "DURABILITY UNCHECKABLE: cannot read clone_names from $cfg" \
             "-- refusing to guess which directories are slots"
        return 1
    }
    [ -n "$names" ] || {
        echo "DURABILITY UNCHECKABLE: $cfg lists no clone_names"
        return 1
    }
    for d in "$clone_root"/*/; do
        c=$(basename "${d%/}")
        printf '%s\n' "$names" | grep -qxF "$c" || continue
        gd=$(git -C "$d" rev-parse --absolute-git-dir 2>/dev/null) || continue
        [ -n "$gd" ] || continue
        b=$(git -C "$d" branch --show-current 2>/dev/null)
        # Read status BEFORE counting. Piping git straight into `grep -c`
        # cannot tell "clean tree" from "git failed": both give an empty pipe
        # and `grep -c .` reports 0, which this function reads as CLEAN. That
        # is fail-open on the one direction the whole check exists to catch.
        status=$(git -C "$d" status --porcelain 2>/dev/null) || {
            echo "DURABILITY UNCHECKABLE $c: cannot read working-tree status"
            rc=1; continue
        }
        dirty=$(printf '%s' "$status" | grep -c . || true)

        if [ -z "$b" ]; then
            echo "DURABILITY DETACHED $c: detached HEAD with a live status file" \
                 "-- anything committed here is unreachable from any branch"
            rc=1
            continue
        fi

        ahead=$(git -C "$d" rev-list --count origin/main..HEAD 2>/dev/null) || ahead=""
        if [ -z "$ahead" ]; then
            echo "DURABILITY UNCHECKABLE $c: cannot count commits against origin/main"
            rc=1; continue
        fi

        # DIRTY IS A FINDING REGARDLESS OF COMMIT COUNT. This originally
        # required `ahead -eq 0` as well, which made uncommitted files
        # INVISIBLE on a branch that has commits: dirty=2, ahead=3, up=0 fired
        # nothing at all -- not UNCOMMITTED (ahead != 0), not UNPUSHED (up =
        # 0), not NO-UPSTREAM. Silent, with two files at risk. Uncommitted work
        # lives only in the working tree whether or not the branch has commits;
        # the risk is identical and only the alarm differs. Narrowing a check
        # to its most alarming instance and missing the general one is the same
        # shape as the absent-vs-future `updated_at` defect in this same file.
        # SEVERITY IS GATED ON SLOT STATE, NOT ON COMMIT COUNT.
        #
        # Detection stayed unconditional (see the comment above) -- but `dirty
        # + commits` is a STATE, not a defect, and reporting a state through a
        # channel meant for defects is what kills a guard. Measured 2026-09-14:
        # with severity undifferentiated this fired on FOUR OF FIVE live slots
        # every cycle, all of them simply working. A reader learns to skip that
        # in a day.
        #
        # "Does the branch have commits" was never what determines risk. "Is
        # this slot about to lose them" is -- and the poll already reads
        # `phase` from the same file in the same cycle, so the split is free.
        #
        #   dirty + ZERO commits, any phase        -> LOUD  (exists nowhere else)
        #   dirty + NO status file                 -> LOUD  (post-land leftovers;
        #                                             the next spawn's prep deletes them)
        #   dirty + completed|needs-human|quality-gate -> LOUD  (finished, stopped,
        #                                             or about to land)
        #   dirty + exploring|coding|testing|awaiting-results -> note only
        #
        # ⚠ `quality-gate` in the loud set is the non-obvious member and is
        # there deliberately: a slot about to land with uncommitted files is
        # the case where those files SILENTLY DO NOT MAKE THE PR. It reads
        # exactly like the routine case unless phase distinguishes it.
        #
        # Only LOUD sets `rc`. The note line exists so a reader can see the
        # state, not so anyone acts on it.
        if [ "$dirty" -gt 0 ]; then
            local ph="" loud=0 why=""
            if [ -f "$d.epic-status.json" ]; then
                ph=$(python3 -c 'import json,sys
try: print(json.load(open(sys.argv[1])).get("phase") or "")
except Exception: print("")' "$d.epic-status.json" 2>/dev/null)
            else
                loud=1; why="no status file -- post-land leftovers, the next spawn's prep deletes these"
            fi
            if [ "$ahead" -eq 0 ]; then
                loud=1; why="ZERO commits on '$b' -- the work exists only in this pooled clone's working tree"
            fi
            case "$ph" in
                completed|needs-human|quality-gate)
                    loud=1
                    why="${why:-phase=$ph -- slot is finished, stopped, or about to land, so these files will not make the PR}" ;;
            esac
            if [ "$loud" = 1 ]; then
                echo "DURABILITY UNCOMMITTED $c: $dirty changed files -- $why"
                rc=1
            else
                echo "DURABILITY note $c: $dirty changed files on '$b' (phase=${ph:-?}, $ahead commit(s))" \
                     "-- informational, a working slot; not relayed"
            fi
        fi

        # `@{u}` fails loudly when there is no upstream -- which is itself the
        # finding, not an error to swallow.
        if up=$(git -C "$d" rev-list --count '@{u}..HEAD' 2>/dev/null); then
            if [ "$up" -gt 0 ]; then
                echo "DURABILITY UNPUSHED $c: $up commit(s) on '$b' not on its remote"
                rc=1
            fi
        elif [ "$ahead" -gt 0 ]; then
            echo "DURABILITY NO-UPSTREAM $c: $ahead commit(s) on '$b' and no remote branch at all"
            rc=1
        fi
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
