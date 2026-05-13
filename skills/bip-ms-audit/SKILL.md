---
name: bip-ms-audit
description: Audit a manuscript against its implementation(s) — read the paper and code in parallel, find places where formulas, algorithms, dimensions, or complexity claims diverge from what the code actually does. Use the skeptic agent to confirm surprising findings before reporting.
allowed-tools: Agent, Bash, Read, Glob, Grep, Edit, Write
---

# /bip-ms-audit

Audit a manuscript against its implementation(s). Find places where the **paper claims one thing** and the **code does another** — wrong tensor dimensions, formulas that disagree between paper and code, complexity claims that don't match the algorithm, parameter values out of step with config files, etc.

This is **not** a paper proofreader (`@scientific-tex-editor`, `@tex-grammar-checker`), **not** a code reviewer for a diff (`@clean-code-reviewer`), and **not** a PR check (`/bip-pr-review`). It is the cross-cutting check that nobody runs because each side feels like someone else's job — yet it's where the worst bugs hide, because each side looks internally consistent.

Run from a **TeX repository** with `.ms-config.json` configured (see `/bip-ms`). The skill reads the paper, reads the tracked code repo(s), and reports mismatches.

## Usage

```
/bip-ms-audit                       # full audit: scan formulas/algorithms in main.tex against tracked_repos
/bip-ms-audit <section>             # audit one section, e.g. "Methods" or "Step 1: Tree Traversal"
/bip-ms-audit --formula <label>     # audit a specific labeled equation/figure
/bip-ms-audit --paper <file.tex>    # explicit paper file (otherwise from .ms-config.json)
/bip-ms-audit --no-skeptic          # skip the skeptic confirmation step (faster, less reliable)
```

## Core principle

**Trust nothing. Read both sides.** A surprising mismatch ("the paper says $f^u_b$ but the code uses $f^d_b$") is exactly the kind of finding that is most often either a real bug or a misreading on one side. Either outcome is valuable, but never report it without a skeptic agent independently verifying.

The two failure modes are symmetric and both cost the user trust:

- **False positive**: report a "bug" that isn't, sending the user on a wild-goose chase. Avoid by running `@surprising-conclusion-skeptic` before propagating any non-trivial finding.
- **False negative**: skim the formulas, miss the bug. Avoid by reading the actual implementation file with `Read`, not just grepping for keywords.

## Workflow

### Step 1: Load config and identify scope

```bash
cat .ms-config.json
```

Determine:
- **Paper file**: from `.manuscript` field (e.g., `main.tex`).
- **Tracked code repos**: from `.tracked_repos[].local_path` — these are the implementations to audit against.
- **Audit scope**: from the user's argument. Default to the full Methods/Algorithms section of the paper.

If `.ms-config.json` is missing, ask the user for the paper path and code repo path(s) and offer to create the config (see `/bip-ms`).

### Step 2: Partition the scope and fan out

The primary partitions the audit scope into independent chunks, then
dispatches one `general-purpose` subagent per chunk **in parallel** —
single message, multiple `Agent` tool calls. Follow the dispatch
pattern in `SUBAGENT-SCAN.md` (bipartite repo root).

Partitioning rules:
- **Whole paper** (default): one subagent per top-level Methods /
  Algorithms section.
- **One section** (`/bip-ms-audit Methods`): one subagent per
  subsection, or per claim cluster (formula + its surrounding text)
  if subsections are missing.
- **One formula or label** (`/bip-ms-audit --formula eq:loss`): one
  subagent.

Brief for each audit subagent (the **line-by-line investigation**
framing is mandatory — this is what separates an audit from a skim):

> **Line-by-line investigation** of paper section `<section>`
> against tracked code repo(s) `<local_path(s)>`. This is not a
> scan. For every checkable claim in scope, you must read the
> implementation file in full with the `Read` tool — not a grep
> excerpt — and cite the specific paper line and code line that
> back your verdict. A verdict without a `file:line` citation on
> both sides will be rejected and re-dispatched.
>
> Tasks:
> 1. Read the paper section with `Read`. List every checkable,
>    falsifiable claim: tensor/array dimensions, formulas
>    (`align`/`equation`/display math), algorithm steps,
>    hyperparameter values, counts/complexity, variable bindings,
>    loss/metric definitions, data pipeline steps, theorem
>    statements. Skip subjective claims ("our results suggest…").
>    Record paper `file:line` for each.
> 2. For each claim, locate the code-side counterpart. Use `rg` to
>    find files, but then `Read` the file. Common keyword maps:
>    tensor dims → `torch.zeros`/`np.zeros`/`torch.full`;
>    formulas → `forward`/`loss`/aggregation; hyperparameters →
>    `config.yaml`/`Snakefile`/CLI defaults; loss/mask → training
>    loop; data filtering → preprocessing pipeline/snakemake rules.
>    If a claim has multiple implementations (e.g., vectorized
>    batch path + tree-iteration reference path), check **both** —
>    that's a frequent source of divergence.
> 3. For each claim, classify with a verdict:
>    - `MATCH` — paper and code agree
>    - `MISMATCH` — disagree on a substantive, falsifiable point
>    - `AMBIGUOUS` — paper is unclear/underspecified; code makes a
>      specific choice
>    - `MULTIPLE IMPLS DISAGREE` — two code paths disagree with
>      each other (and at most one matches the paper)
>    - `STALE PAPER` — paper describes an earlier code version
>    - `STALE CODE` — paper describes a fix code hasn't picked up
>
> **MATCH is the default.** Most claims are correct. If your
> report is mostly `MISMATCH`, you are misreading the code — stop
> and re-check the worst offenders before returning.
>
> Return under 500 words, structured:
> - `findings`: one entry per non-`MATCH` claim with:
>   - paper quote + `file:line`
>   - code quote + `file:line` (use Read, not grep excerpts)
>   - verdict
>   - one-sentence rationale (why they disagree)
> - `matches`: count of `MATCH` verdicts (no quotes needed)
> - `surprises`: claims you couldn't certify a `file:line` for,
>   variable names that mean different things in different places,
>   `RECOMMEND DEEPER LOOK` flags

If a subagent returns a finding without both `file:line` citations,
re-dispatch with a narrower brief covering just that finding.

### Step 3: Primary consolidates findings

The primary collects all subagent reports and assembles a single
working list of non-`MATCH` findings, sorted by verdict severity
(`MULTIPLE IMPLS DISAGREE` and `MISMATCH` first). No prose pasted
verbatim from subagents — quote only the paper and code line excerpts
they cited.

If any subagent's report has many findings (say >5 in a 10-claim
section), treat that as the "everything looks like a mismatch"
failure mode and re-dispatch the subagent with explicit instruction
to verify each claim by reading the full implementation file before
classifying.

### Step 4: Skeptic confirmation for non-trivial findings

For any finding that is **not** a `MATCH`, spawn `@surprising-conclusion-skeptic` to independently verify before reporting it. Skip this step only if the user passed `--no-skeptic`.

Brief the skeptic with:
- The exact paper claim (with line number)
- The exact code location (file:line)
- The mismatch you believe you found
- An explicit list of simpler explanations you want them to rule out:
  - "Maybe I misread the paper."
  - "Maybe I misread the code."
  - "Maybe the two code paths are actually equivalent because of [specific construction]."
  - "Maybe one of them is dead code."
  - "Maybe the variable names mean something different in this context than I think."

The skeptic's job is to **try to refute** the finding. If they confirm it survives scrutiny, escalate it. If they refute it, drop it from the report. If they say "partially," report with the qualification.

This step is the difference between a useful audit and one that cries wolf. Do not skip it.

### Step 5: Write the report

Produce a markdown report in the manuscript repo as `MS-AUDIT-<ISO-date>.md`. Format:

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

For each finding include: paper quote + line, code quote + line, skeptic verdict, suggested action. Group by verdict, with `MULTIPLE IMPLS DISAGREE` and `MISMATCH` first.

### Step 6: Hand off

Show the report path. Offer:
1. Open the report (`zed <path>` or `tmux display-popup -E -- less <path>`).
2. For each `MISMATCH` finding, ask whether to draft a paper edit, file a code issue (via `/bip-issue-check` → `/bip-issue-file`), or both.
3. Do not auto-edit the paper or auto-file issues. The user decides.

## Guidelines

- **Read the actual files.** Every reported finding must have a `Read` (not just a `grep`) backing it on both sides.
- **Default to MATCH.** Most claims are correct. A report with 30 mismatches in a 40-claim audit is almost certainly the auditor misreading.
- **Always run the skeptic on non-MATCH findings** unless the user explicitly waives it. Surprising bug reports cost user trust when wrong; the skeptic round trip is cheap insurance.
- **Cite both sides with line numbers.** Future-you (or the authors) need to be able to jump straight to both the paper line and the code line. `main.tex:296` and `dpvt/wrapper.py:228` — never just "around line 200ish."
- **Don't fix anything.** This skill produces a report. Edits to the paper and changes to the code happen in separate, explicit steps.
- **Two implementations of the same thing are a high-yield search target.** When a repo has both a vectorized batch path and a tree-iteration reference path of the same algorithm, they are an excellent place to find bugs even when neither side disagrees with the paper individually.
- **Hyperparameter drift is real.** Paper says lr=$5 \times 10^{-5}$; config says lr=$1 \times 10^{-4}$ because someone tuned and forgot to update the paper. Cheap to check, often wrong.
- **Beware notation that seems internally consistent.** A formula using $v$ for a node and $v$ for a function is a warning sign; check that the paper's $v$ in storage matches the code's `v` in storage and not just both being plausible.

## When to use this vs siblings

| Need | Skill |
|---|---|
| Cold-start a manuscript session | `/bip-ms` |
| Mid-session check for new results | `/bip-ms-poll` |
| Persist manuscript session state | `/bip-ms-tuckin` |
| Repo-wide code-health weather check | `/bip-decay-audit` |
| Per-PR review against guidelines | `/bip-pr-review` |
| Fact-check a reviewer's comment against code | `/bip-comment-check` |
| **Cross-check paper claims against code** | **this skill** |

## Why this shape

- **Reads paper as source of truth for *intent*, code as source of truth for *behavior*.** Either can be wrong; the audit's job is to surface the gap.
- **Skeptic step is non-optional by default.** Without it the skill is a false-positive machine; with it the skill earns its place. The skeptic exists precisely for "the paper says X but the code says Y" claims, and this skill is its highest-yield caller.
- **Output is a markdown report, not auto-applied edits.** A bad finding propagated as a paper edit is much more expensive than the same finding in a markdown report. The user reads, decides, and dispatches.
- **No state directory.** Unlike `/bip-decay-audit`, this skill does not maintain a per-repo baseline — papers and code change together, so a "regression since last audit" framing isn't useful here. Re-run as needed; the report is the artifact.

## References

- Sibling skills: `/bip-ms`, `/bip-ms-poll`, `/bip-ms-tuckin`, `/bip-decay-audit`, `/bip-comment-check`, `/bip-pr-review`.
- Subagents used: `@surprising-conclusion-skeptic` (mandatory for non-MATCH findings).
- Optional subagents: `@scientific-tex-editor` for follow-up paper edits; `@clean-code-reviewer` for follow-up code review on the code-side of a mismatch.
- Precipitating case: `dpvt` audit 2026-05-06 — paper said tensor was `(n-3) × N × 4` but code allocated `(2n-2, N, 4)`; vectorized RNN path used both downward features for the downward pass while paper and tree-based path used downward+upward. Both bugs sat in `main.tex` and `models.py` for months because no one read them in parallel.
