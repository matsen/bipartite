# Feature Specification: Narrative Digest

**Feature Branch**: `012-narrative-digest`
**Created**: 2026-01-25
**Status**: Draft
**Input**: Create a slash command that generates thematic narrative digests from GitHub activity, configured per-channel via markdown files in nexus.

## Overview

### Architecture Decision
**No LLM calls from Go.** The narrative generation uses Claude Code's LLM via a slash command, not Go code calling external APIs. Small models via Ollama for embeddings are fine, but narrative generation requires the slash command pattern.

### Prerequisite: Change `bip digest` Default Behavior

Currently `bip digest` posts to Slack by default if a webhook is configured. This is dangerous (easy to accidentally post). Change to:
- `bip digest --channel foo` → preview only, no posting
- `bip digest --channel foo --post` → actually post to Slack

The `/bip.digest` slash command should be updated to match.

### Workflow
```
/bip.narrative dasm2
/bip.narrative dasm2 --verbose --since 2w
```

1. Slash command runs `bip digest --channel dasm2` to get raw activity (no longer needs `--dry-run`)
   - With `--verbose`: `bip digest` fetches PR/issue bodies via `gh`, summarizes each using Haiku in parallel, includes summaries in output
2. Reads channel config from `nexus/narrative/dasm2.md`
3. Uses Claude Code's LLM to generate themed prose
4. Writes output to `nexus/narrative/dasm2/YYYY-MM-DD.md`

### Directory Structure (in nexus)
```
nexus/
  narrative/
    preferences.md              # shared defaults (all channels inherit)
    dasm2.md                    # channel config: themes, project-specific prefs
    loris.md                    # channel config for loris
    dasm2/
      2026-01-25.md             # generated digest
      2026-01-18.md             # older digest
    loris/
      2026-01-25.md
```

### Shared Preferences (`narrative/preferences.md`)

```markdown
# Narrative Digest Preferences

These are default preferences inherited by all channel configs. Channel-specific configs can override or extend these.

## Attribution

- Do NOT use attribution ("Will's PR", "Jared is working on") - just describe the work

## Format

- Use hybrid format: bullets for lists of independent items, prose for connected items
- Mark status with prefixes: "In progress:" for open PRs, "Open:" for issues under discussion
- End with "Looking Ahead" bullet list for open issues and in-progress PRs

## Content

- Stick to available information - do not invent technical context
  - Default: PR/issue titles only
  - With `--verbose`: include summarized PR/issue bodies
```

### Channel Config Example (`narrative/dasm2.md`)

```markdown
# dasm2 Narrative Configuration

Inherits from [preferences.md](preferences.md).

## Themes

1. **Data Prep** - External pipelines that create/process raw data (flu-usher, mat-pcp, flu-mut-fitness)
2. **Data Handling** - Internal utilities for using data (registry, formats, loaders, masking)
3. **Architecture** - Model structure (backbones, heads, attention mechanisms, embeddings)
4. **Training & Loss** - Loss functions, multi-objective training, optimization methods
5. **Evaluation & Experiments** - Benchmarks, functional datasets, model comparisons, simulation frameworks
6. **Infrastructure** - CLI commands, config, dependencies, documentation, reproducibility

## Project-Specific Preferences

- Within each theme, write separate paragraphs for viral work (flu, SARS2, H3) and antibody work when both are present
- Use **Viral:** and **General:** (or **Antibody:**) as bold subheadings within themes

## Repo Context

- **dasm2-experiments**: Main DASM2 development (all themes)
- **flu-usher**: Influenza phylogenetic tree tools (Data Prep, viral)
- **mat-pcp**: Mutation counting pipeline (Data Prep)
- **viral-dasm-experiments-1**: Viral-specific DASM experiments (viral paragraphs)
- **dasm-epistasis-experiments**: Epistasis analysis (Evaluation & Experiments)
- **flu-mut-fitness**: Influenza mutation fitness (Data Prep, viral)
```

### Example Output (`narrative/dasm2/2026-01-25.md`)

```markdown
# dasm2 Digest: Jan 18-25, 2026

## Data Prep

**Viral:**
- flu-usher: CDS extraction from tree mutations ([#8](https://github.com/matsengrp/flu-usher/pull/8)), host-specific subtree extraction ([#10](https://github.com/matsengrp/flu-usher/pull/10))
- flu-mut-fitness: adapted for influenza H3N2-HA analysis ([#1](https://github.com/matsengrp/flu-mut-fitness/pull/1))

**General:**
- mat-pcp: integration and validation series ([#42](https://github.com/matsengrp/mat-pcp/pull/42), [#40](https://github.com/matsengrp/mat-pcp/pull/40), etc.), resolved PCP vs mutation count discrepancies ([#51](https://github.com/matsengrp/mat-pcp/pull/51))

## Data Handling

- Modular dataset registry ([#176](https://github.com/matsengrp/dasm2-experiments/pull/176))
- Functional data format utilities ([#182](https://github.com/matsengrp/dasm2-experiments/pull/182))
- netam added as git dependency ([#187](https://github.com/matsengrp/dasm2-experiments/pull/187))
- In progress: user-specified problematic sites masking ([#198](https://github.com/matsengrp/dasm2-experiments/pull/198))

## Architecture

ESM2 backbone support ([#181](https://github.com/matsengrp/dasm2-experiments/pull/181)) and mutation frequency branch length initialization ([#171](https://github.com/matsengrp/dasm2-experiments/pull/171)) merged.

Open: neutral model architecture refactor ([#195](https://github.com/matsengrp/dasm2-experiments/issues/195)), lightweight transformer head for ESM2 ([#184](https://github.com/matsengrp/dasm2-experiments/issues/184)).

## Training & Loss

Joint multi-objective training with weighted loss sum ([#153](https://github.com/matsengrp/dasm2-experiments/pull/153)) enables simultaneous optimization of multiple objectives. Seed parameter now applied for reproducibility ([#179](https://github.com/matsengrp/dasm2-experiments/pull/179)).

## Evaluation & Experiments

- Functional dataset evaluation notebooks ([#189](https://github.com/matsengrp/dasm2-experiments/pull/189))
- Epistasis experiments rerun with updated DASM model ([#35](https://github.com/matsengrp/dasm-epistasis-experiments/pull/35))
- In progress: Whichmut simulation framework ([#170](https://github.com/matsengrp/dasm2-experiments/pull/170))
- Open: ESM2 backbone experiment to isolate paradigm vs architecture ([#183](https://github.com/matsengrp/dasm2-experiments/issues/183))

## Infrastructure

- Native Thrifty SHM and CLI WhichmutHead support in progress ([#202](https://github.com/matsengrp/dasm2-experiments/pull/202))
- Documentation extension with technical details ([#197](https://github.com/matsengrp/dasm2-experiments/issues/197))
- viral-dasm train script refactor to use train_model_from_datasets API ([#63](https://github.com/matsengrp/viral-dasm-experiments-1/pull/63))

Open: likelihood CLI command ([#200](https://github.com/matsengrp/dasm2-experiments/issues/200)), loading config proposal ([#196](https://github.com/matsengrp/dasm2-experiments/issues/196)).

## Looking Ahead

- Multi-step likelihood experiment ([#201](https://github.com/matsengrp/dasm2-experiments/issues/201))
- Maturation stage as model input covariate ([#199](https://github.com/matsengrp/dasm2-experiments/issues/199))
- ESM2 paradigm vs architecture experiment ([#183](https://github.com/matsengrp/dasm2-experiments/issues/183))
- Lightweight transformer head for ESM2 ([#184](https://github.com/matsengrp/dasm2-experiments/issues/184))
```

## User Scenarios & Testing

### User Story 1 - Generate Narrative Digest (Priority: P1)

A team lead wants to create a weekly narrative digest as a markdown file for review.

**Independent Test**: Run `/bip.narrative dasm2` and verify `narrative/dasm2/YYYY-MM-DD.md` is created.

**Acceptance Scenarios**:

1. **Given** channel with config file, **When** user runs `/bip.narrative dasm2`, **Then** markdown file is created
2. **Given** no config file for channel, **When** user runs slash command, **Then** error with instructions to create config
3. **Given** `--since 2w` argument, **When** user runs slash command, **Then** two weeks of activity is included
4. **Given** `--verbose` flag, **When** user runs slash command, **Then** PR/issue bodies are summarized and included
5. **Given** no activity in period, **When** user runs slash command, **Then** informative message printed and no file created

---

### User Story 2 - Theme Classification with Viral/Antibody Paragraphs (Priority: P1)

Items should be classified into themes, with viral and antibody work as separate subsections within themes.

**Acceptance Scenarios**:

1. **Given** flu-usher PR and dasm2-experiments PR both in Data Prep theme, **When** narrative generates, **Then** flu-usher appears under **Viral:** subheading
2. **Given** only antibody work in a theme, **When** narrative generates, **Then** no viral subheading appears
3. **Given** config specifies viral/antibody separation, **When** narrative generates, **Then** that preference is respected

---

### User Story 3 - Hybrid Formatting (Priority: P1)

Output uses bullets for independent items, prose for connected items.

**Acceptance Scenarios**:

1. **Given** Data Handling section with 4 independent utilities, **When** narrative generates, **Then** output is bullet list
2. **Given** Architecture section with 2 related merged PRs, **When** narrative generates, **Then** output is prose paragraph
3. **Given** mix of merged and open items, **When** narrative generates, **Then** "In progress:" and "Open:" markers are used

---

### User Story 4 - Safe Digest Default (Priority: P0 - Prerequisite)

`bip digest` should not post by default.

**Acceptance Scenarios**:

1. **Given** configured webhook, **When** user runs `bip digest --channel foo`, **Then** preview shown but NOT posted
2. **Given** configured webhook, **When** user runs `bip digest --channel foo --post`, **Then** digest IS posted
3. **Given** `/bip.digest foo` slash command, **When** user runs it, **Then** it defaults to preview (matches CLI)

---

## Requirements

### Functional Requirements

#### Prerequisite: Digest Default Change

- **FR-000**: `bip digest` MUST default to preview-only (no posting)
- **FR-000a**: `bip digest --post` MUST be required to actually post
- **FR-000b**: `/bip.digest` slash command MUST be updated to match
- **FR-000c**: `bip digest --verbose` MUST fetch PR/issue bodies via `gh` and summarize using Claude Haiku in parallel

#### Slash Command

- **FR-001**: Slash command `/bip.narrative {channel}` MUST exist
- **FR-002**: Command MUST run `bip digest --channel {channel}` to get raw data
- **FR-003**: Command MUST read config from `narrative/{channel}.md`
- **FR-004**: Command MUST write output to `narrative/{channel}/YYYY-MM-DD.md`
- **FR-005**: Command MUST support `--since` argument (default 1w)
- **FR-005a**: Command MUST support `--verbose` flag to include summarized PR/issue bodies

#### Configuration

- **FR-006**: Shared `preferences.md` MUST define default formatting and content rules
- **FR-007**: Channel config MUST define theme categories (ordered list)
- **FR-008**: Channel config MUST inherit from `preferences.md` and MAY override
- **FR-009**: Channel config MAY include repo context for classification hints
- **FR-010**: Missing config file MUST produce helpful error message

#### Output Format

- **FR-011**: Output MUST be standard markdown with GitHub links
- **FR-012**: Output MUST use hybrid format (bullets for lists, prose for connected items)
- **FR-013**: Output MUST use subheadings within themes as specified by channel config (e.g., **Viral:** / **General:** or **Antibody:** — channel decides which apply)
- **FR-014**: Output MUST mark status with "In progress:" and "Open:" prefixes
- **FR-015**: Output MUST NOT invent technical context beyond available information
- **FR-016**: Output MUST create `narrative/{channel}/` directory if needed
- **FR-017**: Output MUST NOT be auto-committed
- **FR-018**: If output file exists, command MUST overwrite it
- **FR-019**: Output header MUST show date range based on `--since` value (e.g., "Jan 18-25, 2026")

#### Theme Handling

- **FR-020**: Themes with no activity MUST be omitted
- **FR-021**: "Looking Ahead" section MUST list open issues and active PRs (intentionally repeating items from theme sections for emphasis)

#### Error Handling

- **FR-022**: If no activity found, command MUST skip file creation and print informative message
- **FR-023**: Malformed config file MUST produce helpful error with guidance on expected format

#### Documentation

- **FR-024**: CLAUDE.md MUST be updated to document `/bip.narrative` alongside other bip commands

### Non-Functional Requirements

- **NFR-001**: Generated markdown should be scannable (bullets) and readable (prose where appropriate)
- **NFR-002**: No LLM calls from Go code (slash command pattern only)

## Success Criteria

- **SC-001**: `/bip.narrative dasm2` creates well-formed markdown file
- **SC-002**: All items from `bip digest` appear in narrative
- **SC-003**: Theme classification respects config file
- **SC-004**: Viral/antibody items appear under appropriate subheadings
- **SC-005**: Hybrid format used appropriately (bullets vs prose)
- **SC-006**: No confabulated technical details

## Technical Notes

### Slash Command Implementation

The skill file at `.claude/skills/bip.narrative/` should:

1. Parse channel argument and flags (`--since`, `--verbose`)
2. Run `bip digest --channel {channel} --since {since} [--verbose]`
   - With `--verbose`, digest output includes Ollama-generated summaries of PR/issue bodies
3. Read `narrative/preferences.md` (shared defaults)
4. Read `narrative/{channel}.md` (themes, repo context, project-specific prefs)
5. Construct prompt with:
   - Raw activity data (with summaries if `--verbose`)
   - Theme definitions from channel config
   - Merged preferences (shared + project-specific)
6. Generate narrative using Claude Code's context
7. Write to `narrative/{channel}/YYYY-MM-DD.md`

### Verbose Mode

With `--verbose`, the command fetches PR/issue bodies via `gh` and uses Claude Haiku to summarize them in parallel, giving richer context than titles alone. This allows more accurate prose without confabulation. Without `--verbose`, only titles are used.

**Implementation**: Shell out to `claude --model haiku --print` for each PR/issue body, parallelized with goroutines. The summarization should:
1. Fetch PR/issue bodies via `gh api`
2. For each body, spawn goroutine calling `claude --model haiku --print "Summarize in 1-2 sentences: {body}"`
3. Collect summaries with bounded concurrency (e.g., 10 parallel)
4. Include summaries in digest output alongside titles

Cost is negligible (~$0.01-0.02 per digest run).

## Out of Scope

- Automatic Slack posting (use existing `bip digest --post` for that)
- Multi-channel combined digests
- Auto-commit generated files
- LLM calls from Go code

## Dependencies

- Bead bipartite-8d5: Add --dry-run flag (DONE)
- Change `bip digest` default to preview-only (NEW - do before slash command)
- `claude` CLI available for Haiku summarization (for --verbose)
- Bead bipartite-c39: Narrative digest format (this feature)

## Constitution Note

This spec establishes a pattern: **No LLM calls from Go for complex generation tasks.** Use slash commands that leverage Claude Code's LLM instead. Small/local models via Ollama for embeddings are acceptable.
