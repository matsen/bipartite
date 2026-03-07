---
name: issue-check
description: Review an issue markdown file for completeness, then submit via /issue-file
allowed-tools: Agent, Bash, Read, Edit, Skill
---

# /issue-check

Review a GitHub issue markdown file for implementation-readiness, fix
gaps, then submit via `/issue-file`.

## Usage

```
/issue-check ISSUE-feature-name.md
```

## Workflow

### Step 1: Determine the file path

- If `$ARGUMENTS` is provided, use that as the file path
- Otherwise, check conversation context for the most recently discussed
  issue file (ISSUE-*.md)
- If unclear, ask the user which file to use

### Step 2: Spawn a review subagent

Use the Agent tool to launch a general-purpose subagent that reads the
issue file and checks for the following. The subagent should also read
any referenced files (data files, config files, source code) to verify
claims.

#### Completeness checks

1. **Data paths**: Are all file paths concrete and verifiable? Can
   someone find every referenced file without asking questions? Check
   that paths exist on disk or that remote paths have copy instructions.

2. **Column names / API contracts**: If the issue references specific
   data formats (CSV columns, API fields, config keys), verify them
   against the actual source (read the relevant code or data files).

3. **Algorithm specification**: Is the core algorithm described with
   enough detail to implement? Check for:
   - Mathematical formulas written out explicitly
   - Clear input/output types
   - Edge cases addressed (what to skip, what to include)
   - References for non-obvious algorithms

4. **Prerequisites**: Are all needed packages, tools, and data listed?
   Are version constraints noted where they matter?

5. **Directory structure**: Does the proposed structure follow project
   conventions? (Check CLAUDE.md or experiments/CLAUDE.md for patterns.)

6. **Test config**: If the project requires fast test configs (e.g.,
   < 1 minute), is one specified with concrete parameters?

#### Validation and benchmarking checks

7. **Success criteria**: Are there concrete, measurable success criteria?
   Not vague ("should improve") but specific ("held-out lnL improves by
   >1 nat per lineage on average").

8. **Null model / baseline**: Is there a clearly specified baseline for
   comparison? Is the baseline computation described in enough detail
   to reproduce (formula, software, parameters)?

9. **Evaluation metric**: Is the primary metric well-defined? Is it
   clear how to compute it (what software, what formula, what data)?

10. **Cross-validation / held-out evaluation**: If the issue involves
    model fitting, is the train/test split strategy specified? Are
    leakage risks addressed?

11. **Benchmarks**: Are runtime expectations stated? Are absolute
    numbers reported (not just relative improvements) so future work
    can compare?

12. **Diagnostics**: Are there diagnostic outputs that help debug
    problems (e.g., coverage histograms, convergence plots, sanity
    checks)?

### Step 3: Fix gaps

Based on the subagent's report, edit the issue file to address all
gaps. Prefer adding concrete details (exact column names, formulas,
file paths) over vague placeholders.

### Step 4: Submit via /issue-file

After fixes are applied, invoke the `/issue-file` skill with the same
file path to create or update the GitHub issue:

```
/issue-file <file_path>
```

### Step 5: Report

Summarize:
- Number of gaps found and fixed
- The GitHub issue URL
- Any remaining open questions that need user input
