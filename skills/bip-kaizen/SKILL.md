---
name: bip-kaizen
description: Reflect on what just happened and propose a concrete improvement to skills, CLAUDE.md, docs, or workflow.
---

# /bip-kaizen

Reflect on friction or confusion from the current session and propose a concrete improvement.

## Context

"Kaizen" means continuous improvement.
After any interaction where something didn't go smoothly—agent couldn't find files, a skill was confusing, documentation was missing, a workflow was clunky—the user can invoke `/bip-kaizen` and the agent will:

1. Diagnose what went wrong (or what could be better)
2. Propose a specific, actionable fix
3. Optionally implement the fix

## Usage

```
/bip-kaizen
/bip-kaizen "the bip search kept failing because the schema was stale"
```

`$ARGUMENTS` is an optional hint about what to improve.
If empty, infer from conversation context.

## Workflow

### Step 1: Diagnose

Review the conversation history for friction points.
Look for:

- **Confusion**: Agent searched for files in the wrong place, couldn't find source code, misunderstood project structure
- **Failed commands**: CLI errors, missing flags, wrong syntax, stale schemas
- **Missing context**: Agent didn't know about a convention, config location, or workflow step
- **Skill gaps**: A skill gave bad advice, was missing a step, or had outdated information
- **Repetitive work**: Something the agent had to figure out repeatedly that should be documented
- **Workflow friction**: Too many manual steps, missing automation, unclear handoffs

If `$ARGUMENTS` is provided, focus the diagnosis there.
Otherwise, scan the full conversation.

Identify the **root cause**, not just the symptom.
For example:
- Symptom: "Agent couldn't find the database" → Root cause: Database path not documented in CLAUDE.md
- Symptom: "bip search returned errors" → Root cause: Skill doc doesn't mention `bip rebuild` as a first step

### Step 2: Classify the improvement target

Determine where the fix belongs:

| Target | When | Example |
|--------|------|---------|
| **CLAUDE.md** (current project) | Agent lacked project-specific context | Missing build command, file path, convention |
| **CLAUDE.md** (global `~/.claude/CLAUDE.md`) | Agent lacked cross-project context | Personal workflow preference, tool config |
| **Skill file** (bipartite `skills/`) | A `/skill` gave wrong or incomplete guidance | Missing flag, outdated workflow, bad example |
| **Auto-memory** (`~/.claude/projects/*/memory/`) | Pattern worth remembering but not suitable for CLAUDE.md | Debugging insight, one-off workaround |
| **Code/docs in current repo** | Missing README, help text, or inline docs | CLI `--help` text doesn't match behavior |
| **Code in bipartite repo** | Bug or missing feature in a bip command or skill | Skill needs new step, CLI needs better error message |
| **New skill** | Repeated workflow that should be a `/command` | Multi-step process done manually every time |
| **GitHub issue** | Improvement too large for a quick fix | Needs design discussion, multi-file refactor |

### Step 2b: Find the rule you are about to collide with

**Mandatory, and skipping it is the single most common way kaizen makes things worse.** This table tells you where a fix *could* go; it does not tell you whether guidance on this already exists somewhere, possibly saying something different. Before writing anything, `grep` for the rule's existing home:

```bash
cd ~/re/bipartite
grep -rn '<the key term>' skills/ *.md          # every restatement, not just the obvious one
```

Then state one of these explicitly in your proposal:

- **"Supersedes: `<file>` `<section>`"** — an existing rule is wrong or narrower than reality. **Edit it in place.** Do not add a second rule beside it; two rules on one topic have no tiebreak, and a later reader gets whichever they hit first.
- **"Amends: `<file>` `<section>`"** — the rule is right but incomplete. Extend that passage; don't start a new one elsewhere.
- **"No existing rule"** — you grepped and found nothing. Say so, so a reviewer can check the claim.

Then list **every other file that restates the rule**, and update all of them or say why not. A fix that lands in one of two copies is worse than no fix: the un-updated copy is now authoritative-looking and stale.

Two failures this exists to prevent, both measured 2026-09-01 in one session:
- An addressing rule was strengthened in `bip-conductor` but not in `bip-conductor-poll` — which is *deliberately* the copy a mid-cycle conductor reads. The conductor that made the original error was mid-cycle.
- A generalized gitignore rule in `bip-conductor` never reached the concrete checklist in `bip-conductor-spawn`, so the checklist kept omitting a file that "has bitten a real fleet twice".

**Budget discipline.** If you are adding a rule and deleting nothing, the question is not how many lines — it is **what does this supersede?** Answer it in the proposal, or say explicitly that nothing does and why the fact is new. That question, not any ratio, is what catches a rule being re-litigated in a second file.

Report the size of **your own change** alongside it, in words, because this corpus is written one paragraph per line and a line count measures newlines rather than context:

```bash
git diff HEAD -- '*.md' | grep '^+' | grep -v '^+++' | wc -w   # words you are adding
```

No threshold. A number with a cutoff invites an argument about the cutoff; a number stated next to "what does this supersede?" does not.

**This paragraph previously set a 3:1 line ratio and was wrong three ways, all caught in review of the diff that introduced it** — worth keeping because it is the same failure the rule is about. Its check used `git diff` with no ref, so a *staged* change reported `+0/-0`, fail-open at exactly the moment the instruction said to run it. Its trend command divided by zero — `awk: fatal`, exit 2 — when nothing had been deleted, the precise condition it existed to detect, a crash already catalogued in `EVIDENCE-DISCIPLINE.md`. And measured on its own branch, one file added **782 words in 6 lines and passed** at exactly 3.0:1 while another added 357 words in 42 lines and failed worst: the threshold rewarded the writing style that costs the most context.

**Before compressing a rule, check that anyone can cite an instance of it.** Some of what accumulates is anticipatory — written against a failure nobody has had. That is a candidate for deletion, not for shortening, and a pass that only shortens will preserve it while cutting a rule whose evidence was merely already thin. The compression pass is `matsen/bipartite#218`; its reviewer brief is `skills/lib/compression-reviewer-prompt.md`.

### Step 3: Propose the fix

Present the diagnosis and proposed fix clearly:

```
## Kaizen: [short title]

**What happened**: [1-2 sentence description of the friction]

**Root cause**: [Why it happened]

**Proposed fix**: [What to change and where]

**Target**: [CLAUDE.md / skill / code / issue / etc.]

**Supersedes / Amends**: [`file` `section`, or "no existing rule — grepped `<term>`"]

**Other restatements**: [every other file stating this rule, and whether you updated it — or "none found"]

**Net lines**: [+N / -M. If you are adding and deleting nothing, justify it.]
```

Then show the specific change—either as a diff, a new section to add, or a description of the code change.

### Step 4: Ask before acting

**STOP and ask the user** before making any changes.
Present options:

1. **Apply now** — Make the edit directly (for small CLAUDE.md or skill changes)
2. **Create a PR** — For changes to the bipartite repo (skills, code)
3. **Write an issue** — For larger improvements that need discussion
4. **Skip** — User disagrees with the diagnosis

### Step 5: Implement (if approved)

Based on user choice:

**Apply now** (CLAUDE.md, skill, or memory edits):
- Edit the target file directly
- For bipartite skill changes, the working tree is at `~/re/bipartite`

> **That working tree is shared, and it is the one at `~/re/bipartite` for every session on the machine.** There is no per-session checkout and no locking: two Claude sessions running kaizen are editing the same bytes on disk. `git` never reports a conflict, because there is only one tree — so the failure is silent.
>
> **Before you edit**, run `git status`. Modified files you did not touch mean another session is live in here; coordinate or wait rather than writing.
> **Before you commit**, run `git diff --cached` and confirm every hunk is yours. Per-file `git add` limits the blast radius but does not close it — if the other session edited *the same file*, staging that one path still ships its work. Never `git add -A` or `git add .` here.
> **Commit a cross-file dependency in one commit.** A pointer and its target must land together.
>
> Measured 2026-09-01: two sessions each committed the other's in-flight uncommitted work under their own commit message, in opposite directions, within one hour. Both then misdiagnosed it as their own stale read — `git fetch` updates remote-tracking refs and not the working tree, which is true and was not the cause. The worst outcome was structural rather than a lost edit: one commit shipped a `bip-epic` pointer to a `PROSE-DISCIPLINE.md` section that existed only in the *other* session's uncommitted tree, so for ~20 minutes `origin/main` carried a reference to nothing and the two discriminators it replaced were unreachable. A separate `git worktree` per session would prevent the class outright — **and it was considered and declined, so don't re-propose it as an obvious fix.** The user's reasoning (2026-09-01): that day's collision rate came from an unusually intense burst of skill editing, not the steady state, and standing up a worktree pool to serve two writers who are rarely both in here is overengineering. The three guards above are the proportionate answer. Revisit only if a third concurrent writer appears.

**Create a PR** (bipartite repo changes):

The bipartite repo (`matsen/bipartite`) structure:
```
skills/          # Skill definitions (each skill is a directory with SKILL.md)
cmd/             # bip binary source (Go, spf13/cobra)
internal/        # bip binary source (Go internal packages)
docs/            # Guides and documentation
agents/          # Agent definitions
```

Skills are symlinked from `skills/<name>/` to `~/.claude/skills/<name>` for global availability.

```bash
cd ~/re/bipartite
git pull origin main
git checkout -b kaizen/<short-description>
# Make the changes
git add <files>
git commit -m "kaizen: <description>"
gh pr create --title "kaizen: <description>" --body "..."
```

**Write an issue** (using /bip-issue-file pattern):
- Create `ISSUE-kaizen-<topic>.md` in the current repo
- Use the `/bip-issue-file` workflow to submit

## Guidelines

- **Be specific**: "Add database path to CLAUDE.md" not "improve documentation"
- **Be minimal**: Propose the smallest change that fixes the problem.
  Don't refactor adjacent code
- **One improvement per invocation**: If you see multiple issues, pick the highest-impact one and mention the others briefly
- **Respect existing structure**: Follow the conventions already in CLAUDE.md and skill files
- **Bipartite repo awareness**: Skills live in `~/re/bipartite/skills/`.
  The repo is `matsen/bipartite` on GitHub
- **Spend emphasis like it is scarce, because it is.** <!-- emphasis-budget-exempt --> ⭐ ⛔ ⚠ ➡ and ALL-CAPS BOLD are a shared budget across the file, not a per-paragraph decision. **The mechanism a budget only caps: each kaizen reaches for a louder marker than the last one it sees, so the escalation is driven by the file's current state rather than by the new passage's importance — which is why it compounds and why no author notices their own step.** The numbers: `skills/bip-conductor/SKILL.md` carried **0** such markers on 2026-09-05, 13 on 09-14, 148 on 09-18, and 214 on 09-20 — one per 5.7 lines — while the file grew 2.1x in lines over the same window. `EVIDENCE-DISCIPLINE.md` went 0 → 201 across those same two dates. A marker every few lines marks nothing. Before adding one, count what is already there and decide whether your passage outranks them:

  ```bash
  # Skips this rule's own glyphs, which are mentions, not uses. <!-- emphasis-budget-exempt -->
  grep -v 'emphasis-budget-exempt' <file> | grep -oE '⭐|⛔|⚠|➡' | wc -l
  ```

  **Expected baseline is 0** — every glyph in a clean file is a use, so any count is the real one, and this file reads 0 rather than the 8 a naive `grep` returns by counting the rule against itself. Without that exclusion the floor is inherited by every future run, and an author who genuinely cut their emphasis still sees a residue and concludes the budget is unmeetable. Then: does your marker displace one, or just join them? If the passage needs a marker to be taken seriously, the sentence is probably not carrying its own weight yet.
- **Don't over-document**: If something is obvious from the code, it doesn't need a CLAUDE.md entry.
  Only document things the agent genuinely couldn't figure out on its own
