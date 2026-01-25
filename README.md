# bipartite

A command-line reference manager designed for AI agents and researchers. Import from Paperpile, search with full-text, open PDFs, export to BibTeX.

The name comes from a bipartite graph connecting two worlds: the researcher's artifacts (notes, code, concepts) and the academic literature (papers, citations, authors).

## What Makes Bipartite Special

### Agent-First in an Agent-First World

Traditional reference managers—Zotero, Mendeley, Paperpile—are GUI applications designed for humans clicking through menus. Bipartite inverts this. As AI coding agents move to the terminal, research tools need to follow. Bipartite outputs JSON by default, operates entirely via CLI, and needs no MCP server—agents just use bash. When an agent helps you write a paper, it can search your library, find relevant citations, and open PDFs for you to read, all through natural command-line interaction.

### Your Library Lives in Git

Most reference managers lock your data in proprietary databases or cloud services. Bipartite uses JSONL as the single source of truth—human-readable text files that git handles naturally. This means:

- **Real collaboration**: Multiple researchers add papers independently, resolve conflicts through standard git merges
- **Full provenance**: Every change tracked, every decision auditable, complete history preserved
- **No vendor lock-in**: Your data is portable text, not trapped in a database
- **Reproducibility**: Git enables transparent, reproducible science—your reference library should be part of that

The SQLite index is ephemeral, rebuilt on demand from the source JSONL. Pull changes, run `bip rebuild`, and you're synchronized.

### A Knowledge Graph That Includes Your Ideas

Tools like Semantic Scholar and ResearchRabbit are powerful, but they only know about published papers. Your research group has concepts, methods, and ideas that don't exist in the public literature yet. Bipartite's concept graph bridges this gap:

- Define concepts private to your group (a new algorithm you're developing, a hypothesis you're testing)
- Link papers to your concepts: "Paper A applies our method", "Paper B critiques the same problem we're solving"
- Build a knowledge graph where the published literature connects to your unpublished work

This is the bipartite vision—one side is the public academic graph, the other is your private research world, and edges connect them with semantic meaning.

### Deep Integration with Academic Search

Bipartite integrates two complementary services from Allen AI:

- **Semantic Scholar (S2)**: Structured database access to 200M+ papers. Add papers by DOI, track citations, discover gaps in your collection.
- **ASTA (Academic Search Tool API)**: LLM-powered discovery that searches like an expert researcher. Find specific passages within papers, get relevancy-ranked results with evidence.

S2 answers "give me this paper's citations" (structured). ASTA answers "find papers discussing convergence of variational inference in phylogenetics" (semantic). Together, they make literature discovery a conversation, not a keyword hunt.

### Single Binary, No Infrastructure

No database server. No heavyweight frameworks. No configuration complexity. Bipartite is a single Go binary with fast startup. Install it, run `bip init`, and you're working. The ephemeral SQLite cache means you never manage database state—if something goes wrong, delete the cache and rebuild. Local semantic search over your abstracts via Ollama embeddings means you can find conceptually related papers without external API calls. This simplicity matters when agents need to operate autonomously.

## A Workflow in Practice

To make this concrete, here's how a research group might use bipartite:

- **Alice**, a graduate student, learns of two new papers—X and Y—through her research notifications feed. She downloads the PDFs to her Paperpile folder.

- Her **coding agent reads both papers**. It determines that X is directly relevant to their current project, but Y is only tangentially related. However, Y cites an earlier paper Z that turns out to be central to the group's research program.

- The agent **fetches paper Z** via Semantic Scholar, then adds both X and Z to Alice's local bipartite database with `bip s2 add`. It creates edges linking X and Z to the group's concepts.

- `bip open` **opens both PDFs** so Alice can verify they're indeed relevant. She reads them, confirms the agent's judgment, and approves.

- The agent **commits and pushes** to the group's shared paper repository with a message explaining that X is directly relevant to their current project and Z is a foundational paper cited by Y.

- **Bernadetta**, the P.I., pulls these new papers to her machine and runs `bip rebuild`. Her agents scan the additions against her in-progress manuscripts.

- An agent determines that **Paper X should be cited** in one of her drafts and adds it to the references at an appropriate location, updating the manuscript's `.bib` file.

- Reading the commit message, Bernadetta realizes they should **compare their method to Paper Z's approach**. She spins up coding agents to develop such a comparison, with Z's citation already in hand.

## Design Principles

**Agent-first**: CLI is the primary interface. JSON output by default. No MCP server needed—agents use bash directly.

**Git-versionable**: JSONL is the source of truth, human-readable and merge-friendly. SQLite is an ephemeral cache rebuilt on demand. Multiple researchers can add papers and resolve conflicts through git.

**Minimal dependencies**: Fast startup, single binary, no heavyweight frameworks.

## GitHub Activity & Project Management

Bipartite includes built-in GitHub activity tracking and project board management:

| Command | Description |
|---------|-------------|
| `bip checkin` | Check recent GitHub activity across tracked repos |
| `bip checkin --summarize` | Generate LLM summaries for each item |
| `bip checkin --since 2d` | Check activity from the last 2 days |
| `bip checkin --repo org/repo` | Check a single repo |
| `bip checkin --all` | Show all activity (disable ball-in-my-court filtering) |
| `bip board list` | List items on project boards by status |
| `bip board add 123 --repo org/repo` | Add issue to board |
| `bip board move 123 --status Done --repo org/repo` | Move issue to new status |
| `bip board remove 123 --repo org/repo` | Remove issue from board |
| `bip board sync` | Compare P0 beads with board items |
| `bip board sync --fix` | Auto-add missing P0 beads to board |
| `bip spawn org/repo#123` | Spawn tmux window for issue review |
| `bip digest --channel <name>` | Preview activity digest (use `--post` to send to Slack) |
| `bip digest --channel <name> --verbose` | Include PR/issue body summaries |
| `bip tree` | Generate interactive HTML tree of beads issues |
| `bip tree --open` | Generate and open in browser |

These commands require a `sources.json` configuration file in the current directory (the "nexus" directory).

## Installation

### bip (Go)

```bash
# Build and install globally (recommended)
go install ./cmd/bip

# Or build locally and symlink
go build -o bip ./cmd/bip
ln -sf $(pwd)/bip ~/.local/bin/bip
```

After installation, `bip` is available globally. Run commands from your nexus directory, which contains both the reference library (`.bipartite/`) and GitHub activity config (`sources.json`).

Requires Go 1.21+ and `~/go/bin` or `~/.local/bin` in your PATH.

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
| `bip open <id>...` | Open PDFs by ID (supports multiple) |
| `bip open --recent N` | Open N most recently added papers |
| `bip open --since <commit>` | Open papers added since commit |
| `bip export --bibtex [<id>...]` | Export to BibTeX (all or specific papers) |
| `bip export --bibtex --append <file> <id>...` | Append to .bib with deduplication |
| `bip diff` | Show papers added/removed since last commit |
| `bip new --since <commit>` | List papers added since commit |
| `bip new --days N` | List papers added in last N days |
| `bip check` | Validate repository integrity |
| `bip groom` | Detect orphaned edges; use `--fix` to remove |
| `bip resolve` | Domain-aware merge conflict resolution for refs.jsonl |
| `bip resolve --dry-run` | Preview conflict resolution without modifying files |
| `bip resolve --interactive` | Interactively resolve true conflicts |
| `bip dedupe --dry-run` | Show duplicate papers (by source ID) |
| `bip dedupe --merge` | Merge duplicates: keep first, update edges |

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
| `bip edge list` | List all edges in the knowledge graph |
| `bip edge list <paper-id>` | List edges for a specific paper (`--incoming`, `--all`) |
| `bip edge list --paper <id>` | Same as above (flag form) |
| `bip edge list --concept <id>` | List edges involving a concept |
| `bip edge list --project <id>` | List edges involving a project |
| `bip edge search --type <type>` | Find edges by relationship type |
| `bip edge export` | Export edges to JSONL (`--paper` to filter) |

Relationship types: `cites`, `extends`, `contradicts`, `implements`, `applies-to`, `builds-on` (custom types also allowed).

### Concept Commands

Concepts are named ideas, methods, or phenomena that papers relate to. They enable organizing your library by topic.

| Command | Description |
|---------|-------------|
| `bip concept add <id> --name <name>` | Create a concept with optional `--aliases`, `--description` |
| `bip concept get <id>` | Get a concept by ID |
| `bip concept list` | List all concepts |
| `bip concept update <id>` | Update concept `--name`, `--aliases`, or `--description` |
| `bip concept delete <id>` | Delete concept (use `--force` if edges exist) |
| `bip concept papers <id>` | Find papers linked to a concept (`--type` to filter) |
| `bip concept merge <source> <target>` | Merge one concept into another |
| `bip paper concepts <id>` | Find concepts linked to a paper (`--type` to filter) |

Paper-concept relationship types: `introduces`, `applies`, `models`, `evaluates-with`, `critiques`, `extends`.

### Project Commands

Projects represent ongoing research work (papers being written, software tools). They connect to concepts, forming the complete bipartite graph: papers ↔ concepts ↔ projects.

| Command | Description |
|---------|-------------|
| `bip project add <id> --name <name>` | Create a project with optional `--description` |
| `bip project get <id>` | Get a project by ID |
| `bip project list` | List all projects |
| `bip project update <id>` | Update project `--name` or `--description` |
| `bip project delete <id>` | Delete project (use `--force` if repos/edges exist) |
| `bip project repos <id>` | List repos belonging to a project |
| `bip project concepts <id>` | List concepts linked to a project (`--type` to filter) |
| `bip project papers <id>` | List papers transitively linked via concepts |

Concept-project relationship types: `implemented-in`, `applied-in`, `studied-by`, `introduces`, `refines`.

### Repo Commands

Repos are GitHub repositories belonging to projects. They store metadata but cannot have edges.

| Command | Description |
|---------|-------------|
| `bip repo add <github-url> --project <id>` | Add GitHub repo to project (auto-fetches metadata) |
| `bip repo add --manual --project <id> --id <id> --name <name>` | Add non-GitHub repo |
| `bip repo get <id>` | Get a repo by ID |
| `bip repo list` | List all repos (`--project` to filter) |
| `bip repo update <id>` | Update repo `--name`, `--description`, or `--topics` |
| `bip repo delete <id>` | Delete a repo |
| `bip repo refresh <id>` | Re-fetch GitHub metadata |

Accepts GitHub URLs or shorthand: `matsen/bipartite` or `https://github.com/matsen/bipartite`.

### Visualization Commands

| Command | Description |
|---------|-------------|
| `bip viz` | Generate interactive HTML knowledge graph to stdout |
| `bip viz --output <file>` | Generate to file |
| `bip viz --layout <type>` | Layout: `force` (default), `circle`, `grid` |
| `bip viz --offline` | Bundle Cytoscape.js inline for offline use |

The visualization shows papers (blue circles) and concepts (orange diamonds) with edges colored by relationship type. Hover for details, click to highlight connections.

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
├── concepts.jsonl  # Concept nodes - git-mergeable
├── projects.jsonl  # Project nodes - git-mergeable
├── repos.jsonl     # Repo nodes - git-mergeable
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

### Resolving Merge Conflicts

When two researchers add the same paper independently (common with popular papers), git sees a conflict in refs.jsonl. But bip understands paper metadata:

```bash
# After git merge with conflicts in refs.jsonl
bip resolve --dry-run    # Preview what would happen
bip resolve              # Auto-resolve: keep more complete version, merge complementary metadata

# For true conflicts (same field, different values)
bip resolve --interactive  # Prompts for each unresolvable field
```

Resolution logic:
- **Same paper, different completeness**: Keeps the version with more metadata (abstract, authors, venue)
- **Complementary metadata**: Merges both (e.g., one has abstract, other has venue)
- **Different papers**: Includes both
- **True conflicts**: Requires `--interactive` (both have different abstracts, for example)

## Roadmap

- **Phase I** ✓: Core reference manager with Paperpile import
- **Phase II** ✓: RAG index for semantic search over abstracts
- **Phase III-a** ✓: Knowledge graph with directed edges between papers
- **Phase III-b** ✓: Concept nodes and artifact connections
- **Phase IV** ✓: Semantic Scholar integration for metadata enrichment

See [VISION.md](VISION.md) for details.

## License

MIT
