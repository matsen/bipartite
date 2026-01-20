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

# Metadata editing
bp supersedes <id> <doi>         # Mark paper as superseding another (e.g., published replaces preprint)

# Maintenance
bp rebuild                       # Rebuild ephemeral DB from JSONL
bp check                         # Verify integrity
bp groom                         # Find duplicates, problems; suggest/apply fixes
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
- `id`: Internal identifier, defaults to source's citekey (e.g., `Ahn2026-rs` from Paperpile)
- `doi`: DOI if available (**primary key for deduplication** across re-imports)

### Import Deduplication Logic

1. **DOI match**: If incoming paper's DOI matches existing entry → update metadata, keep existing `id`
2. **New paper, no ID collision**: Create entry using source's citekey as `id`
3. **New paper, ID collision**: Verify papers are different (DOIs don't match), then create with modified `id` (e.g., `Ahn2026-rs-2`)

This means `id` is stable once assigned. Re-imports update metadata but don't change identifiers. One import source at a time.

### Grooming

`bp groom` performs deeper analysis that would slow down regular imports:

- **Duplicate detection**: Papers with same title/authors but different IDs (no DOI to match on)
- **Missing PDFs**: Entries where `pdf_path` doesn't resolve to a file
- **Preprint→published**: Suggest `supersedes` relationships (title matching, author overlap)
- **Metadata quality**: Missing abstracts, malformed dates, etc.

Interactive by default; `--fix` to auto-apply safe fixes, `--json` for agent consumption.
- `title`, `authors`, `abstract`: Core metadata
- `published`: Structured date
- `venue`: Journal/preprint server
- `pdf_path`: Relative path to main PDF (combined with configured root)
- `supplement_paths`: Optional array of relative paths to supplementary PDFs

### Paperpile Attachment Structure

Paperpile exports attachments in an `attachments` array per paper:

```json
"attachments": [
  {
    "_id": "...",
    "article_pdf": 1,           // 1 = main PDF, 0 = supplement
    "filename": "All Papers/M/Matsen et al. 2025 - A sitewise model....pdf",
    "filesize": 1780084,
    ...
  },
  {
    "_id": "...",
    "article_pdf": 0,           // This is a supplement
    "filename": "All Papers/M/Matsen et al. 2025 - msaf186_supplementary_data.pdf",
    ...
  }
]
```

The importer maps:
- Attachment with `article_pdf: 1` → `pdf_path`
- Attachments with `article_pdf: 0` → `supplement_paths`
- `supersedes`: DOI of paper this one replaces (e.g., preprint → published)
- `source`: Origin info for re-import matching
  - `type`: Reference manager (`paperpile`, `zotero`, `mendeley`, `manual`, `s2`)
  - `id`: Manager-specific ID (note: Paperpile may change this on re-import)

### Data Flow

1. **Write path**: `bp import` → append to JSONL → rebuild DB index
2. **Read path**: Query DB (fast) → return structured JSON
3. **Sync path**: `git pull` → `bp rebuild` → fresh DB from merged JSONL
4. **Import merge**: Match by `doi` (primary) → replace existing entry with new data
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

- **Nodes**: papers, concepts, code features, artifacts
- **Edges**: directed relationships with semantic summaries

#### Phase III-a: Paper Edges

Foundation for the knowledge graph - directed edges between papers:

- Store edges in JSONL (git-mergeable, like refs.jsonl)
- CLI commands: `bp edge add`, `bp edge import`, `bp edge list`, `bp edge search`, `bp edge export`
- Relationship types: "cites", "extends", "contradicts", "implements", "applies-to", "builds-on"
- External tools (tex-to-edges skill) generate edges; bp provides storage and query

#### Phase III-b: Concept Nodes

Extend the graph with concept nodes extracted from papers:

- **Concept nodes**: Named ideas/methods that papers relate to (e.g., "variational inference", "phylogenetics")
- **Paper↔concept edges**: "Paper A introduces variational inference", "Paper B applies variational inference"
- **Concept↔concept edges**: "variational inference extends probabilistic inference"
- Concept extraction via external tools (LLM-based analysis of abstracts/content)

#### Relational Summaries

Summaries live on edges and describe a node *as it relates to* another node. The same paper has different summaries depending on the relationship:

- "Paper A → variational inference": "Introduces a novel ELBO formulation that reduces variance in gradient estimates"
- "Paper A → phylogenetics": "Applies variational methods to tree inference, achieving 10x speedup over MCMC"
- "Paper A → Paper B": "Provides the theoretical foundation that Paper B extends to non-Euclidean spaces"

When you traverse the graph from a particular node, you get summaries contextualized to your starting point. An agent exploring "what papers relate to my current project?" gets summaries tailored to that project's concepts, not generic abstracts.

#### Edge Semantics

Edges carry:
- **Direction**: A relates-to B (asymmetric)
- **Relationship type**: "cites", "extends", "contradicts", "implements", "applies-to"
- **Relational summary**: prose explaining the connection from A's perspective toward B

#### Edge Generation

Edges are generated by external tools, not bp itself. bp provides storage and query capabilities for the knowledge graph.

**tex-to-edges Claude skill**: A Claude Code skill that analyzes a manuscript and its references:

1. Parse the tex file to extract `\cite{key}` references and their surrounding context
2. Read the bib file to map citekeys to papers
3. For each citation, use the LLM to understand *why* the paper is being cited based on context
4. Generate edges: `manuscript → cited_paper` with relationship type ("cites", "extends", "builds-on", etc.) and a relational summary
5. Output JSONL that bp can ingest via `bp edge add` (or similar)

This separation keeps bp focused on data management while allowing flexible edge generation strategies. The skill approach means edge generation happens in the natural flow of writing—an agent helping with a paper can populate the knowledge graph as a side effect.

### Phase IV: Academic Graph Integration

Connect to the broader academic graph via two complementary services from Allen AI:

#### Phase IV-a: Semantic Scholar (S2) - Structured Database

**`bp s2` commands** - Classical database operations via the [Semantic Scholar API](https://www.semanticscholar.org/product/api):

- `bp s2 add DOI:...` - Add papers by identifier, fetch metadata
- `bp s2 add-pdf paper.pdf` - Extract DOI from PDF, fetch metadata
- `bp s2 citations <id>` / `bp s2 references <id>` - Citation tracking
- `bp s2 gaps` - Find highly-cited papers missing from your collection
- `bp s2 link-published` - Auto-detect preprint→published relationships

S2 provides structured access to 200M+ papers. Good for: adding papers, metadata lookup, citation graphs.

#### Phase IV-b: ASTA - LLM-Powered Discovery

**`bp asta` commands** - Next-generation search via [ASTA](https://allenai.org/blog/asta) (Allen AI's Scientific Tool for Agents):

- `bp asta search <query>` - LLM-powered relevance search ("like Google Scholar on steroids")
- `bp asta snippet <query>` - Search text passages within papers (unique to ASTA)
- `bp asta paper <id>` / `bp asta citations <id>` - Paper lookup with relevancy ratings

ASTA is built on top of Semantic Scholar's database but adds LLM reasoning:
- Multi-step search mimicking expert researchers
- Relevancy ratings (Perfectly Relevant / Relevant / Somewhat Relevant)
- Evidence passages explaining why papers match
- Full-text snippet search across open-access papers

Good for: literature discovery, finding specific passages, exploratory research.

**Key difference**: S2 answers "give me this paper's citations" (structured). ASTA answers "find papers about variational inference in phylogenetics that discuss convergence" (semantic).

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

### Agentic TDD

Tests are written and run by agents in a fully autonomous loop:

```
1. Spec defines behavior
2. Agent writes failing test with real fixture data
3. Agent writes minimal implementation to pass
4. Agent runs tests, iterates until green
5. Agent moves to next spec item
6. Human reviews at PR/checkpoint level
```

**Test fixtures**: Real entries extracted from Paperpile export (`_ignore/` folder, not committed), covering edge cases:
- Papers with/without DOI
- Single and multiple attachments (supplements)
- Preprints and published versions
- Various venues (journals, bioRxiv, conferences)
- Missing fields (no abstract, partial dates)

**Phase I test scope**:
- Import parsing (Paperpile JSON → internal schema)
- Deduplication logic (DOI match, ID collision, suffix generation)
- JSONL serialization round-trip
- DB rebuild from JSONL
- Search and query operations
- BibTeX export format
- PDF path resolution
- CLI commands (integration tests with temp directories)

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
# (idempotent - matches by doi, replaces existing entries, adds new)
bp import --format paperpile export-updated.json
```

### Collaborative Research Group

```bash
# Researcher A adds papers via their reference manager, then imports
bp import --format paperpile researcher-a-export.json
git commit -m "Add papers on phylogenetics"
git push

# Researcher B does the same
bp import --format paperpile researcher-b-export.json
git commit -m "Add papers on ML"
git push

# Merge
git pull  # JSONL merges cleanly (append-only)
bp rebuild  # Refresh local DB
```

## Development Approach

Bipartite will be built using the tools it's designed for:

- **Beads orchestration**: Use beads to manage the agentic development loop
- **Agent-written code**: Agents write the implementation, humans review and guide
- **Dogfooding**: As soon as basic import works, use bipartite to manage literature for the project itself

This is both practical (agents are good at this) and validating (if the CLI is awkward for agents to use during development, fix it).

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
- **Language**: **Go** - chosen for fast CLI startup, simple concurrency, single-binary deployment, excellent standard library for JSON/HTTP, and high-quality agent-generated code (simpler language = fewer agent mistakes)
- **Parser libraries**: BibTeX generation for export (import is JSON, trivial to parse)
- **Principle**: Prefer embeddable over client-server architectures

## Non-Goals

- GUI or web interface (maybe someday, but not the focus)
- PDF storage (point to existing folders)
- Sync service (git is the sync mechanism)
- Full-text PDF indexing (Phase II might index abstracts only)
- Social features (this is for research groups, not social networks)
