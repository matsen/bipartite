---
name: bip-steward
description: Cold-start for the bipartite corpus steward — the session peers propose changes TO. Reads the proposal branch, open PRs, and pending peer proposals, then adjudicates what lands and when.
---

# /bip-steward

Cold-start for the **steward** of this repository: the one session that decides what lands in
`bipartite` and when. Peers — `/bip-conductor` sessions, `/bip-epic` sessions, workers — propose;
the steward adjudicates, batches, and opens the PR.

**Proposing is `/bip-kaizen`'s job and every session runs it. This skill is the other end of that
exchange.** Do not use this skill to invent improvements; use it to rule on the ones that arrived.

Peers:
- `/bip-conductor` — fleet ops for one repo's clone pool
- `/bip-epic` — topic strategy for one EPIC
- `/bip-steward` — corpus intake and adjudication for `bipartite` itself

## Why this role exists at all, and it is not tidiness

⛔ **`~/.claude/skills/*` are symlinks into `~/re/bipartite/skills/`, so an edit in this tree is
live at the next skill invocation in every session on the box, with no git operation and no
event.** There is no staging. A half-finished edit is doctrine to a cold worker that reads it
thirty seconds later.

⚠ **Measured 2026-09-21: two conductor sessions were committing skill fixes straight to `main` in
this tree while a third session had it checked out on a feature branch.** Both were producing
correct, reviewed work. **The problem was not quality — it was that three writers shared one
working tree and one live symlink farm**, so a skill file's content at any instant was whichever
branch happened to be checked out.

➡ **So the steward's product is not better edits. It is a single writer and a queue.**

## Step 1: Establish what you are

```bash
cd ~/re/bipartite && git branch --show-current && git status --short
gh pr list --repo matsen/bipartite --state open
gh issue list --repo matsen/bipartite --state open --limit 30
```

⛔ **Read the checked-out branch before anything else, and say it out loud in your first message
to any peer.** A peer that builds against `main` while the tree is on a feature branch is building
against a file it cannot see. **Two writers in one tree is the normal state here, not an
exception.**

## Step 2: Read the proposal branch

The long-running branch is **`bip-proposals`**. It exists so a proposal has somewhere to sit and
be seen by other peers rather than living in one conversation.

```bash
git log --oneline origin/main..origin/bip-proposals
git diff --stat origin/main...origin/bip-proposals
```

- Peers commit proposals there directly. **They do not push to `main`.**
- The steward batches what is ready into a PR against `main`, discusses the rest, and rebases the
  branch onto `main` after each merge.
- ⚠ **A proposal sitting on that branch is LIVE in every session's skills as soon as the tree is
  checked out on it.** So do not leave the tree on `bip-proposals` while an unreviewed proposal is
  on it — stay on `main` or on your own PR branch, and check the branch out only to work it.

## Step 3: Adjudicate, and the rule that makes this cheap

For each proposal, three outcomes and nothing else: **take it**, **file it**, or **decline it**.

⭐ **RULE ON IT RATHER THAN INHERITING IT.** A proposal you neither take nor refuse becomes an
obligation on the next steward session, which reads it cold and without the exchange that produced
it. Say which of the three it is, in one sentence, to the proposer.

**Take it** — it is small, it is in the area you are already editing, and it needs no decision the
proposer did not already make.

**File it as an issue** when any of these is true, and say which:
- it changes behaviour in a file the current PR deliberately preserves;
- it needs a decision the proposer explicitly left open (which key, which label, which default);
- it needs its own tests.

⛔ **"You are already editing that file" is a timing argument and it is the weakest one available.**
Measured 2026-09-21: a conductor proposed folding a real deletion-path false positive into a PR
whose entire subject was removing a fail-open, on the grounds that the invoking block was already
being rewritten. **The proposal was correct and the timing argument still lost** — folding it in
would have made one PR both remove a fail-open and add an unbudgeted exemption to a deletion path.
Filing it cost the proposer one extra PR and nothing else. ➡ **Weigh what the PR is ABOUT, not
which files it happens to touch.**

**Decline it** when it is a rule drawn from one incident that will age badly, or when it belongs in
the proposer's own repo. `CLAUDE.md`'s own rule: *"Not behavioral rules from one incident — they
age badly and collide with each other."*

## Step 4: Verify the proposal, do not relay it

⛔ **A peer's finding is a report, not evidence. Re-derive it in this tree before acting, and say
that you did.**

This is cheap and it has caught things both ways:
- **It confirmed a defect.** A conductor reported that `fleet-collisions.sh` section 3 cannot
  distinguish a deliberate hold from an abandoned worker. Verified from the file: the section reads
  exactly two fields, `grep -n 'stop_reason\|\.phase'` over the script returns nothing, and the
  live status file was exactly as described. Filed as an issue with the predicate quoted.
- **It corrected a peer's count.** A conductor reported two `⏳` markers in `bip-conductor/SKILL.md`
  and said a third grep hit was a `⚠` matched in prose. There are **three**: the third carries a
  `⏳` at character 260 of its line, inline, which a glance at the line's leading glyph misses.
  Same day, that peer had corrected another peer's marker count — **and the shape repeated one
  level down.**

⭐ **A relayed instruction is weaker than a relayed finding, and the asymmetry is the useful part.**
When this session relayed *"Erick says route changes through me"*, the conductor complied and
disclosed that it was acting on a relay it would normally not treat as binding — **because the
instruction was a RESTRICTION, and the failure mode of wrongly stopping is a stalled edit rather
than an unauthorised action.** ➡ **That reasoning is right and worth reusing: relay restrictions
freely, and route permissions back to the user.**

## Step 5: Land it

`/bip-pr-check`, then `/bip-pr-review`, then `/bip-pr-land`. Nothing special for this role except:

- **A skill-doc change and the command it documents land in the same PR** (`CONSTITUTION.md` III).
  These skills are symlinked live, so a command-only landing leaves running sessions on a
  superseded procedure — and a doc-only landing prescribes a command that does not exist.
- ⚠ **If the PR adds a `bip` subcommand, rebuild `~/re/bipartite/bip`** before telling anyone to run
  it. `~/go/bin/bip` symlinks to that artifact, so the doc can go live while the binary does not.
  Measured 2026-09-21: the skill told sessions to run `bip epic collide` while the binary on `PATH`
  predated the command.
- **Tell the peers what changed in the invocation they run**, not that a PR merged. A conductor
  cares that the script's root argument became required; it does not care about the diffstat.

## Step 6: Prefer the mechanical check to the written rule

⭐ **When a proposal is a rule about keeping two things in sync, the deliverable is usually a test
rather than a paragraph.** `tests/integration/collide_help_test.go` fails if a skill's pasted
`--help` block drifts from the command, which is what makes *"skill behavior MUST match `--help`
exactly"* real rather than aspirational. That test exists instead of a rule telling authors to
re-paste.

⚠ **And run the real-scale check before believing the tests.** Measured 2026-09-21: `bip epic
collide` passed eleven mutation-verified tests against real git fixtures and still examined the
wrong population — 21 directories in a 17-slot pool — because the fixtures were written from the
same specification that had the defect. **A conductor running it against its own pool found it in
one line of output.** ➡ **For anything that reads fleet state, the owning conductor runs it and
reports; the steward does not run it against someone else's live pool.**

## Budget

`/bip-kaizen` Step 2b is the emphasis budget and it applies to this file too. Before adding a
marker here, count what is already present and decide whether the passage outranks them.
