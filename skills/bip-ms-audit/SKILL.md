---
name: bip-ms-audit
description: Audit a manuscript against its implementation(s) — read the paper and code in parallel, find places where formulas, algorithms, dimensions, or complexity claims diverge from what the code actually does. Use the skeptic agent to confirm surprising findings before reporting.
allowed-tools: Agent, Bash, Read, Glob, Grep, Edit, Write
---

# /bip-ms-audit

Find places where the paper claims one thing and the code does another: tensor dimensions, formulas, complexity claims, parameter values against config files.
Not a proofreader (`@scientific-tex-editor`), a diff reviewer (`@clean-code-reviewer`), or a PR check (`/bip-pr-review`).

Run from a TeX repository with `.ms-config.json` (see `/bip-ms`).
The output is a report: change the paper or code, or file an issue, only when the user says to.

## Usage

```
/bip-ms-audit                       # full audit: scan formulas/algorithms in main.tex against tracked_repos
/bip-ms-audit <section>             # audit one section, e.g. "Methods" or "Step 1: Tree Traversal"
/bip-ms-audit --formula <label>     # audit a specific labeled equation/figure
/bip-ms-audit --paper <file.tex>    # explicit paper file (otherwise from .ms-config.json)
/bip-ms-audit --no-skeptic          # skip the skeptic confirmation step (faster, less reliable)
```

## Workflow

### Step 1: Load config and identify scope

```bash
cat .ms-config.json
```

- Paper file: `.manuscript` (e.g., `main.tex`).
- Code to audit against: `.tracked_repos[].local_path`.
- Scope: the user's argument; default is the full Methods/Algorithms section.

If `.ms-config.json` is missing, ask for the paper and code paths and offer to create the config (see `/bip-ms`).

### Step 1b: Provenance check (when `provenance.yaml` sits beside the paper)

```bash
bip prov check --human --main <paper-file>
```

It runs every mechanical check against git objects of the ledger's repos and exits nonzero on any `error`.
The audit scope is then only its `error` and `review` findings: each names a tag id, its `file:line` and the entry's `scope`.
Give each subagent the sentence, the entry, and its `scope`, and have it judge whether the sentence says what the source shows for that population and stage.
`info` findings (`unsourced` entries, unused entries, runs whose producing commit is HEAD at collection, code claims that no longer hold at origin/main) go to the report without a subagent.
If nothing is at `error` or `review`, skip to the report.

The check cannot see a neighbouring-sentence change that alters a number's meaning, a flag overridden elsewhere (`params/*.yml` over `nextflow.config`), or a superseded result file still present.
A periodic full audit without the scope limit covers those.

### Step 2: Partition the scope and fan out

Partition the scope and dispatch one `general-purpose` subagent per chunk in parallel (one message, several `Agent` calls), following `SUBAGENT-SCAN.md` (bipartite repo root).

- Whole paper: one subagent per top-level Methods / Algorithms section.
- One section: one per subsection, or per claim cluster (formula plus its text) if there are no subsections.
- One formula or label: one subagent.

Brief for each subagent:

> **Line-by-line investigation** of paper section `<section>` against tracked code repo(s) `<local_path(s)>`.
> This is not a scan.
> For every checkable claim in scope, read the implementation file in full with `Read` — not a grep excerpt — and cite the paper line and code line behind your verdict.
> A verdict without a `file:line` citation on both sides will be rejected and re-dispatched.
>
> 1. Read the paper section. List every falsifiable claim with its `file:line`: dimensions, formulas, algorithm steps, hyperparameter values, counts/complexity, variable bindings, loss/metric definitions, data pipeline steps, theorem statements. Skip subjective claims ("our results suggest…").
> 2. Locate each claim's code counterpart: `rg` to find the file, then `Read` it. Common keyword maps: tensor dims → `torch.zeros`/`np.zeros`/`torch.full`; formulas → `forward`/`loss`/aggregation; hyperparameters → `config.yaml`/`Snakefile`/CLI defaults; loss/mask → training loop; data filtering → preprocessing pipeline/snakemake rules. If a claim has several implementations (e.g., a vectorized batch path and a tree-iteration reference path), check each; they often diverge from each other.
> 3. Give each claim a verdict:
>    - `MATCH` — paper and code agree
>    - `MISMATCH` — disagree on a substantive, falsifiable point
>    - `AMBIGUOUS` — paper is unclear/underspecified; code makes a specific choice
>    - `MULTIPLE IMPLS DISAGREE` — two code paths disagree with each other (and at most one matches the paper)
>    - `STALE PAPER` — paper describes an earlier code version
>    - `STALE CODE` — paper describes a fix code hasn't picked up
>
> `MATCH` is the default. If your report is mostly `MISMATCH`, you are misreading the code; re-check the worst before returning.
> Also check that a symbol means the same thing on both sides, not only that each side is plausible.
>
> Return under 500 words:
> - `findings`: per non-`MATCH` claim, paper quote + `file:line`, code quote + `file:line`, verdict, one-sentence rationale
> - `matches`: count of `MATCH` verdicts
> - `surprises`: claims you couldn't cite a `file:line` for, names that mean different things in different places, `RECOMMEND DEEPER LOOK` flags

Re-dispatch any finding missing either citation with a brief narrowed to that finding.

### Step 3: Primary consolidates findings

Merge the non-`MATCH` findings into one list, `MULTIPLE IMPLS DISAGREE` and `MISMATCH` first.
Quote only the paper and code excerpts the subagents cited.
If a subagent reports many findings (say >5 in a 10-claim section), re-dispatch it with the instruction to read the full implementation file before classifying each claim.

### Step 4: Skeptic confirmation

Unless the user passed `--no-skeptic`, spawn `@surprising-conclusion-skeptic` on every non-`MATCH` finding to try to refute it.
Give it the paper claim with line number, the code `file:line`, the mismatch, and the simpler explanations to rule out:
- "Maybe I misread the paper."
- "Maybe I misread the code."
- "Maybe the two code paths are actually equivalent because of [specific construction]."
- "Maybe one of them is dead code."
- "Maybe the variable names mean something different in this context than I think."

Confirmed: report it. Refuted: drop it. Partial: report it with the qualification.

### Step 5: Write the report

Write `MS-AUDIT-<ISO-date>.md` in the manuscript repo, grouped by verdict with `MULTIPLE IMPLS DISAGREE` and `MISMATCH` first:

```markdown
# Manuscript audit — <paper title> — <ISO date>

Paper: `<paper-file>` @ <git-sha-short>
Code:  `<repo>/<commit-sha-short>`

## Summary

| Verdict | Count |
|---|---|
| MATCH                | N |
| MISMATCH             | N |
| MULTIPLE IMPLS DISAGREE | N |
| AMBIGUOUS            | N |
| STALE PAPER / CODE   | N |

## Findings

### 1. Tensor dimension `(n-3) × N × 4` — MISMATCH

**Paper** (`main.tex:296`): "creating a tensor of dimension $(n-3) \times N \times 4$"

**Code** (`dpvt/wrapper.py:228-234`):
\`\`\`python
mutations = torch.full(
    (len(trees), max_n_nodes, max_n_sites, 4), -1, ...)
\`\`\`
Allocates `max_n_nodes` ≈ 2n-2 entries, indexed by every non-root node — paper's `(n-3)` is too small.

**Skeptic verdict**: confirmed (cited file:line, ruled out the "pendant edges are always zero" alternative).

**Suggested action**: change paper to `(2n-3) × N × 4` and clarify "indexed by non-root nodes."

### 2. Vectorized vs tree-based RNN downward pass — MULTIPLE IMPLS DISAGREE

[same format]

...
```

### Step 6: Hand off

Show the report path, then:
1. Offer to open it (`zed <path>` or `tmux display-popup -E -- less <path>`).
2. For each `MISMATCH`, ask whether to draft a paper edit, file a code issue (`/bip-issue-check` → `/bip-issue-file`), or both.
3. With a `provenance.yaml`, once every finding is resolved, set its `audited_through` to the paper commit the audit read and commit it; the next check then marks only sentences changed since.
