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

### Step 2: Extract checkable claims from the paper

Read the paper section(s) in scope. Build a list of **checkable, falsifiable claims**. Focus on:

- **Tensor / array dimensions**: "tensor of dimension $X \times Y \times Z$"
- **Formulas**: equations in `align`, `equation`, or display math; especially RNN/aggregation/loss expressions
- **Algorithm steps**: pseudocode in `algorithm` blocks; described traversal orders
- **Hyperparameter values**: "we set learning rate to $5 \times 10^{-5}$", "batch size $4$"
- **Counts / complexity**: "$n - 3$ internal edges", "$O(n \log n)$ time", "200 trees per alignment"
- **Variable bindings**: "$v$ is the node bounding the current edge on its root side"
- **Loss / metric definitions**: "we use binary cross entropy", "mask edges incident to leaves"
- **Data pipeline steps**: "we remove sites containing gaps", "$80\%$ training / $20\%$ testing split"
- **Theorem / proposition statements**: definitions and the variables they reference

Skip subjective claims that cannot be verified against code ("our results suggest...", "this is promising...").

For each claim, record:
- The exact paper line number(s)
- A one-sentence summary of what the paper asserts
- The code-side concept to look for (function name, file, hyperparameter key, etc.)

### Step 3: Locate the code-side counterpart

For each claim, find the corresponding implementation:

```bash
LOCAL_PATH=<expanded local_path>
# Search by likely keyword:
rg -n "<keyword>" "$LOCAL_PATH" --type py --type zig --type go
# Look in obvious places:
ls "$LOCAL_PATH"/<package>/
```

Common keyword maps:

| Paper claim | Where to look |
|---|---|
| Tensor dimension | `*.py` allocation sites — `torch.zeros`, `np.zeros`, `torch.full` |
| Formula | model `forward`, `loss`, or aggregation function |
| Algorithm step | function with the matching name; or pseudocode-style helper |
| Hyperparameter | `config.yaml`, `Snakefile`, CLI defaults, `__init__` signature |
| Loss / mask | training loop, `loss_fn`, `mask_*` helpers |
| Data filtering | preprocessing pipeline, snakemake rules |

If a claim has multiple implementations (e.g., a vectorized batch path AND a tree-iteration reference path), check **both** — they are a frequent source of divergence between each other and between either of them and the paper.

### Step 4: Compare in detail

**Read the code with the `Read` tool.** Do not rely on `grep` excerpts for anything more than locating files. The whole skill collapses if you skim. Read enough surrounding context to know what each variable in the formula refers to.

For each claim, classify:

| Verdict | Meaning |
|---|---|
| **MATCH** | Paper and code agree. No action. |
| **MISMATCH** | Paper and code disagree on a substantive, falsifiable point. The user must decide whether to fix the paper, the code, or both. |
| **AMBIGUOUS** | Paper is unclear/underspecified; code makes a specific choice. Suggest paper clarification. |
| **MULTIPLE IMPLS DISAGREE** | Two code paths in the same repo disagree with each other (and at most one matches the paper). Highest-priority finding. |
| **STALE PAPER** | Paper describes an earlier version of the code; the code has moved on. |
| **STALE CODE** | Paper describes a fix that the code hasn't picked up yet. |

A **MATCH** verdict is the default — most claims are correct most of the time. If everything is `MISMATCH`, you are probably misreading the code. Stop and re-check the worst offenders.

### Step 5: Skeptic confirmation for non-trivial findings

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

### Step 6: Write the report

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

### Step 7: Hand off

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
