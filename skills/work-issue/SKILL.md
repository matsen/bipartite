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

### Step 5: Verify

Before creating a PR, run quality checks. Most projects use:

```bash
make format   # Apply consistent formatting
make check    # Static analysis / linting
make test     # Run test suite
```

Not all projects define all of these — check the Makefile or CLAUDE.md for what's available.

### Step 6: Create the PR

```bash
gh pr create --title "<concise title>" --body "Closes #$ARGUMENTS"
```

- Include `Closes #$ARGUMENTS` to auto-close the issue on merge
- Do NOT manually close the issue — GitHub handles it when the PR merges
