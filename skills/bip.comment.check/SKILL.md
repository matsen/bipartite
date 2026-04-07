---
name: bip.comment.check
description: Fact-check a PR review or comment against the actual code, docs, and tests
allowed-tools: Agent, Bash, Read, Glob, Grep, Edit
---

# /bip.comment.check

Fact-check a PR review, comment, or pasted review text against the actual code. Reviews often contain false claims about missing code, broken imports, or incorrect logic. This skill verifies each claim by reading the relevant files.

## Usage

```
/bip.comment.check [PR number or URL]
/bip.comment.check          # uses current branch's PR
```

Also works when the user pastes review text directly into the conversation.

## Core Principle

**Read the code before evaluating any claim.** Never accept a reviewer's assertion at face value. Every factual claim must be verified against the actual source on the PR branch.

## Workflow

### Step 1: Gather the review claims

Determine the source of claims to check:

1. **If review text was pasted** into the conversation, use that directly.
2. **If a PR number/URL is given**, fetch review comments:
   ```bash
   gh pr view <N> --json reviews,comments
   gh api repos/{owner}/{repo}/pulls/<N>/comments
   ```
   Note: `reviews` contains top-level review bodies; inline code comments (attached to specific lines) come from the `/pulls/<N>/comments` endpoint. Fetch both to get the full picture.
3. **If no argument**, detect the current branch's PR:
   ```bash
   gh pr view --json number,reviews,comments
   ```

### Step 2: Switch to the PR branch

Ensure you're reading the code that the review is actually about:

```bash
git fetch origin
git checkout origin/<pr-branch>
```

If checking out would be disruptive (e.g., you're in a worktree or mid-session), you can read individual files non-destructively:

```bash
git show origin/<pr-branch>:path/to/file.py
```

### Step 3: Identify checkable claims

Parse the review into discrete, falsifiable claims. Focus on:

- **"File X is missing / not updated"** — verify the file exists and contains what's expected
- **"Line N does Y"** — read the actual line
- **"Import / attribute will fail"** — check the actual module
- **"Code at line N has bug Z"** — read surrounding context and evaluate
- **"Test doesn't cover X"** — read the test
- **"Pattern A should be pattern B"** — check if the code already uses pattern B
- **"This is fragile / breaks when..."** — evaluate whether the failure scenario is actually possible given the math or data invariants
- **"Variable/dict is indexed by position N"** — check actual data structure (list vs dict, positional vs keyed)

Skip subjective style preferences that have no factual component.

### Step 4: Verify each claim against the code

For each claim, **read the relevant source files**. Use dedicated tools:

- `Read` to examine specific files and line numbers
- `Grep` to find patterns across the codebase
- `Glob` to check file existence

**Also read project documentation** that may explain design decisions:
- `docs/` directory for mathematical foundations or design docs
- `CLAUDE.md` or `README.md` for project conventions
- Docstrings and comments in the code itself
- Test files that exercise the disputed code

**Run tests** if claims involve broken imports or runtime errors:
```bash
python -c "from module import thing"   # quick import check
pytest tests/relevant_test.py -x       # run specific tests
```

When a claim involves mathematical correctness:
- Read the docs that explain the math
- Trace the actual tensor operations
- Check whether stated invariants actually hold by construction (not "by coincidence")

### Step 5: Classify each claim

For each claim, assign a verdict:

| Verdict | Meaning |
|---------|---------|
| **FALSE** | The claim is factually wrong. The code already does what the reviewer says is missing, or the stated failure doesn't occur. |
| **TRUE** | The claim is correct and actionable. |
| **PARTIALLY TRUE** | The observation is real but the severity, cause, or fix is wrong. |
| **SUBJECTIVE** | Style/taste preference with no factual basis to verify. |
| **WORTH A COMMENT** | Not a bug, but a clarifying comment in the code would prevent future confusion. |

### Step 6: Report

Present findings as a numbered list matching the original review's numbering. For each claim:

1. **State the verdict** (bold)
2. **Quote the key assertion** from the review
3. **Cite the evidence** — specific file:line references showing what the code actually does
4. If FALSE, explain what the reviewer likely misread or missed
5. If TRUE, note whether the suggested fix is appropriate

### Example output format

```
### 1. "model_types.py not updated" — FALSE

The reviewer claims ModelType._VT variants are missing from model_types.py.
Actual: model_types.py:35-48 contains all 10 _VT members. All tests pass.
The reviewer likely looked at an earlier revision or a partial diff.

### 2. "Extra forcing condition needs comment" — WORTH A COMMENT

The observation is correct: variable_tips_model.py:106 adds
`| (nd_dep[:, :, 3] == 1)` which isn't in the parent. The author confirms
this is intentional. A comment explaining the mathematical reason would help.
```

## Guidelines

- **Err on the side of reading more code**, not less. A 30-second file read prevents a false verdict.
- **Check the actual branch**, not main. Reviews are about PR code.
- **Don't assume the reviewer is right** just because they sound confident. Detailed, specific-sounding claims are often the most wrong.
- **Don't assume the reviewer is wrong** either. Verify, don't assume.
- **When claims involve data structures**, check the actual type (dict vs list, keyed vs positional). Reviewers frequently misremember these.
- **When claims involve math**, read the documentation. Mathematical code often looks wrong to someone unfamiliar with the domain.
- **Run the tests.** A claim that "all tests fail" is trivially verifiable.
- **Restore the original branch** when done:
  ```bash
  git checkout <original-branch>
  ```
