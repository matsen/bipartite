---
name: work-issue
description: Read a GitHub issue and implement the work described in it
allowed-tools: Bash, Read, Edit, Write, Glob, Grep, Task
---

# /work-issue

Read a GitHub issue and do the work described in it.

## Usage

```
/work-issue 123
```

The argument `$ARGUMENTS` is the issue number.

## Workflow

### Step 1: Read the issue

```bash
gh issue view $ARGUMENTS
gh issue view $ARGUMENTS --comments
```

Read both the issue body and any comments for full context.

### Step 2: Brainstorm clarifying questions

Think hard about the issue requirements. If you have any clarifying questions, **STOP and ask them before writing any code**.

Once everything is clear, proceed.

### Step 3: Create a feature branch

```bash
git pull origin main
git checkout -b $ARGUMENTS-<short-description>
```

Use the issue number as a branch prefix for traceability.

### Step 4: Implement

- Use code from the issue as a starting point when provided
- Follow CLAUDE.md guidelines for the project
- If you start deviating significantly from the issue, **STOP and discuss**
- Continue until the issue is done and all tests pass

### Step 5: Pre-merge check

Run `/pre-merge-check` before creating the PR. This runs the project's quality checklist (formatting, tests, parity, code review, etc.).

### Step 6: Fix review findings

After the pre-merge check:

- **If the review found concrete issues** (stale comments, dead code, missing docs, etc.), fix them immediately and re-run affected checks.
- **If the review raised real design questions** (e.g., "should this be architected differently?"), **STOP and present the question to the user** before proceeding.

### Step 7: Create the PR

```bash
gh pr create --title "<concise title>" --body "Closes #$ARGUMENTS"
```

- Include `Closes #$ARGUMENTS` to auto-close the issue on merge
- Do NOT manually close the issue — GitHub handles it when the PR merges

### Step 8: Summary for the user

After creating the PR, provide a concise summary that highlights anything the user should know or decide on. Always surface:

- **Underlying bugs or tech debt** discovered during the work (even if worked around)
- **Structural concerns** — places where the fix is a workaround rather than a root-cause fix
- **Performance implications** of the approach taken
- **Follow-up issues** that should be filed

The goal is that the user can scan the summary and immediately see if there's something they want to take further action on (e.g., filing a deeper bug, reconsidering the approach, or adding to a future milestone).
