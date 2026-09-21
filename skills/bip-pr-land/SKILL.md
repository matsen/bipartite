---
name: bip-pr-land
description: Land a PR branch — squash merge, clean up local and remote branches, return to main.
---

# /bip-pr-land

Squash-merge the current branch's PR and clean up.

## Usage

```
/bip-pr-land           # Land the current branch's PR
/bip-pr-land #42       # Land PR #42 (if not on that branch)
```

## Worktree mode

If `bip spawn` created the working tree as a linked git worktree (because the user opted into the global `layout: { mode: worktree }` block in `~/.config/bip/config.yml`), this skill detects that in Step 6a and performs the base-branch pull from the primary clone, then removes the worktree in Step 8 before deleting the branch.
In the common clone-mode case (no `layout:` block) every step runs as it always has — the worktree detection is a no-op.

## Where to stand

⛔ **This skill moves `HEAD` in the working directory. Never run it in a clone another live session is working in** — Step 7's `git checkout <base>` and Step 8's branch delete are exactly the operations that will move a peer's checkout out from under it. **A merge is a server-side operation; the local clone is only a place to stand, and what makes one the wrong place is that somebody else is standing there.**

**So, in order of preference:**

1. **The PR's own clone or worktree**, if a live session there is landing its own work. The normal case.
2. ⭐ **Any idle pool clone**, when the lander is not the author — a conductor landing a peer's PR, say. Fetch the branch there and land from there. **It has no relationship to the PR and that is fine**; the only property that matters is that nobody else is live in it.
3. ⛔ **Never the conductor's own clone if an epic or another session shares it**, and never a peer's.

⚠ **Check rather than assume, because co-tenancy is invisible from inside:**

```sh
for p in $(pgrep -x claude); do readlink /proc/$p/cwd; done | grep -cE "^$PWD(/|$)"
```

**More than one is co-tenancy.** ⛔ **Anchor the match — a bare `grep -c "$PWD"` is a substring test and sibling clones collide.** Measured: with `~/re/pz/ash` and `~/re/pz/ash-iqtree-a00094e0` both present, the substring form counts every session in the sibling as a co-tenant (3 against an anchored 2 on a synthetic triple; both forms return 3 on a genuinely shared clone, so the fix costs nothing on the case you care about). The `(/|$)` keeps a session cwd'd into a **subdirectory** of the clone, which is a real co-tenant, and drops the sibling.

⛔ **And say what it cannot see, because its two failure directions are asymmetric.** It reads each `claude` process's **own** cwd. **A session launched elsewhere that operates in this clone by `cd`-ing inside its Bash calls is invisible to it.** The substring looseness above fails toward a **false positive**, which costs a clone hop; this limit fails toward a **false negative**, which costs a peer's checkout. ➡ **The count is a floor, not a census — `1` is not proof of solitude.** Measured on `matsengrp/phyz` 2026-09-18: a conductor and an epic session had both been operating in one clone for a full day, each believing it was theirs; the conductor moved the epic's checkout to a temp branch and back while verifying a PR, and it was harmless **only because the epic's branch happened to sit at the PR head and it happened not to be mid-write.**

⭐ **Read-only inspection is always safe in a shared clone — `git show <sha>:<path>`, `git diff <a> <b>`, `git log`.** It is only the operations that move `HEAD` that need a clone of your own. **Reach for `git show` before reaching for a checkout.**

⛔ **AND IF YOU FETCH A PR BRANCH TO INSPECT IT: `git fetch origin pull/N/head:<branch>` SILENTLY LEAVES AN EXISTING LOCAL BRANCH AT ITS OLD SHA.** No output, exit 0, stale ref. **Two sessions hit this independently on `matsengrp/phyz` on 2026-09-18, hours apart and neither from the other's mistake** — one reviewed the wrong head for several minutes believing it was current and would have approved an artifact lacking the assertion it was describing as verified; the other found a local `pr<N>` branch two heads behind the live PR. ➡ **Use `git fetch -f`, or compare `git rev-parse <branch>` against `gh pr view <N> --json headRefOid` before reading a line of it.** ⭐ **Better still, skip the branch entirely: `gh pr view <N> --json headRefOid` gives you the SHA, and `git show <sha>:<path>` reads it with nothing to go stale.**

## Workflow

### Step 1: Check for uncommitted work

⚠ **If you skipped straight here, read “Where to stand” above first** — it is a precondition to every step below, and a reader who meets it after Step 1 has already moved `HEAD` somewhere they may not own.

```bash
git status --porcelain
git diff --stat
```

If there are uncommitted changes or untracked files, you MUST resolve each one explicitly — **never stash and move on**:

1. **Identify every dirty file.**
   For each one, read enough of the diff or file content to understand what it is and why it exists.
2. **Categorize each file:**
   - **Belongs to this PR** (e.g. forgotten formatting fix, test update): stage and commit with a short message.
   - **Unclear**: show the user the file and diff, explain what you see, and ask whether to commit it with the PR or move it aside.
   - **Unrelated / stray**: move it to `_ignore/$(date -I)-landing/` so main stays clean.
     Create the directory if needed.
     Tell the user what you moved and why.
3. **Never use `git stash`.**
   Stashing hides work and risks losing it.
   Every file must be either committed or moved to `_ignore/`.
4. **Ask the user if unsure.**
   If you can't confidently categorize a file, ask.
   A quick question is always better than guessing wrong.

### Step 2: Identify the PR

```bash
# Get current branch
BRANCH=$(git branch --show-current)

# Find the PR for this branch
gh pr view "$BRANCH" --json number,title,state,baseRefName
```

If no PR found, abort: "No PR found for branch `$BRANCH`."
If PR is not open, abort: "PR is already `$STATE`."

Save the base branch name (usually `main` or `master`) from `baseRefName`.

### Step 3: Log and proceed

Print the PR summary line, then continue without waiting for confirmation:

```
Landing: #42 "Add feature X" (branch: my-feature → main)
```

### Step 4: Update base branch and rebase

```bash
git fetch origin
git rebase origin/<base>
```

If rebase has conflicts, stop and report.
Do not force-push or auto-resolve.

### Step 5: Force-push rebased branch

```bash
git push --force-with-lease
```

### Step 5.5: Wait for CI to pass

Check whether the PR has any CI checks configured, and if so, block until they all pass:

```bash
gh pr checks "$BRANCH" --json name,state,conclusion
```

- **No checks configured** (empty array): proceed immediately.
  This repo has no CI for this PR.
- **Checks present**: wait until all required checks are `COMPLETED` with conclusion `SUCCESS` (or `NEUTRAL`/`SKIPPED`).
  Use `gh pr checks "$BRANCH" --watch --fail-fast` to block.
- **Any check fails**: abort with the failing check name and a link via `gh pr view --web`.
  Do **not** merge.
  Report to user and stop.

Never merge a PR with failing or pending required checks.
If checks are still queued/in progress, wait — do not assume they will pass.

### Step 6: Squash merge via gh

⛔ **ASSERT BASE CURRENCY AND HEAD IDENTITY *INSIDE* THE MERGE COMMAND, NOT AS A PRECEDING STEP.**

```bash
# If PR closes an issue (check PR body for "closes #N" or "fixes #N"):
test "$(git rev-parse HEAD)" = "$(gh pr view <N> --json headRefOid -q .headRefOid)" \
  && git fetch -q origin <base> \
  && git merge-base --is-ancestor origin/<base> HEAD \
  && gh pr merge --squash --body "closes #N"

# Otherwise: same guard, with --body ""
```

⭐ **THE POSITION IS THE POINT, NOT THE THOROUGHNESS. A check in a preceding step has a window after it; this one has none.** Measured on `matsengrp/phyz` 2026-09-18: **the base moved under a fully approved head five times in one evening, and every catch came from this check at merge time — none from an approval SHA and neither approver noticed.** One instance landed inside **one minute** of both 🤖 approvals being posted; the worker re-checked, found `origin/main` was no longer an ancestor, and did not merge. ⛔ **An approval names a SHA. If the base moves, every approval on that PR is void** — so a merge that "was approved" is not the same claim as a merge whose approvals describe the tree being merged.

⚠ **`HEAD == headRefOid` is the other half and catches a different thing: an unpushed local commit, or a push that raced the approval.** **Both, in the same command, or the guard has a gap.**

⚠ **Do not refactor this back into a tidy preceding step.** It will read better and it will stop working.

Follow the squash merge conventions from global CLAUDE.md — PR title becomes the commit message, body is minimal.

⛔ **`--body` IS NOT OPTIONAL AND IS NOT COSMETIC. Omitting it silently changes which issues the merge closes.** Without `--body`, `gh` defaults the squash commit message to **every branch commit body concatenated**, and GitHub parses that text for closing keywords. Measured on `matsengrp/phyz` 2026-09-15, three merge commits:

| commit | how merged | body lines |
|---|---|---|
| `5dd7e2e5` | this skill (`--body "closes #2671"`) | **3** |
| `99e09aa5` | this skill (`--body "closes #2650"`) | **3** |
| `3af8d743` | a bare `gh pr merge --squash`, no `--body` | **103** |

Inside those 103 lines sat a branch commit whose entire purpose was *"stop closing the issue"* — and which therefore **quoted the line it had removed**, wrapped across a newline:

```
The issue-lead's evaluation found two defects: the PR body said "Closes
#2620" although criteria 1, 3, 4, 5, 7, and 9 remain unmet
```

`Closes` + newline + `#2620` parses. Quotation marks do nothing; there is no escaping syntax. **#2620 closed on merge against the explicit intent of the PR body, the author, and two reviewers.**

⭐ **THE CHECK EVERYONE RAN WAS THE PR BODY, AND IT PASSED.** Two independent readers ran `gh pr view --json body | grep -i closes` and both correctly got nothing, because the PR body genuinely had no `Closes` line — it had been deliberately removed. **The right question is not "does the PR body close an issue" but "what text will GitHub parse at merge time," and under a default squash those are different artifacts.**

➡ **So: always pass `--body` explicitly, and additionally enumerate every issue number the merge would close.** `--body` alone does not cover a wrong or extra number inside the body you wrote, nor anyone landing outside this skill. Run this **before** Step 6 and compare its output against the issues you intend to close:

```bash
BASE=$(gh pr view --json baseRefName -q .baseRefName)
git log --format='%B' "origin/$BASE"..HEAD \
  | tr '\n' ' ' \
  | /usr/bin/grep -oiE '\b(close[sd]?|fix(e[sd])?|resolve[sd]?)\b:?[[:space:]]*((([[:alnum:]._-]+/[[:alnum:]._-]+)?#|GH-)[0-9]+|https?://[^[:space:]]*/issues/[0-9]+)' \
  | /usr/bin/grep -oE '[0-9]+$' | sort -u
```

⛔ **THE `tr '\n' ' '` IS THE ENTIRE POINT AND IT IS NOT DECORATION. `grep` IS LINE-BASED, SO NO PATTERN — INCLUDING `[[:space:]]`, `\s`, OR `[[:space:]]*` — CAN MATCH ACROSS A NEWLINE IN ORDINARY `grep`.** The wrap is the mechanism, so a grep without the `tr` (or without `-Pz`) reads the two halves as unrelated lines and **reports clean on the real defect.**

⚠ **Verified, because the first draft of this very gate had that bug.** Run against `3af8d743`, the actual offending merge commit, a `grep -inE '…[[:space:]]*#?[0-9]+'` with no `tr` exits **1 with no output** — a clean bill of health on the commit the rule exists to catch. **A gate written from a correct diagnosis, by someone who had just finished writing up that diagnosis, still shipped unable to detect the instance.** Test a gate against the artifact that motivated it; a plausible pattern is not evidence.

⛔ **And when you re-test it, extract the pattern from THIS FILE — then verify the extraction round-trips, because a mangled pattern can still return every expected answer.** Three separate manglings were hit while verifying this one gate, and **all three reported PASS**:

- A shell `grep`+`sed` extraction silently dropped the leading `\b`. The run still showed `discloses #999` as clean, so the result looked right while the pattern under test did not exist.
- `echo "$PAT"` renders `\b` as a **backspace**, eating the preceding character — so the *display* of a correct pattern looks truncated, inviting a "fix" to a non-problem. Use `printf '%s'`, or `od -c` for the leading bytes.
- A `${PAT#\\b}` strip intended as the control arm silently did not strip, so the "without-`\b`" arm was actually the *with*-`\b` pattern and both arms agreed — which reads as "the boundary does nothing."

➡ **Assert the extraction, don't eyeball it** (`assert pat.startswith('\\b')`), and where a character's *effect* is the claim, prove it behaviourally with a real control arm rather than by reading the string. Doing that is what showed the boundary is load-bearing; every display-level check had been consistent with it being inert.

⛔ **AND ASSERT THAT THE CONTROL ARM ACTUALLY DIFFERS FROM THE TEST ARM — `assert arm_a != arm_b` BEFORE INTERPRETING EITHER.** The third mangling above is this exact failure: a `${PAT#\\b}` strip silently stripped nothing, so the "without-`\b`" arm *was* the with-`\b` pattern, both arms agreed, and the honest reading of that output was **"the boundary does nothing"** — the precise opposite of the truth. An A/B whose two arms are secretly identical does not report an error; it reports a null.

⭐ **The property that unites all four manglings: a verification harness fails toward AGREEMENT.** A mangled pattern, a stripped-nothing control arm, and a display that eats a character all produce output shaped like confirmation. **A broken gate is loud eventually, because something slips past it. A broken instrument is silent forever, because nobody checks the checker.** That asymmetry is why these asserts are worth their keystrokes and a fifth careful read is not.

⛔ **`/usr/bin/grep` IS PINNED AND THAT IS NOT PEDANTRY — A BARE `grep` ON THIS WORKSTATION IS `ugrep 7.8.4`, WHICH TRUNCATES THIS PATTERN'S ISSUE NUMBER TO ONE DIGIT.** Measured 2026-09-21 on `pax`, this file's own pattern, same input, full pipeline:

| keyword | `/usr/bin/grep` (GNU 3.11) | bare `grep` (ugrep 7.8.4) |
|---|---|---|
| `close #2872` | `2872` | **`2`** ⛔ |
| `fix #2872` | `2872` | **`2`** ⛔ |
| `resolve #2872` | `2872` | **`2`** ⛔ |
| the other six forms | `2872` | `2872` |

**It breaks on exactly the three BARE forms — the ones where `[sd]?` / `(e[sd])?` matches empty — and GitHub honours all three.**

⛔ **THE DIRECTION IS WHAT MAKES THIS WORSE THAN A MISS: IT DOES NOT FAIL TO FIRE. IT FIRES AND NAMES THE WRONG ISSUE.** A reader sees `#2`, finds it nonexistent or ancient, and **dismisses a true positive.** That is this file's own *"a check can emit a confident, specific, wrong answer"*, occurring inside the gate that warning is attached to.

⚠ **And "the `closingIssuesReferences` check below is authoritative anyway" is a reason this was survivable, NOT a reason to leave it: a reader who learns the grep lies stops running the STEP, not just the grep — including the authoritative check sitting beside it.** A gate that is both redundant and wrong is worse than either alone.

⭐ **Why the 20-row table below could not catch it, which is the generalisable half: every pre-existing row used a SUFFIXED keyword (`Closes`, `Fixes`, `Resolves:`), because that is what everyone writes.** The fixture set was drawn from the same habit as the code, so the table was green on a machine where three of GitHub's nine keywords were broken. ➡ **GitHub's closing keywords are a CLOSED SET OF NINE. There is now one row per keyword, so the table is complete BY CONSTRUCTION rather than green by what it happens to contain** — and the next engine change is caught mechanically instead of by someone noticing.

⚠ **Two independent readers each tested a SIMPLIFIED pattern first (`grep -oE '#[0-9]+'` and variants), got agreement between the two engines, and were one message from reporting "cannot reproduce" about a real machine-wide defect.** **A negative result about a PROXY, reported as a negative about the ARTIFACT.** ➡ **Extract the pattern from this file; do not test something like it.**

⚠ **Three details in that pattern are each load-bearing, and all three were added only after a narrower version was tested and found to fail OPEN — silently clean, which is the exact failure mode this gate exists to remove:**

- **`:?`** — GitHub honours `Closes: #N`. Without it, one colon defeats the gate.
- **`([[:alnum:]._-]+/[[:alnum:]._-]+)?#` and the `issues/` URL branch** — GitHub closes on cross-repo `org/repo#N` and on a full issue URL. A pasted issue link is ordinary, and a fleet spanning two repos writes `org/repo#N` routinely.
- **`GH-`** — GitHub honours `GH-123` as an issue reference.
- **`[0-9]+$`, anchored, not bare `[0-9]+`** — once a URL or `org/repo` is inside the match, an unanchored digit class harvests numbers out of the repo path or the hostname.

**Pinned behaviour — 29 rows, all verified. Re-run the whole table if you touch the command; a row with no artifact behind it is not a pin.** ⚠ *This header said `17` while the table held 20; a pinned table whose own count is stale is the shape the table exists to prevent. Count it when you add to it.*

| input | gate reports | verdict |
|---|---|---|
| `3af8d743` — the real defect commit | `2620` | ⛔ fires; intended *nothing* |
| `5dd7e2e5` — clean land | `2671` | ok, matches intent |
| `99e09aa5` — clean land | `2650` | ok, matches intent |
| `81bfa870` — a bare merge whose body happened to be clean | *nothing* | ok |
| `Closes #99`, same line | `99` | positive control |
| `"Closes` ⏎ `#77"` wrapped | `77` | positive control — **the real shape** |
| `Closes matsengrp/phyz#2620` | `2620` | cross-repo |
| `Closes https://github.com/…/issues/2620` | `2620` | full URL |
| same URL with a trailing `/` | `2620` | ok |
| `Fixes https://…/issues/2620#issuecomment-99` | `2620` | anchor does not leak |
| `Closes GH-123` | `123` | `GH-` form |
| `Closes: #2620` | `2620` | colon |
| `Resolves:` ⏎ `#2620` | `2620` | colon **and** wrap together |
| `Closes #11` ⏎ `Fixed #22` | `11 22` | multiple, each keyworded |
| `see #123 for context` | *nothing* | negative — no keyword |
| `this was closed in 2024 by someone` | *nothing* | negative — prose past tense |
| `Closes 2620` (no `#`) | *nothing* | negative — GitHub does not honour it either |
| **`2672`'s own merge: body cites `#1728`, closes `#2672`** | **`2672`** | ⭐ **the only row sourced from production, not construction** |
| `discloses #999` | *nothing* | ⭐ **word-boundary guard** |
| `foreclosed #55` | *nothing* | ⭐ **word-boundary guard** |
| `close #2872` | `2872` | ⭐ **keyword 1/9** — GitHub's closing set is closed; one row each ⛔ **truncates to `2` under bare `grep`/ugrep** |
| `closes #2872` | `2872` | ⭐ **keyword 2/9** — GitHub's closing set is closed; one row each |
| `closed #2872` | `2872` | ⭐ **keyword 3/9** — GitHub's closing set is closed; one row each |
| `fix #2872` | `2872` | ⭐ **keyword 4/9** — GitHub's closing set is closed; one row each ⛔ **truncates to `2` under bare `grep`/ugrep** |
| `fixes #2872` | `2872` | ⭐ **keyword 5/9** — GitHub's closing set is closed; one row each |
| `fixed #2872` | `2872` | ⭐ **keyword 6/9** — GitHub's closing set is closed; one row each |
| `resolve #2872` | `2872` | ⭐ **keyword 7/9** — GitHub's closing set is closed; one row each ⛔ **truncates to `2` under bare `grep`/ugrep** |
| `resolves #2872` | `2872` | ⭐ **keyword 8/9** — GitHub's closing set is closed; one row each |
| `resolved #2872` | `2872` | ⭐ **keyword 9/9** — GitHub's closing set is closed; one row each |

⭐ **The `2672` row is the one to keep if the table is ever trimmed: it is a real merge that landed hours after this gate went in, on precisely the shape that had cost an issue that same morning** — a PR body naming an adjacent issue number while closing only its own. Merge body 3 lines (so `--body` was passed), gate returned `2672` alone, `#1728` still open. **A synthetic row proves the pattern matches; that row proves the safe path was actually taken under pressure.**

⭐ **The two word-boundary rows exist to protect the leading `\b`, and they are the cheapest row in the table.** Dropping the boundary — the obvious "simplification" for anyone tidying this pattern — **fails open on every English word ending in `-close`/`-closed`/`-fix`**: verified, without the `\b`, `discloses #999` matches as `closes #999` and `foreclosed #55` matches as `closed #55`. The boundary is load-bearing rather than decorative, and these rows are what say so to the next editor.

Every number in the output that you do **not** intend to close is a defect. Fix it by rewording the commit body — break the keyword token, or move the number away from it — **not** by narrowing the gate. There is no escaping syntax and quotation marks do nothing.

⚠ **The gate reports bare numbers, not `(repo, number)` pairs.** On a cross-repo `org/repo#N` it tells you `N` and not which repo, so in a multi-repo fleet read the raw `grep -oiE` output (drop the second stage) when a number looks unfamiliar.

⛔ **And do not hand-roll the merge to skip this.** A bare `gh pr merge` run outside this skill also skips Step 6a (EPIC worklog preservation), Step 9.5 (state cleanup), and the `🤖 EPIC worklog preserved` PR comment that `bip-conductor-poll`'s rescue rule reads. The 2026-09-15 instance lost nothing only because that slot's worklog had been preserved the previous night for an unrelated reason.

⛔ **THIS TABLE IS ITSELF A LOADED GUN, AND IT WENT OFF ONCE. When you edit these rows, DO NOT quote them verbatim in your commit message — break the tokens.**

The commit that *added* the two word-boundary rows described them in its own body as "`discloses #999`, `foreclosed #55`". Run the gate on that commit and it reports **`999` and `55`**: `discloses` contains `closes`, `foreclosed` contains `closed`, and this pattern's own `\b` — present in the *file* — is not present in a commit body. It was pushed straight to the default branch, where GitHub honours closing keywords. `matsen/bipartite#999` does not exist and `#55` had already been closed eight months earlier, so the timeline shows only a `referenced` event and **nothing was harmed by luck of two numbers**. Had `#55` been open, the commit adding the gate would have closed an unrelated issue.

⭐ **That is the second time in one session that the documentation of this defect re-committed the defect** — the first was the phyz branch commit whose purpose was removing a `Closes` line and which quoted the line it removed. **It is not a coincidence and it will happen to you: writing about a trigger is the activity most likely to put the trigger in your commit message.** Run the gate on your own commit before pushing, including a direct push with no PR.

⭐ **The generalisation, which transfers well past `gh`: a defect's own documentation is executable context, so quoting a trigger re-arms it — prose is not inert.** Same class as the ⛔-citation rule in `bip-conductor` (a stale prohibition is more durable than a stale fact, because it suppresses the measurement that would expose it). Whenever a commit message, issue body, or skill file needs to *discuss* a syntax that something downstream acts on, break the token rather than quoting it.
⭐ **AND THE THIRD INSTANCE, which shows what has been saving us: a closing keyword inside a CODE SPAN is inert, and the identical text unquoted is an ACTION. The safety comes from MARKDOWN FORMATTING, not from intent or review.** ⚠ **Measured on `matsengrp/phyz` 2026-09-20, and the recursion is exact.** A PR body contained a backticked `` `Closes #2620` `` inside narrative *about* `#2620` having once been closed by a stray closing keyword -- **the failure reproduced inside its own description, for the third time in this document's history.** GitHub did not parse it, so nothing closed; had the author written it unquoted it would have closed a resolved issue. ⛔ **And the same review turned up the other half: that PR closed NOTHING AT ALL, because its own `Closes #<its issue>` line was missing.** One body, one inert keyword that should not have been there and one absent keyword that should.

➡ **So the check is one command and it answers both halves at once, which grepping the body for `closes` does not:**

```sh
gh pr view <N> --json closingIssuesReferences --jq '.closingIssuesReferences[].number'
```

⚠ **The field is EVENTUALLY CONSISTENT: an empty read seconds after a body edit is a LAG, not a failed edit.** Measured on the same PR 2026-09-20 -- immediately after adding the keyword, `gh pr view --json` returned empty while the GraphQL form already resolved; minutes later both returned the number. ➡ **Re-check before concluding the edit did not take, and treat only a PERSISTENT empty as a defect.**

⭐ **It is GitHub's OWN parse, so it is authoritative about code spans, word boundaries and every other tokenisation question this document keeps re-discovering.** ⛔ **Empty output means the PR closes nothing -- which is a defect when the PR has an issue, and the correct state when it does not.** ⚠ **A grep cannot distinguish those and cannot see a code span at all: it reports the inert quoted keyword as live and the missing one as absent, both wrong in opposite directions.** **Run it before requesting a gate, and again after any body edit.**


### Step 6a: Detect worktree mode, and preserve EPIC state before anything can destroy it

**Run this whole step as a single Bash invocation, start to finish — do not split it across separate tool calls.** Cross-invocation persistence is **thread-type dependent**, not a fixed rule: in an agent/subagent thread, both shell variables *and* the working directory reset between separate Bash calls (measured directly this session); in a main interactive session, cwd persists but shell variables still do not. Either way, `$PRIMARY`, `$LAND_DIR`, and `$DEST` below must be set and consumed within this one script to be safe across both thread types. An earlier version of this step split the `cd "$PRIMARY"` out into its own later command, which silently broke worktree-mode landing (`git checkout` in Step 7 would run inside the about-to-be-removed worktree instead of the primary clone) — see issue #2216's follow-up finding.

Before pulling the base branch, check whether you are landing from a linked git worktree (created by `bip spawn` in worktree mode), and — before doing anything else, including the `cd` below — preserve `.epic-status.json`/`.epic-worklog.md` from `$LAND_DIR` if this is an EPIC-managed clone (issue #2216). This has to happen here, not at the old Step 9.5: in worktree mode, Step 8 below runs `bip worktree remove --force "$LAND_DIR"`, which deletes the whole worktree directory — gitignored files included — before Step 9.5 ever gets a chance to run. Doing it here, before any `cd` or removal, covers both modes uniformly: in clone mode `$LAND_DIR` never disappears, so this is equivalent to preserving right before the eventual `rm`; in worktree mode it's the only point that isn't already destroyed.

```bash
LAND_DIR=$(pwd -P)
# Resolve the helper WITHOUT a placeholder you have to fill in, and fail
# loudly if it is missing -- a bare `source` of a nonexistent file does not
# stop the script, it just leaves the functions undefined, and every guard
# below then silently takes its failure branch. See "If you cannot source it".
SPAWN_INTENT=""
for cand in ~/.claude/skills/lib/spawn-intent.sh \
            ~/re/bipartite/skills/lib/spawn-intent.sh; do
    [ -r "$cand" ] && { SPAWN_INTENT="$cand"; break; }
done
if [ -n "$SPAWN_INTENT" ]; then
    source "$SPAWN_INTENT"
else
    echo "PRESERVATION HELPER NOT FOUND -- read 'If you cannot source it' below before continuing. Do NOT search from / for it." >&2
fi
if PRIMARY=$(bip worktree primary 2>/dev/null); then
    echo "Landing from worktree $LAND_DIR (primary: $PRIMARY)"
fi
# THREE-way branch, not two. Neither file -> silent no-op (a plain non-EPIC
# repo; see the paragraph below on why that must stay silent). Status file
# present but NO config -> the gap that lost two worklogs on 2026-09-16;
# warn loudly and preserve into the parent directory, which for a pooled
# clone IS the clone root. Both -> preserve as before.
if [ -f "$LAND_DIR/.epic-status.json" ] && [ ! -f "$LAND_DIR/.epic-config.json" ]; then
    CLONE_ROOT=$(dirname "$LAND_DIR")
    echo "WARNING: $LAND_DIR/.epic-status.json exists but .epic-config.json does not." >&2
    echo "         Falling back to CLONE_ROOT=$CLONE_ROOT (this clone's parent)." >&2
    echo "         Preserving there; the originals are NOT deleted unless that copy verifies." >&2
fi
if [ -f "$LAND_DIR/.epic-status.json" ]; then
    if [ -n "$CLONE_ROOT" ] || CLONE_ROOT=$(resolve_clone_root "$LAND_DIR/.epic-config.json"); then
        DEST=$(preserve_epic_state "$LAND_DIR" "$CLONE_ROOT" \
            "at land of PR #<PR number from Step 2> (\"<PR title from Step 2>\").")
        rc=$?
        # ONE condition, deliberately dull: preserve_epic_state returned 0 AND
        # the destination actually holds a non-empty file. An earlier draft of
        # this line mixed && and || in one test -- a precedence trap, in a fix
        # about correctness. Do not make it clever again.
        if [ "$rc" -eq 0 ] && [ -n "$(find "$DEST" -maxdepth 1 -type f -size +0c 2>/dev/null)" ]; then
            echo "Preserved worklog+status to $DEST"
            # VERIFIED: a non-empty file exists at the destination. Only now is
            # it safe to remove the originals, and this is where it happens --
            # NOT at Step 9.5. `find ... -delete` has no `rm`/`rmdir` token, so
            # the `$`+`rm` destructive-removal guard cannot fire despite the
            # `$`s in scope here.
            find "$LAND_DIR" -maxdepth 1 \( -name '.epic-status.json' -o -name '.epic-worklog.md' \) -delete
            echo "Removed the now-redundant originals from $LAND_DIR"
            # THE THIRD ORCHESTRATION FILE. It is not preserved -- it is machine
            # state (iteration count, max, completion promise, owning session_id),
            # not content -- so it is deleted rather than copied. It is deleted
            # HERE, under the SAME verified-preservation condition as the two
            # above, rather than at its own site: this step's own rule is that
            # the condition to DELETE must never be weaker than the condition to
            # PRESERVE, and a separate unconditional delete elsewhere is exactly
            # the shape that lost two worklogs on 2026-09-16.
            find "$LAND_DIR/.claude" -maxdepth 1 -name 'ralph-loop.local.md' -delete 2>/dev/null
            echo "Removed the ralph-loop state file from $LAND_DIR/.claude"
            if gh pr comment <PR number from Step 2> --body "🤖 EPIC worklog preserved to \`$DEST\` (issue #2216)."; then
                echo "Posted preservation pointer to PR #<PR number from Step 2>"
            else
                echo "WARNING: preservation succeeded ($DEST) but the gh pr comment pointer failed to post -- note the path in Step 10's report so it isn't lost" >&2
            fi
        elif [ "$rc" -eq 2 ]; then
            echo "PRESERVATION FAILED: cp into .preserved did not succeed -- originals left in place ON PURPOSE. Investigate before continuing." >&2
        else
            echo "PRESERVATION DID NOT VERIFY (rc=$rc) -- originals left in place ON PURPOSE." >&2
        fi
    else
        echo "PRESERVATION FAILED: $LAND_DIR/.epic-config.json exists but its clone_root could not be resolved -- stop and investigate. If you hand-copy, the destination is the CLONE ROOT's .preserved/ (the parent of this clone), never a path inside this clone -- see 'If you cannot source it' below" >&2
    fi
fi
if [ -n "$PRIMARY" ]; then cd "$PRIMARY"; fi
```

**If you cannot source it: the destination is `<clone_root>/.preserved/`, NOT anywhere inside this clone — and do not go looking for the helper with an unbounded `find`.**

Both halves of that were measured on `matsengrp/phyz` 2026-09-12/13, on the same clone, twice. A worker could not resolve the old `"$(dirname "<this-skill's-base-directory>")/lib/spawn-intent.sh"` placeholder, ran **`find / -iname "spawn-intent.sh"`** — which on that host descends into a read-only sshfs mount with a documented 25-hour-hang incident behind it — then hand-rolled `mkdir -p` + `cp` and wrote to **`<clone_root>/<clone>/.preserved/`**. Three PRs landed that way. The files survived only because nothing had reused the clone yet; a `bip spawn` prep or a fresh clone would have taken them, and nothing looking in `.preserved/` would ever have found them. Two sibling clones had **43 and 10 files** sitting in the same wrong place from earlier instances of exactly this.

The failure is silent by construction: `source` of a missing file does not abort the script, so `resolve_clone_root` and `preserve_epic_state` are simply undefined, every `if` below takes its failure branch, and the step reports a handled error rather than a missing helper.

So if the loop above found nothing:

```bash
CLONE_ROOT=$(jq -r '.clone_root' "$LAND_DIR/.epic-config.json" | sed "s|^~|$HOME|")
ISSUE=$(jq -r '.issue' "$LAND_DIR/.epic-status.json")
CLONE=$(basename "$LAND_DIR")
DEST="$CLONE_ROOT/.preserved/$ISSUE-$(date -I)"
mkdir -p "$DEST"
cp "$LAND_DIR/.epic-status.json" "$DEST/i$ISSUE-$CLONE.status.json"
cp "$LAND_DIR/.epic-worklog.md"  "$DEST/i$ISSUE-$CLONE.worklog.md"
```

Note `$CLONE_ROOT/.preserved`, not `$LAND_DIR/.preserved`. If a sibling clone appears to have its own `.preserved/`, that is a previous instance of this same bug — do not copy it.

`preserve_epic_state` (from `skills/lib/spawn-intent.sh`) is the shared preserve-then-report pipeline all four preserve sites in this repo now use — extracted per issue #2216 after unchecked `cp`, wrong-order `resolve_clone_root`, and guard-tripping wording each independently recurred across the sites that used to hand-roll it. It produces the un-dotted, clone-namespaced filenames (`i<issue>-<clone>.worklog.md`/`.status.json`) matching this repo's existing `.preserved/` convention (23 of 25 prior entries use this form) rather than the dotted originals, which a plain `ls` or `*` glob would otherwise show as an empty directory. Its return code distinguishes "nothing to preserve" (1, silent) from "preservation failed" (2, must not proceed to delete) — check `$rc` immediately after the assignment, in the same command, since a later command would not see it.

The `gh pr comment` runs immediately, inside this same step, rather than being deferred to Step 9.5 — the PR was already merged in Step 6 above, so the PR number is available here, and posting now means the "committed pointer" `bip-conductor-poll`'s three-part rescue rule requires doesn't depend on a variable surviving through Steps 7/7.5/8/9 to reach Step 9.5 (it wouldn't: same cross-invocation problem as `$PRIMARY`, just further away).

`bip worktree primary` exits 0 and prints the primary clone path **only** when the current directory is a linked worktree; in every other case (primary clone, non-bip checkout, non-git directory) it exits non-zero with no stdout and `$PRIMARY` remains empty. **No `.epic-config.json` at all is not a failure** — it means this land isn't part of a multi-clone EPIC fleet, and there is nothing to preserve into. Every session on the machine picks up this skill the moment it merges (it's a symlink into `~/.claude/skills/bip-pr-land`), including plain non-EPIC repos that have never heard of `$CLONE_ROOT`. Those must see nothing here, not a scary-looking failure message — hence gating on `.epic-config.json`'s presence too, not just `.epic-status.json`'s. A repo that *has* `.epic-config.json` but still fails to resolve `clone_root` from it is a genuinely broken EPIC clone, and that case keeps the loud report.

`resolve_clone_root` (from `skills/lib/spawn-intent.sh`, already used by `bip-conductor-spawn` and `bip-conductor-poll`) fails loudly on stderr and returns non-zero if `.clone_root` is missing or unresolvable, rather than silently yielding an empty path — use it here instead of hand-rolling the same jq+tilde-expand logic a third time (that duplication is exactly what the helper was extracted to stop, per its own header comment). If either failure branch above fires, **do not proceed past it silently** — the whole point of this step is that a `cp` or `mkdir` failure must not read as success.

If `$PRIMARY` was set, Steps 7 and 7.5 below run in the primary clone; otherwise they run in `$LAND_DIR` exactly as today.

### Step 7: Return to base branch and pull

```bash
git checkout <base>
git pull
```

### Step 7.4: Confirm the LANDED TREE is the GATED TREE

⭐ **Every other post-land check confirms the merge happened. This is the only one that confirms it carried what was reviewed** — it closes *approved at SHA X* to *landed content*.

```bash
git rev-parse <landed-sha>^{tree}    # must equal
git rev-parse <gated-sha>^{tree}
```

**It is a TREE comparison, not a range one**, so it is immune to the rebase rewrite that makes `A..HEAD` lie about a branch whose SHAs were rewritten — the same tree-vs-range distinction this fleet keeps re-finding. Measured on `matsengrp/phyz` 2026-09-17: squash `05ee9d02` and gated head `18f327fd` both `b239be25f772dbec…`, `git diff --stat` empty, so both approvals described landed `main` byte-for-byte.

⛔ **Valid only when the head was CURRENT with the base at merge time, and say so wherever you report it.** If the base moved between the gate and the merge, the squash legitimately carries those commits, the trees differ, and this check raises **a false alarm on a good land**. ⭐ **That precondition is the half that must travel with the command** — `/bip-pr-land`'s own `:403` records that *"a guard that degrades silently when its own precondition changes is not a guard,"* and this one's precondition is exactly the base currency the merge step already verifies. **If the base was not an ancestor at merge, this comparison does not apply and the land needs a re-gate rather than a tree check.**

### Step 7.5: Sync the primary clone (clone-mode only)

**Skip this step if Step 6a already `cd`'d to the primary** — Step 7 has already pulled it.

If you landed from a scratch clone (EPIC worker, `bip spawn` without worktree mode, or any working copy that isn't the canonical one in `sources.yml`), the primary clone is now behind `origin/<base>`.
Pull it forward so the canonical checkout matches `main`.

1. Resolve the primary clone path the way `bip spawn` does (mirrors `flow.ResolveRepoPath`: `nexus_path` from `~/.config/bip/config.yml`, repo from `git remote get-url origin`, then `sources.yml` + `config.yml` paths).
   If `$(pwd -P)` already equals it, or the repo isn't listed, skip this step.

2. `git -C "$PRIMARY" pull --ff-only`.
   On failure, warn with the error and continue — the merge is already upstream, nothing is lost.
   Never stash.

3. Report what you did in Step 10.

### Step 8: Remove worktree (if applicable) and delete branch

If Step 6a found that you were landing from a linked worktree, remove the worktree **before** deleting its branch — git refuses to delete a branch that still has a worktree checked out on it:

```bash
if [ -n "$PRIMARY" ] && [ -n "$LAND_DIR" ] && [ "$LAND_DIR" != "$PRIMARY" ]; then
    bip worktree remove "$LAND_DIR"
fi
```

`bip worktree remove` defaults to `--force`, which is required because the squash-merge leaves the worktree carrying commits unreachable from the merged branch.
Then delete the local branch:

```bash
git branch -d <branch>
```

The remote branch is already deleted by `gh pr merge` (GitHub default).
If not, also run: `git push origin --delete <branch>`

### Step 9: Ensure clean main

```bash
git status --porcelain
```

If any untracked or modified files remain on main:
- Move them to `_ignore/$(date -I)-landing/` (create the directory if needed)
- Report what was moved

The goal is a **totally clean `git status`** on main when landing is done.

### Step 9.5: Clean up orchestration files

`.epic-status.json` and `.epic-worklog.md` were already preserved — and, if
preservation succeeded, the "committed pointer" `bip-conductor-poll`'s
three-part rescue rule requires was already posted via `gh pr comment` —
inside Step 6a (issue #2216; see that step for why both have to happen
there, before Step 8's worktree removal, rather than here).

**There is a THIRD orchestration file, `.claude/ralph-loop.local.md`, and it is
also cleared in Step 6a.** It holds the ralph-loop plugin's own machine state —
iteration count, max, completion promise, and the `session_id` that owns it — so
it is *deleted rather than preserved*: there is no content in it anyone will want
back. ⚠ **Landing used to clear two of the three, and the one it left behind is
the one nothing else clears.** Measured 2026-09-18 on `matsengrp/phyz`: a clone
that had just landed a PR sat on clean `main` with no status file and no worklog,
carrying a `ralph-loop.local.md` reading `active: true, iteration: 11`.

⭐ **The failure mode is that it misinforms a READER, not that it breaks a
worker** — and that is precisely why it survived. The plugin's stop hook exits on
a `session_id` mismatch, so a file left by a dead session **cannot** drive a new
one; execution is unaffected. But a conductor deciding whether a slot is free
reads it as a live loop, so the cost is a reclaimed slot held out of service, or
a spawn withheld from a clone that is actually idle.

⛔ **On a `session_id` that is not this land's: delete it anyway, and here is why
that is safe rather than merely convenient.** A mismatched id means the owning
session is gone (the common case — the file is already inert), or a second
session is live in this clone. **The second case is already broken by the two
deletes on the lines above it**, which remove that co-tenant's status file and
worklog without asking, so this line adds no exposure that landing did not
already carry. The guard against co-tenancy is `bip spawn`'s refusal to launch
into a directory a live pane occupies — not a conditional here, which would only
make this one file behave differently from the two beside it. **Do not add a
`session_id` check to this line without also adding one to those.** If Step 6a
reported a `PRESERVATION FAILED` line, **stop and resolve it before
deleting anything** — do not let this step destroy the only copy of a
file Step 6a couldn't back up.

⛔ **NOTHING TO DO HERE ANY MORE, AND THAT IS THE POINT — DO NOT REINSTATE A DELETE AT THIS STEP.**

This step used to run an unconditional `rm -f .epic-status.json .epic-worklog.md`
whenever the status file was present. **That was one half of a defect that lost
two worklogs on 2026-09-16** (`matsengrp/superfamily-pcp`, clones `iron` and
`argon`): Step 6a's preserve was gated on **two** files existing and this delete
was gated on **none**, so a clone with a status file but no `.epic-config.json`
was silently skipped by the preserve and then emptied by this `rm`. Four of nine
clones in that pool were in exactly that state.

**The originals are now removed inside Step 6a, immediately after the preserved
copy is verified non-empty, where `$DEST` is still in scope.** That is not a
stylistic move: a delete in a *later* Bash invocation cannot see whether the
earlier one succeeded (shell variables do not persist across invocations — the
same constraint Step 6a's own opening paragraph is about), so a delete here is
*structurally incapable* of being conditional on the preservation. **The two
operations have to live in one invocation or the weaker condition always wins.**

⭐ **The general rule, worth more than this instance: the condition to DELETE
must never be weaker than the condition to PRESERVE, and a preservation step
must never have a silent no-op branch.** Both halves were violated here, and
neither was visible in the output — the skip printed nothing at all.

⭐ **The family, one layer up: THE SAFETY STEP AND THE PROTECTED STEP MUST AGREE ABOUT WHAT STATE THEY ARE REASONING OVER.** Here the delete's condition was weaker than the preserve's. The sibling is a guard whose comparand is **invalidated by the preserve itself** — measured 2026-09-17, an EPIC body replace whose archive comment moved the very `updatedAt` the conflict check compared against, so a *correct* guard would have fired on a delta its own author created. **A guard that degrades silently when its own precondition changes is not a guard, and a preserve-then-mutate changes that precondition by construction.** The construction, and the second and load-bearing guarantee — verify the preserved copy's **content**, not just the timestamp — are in `/bip-epic` beside its body-push snippet.

### Step 10: Confirm

Report: "Landed #42.
On `<base>`, up to date, worktree clean.
Branch `<branch>` deleted."
If any files were moved to `_ignore/`, list them.
If the primary clone was synced in Step 7.5, say so: "Primary clone `<path>` pulled."
If Step 8 removed a linked worktree, say so: "Worktree `<path>` removed."
⛔ **A LANDING THAT BYPASSES THIS SKILL SILENTLY SKIPS PRESERVATION, AND THE
OBSERVABLE IS THE PR COMMENT.** `gh pr merge` run outside `/bip-pr-land` skips
Step 6a (preserve), Step 9.5 (delete), and the `🤖 EPIC worklog preserved`
comment in one move — **no error, no warning, no trace.** Measured 2026-09-14 on
`matsengrp/phyz`: a PR landed that way and left a **35,432-byte** worklog live
in a pooled clone, where the next spawn's prep would have deleted it with
nothing to show it had existed.

➡ **The check is the pointer comment, not the files.** After any merge:
`gh pr view <N> --json comments --jq '.comments[].body' | grep -c 'worklog preserved'`
— **1 means this skill ran, 0 means it did not.** That contrast is what
diagnosed the instance: the PR that used the skill had 1, the one that did not
had 0, and the second's state files were still sitting in the clone (which
Step 9.5 would have removed). ⚠ **Do not diagnose it from the presence or
absence of `.preserved/<issue>-*`** — absent is also what a preservation that
ran *to the wrong directory* looks like, and that failure mode is real
(`<clone_root>/<clone>/.preserved/` instead of `<clone_root>/.preserved/`; see
"If you cannot source it" above). The comment distinguishes *did not run* from
*ran and failed*; the directory does not.

⚠ A conductor-side worklog mirror (`/bip-conductor-poll`'s
`mirror_worklogs`) now runs every poll cycle as a floor under this, so a
bypassed landing no longer loses the worklog outright. **That is a backstop, not
a licence** — it covers `.epic-worklog.md` and `.epic-status.json` only, is
overwritten rather than archived, and does not post the pointer a reader six
weeks out needs.

If Step 6a preserved EPIC state, say so: "Worklog preserved to `<DEST>`, noted on the PR." (recall the exact path from what Step 6a printed earlier in this same session — it was not carried forward as a shell variable)
