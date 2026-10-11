---
name: bip-goals
description: Goal-setting round for an EPIC — its epic and manuscript sessions propose a short list of directions, the user discusses them, and only then is anything filed or spawned.
---

# /bip-goals

The pair proposes, the user discusses, and nothing is filed or spawned until they have.
The test for each direction is whether it is the most important next step for the project, not whether it continues the pair's own line.

Run it in either session of an EPIC's pair (`/bip-epic`, `/bip-ms`); the other is the partner and checks your inputs.
Start the round in a fresh session (after `/bip-cycle`), not at the end of a long one.

1. **Name the target** the round serves, from the EPIC body or the latest ruling.
   Usually it is the paper's claim ledger (`grep '^%CLAIM\|^%TARGET\|^%PIECE' main.tex`); sometimes it is another target the user named, such as a pipeline rebuild.
   With no ledger or stated target, writing one is the round's first piece of work.
2. **Ask a blind, no-stake critic first**: a read-only `general-purpose` subagent given the target, the EPIC's state, and the standing rulings, but not your draft, asked for the single most important next step and what it would stop doing.
   Check its quotes against their source like anyone else's.
3. **Draft a short, ranked D-list**, working backwards from the target.
   Each direction is one line: what it moves (a claim that falls if the item is missing or wrong, or a decision the target waits on) and the one question it puts to the user.
   Item 1 says why it beats item 2, and "stop or wind down this line" is always on the list.
   Ask "which claim does this move?" before "can a worker start it today?".
   Run the PI questions (`docs/guides/pi-questions.md`) against each.
   Before proposing a run, check whether committed results already answer it.
   Compare the list with the critic's answer; where they diverge goes to the user.
   The pair agreeing is not corroboration; ask the session that consumes the output whether the draft misstates its gates.
4. **Put it to the user** as one numbered list in plain text, so they can answer by number; if they are away, escalate to `staff`.
   Record the list in the EPIC body as PROPOSED.
5. **After the discussion**, record the user's words verbatim as FINAL in the EPIC body, labelling any answer given in a dialog (picked option or typed into it) as such, since `user-said.sh` cannot find it; only then file and spawn, per `/bip-epic`.
