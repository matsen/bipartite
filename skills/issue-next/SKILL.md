---
name: issue-next
description: Draft a follow-up GitHub issue from a PR decision, review it with /issue-check, and submit
allowed-tools: Agent, Bash, Read, Edit, Write, Glob, Grep, Skill
---

# /issue-next

After a PR lands a decision or result, draft the obvious next issue,
quality-check it, and submit — all in one command.

## Usage

```
/issue-next <PR-URL-or-number>
/issue-next                      # infer PR from conversation context
```

## Workflow

### Step 1: Identify the source PR

- If `$ARGUMENTS` looks like a URL or number, use it
  (`gh pr view <arg> --json number,title,body,comments,reviews,baseRefName,headRefName`)
- Otherwise scan conversation history for the most recently discussed PR
- If still unclear, ask the user

### Step 2: Extract the "what's next" signal

Read the PR body, comments, and review threads. Look for:

- **Explicit next-steps**: "next issue should …", "follow-up:", "TODO for next PR"
- **Decisions with implications**: a hypothesis confirmed/falsified, an
  approach chosen over alternatives, a scope item deferred
- **Unfinished checkboxes** in the PR's test plan or task list
- **Reviewer requests** that were marked out-of-scope for this PR

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

### Step 5: Run /issue-check

Invoke the `/issue-check` skill on the drafted file:

```
/issue-check ISSUE-<slug>.md
```

This will review the issue for completeness (constitution alignment,
data paths, algorithm spec, success criteria, vague language, etc.),
fix gaps, and submit the issue via `/issue-file`.

### Step 6: Report

Summarize:
- The source PR and the decision that triggered this issue
- The new issue URL
- Key success criteria from the issue
- Any open questions or items the user should weigh in on
