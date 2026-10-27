---
name: bip-decay-audit
description: Sweep a repo for the decay modes Armin Ronacher calls out in agent-written codebases — duplicate symbols, monster files, zombie code, scaffolding duplication, test asymmetries — and diff against a previous run. Not a PR reviewer; a whole-repo weather check.
---

# /bip-decay-audit

Periodic codebase-health sweep for an agent-written repository.

## Context

The decay axes (greppability, component extraction, monster files, zombie code, test asymmetry) come from Armin Ronacher, ["Agentic Coding Recommendations"](https://lucumr.pocoo.org/2025/6/12/agentic-coding/).

This skill produces signals; the user decides what to act on.
For per-diff review use `/bip-pr-review` or `@clean-code-reviewer`.

## Usage

```
/bip-decay-audit                 # sweep current repo, compare to last run
/bip-decay-audit --baseline      # record this run as the new baseline (first use, or after a refactor)
/bip-decay-audit --no-compare    # run fresh, don't load previous state
/bip-decay-audit --src <path>    # sweep a subdirectory (default: src/ if it exists, else repo root)
/bip-decay-audit --full          # include expensive optional signals
```

## Workflow

### Step 1: Locate state

Per-repo state lives at `.bipartite/audit/` (add to `.gitignore` if not already covered).
Structure:

```
.bipartite/audit/
  baseline.json       # the last committed-to-baseline run
  history/
    2026-04-21.json
    2026-04-28.json
    ...
  config.yml          # repo-specific thresholds (optional)
```

If `.bipartite/audit/baseline.json` does not exist, this is a first run: collect signals and write the report with no deltas, and suggest `--baseline` unless it was passed.

### Step 2: Detect languages and pick metric adapters

Detect languages from the source tree and pick adapters:

| Language | Detection | Generic-name set | "Public function" pattern |
|---|---|---|---|
| Zig | `*.zig` present, `build.zig` at root | `run`, `compute`, `get`, `init`, `build`, `apply`, `step`, `score`, `update`, `handle`, `process` | `^pub fn <name>(` |
| Go | `go.mod` at root | `Run`, `Get`, `Process`, `Handle`, `Init`, `Build`, `New` | `^func [A-Z][A-Za-z]+(` |
| Python | `pyproject.toml` / `setup.py` at root | `run`, `get`, `process`, `handle`, `main`, `step` | `^def <name>\(` (module-level only) |
| Rust | `Cargo.toml` at root | `run`, `get`, `process`, `handle`, `new`, `init`, `build` | `^pub fn <name>(` |

If no adapter matches, print the detected files and stop.

### Step 3: Collect signals

Each signal is one `rg`/`wc`/`find` invocation; do not read full files or spawn subagents.
Skip `_ignore/`, `zig-cache/`, `zig-out/`, `target/`, `build/`, `.pixi/`, `node_modules/`, `experiments/`, and `testdata/`.
Thresholds (e.g. the 1500-LOC monster cutoff; a Python webapp might use 500) are overridden in `.bipartite/audit/config.yml`.

Required signals, for each detected language:

1. **`pub fn <generic>` counts.**
   For each name in the generic-name set, `rg -c "^<pattern>" <src>` and sum across files.
   Report per-name count plus list of files.

2. **Top 20 duplicated public symbols.**
   `rg -oh "^<public-fn-pattern>" <src> | sort | uniq -c | sort -rn | head -20`.
   Report name, count, and 5-file sample.

3. **Monster files.**
   `find <src> -name '*.<ext>' -exec wc -l {} +` then threshold.
   Report files > 1500 LOC; flag any new since baseline.

4. **TODO/FIXME/XXX/HACK/BUG.**
   `rg -c "TODO|FIXME|XXX|HACK|BUG"` per-file, sorted by density (markers per 100 LOC).
   Top 10 files plus total count.

5. **Disabled/skipped tests.**
   `rg "if \(false\)|SkipZigTest|@pytest\.mark\.skip|t\.Skip\(|\.skip\("` per-file.
   Total count.

6. **Commented-out code heuristic.**
   Per file: count lines matching `^\s*//\s*(if|while|for|fn|return|try|const|var|pub|test|def|func)\b` (adjust comment token per language).
   Top 10 files by count.

7. **CLI parser duplication.**
   Report the count of distinct CLI sites:
   - Zig: `rg -l "^pub const CliConfig"` ∪ `rg -l "^pub const CliError"` ∪ `rg -l "^pub fn parseArgs"`.
   - Go: `rg -l "cobra\.Command{"` and total `RunE:` function count.
   - Python: `rg -l "argparse.ArgumentParser|click.command"`.

8. **Files larger by > 30% since baseline.**
   `wc -l` compared to `baseline.json`.

9. **New files since baseline without a sibling test.**
   Diff `find <src>` against baseline; for each new file, check for a corresponding `tests/test_<name>.*` or `<name>_test.<ext>` (per language); flag any without one.

10. **Module coupling (optional, if supported by language).**
    For Zig: `rg "@import\(\"\.\./" <src> | awk -F/ '{print $1,$3}' | sort -u` → edges; flag any bidirectional edge.

Under `--full`, also collect: function LOC distribution, per-module test LOC ratio, and symbols defined in > 1 file that were defined in exactly 1 file in baseline.

### Step 4: Write the report

Produce one markdown file at `.bipartite/audit/history/<ISO-date>.md`.
Format:

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
Skill: `/bip-decay-audit` (see skills/bip-decay-audit/SKILL.md).
Principles: Ronacher 2025-06-12 (linked in SKILL.md).
```

The Methodology section shows the exact `rg`/`wc` command for every cell, so an agent working a regression knows which metric to move.
The report says what went up since baseline, never "fix this".

Open the report.
In tmux: `tmux display-popup -w 90% -h 90% -E -- less <path>`.
Otherwise, print the path and let the user open it.

### Step 5: Save state

Write this run's signals as JSON to `.bipartite/audit/history/<ISO-date>.json`.
If `--baseline`, also overwrite `.bipartite/audit/baseline.json`.

### Step 6: Regressions get a nudge, not a fix

If any metric regressed, list the specific regressions and offer three options:
1. Run `/bip-kaizen` on the biggest regression.
2. Draft an issue with `/bip-issue-check` describing the refactor to bring the metric back down.
3. Promote this run to baseline via `/bip-decay-audit --baseline` (accept the new state).

Do not auto-fix or auto-file; the skill's job ends at the report.

## Guidelines

- Before calling something zombie, read its docstring: scaffolding can be load-bearing for a tracked future issue, and a zero import count is a question, not a verdict.
- Weigh coordinated growth over single-file growth: one file growing 200 LOC may be a feature; three files each growing 50% in a week is a pattern.
- Before recommending a split, apply the deletion test: many real importers means the module earns its keep; zero or one means a shallow module.
- Present numbered candidates, each with the smell, a proposed fix, and a rough size, and let the user grill them before filing.
  Don't pre-design; one well-anchored issue beats several threshold-driven ones, since a size threshold surfaces the architecture question rather than answering it.
- Run on demand; use `/schedule` for a recurring run.
