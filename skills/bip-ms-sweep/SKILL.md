---
name: bip-ms-sweep
description: Pre-share / pre-submission polish sweep of a TeX manuscript. Fans out parallel scans (writing review, citations, orphan floats, terminology, numbers, URLs, tool names, editorial markers), a literature-gap check via bip asta, and submission hygiene. Reports findings as an interactive punch list — does not produce a report file or auto-edit the paper.
allowed-tools: Agent, Bash, Read, Glob, Grep, Edit, Write
---

# /bip-ms-sweep

Pre-share or pre-submission polish sweep of a TeX manuscript, run from its repo.
Eight focused scans run in parallel; the primary consolidates them into a punch list in chat and the user decides what to fix.
Not a code-vs-paper audit (`/bip-ms-audit`) or a manuscript cold start (`/bip-ms`).

## Usage

```
/bip-ms-sweep                       # full sweep of the manuscript at the current cwd
/bip-ms-sweep <paper.tex>           # explicit paper file
/bip-ms-sweep --since <ref>         # restrict the writing-review pass to changes since <ref>
                                    # (mechanical scans always run on the full paper)
```

Default scope is the full paper (main + SI + bib).

## Core principle

- Editorial markers (`%EM`, `%HH`, `% TODO`, `% NOTE`, …) are live co-author communication. List them; flag one only if it looks stale or self-contradictory.
- Assume no repo layout and no `.ms-config.json`. Transient scratch goes in `_ignore/sweep/` in the manuscript repo (create it; warn if not gitignored). No report file.

## Workflow

### Step 1: Identify manuscript files

```bash
ls *.tex *.bib 2>/dev/null
```

- Main file: the `paper.tex` arg if given, else `main.tex`, then `manuscript.tex`, then the largest `.tex`.
- SI file, if any (commonly `si.tex`, `supplement.tex`, `supp.tex`).
- Bib file (commonly `main.bib` or `refs.bib`).

Ask if ambiguous; state what you picked.

### Step 2: Fan out all sweeps in parallel

Dispatch all eight scans in one message, following `SUBAGENT-SCAN.md` (bipartite repo root). Re-dispatch any scan whose output is empty or under 50 words.

#### Scan 1 — Writing review (`scientific-tex-editor`)

Pass the manuscript path(s), any `--since <ref>`, and project style rules from the project's `AGENTS.md` or `CLAUDE.md`.
Cap at ~400 words. Real problems only (factual or logical inconsistencies, broken sentences, undefined terms, awkward phrasings from recent edits); no stylistic reshaping or reorganization.
Group as Must-fix / Should-fix / Optional.

#### Scan 2 — Citation graph audit (`Explore`)

- `\cite{}` keys not in the bib (these break compile).
- Bib entries never referenced.

Two lists with `file:line`, plus a one-line verdict.

#### Scan 3 — Orphan floats + tool-name typesetting (`Explore`)

- Every `\label{fig:…}` / `\label{tab:…}` / `\label{eq:…}` must be referenced via `\ref`, `\autoref`, `\Cref`, `\cref`, or `\eqref`. Account for `xr` cross-document refs: main refers to SI labels through the `\externaldocument` prefix (e.g. `supp-`).
- Tool names (UShER, larch, nextclade, PastML, taxonium, BEAST, IQ-TREE, MUSCLE, matUtils, etc.) whose typesetting varies across the document.

Two labeled lists.

#### Scan 4 — Numerical consistency (`general-purpose`)

Discover numbers that recur in the manuscript (counts, percentages, fold-changes, correlations, p-values, thresholds, hyperparameters) by reading it, not from a fixed list.
One line per quantity: ✓ consistent (with line numbers) or ✗ inconsistent (showing the differing values).

#### Scan 5 — Editorial markers + URL consistency + cross-reference direction (`general-purpose`)

- Editorial markers: every `%EM`, `%HH`, `%CC`, `%TODO`, `%XXX`, `%FIXME`, `% note`, `% NOTE` with `file:line` and truncated text, classed in-flight vs. stale.
- URLs (`\url{…}` and bare `http(s)://`): group by host/path prefix; flag inconsistent trailing slashes, repo names, or subpaths.
- Cross-reference direction: enumerate "as above," "described below," "see above," "see below," "next section," "previous section" with context. The primary judges direction, since section reorderings flip these silently.

#### Scan 6 — Terminology consistency (`general-purpose`)

Across main and SI:
- Terms the paper explicitly distinguishes (e.g., "mutation rate" vs. "substitution rate") used in the correct sense everywhere, especially captions and Methods.
- Hyphenation, capitalization, and spelling variants (e.g., `\nt{CG}` vs. `\texttt{CG}`, "3-mer" vs. "three-mer," `$r$` vs. `$R$` for correlation).

Each finding with `file:line`, classed real inconsistency vs. stylistic variation.

#### Scan 7 — Literature-gap check (`general-purpose`)

Brief the agent with:
- The topic in 1–2 sentences, from the abstract and introduction.
- The deduplicated `\cite{}` keys from main and SI (a sample if too long).
- Use `bip asta` for recent papers and `bip search` for library papers imported but not cited:

```bash
bip asta search "<topic-specific keywords>" --limit 30
bip search "<topic-specific keywords>" --limit 30
```

Output: candidates directly relevant (same problem, methods, organism, or framework) and not already cited (diff against the key list), each with title, authors, year, venue, and a one-sentence reason. Precision over recall: five strong candidates at most; say so if none.

#### Scan 8 — Submission hygiene (`journal-submission-checker`)

- Every referenced GitHub / external repository is publicly accessible, and every URL resolves (curl/WebFetch each).
- Preprint citations since published in journals: flag for bib update with DOIs verified against the primary source, since web-search results can be wrong.
- Required acknowledgment text (GISAID, funding) is present with the right wording, if applicable.

Output: submission-blockers vs. nice-to-haves.

### Step 3: Wait for all scans, then consolidate

Post a terse summary of each scan's findings as it completes, so the user can triage early:

```
**Scan N — <name>:**

**Real fixes:**
- <item> (file:line)

**Judgment calls:**
- <item>

**Noise / FYI:**
- <item>
```

Give `file:line` for every actionable item; quote the text where context is ambiguous.
Most claims in a manuscript are correct: a finding of doubtful weight goes in Judgment calls or Noise, not Real fixes. A scan that reports dozens of problems is over-eager; re-dispatch it with a tighter brief.

When all eight are in, give one numbered, deduplicated punch list in three bins with `file:line`: Real fixes (mechanical errors, clear bugs), Judgment calls (semantic or stylistic decisions), Noise / FYI.

### Step 4: Interactive triage

Ask how to proceed: apply chosen fixes with `Edit`, annotate the TeX with comments for a co-author (e.g., `%TODO from sweep`), defer, or ship as is.
Apply nothing without explicit confirmation.

### Step 5: Verify build (only if edits were applied)

```bash
make 2>&1 | tail -10        # or: latexmk main.tex
```

Surface new warnings or errors. Do not commit.
