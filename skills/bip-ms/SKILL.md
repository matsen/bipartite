---
name: bip-ms
description: Cold-start for a manuscript session — the project's resource for answering questions and guiding the work from everything the project and the field know; the paper is where that knowledge is kept, updated as threads complete, and research is directed through issues/PRs
---

# /bip-ms

Cold-start for a manuscript session.
Run from a **TeX repository** (e.g. `~/writing/cosine` or `~/re/peak-origins/paper`).
The manuscript is the **source of truth** about the project: one should be able to understand the project's state by reading it.
The session is the project's resource for answering questions and guiding the work, from everything the project and the field know; writing the paper follows from that perspective, not the reverse.

Use this at **session start** to establish context.
For mid-session updates, use `/bip-ms-poll`.

## Conventions

### Naming
- `iN` = issue #N, `pN` = PR #N.
  Never bare `#N`.
- First mention in bullet lists: full URL inline.
- EPIC issues are referenced as `EPIC-N` when ambiguous across repos.

### Session role

This session is the project's **store of institutional and literature knowledge**: the paper, its notes, rulings and past experiments, and the literature in the bip library (`/bip-lit`).
Its **first job is to answer questions and guide the work from that store**: read new results closely enough to argue about them as a collaborator, judge whether a result holds up and what it means, weigh it against related work and what the project already knows, and decide what should happen next.
When a peer or the user brings a question, a draft, or a new EPIC, answer from the whole store and volunteer what bears on it, not only what touches the paper's text.
A new result or design gets a `/bip-lit` search for prior answers before you reply, not only at issue-check time.
Knowledge the session produces itself, such as a derivation or a numerical check, goes into a notes record marked as checked here or as reported, and into the paper when its thread completes.
A peer's or worker's message is a lead, and the committed file is the fact: check the claim against the file before it reaches the user.

The manuscript is the **source of truth** about the project: it holds the background, framing, and current understanding, and one should be able to grasp the project's state by reading it.
It is where that knowledge is kept and eventually published, and the context you reason *from*, not a running log you append to.
The PI directs the science at the level of the paper while agents handle the implementation.

Research is directed **through issues and PRs**, not through constant edits to the paper.
Comment on PRs/issues to request analyses where `docs/guides/roles.md` ("Posting to GitHub") allows it, otherwise message the owning session, draft issues for the implementor (via `/bip-issue-check`), and advise the code-side owner on what to build next.

The manuscript is **updated when a thread of research is complete** — you then reconcile the finished result into the paper (the manuscript prevails on disagreement; cf. `/bip-ms-audit`), rather than appending every incremental finding as it lands.
Until then the paper stays the stable context the discussion runs against.

Supporting mechanics, in service of the above: monitor tracked EPICs, pull clones and run Makefile fetch targets, import SVGs into `prep-figures/`, open HTML notebooks in Chrome.

**Out of scope — a safety boundary, not a limit on what you may think about or direct:** this session does not itself run experiments or modify remote server state (other agents are actively working there — see the next rule).
Guiding, scoping, and *requesting* that work is central and in scope, and so is reading code and results in the tracked repos, yourself or through a subagent.
Do **not** create issues on your own initiative — surface gaps to the user; when the user explicitly asks to file one, use `/bip-issue-check`.

**Never modify remote server state.**
Do not run `snakemake` (even dry-run), `zig build`, `git pull`, `snakemake --unlock`, or any write command on remote servers (ermine, quokka, orca, etc.).
Other agents are actively running experiments there.
SSH is fine for read-only inspection (`ls`, `cat`, `head`, `grep`, checking file dates/sizes), but never run anything that modifies files, locks, or builds.
Report what you observe and let the user or the responsible agent handle modifications.

**A question that is the user's to answer** follows `/bip-conductor`'s "Paging the user".

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

Fields:
- **manuscript**: Main TeX file to edit
- **prep_figures_dir**: Where SVGs go for inkscape conversion (default: `prep-figures`)
- **tracked_repos**: List of code repositories this manuscript depends on
  - **repo**: `org/repo` for `gh` commands
  - **local_path**: Local checkout of the repo
  - **epics**: EPIC issue numbers to monitor
  - **fetch_cmds**: Shell commands to run **inside `local_path`** to fetch specific result directories from remote.
    Each command should be selective — pull only the results the manuscript needs, not the entire experiment tree.
    Uses the repo's own Makefile targets (which know the remote host and rsync config).
  - **remote_watch** (optional): Configuration for the persistent result monitor (Step 5).
    Fields:
    - **host**: SSH hostname for the remote server
    - **paths**: Remote directories to watch for new results
    - **patterns**: File glob patterns to match (e.g. `*.svg`, `*.html`, `*.tsv`)

**Updating fetch_cmds**: As new experiments land and the manuscript needs different results, update this list.
Old entries can be kept (re-fetching is idempotent) or removed when no longer relevant.

**If the file does not exist**, stop and ask the user:
1. What is the main TeX file?
   (e.g. `main.tex`)
2. Where is `prep-figures/`?
   (or equivalent)
3. Which code repos does this manuscript track?
   For each:
   - GitHub `org/repo`
   - Local checkout path
   - EPIC issue numbers
   - Which result directories should be fetched?
     (check the repo's Makefile for `remote-fetch`, `artifacts-pull`, etc. — run `grep -E '^[a-z].*:' Makefile` to see targets)

Then create `.ms-config.json` and proceed.

## Workflow

### Step 0: Load config and onboarding

```bash
cat .ms-config.json
```

Set its `session` field to this session's `ListAgents` name (the "This session is ..." row), so `/bip-tuckin` and `/bip-continue` don't take another session in this tree for the ms session.

Read `CLAUDE.md` and the top of `misc/session-onboarding.md`: everything above its live-thread index, plus the index itself.
Then read the Introduction's prose, the turn-one read of the paper: it should convey the paper's idea on its own, and may end with an outline of the other sections.
If it falls short, propose an Introduction edit rather than a summary elsewhere.
Durable session state lives in the repo; `/bip-ms-tuckin` Step 4 is the contract for what goes where.
Open a thread's detail only when that thread comes up, by grepping the doc for its handle.

The doc never records open/merged/closed status, so check each thread you act on against live GitHub; where they disagree, trust GitHub.
Batch the status checks (one `gh` call per repo, not one per thread).
Shallow is enough for status; a claim of absence ("nothing new", "that flag doesn't exist") still needs the pulled clone and a full read.

**Reading the manuscript.**
The paper is too large to read whole, and it is not summarized anywhere else, so read it by topic.
Grep the topic, symbol, or ledger marker (`%CLAIM`, `%PIECE`, `%PROV`) across the whole file, then read the ranges it hits; a section-scoped read misses the `%PROV` that guards your sentence from another block.
When a task needs the whole paper (notation consistency, cross-references, duplicate sentences), read the prose only (`grep -v '^\s*%' main.tex`), or hand the read to a subagent and check its edits with a diff rather than its own account.
After a rewrite, read the diff's deleted lines, since a rewrite for one reason silently drops text kept for another.

### Step 1: Check manuscript state

```bash
git status --porcelain | head -10
git log --oneline -5
```

Note any uncommitted changes or recent work.

### Step 2: Fan out per-repo scanners

Skip this step when `/bip-continue` brought you here from a continuation file under a day old: its FIRST CHECKs already re-derived what moved.

For each entry in `tracked_repos`, dispatch one `general-purpose` subagent **in parallel** — single message, multiple `Agent` tool calls.
Follow the dispatch pattern in `SUBAGENT-SCAN.md` (bipartite repo root).
Per-repo (not per-EPIC) granularity avoids racing on `git pull` when one repo has multiple EPICs.

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

If any report has zero `surprises` *and* zero `changes_since_baseline`, dispatch a follow-up with a narrower question ("what changed in the last 7 days on this EPIC even if no boxes were checked?")
before concluding "all quiet" for that repo.

### Step 3: Build status table

Compose from the per-repo subagent reports (do not paste their prose verbatim).
Cross-reference the reports against each other — the primary sees all of them; the subagents don't.
Display a summary of what's happening across all tracked repos:

| Repo | EPIC | New Results | Active Work | Action Needed |
|------|------|-------------|-------------|---------------|
| peak-origins | i281 | 2 new SVGs | 3 clones active | Import figures |
| peak-origins | i295 | notebook updated | PR in review | Draft results |

Then list specific new artifacts:

**New figures to import:**
- `peak-origins/experiments/benchmark/results/fig3-comparison.svg` (fetched just now)

**New notebooks to review:**
- `peak-origins/experiments/benchmark/results/analysis.html` (from i281)

**New findings to write up:**
- EPIC i281 finding #7: "Clamping improves convergence by 3x"

### Step 4: Propose actions

Based on what's new, propose concrete next steps, after running the PI questions (`docs/guides/pi-questions.md`) against the current state:

1. **Import figures**: Copy new SVGs to `prep-figures/`, run `make pdf-figures`
2. **Open notebooks**: Open HTML notebooks in Chrome for review
3. **Draft text**: Summarize findings in bullets, then draft results/methods
4. **Note gaps**: If manuscript work reveals missing experiments or analyses, note them for the user; do not file issues on your own initiative.
   If the user asks you to file one, use `/bip-issue-check`.

Wait for user confirmation before taking action.

### Step 5: Start result monitor

If a tracked repo has a `remote_watch` entry, offer a persistent Monitor that polls each `host` over SSH every minute, read-only: `find <paths> \( <patterns> \) -newer <marker>`, then touch the marker.
When it reports new files, run that repo's `fetch_cmds`, handle SVGs and notebooks per the workflows below, and tell the user what arrived.

## Figure import workflow

When importing an SVG from a fetched result:

```bash
PREP_DIR=$(jq -r .prep_figures_dir .ms-config.json)
cp "<local-path>/figure.svg" "$PREP_DIR/"
make pdf-figures
```

Then check if the figure is already referenced in the manuscript.
If not, suggest where to add it and draft the `\includegraphics` block.

## Notebook review workflow

When a new HTML notebook is found:

```bash
open -a "Google Chrome" "<path-to-notebook.html>"
```

Tell the user what notebook was opened and which EPIC/issue produced it.
After they review, ask which plots or findings to incorporate.

## Text drafting workflow

When drafting new results or methods text:

1. Read the relevant EPIC findings, PR descriptions, and experiment results
2. Read the manuscript around where the text will go, per "Reading the manuscript" in Step 0, for style, notation, and structure
3. Present the key points as a **bullet-point summary** and ask the user which to include and where in the manuscript they belong
4. After confirmation, draft the paragraph(s) in LaTeX
5. Run the `@scientific-tex-editor` agent on the new text for style review
6. Present the edited draft for final approval before inserting into the TeX file

Any prose you write into the paper, drafted or edited, follows these rules from the start, so the editor's pass has less to fix:
- Lead every paragraph with its claim sentence; put all support after it.
- Don't open a sentence with This/That/Such or "The consequence" as its subject.
- Split any sentence past ~40 words or with two clauses joined by comma plus and/so/since/while/where.
- Never use em-dashes; use a period, comma, colon, or parenthetical.
- No clefts ("it is X that", "what P cannot tell us is") and no "there is/are ... that".
- Avoid inert main verbs (is, has, offers, provides, represents, constitutes) that bury the real action.
- Replace vague verbs (emits, captures, encodes, leverages) with the precise relation (estimates, predicts, assigns, defines).
- No meta-commentary on your own claims ("The key insight is", "Importantly", "It is worth noting").
- No rhetorical flourish: punchy closers, announcing topic sentences, coy negative-space framing, escalating tails.
- Plain register; cut intensifiers (exactly, precisely, genuinely, crucially) unless technically load-bearing.
- One sentence per line in the TeX source.

## Remote server awareness

Experiment results and data live on remote servers (orca/ermine), not locally.
When validating claims about experiment results — especially when drafting or checking issues — use `ssh` to verify:
- That data files exist at the stated paths
- That intermediate outputs (filtered FASTAs, DAG protobufs) match what READMEs and Snakefiles describe
- That result TSVs have the expected columns and row counts

Do not assume local READMEs and Snakefiles are the full picture.
Experiments may produce filtered or transformed intermediates that change the data (e.g., filtered FASTAs with different taxa, condensed DAGs with extra leaves).
Always check the actual files on disk.

## Error handling

- **Config missing**: Ask user to configure (see above)
- **Local path doesn't exist**: Warn — repo may need cloning
- **Git pull fails**: Warn (dirty worktree?
  diverged?)
  and continue with stale state
- **Fetch cmd fails**: Warn — remote may be unreachable or path may have changed.
  Report and continue.
- **EPIC not found**: Check if issue number is correct
- **No new results**: Report "all quiet" and suggest checking back later

## Session end

Before ending a manuscript session or resetting context, run `/bip-ms-tuckin` to persist session state to the repo (`misc/session-onboarding.md` and the manuscript's `%PROV`/`%TODO` markers) and commit any manuscript changes.
