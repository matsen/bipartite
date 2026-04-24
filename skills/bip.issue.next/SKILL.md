---
name: bip.issue.next
description: Draft a follow-up GitHub issue from a PR decision, review it with /bip.issue.check, and submit
allowed-tools: Agent, Bash, Read, Edit, Write, Glob, Grep, Skill
---

# /bip.issue.next

After a PR lands a decision or result, draft the obvious next issue,
quality-check it, and submit — all in one command.

## Usage

```
/bip.issue.next <PR-URL-or-number>
/bip.issue.next <PR-URL-or-number> --focus-file <path>
/bip.issue.next                      # infer PR from conversation context
```

The `--focus-file` form lets a caller (typically the issue-lead filing
a specific legitimate deferral) name the next-action directly,
skipping the candidate collection and the "ask the user if ambiguous"
step. The file path points to a plain-text file whose contents are the
focus description (one-line or multi-line both OK). A file is used
instead of a CLI string to avoid shell-quoting hazards when the
description contains quotes, backticks, or parentheses.

## Workflow

### Step 1: Parse arguments and identify the source PR

Parse `$ARGUMENTS` as whitespace-separated tokens:
- Extract the PR reference (first token that looks like a URL or a
  `#?NNN` pattern)
- Extract `--focus-file <path>` if present and read the file contents
  as the focus description. Strip leading/trailing whitespace but
  preserve interior content verbatim. If the file is missing or
  empty, stop and report the error — do not fall back to
  candidate-picking.

Duplicate-detection is the caller's responsibility — if called twice
with the same focus, two issues will be filed. The issue-lead guards
against this via `completed_at` (which makes the terminal ceremony
run at most once per session).

Then resolve the PR:
- If a URL or number was given, use it
  (`gh pr view <arg> --json number,title,body,comments,reviews,baseRefName,headRefName`)
- Otherwise scan conversation history for the most recently discussed PR
- If still unclear, ask the user

### Step 2: Determine the next-action

**If `--focus-file` was provided**, use the file's contents as the
next-action directly. Skip candidate collection. Read the PR body,
comments, and review threads only to enrich the motivation
(quantitative results, linked issues) — not to override the focus.

**Otherwise**, read the PR body, comments, and review threads and
look for:

- **Explicit next-steps**: "next issue should …", "follow-up:", "TODO for next PR"
- **Decisions with implications**: a hypothesis confirmed/falsified, an
  approach chosen over alternatives, a scope item deferred
- **Reviewer requests** that were marked out-of-scope for this PR
- **Unfinished checkboxes** in the PR's test plan or task list

Collect these into a bullet list of candidate next-actions. If there are
multiple independent next-actions, pick the single most impactful one
(ask the user if it's ambiguous).

Also gather from the PR:

- The **repo** (`owner/repo`) — the new issue will be filed here
- The **EPIC** or parent issue if referenced (e.g., "EPIC: #285")
- Related issues referenced in body or comments
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
- **Motivation**: 2-3 sentences linking back to the source PR decision.
  Reference the PR by number. Include quantitative results if relevant
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
- **References**: Link to source PR, EPIC, related issues, papers if
  applicable
- **Depends on / Blocked by**: If there are dependency relationships

**Avoid vague language.** Every adjective must have a measurable
criterion. No "fast", "scalable", "robust" without numbers.

**No hard-wrapping.** Write each paragraph as a single long line. Do NOT insert newlines at 70-80 characters within paragraphs or bullet points. Let GitHub's renderer handle line wrapping. Only use newlines for actual structural breaks (between paragraphs, list items, headings).

### Step 5: Run /bip.issue.check

Invoke the `/bip.issue.check` skill on the drafted file:

```
/bip.issue.check ISSUE-<slug>.md
```

This will review the issue for completeness (constitution alignment,
data paths, algorithm spec, success criteria, vague language, etc.),
fix gaps, and submit the issue via `/bip.issue.file`.

### Step 6: Report

Summarize:
- The source PR and the decision that triggered this issue
- The new issue URL
- Key success criteria from the issue
- Any open questions or items the user should weigh in on
