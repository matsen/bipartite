# Ideas from other projects

This page lists the outside projects that bipartite has learned from. For each one, it says what we took, where it landed, and what we left out and why.
Check it before you study an outside project, so the same project is not read twice. Add a row when an idea lands.

| Project | What it is | What we took | What we left out |
|---|---|---|---|
| Armin Ronacher, ["Agentic Coding Recommendations"](https://lucumr.pocoo.org/2025/6/12/agentic-coding/) (2025-06-12) | An essay on how agent-written codebases decay | The decay axes that `/bip-decay-audit` sweeps: greppability, component extraction, monster files, zombie code, test asymmetry | — |
| [google/elephant-goldfish-model](https://github.com/google/elephant-goldfish-model) (EGM) | A plugin in which a session with context drafts a design and fresh-context reviewers check it | The comprehension reviewer's paraphrase, compared with the author's intent: `/bip-issue-check`'s Methods sentences (#565) | Its fixed multi-stage review with a picker at every stage, its "MUST"/"STRICTLY FORBIDDEN" style, and per-harness test records. Its rules for answering review findings ("a contest must be checkable") wait for a case where an author waved off a valid finding. |
| [DietrichGebert/ponytail](https://github.com/DietrichGebert/ponytail) | A prompt for the smallest complete change, with a code-review skill | A correctness agent in `/bip-pr-review`, with four review rules: a concrete failing case for each finding, re-read to confirm it, a named shortcut is a decision, end with what was not checked (#564). The cut of "proper abstraction" from `clean-code-reviewer` (#564). | The `shortcut:` comment ledger, intensity levels, hooks for each agent harness, and its self-run benchmark numbers |
