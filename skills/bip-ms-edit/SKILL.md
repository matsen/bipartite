---
name: bip-ms-edit
description: Edit a TeX manuscript or blog post live with Erick, line by line, in his style — "pull and evaluate" a coauthor's commit, "let's work on the abstract/intro/paragraph X", or vet a coauthor's rewording. Structure first, then sentences; apply-and-show each edit; keep %EM notes and the response letter in sync; commit only when asked. Not a batch punch list (/bip-ms-sweep) or a mathematics revision (/bip-ms-math).
allowed-tools: Agent, AskUserQuestion, Bash, Read, Edit, Write, Skill, mcp__pdf-navigator__*
---

# /bip-ms-edit

## Usage

```
/bip-ms-edit main.tex "Introduction"
/bip-ms-edit pull and evaluate
```

Erick's writing standards are in `STANDARDS.md` next to this file.
Read it at the start of every session; it replaces the `misc/writing_with_erick.md` copied into paper repos.

**Blog posts** (`~/writing/matsenweb`): also read `posts_prep/blog_voice_with_erick.md`, whose register ladder says which `STANDARDS.md` rules the blog relaxes; social posts follow `socials/CLAUDE.md`.
Skip the TeX-only steps (build check, `%EM` notes, response letter), and check every number against the paper the post describes.

## Step 0: Orient

Read `STANDARDS.md`, the repo's `AGENTS.md` or `CLAUDE.md`, and the whole paper or post with `Read`, not only the target section: you can't judge what an introduction must set up, or whether its promises are kept, without the Results and Discussion.
Find the response letter if there is one (`ls response*`).
Check whether a watcher is already building the repo, usually `make cont`: `pgrep -fl "make cont|latexmk -pvc"`, then `lsof -a -p <pid> -d cwd` for each.
If one is running, read build results from the repo's `main.log` after it rebuilds.
Never build yourself unless Erick says to; ask when you need a build and no watcher is running.

Ask how refined the text is.
A refined draft gets only fixes to what is broken, and for each change you must be able to say what was wrong with the original.

**Pull and evaluate**: `git pull`, read the coauthor's commit with `git show --word-diff=plain`, and check its build (watcher log, or ask).
Report in plain terms what changed and what is wrong (build, leftover TODOs, one sentence per line, the change's place in the paragraph's logic, prose), then propose a rewrite.

## Step 1: Structure first, then wait

Propose structural changes only: paragraph order, what each paragraph claims, what to cut or move.
Use `AskUserQuestion` with previews (one line per paragraph's topic sentence) for reorder choices.
Don't touch sentences until he approves the structure.

After any reorder, re-read the section top to bottom and report dangling referents and ideas used before they are introduced.

## Step 2: The edit loop

When he pastes a sentence with a terse note ("clefty", "However, ?", "delete?"), or when you propose a change:

1. diagnose in one line;
2. apply your recommended fix with `Edit`;
3. quote before and after;
4. give at most two alternatives, one line each, if the choice is close.

Apply small prose edits without asking, provided you can say what was wrong with the original, since the tree stays uncommitted and "revert" undoes your last change.
Ask first only for structural changes, cuts of whole passages, and anything outward-facing.
Keep replies short; he responds to quoted sentences, not to prose about them.
"Give feedback and clean it up" means a bulleted diagnosis, then the cleaned paragraph quoted in full.

Before showing any text you or an agent wrote, check it against `STANDARDS.md`, in particular:

- **Over-clever wording**: every tell in that section.
- **Overclaim**: does this claim also apply to the methods the paper favors, and does any word promise more than you believe?
- **Referents**: name the antecedent of every "this X", "it", "both", and "them".
- **Clauses**: split a sentence with an interrupting aside plus a trailing clause.
- **Size**: a side idea gets one sentence or none.

When a coauthor objects to a claim, decide on the merits whether the objection hits the claim or only its wording, and present both readings before recommending.
Don't side with either author before testing the objection against the paper's own argument.

When he proposes his own wording or asks a question ("However, ?"), answer in one line and offer to apply his version; don't overwrite it with yours.

He edits the file too: re-read the lines before every `Edit`, fix his typos, and mention each in one line.

**Keep in sync, in the same edit:**

- an `%EM` note about text you change;
- the response letter: grep it for any phrase you remove or change, and fix stale quotes, `[QUOTE … "X" to "Y"]` pointers, and examples;
- promises the letter makes ("we cut rhetorical turns"): check new text against them.

Before reporting a batch, audit your own diff: for each hunk, name what was wrong with the original, and revert the hunks where you can't.
After each batch, check the watcher's `main.log` for new warnings and report them.

## Citations

Follow `/bip-lit`: search the local library first, add with `bip export --bibtex --append main.bib <id>`, and ask before calling ASTA.
When you add or keep a claim about a cited result, read the cited PDF (`bip get`, pdf-navigator) and confirm the result and the key in this pass.
Check publication dates before writing "concurrent", "recent", or "building on".
When a cut leaves a reference uncited, say so at once and offer a place for it.

## Agents

**Cold read**, for key passages (abstract, topic sentences, a contested claim): a fresh `general-purpose` agent given only the passage text inline, no file access and no context.
Ask for flow, ambiguous referents, overclaims, undefined terms, and claim/argument tensions, ranked by importance, with minimal fixes only.
Use a new agent for each cold read; one that has seen an earlier version is not cold.
Continuing the same agent with `SendMessage` is fine for a "did this fix it?" check.

**Specialists** (`scientific-tex-editor`, `topic-sentence-stickler`, `tex-grammar-checker`, `tex-verb-tense-checker`): fan out in parallel on a whole section once its structure is settled.
Triage their output against `STANDARDS.md` and show him only what survives; never apply it wholesale.
Their fact checks are valuable, but their style suggestions overreach on refined text; accept a style edit only when you can name what was broken.

## Build check

Only when Erick asks for a build, and no watcher is running, build in a clean copy of the repo; `latexmk -outdir` in the repo can reuse a stale `main.bbl` and report no undefined citations when there are some.
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

Commit and push only when he says so.
Then give the GitHub compare link across the session's commits, and draft a short reply to the coauthor that includes it.
Show the draft; don't send it.
