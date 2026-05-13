---
name: bip-pr-review
description: Run comprehensive pre-merge quality checklist for current branch's PR
---

# /bip-pr-review

Run a comprehensive quality checklist before merging a PR. Automatically detects project type and runs appropriate checks.

## Usage

```
/bip-pr-review
```

## Workflow

### Step 0: Check for Project-Specific Checklist

First, check for a project-specific checklist:
1. Look for `PRE-MERGE-CHECKLIST.md` in the repo root
2. If not found, read the project's `CLAUDE.md` and look for a "Pre-PR Quality Checklist" or "Pre-Merge Checklist" section

**If a project-specific checklist is found, follow those steps exactly** instead of the generic workflow below.

### Step 1: Detect Project Type

Examine the repository to determine what checks apply:

| Indicator | Project Type | Agents to Use |
|-----------|--------------|---------------|
| `workflow/*.smk` or `Snakefile` | Snakemake pipeline | `snakemake-pipeline-expert` |
| `*.py` files in `src/` or project root | Python project | `clean-code-reviewer` + `code-reuse-reviewer` |
| `build.zig` or `*.zig` files in `src/` | Zig project | `zig-code-reviewer` |
| `go.mod` | Go project | `clean-code-reviewer` + `code-reuse-reviewer` |
| `package.json` | Node.js project | `clean-code-reviewer` + `code-reuse-reviewer` |

The two reviewers operate in **different modes** and complement each other — run them in parallel:

- `clean-code-reviewer` reads the diff and evaluates each hunk against clean-code principles (naming, function size, single responsibility, DRY-within-hunk).
- `code-reuse-reviewer` surveys the surrounding codebase *first*, then audits the diff for missed reuse of existing constants, helpers, conventions, and patterns. Its mandatory output tables (inline-import audit vs `pyproject.toml`; function-pair overlap audit) catch things a diff-only reviewer cannot see.

Multiple types can apply (e.g., Snakemake + Python).

### Step 1.5: Fetch and determine base branch

Always fetch first so comparisons are against the true remote state, not a stale local branch:

```bash
git fetch origin
```

Determine the base branch from the PR (if one exists), otherwise default to `main` or `master`:

```bash
gh pr view --json baseRefName -q .baseRefName 2>/dev/null || echo "main"
```

Use `origin/<base>` (not local `<base>`) for all diffs below. This avoids false positives from a stale local main.

### Step 2: Identify Changed Files

```bash
git diff origin/<base>...HEAD --name-only
```

Focus review on changed files, not the entire codebase.

### Step 3: Check for Large Files and Cruft

Scan changed files for artifacts that typically shouldn't be committed:

**Suspicious file types** (flag for user review):
- HTML files from notebook execution (`.html` in output directories)
- Image files (`.png`, `.jpg`, `.jpeg`, `.gif`, `.svg`, `.pdf`) unless in expected locations like `docs/` or `assets/`
- Binary files and archives (`.zip`, `.tar`, `.gz`, `.pkl`, `.pickle`, `.npy`, `.npz`)
- Large data files (`.csv`, `.tsv`, `.parquet`) over 100KB
- Cache/build artifacts (`__pycache__/`, `.pyc`, `node_modules/`, `dist/`, `build/`)

**Acceptable** (don't flag):
- Small JSON result files (< 100KB)
- Config files (`.json`, `.yml`, `.yaml` in root or config directories)
- Test fixtures in `testdata/` or `tests/fixtures/`

**Check file sizes:**
```bash
git diff origin/<base>...HEAD --name-only | xargs -I{} sh -c 'test -f "{}" && stat -f "%z %N" "{}" 2>/dev/null || stat --format="%s %n" "{}" 2>/dev/null' | awk '$1 > 102400 {print}'
```

Flag anything suspicious for user confirmation before proceeding.

### Step 4: Run Agent Reviews (in parallel when possible)

**For Snakemake projects:**
- Launch `snakemake-pipeline-expert` agent to review workflow structure, rule organization, and best practices

**For all projects with code changes:**
- Launch `clean-code-reviewer` agent on modified source files (not tests)
- **In parallel**, launch `code-reuse-reviewer` agent on the same branch — it surveys the surrounding codebase first to catch missed reuse of existing patterns, constants, helpers, and conventions. The two agents have different mandates (clean-code principles vs. pattern adherence) and report independently.

### Step 4.5: Scientific Conclusion Skeptic (conditional)

**Always run this detection step.** Examine the PR title, body, and diff for signals that the PR reports a scientific conclusion or experimental result:

- Benchmark results or comparison tables
- Win/loss rates, accuracy numbers, or performance metrics
- Language like "outperforms", "worse than", "fails to", "never beats", "always prefers"
- Claims about method effectiveness, algorithmic limitations, or model comparisons
- Surprising or strong negative results (e.g., 0% or 100% rates)

**If any scientific conclusion is detected**, launch the `surprising-conclusion-skeptic` agent (Opus model) with this prompt:

```
Review PR #<number> on branch <branch> for scientific credibility.

Claim: <summarize the scientific conclusion from the PR title/body>

The PR diff is at: git diff origin/<base>...HEAD

Work through your full checklist: check for bugs, unfair comparisons,
implausible effect sizes, contradictions with established results, and
unverified upstream assumptions. Read the code that produced the result,
not just the description.

Report your findings with Claim / Confidence / Concerns / Recommended Checks / Verdict.
```

This step runs **in parallel** with the agent reviews in Step 4. Do not wait for it to complete before proceeding.

**If no scientific conclusion is detected**, skip this step and note "No scientific claims detected — skeptic review skipped" in the final report.

### Step 5: Run Automated Checks

Detect and run available quality tools:

| Tool Indicator | Check to Run |
|----------------|--------------|
| `pixi.toml` | Use `pixi run` prefix |
| `pyproject.toml` with ruff | `ruff check .` |
| `Makefile` with `check` target | `make check` |
| `build.zig` | `zig build test` and `zig fmt --check src/ tests/` |
| `go.mod` | `go test ./...` and `go vet ./...` |
| `Snakefile` | `snakemake --lint` |
| `tests/` directory | `pytest` (or project-specific test command) |

### Step 6: Test Audit

Run a thorough test quality review using an agent. Grepping alone is insufficient—tests need semantic analysis to detect:

- Placeholder tests (empty bodies, just `pass`, meaningless assertions)
- Inappropriate use of mocks (if project forbids mocking)
- Tests that don't actually test anything (`assert True`, trivial assertions)
- Tests with `pytest.skip` that should have real implementations
- Tests that silently catch exceptions instead of letting them propagate

**Launch a `clean-code-reviewer` agent specifically for test files:**

```
Review the test files in tests/ for test quality issues:
1. Placeholder tests (empty or pass-only test bodies)
2. Mock usage (if project constitution forbids mocks)
3. Trivial assertions (assert True, assert 1==1)
4. Unconditional pytest.skip() that should be real tests
5. Tests that catch and swallow exceptions
6. Tests without meaningful assertions

Focus on tests for changed code. Report specific file:line references.
```

Report findings with severity (blocking vs advisory).

### Step 7: Mathematical Documentation (for mathematical PRs)

**Skip this step** if the PR doesn't involve mathematical/statistical computation.

**Detect mathematical PR**: Check if the changed source files contain:
- Mathematical formulas in docstrings (Greek letters, summations, matrix notation)
- Scientific computation (matrix operations, probability distributions, statistical models)
- Algorithm implementations with mathematical foundations

**If mathematical**, check the PR for an existing mathematical specification comment:

1. **Check for existing math comment on the PR**: Use `gh pr view` and look for comments containing LaTeX code blocks or mathematical notation

2. **If no math comment exists**:
   - Back-translate the implementation directly into mathematical notation
   - Use LaTeX code blocks for formulas
   - Document the actual computation, not perceived intent
   - Post as a comment on the PR with 🤖 prefix
   - Include in report: "Posted mathematical specification to PR"

3. **If math comment already exists**:
   - Re-read the implementation carefully
   - Compare against the existing mathematical specification
   - Check for discrepancies (formula errors, missing steps, outdated notation)
   - If discrepancies found: post a correction comment and flag in report
   - If accurate: note "Mathematical specification verified" in report

4. **Literature verification for pre-existing formulas**:
   - If the implementation uses formulas from established literature (standard models, known algorithms, published methods), verify against the source paper
   - **Search local bip library first**: `bip search "method name"` or `bip search -a "Author"`
   - The reference paper should already be in our library; if not found locally, ask user before searching externally
   - Use pdf-navigator tools to read the actual paper and verify the implementation matches the published definition
   - Include the bip paper ID in the math comment to demonstrate correctness
   - Flag any discrepancies between implementation and literature as blocking issues

**Mathematical comment format**:
```markdown
🤖 Mathematical specification back-translated from the implementation in PR #XXX:

## [Section Name]

[Description of what this computes]

```latex
[Formula in LaTeX]
```

[Continue for each major mathematical component...]
```

### Step 8: Generate Report

Present a checklist summary:

```markdown
## Pre-Merge Quality Report

### Agent Reviews
- [ ] Snakemake review: [findings or ✓]
- [ ] Code review (clean-code): [findings or ✓]
- [ ] Code review (code-reuse): [findings or ✓]

### Scientific Conclusion Skeptic
- [ ] Skeptic review: [verdict or "No scientific claims detected — skipped"]

### Large Files / Cruft
- [x] No suspicious files found
- [ ] ⚠️ Flagged for review: `output/results.html` (notebook output?)

### Automated Checks
- [x] Linting: passed
- [x] Tests: 124 passed
- [ ] Format: 2 files need formatting

### Test Audit
- [x] No placeholder tests found
- [x] No forbidden mocks found

### Mathematical Documentation (if applicable)
- [x] Mathematical specification posted to PR
- [x] Existing specification verified against implementation
- [x] Literature references provided for pre-existing formulas

### Action Items
1. Fix formatting in `src/foo.py`
2. Address code review suggestion about X
```

## Error Handling

- **Not on a branch**: Warn that this should be run from a feature branch
- **No changes from main**: Note that branch appears to be up-to-date with main
- **Missing tools**: Skip checks for tools not installed, note in report
- **Agent failures**: Report the failure but continue with other checks

## Notes

- This skill coordinates multiple agents and tools; it may take a few minutes
- Agent reviews focus on changed files to keep feedback relevant
- The skill adapts to each project's tooling automatically
