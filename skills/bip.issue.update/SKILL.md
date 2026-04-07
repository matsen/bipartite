---
name: bip.issue.update
description: Re-evaluate an existing GitHub issue against current repo state, fix stale content, then run /bip.issue.check
allowed-tools: Agent, Bash, Read, Edit, Skill, Glob, Grep
---

# /bip.issue.update

Re-evaluate an existing GitHub issue against the current state of all
relevant repositories. Fix stale content (resolved blockers, landed
PRs, outdated flag names, superseded designs), then run `/bip.issue.check`
for mechanical completeness.

This is the "think, then check" complement to `/bip.issue.check`. Use
`/bip.issue.check` for a new issue that hasn't been filed yet. Use
`/bip.issue.update` for an existing issue that may have drifted from
reality.

## Usage

```
/bip.issue.update org/repo#N
/bip.issue.update 123            # uses current repo context
```

## Workflow

### Step 1: Load the issue

Parse `$ARGUMENTS` to get the repo and issue number. If only a number
is given, infer the repo from the current working directory.

```bash
gh issue view <number> --repo <org/repo> --json title,body,state,labels,updatedAt
gh issue view <number> --repo <org/repo> --comments
```

Save the body to a temp file for editing:
```bash
gh issue view <number> --repo <org/repo> --json body --jq '.body' > /tmp/issue-<number>-body.md
```

### Step 2: Gather current state

This is the key step that distinguishes `/bip.issue.update` from
`/bip.issue.check`. Examine the real world, not just the issue text.

#### 2a: Scan referenced issues and PRs

Parse the issue body for references to other issues (`#N`, `org/repo#N`)
and PRs. For each reference, check current status:

```bash
gh issue view <ref> --repo <repo> --json state,title
gh pr view <ref> --repo <repo> --json state,title,mergedAt
```

Build a table of references and their current state. Flag any that
the issue marks as "blocked" or "depends on" but are now closed/merged.

#### 2b: Check dependency repos

If the issue references other repositories (e.g., `matsengrp/phyz#812`),
pull them and check relevant state:

- Has the blocking PR/issue been merged/closed?
- Have the proposed API names (CLI flags, function signatures, file
  formats) changed in the landed implementation?
- Are there new issues or PRs in those repos that affect this issue?

For each referenced repo, if a local checkout exists:
```bash
git -C <local_path> pull --ff-only origin main 2>&1 || true
```

#### 2c: Check the issue's own repo

Pull the repo and look at recent changes:

```bash
git -C <repo_path> pull --ff-only origin main 2>&1 || true
git -C <repo_path> log --oneline -10
```

Check whether:
- Files or directories referenced in the issue exist
- Data on disk (local or remote via ssh) matches what the issue claims
- Python packages or modules referenced exist and have the expected API
- Column names, YAML formats, or other contracts match the actual code

#### 2d: Architectural review

Check whether the issue's proposed code organization aligns with the
repo's current structure:

- Does the repo have a Python package? If so, does the issue put core
  logic there or in naked scripts?
- Have new modules or patterns been established since the issue was
  written that the issue should follow?
- Has the repo's CLAUDE.md or design documentation changed?

#### 2e: Infrastructure reuse check

Search for existing infrastructure that the issue could extend instead
of building from scratch. This is a common failure mode — issues
proposing new Snakefiles, experiment directories, or pipelines when
an existing one already covers 80% of the work.

- Search merged PRs for related keywords (dataset names, method names,
  tool names):
  ```bash
  gh pr list --repo <org/repo> --state merged --search "<keywords>" --limit 20
  ```
- Search the repo for existing Snakefiles, experiment directories, or
  pipeline configs that overlap with the proposed work.
- If a substantial overlap is found, flag as **HIGH** and recommend
  extending the existing infrastructure rather than duplicating it.
  Name the specific file/directory.

### Step 3: Domain-aware review

Think through the issue at a higher level than mechanical checks:

#### Validation sufficiency

- Are the success criteria actually meaningful for the stated goal?
- Is there a baseline/null model comparison?
- Are there important failure modes that aren't checked?
- Would someone implementing this know when they're done?

#### Scope alignment

- Has the scope of the parent EPIC or project shifted?
- Are there new results or findings that change what this issue should do?
- Is any part of the issue already done (landed in another PR, completed
  as part of a different issue)?

#### Dependency freshness

- Are all "depends on" items still accurate?
- Are there new dependencies that have emerged?
- Can any "blocked" items be unblocked?

### Step 4: Apply fixes

Edit the temp file (`/tmp/issue-<number>-body.md`) to fix all
identified problems. Common fixes:

- **Resolved blockers**: Change "BLOCKED on X" to
  "~~BLOCKED on X~~ -- **resolved** by PR Y (merged DATE)"
- **Landed implementations**: Update proposed flag names, output
  formats, and API signatures to match the actual landed code
- **Completed work**: Mark tasks as done (`- [x]`) with a note
- **Stale estimates**: Update PCP counts, file sizes, runtime estimates
  based on current data
- **Code organization**: Move proposed scripts to library modules
  following the repo's established patterns
- **Missing validations**: Add concrete checks with measurable criteria

### Step 5: Run /bip.issue.check

After applying domain-aware fixes, run the mechanical checklist:

```
/bip.issue.check /tmp/issue-<number>-body.md
```

This catches anything the domain review missed: vague language,
missing data paths, unresolved placeholders, etc.

**Important**: `/bip.issue.check` will try to submit via `/bip.issue.file`.
Since we're updating an existing issue (not creating from an
`ISSUE-*.md` file), intercept before submission and instead push
the update directly:

```bash
gh issue edit <number> --repo <org/repo> --body-file /tmp/issue-<number>-body.md
```

### Step 6: Report

Summarize:
- **Staleness**: How many references were out of date (resolved blockers,
  landed PRs, changed APIs)
- **Domain fixes**: Architectural changes, validation additions, scope
  adjustments
- **Mechanical fixes**: From `/bip.issue.check` (grouped by severity)
- The updated issue URL
- Any remaining open questions for the user
