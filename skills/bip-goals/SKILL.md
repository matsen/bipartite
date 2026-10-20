---
name: bip-goals
description: Goal-setting round for an EPIC — its epic and manuscript sessions propose a short list of directions, the user discusses them, and only then is anything filed or spawned.
---

# /bip-goals

The pair proposes, the user discusses, and nothing is filed or spawned until they have; an earlier delegation (ops, "spawn at will") does not cover the items a round proposes.
The test for each direction is whether it is the most important next step for the project, not whether it continues the pair's own line.

Run it in either session of an EPIC's pair (`/bip-epic`, `/bip-ms`); the other is the partner and checks your inputs.
Start the round in a fresh session (after `/bip-cycle`), not at the end of a long one.

1. **Name the project's goal** from the top of the EPIC body and the user's latest words on it.
   The paper's claim ledger (`grep '^%CLAIM\|^%TARGET\|^%PIECE' main.tex`) is one input, not the default target.
   The target you name is the list's first question to the user: is this what the round serves?
2. **Draft a short, ranked D-list**, working backwards from the target.
   Each direction is one line: what it moves (a claim that falls if the item is missing or wrong, or a decision the target waits on) and the one question it puts to the user.
   Item 1 says why it beats item 2, and "stop or wind down this line" is always on the list.
   Ask "which claim does this move?" before "can a worker start it today?".
   Run the PI questions (`docs/guides/pi-questions.md`) against each.
   Before proposing a run, check whether committed results already answer it.
3. **Then ask a blind, no-stake critic**: a read-only `general-purpose` subagent given the EPIC's own question, its state, the standing rulings, and the user's latest words, but not your list or your target, asked to name the target and the single most important next step, and what it would stop doing.
   Check its quotes against their source like anyone else's.
   Compare its answer with your list; where they diverge goes to the user.
   The pair agreeing is not corroboration; ask the session that consumes the output whether any item claims its output decides something it does not or assumes a criterion it lacks, and what of its work is stranded if the item is dropped.
4. **Put it to the user** as one numbered list in plain text, so they can answer by number; if they are away, escalate to `staff`.
   Record the list in the EPIC body as PROPOSED.
5. **After the discussion**, record the user's words verbatim as FINAL in the EPIC body, labelling any answer given in a dialog (picked option or typed into it) as such, since `user-said.sh` cannot find it; only then file and spawn, per `/bip-epic`.
