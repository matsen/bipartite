---
name: bip-issue-iterate
description: Refine an existing issue draft paragraph by paragraph with the user — verify every claim against code, search for prior art before proposing fixes, and cut what earns no action. For a draft that already exists and needs to get smaller and more correct before filing.
allowed-tools: Agent, Bash, Read, Edit, Write, Glob, Grep, Skill
---

# /bip-issue-iterate

Refine an issue draft that already exists.
The draft gets **smaller and more correct** each round.
It may spawn sibling drafts, and it may end in "this isn't worth filing."

## Usage

```
/bip-issue-iterate ISSUE-my-feature.md
/bip-issue-iterate                        # most recently discussed draft
/bip-issue-iterate 1234                   # a filed issue, edited via gh
```

## Distinct from its neighbors

| Skill | Shape |
|---|---|
| `/bip-issue-next` | draft → check → submit, one pass |
| `/bip-issue-check` | one-shot completeness review, then submit |
| `/bip-issue-iterate` | many rounds, user-driven, subtractive |

Use `/bip-issue-check` as the terminal gate here, not as a substitute.

## The core loop: paragraph by paragraph

The default mode is a walk through the document, one paragraph or section per turn.
It is the pleasant mode and the productive one — small scope, one decision at a time, easy for the user to steer.
Offer it at the start and return to it whenever the user has no specific objection.

For each paragraph, in order:

1. **Load-bearing test.**
   What action does this paragraph change — a parameter, a gate, a decision rule, a file someone edits?
   If nothing, cut it.
   Content that only reassures the reader is not load-bearing.
   Expect to cut half: a paragraph supporting one instruction has usually grown to ten lines.
2. **Verify what survives.**
   Every line number, default value, and behavioral claim gets checked against the code *before* you write the replacement.
3. **Rewrite tighter.**
   State the mechanism once, then the consequence.
4. **Propagate** (see below).
5. **Report**: what was cut, what was kept and why, what you had to correct.

Do not run ahead to the next paragraph unless asked.

## When the user raises an objection

Treat it as technical, not editorial.
A challenge to a premise is not a request to reword.

- **Verify before responding**, not after.
- **Let the conclusion move.**
  A premise-level correction can dissolve a section, shrink the issue to near-nothing, or invalidate the fix you proposed.
  Follow it.
  Do not preserve a wrong claim by softening it.
- **Correct prior turns plainly** where the error changes the work, in one sentence, then continue.

## Standing rules

**Verify, then write.**
Assertions about defaults, dispatch, line numbers, and what a function does are wrong often enough that writing them unchecked is the main source of rework — and a plausible-sounding claim about a function's behaviour is the thing a worker will implement against.

**Search for prior art before proposing a fix.**
Before designing anything, grep for an existing helper, an existing convention in sibling experiments, and a closed issue that already diagnosed it.
Proposing a mechanism the repo already has — or contradicting a convention it already follows — is a Constitution VI violation, the most common failure here, and one that usually inverts the proposed fix rather than merely its framing.

**Simulate for magnitude, then check the simulation matches the code path.**
Right arithmetic against the wrong function is a silent, confident error.
Any number that reaches the issue is either reproducible from an inlined seeded snippet — which you have run in the form it appears — or delegated to a measurement the issue schedules.
Never cite an ad-hoc script: a number nobody can regenerate is one nobody can correct.

**Cut self-argument.**
Drafts refined across rounds accumulate the residue `/bip-remove-metaspeak` targets: passages arguing with a position the draft no longer holds, retractions of proposals never filed, and narration of its own revision.
Squash to the final position.
Keep a rejected approach only as a short "what not to do" when someone would otherwise retry it; drop the history of how it was rejected.

**Propagate before reporting.**
Every change chases through: run counts and arithmetic, cross-references and quoted section names, validation item numbering, success criteria, decision rules, sibling draft files, and the body of anything already filed.
Subtractive edits leave dangling references and stale totals, and those are what make a reader stop trusting the rest.

## Sibling issues

Scope regularly reveals that part of the draft belongs elsewhere — a shared library change riding inside an experiment issue, or a bug found while verifying a claim.

**Write the sibling as its own `ISSUE-*.md` file and tell the user.**
Do not file it.
Do not fold it back in.
Then cross-reference both drafts, and state in each whether either blocks the other.

## Before filing

1. **Fresh adversarial read — only after a premise-level change.**
   If a round changed what the issue *claims*, not just how it reads, spawn a subagent to read the whole document cold.
   Brief it explicitly on what was reframed and tell it to hunt for surviving old framing, asks that contradict the new premise, and validations that no longer test anything.
   Skip this for rounds that only tightened wording.
2. **`/bip-issue-check`** for constitution and completeness.
3. **Ask before filing.**
   Then write the assigned number back into every draft that referred to the file by name.

## Ending without filing

A legitimate outcome.
If the load-bearing test empties the issue, or prior art shows the work is done, say so and recommend closing the draft rather than filing a thin issue.
