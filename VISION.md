# Bipartite: An Agent-First Academic Reference Manager

## Concept

Bipartite is a reference management system designed for AI agents and command-line workflows. The name comes from the conceptual framework of a bipartite graph:

- **One side**: The researcher's world (notes, code, artifacts, concepts)
- **Other side**: The academic literature (papers, citations, authors)

The system connects these two worlds through a knowledge graph where nodes are either artifacts or concepts, and edges carry semantic meaning (not just links, but explanations of relationships).

## Core Philosophy

### Agent-First Design

Traditional reference managers (Zotero, Mendeley, Paperpile) are GUI-first, designed for human browsing and visual interaction. Bipartite inverts this:

- **CLI is the primary interface** - agents interact via bash commands
- **Structured output by default** - JSON for machine parsing
- **No MCP needed** - a beautiful CLI means agents just use bash
- **Composable** - works with other tools like beads for orchestration

### Git-Versionable Architecture (Beads-Inspired)

Inspired by [beads](https://github.com/steveyegge/beads), the system uses:

- **JSONL as single source of truth** - human-readable, git-mergeable
- **Ephemeral database** - SQLite rebuilt from JSONL on demand
- **Merge-friendly** - multiple researchers can add papers, conflicts resolved intelligently
- **Standalone repos** - each bipartite repo is self-contained, suitable for private GitHub

This means:
- No database lock-in
- Full git history of all changes
- Collaboration without sync services
- Agents can resolve merge conflicts

### Minimal Dependencies

A small, beautiful thing that doesn't depend on too many big things:
- Minimal external dependencies
- Fast startup
- Easy to install and run
- No heavyweight frameworks

## CLI Design

Short command: `bp`

```bash
# Initialization
bp init                          # Initialize a bipartite repo
bp config pdf-path <path>        # Set PDF folder location (e.g., Paperpile sync)

# Adding references
bp import --format paperpile export.json  # Import from Paperpile JSON
bp import --format zotero library.json    # Import from Zotero (future)
bp import --format bibtex refs.bib        # Import from BibTeX (future)
# bp add paper.pdf                        # Future: extract metadata from PDF (Phase IV)

# Querying
bp search "keyword"              # Search papers
bp get <id> --json               # Get paper metadata
bp list --json                   # List all papers

# PDF access
bp open <id>                     # Open PDF from linked folder

# Export
bp export --bibtex               # Export to BibTeX
bp export --bibtex --keys a,b,c  # Export specific papers

# Maintenance
bp rebuild                       # Rebuild ephemeral DB from JSONL
bp check                         # Verify integrity
```

## Architecture

```
bipartite-repo/
├── .bipartite/
│   ├── refs.jsonl           # Source of truth - all references
│   ├── config.json          # Repository configuration
│   └── cache/
│       └── refs.db          # Ephemeral SQLite (gitignored)
├── .gitignore               # Ignores cache/
└── README.md
```

### Internal Schema (JSONL)

Reference-manager-agnostic format. The schema is bipartite's own - importers transform from various sources:

```jsonl
{"id":"Ahn2026-rs","doi":"10.64898/2026.01.05.697808","title":"Influenza hemagglutinin subtypes...","authors":[{"first":"Jenny J","last":"Ahn","orcid":"0009-0000-3912-7162"}],"abstract":"Abstract Hemagglutinins...","published":{"year":2026,"month":1,"day":6},"venue":"bioRxiv","pdf_path":"All Papers/A/Ahn et al. 2026 - Influenza hemagglutinin....pdf","source":{"type":"paperpile","id":"2773420d-4009-0be9-920f-d674f7f86794"}}
```

Key fields:
- `id`: Citekey, used as primary identifier (e.g., `Ahn2026-rs`)
- `doi`: DOI if available (for deduplication, ASTA lookup)
- `title`, `authors`, `abstract`: Core metadata
- `published`: Structured date
- `venue`: Journal/preprint server
- `pdf_path`: Relative path to PDF (combined with configured root)
- `source`: Origin info for re-import matching
  - `type`: Reference manager (`paperpile`, `zotero`, `mendeley`, `manual`, `asta`)
  - `id`: Manager-specific ID for matching on re-import

### Data Flow

1. **Write path**: `bp add` → append to JSONL → rebuild DB index
2. **Read path**: Query DB (fast) → return structured JSON
3. **Sync path**: `git pull` → `bp rebuild` → fresh DB from merged JSONL
4. **Import merge**: Match by `source.id` or `doi` → update/skip/add
5. **Conflict resolution**: Agents merge JSONL conflicts intelligently

### PDF Access (Key Design Goal)

**Agents opening papers for humans to read is a core use case.**

PDFs are not stored in the repo. Instead:
- Configure a path to your PDF folder (each reference manager has its own sync location)
- `bp open <id>` finds and opens the PDF in the system viewer
- PDF paths stored per-paper (imported from reference manager metadata)
- Goal: `bp open Ahn2026-rs` just works

Examples of PDF folder locations:
- **Paperpile**: Google Drive sync folder (e.g., `~/Google Drive/Paperpile`)
- **Zotero**: Local storage folder (e.g., `~/Zotero/storage`)
- **Mendeley**: Local data folder
- **Manual**: Any folder you configure

The importer extracts relative PDF paths from the reference manager export, and the configured root path completes the location.

### PDF Reader Support

`bp open` should open PDFs to the correct page when possible. Cross-platform reader support:

**macOS:**
- **Skim** (recommended): `skim://path/to/file.pdf#page=N` - precise page navigation
- **Preview**: Basic open, no page targeting

**Linux:**
- **Zathura**: `zathura --page=N file.pdf`
- **Evince**: `evince --page-index=N file.pdf`
- **Okular**: `okular --page N file.pdf`

Configuration via `bp config pdf-reader <reader>`.

## LaTeX/BibTeX Integration

Academic writing workflow is first-class:

- **Import**: Parse existing `.bib` files to populate the database
- **Export**: Generate BibTeX for papers in your collection
- **Parse**: Read `.tex` files to find `\cite{key}` references
- **Roundtrip**: Database is source of truth, BibTeX is export format

**BibTeX over BibLaTeX**: We use classic BibTeX format for maximum compatibility. BibLaTeX is more powerful but creates friction when collaborating across different setups. Everyone can use BibTeX.

## Phased Roadmap

### Phase I: Core Reference Manager

The foundation that must work perfectly:

- `bp` CLI with all core commands
- JSONL source of truth
- Ephemeral SQLite for queries
- Importer architecture (Paperpile JSON first, extensible to other formats)
- Export to BibTeX
- PDF folder linking
- Git-mergeable design
- Beads integration for bulk import orchestration

### Phase II: RAG Index

Semantic search over your literature:

- Index abstracts for vector search
- "Find papers about variational inference in phylogenetics"
- Conceptual queries, not just keyword matching

### Phase III: Knowledge Graph

The full bipartite vision:

- Nodes: papers, concepts, code features, artifacts
- Edges: directed relationships with semantic summaries
- Generic summaries: overview of a paper
- Relational summaries: "Paper A provides the theoretical basis for Model B"

### Phase IV: ASTA/Semantic Scholar Integration

Connect to the broader academic graph:

- `bp add paper.pdf` - extract DOI from PDF, fetch metadata from ASTA
- Find related papers (citations, references)
- Discover literature beyond your collection
- Enrich local graph with external data

Note: Direct DOI fetching from publishers is blocked (403s). ASTA provides an API that works.

### Phase V: Discovery Tracking

Track how papers are discovered (inspired by beads' "discovered-from" dependency type):

- Reading paper A, discover it cites paper B you should add
- Agent working on a concept finds a relevant paper via ASTA
- Build a provenance graph: "how did I find this paper?"
- Surfaces gaps: "papers cited by 3+ of my references that I don't have"

## Code Quality Standards

### Clean Architecture Principles

- **Single Responsibility**: Each class/function does one thing well
- **Dependency Inversion**: Depend on abstractions, not concretions
- **Composition over Configuration**: Inject behavior, don't select via flags
- **Open/Closed**: Extensible without modification

### Fail-Fast Philosophy

- **No silent defaults or fallbacks** - if something is wrong, stop immediately
- **Explicit error messages** - explain what went wrong and what was expected
- **No silent error handling** - all failures must be visible

### Naming Excellence

- Names reveal intent without requiring comments
- Booleans are questions: `has_abstract` not `abstract`
- No generic names: avoid Manager, Handler, Utils, Processor
- Variables describe what they contain: `paper_doi` not `id`

### Real Testing

- No fake mocks - use real fixtures and integration tests
- Tests validate actual behavior, not fake implementations
- No skipped tests or placeholders
- Compatibility tests against real data

### Simplicity

- Minimal dependencies
- No over-engineering or premature abstraction
- Delete unused code completely
- The simplest design that works is usually best

### Static Analysis

All code must pass:
- Linting (language-appropriate: ruff for Python, etc.)
- Type checking where applicable
- Comprehensive test coverage

### Documentation

- Docstrings for non-trivial functions
- Central documentation for complex systems
- Design decisions documented
- No stale documentation - update or delete

## Use Cases

### Daily Research Workflow

```bash
# Found an interesting paper - add it in Paperpile, then re-import
bp import --format paperpile latest-export.json

# Writing a paper, need citations
bp search "MCMC phylogenetics" --json | jq '.[] | .key'
bp export --bibtex --keys paper1,paper2 >> mybib.bib

# Open a paper to read
bp open paper1
```

### Bulk Import

Using beads for orchestration:
```bash
# Export full library from your reference manager
bp import --format paperpile --dry-run export.json  # Shows what would be imported/updated/skipped
bp import --format paperpile export.json            # Orchestrated via beads for large imports

# Re-import after adding papers
# (idempotent - matches by source.id/doi, updates changed, adds new)
bp import --format paperpile export-updated.json
```

### Collaborative Research Group

```bash
# Researcher A adds papers
bp add 10.1234/paperA
git commit -m "Add paper on phylogenetics"
git push

# Researcher B adds papers
bp add 10.1234/paperB
git commit -m "Add paper on ML"
git push

# Merge
git pull  # JSONL merges cleanly (append-only)
bp rebuild  # Refresh local DB
```

## Technology Decisions

Specific tools to be determined during implementation. The constraints that matter:

### Phase I: Query Layer
- **Constraint**: Ephemeral, rebuildable from JSONL
- **Constraint**: No separate server process
- **Constraint**: Fast startup for CLI responsiveness
- **Constraint**: Embeddable in the `bp` binary
- **Options**: SQLite, DuckDB, or even in-memory structures for small collections

### Phase II: Vector Store (RAG)
- **Constraint**: Embeddable, no separate service
- **Constraint**: Persistent but rebuildable from source
- **Constraint**: Good semantic search quality
- **Options**: To be researched (many options exist)

### Phase III: Graph Database
- **Constraint**: Must fit the git-mergeable philosophy (or have a JSONL-like source of truth)
- **Constraint**: Support directed edges with properties (relationship summaries)
- **Constraint**: Queryable for graph traversal
- **Options**: To be researched (could be a dedicated graph DB, or graph-on-relational)

### General
- **Language**: Should support fast CLI startup, good JSON handling
- **Parser libraries**: BibTeX generation for export (import is JSON, trivial to parse)
- **Principle**: Prefer embeddable over client-server architectures

## Non-Goals

- GUI or web interface (maybe someday, but not the focus)
- PDF storage (point to existing folders)
- Sync service (git is the sync mechanism)
- Full-text PDF indexing (Phase II might index abstracts only)
- Social features (this is for research groups, not social networks)
