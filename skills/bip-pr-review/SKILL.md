---
name: bip-pr-review
description: Run the comprehensive pre-merge quality checklist on the current branch's PR, your own or someone else's checked out for review. The user's choice for every PR review, in place of /code-review.
---

# /bip-pr-review

Run the pre-merge quality checklist on the current branch's PR.

## Usage

```
/bip-pr-review
```

To review someone else's PR, run it in a claimed fleet clone after `gh pr checkout <N>`, and push nothing there. Whether to post, the Step 7 math comment included, follows `docs/guides/roles.md` ("Posting to GitHub"): another human's scientific PR gets the findings reported to the user, not posted.
Don't report textual merge or rebase conflicts with sibling PRs; whoever lands second rebases. Report a collision only when it is semantic: two PRs that give one symbol different names, or make contradictory claims.
In an `r<N>-<slot>` review window, once the review is complete (posted, by `/bip-review-post` or otherwise, or the user says it's done), message the conductor named in `$CLONE_ROOT/.conductor-session` to preserve `_ignore/REVIEW-p<N>.md`, close this window, and reclaim `<slot>`, unless the user has said to keep the window.

## Workflow

### Step 0: Check for Project-Specific Checklist

Look for `PRE-MERGE-CHECKLIST.md` in the repo root, else a "Pre-PR Quality Checklist" or "Pre-Merge Checklist" section in the project's `CLAUDE.md` or `AGENTS.md`. If one exists, follow it exactly instead of the generic workflow below.

### Step 1: Detect Project Type

| Indicator | Project Type | Agents to Use |
|-----------|--------------|---------------|
| `workflow/*.smk` or `Snakefile` | Snakemake pipeline | `snakemake-pipeline-expert` |
| `*.py` files in `src/` or project root | Python project | `clean-code-reviewer` + `code-reuse-reviewer` |
| `build.zig` or `*.zig` files in `src/` | Zig project | `zig-code-reviewer` |
| `go.mod` | Go project | `clean-code-reviewer` + `code-reuse-reviewer` |
| `package.json` | Node.js project | `clean-code-reviewer` + `code-reuse-reviewer` |

Several types can apply. `clean-code-reviewer` judges each diff hunk; `code-reuse-reviewer` surveys the codebase first, then finds missed reuse of existing constants, helpers, and conventions.

### Step 1.5: Fetch and determine base branch

```bash
git fetch origin
```

```bash
gh pr view --json baseRefName -q .baseRefName 2>/dev/null || echo "main"
```

Diff against `origin/<base>`, not local `<base>`, which may be stale.

### Step 2: Identify Changed Files

```bash
git diff origin/<base>...HEAD --name-only
```

### Step 3: Check for Large Files and Cruft

Flag for user confirmation before proceeding:
- HTML from notebook execution (`.html` in output directories)
- Images (`.png`, `.jpg`, `.jpeg`, `.gif`, `.svg`, `.pdf`) outside expected places like `docs/` or `assets/`
- Binaries and archives (`.zip`, `.tar`, `.gz`, `.pkl`, `.pickle`, `.npy`, `.npz`)
- Data files (`.csv`, `.tsv`, `.parquet`) over 100KB
- Cache/build artifacts (`__pycache__/`, `.pyc`, `node_modules/`, `dist/`, `build/`)

Don't flag small JSON results (< 100KB), config files (`.json`, `.yml`, `.yaml` in root or config directories), or fixtures in `testdata/` or `tests/fixtures/`.

```bash
git diff origin/<base>...HEAD --name-only | xargs -I{} sh -c 'test -f "{}" && stat -f "%z %N" "{}" 2>/dev/null || stat --format="%s %n" "{}" 2>/dev/null' | awk '$1 > 102400 {print}'
```

### Step 4: Run Agent Reviews (in parallel when possible)

- Snakemake: launch `snakemake-pipeline-expert` on workflow structure and rule organization.
- All code changes: launch `clean-code-reviewer` on modified source files (not tests) and, in parallel, `code-reuse-reviewer` on the same branch. They report independently.
- Point both at experiment and script code too, and ask what the PR could drop: modes, arms, and scripts its final result no longer uses (not controls or arms its README reports, including null ones), and helpers copied rather than imported.

### Step 4.5: Scientific Conclusion Skeptic (conditional)

Always check the PR title, body, and diff for a scientific conclusion: benchmark or comparison tables, win/loss rates or accuracy metrics, language like "outperforms", "fails to", "never beats", claims about method effectiveness or limitations, or surprising or strong negative results (0% or 100% rates).

If one is found, launch the `surprising-conclusion-skeptic` agent (Opus model) in parallel with Step 4, without waiting on it:

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

If none is found, note "No scientific claims detected — skeptic review skipped" in the report.

### Step 4.6: Cold read of a rendered notebook (conditional)

When the diff adds or changes a committed notebook render, launch one `general-purpose` agent per render in parallel with Step 4. Key on the changed `.html` files, since a render need not share its notebook's stem. Give the agent only the render's path and the intended audience if the issue names one: no issue, PR, or thread, because every session that wrote or reviewed the PR already knows what the notebook meant to say.

```
Read <path to .html> as its intended reader (<audience, if named>) would, knowing nothing else.
1. In two sentences: what question does it answer, and what is the answer?
   Say whether you could state both from the first screen.
2. List every term used before it is defined, every forward reference,
   and every result whose statement is interrupted by caveats.
```

A reader who cannot state the answer from the first screen is the finding. If the diff touches a notebook but commits no render, note "no render in diff — skipped".

### Step 5: Run Automated Checks

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

Grepping is not enough; launch a `clean-code-reviewer` agent on the test files:

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

Report findings as blocking or advisory.

### Step 7: Mathematical Documentation (for mathematical PRs)

Run this step when the changed source has formulas in docstrings, scientific computation (matrix operations, distributions, statistical models), or algorithms with a mathematical foundation.

1. Look in `gh pr view` comments for an existing math specification (LaTeX blocks or math notation).
2. If none exists, back-translate the implementation into LaTeX, documenting the actual computation rather than the intent. Post it as a PR comment with the 🤖 prefix and report "Posted mathematical specification to PR".
3. If one exists, compare it against the implementation. Check for formula errors, missing steps, and outdated notation; post a correction comment and flag any discrepancy; otherwise report "Mathematical specification verified".
4. For formulas from established literature, verify against the source paper. Search the local library first (`bip search "method name"` or `bip search -a "Author"`); if the paper is not there, ask the user before searching externally. Read the paper with the pdf-navigator tools, put the bip paper ID in the math comment, and flag any mismatch with the literature as blocking.

Comment format:
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

```markdown
## Pre-Merge Quality Report

### Agent Reviews
- [ ] Snakemake review: [findings or ✓]
- [ ] Code review (clean-code): [findings or ✓]
- [ ] Code review (code-reuse): [findings or ✓]

### Scientific Conclusion Skeptic
- [ ] Skeptic review: [verdict or "No scientific claims detected — skipped"]

### Notebook Cold Read
- [ ] Cold read: [stated question and answer, plus findings, or "no render in diff — skipped"]

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

Not on a feature branch: warn. No changes from base: say so. Missing tool: skip that check and note it. Agent failure: report it and continue.
