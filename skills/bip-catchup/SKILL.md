---
name: bip-catchup
description: The user is back in this session and wants to catch up and decide — print the decisions waiting on them (numbered, recommendation first), what changed since their last message as results, what is running, and what awaits their review. Answers from this session's own conversation and records, not the fleet's.
---

# /bip-catchup

The user has come back to this session and wants to know what needs them.
Answer from this conversation, plus this session's continuation file (the one `/bip-continue` would read) when a `/clear` or compaction may have dropped something. Don't scan the fleet; staff does that.

## Print

Under these headings, in this order. Leave out an empty section, so a session with nothing pending says so in one line.

1. **Decisions waiting on you.** Every question this session asked the user that has no answer yet, including ones buried under peer messages. Number them. Each opens with the recommendation, then one line of why; a question asked neutrally gets a recommendation now. Check each against the record it belongs in (an issue or PR comment, the EPIC body), and drop one that a recorded ruling already answers. A peer message that asks something of the user is a decision here too.
2. **What changed** since the user's last typed message, as results ("#530 is merged", "the gate fired: …"), not as a log of messages. A peer message that carries a result goes here.
3. **Running**: each job, remote run or peer task, and the clock time it should report.
4. **Waiting for your review**: links, such as PRs, served decks, or published pages.

Then stop and wait.

## When the user answers

The user answers by number. `2: yes` accepts decision 2's recommendation; any other answer to 2 replaces it. A bare answer maps to the only open decision, if there is one. In the same turn, for each answer:

- Record it where that decision belongs, quoting the user's words: an issue or PR comment, or the EPIC body. Where nothing fits, it stays in the conversation, and the next tuckin carries it.
- Act on it, and drop it from the list.

An answer that could fit more than one decision is not a decision yet: ask which number it answers.
