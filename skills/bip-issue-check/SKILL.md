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
It reads every file, flag, and PR the issue references — each in the repo and branch the issue names, including another repo's unmerged PR branch (`gh api -H 'Accept: application/vnd.github.raw' 'repos/<owner>/<repo>/contents/<path>?ref=<branch>'` needs no clone) — and checks claims against them rather than against the text.

**The bar:** a worker who has never seen the conversation can implement this without asking a question, and can tell when it is done.
That means concrete paths, formats, and formulas; every named quantity tied to the code or formula that computes it; measurable success criteria with a stated baseline; and, for algorithmic work, at least two validations specific to this method that would catch a wrong implementation.
A constitution violation is **CRITICAL**; a `DESIGN.md` conflict, or anything that forces the worker to guess or choose between two sections, is **HIGH**.

**Could it be smaller?** Ask what the smallest change is that delivers the goal, and what in the proposal could be cut, deferred, or replaced by something that already exists — a helper, pipeline, or Snakefile in the repo (search merged PRs and the tree) — and what cheap check, run first, could make part of it unnecessary.
When the issue extends existing code, spawn the `code-reuse-reviewer` agent on the files it names as integration points.
Flag a cut that would deliver the same goal as **HIGH**; checks the issue lacks are proposals to weigh, not gaps (`CONSTITUTION.md` Article VII).

**The methods-section test.** Write the proposed design as the two or three sentences it would get in a paper's Methods.
If they need a caveat, a bespoke threshold, a proxy for the quantity of interest, or a chain of stages a reader would ask "why?" about, find how published work does the same thing (`/bip-lit`: the local library, then ASTA, both read-only and open to any agent) and propose that plainer design.
A plainer design that answers the same question is **HIGH**.

**Duplicate issues.** Search issues on 3–5 key terms from the draft (`gh issue list --state all --search "<terms>" --limit 30`) and read the body of every plausible match. An open issue with the same core deliverable, one that already contains this work as a subtask, or closed work whose result already answers part of the deliverable is **HIGH**: recommend consolidating, an explicit scope split with cross-references, or closing one — or, for closed work, narrowing the draft to what remains and citing it. Related work with a clearly different goal is **MEDIUM**, worth a link.

**Prior answer.** For an issue that commissions a measurement or experiment, state what theory or published work predicts for its outcome, and whether that prediction transfers to this regime. Look first in every manuscript under `paths.writing` (in `$NEXUS_PATH/config.yml`, default `~/writing`), not only those that track this repo, since a paper's literature often sits in a sibling manuscript: grep each `main.tex` for the topic and read the `\cite` and `%PROV` beside each hit, which rule on what the citation supports, then grep their `notes/`. Then use `/bip-lit`; a miss in the manuscripts is not evidence that no prior answer exists. A confident prediction that transfers is cited, and the issue shrinks to measuring its size; one that may not transfer says why, and the result is the evidence. **HIGH** when a manuscript already cites the answer, otherwise **MEDIUM**.

**Project traps** — each looks fine on reading and isn't:

- **Flag liveness.** For every CLI flag or `--param` the issue arms, check that (a) it parses — run it; a struct field or YAML-only setting is not necessarily a flag; (b) it reaches the targeted code path — trace it to its consumer, not its declaration; (c) it is not already the default; (d) for a disable arm, no sibling path implements the same behaviour and survives the disable — a partial disable produces a refutation that looks like a real one; (e) list every behaviour the arm's flag or input switches; each one other than the named factor must be isolated by another arm or split out in the readout. An arm byte-identical to baseline is a dead knob, not a null result. **HIGH.**
- **Untracked paths.** Every in-repo path the issue references must be tracked on the branch readers have (`git ls-files --error-unmatch <path>`, `git status --porcelain <path>`). A path only one session or host can see, or that a cleanup removes (helper state, `/tmp`, a scratchpad, a pooled clone's working files), fails by definition. Durable shared storage (e.g. `/fh/fast`) passes when cited by exact path with a host that can read it, plus a checksum when a quoted number depends on the file's content. Fix by inlining the load-bearing content — snippets, numbers, the commands behind "ad-hoc" figures — or, when that would be pages long, by putting the artifact, with its checksums, on durable shared storage or where a repo keeps such artifacts (e.g. an experiment's tracked `results/`) within that repo's data rules, and citing it; if neither fits, ask the user. **HIGH.**
- **Stale draft.** If the file predates this session, find the commit or binary and the exact command and flags behind each quoted measurement — numbers from different flags are different code paths — and run `git diff <commit> HEAD -- <source files>` (not a `log` range: a binary's build commit need not be an ancestor of HEAD); if anything there could move the number, re-measure rather than re-pin. A gate on byte-identity of gitignored output is not a gate. **HIGH.**
- **Content in flight.** When the draft quotes a file an open PR is changing, name the PR and the obligation instead of the content. Find those PRs with `gh pr list --state open --limit 200 --json number,files -q '.[] | select(any(.files[]; .path == "<path>")) | .number'` — `--search "<path>"` matches PR text, not changed files, and the default page of 30 truncates silently. An empty result counts only once `gh pr list --state all --limit 5 --json number` on the same repo returns PRs; report that you ran it. **MEDIUM.**

Apply `PROSE-DISCIPLINE.md` and `EVIDENCE-DISCIPLINE.md` (bipartite repo root); violations are **MEDIUM**, **HIGH** when they obscure the deliverable.

## Step 3: Fix gaps

If you wrote the issue earlier in this session, edit it directly.
Otherwise (user-authored, from GitHub, or from a previous session), present the findings by severity with a proposed fix for each, and **wait for the user** to say which to apply.
For more than a few findings, write them to a temp file and open it (`tmux display-popup -w 80% -h 80% -E -- less <file>` under tmux).

Fix with concrete detail, not placeholders, and unwrap hard-wrapped paragraphs to one line each.

## Step 4: Submit and report

Run `/bip-issue-file <file_path>`, then report the gaps found and fixed by severity, the issue URL, and any open questions for the user.
