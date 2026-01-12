# bipartite

A command-line reference manager designed for AI agents and researchers. Import from Paperpile, search with full-text, open PDFs, export to BibTeX.

The name comes from a bipartite graph connecting two worlds: the researcher's artifacts (notes, code, concepts) and the academic literature (papers, citations, authors).

## Design Principles

**Agent-first**: CLI is the primary interface. JSON output by default. No MCP server needed—agents use bash directly.

**Git-versionable**: JSONL is the source of truth, human-readable and merge-friendly. SQLite is an ephemeral cache rebuilt on demand. Multiple researchers can add papers and resolve conflicts through git.

**Minimal dependencies**: Fast startup, single binary, no heavyweight frameworks.

## Installation

```bash
go build -o bp ./cmd/bp
```

Requires Go 1.21+.

## Quick Start

```bash
# Initialize repository
bp init

# Configure PDF location (e.g., Paperpile's Google Drive folder)
bp config pdf-root ~/Google\ Drive/My\ Drive/Paperpile

# Import references
bp import --format paperpile ~/Downloads/paperpile-export.json

# Rebuild search index
bp rebuild

# Search
bp search "phylogenetics"

# Open a paper
bp open Smith2026-ab
```

## Commands

| Command | Description |
|---------|-------------|
| `bp init` | Initialize a new repository |
| `bp config [key] [value]` | Get/set configuration |
| `bp import --format paperpile <file>` | Import from Paperpile JSON |
| `bp rebuild` | Rebuild search index from source data |
| `bp search <query>` | Full-text search across titles, abstracts, authors |
| `bp list` | List all references |
| `bp get <id>` | Get a specific reference by ID |
| `bp open <id>` | Open PDF in configured viewer |
| `bp export --bibtex` | Export to BibTeX format |
| `bp check` | Validate repository integrity |

All commands output JSON by default. Use `--human` for readable output.

## Configuration

| Key | Description |
|-----|-------------|
| `pdf-root` | Path to PDF folder |
| `pdf-reader` | PDF viewer: `system`, `skim`, `zathura`, `evince`, `okular` |

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
├── refs.jsonl      # Source of truth - human-readable, git-mergeable
├── config.json     # Local configuration
└── cache/
    └── refs.db     # SQLite with FTS5 - ephemeral, gitignored
```

The JSONL file can be version-controlled and merged. The SQLite cache is rebuilt with `bp rebuild` after pulling changes.

## Collaboration Workflow

```bash
# Researcher A adds papers
bp import --format paperpile export-a.json
git add .bipartite/refs.jsonl
git commit -m "Add phylogenetics papers"
git push

# Researcher B does the same
bp import --format paperpile export-b.json
git commit -m "Add ML papers"
git push

# After pull/merge
git pull
bp rebuild  # Refresh local index
```

## Roadmap

- **Phase I** (current): Core reference manager with Paperpile import
- **Phase II**: RAG index for semantic search over abstracts
- **Phase III**: Knowledge graph with relational summaries
- **Phase IV**: Semantic Scholar integration for metadata enrichment

See [VISION.md](VISION.md) for details.

## License

MIT
