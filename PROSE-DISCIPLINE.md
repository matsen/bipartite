# Prose discipline for issues, PRs, and comments

A GitHub issue, PR description, or review verdict is read once, by someone who needs to act on it. Write for that reader. Apply these rules when drafting such a body, and flag violations of them when reviewing one.

## Rules

- **Lead with the deliverable, but don't skip context.** The first sentence says what will exist afterwards. Then give one or two sentences of why, written for a reader who knows the project but not this thread. Link related issues if you already know them.
- **When asking someone to decide, lead with the decisive reason.** For each reason, ask whether it would still settle the question if the reader rejected all the others. Lead with the ones that pass, and label the rest as corroboration. The reason you measured yourself feels strongest, and that is not the same as being decisive.
- **State each fact once.** Put it in its strongest location and nowhere else. Have one success-criteria section.
- **A value that can change appears exactly once. A durable rule may be restated where it applies.** Host state, SHAs, line numbers, counts and flags go stale in one copy and not the other. Put the value in the runnable command, and have the prose point at the command.
- **Bullets for enumerations, prose for arguments.** Options, files and steps are bullets. "X fails because Y" is prose.
- **Show the change site, not its surroundings.** Quote the exact line or signature being changed, not the whole enclosing function.
- **Don't write a count of an adjacent list.** "Three instances below" has no test keeping it true, so delete the count instead of maintaining it.
- **Drop options nobody would contest.** If you'd recommend A, propose A. Mention B only if a reviewer would ask "why not B?"
- **A test plan lists bugs, not invariants.** List tests that catch implementation mistakes, not identities that are true by construction.
- **The body is the final state of the diff, not a changelog.** Describe the finished artifact once, in the present tense. Don't write "Reviewer follow-up:", "Update:", a History section, or "originally X, now Y". When the story has settled, rewrite the body from scratch instead of patching it.
- **A label drifts silently from what it labels, and the label is what gets read.** When you change a section, re-read its heading, caption, commit subject or title against the new content.

## Is the claim true?

Checking the claims is a separate discipline, in `EVIDENCE-DISCIPLINE.md`. Apply it whenever you cite a number, not only when you draft.
