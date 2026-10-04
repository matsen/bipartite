---
name: bip-issue-check
description: Review an issue markdown file for completeness, then submit via /bip-issue-file
allowed-tools: Agent, Bash, Read, Edit, Skill
---

# /bip-issue-check

Review a GitHub issue markdown file for implementation-readiness, fix gaps, then submit via `/bip-issue-file`.

```
/bip-issue-check ISSUE-feature-name.md
```

## Step 1: Find the file

Use `$ARGUMENTS`, else the ISSUE-*.md most recently discussed; if unclear, ask.

## Step 2: Spawn a review subagent

Launch a general-purpose subagent on the issue file, passing the repo's `CONSTITUTION.md` and `DESIGN.md` if they exist.
It reads every file, flag, and PR the issue references, and checks claims against them rather than against the text.

**The bar:** a worker who has never seen the conversation can implement this without asking a question, and can tell when it is done.
That means concrete paths, formats, and formulas; every named quantity tied to the code or formula that computes it; measurable success criteria with a stated baseline; and, for algorithmic work, at least two validations specific to this method that would catch a wrong implementation.
A constitution violation is **CRITICAL**; a `DESIGN.md` conflict, or anything that forces the worker to guess or choose between two sections, is **HIGH**.

**Could it be smaller?** Ask what the smallest change is that delivers the goal, and what in the proposal could be cut, deferred, or replaced by something that already exists — a helper, pipeline, or Snakefile in the repo (search merged PRs and the tree) — and what cheap check, run first, could make part of it unnecessary.
When the issue extends existing code, spawn the `code-reuse-reviewer` agent on the files it names as integration points.
Flag a cut that would deliver the same goal as **HIGH**; checks the issue lacks are proposals to weigh, not gaps (`CONSTITUTION.md` Article VII).

**Duplicate issues.** Search open issues on 3–5 key terms from the draft (`gh issue list --state open --search "<terms>" --limit 15`) and read the body of every plausible match. An open issue with the same core deliverable, or one that already contains this work as a subtask, is **HIGH**: recommend consolidating, an explicit scope split with cross-references, or closing one. Related work with a clearly different goal is **MEDIUM**, worth a link.

**Project traps** — each looks fine on reading and isn't:

- **Flag liveness.** For every CLI flag or `--param` the issue arms, check that (a) it parses — run it; a struct field or YAML-only setting is not necessarily a flag; (b) it reaches the targeted code path — trace it to its consumer, not its declaration; (c) it is not already the default; (d) for a disable arm, no sibling path implements the same behaviour and survives the disable — a partial disable produces a refutation that looks like a real one. An arm byte-identical to baseline is a dead knob, not a null result. **HIGH.**
- **Untracked paths.** Every in-repo path the issue references must be tracked on the branch readers have (`git ls-files --error-unmatch <path>`, `git status --porcelain <path>`). Fix by inlining the load-bearing content — snippets, numbers, the commands behind "ad-hoc" figures; ask the user only when that would be pages long. **HIGH.**
- **Stale draft.** If the file predates this session, find the commit or binary each quoted measurement was taken on and run `git diff <commit> HEAD -- <source files>` (not a `log` range: a binary's build commit need not be an ancestor of HEAD); if anything there could move the number, re-measure rather than re-pin. A gate on byte-identity of gitignored output is not a gate. **HIGH.**
- **Content in flight.** When the draft quotes a file an open PR is changing, name the PR and the obligation instead of the content. Find those PRs with `gh pr list --state open --limit 200 --json number,files -q '.[] | select(any(.files[]; .path == "<path>")) | .number'` — `--search "<path>"` matches PR text, not changed files, the default page of 30 truncates silently, and on a repo with no open PRs the result is empty whether or not it works, so prove it once against `--state all`. **MEDIUM.**

Apply `PROSE-DISCIPLINE.md` and `EVIDENCE-DISCIPLINE.md` (bipartite repo root); violations are **MEDIUM**, **HIGH** when they obscure the deliverable.

## Step 3: Fix gaps

If you wrote the issue earlier in this session, edit it directly.
Otherwise (user-authored, from GitHub, or from a previous session), present the findings by severity with a proposed fix for each, and **wait for the user** to say which to apply.
For more than a few findings, write them to a temp file and open it (`tmux display-popup -w 80% -h 80% -E -- less <file>` under tmux).

Fix with concrete detail, not placeholders, and unwrap hard-wrapped paragraphs to one line each.

## Step 4: Submit and report

Run `/bip-issue-file <file_path>`, then report the gaps found and fixed by severity, the issue URL, and any open questions for the user.
