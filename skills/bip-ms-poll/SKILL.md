---
name: bip-ms-poll
description: Quick poll of tracked EPICs and code repos for new manuscript-relevant results
---

# /bip-ms-poll

Lightweight mid-session update for a manuscript session. Checks what
changed in tracked code repos and EPICs since last check, fetches new
artifacts from remote, and reacts to new results.

For continuous monitoring, prefer the **persistent result monitor**
started by `/bip-ms` (Step 5) — it uses SSH polling to detect new
result files on remote servers in real time. Use `/bip-ms-poll` for:
- Full GitHub reconciliation (EPIC updates, merged PRs, new issues)
- Running `fetch_cmds` to pull results locally after the monitor flags them
- When the result monitor isn't running

For periodic auto-polling: `/loop 10m /bip-ms-poll`

**Never modify remote server state.** Do not run `snakemake` (even
dry-run), `zig build`, `git pull`, `snakemake --unlock`, or any
write command via SSH on remote servers (ermine, quokka, orca, etc.).
Other agents are actively running experiments there. SSH is fine for
read-only inspection (`ls`, `cat`, `head`, `grep`, checking file
dates/sizes), but never run anything that modifies files, locks, or
builds. Local `git pull` and `make remote-fetch` (which uses `rsync`
to copy FROM remote) are fine — they only modify the local clone.

## What to check

Fan out one `general-purpose` subagent per entry in `tracked_repos` —
single message, multiple `Agent` calls in parallel. Follow the
dispatch pattern in `SUBAGENT-SCAN.md` (bipartite repo root). Per-repo
granularity avoids racing on `git pull` when one repo has multiple
EPICs.

Brief for each subagent:

> Lightweight delta poll for repo `<org/repo>` since the previous
> manuscript poll. Local path: `<local_path>`. EPIC numbers:
> `<epics>`. Fetch commands: `<fetch_cmds>`. Last-seen EPIC
> `updatedAt`: `<timestamps>` (from primary's memory; if unknown,
> compare against the last 24h).
>
> Tasks:
> 1. For each EPIC, `gh issue view <N> --repo <org/repo> --json
>    body,updatedAt`. If `updatedAt` is unchanged from baseline,
>    skip body parsing. Otherwise extract newly checked items, new
>    key findings, and changes to active clone assignments.
> 2. `gh pr list --repo <org/repo> --search "is:merged
>    sort:updated-desc" --limit 5 --json
>    number,title,body,mergedAt`. Report only PRs newer than the
>    last poll.
> 3. `git -C <local_path> pull --ff-only origin main` and run each
>    fetch_cmd from `<local_path>` (idempotent; safe to re-run).
> 4. `find <local_path> \( -name "*.svg" -o -name "*.html" \) -mmin
>    -60`. For each new file, identify the producing EPIC or PR.
> 5. `gh pr list --repo <org/repo> --json
>    number,title,headRefName,state`. Note any PRs approaching merge
>    that will produce results soon.
>
> Return under 300 words, structured per `SUBAGENT-SCAN.md`:
> - `changes_since_baseline`: EPIC deltas, new merges, new findings
> - `active_items`: PRs approaching merge with brief status
> - `new_artifacts`: paths to new SVGs/notebooks with source EPIC/PR
> - `action_candidates`: import figure X, open notebook Y, draft
>   text for finding Z
> - `surprises`: anything else, including `RECOMMEND DEEPER LOOK`
>   flags
>
> Use Read (not grep excerpts) for PR bodies or finding text you
> cite. Do not paste full bodies.

If every subagent reports zero `changes_since_baseline` and zero
`surprises`, the poll output is one line: "All quiet across tracked
repos." Otherwise compose the "React to new artifacts" sections below
from the structured reports.

## React to new artifacts

### New SVG figures

When new SVGs are found after a fetch:

1. Show the user what's new:
   ```
   **New SVG**: peak-origins/experiments/benchmark/results/fig3.svg (fetched just now)
   ```

2. Ask if it should be imported to `prep-figures/`:
   ```bash
   PREP_DIR=$(jq -r .prep_figures_dir .ms-config.json)
   cp "<source>" "$PREP_DIR/"
   make pdf-figures
   ```

3. If imported, check if the manuscript already references it. If not,
   suggest placement and draft the `\includegraphics` block.

### New HTML notebooks

When new `.html` notebooks are found after fetch:

```bash
open -a "Google Chrome" "<path-to-notebook.html>"
```

Tell the user what notebook was opened and which EPIC/issue produced it.
After they review, ask which plots or findings to incorporate.

### New key findings in EPICs

When an EPIC body has new findings (numbered items in the Key Findings
section that weren't there before):

1. Quote the finding
2. Read the relevant PR or experiment that produced it
3. Present the key points as a **bullet-point summary**
4. Ask the user which to include and where in the manuscript
5. After confirmation, draft the paragraph(s) in LaTeX
6. Run the `@scientific-tex-editor` agent on the new text for style review
7. Present the edited draft for final approval

### Issue creation

If during polling you notice gaps — an experiment that should be run,
a comparison that's missing, a figure variant that would strengthen
the paper — propose raising an issue on the tracked repo:

1. Describe what's needed and why (from the manuscript perspective)
2. After user confirmation, run `/bip-issue-next` targeting the tracked repo
3. The remote EPIC conductor will pick it up on its next poll

## After polling

### Output structure

1. **New results** (lead with this): Figures, notebooks, or findings
   that arrived since last poll. One line each with source and age.

2. **Active work**: Which EPICs have active clones, brief status.
   Only mention if something changed.

3. **Approaching completion**: PRs close to merge that will produce
   results soon.

4. **Proposed actions**: Concrete list — import figure X, open
   notebook Y, draft text for finding Z, raise issue for gap W.

Ring the terminal bell and send a phone notification if a major new
result arrives (new figure or quantitative finding):
```bash
printf '\a'
NTFY_TOPIC=$(grep ntfy_topic ~/.config/bip/config.yml | awk '{print $2}')
[ -n "$NTFY_TOPIC" ] && curl -s -H "Title: bip ms" -d "New result: <description>" "ntfy.sh/$NTFY_TOPIC" > /dev/null
```

### Verify state before reporting

When mentioning any PR or issue in the poll output, always verify its
current state programmatically (`gh pr view --json state`, `gh issue
view --json state`) rather than relying on earlier poll results or
conversation memory. Stale state leads to confusing reports (e.g.,
reporting a merged PR as "open").

### Keep it brief

This is a poll, not a cold start. Only report what changed. If nothing
changed, say so in one line:

> All quiet across tracked repos. No new results since last check.

## Housekeeping (silent)

- Track `updatedAt` timestamps for EPICs to detect changes efficiently
- Track last-seen merged PR numbers to avoid re-processing
- After fetching, note which files are new vs already seen

## Conventions

Same as `/bip-ms`: `iN`/`pN` prefixes, full URLs on first mention.
