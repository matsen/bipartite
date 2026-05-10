# Prose discipline for issues, PRs, and comments

A GitHub issue, PR description, or review verdict is read once by an implementer or reviewer who needs to act. Optimize for that. The default failure mode is verbose drafts that bury the deliverable under restated motivation, paragraph-form lists, pasted unchanged code, and exhaustive justifications. None of this serves the reader.

Apply these rules whenever you draft or review the body of a GitHub issue, a PR description, or a comment-check report.

## Rules

- **Lead with the deliverable.** First sentence of the body: what will exist after this PR (one column, one function, one rerun). Motivation comes second.
- **State each fact once.** Pick the strongest location (Motivation, Problem, or Background) and put it there only. Don't restate Motivation in Problem with different words. One Success-criteria section, not a per-phase one and an issue-level one.
- **Bullets for enumerations, prose for arguments.** A list of options, files, steps, or pre-registered interpretations is bullets. A causal claim ("X fails because Y") is prose. Don't write enumerations as paragraphs.
- **Show the change site, not its surroundings.** Quote the exact line you're changing or the exact signature you're adding. Don't paste the enclosing function. Docstrings are fine when they ARE the spec; cut them when an implementer would write the same thing unprompted.
- **Drop non-contested options.** If you'd recommend A, propose A. Mention B only if a reviewer would ask "why not B?"
- **Test plan = bugs, not invariants.** List tests that catch implementation mistakes. Don't list math identities ("row sums to 1", "entries ≥ 0", "PAM at D=0 is identity") as separate checkboxes — they're true by construction.

## What this looks like

Before (paragraph-form enumeration, ~6 lines):

> Pre-register: `D* < 50` would mean compara substitutions look like very-shallow-divergence PAM, surprising given the dataset spans eukaryotic kingdoms — investigate. `D* > 400` would mean compara substitutions are saturated beyond the standard PAM range — investigate (likely indicates the PAM model itself is mis-specified for compara, e.g., compara's amino-acid equilibrium is far from Dayhoff's `pi`, in which case the next analysis is to refit `pi` and `Q` on compara directly via standard maximum-likelihood phylogenetics rather than to extend PAM_D further).

After (bulleted enumeration, scannable in seconds):

> Pre-registered interpretations:
> - `D* < 50`: compara looks very-shallow — surprising given the eukaryotic-kingdom span. Investigate.
> - `D* > 400`: substitutions saturated beyond standard PAM. Likely PAM is mis-specified (compara's π far from Dayhoff's). Next step: fit Q, π directly on compara via ML phylogenetics.

## Reviewer flags

When reviewing a draft, flag these specific patterns:

- Buried deliverable (reader has to consume Motivation + Problem before learning what gets built).
- Facts repeated across sections (Motivation says it; Problem restates it; Background restates it again).
- Enumerations written as paragraphs (multi-clause sentence with semicolons or dashes that should be a bullet list).
- Pasted code that isn't the change site (the surrounding function included for context the reader doesn't need).
- Test-plan items that are mathematical identities rather than bug-catching tests.
