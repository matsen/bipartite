---
name: bip.ms
description: Manuscript cold-start dashboard — scan tracked EPICs and code repos for new results
---

# /bip.ms

Cold-start dashboard for a manuscript session. Run from a **TeX repository**
(e.g. `~/writing/cosine` or `~/re/peak-origins/paper`). This session
monitors one or more EPIC issues in remote code repositories and reacts
when new results, figures, or findings arrive.

Use this at **session start** to establish context. For mid-session
updates, use `/bip.ms.poll`.

## Conventions

### Naming
- `iN` = issue #N, `pN` = PR #N. Never bare `#N`.
- First mention in bullet lists: full URL inline.
- EPIC issues are referenced as `EPIC-N` when ambiguous across repos.

### Manuscript role
This session **writes the paper**. It does not do feature work, run
experiments, or create/manage issues on tracked repos — that's what
the EPIC workers and conductor do (running on a remote). This session:
- Monitors tracked EPICs for new results
- Pulls local clones, then runs their Makefile fetch targets to bring results from remote
- Imports SVGs into `prep-figures/`, opens HTML notebooks in Chrome
- Drafts results and methods text based on new findings
- Maintains the manuscript TeX files

**Out of scope:** Running experiments, creating issues on tracked repos,
or kicking off computational work. If manuscript work reveals a gap,
note it for the user — they will handle issue creation separately.

**Never modify remote server state.** Do not run `snakemake` (even
dry-run), `zig build`, `git pull`, `snakemake --unlock`, or any
write command on remote servers (ermine, quokka, orca, etc.). Other
agents are actively running experiments there. SSH is fine for
read-only inspection (`ls`, `cat`, `head`, `grep`, checking file
dates/sizes), but never run anything that modifies files, locks, or
builds. Report what you observe and let the user or the responsible
agent handle modifications.

**Issue quality gate:** When the user asks to file an issue during a
manuscript session, always run `/bip.issue.check` on the draft before
submitting via `/bip.issue.file`. Do not shortcut to `gh issue create`
directly, regardless of perceived simplicity.

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
  - **fetch_cmds**: Shell commands to run **inside `local_path`** to fetch specific result directories from remote. Each command should be selective — pull only the results the manuscript needs, not the entire experiment tree. Uses the repo's own Makefile targets (which know the remote host and rsync config).
  - **remote_watch** (optional): Configuration for the persistent result monitor (Step 7). Fields:
    - **host**: SSH hostname for the remote server
    - **paths**: Remote directories to watch for new results
    - **patterns**: File glob patterns to match (e.g. `*.svg`, `*.html`, `*.tsv`)

**Updating fetch_cmds**: As new experiments land and the manuscript
needs different results, update this list. Old entries can be kept
(re-fetching is idempotent) or removed when no longer relevant.

**If the file does not exist**, stop and ask the user:
1. What is the main TeX file? (e.g. `main.tex`)
2. Where is `prep-figures/`? (or equivalent)
3. Which code repos does this manuscript track? For each:
   - GitHub `org/repo`
   - Local checkout path
   - EPIC issue numbers
   - Which result directories should be fetched? (check the repo's Makefile for `remote-fetch`, `artifacts-pull`, etc. — run `grep -E '^[a-z].*:' Makefile` to see targets)

Then create `.ms-config.json` and proceed.

## Workflow

### Step 0: Load config and memory

```bash
cat .ms-config.json
```

Read `MEMORY.md` from the auto-memory directory. For each memory file
listed there, read it and apply:

- **Project memories** (e.g., `project_dasmfit_status.md`): Use as
  the baseline for what's done vs pending. Cross-check against live
  GitHub state — memories can be stale. When a memory says "PR open"
  but `gh pr view` says merged, trust GitHub and update the memory.
- **Pending decisions** (e.g., `project_pending_decisions.md`): Check
  whether they've been resolved since last session. Remove resolved
  items, flag unresolved ones in the status table.
- **Feedback memories**: Apply silently — these are behavioral
  guidelines, not status items.

After loading, briefly note what the memory says the current state is,
then verify it in Steps 1-4. Do not trust memory over live state.

### Step 1: Check manuscript state

```bash
git status --porcelain | head -10
git log --oneline -5
```

Note any uncommitted changes or recent work.

### Step 2: Scan each tracked repo's EPICs

For each repo in `tracked_repos`, for each EPIC number:

```bash
gh issue view <epic-number> --repo <org/repo> --json title,body,updatedAt
```

Parse the EPIC body's **Status dashboard** to extract:
- Completed items (checked boxes) — especially newly completed since last session
- Active items (unchecked, assigned to clones)
- Key findings section — new numbered findings

### Step 3: Pull and fetch results

Two-step process for each tracked repo:

**Step 3a: Git pull** — gets committed code, Makefile updates, and any
committed result files:
```bash
LOCAL_PATH=<expanded local_path>
git -C "$LOCAL_PATH" pull --ff-only origin main
```

**Step 3b: Selective fetch** — run each command in `fetch_cmds` to pull
specific result directories from the remote. These are run inside
`local_path`:
```bash
LOCAL_PATH=<expanded local_path>
cd "$LOCAL_PATH"
# Run each fetch command from config
make remote-fetch DIR=experiments/2026-03-benchmark/results
make artifacts-pull DIR=figures/final
```

**Important**: Only fetch what's configured. Remote experiment trees can
be very large — we pull only the specific directories the manuscript
needs. When a new experiment completes that the manuscript should
incorporate, add its result path to `fetch_cmds` in `.ms-config.json`.

If `fetch_cmds` is empty or missing, skip the fetch and just work with
what's already local after the git pull. Warn the user that remote
results won't be checked.

### Step 4: Identify new artifacts

After fetching, find what's new:

```bash
LOCAL_PATH=<expanded local_path>

# SVGs (recently modified)
find "$LOCAL_PATH" -name "*.svg" -mmin -120 | head -20

# HTML notebooks
find "$LOCAL_PATH" -name "*.html" -mmin -120 | head -20
```

Cross-reference with EPIC findings and recently merged PRs:

```bash
gh pr list --repo <org/repo> --search "is:merged sort:updated-desc" --limit 5 --json number,title,body,mergedAt
```

### Step 5: Build status table

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

### Step 6: Propose actions

Based on what's new, propose concrete next steps:

1. **Import figures**: Copy new SVGs to `prep-figures/`, run `make pdf-figures`
2. **Open notebooks**: Open HTML notebooks in Chrome for review
3. **Draft text**: Summarize findings in bullets, then draft results/methods
4. **Note gaps**: If manuscript work reveals missing experiments or analyses,
   note them for the user (do not create issues — that's out of scope)

Wait for user confirmation before taking action.

### Step 7: Start result monitor

If any tracked repo has a `remote_watch` configuration, offer to start a
**persistent Monitor** that watches remote servers for new result files
via SSH. This provides real-time awareness of experiment completion
without waiting for the next `/bip.ms.poll` cycle.

Use the Monitor tool with `persistent: true`:

```
description: "Remote experiment results"
persistent: true
command: |
  # Built from .ms-config.json remote_watch entries
  touch /tmp/.ms-monitor-baseline

  while true; do
    CHANGED=0
    # For each tracked repo with remote_watch:
    #   HOST=<remote_watch.host>
    #   PATHS=<remote_watch.paths joined by space>
    #   PATTERNS=<-name "*.svg" -o -name "*.html" etc.>
    #
    # SSH to check for new files (read-only):
    NEW=$(ssh -o ConnectTimeout=5 "$HOST" \
      "find $PATHS \( $PATTERNS \) -newer /tmp/.ms-monitor-mark-\$USER 2>/dev/null" \
      || true)
    if [ -n "$NEW" ]; then
      echo "$NEW" | while read f; do
        echo "NEW on $HOST: $f"
      done
      CHANGED=1
      # Update remote marker
      ssh -o ConnectTimeout=5 "$HOST" "touch /tmp/.ms-monitor-mark-\$USER" || true
    fi

    [ "$CHANGED" -eq 0 ] || true
    sleep 60
  done
```

The conductor dynamically builds this script from `.ms-config.json` at
startup — the template above shows the structure. Each repo's
`remote_watch` contributes one SSH check block.

When a notification arrives showing new files:
1. Run the repo's `fetch_cmds` to pull the new results locally
2. Check if the files are SVGs/notebooks and react per the import workflows below
3. Notify the user with a summary of what arrived

**Prerequisites**: SSH access to the remote host with key-based auth
(no password prompts). If SSH fails, the monitor logs the error to
stderr and retries on the next cycle.

**Alternative: sshfs + fswatch** — For lower latency, mount the remote
result directories via `sshfs` and use `fswatch` locally:
```bash
sshfs host:/remote/results /local/mount -o reconnect,ServerAliveInterval=15
fswatch --batch-marker=EOF /local/mount --include '*.svg' --include '*.html' --exclude '.*'
```
This gives true real-time notification but requires `sshfs` (`brew
install macfuse sshfs`) and is less robust on flaky networks. The SSH
poll approach is the default recommendation.

## Figure import workflow

When importing an SVG from a fetched result:

```bash
PREP_DIR=$(jq -r .prep_figures_dir .ms-config.json)
cp "<local-path>/figure.svg" "$PREP_DIR/"
make pdf-figures
```

Then check if the figure is already referenced in the manuscript. If not,
suggest where to add it and draft the `\includegraphics` block.

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
2. Read the current manuscript to understand style, notation, and structure
3. Present the key points as a **bullet-point summary** and ask the user
   which to include and where in the manuscript they belong
4. After confirmation, draft the paragraph(s) in LaTeX
5. Run the `@scientific-tex-editor` agent on the new text for style review
6. Present the edited draft for final approval before inserting into the TeX file

## Remote server awareness

Experiment results and data live on remote servers (orca/ermine), not
locally. When validating claims about experiment results — especially
when drafting or checking issues — use `ssh` to verify:
- That data files exist at the stated paths
- That intermediate outputs (filtered FASTAs, DAG protobufs) match
  what READMEs and Snakefiles describe
- That result TSVs have the expected columns and row counts

Do not assume local READMEs and Snakefiles are the full picture.
Experiments may produce filtered or transformed intermediates that
change the data (e.g., filtered FASTAs with different taxa, condensed
DAGs with extra leaves). Always check the actual files on disk.

## Error handling

- **Config missing**: Ask user to configure (see above)
- **Local path doesn't exist**: Warn — repo may need cloning
- **Git pull fails**: Warn (dirty worktree? diverged?) and continue with stale state
- **Fetch cmd fails**: Warn — remote may be unreachable or path may have changed. Report and continue.
- **EPIC not found**: Check if issue number is correct
- **No new results**: Report "all quiet" and suggest checking back later

## Session end

Before ending a manuscript session or resetting context, run
`/bip.ms.tuckin` to persist session state to memory and commit any
manuscript changes.
