---
name: bip-librarian
description: Cold-start the librarian session — triage one pasted batch of Semantic Scholar and Google Scholar alert emails, cut the noise, ask the relevant EPIC owners about the few survivors, send gems to hhmi, and hand the user a Paperpile list. One batch per round; the user clears between rounds.
---

# /bip-librarian

The librarian turns a pasted batch of alert emails into a short list worth reading.
Most items are noise; start from that prior.
The librarian finds the paper and the EPIC it bears on; the EPIC owner frames it, because only the owner knows the open threads and whether the numbers are comparable.
Other sessions reach it by name, so ask the user to `/rename librarian` if it is not already named that.
Run it in `$NEXUS_PATH`; for command usage see `/bip-lit`.

It keeps no state between rounds.
On cold start, say you are ready for a batch and wait.

## A round

1. **Find the owners.**
   Run `gh search issues "EPIC in:title" --state open --owner matsengrp --owner matsen --limit 200 --json repository,number,title`.
   Keep titles matching `^EPIC`, read each body, and keep those with an `Owner:` line, noting the owner and the goal at the top of the body.
   An owner not in `ListAgents` is not live.
2. **Resolve each item** to an S2 paperId, DOI and abstract.
   Use `bip s2 lookup DOI:…` or `bip s2 lookup ARXIV:…` when the email carries an id; otherwise find the paper with `bip asta search "<title>"`, then `bip s2 lookup <paperId>`.
   An item that won't resolve keeps its original link.
3. **Skip papers already in the library**: `bip s2 lookup <id> --exists`, plus `bip search "<title>"` when the item has no DOI, since `--exists` matches by DOI and misses arXiv preprints.
4. **Cut the noise.**
   An item survives only if you can name, in a few words, the EPIC goal it bears on.
5. **Ask the owners.**
   Each live owner with survivors gets one numbered message; an owner with none gets nothing.
   Each entry: title, S2 link (`https://www.semanticscholar.org/paper/<paperId>`), DOI link (or `https://arxiv.org/abs/<id>` when there is no DOI), the abstract, and your few-word guess.
   Ask for one line per entry: `N: drop`, `N: keep`, `N: keep → request on <PR/issue>`, or `N: gem`.
   Tell the owner that it frames a keeper in nexus `papers/<key>.md` once the paper is imported (`papers/README.md`), that it drafts any request itself, and that posting on another person's PR or issue waits for the user.
   A reply never blocks the round.
6. **Send each gem to hhmi** as a notice: the links, the owner's reply quoted, and the EPIC.
7. **Report to the user** as one numbered list:
   gems; keepers, with who they went to; survivors whose owner isn't live, with their EPIC; the Paperpile list; then a one-line count of what was cut, with the titles underneath.
   Each Paperpile line carries both the DOI link and the S2 link, or says "no DOI": the user prefers S2, but it can take several clicks and a captcha, so the DOI is the quick route.

The round ends when the owners have replied or the user says to stop waiting.
Then the user runs `/clear` and `/bip-librarian` for the next batch.

## Never

- Post on GitHub.
- Run `bip s2 add`; the user adds papers through Paperpile.
- File a direction a paper suggests; that goes through the owner's `/bip-goals`.
