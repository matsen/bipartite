---
name: bip-ingest
description: Cold-start the ingest session — triage one batch of outside reading (Semantic Scholar and Google Scholar alerts, or a newsletter saved as .eml in ~/Downloads), cut the noise, send the few survivors to the sessions they bear on, and hand the user a Paperpile list. One batch per round; the user runs /bip-cycle between rounds.
---

# /bip-ingest

The ingest session turns a batch of outside reading into a short list worth acting on.
Most items are noise; start from that prior.
It finds each item and the session it bears on; that session frames it, because only it knows its open threads and, for a paper, whether the numbers are comparable.
Other sessions reach it by name, so ask the user to `/rename ingest` if it is not already named that.
Run it in `$NEXUS_PATH`; for command usage see `/bip-lit`.

It keeps no state between rounds.
On cold start, say you are ready for a batch and wait.

## Reading an .eml

A batch can arrive as `.eml` files in `~/Downloads`, which keep the links that a paste loses: the files the user names, or else the newest one.
Copy each into a new empty folder in your scratchpad, then run `python3 -I <this skill's base directory>/eml.py <copy>`.
It prints the headers, the text, and numbered links with their anchor text (inline in the text for a plain-only email). It drops most unsubscribe links; treat every other link as carrying the user's key too.
Newsletter links are tracking redirects that carry the user's subscriber id: never save or forward one.
Resolve only a link you keep, with `curl -s -o /dev/null -w '%{redirect_url}' '<href>'`, since resolving counts as a click; then drop the resolved URL's query string.
When the round is done, delete the `.eml` files you read from `~/Downloads`.

## A round of literature alerts

1. **Find the owners**, one line per owned EPIC (owner, EPIC, title):

   ```bash
   gh search issues "EPIC in:title" --state open --owner matsengrp --owner matsen --limit 200 \
     --json repository,number,title,body --jq '.[] | select(.title|test("^EPIC"))
       | (.body|capture("(?m)^Owner: *(?<o>\\S+)")?.o) as $o | select($o)
       | "\($o)\t\(.repository.nameWithOwner)#\(.number)\t\(.title)"'
   ```

   Read a body only when the title is too thin to judge a paper against.
   An owner not in `ListAgents` is not live.
2. **Resolve each item** to an S2 paperId, DOI and abstract.
   Use `bip s2 lookup DOI:…` or `bip s2 lookup ARXIV:…` when the email carries an id; otherwise find the paper with `bip asta search "<title>" --limit 3` (default JSON; `--human` hides the paperId), then `bip s2 lookup <paperId>`.
   Take the abstract with `| jq -r .abstract`.
   An item that won't resolve keeps its original link.
3. **Skip papers already in the library**, after `git pull --ff-only` and `bip rebuild` (`/bip-lit`; again before quoting a key to an owner): `bip s2 lookup <id> --exists`.
4. **Cut the noise.**
   An item survives only if you can name, in a few words, the EPIC goal it bears on.
   Judge from the abstract, not the email's TLDR, which can describe a side point.
5. **Ask the owners.**
   Each live owner with survivors gets one numbered message; an owner with none gets nothing.
   A paper that bears on two EPICs goes to both owners, each told who else has it.
   Each entry: title, S2 link (`https://www.semanticscholar.org/paper/<paperId>`), DOI link (`https://doi.org/<doi>`, or `https://arxiv.org/abs/<id>` when there is no DOI), the abstract, and your few-word guess.
   Ask for one line per entry: `N: drop`, `N: keep`, `N: keep → request on <PR/issue>`, or `N: gem`.
   Tell the owner that it frames a keeper in nexus `papers/<key>.md` once the paper is imported (`papers/README.md`), that it drafts any request itself, and that posting on another person's PR or issue waits for the user.
   A reply never blocks the round.
6. **Send each gem to hhmi** as a notice: the links, the owner's reply quoted, and the EPIC.
7. **Report to the user** as one numbered list:
   gems; keepers, with who they went to; survivors whose owner isn't live, with their EPIC; the Paperpile list; then a one-line count of what was cut, with the titles underneath.
   Each Paperpile line carries both the DOI link and the S2 link, or says "no DOI": the user prefers S2, but it can take several clicks and a captcha, so the DOI is the quick route.
   Tag each keeper and gem "Save in S2": saving it to the user's S2 Library folder trains that folder's Research Feed, and S2 has no API to do it for them.

The round ends when the owners have replied or the user says to stop waiting.
Then the user runs `/bip-cycle` for the next batch.

## A round of a newsletter

1. **Cut the noise.** An item survives only if you can name, in a few words, the session it bears on and why:
   - agent practice, skills, and how the fleet works go to `bipartite`, as a proposed change: the item, its resolved link, and what it would change;
   - other topics go to the session whose role covers them (`ListAgents`), or to the one the user names for that topic;
   - a paper goes through the literature round's steps 1–7;
   - an item no live session covers is cut.
   Judge from the item's text, not its headline.
2. **Send each session one numbered message** with its survivors. A reply never blocks the round.
3. **Report to the user** as one numbered list: what went to which session, then a one-line count of what was cut, with the titles underneath.

## Never

- Post on GitHub.
- Run `bip s2 add`; the user adds papers through Paperpile.
- Save or forward a newsletter's tracking or unsubscribe link.
- File a direction a paper suggests; that goes through the owner's `/bip-goals`.
