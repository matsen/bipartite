---
name: bip-ms-math
description: Revise the mathematics of a TeX manuscript as a series of small, separately checkable commits — proofs in proof blocks, duplicate proofs cut, long paragraphs split, notation and hypotheses repaired, spelling and typos fixed. Works on agent-drafted formal text or on a collaborator's appendix; for a collaborator's text it also leaves review markers and drafts the Slack note.
allowed-tools: Agent, Bash, Read, Edit, Write
---

# /bip-ms-math

Agent-drafted mathematics has a recognizable residue: claims that need proofs sit inside prose, proofs say "routine" or "by the same induction", paragraphs run to fifteen sentences, definitions carry generality nothing uses, and one letter does two jobs.
A professional mathematician cleaning up such a paper does a small number of things, repeatedly.
This skill does them in a fixed order, one commit per kind of change, so a reviewer can skim the mechanical commits and spend attention on the mathematical ones.

## Usage

```
/bip-ms-math main.tex                 # whole manuscript
/bip-ms-math main.tex "Appendix"      # one section, by heading
```

Target one section unless the user asks for the whole paper: every commit gets its own checking agent, so a whole-manuscript run is expensive.

## Two kinds of source

- **Agent-drafted** (the author is the user and an agent): no markers, no Slack note.
- **Human-written** (a collaborator wrote or rewrote the formal text, for example an appendix): every edit a collaborator should check gets a marker (below), and Step 4 produces a Slack note.

Ask which one it is if the git history does not say.

## Step 0: Orient

Read the target whole with `Read`, not `grep`.
Read the repo's `AGENTS.md` or `CLAUDE.md` for house style (for a Matsen-group paper: one sentence per line, `\Cref` for references).
Run the build first so later warnings are attributable to your edits.
List the `\newtheorem` environments and the preamble macros.
Inventory every formal claim: where it is stated, where it is proved, and which hypotheses the proof uses.

## Step 1: Propose the commit list, then wait

Propose the commits below that apply, in this order, and do nothing until the user agrees.
Earlier commits may set up things a later commit has to fix (a theorem labeled before its statement is corrected); that is fine, and the Slack note says so.

1. **One English variety.** Mechanical.
   Skip `%PROV` and comment lines that quote source.
2. **Typos, markup slips, `\Cref`, labels and titles on theorems.** Mechanical.
   No statement changes.
3. **Mathematical corrections.** Marked.
   Inequality directions, definitions used more strongly than stated, a hypothesis an induction needs but the statement omits, unjustified existence steps (well-foundedness), a lemma proved for two sets but applied to a family, base cases that do not match the statement, displays that assert an identity with no proof.
4. **Cut proofs that an appendix now proves.** Marked.
   Replace the prose proof with a pointer; keep examples and intuition.
   Check that the cited result covers the body's generality, and flag any gap.
5. **Formalize prose-only claims.** Marked.
   Each "routine", "by the same induction", or "by stages" becomes a proposition or lemma with a proof block and a label, and the body points to it.
   Facts used by several proofs become lemmas stated first.
6. **Break long paragraphs.**
   One idea per paragraph.
   Move remarks (alternative readings, degenerate cases, implementation caveats) out of definitions into short paragraphs after them.
   Keep the definition complete without them.
7. **Notation: one symbol, one role.** Marked.
   A symbol serving two roles is renamed; each index letter gets one role paper-wide.
   Count occurrences first and choose the letter that changes the fewest lines, even when it is a third letter rather than a swap.
8. **Generality trim.** Ask first.
   State the special form the results use and move the general form to a remark.
   This changes the paper's framing, so it needs the author's decision.

Spot clefts and inert main clauses in the text you add ("it is X that", "the claim to prove is", "holds because"); `@scientific-tex-editor` on only the new text catches them.
Do not run it on the whole paper.

## Step 2: Do each commit, with an independent check

For each commit, in order:

1. Make the change, rebuild, and confirm no new undefined references, duplicate labels or warnings.
2. Spawn an independent `general-purpose` agent to check it before you commit.
   Give it the `git diff` command, the files to `Read` in full, and what to verify; forbid edits.
   Brief a mathematical commit as a line-by-line proof check: wrong directions, definitions stronger than stated, finite versus infinite unions, empty and infinite cases, small examples (including the paper's own), and whether each cited result proves what the pointer claims.
   Brief a pure restructuring as a content-preservation check: every deleted sentence reappears, referents still resolve, no forward references.
   Brief a rename as a role check: every changed line replaced the right index, none was missed, and no letter has two roles.
   Ask for a verdict per item with line numbers and fixes, under 500 words.
3. Read the report yourself, fix every gap, and re-check anything the agent called wrong.
4. For a rename, also confirm mechanically that each changed line differs from the old line only by the renamed token.
5. Commit with a plain one-line subject.
   Push only when the user has said to.

## Markers (human-written source)

Put `%CC @Name -- <what changed and why; what to check>` on the line before every spot the collaborator should look at.
The `%EM`-style convention applies: the comment refers to the line that follows.

- Mark any change to what a statement says or needs, any inequality fix, any new lemma, any removed proof, and any change of notation.
- Do not mark spelling, spacing, `\Cref`, or macro use.
- Leave their own margin comments and `%PROV` lines alone.
- If a body result is more general than the appendix that proves it, ask the collaborator in the marker instead of editing their appendix.

## Step 3: Keep notes as you go

Append to a scratch file after every commit: the hash, what it changed, why it is separate, which earlier commit it depends on or corrects, what the independent check found.
Note open questions for the collaborator.

## Step 4: Slack note (human-written source only)

Write it to `_ignore/` (gitignored) and give the user its path.
Address the collaborator.
Open with thanks and the count of commits, then one short entry per commit with its hash.
Say why the commits are split: the mechanical ones need a skim and the mathematical ones need review.
Say plainly which early commits set up something a later one had to fix.
State how each mathematical commit was checked and what the check found.
End with the open questions, each self-contained, with the options spelled out.

## What not to do

- Do not weaken a statement without saying so.
- Do not edit a collaborator's appendix to resolve a disagreement with the body; ask.
- Do not invent a citation; name the fact and leave a `%TODO`.
- Do not amend commits.
