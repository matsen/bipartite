---
name: bip-ms-edit
description: Revise a section of a TeX manuscript interactively with Erick, line by line — structure first, then sentences, in his style. He pastes a sentence with a terse note ("clefty", "revert", "delete?"); you diagnose, fix, and quote before/after. For live co-editing, not for a sweep (/bip-ms-sweep) or mathematics (/bip-ms-math).
allowed-tools: Agent, AskUserQuestion, Bash, Read, Edit, Write
---

# /bip-ms-edit

## Usage

```
/bip-ms-edit main.tex "Introduction"
```

## Standards

Erick's writing guide is the source of truth: `/Users/matsen/writing/bio-informed-ml-genetics-perspective-tex/misc/writing_with_erick.md`.
Most manuscript repos have no copy, so read it from there at the start of every session.
It covers one sentence per line, section > paragraph > sentence > word order, topic sentences, understatement, strong actor and verb, zombie nouns, cutting words, and `%EM` comments.

He also flags **over-clever wording**, which the guide does not spell out.
Tells:

- clefts and pseudo-clefts ("What is missing is…", "It is X that…");
- set-up frames ("Two kinds of tool exist, and neither…", "The first is… The second is…");
- metaphors ("a single hand-chosen projection", "an honest sum", "the payoff is visible");
- colon reveals ("Our networks posit no generative process: …");
- fancy verbs ("posit", "couples", "rest on", "strictly contains");
- analogies that restate the point, and "X rather than Y" contrasts that add nothing.

Plain-looking sentences can still carry one of these; check each sentence, not only the showy ones.
He likes em dashes (`---`) around a list appositive in a topic sentence.

Precision beats hedging: scope a claim exactly instead of softening it.
Check every number and model description against the tables and methods, since a claim often holds for one setting only.
Watch for terminology collisions between the paper's terms and similar terms from cited work.

For an introduction: introduce background objects before any paragraph about them, prior-work paragraphs included.
"In this paper" opens the novel material, and no prior-work paragraph comes after it.
The last paragraph gives the results, then the roadmap.

## Workflow

### Step 0: Orient

Read the guide, then the whole target section with `Read`.
Read the repo's `AGENTS.md` or `CLAUDE.md`.
Check whether a `latexmk -pvc` is already watching the repo (`pgrep -fl latexmk`, then `lsof -p <pid> | grep cwd`), so you don't fight it over build files.

### Step 1: Structure pass, then wait

Propose structural changes only: paragraph order, what each paragraph claims, what to cut or move.
Use `AskUserQuestion` with previews (one line per paragraph's topic sentence) for reorder choices.
Do not touch sentences until he approves the structure; editing both at once makes him reorder twice and throws away polished prose.

After any reorder, re-read the section top to bottom and report referents left dangling ("this", "these", "that prior") and ideas now used before they are introduced.

### Step 2: Clean before showing

Before you show him any rewritten paragraph, run the over-clever tells above against your own text and the agents' text.
Flourishes that reach him cost him a round trip each.

### Step 3: Sentence loop

He pastes a sentence with a terse note.
Reply with:

1. a one-line diagnosis;
2. the fix, applied with `Edit`;
3. before and after, quoted;
4. at most two alternatives, one line each, if the choice is close.

Keep replies short; he responds to quoted sentences, not to prose about them.
"revert" undoes your last change only.
"Give feedback and clean it up" means a bulleted diagnosis, then the cleaned paragraph quoted in full.

He edits the file too: re-read the lines before every `Edit`.
Fix his typos and mention each in one line.
Follow the guide's `%EM` rule; never delete one silently.

### Citations

Search the local library first (`bip search`, or the `refs.jsonl` grep in bipartite's `AGENTS.md`), and read before citing.
When you add or keep a claim about a cited result, read the cited PDF (`bip get`, pdf-navigator) and confirm the result and the key in this pass.
Check publication dates before writing "concurrent", "recent", or "building on".
When a cut leaves a reference with no remaining citation, say so at once and offer a place for it.

### Agents

`scientific-tex-editor`, `topic-sentence-stickler`, `tex-grammar-checker`, and `tex-verb-tense-checker` give suggestions to vet, not edits to apply.
The most useful is a cold read: a `general-purpose` agent given only the section text inline, asked for ambiguous referents and places where a claim and its argument disagree.
Run Step 2 on anything an agent proposes.

### Build check

Build in a clean directory; `latexmk -outdir` can reuse a stale `main.bbl` and report no undefined citations when there are some.

```bash
d=<scratch>/build; rm -rf "$d"; mkdir -p "$d"
cp main.tex main.bib "$d"/; cp -R figures "$d"/     # plus any other inputs
cd "$d"
pdflatex -interaction=nonstopmode main.tex; bibtex main
pdflatex -interaction=nonstopmode main.tex; pdflatex -interaction=nonstopmode main.tex
grep -n "undefined\|^!" main.log
```

## Guardrails

No commit, push, or GitHub action until he says so.
