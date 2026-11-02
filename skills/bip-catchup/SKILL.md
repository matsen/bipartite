---
name: bip-catchup
description: The user is back in this session and wants to catch up and decide — print the decisions waiting on them (numbered, recommendation first), what changed since their last message as results, what is running, and what awaits their review. Answers only from this session's conversation and its continuation file.
---

# /bip-catchup

The user has come back to this session and wants to know what needs them.
Answer from this conversation, plus `_ignore/CONTINUE-<role>.md` when a `/clear` or compaction may have dropped something (`docs/guides/continuation-prompt.md`). Don't scan the fleet; staff does that.

## Print

Under these headings, in this order. Leave out an empty section, so a session with nothing pending says so in one line.

1. **Decisions waiting on you.** Every question this session asked the user that has no answer yet, including ones buried under peer messages. Number them. Each opens with the recommendation, then one line of why. Check each against the durable record first (an issue comment, the EPIC body, staff's rulings), and drop one that a recorded ruling already answers.
2. **What changed** since the user's last typed message, as results ("#530 is merged", "the gate fired: …"), not as a log of messages.
3. **Running**: each job, remote run or peer task, and when it reports.
4. **Waiting for your review**: links, such as PRs, served decks, or published pages.

Then stop and wait.

## When the user answers

The user answers by number (`2: yes`). In the same turn, for each answer:

- Record it where that decision belongs, quoting the user's words: an issue or PR comment, the EPIC body, or staff's rulings log. Where nothing fits, it stays in the conversation, and the next tuckin carries it.
- Act on it, and drop it from the list.

An answer that doesn't map to one numbered decision is not a decision: ask which number it answers.
