---
name: bip.issue.check
description: Review an issue markdown file for completeness, then submit via /bip.issue.file
allowed-tools: Agent, Bash, Read, Edit, Skill
---

# /bip.issue.check

Review a GitHub issue markdown file for implementation-readiness, fix
gaps, then submit via `/bip.issue.file`.

## Usage

```
/bip.issue.check ISSUE-feature-name.md
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

6. **Code organization — library vs scripts**: If the repo has a Python
   package (look for `__init__.py` under a top-level directory, or
   `pyproject.toml` with `[project]`), and the issue proposes new `.py`
   files in `scripts/`, `workflow/scripts/`, or `bin/`, check whether
   the core logic should instead live in the library package as a
   reusable module, with only a thin CLI wrapper in the scripts
   directory. Flag as **HIGH** if:
   - The proposed script contains non-trivial logic (algorithms, data
     transformations, model fitting) rather than just CLI argument
     parsing and a `main()` call
   - Similar modules already exist in the library package (check for
     a pattern of library + wrapper)
   - The logic would benefit from unit testing independent of the CLI
   Suggest a concrete module path following the existing package naming
   conventions (check sibling modules for style).

7. **Test config**: If the project requires fast test configs (e.g.,
   < 1 minute), is one specified with concrete parameters?

#### Infrastructure reuse

8. **Existing infrastructure reuse**: Before accepting that the issue
   should build new infrastructure (Snakefiles, pipelines, experiment
   directories, scripts, configs), search for existing work that could
   be extended. This is one of the most common and costly mistakes in
   issue design — building from scratch when 80% of the pipeline
   already exists.

   **How to check:**
   - Search merged PRs for related keywords (dataset names, method
     names, tool names):
     ```bash
     gh pr list --repo <org/repo> --state merged --search "<keywords>" --limit 20
     ```
   - Search the repo for existing Snakefiles, experiment directories,
     or pipeline configs that overlap with the proposed work:
     ```bash
     find <repo_path> -name "Snakefile" -o -name "*.smk" -o -name "config.yml" | head -20
     ```
   - Read the most promising matches to assess overlap.

   **Flag as HIGH if:**
   - An existing Snakefile/pipeline already implements >50% of the
     proposed workflow steps (e.g., same data ingestion, same tool
     invocations, same output structure)
   - The issue proposes a new experiment directory when an existing
     one uses the same datasets and tools
   - The issue creates new wrapper scripts for tools that already
     have wrappers in the repo

   **Recommend:** Extend the existing infrastructure (add rules to the
   existing Snakefile, add config entries, add new targets) rather
   than duplicating it. Name the specific existing file/directory and
   explain what can be reused.

#### Codebase style and structure alignment

8b. **Existing code pattern conformance**: Read the source files most
   relevant to the proposed work (the directory where new code would
   land, plus 2-3 sibling modules) and check that the issue's design
   fits the patterns already established in the codebase. This catches
   drift that DESIGN.md and CONSTITUTION.md cannot — emergent
   conventions that live only in the code.

   **How to check:**
   - Identify where the proposed code would live (package, directory,
     module).
   - Read 2-3 existing files in that area to extract patterns: naming
     conventions (functions, types, files), error handling idiom,
     constructor/factory style, test file layout, and public API shape.
   - Compare the issue's proposed interfaces, type names, function
     signatures, and file organization against those patterns.

   **Flag as HIGH if:**
   - The issue proposes a naming convention that conflicts with
     neighbors (e.g., `NewFooClient()` when siblings use `OpenFoo()`)
   - The issue introduces a structural pattern not used elsewhere
     (e.g., a global registry when the codebase uses dependency
     injection, or callbacks when the codebase uses interfaces)
   - The issue puts files in a location that breaks the existing
     package layout (e.g., a new top-level package when similar
     functionality lives under `internal/`)
   - The issue proposes a public API surface that is inconsistent
     with sibling modules (e.g., exposing struct fields when neighbors
     use getter methods, or vice versa)

   **Recommend:** Name the specific existing files that set the
   pattern and show what the issue should match.

#### Redundancy with existing issues

8c. **Duplicate or overlapping issues**: Search open GitHub issues to
   check whether the proposed work duplicates or substantially overlaps
   with an existing issue. This prevents wasted effort and conflicting
   implementations.

   **How to check:**
   - Extract 3-5 key terms from the issue title and body (feature
     names, tool names, data types, package names).
   - Search open issues:
     ```bash
     gh issue list --repo <org/repo> --state open --search "<keywords>" --limit 15
     ```
   - For any promising matches, read the issue body:
     ```bash
     gh issue view <number> --repo <org/repo>
     ```
   - Assess whether the scope overlaps meaningfully (>30% of tasks
     or the same core deliverable).

   **Flag as HIGH if:**
   - An open issue targets the same feature, dataset, or pipeline
     with substantial overlap in deliverables
   - An open issue is a superset that already includes this work
     as a subtask

   **Flag as MEDIUM if:**
   - An open issue touches related code or data but with a clearly
     different goal (mention it for awareness, not as a blocker)

   **Recommend:** Link to the overlapping issue and suggest one of:
   consolidate into the existing issue, explicitly scope-split with
   cross-references, or close the duplicate.

#### Ambiguity and placeholder checks

9. **Vague language**: Scan the entire issue for adjectives and adverbs
   that lack measurable criteria. Flag instances of words like "fast",
   "scalable", "robust", "intuitive", "efficient", "significant",
   "reasonable", "appropriate", "properly", "should improve". Each
   flagged term must be replaced with a concrete, quantified criterion.

10. **Unresolved placeholders**: Scan for `TODO`, `TKTK`, `TBD`, `???`,
   `<placeholder>`, `[NEEDS CLARIFICATION]`, `XXX`, or similar markers
   that indicate unfinished thinking. Every placeholder must be resolved
   with concrete content before the issue is submitted.

#### Validation and benchmarking checks

11. **Success criteria**: Are there concrete, measurable success criteria?
   Not vague ("should improve") but specific ("held-out lnL improves by
   >1 nat per lineage on average").

12. **Null model / baseline**: Is there a clearly specified baseline for
    comparison? Is the baseline computation described in enough detail
    to reproduce (formula, software, parameters)?

13. **Evaluation metric**: Is the primary metric well-defined? Is it
    clear how to compute it (what software, what formula, what data)?

14. **Cross-validation / held-out evaluation**: If the issue involves
    model fitting, is the train/test split strategy specified? Are
    leakage risks addressed?

15. **Benchmarks**: Are runtime expectations stated? Are absolute
    numbers reported (not just relative improvements) so future work
    can compare?

16. **Diagnostics**: Are there diagnostic outputs that help debug
    problems (e.g., coverage histograms, convergence plots, sanity
    checks)?

#### Correctness validation brainstorm (REQUIRED)

This is one of the most important parts of the review. Scientific code
must be validated beyond basic smoke tests. The subagent MUST actively
brainstorm additional validations that would be convincing evidence the
math and implementation are correct, then check whether the issue
already includes them. If not, flag as **HIGH**.

Think creatively about what tests would actually catch bugs in the
mathematical or algorithmic core. Common categories:

17. **Known-answer tests**: Are there cases where the correct answer
    is known analytically or from a trusted reference implementation?
    For example: degenerate inputs where the formula simplifies,
    textbook examples with published answers, or toy cases small
    enough to verify by hand. Every non-trivial algorithm should have
    at least one known-answer test specified in the issue.

18. **Symmetry and invariance checks**: Does the algorithm have
    mathematical properties that can be tested? Examples: commutativity
    (swapping inputs gives the same result), idempotency (applying
    twice gives the same result as once), conservation laws (quantities
    that should sum to a constant), invariance under permutation or
    relabeling. Each such property is a free correctness check.

19. **Limit and boundary behavior**: What happens at extremes? Does the
    algorithm degrade gracefully or produce known results at boundaries?
    Examples: uniform input, all-zeros, single-element input, very
    large/small values, identity transformations. These often expose
    off-by-one errors and numerical issues.

20. **Comparison to reference implementation**: If a reference
    implementation exists (in R, Python, another language, or a
    published software package), is there a test that runs both and
    compares outputs on realistic data? Matching a trusted
    implementation on non-trivial inputs is strong evidence of
    correctness.

21. **Stochastic / statistical tests**: For randomized algorithms, are
    there distribution-level tests? Examples: checking that samples
    have the correct mean/variance, that a sampler passes a
    goodness-of-fit test, that Monte Carlo estimates converge to known
    values as sample size increases.

22. **Gradient / sensitivity checks**: For optimization or
    differentiable code, are there finite-difference gradient checks?
    For any continuous function, is the output tested for reasonable
    sensitivity to input perturbations?

23. **Round-trip and self-consistency**: Can the computation be checked
    by inverting it or by verifying an internal consistency relation?
    Examples: encode then decode should recover the original,
    likelihood of the MAP estimate should be >= likelihood of
    perturbed values, forward and reverse computations should agree.

The subagent should propose at least 2-3 concrete, specific
validations tailored to the particular algorithm or method in the
issue. Generic suggestions like "add more tests" are not acceptable —
each suggestion must name the specific test, what inputs to use, and
what the expected output or property is.

### Step 4: Fix gaps (with approval gate)

Determine whether the agent authored this issue file in the current
session. This controls whether edits require approval:

- **Agent-authored** (the agent wrote the issue earlier in this same
  session): Edit the file directly to fix all gaps. No approval
  needed — the agent owns the content.
- **User-authored or external** (the file was provided by the user,
  loaded from GitHub, or written in a previous session): **Do not
  edit without explicit user approval.** Present findings first.

#### When approval is required

Present the subagent's findings as a concise summary:
- List each gap found, grouped by severity (CRITICAL / HIGH / MEDIUM / LOW)
- For each gap, state what the problem is and what the proposed fix would be
- Note any hard-wrapping issues (paragraphs broken at ~70-80 chars that should be single lines)

If the findings are substantial (more than 3-4 items, or any
CRITICAL/HIGH), write them to a temporary file and open for review:

```bash
cat > /tmp/bip.issue.check-findings.md <<'EOF'
... findings ...
EOF

if [ -n "$TMUX" ]; then
    tmux display-popup -w 80% -h 80% -E -- less /tmp/bip.issue.check-findings.md
elif [ "$TERM_PROGRAM" = "zed" ]; then
    zed /tmp/bip.issue.check-findings.md
fi
```

Then **stop and wait** for the user to confirm which edits to apply.
Do not proceed until the user says to go ahead.

#### Applying edits

Whether auto-applying (agent-authored) or after approval
(user-authored), fix gaps by adding concrete details (exact column
names, formulas, file paths) over vague placeholders.

**Fix hard-wrapping.** If the file contains hard-wrapped paragraphs (newlines inserted mid-sentence at ~70-80 characters), unwrap them so each paragraph is a single long line. Only newlines for actual structural breaks (between paragraphs, list items, headings). GitHub renders markdown with soft wrapping.

### Step 6: Submit via /bip.issue.file

Invoke the `/bip.issue.file` skill with the same file path. It will open
the file for the user to review after submitting:

```
/bip.issue.file <file_path>
```

### Step 7: Report

Summarize:
- Number of gaps found and fixed (grouped by severity if constitution
  checks were run: CRITICAL / HIGH / MEDIUM / LOW)
- The GitHub issue URL
- Any remaining open questions that need user input
