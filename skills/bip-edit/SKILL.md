---
name: bip-edit
description: Edit prose live with the author, line by line — a TeX manuscript or a blog post. "Pull and evaluate" a coauthor's commit, "let's work on the abstract/intro/paragraph X", or vet a suggested rewording. Structure first, then sentences; apply-and-show each edit; keep margin notes and the response letter in sync; commit only when asked. Not a batch punch list (/bip-ms-sweep) or a mathematics revision (/bip-ms-math).
allowed-tools: Agent, AskUserQuestion, Bash, Read, Edit, Write, Skill, SendMessage, mcp__pdf-navigator__*
---

# /bip-edit

## Usage

```
/bip-edit main.tex "Introduction"
/bip-edit <post>.md
/bip-edit pull and evaluate
```

## Style source

Style comes from the repo, not from this skill.
Read the repo's `CLAUDE.md`, `AGENTS.md`, and `.claude/rules/`, and every style guide or agent they name (for a TeX paper, usually `scientific-tex-editor`).
Where they and this skill disagree, the repo wins.
If they name no style source, say so once and apply only the checks below.

## Step 0: Orient

Read the whole document with `Read`, not only the target section: you can't judge what an introduction must set up, or whether its promises are kept, without the rest.
Find the response letter if there is one (`ls response*`).

Ask how refined the text is.
A refined draft gets only fixes to what is broken.
Once its structure is settled (Step 1), your first pass over a refined draft is one batch: fix what is broken, audit it, and show the author `git diff --word-diff` before starting the edit loop.

**Not a TeX manuscript** (a blog post, say): skip the build, `%XX`, and response-letter steps.
Verify every number and claim against the source the text reports on, and ask where that source is if the repo doesn't say.

**Builds.** Check whether a background build is running, usually `make cont`: `pgrep -fl "make cont|latexmk -pvc"`, then `lsof -a -p <pid> -d cwd` for each.
If one is, read results from the repo's `main.log` after it rebuilds.
Never build yourself unless the author says to (recipe below).

**Pull and evaluate**: `git pull`, read the coauthor's commit with `git show --word-diff=plain`, and read its build log if a background build is running.
Report plainly what changed and what is wrong (build, leftover TODOs, one sentence per line, the change's place in the paragraph's logic, prose), then propose a rewrite.

## Step 1: Structure first, then wait

Propose structural changes only: section and paragraph order, what each paragraph claims, what to cut or move.
Use `AskUserQuestion` with previews (one line per paragraph's topic sentence) for reorder choices.
Don't touch sentences until the author approves the structure; editing both at once throws away polished prose.

After any reorder, re-read the section top to bottom and report dangling referents and ideas used before they are introduced.

## Step 2: The edit loop

When the author pastes a sentence with a terse note ("clefty", "delete?"), or when you propose a change:

1. diagnose in one line;
2. apply your recommended fix with `Edit`;
3. quote before and after;
4. give at most two alternatives, one line each, if the choice is close.

Apply small prose edits without asking, provided you can say what was wrong with the original; the tree stays uncommitted and "revert" undoes your last change.
Ask first for structural changes, cuts of whole passages, and anything outward-facing.
When the author proposes a wording or asks a question ("However, ?"), answer in one line and offer to apply their version; don't overwrite it with yours.
"Give feedback and clean it up" means a bulleted diagnosis, then the cleaned paragraph quoted in full.
Keep replies short; the author responds to quoted sentences, not to prose about them.

Before showing any text you or an agent wrote, check it against the style source and these:

- **Overclaim**: does the claim also apply to the methods the paper favors, and does any word promise more than the evidence supports?
  Scope a claim exactly instead of hedging it.
- **Numbers**: check each against the tables, methods, or the paper a post describes; a claim often holds for one setting only.
- **Referents**: name the antecedent of every "this X", "it", "both", and "them"; repeat the exact noun phrase rather than a synonym.
- **Clauses**: split a sentence with an interrupting aside plus a trailing clause.
- **Size**: late in revision, add less; a side idea gets one sentence or none.
- **Attribution**: never put a feeling or claim in the author's mouth that they didn't state.
- **Caveats**: fix a misleading implication in one clause, framed as a finding where that is honest, and describe a baseline by what it does, not by judging it.

When a coauthor objects to a claim, decide on the merits whether the objection hits the claim or only its wording, and present both readings before recommending.

The author edits the file too: re-read the lines before every `Edit`, fix their typos, and mention each in one line.

Before reporting a batch, audit your diff against the last commit (`git diff`), not against your previous edit: for each hunk, name what was wrong with the original, and revert the hunks where you can't.

**Keep in sync, in the same edit:**

- margin comments tagged with initials (`%XX`): never delete one yourself; when an edit addresses one, offer to delete it, and update a note you added this session when you change the text it describes;
- the response letter: grep it for any phrase you remove or change, and fix stale quotes, pointers, and examples;
- promises the letter makes ("we cut rhetorical turns"): check new text against them.

## Citations

Follow `/bip-lit`: search the local library first and add with `bip export --bibtex --append main.bib <id>`.
Don't cite or characterize work that no author has read.
When you add or keep a claim about a cited result, read the cited PDF (`bip get`, pdf-navigator) and confirm the result and the key in this pass.
Check publication dates before writing "concurrent", "recent", or "building on".
When a cut leaves a reference uncited, say so at once and offer a place for it.

## Agents

**Cold read**, for key passages (abstract, topic sentences, a contested claim): a fresh `general-purpose` agent given only the passage text inline, no file access and no context.
Ask for flow, ambiguous referents, overclaims, undefined terms, and claim/argument tensions, ranked by importance, with minimal fixes only.
Use a new agent for each cold read; one that has seen an earlier version is not cold.
Continuing the same agent with `SendMessage` is fine for a "did this fix it?" check.

**Specialists** (`scientific-tex-editor`, `topic-sentence-stickler`, `tex-grammar-checker`, `tex-verb-tense-checker`): fan out in parallel on a whole section once its structure is settled.
Triage their output and show the author only what survives; never apply it wholesale.
Their fact checks are valuable, but their style suggestions overreach on refined text.

## Build recipe

Only when the author asks for a build and no background build is running, build in a clean copy of the repo; `latexmk -outdir` in the repo can reuse a stale `main.bbl` and report no undefined citations when there are some.
Use absolute paths, not `cd`.

```bash
d=<scratch>/build; rm -rf "$d"; mkdir -p "$d"
rsync -a --exclude .git --exclude '*.aux' --exclude '*.bbl' --exclude '*.fdb_latexmk' \
  --exclude '*.fls' --exclude '*.blg' --exclude '*.log' --exclude main.pdf <repo>/ "$d"/
latexmk -pdf -cd -interaction=nonstopmode "$d/main.tex"; echo "latexmk exit $?"
grep -n "undefined\|^!" "$d/main.log"
```

A nonzero exit is a failure even if the grep is clean; after a bibtex error `main.log` can be stale or missing.

## Wrap-up

Don't rename or move the file while iterating, since that breaks the diff; rename last, in its own commit.
Commit and push only when the author says so.
Then give the GitHub compare link across the session's commits, and draft a short reply to the coauthor that includes it.
Show the draft; don't send it.

Lessons about the author's taste go to the style source the repo names, not into this skill.
