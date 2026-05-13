# Subagent scan pattern for bip skills

Many cold-start and poll skills (`/bip-ms`, `/bip-ms-poll`, `/bip-epic`, `/bip-epic-poll`, `/bip-ms-audit`) scan many independent sources — every tracked EPIC, every active clone, every recently merged PR. Reading those bodies directly into the primary agent burns context: a single `/bip-ms` invocation can spend 85K tokens loading EPIC bodies and PR descriptions that the primary doesn't need verbatim.

This file is the shared pattern for offloading that scan work to subagents. Reference it from a skill; do not restate its rules inline.

## When to fan out

Use a subagent when the primary would otherwise:

- Read a full EPIC body, full PR body, or large status file just to extract a few facts.
- Iterate over N items (repos, EPICs, clones, sections) where each iteration is the same shape and the items are independent.
- Walk a directory tree for new artifacts and cross-reference each against something else.

Skip the subagent when:

- The check is one `gh` query that returns a small, structured JSON. Just read it.
- The scan is so cheap that delegation overhead exceeds the savings.

## Agent type

Use `general-purpose`. `Explore` is read-only and explicitly not for open-ended analysis — it would miss the "what's new and why it matters" framing these skills need.

## Dispatch in parallel

Launch all independent scanners in a single message with multiple Agent tool calls. Sequential dispatch defeats the purpose.

## Brief template

Each subagent prompt must contain:

1. **What it's scanning** — exact target (repo, EPIC number, clone path, section name).
2. **What to return** — a fixed list of structured fields the primary can compose into a dashboard. Required field set:
   - `changes_since_baseline` — bulleted list of what's new/changed (with timestamps or PR numbers when relevant)
   - `active_items` — items in flight, brief status, who/what is working on them
   - `action_candidates` — concrete next steps the primary might propose to the user
   - `surprises` — anything unexpected, contradictory, or that doesn't fit the above buckets, including notes like `RECOMMEND DEEPER LOOK: <reason>`
3. **A word cap** — usually 300 words total, up to 500 if the target is unusually rich.
4. **A "use Read, not grep, for files that matter" clause** — for any code or paper file whose content is being evaluated (not just located).

## Subagent failure modes the primary must catch

Subagents skim. An empty `surprises` field does not mean "nothing surprising" — it sometimes means "the subagent didn't look hard." Treat each report as a starting point:

- If a report has zero surprises **and** zero `changes_since_baseline`, dispatch a follow-up with a narrower question before concluding "all quiet."
- If a report contradicts something the primary already knows (e.g., it claims a PR is open that you just saw merged), trust the live state and re-dispatch.
- If a `RECOMMEND DEEPER LOOK` flag appears, decide whether to read the source directly or send a follow-up subagent with a tighter brief — do not silently drop it.
- For high-stakes skills (`/bip-ms-audit`), frame the brief as a **line-by-line investigation, not a scan** — require the subagent to cite `file:line` for every claim and to read full files with the `Read` tool (not grep excerpts). Reports without citations don't count and must be re-dispatched.

## Primary responsibilities after fan-out

1. Compose the dashboard / status table / report from the structured fields. Do not paste subagent prose into user-facing output without trimming.
2. Cross-reference reports against each other — the primary sees all of them; the subagents don't.
3. Verify any state claim that will drive user action with a one-line live check (`gh pr view --json state`, etc.) before reporting.
4. Decide whether the surprises bucket warrants a follow-up scan, and dispatch one if so.

The fan-out reduces context, not responsibility. The primary still owns the synthesis and the user-facing summary.
