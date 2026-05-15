---
name: bip-issue-next
description: Draft the next GitHub issue — from a PR follow-up, a focus file, or the current conversation — review it with /bip-issue-check, and submit
allowed-tools: Agent, Bash, Read, Edit, Write, Glob, Grep, Skill
---

# /bip-issue-next

Draft an issue, quality-check it, and submit — all in one command. The
"next" issue may come from any source: a PR decision, a deferral filed
by the issue-lead, an idea raised in conversation, or a topic given
inline. There is no requirement that a PR exist.

## Usage

```
/bip-issue-next <PR-URL-or-number>
/bip-issue-next <PR-URL-or-number> --focus-file <path>
/bip-issue-next --focus-file <path>   # no PR; focus comes from file
/bip-issue-next                       # infer from current conversation
```

The `--focus-file` form lets a caller (the issue-lead filing a
specific deferral, or the user piping in a pre-written description)
name the next-action directly, skipping candidate collection and the
"ask if ambiguous" step. The file path points to a plain-text file
whose contents are the focus description (one-line or multi-line both
OK). A file is used instead of a CLI string to avoid shell-quoting
hazards when the description contains quotes, backticks, or
parentheses.

## Workflow

### Step 1: Parse arguments and identify the source

Parse `$ARGUMENTS` as whitespace-separated tokens:
- Extract a PR reference if present (first token that looks like a
  URL or a `#?NNN` pattern). A PR is optional.
- Extract `--focus-file <path>` if present and read the file contents
  as the focus description. Strip leading/trailing whitespace but
  preserve interior content verbatim. If the file is missing or
  empty, stop and report the error — do not fall back to
  candidate-picking.

Duplicate-detection is the caller's responsibility — if called twice
with the same focus, two issues will be filed. The issue-lead guards
against this via `completed_at` (which makes the terminal ceremony
run at most once per session).

Then resolve the source, in this order:
- If a PR URL or number was given, use it
  (`gh pr view <arg> --json number,title,body,comments,reviews,baseRefName,headRefName`)
- Else if `--focus-file` was given without a PR, the focus is the
  source — no PR resolution needed
- Else scan conversation history: the source may be a recently
  discussed PR, an idea raised in conversation, or a stated topic.
  Use whichever is most recent and most relevant
- If still unclear, ask the user what the next issue should be about

The repo for the new issue is whichever is most natural: the PR's
repo if a PR is in play, otherwise the current working directory's
repo (inferred via `gh repo view --json nameWithOwner`).

### Step 2: Determine the next-action

**If `--focus-file` was provided**, use the file's contents as the
next-action directly. Skip candidate collection. If a PR is also in
play, read its body, comments, and review threads only to enrich the
motivation (quantitative results, linked issues) — not to override
the focus.

**Else if a PR is in play**, read the PR body, comments, and review
threads and look for:

- **Explicit next-steps**: "next issue should …", "follow-up:", "TODO for next PR"
- **Decisions with implications**: a hypothesis confirmed/falsified, an
  approach chosen over alternatives, a scope item deferred
- **Reviewer requests** that were marked out-of-scope for this PR
- **Unfinished checkboxes** in the PR's test plan or task list

Collect these into a bullet list of candidate next-actions. If there are
multiple independent next-actions, pick the single most impactful one
(ask the user if it's ambiguous).

**Else (no PR, no focus-file)**, use the current conversation as the
source. The next-action is whatever the user has been discussing or
asked you to file. Skip candidate collection — there is one focus,
and it is the topic at hand. Ask the user only if the topic is
genuinely ambiguous.

Also gather what's relevant from whichever source is in play:

- The **EPIC** or parent issue if referenced (e.g., "EPIC: #285")
- Related issues referenced in the PR or conversation
- Key files, data paths, and function names relevant to the next step

### Step 3: Gather supporting context

Use the repo to fill in concrete details:

1. **Code context**: Read key source files mentioned in the PR to
   understand current state after the PR merges
2. **Experiment results**: If the PR includes benchmark numbers or
   experiment outcomes, note them as motivation / baseline
3. **Existing issues**: Run `gh issue list -R <repo> --limit 20 --json number,title`
   to check for duplicates or related open issues
4. **Project docs**: Check for `CONSTITUTION.md`, `DESIGN.md`, or
   `experiments/CLAUDE.md` in the repo for conventions

### Step 4: Draft the issue file

Write `ISSUE-<slug>.md` in the current working directory. The slug
should be a short kebab-case summary (e.g., `ISSUE-quartet-timing-instrumentation.md`).

**The issue MUST follow matsengrp standards:**

- **Title** (H1): concise, imperative mood (e.g., "Add timing instrumentation for quartet NNI")
- **🤖** robot emoji as the first character of the body (after the H1 title)
- **Motivation**: 2-3 sentences linking back to the source — the PR
  decision, the conversation, or the focus-file context. Reference
  the PR by number if one exists. Include quantitative results if
  relevant
- **Problem / Root cause**: What gap remains after the source PR
- **Proposed implementation**: Phased if complex, with numbered phases
  and explicit phase-gating. Reference exact file paths and function
  names where possible
- **Files to modify**: Bulleted list of files with brief description of
  changes
- **Test plan**: Concrete test cases with expected outcomes; include a
  fast test config if the project expects one (< 1 minute)
- **Experiment** (if applicable): Question, Hypothesis, Conditions
  table, Dataset, Running instructions (exact CLI), Success criteria
  (quantitative thresholds), Diagnostics
- **Success criteria**: Numbered, falsifiable, with concrete thresholds
- **Scope boundaries**: Explicit "In scope" / "Out of scope" lists
- **References**: Link to source PR (if any), EPIC, related issues,
  papers if applicable
- **Depends on / Blocked by**: If there are dependency relationships

**Avoid vague language.** Every adjective must have a measurable
criterion. No "fast", "scalable", "robust" without numbers.

**Apply prose discipline.** Read `PROSE-DISCIPLINE.md` at the bipartite repo root before drafting and apply its rules. Defaults toward shorter: lead with the deliverable, state each fact once, bullets for enumerations, show the change site (not its surroundings), drop non-contested options, list bug-catching tests (not invariants).

**No hard-wrapping.** Write each paragraph as a single long line. Do NOT insert newlines at 70-80 characters within paragraphs or bullet points. Let GitHub's renderer handle line wrapping. Only use newlines for actual structural breaks (between paragraphs, list items, headings).

### Step 5: Run /bip-issue-check

Invoke the `/bip-issue-check` skill on the drafted file:

```
/bip-issue-check ISSUE-<slug>.md
```

This will review the issue for completeness (constitution alignment,
data paths, algorithm spec, success criteria, vague language, etc.),
fix gaps, and submit the issue via `/bip-issue-file`.

### Step 6: Report

Summarize:
- The source (PR + decision, focus-file, or conversation topic) that
  triggered this issue
- The new issue URL
- Key success criteria from the issue
- Any open questions or items the user should weigh in on
