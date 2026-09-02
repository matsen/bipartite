# Prose discipline for issues, PRs, and comments

A GitHub issue, PR description, or review verdict is read once by an implementer or reviewer who needs to act. Optimize for that. The default failure mode is verbose drafts that bury the deliverable under restated motivation, paragraph-form lists, pasted unchanged code, and exhaustive justifications. None of this serves the reader.

Apply these rules whenever you draft or review the body of a GitHub issue, a PR description, or a comment-check report.

## Rules

- **Lead with the deliverable, but don't skip context.** First sentence of the body: what will exist after this PR (one column, one function, one rerun). Motivation comes second — sized for a reader who knows the project but hasn't seen this specific issue or thread: one or two sentences of *why this exists*, not an assumption that the reader already has the backstory. Link related issues if you already know of them (the issue this closes, a prior PR that touched the same area) — but don't go digging for connections that aren't already at hand.
- **State each fact once.** Pick the strongest location (Motivation, Problem, or Background) and put it there only. Don't restate Motivation in Problem with different words. One Success-criteria section, not a per-phase one and an issue-level one.
- **Bullets for enumerations, prose for arguments.** A list of options, files, steps, or pre-registered interpretations is bullets. A causal claim ("X fails because Y") is prose. Don't write enumerations as paragraphs.
- **Show the change site, not its surroundings.** Quote the exact line you're changing or the exact signature you're adding. Don't paste the enclosing function. Docstrings are fine when they ARE the spec; cut them when an implementer would write the same thing unprompted.
- **Drop non-contested options.** If you'd recommend A, propose A. Mention B only if a reviewer would ask "why not B?"
- **Test plan = bugs, not invariants.** List tests that catch implementation mistakes. Don't list math identities ("row sums to 1", "entries ≥ 0", "PAM at D=0 is identity") as separate checkboxes — they're true by construction.
- **The body is the mental diff, not a changelog.** Write it as if you'd produced the final, correct artifact on the first attempt, then described it once, after the fact. State the premise — enough for someone who hasn't seen the issue to follow — then the current results and interpretation, as present-tense facts. This isn't only about review rounds: a bug you introduced and fixed yourself before ever pushing, an approach you tried and abandoned, a number that changed because you found your own error — none of it belongs in the body, whether or not a reviewer ever saw it. When a review round or a self-caught mistake changes the picture, rewrite the affected section in place. Don't append a new "Reviewer follow-up:", "Round 2:", "Update:", or "History" section on top of the old ones — that forces every future reader to reconstruct the current truth by diffing narrative against itself. Revision narrative (what a reviewer flagged, what turned out to be a bug, what changed between recomputes) belongs in commit messages or PR comments, which are inherently chronological — not in the body, which isn't. This applies with extra force to PRs that get cited by later work: a citable artifact must read as a conclusion, not a transcript of arriving at one.

## What this looks like

Before (paragraph-form enumeration, ~6 lines):

> Pre-register: `D* < 50` would mean compara substitutions look like very-shallow-divergence PAM, surprising given the dataset spans eukaryotic kingdoms — investigate. `D* > 400` would mean compara substitutions are saturated beyond the standard PAM range — investigate (likely indicates the PAM model itself is mis-specified for compara, e.g., compara's amino-acid equilibrium is far from Dayhoff's `pi`, in which case the next analysis is to refit `pi` and `Q` on compara directly via standard maximum-likelihood phylogenetics rather than to extend PAM_D further).

After (bulleted enumeration, scannable in seconds):

> Pre-registered interpretations:
> - `D* < 50`: compara looks very-shallow — surprising given the eukaryotic-kingdom span. Investigate.
> - `D* > 400`: substitutions saturated beyond standard PAM. Likely PAM is mis-specified (compara's π far from Dayhoff's). Next step: fit Q, π directly on compara via ML phylogenetics.

Before (a PR body that accumulated one section per review round):

> ## Summary
> Fit the codon-level term.
>
> ## Reviewer follow-up: gate decomposition
> Per reviewer request, we also computed...
>
> ## Human-flagged follow-up: CSC/tAI shape check
> A follow-up review asked us to check...
>
> ## History
> Originally the pseudo-R^2 was 0.31, but after fixing a stale-venv bug
> that silently dropped `wiggle_beta`, the recomputed value is 0.42...

After (rewritten from scratch as current state, once the story was done):

> ## Summary
> The codon-level term explains 42% of substitution-conditional variance
> (pseudo-R^2 = 0.42)... [premise, current results, interpretation — no
> mention of which round produced which number]

## Reviewer flags

When reviewing a draft, flag these specific patterns:

- Buried deliverable (reader has to consume Motivation + Problem before learning what gets built).
- Missing context (the body assumes the reader already knows why this issue exists — no orientation for someone who hasn't read the linked issue or thread).
- Facts repeated across sections (Motivation says it; Problem restates it; Background restates it again).
- Enumerations written as paragraphs (multi-clause sentence with semicolons or dashes that should be a bullet list).
- Pasted code that isn't the change site (the surrounding function included for context the reader doesn't need).
- Test-plan items that are mathematical identities rather than bug-catching tests.
- Revision-history framing — "Reviewer follow-up:", "Human-flagged follow-up:", "Update:", a growing "History" section, or "originally X, now Y" phrasing — instead of one narrative describing the current state. Offer to rewrite from scratch rather than patch in place.

## Before citing a measurement

A number in an issue body licenses work. Before one goes in, establish two things about it that its presence in a file does not establish on its own.

- **Is it derived from the data, or from the configuration?** A field can vary across runs, carry a plausible name, and still be a restatement of the flags. Measured on `matsengrp/phyz` 2026-09-02: `--summary-tsv`'s `search_rounds` read exactly 42 in all 43 cells of a sweep spanning 51-351 leaves, 9 parameter values and 3 seeds, because it resolved to `max_failures + free_pool_slots` — inputs the operator had chosen. It looked like search depth and was an echo of the command line. The cheap tell is to predict the value from the flags alone before opening the file; if you can, it is not evidence. A field that varies with `num_leaves` is the harder case, because tracking one input looks more like measurement than tracking none.
- **Does it reach disk, where a reader can open it?** Four separate times in one day the answer was no, in four different ways: a cited `results/raw/` path was gitignored and absent from a fresh clone; the stderr files holding an experiment's primary evidence matched no `*.tsv` negation at any depth; a harness discarded the subprocess stdout carrying the column it was there to collect; and a rescued copy lived at a path nothing committed referenced. Each was invisible in the same way — the run produced the quantity, and nothing that survived the run recorded it.

The two failures compose, and the composition is the one to watch for: a number that is a configuration artifact **and** unreadable by the next person reads exactly like a hard-won result, because nobody can check it. Verify the capture path end to end on one case before running anything at scale, and quote the command that produced the number rather than the file it landed in.

This is the evidence-side twin of the guard rule above: there, a check that cannot read its input reports a safety it never established; here, a citation that cannot be opened reports an observation nobody made.

## Before writing a correction or a unification

Three checks, each answerable in about a minute by someone who does not yet understand the finding — which is what makes them usable, since motivated correction and over-unification both happen under time pressure.

- **A correction that reduces a previously-reported quantity to *no* information deserves a second look before it's written.** Debunking has its own momentum: before recording the stronger retraction, check whether a weaker, still-true claim survives. A count found to be guaranteed in direction by construction is not thereby guaranteed in magnitude — "carries no information" is a more quotable sentence than "carries less than it appears to, in this specific way," and only one of them is usually true.
- **Before accepting a critique of a result: would this objection have been raised if the result had come out the other way?** This separates a validity critique from moving the goalposts, and pre-registration does not — pre-registration protects against choosing a lens after seeing the numbers, not against a measurement having been confounded. "The median is -5.4, so let me find a weighting that shrinks it" fails the test. "The two engines did 13x different amounts of work, so the quantity the rule was applied to may not be the quantity it was designed to judge" passes it — that would be worth stating at +5.4 exactly as much as at -5.4.
- **Before merging two findings into one story: do their remedies land in the same subsystem?** A shared *symptom* is not a shared *mechanism*, and the remedy test is the cheap tell. Two failures both presenting as "far less topology search than the configuration implies" were one search that ran and was cut short by an adaptive stopping rule and one that never started because the tree handed to it had immovable branches — fixes in different subsystems, so not one pattern, and presenting them as one invites a single fix that addresses neither. "Two independent defects with a common symptom" is still a strong claim and a more useful one.
