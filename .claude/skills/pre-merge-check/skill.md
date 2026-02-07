---
name: pre-merge-check
description: Run comprehensive pre-merge quality checklist for current branch's PR
---

# /pre-merge-check

Run a comprehensive quality checklist before merging a PR. Automatically detects project type and runs appropriate checks.

## Usage

```
/pre-merge-check
```

## Workflow

### Step 0: Check for Project-Specific Checklist

First, read the project's `CLAUDE.md` and look for a "Pre-PR Quality Checklist" or "Pre-Merge Checklist" section. **If found, follow those steps exactly** instead of the generic workflow below.

### Step 1: Detect Project Type

Examine the repository to determine what checks apply:

| Indicator | Project Type | Agents to Use |
|-----------|--------------|---------------|
| `workflow/*.smk` or `Snakefile` | Snakemake pipeline | `snakemake-pipeline-expert` |
| `*.py` files in `src/` or project root | Python project | `clean-code-reviewer` |
| `go.mod` | Go project | `clean-code-reviewer` |
| `package.json` | Node.js project | `clean-code-reviewer` |

Multiple types can apply (e.g., Snakemake + Python).

### Step 2: Identify Changed Files

```bash
git diff main...HEAD --name-only
```

Focus review on changed files, not the entire codebase.

### Step 3: Run Agent Reviews (in parallel when possible)

**For Snakemake projects:**
- Launch `snakemake-pipeline-expert` agent to review workflow structure, rule organization, and best practices

**For all projects with code changes:**
- Launch `clean-code-reviewer` agent on modified source files (not tests)

### Step 4: Run Automated Checks

Detect and run available quality tools:

| Tool Indicator | Check to Run |
|----------------|--------------|
| `pixi.toml` | Use `pixi run` prefix |
| `pyproject.toml` with ruff | `ruff check .` |
| `Makefile` with `check` target | `make check` |
| `go.mod` | `go test ./...` and `go vet ./...` |
| `Snakefile` | `snakemake --lint` |
| `tests/` directory | `pytest` (or project-specific test command) |

### Step 5: Test Audit

Search for problematic test patterns:

```bash
# Look for placeholder tests
grep -r "pytest.skip\|pytest.mark.skip\|pass$" tests/ --include="*.py"

# Look for mock usage (if project forbids it)
grep -r "Mock()\|@patch\|MagicMock" tests/ --include="*.py"
```

Report any findings as warnings.

### Step 6: Generate Report

Present a checklist summary:

```markdown
## Pre-Merge Quality Report

### Agent Reviews
- [ ] Snakemake review: [findings or ✓]
- [ ] Code review: [findings or ✓]

### Automated Checks
- [x] Linting: passed
- [x] Tests: 124 passed
- [ ] Format: 2 files need formatting

### Test Audit
- [x] No placeholder tests found
- [x] No forbidden mocks found

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
