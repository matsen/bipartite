# Feature Specification: Narrative Digest

**Feature Branch**: `012-narrative-digest`
**Created**: 2026-01-25
**Status**: Draft
**Input**: Create a slash command that generates thematic narrative digests from GitHub activity, configured per-channel via markdown files in nexus.

## Overview

### Architecture Decision
**No LLM calls from Go.** The narrative generation uses Claude Code's LLM via a slash command, not Go code calling external APIs. Small models via Ollama for embeddings are fine, but narrative generation requires the slash command pattern.

### Workflow
```
/bip.narrative dasm2
```

1. Slash command runs `bip digest --channel dasm2 --dry-run` to get raw activity
2. Reads channel config from `nexus/narrative/dasm2.md`
3. Uses Claude Code's LLM to generate themed prose
4. Writes output to `nexus/narrative/dasm2/YYYY-MM-DD.md`

### Directory Structure (in nexus)
```
nexus/
  narrative/
    dasm2.md                    # channel config: themes, preferences
    loris.md                    # channel config for loris
    dasm2/
      2026-01-25.md             # generated digest
      2026-01-18.md             # older digest
    loris/
      2026-01-25.md
```

### Channel Config Example (`narrative/dasm2.md`)

```markdown
# dasm2 Narrative Configuration

## Themes

1. **Architecture** - Model structure (backbones, heads, attention mechanisms)
2. **Training & Loss** - Loss functions, multi-objective training, optimization
3. **Data Prep** - Dataset processing, format conversion (flu-usher, mat-pcp)
4. **Benchmarking** - Evaluation, functional datasets, comparisons, Whichmut
5. **Infrastructure** - CLI, config, dependencies, documentation

## Preferences

- Within each theme, write separate paragraphs for viral work (flu, SARS2, H3) and antibody work when both are present
- Use inline attribution ("Will's PR...", "Jared is working on...")
- End with "Looking Ahead" bullet list for open issues and in-progress PRs
- Keep each theme section to 1-3 short paragraphs

## Repo Context

- **flu-usher**: Influenza phylogenetic tree tools (Data Prep, viral)
- **mat-pcp**: Mutation counting pipeline (Data Prep)
- **dasm2-experiments**: Main DASM2 development (all themes)
- **viral-dasm-experiments-1**: Viral-specific experiments (viral paragraphs)
```

### Example Output (`narrative/dasm2/2026-01-25.md`)

```markdown
# dasm2 Digest: Jan 18-25, 2026

## Architecture

Will landed ESM2 backbone support ([#181](https://github.com/matsengrp/dasm2-experiments/pull/181)),
expanding the model's representational capacity. His mutation frequency branch length
initialization ([#171](https://github.com/matsengrp/dasm2-experiments/pull/171)) should
improve convergence. Open discussion on neutral model architecture refactor
([#195](https://github.com/matsengrp/dasm2-experiments/issues/195)).

## Training & Loss

Joint multi-objective training with weighted loss sum
([#153](https://github.com/matsengrp/dasm2-experiments/pull/153)) merged, enabling
simultaneous optimization of multiple objectives.

## Data Prep

The modular dataset registry ([#176](https://github.com/matsengrp/dasm2-experiments/pull/176))
and functional data format utilities ([#182](https://github.com/matsengrp/dasm2-experiments/pull/182))
landed. Jared is working on problematic sites masking
([#198](https://github.com/matsengrp/dasm2-experiments/pull/198)).

On the viral side, flu-usher now extracts CDS from tree mutations
([#8](https://github.com/matsengrp/flu-usher/pull/8)) and supports host-specific
subtree extraction ([#10](https://github.com/matsengrp/flu-usher/pull/10)).

## Benchmarking

New functional dataset evaluation notebooks
([#189](https://github.com/matsengrp/dasm2-experiments/pull/189)) provide tooling
for model assessment. Will's Whichmut simulation framework
([#170](https://github.com/matsengrp/dasm2-experiments/pull/170)) is in progress.

## Infrastructure

Seed parameter now applied in training for reproducibility
([#179](https://github.com/matsengrp/dasm2-experiments/pull/179)). New CLI commands
proposed: likelihood ([#200](https://github.com/matsengrp/dasm2-experiments/issues/200)).
Loading config proposal under discussion
([#196](https://github.com/matsengrp/dasm2-experiments/issues/196)).

## Looking Ahead

- Multi-step likelihood experiment ([#201](https://github.com/matsengrp/dasm2-experiments/issues/201))
- Maturation stage as model covariate ([#199](https://github.com/matsengrp/dasm2-experiments/issues/199))
- ESM2 backbone experiment: isolating paradigm vs architecture ([#183](https://github.com/matsengrp/dasm2-experiments/issues/183))
```

## User Scenarios & Testing

### User Story 1 - Generate Narrative Digest (Priority: P1)

A team lead wants to create a weekly narrative digest as a markdown file for review.

**Independent Test**: Run `/bip.narrative dasm2` and verify `narrative/dasm2/YYYY-MM-DD.md` is created.

**Acceptance Scenarios**:

1. **Given** channel with config file, **When** user runs `/bip.narrative dasm2`, **Then** markdown file is created
2. **Given** no config file for channel, **When** user runs slash command, **Then** error with instructions to create config
3. **Given** `--since 2w` argument, **When** user runs slash command, **Then** two weeks of activity is included

---

### User Story 2 - Theme Classification with Viral/Antibody Paragraphs (Priority: P1)

Items should be classified into themes, with viral and antibody work as separate paragraphs within themes.

**Acceptance Scenarios**:

1. **Given** flu-usher PR and dasm2-experiments PR both in Data Prep theme, **When** narrative generates, **Then** flu-usher appears in separate "viral" paragraph
2. **Given** only antibody work in a theme, **When** narrative generates, **Then** no viral paragraph appears
3. **Given** config specifies viral/antibody separation, **When** narrative generates, **Then** that preference is respected

---

### User Story 3 - Channel-Specific Configuration (Priority: P1)

Different channels have different themes and preferences.

**Acceptance Scenarios**:

1. **Given** dasm2.md defines 5 themes, **When** narrative generates for dasm2, **Then** those 5 themes are used
2. **Given** loris.md defines different themes, **When** narrative generates for loris, **Then** loris themes are used
3. **Given** config includes repo context, **When** narrative generates, **Then** repos are correctly associated with themes

---

## Requirements

### Functional Requirements

#### Slash Command

- **FR-001**: Slash command `/bip.narrative {channel}` MUST exist
- **FR-002**: Command MUST run `bip digest --channel {channel} --dry-run` to get raw data
- **FR-003**: Command MUST read config from `narrative/{channel}.md`
- **FR-004**: Command MUST write output to `narrative/{channel}/YYYY-MM-DD.md`
- **FR-005**: Command MUST support `--since` argument (default 1w)

#### Configuration

- **FR-006**: Config file MUST define theme categories
- **FR-007**: Config file MUST support preferences section
- **FR-008**: Config file MAY include repo context for classification hints
- **FR-009**: Missing config file MUST produce helpful error message

#### Output

- **FR-010**: Output MUST be standard markdown with GitHub links
- **FR-011**: Output MUST create `narrative/{channel}/` directory if needed
- **FR-012**: Output MUST NOT be auto-committed

#### Theme Handling

- **FR-013**: Themes with no activity MUST be omitted
- **FR-014**: Viral and antibody work MUST be separate paragraphs within themes (per config)
- **FR-015**: "Looking Ahead" section MUST list open issues and active PRs

### Non-Functional Requirements

- **NFR-001**: Generated markdown should be ~300-500 words for typical week
- **NFR-002**: No LLM calls from Go code (slash command pattern only)

## Success Criteria

- **SC-001**: `/bip.narrative dasm2` creates well-formed markdown file
- **SC-002**: All items from `bip digest --dry-run` appear in narrative
- **SC-003**: Theme classification respects config file
- **SC-004**: Viral/antibody paragraphs are separated within themes
- **SC-005**: Links are valid GitHub URLs

## Technical Notes

### Slash Command Implementation

The skill file at `.claude/skills/bip.narrative/` should:

1. Parse channel argument
2. Run `bip digest --channel {channel} --dry-run --since {since}`
3. Read `narrative/{channel}.md` config
4. Construct prompt with:
   - Raw activity data
   - Theme definitions from config
   - Preferences from config
5. Generate narrative using Claude Code's context
6. Write to `narrative/{channel}/YYYY-MM-DD.md`

### Prompt Construction

```
You are generating a narrative digest for the {channel} channel.

## Theme Definitions (from config)
{themes from narrative/{channel}.md}

## Preferences (from config)
{preferences from narrative/{channel}.md}

## Raw Activity Data
{output from bip digest --dry-run}

Generate a markdown digest following the theme structure. For each theme with activity:
- Write 1-3 short prose paragraphs
- Use inline attribution ("Will's PR...", "Jared is working on...")
- Separate viral and antibody work into distinct paragraphs when both present
- Include proper markdown links: [#N](URL)

End with "## Looking Ahead" listing open issues and in-progress PRs.
```

## Out of Scope

- Automatic Slack posting (use existing `bip digest` for that)
- Multi-channel combined digests
- Auto-commit generated files
- LLM calls from Go code

## Dependencies

- Bead bipartite-8d5: Add --dry-run flag (DONE)
- Bead bipartite-c39: Narrative digest format (this feature)

## Constitution Note

This spec establishes a pattern: **No LLM calls from Go for complex generation tasks.** Use slash commands that leverage Claude Code's LLM instead. Small/local models via Ollama for embeddings are acceptable.
