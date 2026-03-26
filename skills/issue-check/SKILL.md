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

### Step 2: Load project constitution and design docs

Search for `CONSTITUTION.md` and `DESIGN.md` in the repository root.
If found, load them. The constitution contains process/workflow
principles (MUST/SHOULD rules). The design doc contains technical
architecture decisions. Both will be checked against the issue in
Step 3.

### Step 3: Spawn a review subagent

Use the Agent tool to launch a general-purpose subagent that reads the
issue file and checks for the following. The subagent should also read
any referenced files (data files, config files, source code) to verify
claims. Pass the contents of `CONSTITUTION.md` and `DESIGN.md` to the
subagent if they exist.

#### Constitution alignment

If a `CONSTITUTION.md` exists, check every MUST/SHOULD principle against
the issue. Flag violations as **CRITICAL**. Common things to catch:

- Issue proposes an approach that contradicts a stated principle
- Issue omits something the constitution requires (e.g., file-based
  state for agent workflows, scope tied to a GitHub issue, quality
  gate expectations)
- Issue describes manual steps where the constitution calls for
  automation (or vice versa — automation where human judgment is needed)

#### Design alignment

If a `DESIGN.md` exists, check technical proposals against architecture
decisions. Flag conflicts as **HIGH**. Common things to catch:

- Issue proposes a storage approach that contradicts the data model
  (e.g., database migrations instead of rebuild-from-JSONL)
- Issue introduces dependencies that violate constraints (e.g., CGO,
  cloud-only services without offline fallback)
- Issue writes data outside the nexus pattern

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

#### Ambiguity and placeholder checks

7. **Vague language**: Scan the entire issue for adjectives and adverbs
   that lack measurable criteria. Flag instances of words like "fast",
   "scalable", "robust", "intuitive", "efficient", "significant",
   "reasonable", "appropriate", "properly", "should improve". Each
   flagged term must be replaced with a concrete, quantified criterion.

8. **Unresolved placeholders**: Scan for `TODO`, `TKTK`, `TBD`, `???`,
   `<placeholder>`, `[NEEDS CLARIFICATION]`, `XXX`, or similar markers
   that indicate unfinished thinking. Every placeholder must be resolved
   with concrete content before the issue is submitted.

#### Validation and benchmarking checks

9. **Success criteria**: Are there concrete, measurable success criteria?
   Not vague ("should improve") but specific ("held-out lnL improves by
   >1 nat per lineage on average").

10. **Null model / baseline**: Is there a clearly specified baseline for
    comparison? Is the baseline computation described in enough detail
    to reproduce (formula, software, parameters)?

11. **Evaluation metric**: Is the primary metric well-defined? Is it
    clear how to compute it (what software, what formula, what data)?

12. **Cross-validation / held-out evaluation**: If the issue involves
    model fitting, is the train/test split strategy specified? Are
    leakage risks addressed?

13. **Benchmarks**: Are runtime expectations stated? Are absolute
    numbers reported (not just relative improvements) so future work
    can compare?

14. **Diagnostics**: Are there diagnostic outputs that help debug
    problems (e.g., coverage histograms, convergence plots, sanity
    checks)?

### Step 4: Fix gaps

Based on the subagent's report, edit the issue file to address all
gaps. Prefer adding concrete details (exact column names, formulas,
file paths) over vague placeholders.

**Fix hard-wrapping.** If the file contains hard-wrapped paragraphs (newlines inserted mid-sentence at ~70-80 characters), unwrap them so each paragraph is a single long line. Only newlines for actual structural breaks (between paragraphs, list items, headings). GitHub renders markdown with soft wrapping.

### Step 5: Submit via /issue-file

After fixes are applied, invoke the `/issue-file` skill with the same
file path to create or update the GitHub issue:

```
/issue-file <file_path>
```

### Step 6: Report

Summarize:
- Number of gaps found and fixed (grouped by severity if constitution
  checks were run: CRITICAL / HIGH / MEDIUM / LOW)
- The GitHub issue URL
- Any remaining open questions that need user input
