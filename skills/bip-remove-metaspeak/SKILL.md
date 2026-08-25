---
name: bip-remove-metaspeak
description: Cut what does not earn its place out of a document or PR body — internal dialogue, meta-commentary, and whole passages the document itself undercuts — while preserving every fact, number, citation, gate, and argument that justifies a decision the document is actually making. For issues, EPICs, brainstorms, design docs, and PR descriptions that read like a transcript of someone reasoning out loud.
allowed-tools: Agent, Bash, Read, Edit, Write
---

# /bip-remove-metaspeak

Documents written incrementally with an agent accumulate a characteristic
residue: they narrate their own drafting, argue with positions they used to
hold, and pad every claim with rhetorical scaffolding. They also keep
material that stopped being load-bearing — an argument the document raises
and then demolishes, a proposal it talks itself out of, a retraction of
something never published. Both have to go: the register is wrong, and some
of the content has not earned its place. It reads like a transcript of
someone still making up
their mind, not a settled technical document.

The fix is a squash, in the git sense: many small revisions, each arguing
with the one before it, collapse into a single statement of the final
position. Nothing about the reasoning that justified a decision is lost —
only the record of having changed your mind about it.

This applies just as well to a PR body as to a file on disk: a description
written across several rounds of push-and-revise tends to accumulate the same
residue, and it's exactly the artifact reviewers read first.

## Usage

```
/bip-remove-metaspeak ISSUE-my-epic.md
/bip-remove-metaspeak brainstorming/my-doc.md
/bip-remove-metaspeak                     # most recently discussed doc
/bip-remove-metaspeak PR 123              # squash that PR's body via gh
/bip-remove-metaspeak PR                  # PR for the current branch
```

## When NOT to use it

- **On a document that is genuinely a conversation record** — meeting notes,
  a decision log, a postmortem where "we first thought X, then found Y" is
  the content rather than the packaging.
- **On someone else's prose**, unless asked. This changes voice.
- **Before the content is settled.** Compressing an argument you are still
  having wastes the pass; run it when the thinking is done.

## Workflow

### Step 0: Decide what earns its place

Do this before the register pass, on the whole document, and act on it — this
is a removal step, not a report. Four signals mark content that has stopped
being load-bearing:

1. **A section raises an argument and then argues it fails.** Keep the
   conclusion as a plain statement if a reader would otherwise walk into the
   same argument. Delete the rest. Never keep both the argument and its
   refutation.
2. **A proposal arrives with reasons not to take it.** Delete the proposal.
   If the reasons matter, one sentence: "not X, because Y."
3. **Prose adjacent to a table or list restates its contents.** Keep the
   table. Keep only the prose that argues something the cells cannot.
4. **A section is wholly conditional on another section's outcome.** Collapse
   it to one conditional sentence where the outcome is decided.

Three more that recur, all deletions with nothing kept: framing the reader
already agreed to in an earlier exchange or an earlier section; a retraction
of a claim never published; advice about a method nobody proposed.

This step is why the pass exists at all. Observed twice in one session:
register-only passes cut 2.7% and 8% of two documents and reported them
"already dense," after which the author cut 60% and 24% of the same
documents — almost all of it material matching the signals above, which the
register pass had carefully tidied on its way past.

Section-level deletions are recoverable: Step 2 takes a copy, and the
deliverable requires every cut passage be quoted back. Cut, then let the
review in Step 5 restore what mattered. Erring toward the cut is correct
here; erring toward keeping is what produced the two failures above.

### Step 1: Determine the target

Use `$ARGUMENTS` if given; otherwise the most recently discussed document in
conversation. If unclear, ask.

If the target is a PR (`PR 123`, a PR URL, or bare `PR` for the current
branch's PR), there is no file on disk — fetch the body instead:

```bash
gh pr view <number-or-branch> --json body -q .body
```

Keep the fetched text in memory as the "before" copy; the rest of this
workflow treats it the same as a file's contents, substituting "PR body" for
"file" throughout. Applying the result means posting it back, per Step 5a.

### Step 2: Get a recovery copy before the pass

This is a large in-place rewrite, so a clean way back matters more than how
it's taken.

```bash
git status --short <file>
```

- **Tracked and clean:** nothing to do — `git diff`/`git checkout <file>` are
  the recovery path.
- **Tracked and dirty:** say so and offer to commit before the pass. Do not
  commit without approval.
- **Untracked or gitignored** — this is the common case for `ISSUE-*.md`
  files, which this repo's CLAUDE.md forbids ever committing — there is no
  git history to fall back on. Instead copy the file to the scratchpad
  (`cp <file> <scratchpad>/<name>.before.md`) and use that copy, not git, as
  both the recovery path and the diff baseline in Step 5.

PR bodies have no working tree to protect — skip this step for a PR target;
the "before" copy from Step 1 already serves the same purpose.

### Step 3: Spawn a subagent — one per section on anything large

Do this in a subagent, not inline. The pass requires reading the whole
document closely and making hundreds of small edits, and the judgment calls
are local. Pass the file path (or PR body text) and the rules below **in
full** — the keep-list is what stops the agent from cutting the argument
along with the padding.

**Size matters, and getting it wrong wastes the run.** A single agent doing
sequential in-place edits over a long document is a long-running,
non-resumable job: one API hiccup and you get a partial rewrite with no
report. Observed: a ~5,500-word EPIC completed cleanly in one pass; an
~18,000-word document died partway with ~2% done and no deliverable.

- **Under ~8,000 words:** one subagent, whole document.
- **Over that:** fan out, one subagent per top-level section, run in
  parallel. Give each the same rules and keep-list plus its own section
  range. Sections are independent for this purpose — the pass makes local
  edits and never needs cross-section context.
- **Either way, get the recovery copy first (Step 2).** A partial rewrite is
  recoverable only if there is a clean commit or before-copy behind it.

If a run does die partway, do not assume the target is broken. Verify instead
(Step 5's integrity checks are fast), keep the partial work if it passes,
and re-run only the untouched sections.

### Step 4: Add document-specific keep items

Before spawning, skim the document for what would be expensive to lose, and
name it explicitly in the prompt. Typically:

- Quantitative claims and their sources ("22 of 28 rates pegged at bounds")
- Issue/PR references and citation keys
- Gates, success criteria, acceptance thresholds
- File paths, function names, command examples
- Structural elements: heading hierarchy, phase numbering, tables,
  placeholder sections, a leading emoji convention

Generic instructions produce generic damage. A named keep-list does not.

## The rules to pass to the subagent

### Remove

1. **Narration of the drafting process.** "Two gaps assumed while drafting
   this turned out to be already closed"; "this section used to claim";
   "the document is written that way". State the current fact.
2. **Arguing with a prior or anticipated position.** "This is *not* the
   circularity it first looks like"; "worth being precise about what this
   does *not* address"; "the right mental model is X, not Y". Assert the
   correct thing once.
3. **Throat-clearing and meta-framing.** "Worth noting", "worth stating",
   "note that", "two things this is not", "work it through and…",
   "it is worth being precise".
4. **Hedging and intensifying adverbs that add no meaning**: frankly,
   genuinely, deliberately, simply, actually, emphatically, clearly, and
   *precisely* when used for emphasis rather than for precision.
5. **Performative standards language.** "'We did X because it was
   convenient' is not an outcome this gate passes" → "X is adopted only if
   its diagnostics support it." State the requirement.
6. **Tangents and asides** — usually dash-set — that do not affect a
   decision.

### Keep

- Every number, and its source.
- Every reference, link, citation key, path, function name, command.
- Every gate and criterion, in full substance.
- **All reasoning that justifies a decision the document is actually
  making.** This is the load-bearing distinction and the one an agent gets
  wrong. "We chose A over B because B fails when C" is content, not padding,
  even when it runs several sentences. Compress the prose; keep the logic.
  The qualifier is what stops this rule from protecting Step 0's material:
  an argument the document raises and refutes, or a proposal it undercuts,
  is also reasoning and also justifies a decision — the decision not to do
  it. State that decision once and cut the deliberation behind it.
- Markdown structure: heading hierarchy, numbering, tables,
  placeholders. Heading *hierarchy* is structure and must survive; heading
  *text* is prose and is in scope — a heading can narrate ("What I do not
  know about X, and how to find out") and should be squashed like any other
  sentence.

### Style target

Declarative and dense. Short direct sentences over long em-dash-chained
ones. Do not convert argument-carrying prose to bullet fragments. Do not add
claims, soften technical specificity, or invent anything.

Do not set a numeric cut target in the prompt: an agent told to cut a
percentage will hit it regardless of whether the material supports it, and
the last few points come out of the argument. Ask instead for "as much as
comes out under these rules" and let the result be whatever size the squashed
version actually is — sometimes that's a large cut, sometimes barely
anything, if the document was already tight.

"Already tight" is a real outcome and also the easiest wrong answer, so it has
to be earned rather than asserted. A report claiming it must name the passages
it considered and rejected, or it is indistinguishable from a pass that did
not look. If Step 0's signals were checked and none fired, say so in the
report.

### Deliverable to request

1. Every passage cut entirely, quoted by its first few words, with
   Step 0 removals listed separately from register cuts — those are the ones
   most worth a second look.
2. Anything it was unsure about, for review.

Item 2 is the point. The uncertain calls are where substance gets lost, and
they are cheap to review when listed.

## Step 5: Review before accepting

**First, integrity-check the target** — always, and especially if a run died
partway. Diff against whatever the recovery copy from Step 2 turned out to
be: for a tracked file, that's git; for an untracked/gitignored file (e.g.
`ISSUE-*.md`) or a PR body, it's the scratchpad/in-memory "before" copy.

```bash
# tracked file:
git diff --stat <file>
git show HEAD:<file> | grep -oE '`[A-Z][A-Za-z-]+[0-9]{4}-[a-z]{2}`' | sort -u > /tmp/k_old
# untracked file or PR body — diff against the before-copy instead:
diff <scratchpad>/<name>.before.md <file>
grep -oE '`[A-Z][A-Za-z-]+[0-9]{4}-[a-z]{2}`' <scratchpad>/<name>.before.md | sort -u > /tmp/k_old
# either way, check citation keys and structure survive:
grep -oE '`[A-Z][A-Za-z-]+[0-9]{4}-[a-z]{2}`' <file> | sort -u | diff /tmp/k_old -
grep -c '^## \|^### ' <file>
```

Then spot-check a handful of the document's most distinctive numbers. A pass
that silently dropped a citation or a figure is the failure mode worth
catching, and it takes thirty seconds to rule out.

Then read the report, not just the diff. Two checks:

- **Scan the Step 0 removals first.** A whole section is the expensive
  mistake to get wrong in either direction. Restore one only if it states a
  decision the document is making, rather than deliberation behind a decision
  already stated elsewhere.
- **Scan the register "cut entirely" list** for anything that was a policy
  statement or a decision rather than framing. Restore those.
- **Check flagged uncertainties.** The agent is right to be unsure about
  short sharp lines that read as illustrative but are actually the crispest
  statement of a rule.

If the agent reports it found little to cut because the remaining prose is
fact-carrying, that is usually true and usually points at *structural*
redundancy — the same argument restated in a design section, a gate, a risk,
and a success criterion. That is a document-structure decision for the
author, not something this pass should fix silently.

### Step 5a: Applying a PR body

A PR body is visible to others the moment it's posted — per this repo's
GitHub Action Sequencing rule, show the squashed body and get explicit
approval before applying it. Once approved:

```bash
gh pr edit <number-or-branch> --body-file <scratchpad>/pr-body-squashed.md
```

## Step 6: Report

What came out by category, anything restored, and any structural redundancy
the pass surfaced but did not touch.
