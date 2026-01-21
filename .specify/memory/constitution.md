<!--
Sync Impact Report
==================
Version change: 1.0.1 → 1.1.0 (minor: principle expansion)
Modified principles: VI. Simplicity (added breaking-changes-over-compatibility guidance)
Added sections: None
Removed sections: None
Templates requiring updates: None
Follow-up TODOs: None
-->

# Bipartite Constitution

## Core Principles

### I. Agent-First Design

Bipartite is designed for AI agents and command-line workflows, not GUI interaction.

- CLI (`bip`) MUST be the primary interface—all functionality accessible via bash commands
- Output MUST be structured (JSON) by default for machine parsing
- Human-readable output MUST be available as an alternative format flag
- No MCP server required—agents interact directly via bash
- Commands MUST be composable with other tools (pipes, beads orchestration)

**Rationale**: Traditional reference managers are GUI-first. Bipartite inverts this to enable autonomous agent workflows where agents can search, retrieve, and open papers without human intervention.

### II. Git-Versionable Architecture

All persistent state MUST be stored in formats that support git versioning and collaboration.

- JSONL MUST be the single source of truth for all reference data
- Query layer (SQLite, DuckDB, or in-memory) MUST be ephemeral—rebuilt from JSONL on demand via `bip rebuild`
- Database files MUST be gitignored; never committed
- JSONL format MUST support clean merges (append-only where possible)
- Each bipartite repository MUST be self-contained and standalone

**Rationale**: Inspired by beads, this architecture enables full git history, collaboration without sync services, and agent-assisted merge conflict resolution.

### III. Fail-Fast Philosophy

The system MUST fail immediately and explicitly when something is wrong.

- No silent defaults or fallbacks—if configuration is missing, error immediately
- Error messages MUST explain what went wrong AND what was expected
- No silent error swallowing—all failures MUST be visible to the caller
- Invalid input MUST be rejected, not silently corrected
- Missing files (PDFs, config) MUST produce clear errors, not empty results

**Rationale**: Silent failures create debugging nightmares. Agents and humans both benefit from immediate, clear feedback when something is wrong.

### IV. Real Testing (Agentic TDD)

Tests MUST validate actual behavior using real data, written and executed by agents.

- No fake mocks—use real fixtures extracted from actual Paperpile exports
- Test fixtures MUST cover edge cases: missing DOIs, multiple attachments, partial dates
- TDD cycle: Agent writes failing test → Agent implements → Agent iterates until green
- Human reviews at PR/checkpoint level, not individual test cycles
- Integration tests MUST use real file I/O and database operations
- No skipped tests or TODO placeholders in committed code

**Rationale**: Fake mocks test the mock, not the code. Real fixtures from actual data ensure the system works with real-world inputs.

### V. Clean Architecture

Code MUST follow clean architecture principles with excellent naming.

- Single Responsibility: Each function/module does one thing well
- Dependency Inversion: Depend on abstractions, not concretions
- Composition over Configuration: Inject behavior, don't select via flags
- Names MUST reveal intent without requiring comments
- Booleans MUST be questions: `has_abstract`, `is_preprint`, not `abstract`, `preprint`
- No generic names: avoid Manager, Handler, Utils, Processor, Helper
- Variables MUST describe contents: `paper_doi` not `id`, `pdf_file_path` not `path`

**Rationale**: Clean architecture enables maintainability and testability. Good names eliminate the need for comments and reduce cognitive load.

### VI. Simplicity

The system MUST remain minimal and avoid over-engineering.

- Minimal external dependencies—prefer standard library where sufficient
- Fast startup time—CLI MUST feel instant (compiled language preferred)
- No premature abstraction—wait until patterns emerge from actual use
- Delete unused code completely—no commented-out code or "maybe later" features
- YAGNI: Don't build for hypothetical future requirements
- The simplest design that works is the correct design
- Prefer good breaking changes over backward-compatibility hacks—this is an internal tool
- No deprecation shims, re-exports for old names, or `_unused` parameters

**Rationale**: A small, beautiful thing that doesn't depend on too many big things. Complexity is the enemy of reliability and maintainability. As an internal tool, we can break things to make them better.

## Technology Constraints

Constraints that guide technology choices without mandating specific tools.

### CLI Responsiveness
- Startup time MUST be fast enough to feel instant (<100ms target)
- Compiled language preferred (Rust, Go) over interpreted (Python, Ruby)
- No JVM-based implementations (startup time unacceptable)

### Embeddable Over Client-Server
- All databases MUST be embeddable (SQLite, DuckDB)—no separate server processes
- No dependencies requiring daemon processes or network services
- Self-contained binary distribution preferred

### Data Portability
- All data formats MUST be human-readable and editable (JSON, JSONL, plain text)
- No proprietary or binary-only formats for source-of-truth data
- BibTeX export MUST use classic BibTeX (not BibLaTeX) for maximum compatibility

### Platform Support
- macOS and Linux MUST be supported
- PDF reader integration: Skim (macOS), Zathura/Evince/Okular (Linux)

## Development Workflow

### Agentic Development Loop

Development follows a beads-orchestrated agentic workflow:

1. **Spec defines behavior**: Feature specifications written in `.specify/` format
2. **Agent writes failing tests**: Using real fixture data from `_ignore/`
3. **Agent implements**: Minimal code to pass tests
4. **Agent iterates**: Red-green-refactor until all tests pass
5. **Human reviews**: At PR level, not individual commits

### Dogfooding

As soon as basic import works, bipartite MUST be used to manage literature for its own development. If the CLI is awkward for agents to use, fix it.

### Code Review Standards

- All PRs MUST pass linting and type checking
- All PRs MUST include tests for new functionality
- All PRs MUST update documentation if behavior changes
- Constitution principles MUST be verified in review

### Fixture Management

- Test fixtures extracted from real Paperpile exports stored in `_ignore/` (gitignored)
- Fixtures MUST NOT contain personally identifying information
- Fixtures MUST cover: papers with/without DOI, single/multiple attachments, various venues, missing fields

## Governance

This constitution supersedes all other practices and documentation. When conflicts arise, the constitution wins.

### Amendment Process

1. Propose amendment with rationale
2. Verify impact on existing code and tests
3. Update constitution with version bump
4. Update dependent templates if affected
5. Document migration path for breaking changes

### Versioning Policy

- **MAJOR**: Backward-incompatible principle changes or removals
- **MINOR**: New principles added or existing principles materially expanded
- **PATCH**: Clarifications, wording improvements, non-semantic refinements

### Compliance

- All PRs MUST verify compliance with constitution principles
- Violations MUST be justified in the Complexity Tracking table (see plan template)
- Unjustified violations MUST block merge

**Version**: 1.1.0 | **Ratified**: 2026-01-12 | **Last Amended**: 2026-01-12
