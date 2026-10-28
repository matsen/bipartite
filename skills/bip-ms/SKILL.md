---
name: bip-ms
description: Cold-start for a manuscript session — the project's storehouse of results and their connection to the literature, and its resource for answering questions and guiding the work; the paper is where that knowledge is kept, updated as threads complete without asking, and rewritten by people before submission
---

# /bip-ms

Cold-start for a manuscript session, run at session start from a TeX repository (e.g. `~/writing/cosine` or `~/re/peak-origins/paper`).

## Conventions

### Naming
- `iN` = issue #N, `pN` = PR #N; never bare `#N`.
- First mention in a bullet list: full URL inline.
- `EPIC-N` when an EPIC number is ambiguous across repos.

### Session role

The session holds the project's institutional and literature knowledge: the paper, its notes, rulings, past experiments, and the bip library (`/bip-lit`).
Its first job is to answer questions and guide the work from that store: read new results as a collaborator, judge whether they hold up and what they mean against related work, and decide what happens next.
Answer a peer's or the user's question, draft, or new EPIC from the whole store, and volunteer what bears on it, not only what touches the paper's text.
Run a `/bip-lit` search for prior answers to a new result or design before you reply.
Knowledge the session produces (a derivation, a numerical check) goes into a notes record marked "checked here" or "reported", and into the paper when its thread completes.
A peer's or worker's message is a lead; check the claim against the committed file before it reaches the user.

The manuscript is the source of truth about the project's state: background, framing, current understanding.
It is the context you reason from, not a running log.
It is also a storehouse, not the text that gets submitted.
Write Methods and Results as readable prose, so anyone can follow the project's full scope.
Keep the Introduction and Discussion as outlines that place the work in the literature.
People write those two sections from the outlines, and go over the whole paper line by line, before submission.
The PI directs the science at the level of the paper; agents handle the implementation.

Direct research through issues and PRs, not through constant edits to the paper.
Comment on PRs/issues to request analyses where `docs/guides/roles.md` ("Posting to GitHub") allows it; otherwise message the owning session, draft issues for the implementor (via `/bip-issue-check`), and advise the code-side owner.

Update the manuscript when a research thread is complete: reconcile the finished result into the paper (the manuscript prevails on disagreement; cf. `/bip-ms-audit`).
Decide where it goes and how to word it yourself, then tell the user what changed with the diff; ask the user about the science, never about placement or wording.
A point raised in a talk that changes what the paper says updates the paper too.
Until then, keep findings in notes, not in the paper; the paper stays the stable context.

Out of scope, as a safety boundary: running experiments.
Guiding, scoping, and requesting that work is in scope, and so is reading code and results in tracked repos, yourself or through a subagent.
Do not create issues on your own initiative; surface gaps to the user, and use `/bip-issue-check` when the user asks to file one.

Never modify remote server state (ermine, quokka, orca, etc.): no `snakemake` (even dry-run), `zig build`, `git pull`, `snakemake --unlock`, or any other write, since other agents are running experiments there.
Read-only SSH (`ls`, `cat`, `head`, `grep`, file dates/sizes) is fine; report what you see and let the user or the owning agent make changes.

A question that is the user's to answer follows `/bip-conductor`'s "Paging the user".

## Configuration

The skill reads `.ms-config.json` from the manuscript root (gitignored).

```json
{
  "manuscript": "main.tex",
  "prep_figures_dir": "prep-figures",
  "tracked_repos": [
    {
      "repo": "matsen/peak-origins",
      "local_path": "~/re/peak-origins",
      "epics": [281, 295],
      "fetch_cmds": [
        "make remote-fetch DIR=experiments/2026-03-benchmark/results",
        "make artifacts-pull DIR=figures/final"
      ],
      "remote_watch": {
        "host": "orca02",
        "paths": [
          "~/re/peak-origins/experiments/2026-03-benchmark/results",
          "~/re/peak-origins/figures/final"
        ],
        "patterns": ["*.svg", "*.html", "*.tsv"]
      }
    }
  ]
}
```

- `manuscript`: main TeX file.
- `prep_figures_dir`: where SVGs go for inkscape conversion (default `prep-figures`).
- `tracked_repos`: code repos the manuscript depends on.
  - `repo`: `org/repo` for `gh`.
  - `local_path`: local checkout.
  - `epics`: EPIC issue numbers to monitor.
  - `fetch_cmds`: run inside `local_path`; each pulls only the result directories the manuscript needs, via the repo's own Makefile targets. Update the list as the manuscript's needs change (re-fetching is idempotent).
  - `remote_watch` (optional): `host`, `paths`, and `patterns` for the result monitor (Step 5).
- `session`: set in Step 0.

If the file does not exist, stop and ask the user for the main TeX file, the `prep-figures/` location, and, for each tracked repo, its `org/repo`, local path, EPIC numbers, and result directories to fetch (`grep -E '^[a-z].*:' Makefile` lists targets such as `remote-fetch`, `artifacts-pull`).
Then create `.ms-config.json` and proceed.

## Workflow

### Step 0: Load config and onboarding

```bash
cat .ms-config.json
```

Set its `session` field to this session's `ListAgents` name (the "This session is ..." row), so `/bip-tuckin` and `/bip-continue` don't take another session in this tree for the ms session.

Read `CLAUDE.md` and the top of `misc/session-onboarding.md`: everything above its live-thread index, plus the index.
Then read the Introduction's outline: it should convey the paper's idea and its place in the literature on its own.
If it falls short, extend the outline rather than writing a summary elsewhere.
`/bip-ms-tuckin` Step 4 is the contract for where durable state lives.
Open a thread's detail only when it comes up, by grepping the doc for its handle.

The doc never records open/merged/closed status; check each thread you act on against live GitHub, and trust GitHub on disagreement.
Batch the checks: one `gh` call per repo.
Shallow is enough for status; a claim of absence ("nothing new", "that flag doesn't exist") needs the pulled clone and a full read.

**Reading the manuscript.**
The paper is too large to read whole and is not summarized elsewhere, so read it by topic.
Grep the topic, symbol, or ledger marker (`%CLAIM`, `%PIECE`, `%PROV`) across the whole file, then read the ranges it hits; a section-scoped read misses the `%PROV` in another block that guards your sentence.
For whole-paper tasks (notation, cross-references, duplicate sentences), read the prose only (`grep -v '^\s*%' main.tex`), or hand the read to a subagent and check its edits by diff, not by its account.
After a rewrite, read the diff's deleted lines: a rewrite for one reason silently drops text kept for another.

### Step 1: Check manuscript state

```bash
git status --porcelain | head -10
git log --oneline -5
```

### Step 2: Fan out per-repo scanners

Skip this step when `/bip-continue` brought you here from a continuation file under a day old.

Dispatch one `general-purpose` subagent per `tracked_repos` entry, in parallel, following `SUBAGENT-SCAN.md` (bipartite repo root).
Per-repo, not per-EPIC, so two EPICs in one repo don't race on `git pull`.

Brief for each subagent:

> Scan repo `<org/repo>` for the manuscript session.
> Local path: `<local_path>`.
> EPIC numbers: `<epics>`.
> Fetch commands: `<fetch_cmds>`.
>
> Tasks:
> 1. `git -C <local_path> pull --ff-only origin main` (continue on failure; note it).
> 2. From `<local_path>`, run each fetch_cmd.
> 3. For each EPIC, `gh issue view <N> --repo <org/repo> --json title,body,updatedAt`.
>    Parse Status dashboard, Key findings, and active clone assignments.
> 4. `gh pr list --repo <org/repo> --search "is:merged sort:updated-desc" --limit 10 --json number,title,mergedAt,body`.
>    Note which produced figures or quantitative results.
> 5. Find SVGs and HTMLs under `<local_path>` modified in the last 120 minutes.
>    For each, identify the EPIC or PR that produced it.
>
> Return under 400 words, structured per `SUBAGENT-SCAN.md`:
> - `changes_since_baseline`: EPIC items newly checked, new key findings, PRs merged that produced relevant results
> - `active_items`: clone assignments from EPIC tables, brief status
> - `new_artifacts`: paths to new SVGs/notebooks with EPIC/PR source
> - `action_candidates`: figures to import, notebooks to open, text to draft
> - `surprises`: anything else, including `RECOMMEND DEEPER LOOK` flags
>
> Quote specific EPIC/PR lines that show what changed — do not paste full bodies.
> Use Read (not grep excerpts) for any PR body or finding you cite.

If a report has zero `surprises` and zero `changes_since_baseline`, dispatch a follow-up ("what changed in the last 7 days on this EPIC even if no boxes were checked?") before calling that repo quiet.

### Step 3: Build status table

Compose from the reports, cross-referencing them (only the primary sees all of them):

| Repo | EPIC | New Results | Active Work | Action Needed |
|------|------|-------------|-------------|---------------|
| peak-origins | i281 | 2 new SVGs | 3 clones active | Import figures |
| peak-origins | i295 | notebook updated | PR in review | Draft results |

Then list new figures to import, notebooks to review, and findings to write up, each with its path or EPIC source.

### Step 4: Propose actions

Run the PI questions (`docs/guides/pi-questions.md`) against the current state, then propose next steps: import figures, open notebooks, fold completed threads into the paper, and note gaps for the user.
Fold completed threads in without waiting; wait for the user's word on the rest.

### Step 5: Start result monitor

If a tracked repo has a `remote_watch` entry, offer a persistent Monitor that polls each `host` over SSH every minute, read-only: `find <paths> \( <patterns> \) -newer <marker>`, then touch the marker.
When it reports new files, run that repo's `fetch_cmds`, handle SVGs and notebooks per the workflows below, and tell the user what arrived.

## Figure import workflow

```bash
PREP_DIR=$(jq -r .prep_figures_dir .ms-config.json)
cp "<local-path>/figure.svg" "$PREP_DIR/"
make pdf-figures
```

If the manuscript does not reference the figure yet, place it and write the `\includegraphics` block.

## Notebook review workflow

```bash
open -a "Google Chrome" "<path-to-notebook.html>"
```

Tell the user which notebook opened and which EPIC/issue produced it; after review, fold in the plots and findings that a completed thread rests on.

## Text drafting workflow

1. Read the relevant EPIC findings, PR descriptions, and results.
2. Read the manuscript around the target spot, per "Reading the manuscript" in Step 0.
3. Write it: prose in Methods and Results, outline points in the Introduction and Discussion.
4. Run the `@scientific-tex-editor` agent on new prose.
5. Commit, and tell the user what changed with the diff.

An outline point is one full-sentence claim, followed by the `\cite` keys that support it and the `\ref` to the results it rests on.
A bare topic ("related work on X") is not a point.

Prose you write into the paper, drafted or edited, follows these rules from the start:
- Lead every paragraph with its claim sentence; support follows.
- No This/That/Such or "The consequence" as a sentence-opening subject.
- Split any sentence past ~40 words or with two clauses joined by comma plus and/so/since/while/where.
- Em-dashes: per `@scientific-tex-editor`'s rule.
- No clefts ("it is X that", "what P cannot tell us is") and no "there is/are ... that".
- Avoid inert main verbs (is, has, offers, provides, represents, constitutes).
- Replace vague verbs (emits, captures, encodes, leverages) with the precise relation (estimates, predicts, assigns, defines).
- No meta-commentary ("The key insight is", "Importantly", "It is worth noting").
- No rhetorical flourish: punchy closers, announcing topic sentences, coy negative-space framing, escalating tails.
- Plain register; cut intensifiers (exactly, precisely, genuinely, crucially) unless technically load-bearing.
- One sentence per line in the TeX source.

## Remote server awareness

Results and data live on remote servers (orca/ermine).
When validating claims about results, especially in issues, check over read-only `ssh` that data files exist at the stated paths, that intermediates (filtered FASTAs, DAG protobufs) match what READMEs and Snakefiles describe, and that result TSVs have the expected columns and row counts.
Local READMEs and Snakefiles miss intermediates that change the data (e.g. filtered FASTAs with different taxa, condensed DAGs with extra leaves).

## Error handling

- Local path missing: warn that the repo may need cloning.
- Git pull or fetch cmd fails: warn (dirty worktree, diverged, remote unreachable, path changed) and continue with stale state.

## Session end

Before ending or resetting context, run `/bip-ms-tuckin` to persist state (`misc/session-onboarding.md`, the manuscript's `%PROV`/`%TODO` markers) and commit manuscript changes.
