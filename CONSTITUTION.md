# bipartite Constitution

Development principles for the bipartite project.

## I. Agents Execute, Humans Steer

Automate execution up to the point of judgment. Workers run autonomously
through implementation, testing, and quality gates. Humans choose what
to work on and resolve genuine ambiguity (design questions, ambiguous
requirements, architectural tradeoffs).

The issue lead pattern is the model: a fresh-eyes subagent evaluates
worker progress from file-based state alone and nudges the worker
forward. It escalates to the human only when real judgment is needed.
Loop prevention (8 lead iterations max, then escalate) keeps
automation from spinning.

## II. File-Based State, No Hidden Memory

All state that matters MUST survive context compaction and session
boundaries. Use files (`.epic-status.json`, `.epic-worklog.md`, JSONL
stores), not conversation history. A fresh agent reading the files MUST
be able to pick up where the last one left off.

Corollary: if you can't reconstruct state from files, you have a bug.

## III. Skills Are the Product

Skills (slash commands) are the primary interface between humans, agents,
and the CLI. Skill docs are canonical — when CLI behavior changes, skill
docs MUST update in the same PR. Skill behavior MUST match `--help`
output exactly. New capabilities ship as skills first, CLI commands
second.

## IV. Scope Discipline

Every piece of work is tied to a GitHub issue. Workers stay within issue
scope. Adjacent discoveries become follow-up issues, not scope creep.
The issue lead enforces this by re-reading the issue body each iteration
and comparing recent commits against it.

## V. Quality Gates Over Manual Review

Automated checks (`/pr-check`, `/pr-review`) run in a loop until clean.
Fix everything they flag — don't skip, don't `--no-verify`. But the
checks themselves MUST be fast and actionable, not ceremonial.

## VI. Fail Fast, Explain Clearly

Misconfiguration, missing data, and unbuilt indexes MUST produce
immediate, actionable errors. Never silently return empty results when
the real problem is misconfiguration. Error messages MUST say what was
expected and what was received.
