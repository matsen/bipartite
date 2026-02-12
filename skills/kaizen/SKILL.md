---
name: kaizen
description: Reflect on what just happened and propose a concrete improvement to skills, CLAUDE.md, docs, or workflow.
---

# /kaizen

Reflect on friction or confusion from the current session and propose a concrete improvement.

## Context

"Kaizen" means continuous improvement. After any interaction where something didn't go smoothly—agent couldn't find files, a skill was confusing, documentation was missing, a workflow was clunky—the user can invoke `/kaizen` and the agent will:

1. Diagnose what went wrong (or what could be better)
2. Propose a specific, actionable fix
3. Optionally implement the fix

## Usage

```
/kaizen
/kaizen "the bip search kept failing because the schema was stale"
```

`$ARGUMENTS` is an optional hint about what to improve. If empty, infer from conversation context.

## Workflow

### Step 1: Diagnose

Review the conversation history for friction points. Look for:

- **Confusion**: Agent searched for files in the wrong place, couldn't find source code, misunderstood project structure
- **Failed commands**: CLI errors, missing flags, wrong syntax, stale schemas
- **Missing context**: Agent didn't know about a convention, config location, or workflow step
- **Skill gaps**: A skill gave bad advice, was missing a step, or had outdated information
- **Repetitive work**: Something the agent had to figure out repeatedly that should be documented
- **Workflow friction**: Too many manual steps, missing automation, unclear handoffs

If `$ARGUMENTS` is provided, focus the diagnosis there. Otherwise, scan the full conversation.

Identify the **root cause**, not just the symptom. For example:
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

### Step 3: Propose the fix

Present the diagnosis and proposed fix clearly:

```
## Kaizen: [short title]

**What happened**: [1-2 sentence description of the friction]

**Root cause**: [Why it happened]

**Proposed fix**: [What to change and where]

**Target**: [CLAUDE.md / skill / code / issue / etc.]
```

Then show the specific change—either as a diff, a new section to add, or a description of the code change.

### Step 4: Ask before acting

**STOP and ask the user** before making any changes. Present options:

1. **Apply now** — Make the edit directly (for small CLAUDE.md or skill changes)
2. **Create a PR** — For changes to the bipartite repo (skills, code)
3. **Write an issue** — For larger improvements that need discussion
4. **Skip** — User disagrees with the diagnosis

### Step 5: Implement (if approved)

Based on user choice:

**Apply now** (CLAUDE.md, skill, or memory edits):
- Edit the target file directly
- For bipartite skill changes, the working tree is at `/Users/matsen/re/bipartite`

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
cd /Users/matsen/re/bipartite
git pull origin main
git checkout -b kaizen/<short-description>
# Make the changes
git add <files>
git commit -m "kaizen: <description>"
gh pr create --title "kaizen: <description>" --body "..."
```

**Write an issue** (using /issue-file pattern):
- Create `ISSUE-kaizen-<topic>.md` in the current repo
- Use the `/issue-file` workflow to submit

## Guidelines

- **Be specific**: "Add database path to CLAUDE.md" not "improve documentation"
- **Be minimal**: Propose the smallest change that fixes the problem. Don't refactor adjacent code
- **One improvement per invocation**: If you see multiple issues, pick the highest-impact one and mention the others briefly
- **Respect existing structure**: Follow the conventions already in CLAUDE.md and skill files
- **Bipartite repo awareness**: Skills live in `/Users/matsen/re/bipartite/skills/`. The repo is `matsen/bipartite` on GitHub
- **Don't over-document**: If something is obvious from the code, it doesn't need a CLAUDE.md entry. Only document things the agent genuinely couldn't figure out on its own
