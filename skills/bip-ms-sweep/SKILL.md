---
name: bip-ms-sweep
description: Pre-share / pre-submission polish sweep of a TeX manuscript. Fans out parallel scans (writing review, citations, orphan floats, terminology, numbers, URLs, tool names, editorial markers), a literature-gap check via bip asta, and submission hygiene. Reports findings as an interactive punch list — does not produce a report file or auto-edit the paper.
allowed-tools: Agent, Bash, Read, Glob, Grep, Edit, Write
---

# /bip-ms-sweep

Pre-share or pre-submission polish sweep of a TeX manuscript. Run from a TeX repository when you want to look the manuscript over carefully before sending it to a colleague or to a journal.

This is **not** a code-vs-paper audit (`/bip-ms-audit`), **not** a cold-start dashboard (`/bip-ms`), and **not** a writing-style review on its own (`@scientific-tex-editor`). It is the cross-cutting manuscript sweep that runs many small, mechanical scans in parallel, plus a literature-gap check and a submission-hygiene check, and then talks through the findings with the user interactively.

The output style mirrors a careful colleague reading the manuscript with you: a conversational punch list, not a markdown report. The user decides what to fix; the skill does not auto-edit the paper.

## Usage

```
/bip-ms-sweep                       # full sweep of the manuscript at the current cwd
/bip-ms-sweep <paper.tex>           # explicit paper file
/bip-ms-sweep --since <ref>         # restrict the writing-review pass to changes since <ref>
                                    # (mechanical scans always run on the full paper)
```

Default scope is the **full paper** (main + SI + bib). The mechanical scans always run on the full paper. Only the writing-style review can be narrowed with `--since`.

## Core principle

**Many small, focused scans beat one big "review my paper."** A single scientific-tex-editor invocation against the whole paper will skim. Eight specialist subagents running in parallel — each with one job and a tight output format — produce a tighter, higher-precision punch list and don't waste primary-agent context on intermediate reasoning. The primary's job is to consolidate, dedupe, and bin findings into "real fix," "judgment call," and "noise" for the user to triage.

**Editorial markers are signal, not noise.** `%EM`, `%HH`, `% TODO`, `% NOTE` etc. are often live communication channels between co-authors. The sweep lists them but does not classify them as warts unless they look unfinished or self-contradictory. The user decides what's a wart.

**No assumptions about repo layout.** Do not require `.ms-config.json`. Do not assume `main.tex`. Detect TeX files in the cwd, accept an explicit path, or ask. If you need transient scratch space, use `_ignore/sweep/` in the manuscript repo (gitignored by convention; create if missing).

## Workflow

### Step 1: Identify manuscript files

```bash
ls *.tex *.bib 2>/dev/null
```

Identify:
- The **main manuscript file** (if a `paper.tex` arg was passed, use it; else look for `main.tex`, then `manuscript.tex`, then the largest `.tex` file in the cwd).
- The **SI file**, if any (commonly `si.tex`, `supplement.tex`, `supp.tex`).
- The **bib file** (commonly `main.bib` or `refs.bib`).

If ambiguous, ask the user before proceeding. State what you identified.

If the user passed `--since <ref>`, capture that ref for Phase 2.

### Step 2: Fan out all sweeps in parallel

Dispatch **all eight scans in one message** with multiple Agent tool calls. Follow the pattern in `SUBAGENT-SCAN.md`. Sequential dispatch defeats the purpose.

The eight scans:

#### Scan 1 — Writing review (`scientific-tex-editor`)

Brief the agent on the manuscript path(s), whether `--since <ref>` was passed, and any project-specific style rules (semantic line breaks, citation style, etc. — read from the project's `CLAUDE.md` if one exists). Cap output around 400 words. Ask for **real problems only** (factual or logical inconsistencies, broken sentences, undefined terms, awkward phrasings introduced by recent edits) — explicitly tell it not to propose stylistic reshaping or reorganization. Group findings as **Must-fix / Should-fix / Optional**.

#### Scan 2 — Citation graph audit (`Explore`)

Find:
- `\cite{}` keys not in the bib (these break compile).
- Bib entries never referenced (pruning candidates).

Output two lists with `file:line` citations, plus a one-line verdict.

#### Scan 3 — Orphan floats + tool-name typesetting (`Explore`)

Combined because both are label/string-level greps.

- For every `\label{fig:…}` / `\label{tab:…}` / `\label{eq:…}`, check that the label is referenced via `\ref`, `\autoref`, `\Cref`, `\cref`, or `\eqref` somewhere. Account for the `xr` cross-document convention (SI labels referenced from main with `supp-` prefix).
- Scan for common tool names (UShER, larch, nextclade, PastML, taxonium, BEAST, IQ-TREE, MUSCLE, matUtils, etc.) and report any tool whose typesetting varies across the document.

Output: two labeled lists.

#### Scan 4 — Numerical consistency (`general-purpose`)

Find numbers that appear in multiple places in the manuscript (counts, percentages, fold-changes, correlation coefficients, p-values, thresholds, hyperparameters). For each, list every appearance with `file:line` and confirm they match. Flag any inconsistency.

The agent should **discover** the numbers to check by reading the manuscript, not work from a fixed list — different papers have different repeated quantitative claims.

Output: one line per checked quantity: ✓ consistent (with line numbers) or ✗ inconsistent (showing the differing values).

#### Scan 5 — Editorial markers + URL consistency + cross-reference direction (`general-purpose`)

Combined because all three are grep-driven with simple classification.

- **Editorial markers**: list every `%EM`, `%HH`, `%TODO`, `%XXX`, `%FIXME`, `% note`, `% NOTE` with `file:line` and truncated text. Classify by whether the comment looks **still in flight** (a question to a co-author, a suggestion not yet acted on) vs. **stale** (refers to deleted code, or to changes that have since been made). Do not call live in-flight comments "warts" — they're communication.
- **URL consistency**: extract every URL (`\url{…}` and bare `http(s)://`); group by host/path prefix; flag any inconsistencies in trailing slashes, repo names, or subpath conventions.
- **Cross-reference direction**: list every "as above," "described below," "see above," "see below," "next section," "previous section" with surrounding context. The primary decides whether each points in the right direction (the subagent just enumerates — section reorderings can flip these without the writer noticing).

#### Scan 6 — Terminology consistency (`general-purpose`)

Cross-check terminology between main and SI:

- Terms whose meaning the paper explicitly distinguishes (e.g., "mutation rate" vs. "substitution rate," "host-agnostic" vs. "host-specific") — verify they are used in the technically-correct sense everywhere, especially in figure captions and Methods.
- Hyphenation, capitalization, and spelling variants of the same term across files (e.g., `\nt{CG}` vs. `\texttt{CG}`, "3-mer" vs. "three-mer," `$r$` vs. `$R$` for correlation).

Output: each finding with `file:line`, classified "real inconsistency that should be fixed" vs. "stylistic variation."

#### Scan 7 — Literature-gap check (`general-purpose`)

This is the one that's easy to skip but valuable. Brief the agent with:

- The manuscript's **topic** (read the abstract + introduction to summarize in 1–2 sentences).
- The **current citation list** (extract all `\cite{}` keys from main.tex and si.tex; pass as a deduplicated list, or `gh`-style sample if too long).
- The instruction: use **`bip asta`** to sweep for recent papers in the topic area, plus `bip search` against the local library to catch anything already imported but not cited. Look for papers that look directly relevant — same problem, same methods, same organism, same theoretical framework — that are not cited.

Example shape of the agent's queries:
```bash
bip asta search "<topic-specific keywords>" --limit 30
bip search "<topic-specific keywords>" --limit 30
```

Output: a short list of candidate citations to consider, each with title, authors, year, venue, and a one-sentence reason why the manuscript might want to cite it. Do **not** include papers the manuscript already cites — diff against the citation list.

Aim for high precision over recall: the user does not want to read about 40 tangentially-related papers. Five strong candidates beat fifty weak ones. If the agent finds nothing relevant, say so.

#### Scan 8 — Submission hygiene (`journal-submission-checker`)

Check:
- Every GitHub / external repository referenced in the manuscript is publicly accessible (curl/WebFetch each URL).
- Every URL in the manuscript resolves (no 404s).
- Recent preprint citations may have been published in journals since posting — flag candidates for bib updates, with **verified DOIs**. Web-search results can be wrong; the agent must verify each preprint→published claim against the primary source before reporting it.
- Standard acknowledgment text (GISAID, funding sources) is present and matches required wording, if applicable.

Output: a short punch list of submission-blockers vs. nice-to-haves.

### Step 3: Wait for all scans, then consolidate

As each agent completes, deliver its findings to the user immediately — don't batch them up. The user can start triaging in parallel with later scans completing.

For each completed scan, give a **terse, conversational summary** in the chat:

```
**Scan N — <name>:**

**Real fixes:**
- <item> (file:line)

**Judgment calls:**
- <item>

**Noise / FYI:**
- <item>
```

Cite specific `file:line` for every actionable finding so the user can jump straight to it. Quote the exact text where ambiguity would otherwise force the user to re-read context.

When all eight scans are in, give a single consolidated punch list with three bins:

- **Real fixes** — mechanical errors and clear bugs.
- **Judgment calls** — semantic or stylistic decisions the user should make.
- **Noise / FYI** — observations not requiring action.

Numbered, deduplicated across scans, with `file:line` for each item.

### Step 4: Interactive triage

Ask the user how they want to proceed. Common patterns:

- **Apply the mechanical fixes** — the user says "fix items 1–6," you apply them with `Edit`.
- **Annotate the TeX** — the user wants `%EM`-style comments left in the source as TODOs for a co-author. Draft them inline near the relevant lines, with attribution (e.g., `%TODO from sweep`).
- **Defer** — the user wants a few items kept in mind for later but not acted on now.
- **Ship as is** — the user reviews and decides nothing needs changing.

Do **not** auto-apply any change without explicit confirmation. Treat the sweep's findings as recommendations, not commits.

### Step 5: Verify build (only if edits were applied)

If you applied edits, recompile:

```bash
make 2>&1 | tail -10        # or: latexmk main.tex
```

Surface any new warnings/errors. Do not commit.

## Subagent dispatch reference

See `SUBAGENT-SCAN.md` in the bipartite repo root for the shared fan-out pattern (parallel dispatch, structured output, when to re-dispatch). The eight scans above plug into that pattern. Re-dispatch any scan whose output is empty or shorter than 50 words — that's the "the subagent didn't look hard" failure mode.

## Guidelines

- **Default to MATCH / consistent / clean.** Most claims in a manuscript are correct. A sweep that returns 40 problems in a 50-page paper is almost certainly an over-eager subagent. If a scan looks like that, re-dispatch with a tighter brief.
- **Quote `file:line` for every actionable item.** The user must be able to jump straight to the source. Never say "around line 200ish."
- **Treat `%EM` / `%HH` markers as communication, not warts.** Classify them, list them, but don't tell the user to strip them unless they look stale or self-contradictory.
- **The literature-gap check is on by default.** It is the highest-yield thing the user is most likely to forget to run.
- **No report file.** Findings live in the chat. If you need transient scratch (e.g., a long list of citation keys you want to pass between subagents), write to `_ignore/sweep/` in the manuscript repo (create the directory if missing; assume it's gitignored by convention or warn the user if not).
- **Do not auto-edit, do not auto-commit.** The user drives.

## When to use this vs siblings

| Need | Skill |
|---|---|
| Cold-start a manuscript session (monitor EPICs and code repos) | `/bip-ms` |
| Mid-session check for new experiment results | `/bip-ms-poll` |
| Persist manuscript session state | `/bip-ms-tuckin` |
| Cross-check paper claims against the code that ran the experiments | `/bip-ms-audit` |
| **Pre-share / pre-submission polish sweep of the paper itself** | **this skill** |
| Per-PR review against guidelines | `/bip-pr-review` |
| Fact-check a reviewer's comment against the code | `/bip-comment-check` |
| Add new TeX citations to the bipartite knowledge graph | `/bip-lit-edges` |

## Why this shape

- **Eight parallel scans, not one big review.** Each scan has one job and a tight output contract. Parallelism keeps wall-clock latency low and primary-agent context clean. The primary's job is consolidation, not analysis.
- **Literature gap is in the default flow.** It's the single most often-skipped check before submission, and `bip asta` makes it cheap. Putting it behind a flag would mean it never runs.
- **Interactive output, not a markdown report.** The user reads the punch list in the chat and decides as findings arrive. A report file is overhead for a workflow designed to drive same-session edits.
- **No config dependency.** Most TeX manuscripts have an obvious main file. Requiring `.ms-config.json` would add friction for the common case. The skill detects what it can, asks when ambiguous.
- **Editorial markers are communication.** Co-authors leave `%EM`/`%HH` notes for each other. The sweep helps the user see them all in one place but doesn't moralize about them.

## References

- Sibling skills: `/bip-ms`, `/bip-ms-poll`, `/bip-ms-tuckin`, `/bip-ms-audit`, `/bip-comment-check`, `/bip-pr-review`.
- Subagents used: `@scientific-tex-editor`, `@journal-submission-checker`, `general-purpose`, `Explore`. None mandatory.
- Subagent dispatch pattern: `SUBAGENT-SCAN.md` (bipartite repo root).
- Precipitating session: flu mutation rates manuscript pre-share sweep, 2026-05-14. Ran the eight scans by hand and caught: 7-panel-letter cascade after a figure caption change; orphan SI floats; bib duplicates; URL-trailing-slash inconsistency; `$R$` vs. `$r$` for Pearson; mutation-vs-substitution rate slippage; missing `float` package for `[H]` floats; a stale `\autoref{fig:summary}D and …` cross-reference. Without the parallel-scan structure, any one of these would likely have shipped.
