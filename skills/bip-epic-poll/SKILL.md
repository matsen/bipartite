---
name: bip-epic-poll
description: Quick poll of GitHub activity and clone status since last check
---

# /bip-epic-poll

Lightweight mid-session update. Checks what changed on GitHub and in
active clones since last check. Use this instead of `/bip-epic` when
you already have context established.

For continuous monitoring, prefer `bip epic watch` (started by
`/bip-epic` Step 7). It writes phase transitions to
`.epic-notifications.log` in the conductor cwd; this poll skill reads
new entries from that log to catch transitions the conductor may have
missed. Use `/bip-epic-poll` for:
- Full GitHub reconciliation (merged PRs, new issues, comments)
- Slot cleanup and EPIC body updates
- Catching up on log entries written while the conductor was idle

For periodic auto-polling: `/loop 10m /bip-epic-poll`

## What to check

Two things stay in the primary because they're cheap and structured:

**Notifications log tail** — if `bip epic watch` is running, it
appends one JSONL line per phase transition to
`.epic-notifications.log` in the conductor cwd. The state file
`/tmp/.epic-poll-last-read` records the previous poll time as Unix
seconds; pass the elapsed window to `--since`:

```bash
LAST=$(cat /tmp/.epic-poll-last-read 2>/dev/null || echo $(($(date +%s) - 3600)))
NOW=$(date +%s)
bip epic watch --since "$((NOW - LAST))s" 2>/dev/null
echo "$NOW" > /tmp/.epic-poll-last-read
```

Surface any `needs-human` / `completed` transitions prominently per
the section below.

**Tmux window list** — `tmux list-windows -F "#W"` is one line of
output and tells the subagent which slots to inspect.

Everything else is delegated. Dispatch one `general-purpose` subagent
for the combined poll (the poll is lighter than the cold start, so a
single combined scan keeps round-trips down). Follow the dispatch
pattern in `SUBAGENT-SCAN.md` (bipartite repo root). Brief:

> Delta poll for the EPIC conductor since the previous poll. Tmux
> windows currently open: `<list from tmux list-windows>`. Tasks:
>
> 1. `gh pr list --search "is:pr is:merged sort:updated-desc"
>    --limit 5 --json number,title,mergedAt,body`. For each new
>    merge: note key results and whether it closes an issue.
> 2. `gh pr list --json number,title,headRefName,state`. Note new
>    PRs or CI status changes.
> 3. `gh issue list --search "sort:created-desc" --limit 5 --json
>    number,title,state,createdAt`.
> 4. For each active slot (per the tmux list and any
>    `.epic-status.json` in `<clone_root>`), check the latest
>    issue-lead comment: `gh api
>    repos/<owner>/<repo>/issues/<N>/comments --jq '.[-1].body'`.
>    Look for the `🤖 **Issue Lead**` prefix.
> 5. Inventory slots from `.epic-config.json`. Clone mode: iterate
>    `clone_names`. Worktree mode: `find <clone_root> -maxdepth 1
>    -name 'issue-*' -type d`. For each slot, read
>    `.epic-status.json` and surface: phase, summary, scope,
>    stop_reason, lead_guidance. Migrate legacy phases:
>    `blocked → needs-human`, `pr-review → quality-gate`.
> 6. For active slots, `git -C <slot> log --oneline main..HEAD |
>    head -5` to see recent commits.
> 7. For slots that look finished or blocked, capture the last 20
>    lines of tmux: `tmux capture-pane -t <window> -p | tail -20`.
> 8. Verify state with `gh pr view --json state` / `gh issue view
>    --json state` for anything you plan to flag — never claim
>    "open" or "merged" without a live confirmation.
>
> Return under 400 words, structured per `SUBAGENT-SCAN.md`:
> - `changes_since_baseline`: merged PRs, new issues, new
>   issue-lead comments, slot phase changes
> - `active_items`: per active slot — clone, issue, phase,
>   stop_reason, lead assessment (one line each)
> - `action_candidates`: open issues ready for unassigned slots;
>   merged PRs that should trigger slot cleanup; EPIC body updates
>   needed
> - `surprises`: `needs-human`/`completed` slots, stale status
>   files, contradictions, `RECOMMEND DEEPER LOOK` flags

If the report has zero `changes_since_baseline` and zero `surprises`,
the poll output is one line: "All quiet."

## After polling

### Focus on what matters

**Lead with unblocked issues** — issues that are ready to work on but
not assigned to any clone. This is the most actionable information.

**Surface lead evaluations** — if a clone's status shows a recent lead
evaluation (stop_reason set, lead_guidance present), mention the lead's
assessment briefly. This tells the conductor what the workers are doing
without having to read full issue comments.

**Flag needs-human and completed** — if any clone has `phase: "needs-human"`
(or legacy `blocked`) or `phase: "completed"`, highlight it prominently.
These require conductor attention. **Ring the terminal bell and send a
phone notification** so the user notices even if away:
```bash
printf '\a'
NTFY_TOPIC=$(grep ntfy_topic ~/.config/bip/config.yml | awk '{print $2}')
[ -n "$NTFY_TOPIC" ] && curl -s -H "Title: bip epic" -d "<clone> <phase> (<issue>)" "ntfy.sh/$NTFY_TOPIC" > /dev/null
```

**Only report active clones** — clones with a tmux window that are
actually doing something. Don't list completed or idle clones; that's
noise. Completed clones can be mentioned briefly ("fir completed i374")
but don't need a table row.

**Mention recent merges** only if they unblock something or change
the plan.

### Output structure

1. **Unblocked issues**: Issues ready for work, not assigned to a clone.
   Cross-reference with EPIC dashboards to find next items.

2. **Active work**: Clones with tmux windows that are mid-task. One line
   each: clone, issue, phase, stop_reason (if set), lead assessment.

3. **Needs human**: Clones in `needs-human` phase — show the lead's
   assessment and what decision is needed.

4. **Recently landed** (brief): PRs merged since last poll, only if
   noteworthy.

5. **Propose spawns**: If unblocked issues and idle clones exist, propose
   which to spawn. Wait for confirmation.

### Housekeeping (do silently, don't report unless problems)

This is the ongoing cleanup that keeps slots and EPICs current between
cold starts. Do it every poll cycle — don't wait for `/bip-epic`.

#### Slot cleanup for merged PRs

For each slot whose PR has merged (cross-reference merged PRs from
check 1 with slot branches):

**Worktree mode**:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
# Confirm PR is merged before removing
gh pr list --head <branch> --state merged --json number | jq length
# If merged:
git worktree remove "$CLONE_ROOT/issue-<N>"
git branch -d <branch>
```

**Clone mode**:
```bash
CLONE_ROOT=$(jq -r .clone_root .epic-config.json)
git -C "$CLONE_ROOT/<clone>" checkout main
git -C "$CLONE_ROOT/<clone>" pull --ff-only origin main
rm -f "$CLONE_ROOT/<clone>/.epic-status.json" "$CLONE_ROOT/<clone>/.epic-worklog.md"
```

Also clean up stale slots: no tmux window AND `.epic-status.json`
older than 30 minutes. Same cleanup as above.

#### EPIC body updates

If merges closed issues tracked in an EPIC, update the EPIC body:
follow the **EPIC body update pattern** from `/bip-epic` (pull →
edit → conflict-check → push). Check the box for completed items,
update the clone assignments table.

#### Memory

- Update MEMORY.md only for orchestrator-level decisions/patterns

## Conventions

Same as `/bip-epic`: `iN`/`pN` prefixes, full URLs on first mention.
Tmux windows named `NNN-YYY` where NNN is the issue number and YYY is the
clone/slot name (e.g. `281-cedar` in clone mode, `281-issue-281` in worktree mode).
