---
name: bip.audit
description: Sweep a repo for the decay modes Armin Ronacher calls out in agent-written codebases — duplicate symbols, monster files, zombie code, scaffolding duplication, test asymmetries — and diff against a previous run. Not a PR reviewer; a whole-repo weather check.
---

# /bip.audit

Periodic codebase-health sweep for an agent-written repository.

## Context

Agent-written codebases decay monotonically along a small set of axes that Armin Ronacher describes in ["Agentic Coding Recommendations"](https://lucumr.pocoo.org/2025/6/12/agentic-coding/) (2025-06-12):

- **Greppability.** Ronacher recommends "functions with clear, descriptive and longer than usual function names." Once a generic symbol (`run`, `compute`, `get`) is used in three files it will be used in thirty, and grep stops finding a specific definition.
- **Component extraction.** Ronacher recommends extracting components "when code becomes scattered across numerous files." Once a pattern (CLI parsing, progress reporting, error formatting) is copy-pasted to a third file, agents will copy it to every new file until someone factors it.
- **Monster files.** Algorithm cores grow; new helpers land in the same file rather than a sibling.
- **Zombie code.** `TODO`/`FIXME`, disabled tests, and commented-out blocks accrete.
- **Test asymmetry.** Some modules get heavy regression coverage; new small modules land without tests.

None of these are catastrophic on any single PR. Over months they compound. `/bip.audit` is a cheap, reproducible sweep that makes the compounding visible.

`/bip.audit` is NOT a PR reviewer (use `/bip.pr.review`), NOT a code reviewer for a diff (use `@clean-code-reviewer`), and NOT an architectural verdict. It produces signals; humans decide what to act on.

## Usage

```
/bip.audit                 # sweep current repo, compare to last run
/bip.audit --baseline      # record this run as the new baseline (first use, or after a refactor)
/bip.audit --no-compare    # run fresh, don't load previous state
/bip.audit --src <path>    # sweep a subdirectory (default: src/ if it exists, else repo root)
/bip.audit --full          # include expensive optional signals
```

## Workflow

### Step 1: Locate state

Per-repo state lives at `.bipartite/audit/` (add to `.gitignore` if not already covered). Structure:

```
.bipartite/audit/
  baseline.json       # the last committed-to-baseline run
  history/
    2026-04-21.json
    2026-04-28.json
    ...
  config.yml          # repo-specific thresholds (optional)
```

If `.bipartite/audit/baseline.json` does not exist, this is a first run — everything is a new metric, no deltas. Warn the user and suggest `--baseline`.

### Step 2: Detect languages and pick metric adapters

Run `file`/`find` over the source tree to decide which per-language metrics apply. Adapters this skill knows about:

| Language | Detection | Generic-name set | "Public function" pattern |
|---|---|---|---|
| Zig | `*.zig` present, `build.zig` at root | `run`, `compute`, `get`, `init`, `build`, `apply`, `step`, `score`, `update`, `handle`, `process` | `^pub fn <name>(` |
| Go | `go.mod` at root | `Run`, `Get`, `Process`, `Handle`, `Init`, `Build`, `New` | `^func [A-Z][A-Za-z]+(` |
| Python | `pyproject.toml` / `setup.py` at root | `run`, `get`, `process`, `handle`, `main`, `step` | `^def <name>\(` (module-level only) |
| Rust | `Cargo.toml` at root | `run`, `get`, `process`, `handle`, `new`, `init`, `build` | `^pub fn <name>(` |

If the adapter set is empty, print the detected files and stop — don't guess.

### Step 3: Collect signals

Each signal is a single `rg`/`wc`/`find` invocation. Nothing expensive; the whole sweep should finish in under five seconds on a 100kLOC repo. Do NOT read full files. Do NOT spawn LLM subagents for the data-collection phase.

Required signals (compute all for the detected language(s)):

1. **`pub fn <generic>` counts.** For each name in the generic-name set, `rg -c "^<pattern>" <src>` and sum across files. Report per-name count plus list of files.

2. **Top 20 duplicated public symbols.** `rg -oh "^<public-fn-pattern>" <src> | sort | uniq -c | sort -rn | head -20`. Report name, count, and 5-file sample.

3. **Monster files.** `find <src> -name '*.<ext>' -exec wc -l {} +` then threshold. Report files > 1500 LOC (language-configurable). Flag any new since baseline.

4. **TODO/FIXME/XXX/HACK/BUG.** `rg -c "TODO|FIXME|XXX|HACK|BUG"` per-file, sorted by density (markers per 100 LOC). Top 10 files plus total count.

5. **Disabled/skipped tests.** `rg "if \(false\)|SkipZigTest|@pytest\.mark\.skip|t\.Skip\(|\.skip\("` per-file. Total count.

6. **Commented-out code heuristic.** Per file: count lines matching `^\s*//\s*(if|while|for|fn|return|try|const|var|pub|test|def|func)\b` (adjust comment token per language). Top 10 files by count.

7. **CLI parser duplication.** Language-specific:
   - Zig: `rg -l "^pub const CliConfig"` ∪ `rg -l "^pub const CliError"` ∪ `rg -l "^pub fn parseArgs"`.
   - Go: `rg -l "cobra\.Command{"` and total `RunE:` function count.
   - Python: `rg -l "argparse.ArgumentParser|click.command"`.
   Report count of distinct CLI sites.

8. **Files larger by > 30% since baseline.** `wc -l` compared to `baseline.json`.

9. **New files since baseline without a sibling test.** Diff `find <src>` against baseline; for each new file, check for a corresponding `tests/test_<name>.*` or `<name>_test.<ext>` (per language). Flag any.

10. **Module coupling (optional, if supported by language).** For Zig: `rg "@import\(\"\.\./" <src> | awk -F/ '{print $1,$3}' | sort -u` → edges; flag any bidirectional edge.

Optional signals (add under `--full`; more expensive):

- Function LOC distribution (requires a language-aware LOC counter).
- Per-module test LOC ratio.
- Symbols defined in > 1 file that were defined in exactly 1 file in baseline.

### Step 4: Write the report

Produce one markdown file at `.bipartite/audit/history/<ISO-date>.md`. Format:

```markdown
# <repo-name> audit — <ISO-date>

## Signals

| metric                                | this run  | baseline  | Δ     | status |
|---------------------------------------|-----------|-----------|-------|--------|
| monster files (>1500 LOC)             | 8         | 8         |  0    | ok     |
| `pub fn run(` collisions              | 38        | 38        |  0    | ok     |
| top duplicate public symbol           | deinit×78 | deinit×78 |  0    | ok     |
| TODO/FIXME/XXX markers                | 9         | 8         | +1    | warn   |
| disabled tests                        | 0         | 0         |  0    | ok     |
| CLI parser sites                      | 11        | 11        |  0    | ok     |
| new files without tests               | 2         | 0         | +2    | warn   |

## Regressions

- `src/<path>` grew from 2844 → 3012 LOC (+5.9%); split candidate.
- `src/<path>` added this week; defines `pub fn run` — collides with 38 existing. Rename.
- `src/<dir>/` still has 0 test files.

## Wins

- CLI migration PR #1247 landed; pcp/cli.zig 198 → 82 LOC.
- No new TODO/FIXME in `src/prank/`.

## Top duplicates (full table, for drilling)

<table of top-20 duplicated symbols>

## Methodology

Reproduce any number with:
- `<the exact rg/wc command that produced it>`

Baseline: `.bipartite/audit/baseline.json` (captured <date>, commit <hash>).
Skill: `/bip.audit` (see skills/bip.audit/SKILL.md).
Principles: Ronacher 2025-06-12 (linked in SKILL.md).
```

Open the report. In tmux: `tmux display-popup -w 90% -h 90% -E -- less <path>`. Otherwise, print the path and let the user open it.

### Step 5: Save state

Write this run's signals as JSON to `.bipartite/audit/history/<ISO-date>.json`. If `--baseline`, also overwrite `.bipartite/audit/baseline.json`.

### Step 6: Regressions get a nudge, not a fix

If any metric regressed, list the specific regressions and offer three options:
1. Run `/bip.kaizen` on the biggest regression.
2. Draft an issue with `/bip.issue.check` describing the refactor to bring the metric back down.
3. Promote this run to baseline via `/bip.audit --baseline` (accept the new state).

Do not auto-fix. Do not auto-file. The skill's job ends at the report.

## Guidelines

- **Keep the data-collection phase dumb.** `rg` + `wc` + `find`, nothing else. No LLM calls during step 3. Subagents are only useful in step 6 if the user asks for a kaizen pass on a specific regression.
- **Thresholds are language- and repo-specific.** A monster-file cutoff of 1500 LOC is reasonable for Zig with dense DP kernels; a Python webapp might use 500. Store overrides in `.bipartite/audit/config.yml`.
- **Don't sweep `_ignore/`, `zig-cache/`, `zig-out/`, `target/`, `build/`, `.pixi/`, `node_modules/`, `experiments/`, or `testdata/`.** These inflate every count without contributing signal.
- **Signals, not verdicts.** The report never says "fix this." It says "this went up since baseline." A single-file 200-LOC growth may be a legitimate feature; three files that all grew 50% in one week is a pattern.
- **Make every number reproducible.** The "Methodology" section must show the exact `rg`/`wc` command for every cell. Agents working on a regression then know exactly what metric to move.
- **First run has no deltas.** Suggest `--baseline` and stop; don't pretend every metric is a regression.
- **Run on demand, not automatically.** Weekly cadence is a suggestion. Use `/bip.schedule` to wire a cron if you want it recurring.

## Why this shape

- **Not a PR reviewer**: `/bip.pr.review` and `@clean-code-reviewer` handle per-diff review. `/bip.audit` looks at whole-repo state, which they cannot.
- **Not an LLM-semantic pass**: tempting to run cosine similarity across function bodies to find near-duplicates. Expensive, noisy, and the useful signal ("three CLIs copy-pasted the same `while (i < args.len)` loop") is better caught by a cheap `rg -l "pub fn parseArgs"` count.
- **State in-repo, not in a DB**: `.bipartite/audit/` is committable (or gitignorable) at the user's discretion. Baselines live in git if kept; deltas show up in `git log`. No bip-side storage required.
- **Output is markdown, not a dashboard**: agents read markdown; `less` in a popup renders it; humans can paste it into Slack. No web UI to maintain.

## References

- Armin Ronacher, ["Agentic Coding Recommendations"](https://lucumr.pocoo.org/2025/6/12/agentic-coding/) (2025-06-12) — primary source for the greppability and extract-components principles this skill measures against. Key quote: "functions with clear, descriptive and longer than usual function names."
- Armin Ronacher, ["Agent Design Is Still Hard"](https://lucumr.pocoo.org/2025/11/21/agents-are-hard/) (2025-11-21) — observability and failure-isolation framing that informs why regressions get reported but never auto-fixed.
- Armin Ronacher, ["AI And The Ship of Theseus"](https://lucumr.pocoo.org/2026/3/5/theseus/) (2026-03-05) — context on how agent-authored rewrites drift.
- Precipitating case: phyz repo audit 2026-04-21 found 38 `pub fn run` collisions and ~5000 LOC of CLI-parser duplication across 10 modules. The initial version of this skill's signal list is exactly what that audit ran by hand.
- Sibling skills: `/bip.kaizen` (continuous improvement), `/bip.pr.review` (per-PR review), `/bip.issue.check` (issue drafting), `/bip.scout` (remote resource scan — similar name, orthogonal scope).
