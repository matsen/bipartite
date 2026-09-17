---
name: bip-pr-file
description: Write a PR body and file the PR — lead with what it does and why, for a reviewer who is not in the details. Use before `gh pr create`.
---

# /bip-pr-file

Write the body, then file the PR.

`/bip-pr-check` and `/bip-pr-review` run on a PR that already exists, and
`/bip-pr-land` merges it. This is the missing first stage: creation is where the
body gets written, and a body nobody can read is not fixable by a later review
pass, because the reviewer has already given up.

## Who is reading

**A reviewer is by definition not involved in the details of the work.** They
know the project in general. They do not know which function you were in, which
issue number covers which defect, or what you found on the way.

So the body's job is to tell them what they are being asked to merge, and why,
before it tells them anything else.

## The shape

### 1. Open with a short paragraph of prose

What this PR does and why, in human-readable form, with whatever background the
reader needs to place it. No heading above it, no table, no bullets. Concise —
a few sentences, not a page.

A PR covering several unrelated changes still gets one opening paragraph.
Combining changes is not a problem and does not stop the body having a thesis;
if you cannot write the paragraph, you do not yet know what you are filing.

This paragraph is not the place for how the branch came about. "Three fixes
found while running step 4", "branched off each other, so this is one PR" —
that is the PR's provenance, and the reader did not ask.

### 2. Then one section per change

Each section: what the problem was, what changed, and why. In that order or
close to it, but the section must **say what changed** — that is the part that
goes missing.

**Open each section with one sentence naming the change**: subject, verb,
object. Then the problem it solves. Then the evidence, last and shortest.

The check is mechanical. Read the first sentence of each section in sequence.
If that sequence does not describe the diff, the body is wrong, however
well-written each section is on its own.

A heading that names a *defect* is the tell:

| this | not this |
|---|---|
| `## alt_2x2_plot now asks where the cache is instead of guessing` | `## alt_2x2_plot read a directory nothing writes` |
| `## The sonnia evaluation now takes a process count` | `## The pgen pool could not be capped` |
| `## Deletes a warning that no longer applies` | `## The stale sonnia-correction comment` |

### 3. Leave out the meandering

The route you took to the change is not content. No "I first thought", no
arguing with a position the PR no longer holds, no narration of what you
checked and in what order. This is `/bip-remove-metaspeak`'s rule; it applies
to a body being written as much as to one being cleaned up.

**One exception, and it is worth taking.** An approach that did not work earns
a sentence when it is an obvious thing to try that cost real time to rule out,
and someone will try it again. Write what not to do and why, not the story of
discovering it:

> Not `duplicate-names.csv`: it crashes on IGKV6D-21, where two OLGA alleles
> map to conflicting partis base genes.

## Also drop

- **A table mapping commit hashes to their own subject lines.** That is
  `git log`, reformatted.
- **A caveat longer than the change it limits.** A "known incomplete" section
  that outweighs the fix reads as a PR that is not ready.
- **Auto-generated commit-message bullets**, and one section per review round.
  `/bip-pr-check` covers those; they are the same failure at a different stage.

Keep, always: every number and its source, every issue and PR reference, every
file path, line number and command, every gate and threshold, and the reasoning
behind a decision the PR is actually making.

## Filing

```bash
gh pr create --title "<what it does, not what was broken>" --body "$(cat <body-file>)"
```

Write the body to a file first and pass it with `--body "$(cat …)"`. `gh` cannot
always read a path given to `--body-file`, and a heredoc inline makes the body
hard to revise.

Anything pushed to GitHub from the command line opens with `> Claude Code:`.

Then `/bip-pr-check` for the mechanical gates.
