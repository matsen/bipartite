# Prose discipline for issues, PRs, and comments

A GitHub issue, PR description, or review verdict is read once, by someone who needs to act on it. Write for that reader. Apply these rules when drafting such a body, and flag violations of them when reviewing one.

## Rules

- **Open an issue with Background, Goal and Design.** Three short paragraphs, for a scientist who knows the field and knows the project only from the Background. **Background.** sets up what the work sits in, with its defining equation when there is one, and links related issues. **Goal.** says what we'll know or be able to do afterwards, not what the issue measures. **Design.** says how the work gets there, with the quantity it compares, if any, in bold. Use an equation or symbol when it is the clearest statement, and define it where it first appears. Keep each paragraph short by stating each fact once, not by counting sentences.
- **Open a PR with what it delivers.** An experiment PR opens with **Result.**: what we learned before giving any number, claiming no more than the body's licensed reading (an epic's claims sign covers it too). Any other PR's first sentence says what exists once it lands, then one or two sentences of why, for a reader who knows the project but not this thread.
- **Openings go after any machine-read header line** such as `EPIC: owner/repo#N`. On GitHub, write each paragraph as one line, since a single newline renders as a line break.
- **When asking someone to decide, lead with the decisive reason.** For each reason, ask whether it would still settle the question if the reader rejected all the others. Lead with the ones that pass, and label the rest as corroboration. The reason you measured yourself feels strongest, and that is not the same as being decisive.
- **State each fact once.** Put it in its strongest location and nowhere else. Have one success-criteria section.
- **A value that can change appears exactly once. A durable rule may be restated where it applies.** Host state, SHAs, line numbers, counts and flags go stale in one copy and not the other. Put the value in the runnable command, and have the prose point at the command.
- **Bullets for enumerations, prose for arguments.** Options, files and steps are bullets. "X fails because Y" is prose.
- **Show the change site, not its surroundings.** Quote the exact line or signature being changed, not the whole enclosing function.
- **Don't write a count of an adjacent list.** "Three instances below" has no test keeping it true, so delete the count instead of maintaining it.
- **Drop options nobody would contest.** If you'd recommend A, propose A. Mention B only if a reviewer would ask "why not B?"
- **A test plan lists bugs, not invariants.** List tests that catch implementation mistakes, not identities that are true by construction.
- **The body is the final state of the diff, not a changelog.** Describe the finished artifact once, in the present tense. Don't write "Reviewer follow-up:", "Update:", a History section, or "originally X, now Y". When the story has settled, rewrite the body from scratch instead of patching it.
- **A label drifts silently from what it labels, and the label is what gets read.** When you change a section, re-read its heading, caption, the body's opening paragraph, commit subject or title against the new content.

## Showing a draft to the user

Write the draft to a gitignored path inside the repo (e.g. `_ignore/<name>.md`), not a scratchpad: the user can't see a scratchpad path, and `glow` is a snap with a private `/tmp`. Open it beside this session's own pane, in the user's light or dark theme (`docs/guides/theme.md`):

```bash
~/.claude/skills/lib/glow-pane _ignore/<name>.md
```

The pane reloads when the file changes, so revise by editing the file; don't paste the draft inline or reopen the pane.

Text the user will paste somewhere themselves (a Slack message, a DM) goes inline in the reply instead, as plain lines set off by blank lines, with no `>` and no code fence. The user copies by Shift-selecting in this session's pane, which doesn't reach a side pane; and the user doesn't want a blockquote's left-hand bar.

## Is the claim true?

Checking the claims is a separate discipline, in `EVIDENCE-DISCIPLINE.md`. Apply it whenever you cite a number, not only when you draft.
