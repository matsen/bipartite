# Prose discipline for issues, PRs, and comments

A GitHub issue, PR description, or review verdict is read once by an implementer or reviewer who needs to act. Optimize for that. The default failure mode is verbose drafts that bury the deliverable under restated motivation, paragraph-form lists, pasted unchanged code, and exhaustive justifications. None of this serves the reader.

Apply these rules whenever you draft or review the body of a GitHub issue, a PR description, or a comment-check report.

## Rules

- **Lead with the deliverable, but don't skip context.** First sentence of the body: what will exist after this PR (one column, one function, one rerun). Motivation comes second — sized for a reader who knows the project but hasn't seen this specific issue or thread: one or two sentences of *why this exists*, not an assumption that the reader already has the backstory. Link related issues if you already know of them (the issue this closes, a prior PR that touched the same area) — but don't go digging for connections that aren't already at hand.
- **State each fact once.** Pick the strongest location (Motivation, Problem, or Background) and put it there only. Don't restate Motivation in Problem with different words. One Success-criteria section, not a per-phase one and an issue-level one.
- **A MUTABLE VALUE APPEARS EXACTLY ONCE; A DURABLE RULE MAY BE RESTATED AT ITS POINT OF USE.** "State each fact once" above is a *verbosity* rule — it stops a reader reading the same argument three times. **This is the correctness half, and it is a different axis: a value that can change between two readings of the document goes stale in one copy and not the other, silently.** Host state, a build flag, a `main` SHA, a line number, an occupancy count, a cluster count, a test total. **Restating a durable RULE next to the place it bites is good and should not be deduped** — a format spec read once at the top has decayed by the time someone writes the block forty minutes later, and the same spec at the point of use has not. The distinction is not length or redundancy; it is whether the thing can silently become false.

  ⛔ **And when a mutable value must appear at all, it belongs in the RUNNABLE COMMAND, with prose pointing at the command — never the reverse.** A concrete, copy-pasteable command is read as *the instruction*; prose around it is read as commentary. Measured 2026-09-15 on `matsengrp/phyz`: a spawn brief gave `zig build --cache-dir .zig-cache-<clone> -j28 <targets>` as its canonical command and said "use `-j8`" in two prose paragraphs elsewhere in the same document. **The worker used `-j28`, which was the correct instinct** — and the author had updated only the prose, because he was editing "the host facts block" and the command lived in "the build conventions block" forty lines away. ➡ **The remedy is to strip the number from the command (`-jN`) and state the value once, in the block that is rewritten wholesale each time** — not to keep two copies in sync.

  ⚠ **Note the asymmetry that makes this worse than ordinary redundancy: the authoritative-looking copy is the one that goes stale, because it is the one nobody is editing when the fact changes.** Four instances the same afternoon, same shape: the `-j` above; an issue body citing `tree_search.zig` line numbers that had drifted; a provenance writer emitting a sha256 prefix under an `_md5` key; and a review stating "all 17 calls" against a measured 36. **In every one, the explanatory copy was current and the load-bearing copy was not.**

- **Bullets for enumerations, prose for arguments.** A list of options, files, steps, or pre-registered interpretations is bullets. A causal claim ("X fails because Y") is prose. Don't write enumerations as paragraphs.
- **Show the change site, not its surroundings.** Quote the exact line you're changing or the exact signature you're adding. Don't paste the enclosing function. Docstrings are fine when they ARE the spec; cut them when an implementer would write the same thing unprompted.
- **When prose enumerates an adjacent list, delete the count instead of maintaining it.** "Three instances below", "all six new tests", "57 keys, leaving 53" — a count in prose that describes a neighbouring list has no test holding it true, so it is a latent desync from the moment it is written, and the next person to add an item desyncs it without noticing. Write "every instance below" or name the property instead. Two measured instances on 2026-09-11, and the pair is the argument: one was a *false* positive (a count comment near an array described a different array 50 lines down, so two reviewers agreed it was broken when nothing was), and one was real (a new bullet silently invalidated its own section's intro count). **A number nobody can check costs a round whether it is right or wrong.** Corollary for the reader: a count near a list is an assumption, not a reading — verify which list it describes before acting on it.
- **Drop non-contested options.** If you'd recommend A, propose A. Mention B only if a reviewer would ask "why not B?"
- **Test plan = bugs, not invariants.** List tests that catch implementation mistakes. Don't list math identities ("row sums to 1", "entries ≥ 0", "PAM at D=0 is identity") as separate checkboxes — they're true by construction.
- **The body is the mental diff, not a changelog.** Write it as if you'd produced the final, correct artifact on the first attempt, then described it once, after the fact. State the premise — enough for someone who hasn't seen the issue to follow — then the current results and interpretation, as present-tense facts. This isn't only about review rounds: a bug you introduced and fixed yourself before ever pushing, an approach you tried and abandoned, a number that changed because you found your own error — none of it belongs in the body, whether or not a reviewer ever saw it. When a review round or a self-caught mistake changes the picture, rewrite the affected section in place. Don't append a new "Reviewer follow-up:", "Round 2:", "Update:", or "History" section on top of the old ones — that forces every future reader to reconstruct the current truth by diffing narrative against itself. Revision narrative (what a reviewer flagged, what turned out to be a bug, what changed between recomputes) belongs in commit messages or PR comments, which are inherently chronological — not in the body, which isn't. This applies with extra force to PRs that get cited by later work: a citable artifact must read as a conclusion, not a transcript of arriving at one.

- **Prose that labels other prose drifts from it silently, and the label is what gets read.** Headings, code comments, commit subjects, issue titles, table captions. **A heading is read by everyone and edited by no one** — widest reach, least scrutiny in the document. Measured 2026-09-17: an EPIC section was rewritten to retract a mechanism and replace it with the measured one, the retraction was added in the body, **and the heading kept asserting the retracted claim.** A skimmer would have taken away exactly the thing the section retracts. Same shape as a code comment claiming a gate the code does not implement — an earlier draft of `/bip-epic`'s push snippet carried `# gated on T2` above a line that compared nothing. ⚠ **And the tripwire that missed it: a retraction audit grepped for the old *phrasing* and found it correctly quarantined inside retractions. It never asked whether anything still ASSERTED the claim in different words** — which a heading, by nature, does. **A retraction check cannot be a grep for the old string.** After any rewrite, **read every heading against the section it heads**; there are never many and it costs a minute. ⛔ **AND THE RULE ABOVE IS NOT ENOUGH ON ITS OWN — it was followed, and a defect shipped anyway.** Measured 2026-09-19 on `matsen/bipartite`: a bullet inserted into `EVIDENCE-DISCIPLINE.md` falsified **both** clauses the *next* bullet used to distinguish itself (*"the only entry here that produces a right-LOOKING answer rather than a wrong one"*, and *"Everything else here is defeated by looking harder"* against a new entry stating in as many words that attention does not reach it). The author read that adjacent sentence and cleared it. Two clauses fix this:
  - ⭐ **SCOPE — it is not only headings. An insertion automatically falsifies any neighbouring claim that QUANTIFIES OVER THE SET YOU JUST ADDED TO**: *"the only entry here"*, *"everything else here"*, *"all three"*, *"in every case"*, *"no other"*. ➡ **So the check is a grep, not a judgement: after inserting into a list, search the surrounding text for `only|every|all|no other|rather than` and read each hit against your new item.** (No tension with *"a retraction check cannot be a grep for the old string"* above — that forbids searching for the **retracted wording**; this searches the **neighbours' quantifiers**, which your insertion never contains.) Both broken claims above are hits, and neither requires knowing what the section is about.
  - ⛔ **VERDICT — ask whether your new text makes the neighbouring claim FALSE. Do NOT ask whether a reading exists in which it survives.** ⚠ **The second question always succeeds, and the author of the insertion is the one person motivated to reach for it.** In the case above the author constructed *"mine produces a wrong answer that looks right"* as a distinction from *"a right-looking answer"* — the same property, offered as the reason it survives. **Same asymmetry as a phrase stronger than its evidence: the author reads their own insertion with the full picture loaded, so the survival reading is available to them and to nobody else.**

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

## Is the claim true?

That is a separate discipline with its own file: **`EVIDENCE-DISCIPLINE.md`** — before citing a measurement, and before writing a correction or a unification. Apply it whenever you cite a number, not only when you draft.
