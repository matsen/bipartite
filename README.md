# bipartite

A command-line reference manager designed for AI agents and researchers. Import from Paperpile, search with full-text, open PDFs, export to BibTeX.

The name comes from a bipartite graph connecting two worlds: the researcher's artifacts (notes, code, concepts) and the academic literature (papers, citations, authors).

## Design Principles

**Agent-first**: CLI is the primary interface. JSON output by default. No MCP server needed—agents use bash directly.

**Git-versionable**: JSONL is the source of truth, human-readable and merge-friendly. SQLite is an ephemeral cache rebuilt on demand. Multiple researchers can add papers and resolve conflicts through git.

**Minimal dependencies**: Fast startup, single binary, no heavyweight frameworks.

## Installation

```bash
go build -o bip ./cmd/bip
```

Requires Go 1.21+.

## Quick Start

```bash
# Initialize repository
bip init

# Configure PDF location (e.g., Paperpile's Google Drive folder)
bip config pdf-root ~/Google\ Drive/My\ Drive/Paperpile

# Import references
bip import --format paperpile ~/Downloads/paperpile-export.json

# Rebuild search index
bip rebuild

# Search
bip search "phylogenetics"

# Open a paper
bip open Smith2026-ab
```

## Commands

| Command | Description |
|---------|-------------|
| `bip init` | Initialize a new repository |
| `bip config [key] [value]` | Get/set configuration |
| `bip import --format paperpile <file>` | Import from Paperpile JSON |
| `bip rebuild` | Rebuild search index from source data |
| `bip search <query>` | Full-text search across titles, abstracts, authors, years |
| `bip list` | List all references |
| `bip get <id>` | Get a specific reference by ID |
| `bip open <id>` | Open PDF in configured viewer |
| `bip export --bibtex` | Export to BibTeX format |
| `bip check` | Validate repository integrity |
| `bip groom` | Detect orphaned edges; use `--fix` to remove |

### Semantic Scholar (S2) Commands

| Command | Description |
|---------|-------------|
| `bip s2 add <paper-id>` | Add paper by DOI, arXiv ID, or S2 ID |
| `bip s2 add-pdf <file>` | Add paper by extracting DOI from PDF |
| `bip s2 lookup <paper-id>` | Look up paper info without adding |
| `bip s2 citations <paper-id>` | Find papers that cite this paper |
| `bip s2 references <paper-id>` | Find papers referenced by this paper |
| `bip s2 gaps` | Discover highly-cited papers you're missing |
| `bip s2 link-published` | Link preprints to published versions |

Paper IDs support: `DOI:10.xxx`, `ARXIV:xxxx.xxxxx`, `PMID:xxxxxxxx`, or local IDs.

### ASTA (Academic Search Tool API) Commands

Read-only exploration of academic papers via Allen AI's ASTA service.

| Command | Description |
|---------|-------------|
| `bip asta search <query>` | Search papers by keyword relevance |
| `bip asta snippet <query>` | Search text snippets within papers |
| `bip asta paper <paper-id>` | Get paper details |
| `bip asta citations <paper-id>` | Get papers that cite this paper |
| `bip asta references <paper-id>` | Get papers referenced by this paper |
| `bip asta author <name>` | Search for authors by name |
| `bip asta author-papers <author-id>` | Get papers by an author |

Common flags: `--limit N`, `--year YYYY:YYYY`, `--venue <name>`, `--human`.

Requires `ASTA_API_KEY` environment variable.

### Knowledge Graph Commands

| Command | Description |
|---------|-------------|
| `bip edge add -s <source> -t <target> -r <type> -m <summary>` | Add a directed edge between papers |
| `bip edge import <file>` | Bulk import edges from JSONL |
| `bip edge list <paper-id>` | List edges for a paper (`--incoming`, `--all`) |
| `bip edge search --type <type>` | Find edges by relationship type |
| `bip edge export` | Export edges to JSONL (`--paper` to filter) |

Relationship types: `cites`, `extends`, `contradicts`, `implements`, `applies-to`, `builds-on` (custom types also allowed).

All commands output JSON by default. Use `--human` for readable output.

## Configuration

| Key | Description |
|-----|-------------|
| `pdf-root` | Path to PDF folder |
| `pdf-reader` | PDF viewer: `system`, `skim`, `zathura`, `evince`, `okular` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `S2_API_KEY` | Semantic Scholar API key for higher rate limits (optional) |
| `ASTA_API_KEY` | ASTA API key for academic search (required for `bip asta` commands) |

Add to `.env` file (gitignored):
```
S2_API_KEY=your_key_here
ASTA_API_KEY=your_key_here
```

Get an S2 API key at: https://www.semanticscholar.org/product/api#api-key-form

## Performance

Tested on a 6,400 paper library (32MB Paperpile export):

| Operation | Time |
|-----------|------|
| Import | 0.4s |
| Rebuild index | 7s |
| Search | 9ms |

## Data Storage

```
.bipartite/
├── refs.jsonl      # Papers - human-readable, git-mergeable
├── edges.jsonl     # Knowledge graph edges - git-mergeable
├── config.json     # Local configuration
└── cache/
    └── refs.db     # SQLite with FTS5 - ephemeral, gitignored
```

JSONL files are the source of truth and can be version-controlled. The SQLite cache is rebuilt with `bip rebuild` after pulling changes.

## Collaboration Workflow

```bash
# Researcher A adds papers
bip import --format paperpile export-a.json
git add .bipartite/refs.jsonl
git commit -m "Add phylogenetics papers"
git push

# Researcher B does the same
bip import --format paperpile export-b.json
git commit -m "Add ML papers"
git push

# After pull/merge
git pull
bip rebuild  # Refresh local index
```

## Roadmap

- **Phase I** ✓: Core reference manager with Paperpile import
- **Phase II** ✓: RAG index for semantic search over abstracts
- **Phase III-a** ✓: Knowledge graph with directed edges between papers
- **Phase III-b**: Concept nodes and artifact connections
- **Phase IV** ✓: Semantic Scholar integration for metadata enrichment

See [VISION.md](VISION.md) for details.

## License

MIT
