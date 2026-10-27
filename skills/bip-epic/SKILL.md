---
name: bip-epic
description: EPIC topic strategy — GitHub issue/PR triage, EPIC body composition, dependency-direction and collision detection between issues
---

# /bip-epic

Topic strategy for one EPIC: program state, issue/PR triage, EPIC body composition, and the semantic calls a topic-agnostic fleet scanner cannot make (a deletion in one issue that a sibling's patch depends on).
Fleet mechanics (clone/tmux inventory, staleness, spawning, pruning) belong to `/bip-conductor`; ask it for fleet state rather than scanning clones or tmux.
The two roles coordinate over `SendMessage` and a shared `.spawn-prompts/` directory (Step 6).
Step 2 uses the conductor's occupancy table to scope Group A; Step 6 uses it as a readiness filter.

## One EPIC at a time

`/bip-epic 369` means EPIC #369 and nothing else. If the invocation names no EPIC, ask which one before doing anything; do not infer it from the repo, recency, or the fleet.

- A file collision is not EPIC membership: report the collision to the conductor and leave the out-of-scope issue.
- A real defect in another programme gets filed and parked, not pursued.
- Work in flight found to be outside the boundary: say so and hand it back; preserve WIP to a branch.

## Role

- Decide an issue is ready and draft its spawn brief; the conductor spawns it.
- Own the user-facing topic channel: scientific reasoning and decisions happen here. A semantic question raised in the conductor's window arrives here.
- Never write code, create branches, or spawn tmux windows for numbered issues.

## Conventions

### Issue/PR naming
- `i281` = issue #281, `p275` = PR #275. Never bare `#N`.
- First mention in a bullet list: full URL inline.
- When a live slot holds the work, add its tmux window: `p1344 (1343-peach)`.


### Message economy

`/bip-conductor`'s "Message economy" applies here too. Send raw measurements, not conclusions. A drafted worker correction (Step 7) states the change itself, never "re-read the EPIC body."

### The epic's own claims

- Before writing that the EPIC's question is answered, quote the question and state which measured quantity closes it, and by how much (say so if it is one stage at one fixture).
- Run a skeptic subagent before claiming a question answered and before filing an issue you wrote. File every new issue through `/bip-issue-check`, never raw `gh issue create`.
- Check attribution ("who found it") against the log or PR before repeating it.
- Escalate a question to the user only if its answers lead to different actions; take any shared first step now and ask only about the fork. How to test (seeds, sample size, decision rule) is yours to rule on, or staff's. Put the question as one line in your report, escalate per `/bip-conductor`'s "Paging the user", park only that item, and keep working.

## Configuration

Reads `.epic-config.json` from the repo root (shared with `/bip-conductor`). Fields used here:
- **github_repo**: `org/repo` for `gh`
- **clone_root**: locates `$CLONE_ROOT/.spawn-prompts/`

If the file is missing, do not create it; run `/bip-conductor` first, which owns the layout questions.
The `layout:` block in `~/.config/bip/config.yml` is for non-EPIC `bip spawn`, not for this skill. If no EPIC issue exists, report it and offer to create one.

## Workflow

### Step 1: Load config

```bash
git pull --ff-only origin main || echo "PULL FAILED — you are on a stale tree; fix this before reading anything"
cat .epic-config.json
```

Never suppress the pull's stderr. In a pooled clone it often fails because an untracked leftover collides with a newly tracked path: move the blocker aside (it may be unpreserved work), pull again, and confirm `git rev-list --count HEAD..origin/main` is `0`.
Re-run the pull at the top of every scan cycle, not only at cold start: `gh` reads stay correct on a stale tree, so nothing else forces it, and a stale file written back reverts an EPIC body or skill. `git fetch` alone does not update the working tree.

Read every tracked subdirectory `CLAUDE.md`/`AGENTS.md` (only the root one auto-loads):

```sh
git ls-files | /usr/bin/grep -E '(CLAUDE|AGENTS)\.md$'
```

Use `git ls-files`, not `find`: vendored copies and stale nested clones also hold these files. Before claiming a measured number is new, grep the experiment's own README for it.

**Own the EPIC** (`docs/guides/roles.md` owns the rules):
- At every cold start, set the body's `Owner:` line to this session's exact `ListAgents` name (the "This session is ..." row). If it names another live session, stop and ask staff.
- Make `_ignore/CONTINUE-epic-<N>.md` carry `Session: <that name>`, creating it if needed, so `/bip-tuckin` can identify this session before its first tuckin.
- A claim that your own hypothesis holds enters the body only after someone else re-derives it; the entry names the checker.
- Consult the papers in the `Feeds:` line before changing direction; each has a summary at `nexus/papers/ms-<owner>-<repo>.md`. Send each landed or reported result to the ms session named there as it arrives, marked checked, reported or provisional.
- When a result names a consumer outside this EPIC (a repo, `<repo>#N`, or an owner), send it after merge to that EPIC's `Owner:`, saying what it licenses and what it doesn't; never infer an unnamed consumer. The consumer's owner replies with its action and records it on the consumer issue.

### Step 2: Fan out scanners

Handshake with the conductor before dispatching Group A:
- Resolve `CLONE_ROOT`, read `$CLONE_ROOT/.conductor-session` for its address, and `SendMessage` it for its slot→issue occupancy table (with each slot's phase) and its negative list (`/bip-conductor` Step 5).
- The table lets Group A skip re-confirming items on in-progress slots (its task 3); a `completed` slot's issue can close before cleanup, so those are still confirmed. It does not replace dependency reasoning.
- If `.conductor-session` is absent or the send fails, dispatch Group A unscoped; do not wait.

Dispatch both groups as `general-purpose` subagents in one message, per `SUBAGENT-SCAN.md` (bipartite repo root). Scope to your EPIC; confirm it resolves and do not enumerate others:

```bash
gh issue view <your-epic-N> --json number,title,updatedAt
```

List every `EPIC in:title` issue only when offering options to a user who has not named one.

**Group A: one subagent for your EPIC.**

> Read EPIC `i<N>` and report its current state.
> Tasks:
> 1. `gh issue view <N> --json title,body,updatedAt`.
> 2. Parse the Status dashboard and Key findings. A legacy clone-assignment table in the body is a `surprise` for cleanup, not current truth.
> 3. For each open dashboard item, run `gh issue view <child-N> --json state,stateReason` to confirm it is open — except items the conductor's table shows on a slot in an in-progress phase (`coding`, `exploring`, `testing`, `awaiting-results`, `quality-gate`). Still confirm `completed`/`needs-human` slots and anything the table doesn't cover.
>
> Return under 400 words:
> - `changes_since_baseline`: completed items, new findings, items newly opened
> - `active_items`: open work with brief status (not which clone runs it)
> - `action_candidates`: items the EPIC marks ready but unassigned
> - `surprises`: contradictions, legacy clone-assignment tables, `RECOMMEND DEEPER LOOK` flags

**Group B: one recent-activity subagent.**

> Report recent issue/PR activity for the epic. This is a recency feed, not backlog coverage.
> Tasks:
> 1. `gh issue list --search "sort:updated-desc" --limit 20 --json number,title,state,labels,body`.
> 2. `gh pr list --json number,title,headRefName,state`.
> 3. `gh pr list --search "is:pr is:merged sort:updated-desc" --limit 10 --json number,title,mergedAt`.
> 4. For each open issue, check its `depends_on` field and blocking context. An issue depending on an unmerged PR or unfinished experiment is not ready — omit silently.
> 5. Confirm with `gh issue view <N> --json state` any issue you plan to flag.
>
> Return under 400 words:
> - `changes_since_baseline`: PRs merged since last session
> - `active_items`: open PRs with state/CI status
> - `action_candidates`: open issues ready to spawn (unblocked, unassigned, dependencies satisfied), by priority, restricted to issues this EPIC tracks; an urgent out-of-EPIC item goes under `surprises`
> - `surprises`: closed/merged items the EPICs don't reflect, unclear blocker state, `RECOMMEND DEEPER LOOK` flags

Group B does not cover the backlog: `--limit 20` reaches only a couple of days back on an active repo. Coverage is Group A plus a periodic audit for issues no EPIC references (`matsengrp/phyz`#2093).

Never ask the user what a `gh` query answers, and never ask a sequencing or staging question. `/bip-conductor`'s "Arbitration" applies here: escalate only what it names, or whether work is scientifically worth doing.

### Step 3: Reconcile

- Write the reconciliation yourself; do not paste subagent prose.
- Confirm an issue is OPEN on GitHub before presenting it as ready or needing action.
- Flag anything merged/closed that the EPIC body doesn't reflect.
- If a group reports zero `surprises` and zero `changes_since_baseline`, send it a narrower follow-up before concluding nothing changed.

### Step 4a: Dependency-direction detection

Before writing any gate, dependency, or scope claim into a brief or EPIC body, read it in the issue's own Dependencies, Scope, Out-of-scope, and Success criteria sections (whole issue if under ~200 lines); a scanner's summary of a gate is not the gate. File an out-of-scope precondition as its own issue rather than amending scope.

Run this before proposing any issue as ready; once a branch exists the collision cost is sunk.
1. Read every open, unassigned or about-to-be-spawned issue body against every other such body: can both land as written, in either order? Or does one delete, rename, or restructure something the other depends on (deletes a function the sibling patches; a test asserts a value the sibling changes)? That conflict has no fix by ordering; one issue must change shape.
2. Record findings in the EPIC dashboard, and in the brief of any issue about to be handed off (Step 6).

**Completion criterion**: every pair of open, about-to-be-spawned issues has been read against each other for dependency direction, not just for cross-references.

### Step 4b: File-overlap collision detection

Separate from 4a; clearing a pair in 4a does not clear it here.
1. Extract file/module paths from each open, unassigned or about-to-be-spawned issue body (code blocks, "Files:" sections, prose). Resolve each against the actual repo before recording an overlap or its absence: a bare filename is unresolved, and per-experiment `scripts/` copies share names. Name files with their directory in briefs and messages.
2. Build an overlap matrix: two issues naming the same file or module are a candidate collision.
3. For each overlap, decide whether the edits coexist or one invalidates the other.
4. Record findings as in 4a.

**Scope:**
- 4b covers filed issues only. Live branches are usually unpushed, so report "collisions among the filed issues" and leave live-branch intersection to the conductor (`bip fleet collisions`).
- Once a branch exists, its diff supersedes the issue body: `git diff --name-only origin/main...<branch>`.
- An issue editing a shared definition (constant, params list, fixture path) has a footprint equal to its consumers. Run `git grep -ln <SYMBOL> -- <subtree>`, report code and prose hits separately with the commit, and tell the worker to re-run it at branch time. The grep misses files that import the module without naming the symbol.
- Meaning collisions have no detector. For an issue that regenerates a committed artifact, ask whether anything cites it as evidence, and whether its value is being current or being the record a landed result was computed from.
- When two branches edit one machine-checked document, a clean merge is not evidence; run its checker.

**Completion criterion**: every overlapping pair has an explicit verdict — "compatible" or "conflicts, because ___".

Report 4b findings in 4b's vocabulary: a collision governs sequencing ("cannot be in flight at once"); a dependency governs eligibility ("cannot start"). Never phrase a 4b finding as "queues behind" or "blocked on"; that reads as ineligibility.

### Step 5: Prune and update the EPIC body

Follow the **EPIC body update pattern** below whenever findings come in, items complete, or work starts.

- Prune on every write: cut fully complete sections in the same edit. When new text replaces old, delete the old rather than qualify it, and grep the body for the item's other mentions. When a deleted line was a RULED OUT or settled entry, tell the conductor so it greps live briefs (`/bip-conductor` "Citations").
- Before writing a correction, retraction, or unification of two findings, apply `EVIDENCE-DISCIPLINE.md` ("Before writing a correction or a unification").
- Identified work needs an issue number: Group A, spawn intent files (`<N>.md`), and the conductor's slot map all key on it. File it, or list it in Open work as an explicit `**UNFILED**` bullet.
- Type every collision claim you send as VERIFIED (against a landed diff) or FORECAST (about an unwritten edit). A FORECAST same-file overlap is weak; the conductor resolves it at spawn time, so don't hold an issue on it.
- Re-derive any load-bearing number from a conductor, a worker, or your own earlier round, and quote the command.

**Fleet state is derived, never authored.** Never write which clone holds which issue, free slots, or live windows into an EPIC body or continuation doc; recompute it or ask the conductor (`/bip-conductor` Step 5). Any present-tense claim about what a slot, host, PR, or target is doing now is fleet state, even when only reported. Fleet policy (who lands PRs, whether CI exists, standard spawn-prompt content) is stable: ask once and keep it. Handoff artifacts (spawn intent, unfiled drafts) are authored deliberately, to a durable path outside git.

**User decisions enter the body only against a `FINAL` shape** (`/bip-conductor` "Decision relays: PROVISIONAL and FINAL").
- From a `PROVISIONAL` relay, write only that a decision is pending. `FINAL` arrives in `.epic-decisions.md` in the conductor cwd, or from staff.
- Only `FINAL`, or a decision reached in this session, authorizes the write; restate the decision's substance. Quote marks hold only the user's words; the owner's gloss goes outside, labelled.
- Staff's "staff applying <ruling, link>" and "staff ruling" relays carry no `FINAL` mark; write each as labelled (the latter as "staff ruling (<date, link>)"), never as a new user decision.
- An unmarked relay is not a decision and not a cue to ask the user: ask the relaying session for the marked form.

### Step 6: Hand spawn intent to the conductor

Run the PI questions (`docs/guides/pi-questions.md`) against the EPIC's state first.
For each issue ready per Step 3, clear per Step 4a/4b, **and holding no live slot in the conductor's occupancy table**, write the brief:

```bash
source "$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"
CLONE_ROOT=$(resolve_clone_root .epic-config.json)
mkdir -p "$CLONE_ROOT/.spawn-prompts"
# Use the Write tool to create "$CLONE_ROOT/.spawn-prompts/<N>.md" with the brief
```

`<this-skill's-base-directory>` is the base directory given at invocation; `lib/spawn-intent.sh` is a sibling of every skill directory. `resolve_clone_root` expands the tilde-form `clone_root`.

- Every brief opens with an `EPIC: <owner/repo>#<N>` line, cross-repo included: `/bip-conductor-spawn` requires it and counts live slots by it.
- Intersect candidates with the occupancy table as a discrete step and write "checked against conductor table of HH:MM" in the handoff; re-ask if the table is over a cycle old. In clone mode nothing downstream refuses the same issue spawned into a second idle clone.
- Older intent files are named `spawn-<N>.txt`; `/bip-conductor-spawn` reads either. Use `<N>.md` and don't rename old files.

`/bip-conductor-spawn` reads the directory, checks it against live fleet state, appends fleet facts, and executes after user confirmation.

**What a brief contains:**
- Only what the issue does not say; point at its scope sections. Open the issue before every brief.
- Its own footprint and no parallel-safety verdict: "this issue touches A, B, C; ask the conductor whether that intersects live work."
- Collision constraints in 4b's vocabulary ("cannot run concurrently with `iN`, because both edit X"), never another issue's gate, hold, or queue status.
- Only what has stopped moving: if the motivation cites a result still in review, wait for the merge.
- The host class for real compute: "run the sweep on an orca, not on `pax`."
- Only attributions you checked. When briefing a subagent, separate claim from attribution; it will re-emit your framing as a finding.

Do not spawn or touch tmux/clones yourself. Check `ListAgents` for a conductor before announcing a handoff:
- **Conductor addressable**: tell the user which issues have pending intent. If urgent, `SendMessage` it.
- **None addressable**: say so and offer both options — start `/bip-conductor` in a separate tmux window, or, for one small job, run `/bip-conductor-spawn` here for that intent file. Say plainly that the second mixes the two roles for this one spawn, and let the user pick.

### Step 6b: The design question, asked of every arm including the null

The epic half of a landing gate: "could this experiment have said otherwise?" Ask it of every arm carrying a claim, including a null. An arm cannot speak when it is:
- **vacuous-positive**: the check could not have failed given how the arms were built. Redesign the check.
- **no-power**: the pathway under test never engaged. Measure engagement before interpreting a null.

### Step 6c: Pre-register on your side of the wall

Write your expectation down before the data exists, and do not send it to the arm producing the data. Send the scheme (categories, discriminators, decision rules, known-positive controls) and keep the prior. Hold adjacent confounded evidence against a release condition written before the data, and state the condition.

### Step 7: Correcting a live worker — the judgment half

When a live worker's scope needs correcting before its next stopping point, the epic decides: durable (changes what it produces: scope, target, artifact, gate criterion) or transient (host load, peer timing, a dependency that landed)?
- **Durable**: draft a one-line correction and `SendMessage` the conductor to deliver it now; it `SendMessage`s the worker and appends a timestamped, attributed `.epic-worklog.md` entry in the same step (never edits `lead_guidance`). Route it through the conductor even though you could reach the worker, so message and worklog cannot diverge.
- **Transient**: message only, still through the conductor.

Editing an issue a live worker holds is a Step 7 correction: the worker read the issue once at spawn, so the edit is the record and the relayed delta is the delivery. Ask the conductor whether a slot is live on an issue before editing it. If a worker reports something you already fixed, the relay failed: tell it, and check what else failed to reach it.

### Step 8: Approving a merge, holding, or discharging a gate — post it to the PR

When you approve a worker's merge, place a hold, or discharge a gate you set, post a `🤖`-prefixed PR comment as well as messaging the worker. Not `gh pr review --approve`: you review claims, not code.
1. Post at decision time; a comment after the merge cannot show the approval came first. Label a retroactive one as a record.
2. State what was approved or held and why: the commit, what changed, what you checked. For a hold: `🤖 Hold (claims) at <sha>: <condition>`, plus what you have already ruled on.
3. State that it is not a code review.

Put rulings a future reader needs on the PR, not in slot files, which `/bip-pr-land` deletes.

## EPIC body update pattern

The EPIC body is the source of truth for project status; prune completed sections in the same edit (Step 5).
Keep a local copy as `ISSUE-EPIC-<N>.md` in the repo root (gitignored via `ISSUE-*.md`).

```bash
# Pull current body and record the timestamp
gh issue view <number> --json body,updatedAt > /tmp/epic-pull.json
jq -r .body /tmp/epic-pull.json > ISSUE-EPIC-<N>.md
PULLED_AT=$(jq -r .updatedAt /tmp/epic-pull.json)
rm -f /tmp/epic-pull.json

# Edit the file (add findings, check boxes, cut finished sections)
# ...

# Ownership can move between pull and push; the conflict check below catches a body changed since the pull.
LIVE=$(gh issue view <number> --json body -q .body) || { echo "ABORT: could not read the live body"; exit 1; }
OWNER=$(printf '%s\n' "$LIVE" | grep -m1 '^Owner: ')
[ -z "$OWNER" ] || [ "$OWNER" = "Owner: <this session's name>" ] || { echo "ABORT: $OWNER is not this session"; exit 1; }
grep -q '^Owner: ' ISSUE-EPIC-<N>.md || { echo "ABORT: the draft lost its Owner: line"; exit 1; }

# Before pushing: check if someone else edited since our pull.
# Both values MUST be non-empty: an unreadable timestamp compares "" to "" and passes.
CURRENT_AT=$(gh issue view <number> --json updatedAt -q .updatedAt)
if [ -z "$PULLED_AT" ] || [ -z "$CURRENT_AT" ]; then
  echo "ABORT: could not read updatedAt (network? auth?) — cannot establish safety, not pushing"
elif [ "$PULLED_AT" != "$CURRENT_AT" ]; then
  echo "CONFLICT: Issue was updated since pull ($PULLED_AT → $CURRENT_AT)"
  echo "Re-pull, merge changes, then try again."
  # Stop here — do NOT push
else
  gh issue edit <number> --body-file ISSUE-EPIC-<N>.md || echo "PUSH FAILED — the edit did not land"
fi
```

For any check that gates something destructive or outward-facing:
- A diagnostic that does not set exit status is not a guard: `print("MISS")` must be `sys.exit(1)`.
- Verify the mutation, not the command: grep for the new text; after a commit, `git diff <base> HEAD -- <file>` shows exactly what you intended.
- A guard that behaves the same when its query target is unavailable as when all is fine is not a guard.

**Size limits.** GitHub rejects a body over 2^18 = 262,144 bytes (`GraphQL: Body is too long`; its error text misstates the limit). That cap is for the body; comments of 131,076 characters have been accepted and 264,058 rejected, so size an archive comment against those. Past 64 KB a body stops being re-read whole, so give it an archive pass between findings, not mid-edit.
- Move completed items, superseded findings and Decisions history (`FINAL` quotes verbatim) to the archive comment, leaving a pointer where each was, since peers cite body text. Keep live gates, open items and current Decisions.
- Never move `Owner:`, or a live landing exception or NEGATIVE that an in-flight brief cites.
- Past ~250 KB, every addition needs a cut in the same edit; cut expired caveats and prose figures that have a live command first.
- Measure a stored body with `gh api repos/<org>/<repo>/issues/<n>` and `len(body.encode())`; `gh ... -q .body | wc -c` counts one extra newline.

**Replacing a body while archiving the old one to a comment.** Posting the archive moves `updatedAt`, so run the conflict check before your own write, and let the verified archive authorise the replacement:

```bash
T0=$(gh issue view <N> --json updatedAt -q .updatedAt)          # at pull
# ... edit locally ...
T1=$(gh issue view <N> --json updatedAt -q .updatedAt)          # BEFORE your own write
[ -n "$T0" ] && [ -n "$T1" ] || { echo "ABORT: could not read updatedAt"; exit 1; }
[ "$T0" = "$T1" ] || { echo "CONFLICT: a real concurrent editor"; exit 1; }

CID=$(gh issue comment <N> --body-file archive.md | /usr/bin/grep -oE '[0-9]+$')
[ -n "$CID" ] || { echo "ABORT: the archive post failed"; exit 1; }

# Characters, matching gh's .body|length. Null-check: an empty value is 0 in arithmetic.
EXPECTED=$(jq -Rs 'length' < archive.md)
[ -n "$EXPECTED" ] && [ "$EXPECTED" -gt 0 ] || { echo "ABORT: could not measure archive.md"; exit 1; }

LEN=$(gh api repos/<org>/<repo>/issues/comments/"$CID" -q '.body|length')
[ -n "$LEN" ] || { echo "ABORT: could not read the archive back"; exit 1; }
[ "$LEN" -ge $((EXPECTED * 95 / 100)) ] || { echo "ABORT: archive short — $LEN of ~$EXPECTED"; exit 1; }

# Truncation removes the tail, so check a phrase from the archive's LAST section.
gh api repos/<org>/<repo>/issues/comments/"$CID" -q .body | tail -40 \
  | /usr/bin/grep -q '<a phrase from the archive'"'"'s final section>' \
  || { echo "ABORT: archive tail missing — truncated"; exit 1; }

gh issue edit <N> --body-file new-body.md   # authorised by the verified copy; new-body.md keeps the Owner:/Feeds: lines
```

Do not add a third timestamp read after your write: `updatedAt` can lag a few seconds.

Sections to maintain: **Status dashboard** (check boxes, add items, cut finished ones), **Key findings** (numbered, append), **Related experiments** (add rows). No clone/slot assignment table (Step 5); a dashboard pointer names the issue, not the clone.
