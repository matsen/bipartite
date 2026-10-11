---
name: bip-goals
description: Goal-setting round for an EPIC — its epic and manuscript sessions propose a short list of directions, the user discusses them, and only then is anything filed or spawned.
---

# /bip-goals

The pair proposes, the user discusses, and nothing is filed or spawned until they have.

Run it in either session of an EPIC's pair (`/bip-epic`, `/bip-ms`); the other is the partner and checks your inputs.

1. **Name the target** the round serves, from the EPIC body or the latest ruling.
   Usually it is the paper's claim ledger (`grep '^%CLAIM\|^%TARGET\|^%PIECE' main.tex`), but not always: "I don't really care about the manuscript at this point, but I want to run any experiment that will inform the rebuild" (user, 2026-10-11).
   With no ledger or stated target, writing one is the round's first piece of work.
2. **Draft a short D-list**, working backwards from the target.
   Each direction is one line: what it moves (a claim that falls if the item is missing or wrong, or a decision the target waits on) and the one question it puts to the user.
   Ask "which claim does this move?" before "can a worker start it today?".
   Run the PI questions (`docs/guides/pi-questions.md`) against each.
   Before proposing a run, check whether committed results already answer it.
3. **Attack the draft with a no-stake critic**: a read-only `general-purpose` subagent given the target, the standing rulings, and the draft verbatim, asked to attack it against the issues.
   Check its quotes against their source like anyone else's.
   The pair agreeing is not corroboration.
4. **Put it to the user** as one numbered list in plain text, so they can answer by number; if they are away, page them per `/bip-conductor`'s "Paging the user".
   Record the list in the EPIC body as PROPOSED.
5. **After the discussion**, record the user's words verbatim as FINAL in the EPIC body; only then file and spawn, per `/bip-epic`.
