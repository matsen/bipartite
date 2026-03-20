---
name: deep-plan
description: Enter plan mode with full context preserved — gathers branch/PR/file context before the planning agent starts, so nothing is lost when plan mode clears conversation history.
---

# /deep-plan

Enter plan mode with a rich context brief so the planning agent has everything it needs to produce a self-contained, handoff-ready plan — including verification steps and implementing agent notes.

## Why this skill exists

Plan mode clears conversation history. If you've spent time discussing a bug, constraints, or pitfalls, that context is lost the moment the planning agent starts. This skill captures everything *before* entering plan mode and hands it off in a structured brief.

## Usage

```
/deep-plan
/deep-plan "focus on the serialization bug we discussed"
```

`$ARGUMENTS` is an optional hint to focus the plan on a specific aspect.

---

## Step 1: Gather context

Before entering plan mode, collect the following. Run commands in parallel where possible.

### Git state
```bash
git status --short
git log --oneline -10
git diff origin/main...HEAD --stat
git branch --show-current
```

### PR and issue (if any)
```bash
gh pr view --json number,title,body,baseRefName,headRefName,url 2>/dev/null
gh pr view --json number -q .number 2>/dev/null | xargs -I{} gh issue view {} 2>/dev/null || true
```

### Continuation file (if exists)
```bash
cat _ignore/CONTINUE.md 2>/dev/null || true
```

### Key changed files
```bash
git diff origin/main...HEAD --name-only
```

Read the most relevant changed files to understand the current state of implementation.

---

## Step 2: Compose the brief

Synthesize everything gathered into a structured **Context Brief**. This is what you'll hand to the planning agent. Include:

```markdown
## Context Brief for Planning Agent

### Task
[1–3 sentence description of what needs to be done, inferred from conversation, issue, and/or ARGUMENTS]

### Current Branch & PR
- Branch: `<branch-name>`
- PR: #<number> — <title> (<url>)
- Base: `<base-branch>`

### Implementation State
[What's already been done, what's in progress, what's missing]
- Completed: ...
- In progress: ...
- Not started: ...

### Key Files
[List the most important files with a one-line description of their role]
- `path/to/file.go` — does X
- ...

### Known Constraints & Pitfalls
[From conversation, issue comments, or code inspection]
- ...

### Build & Test Commands
[Project-specific commands for building and running tests]
- Build: `...`
- Test: `...`
- Lint: `...`
```

---

## Step 3: Enter plan mode

Use the `EnterPlanMode` tool now, passing the full Context Brief as the prompt input.

**Instruct the planning agent** (append to the brief):

```markdown
---

## Instructions for the Planning Agent

Produce a complete, self-contained implementation plan. Someone with no prior context should be able to execute this plan without asking questions.

The plan must end with two required sections:

### Verification

Numbered list of concrete shell commands to confirm the implementation is correct. Each step should be runnable and produce observable output. Examples:
1. `go build ./...` — compiles without errors
2. `go test ./... -run TestFoo` — specific test passes
3. `./bip <command> --arg` — expected output shown
4. End-to-end smoke test with real data

### Notes for the Implementing Agent

Key facts the implementing agent needs before starting:
- **Branch**: `<branch-name>` (already exists / needs to be created)
- **PR**: #<number> (already created / create with `gh pr create`)
- **Pitfalls**: [language/library gotchas, version constraints, known tricky areas]
- **Patterns to follow**: [reference files to read as templates]
- **Performance constraints**: [if any]
- **Remote execution**: [if builds must run on a remote host, include the make target]

### Post-Implementation Checklist

After the implementation is complete and verification passes:
1. Run `/pr-review` to catch quality issues before merging
2. Address any blocking findings from the review
3. **Stop and report to the user** — do not merge. The user will run `/land` explicitly when ready.
```

---

## Step 4: Report

After the planning agent finishes, summarize for the user:

- The branch and PR this plan targets
- The top 3 things the implementing agent must know before starting
- Any open questions the plan could not resolve (if any)
- Confirmation that verification steps and agent notes are present in the plan

If the plan is missing verification steps or agent notes, **flag this to the user** — those sections are required.
