# bipartite

Bipartite is:

* A **Go CLI** for querying local and remote academic literature, GitHub activity across repos, and your group's knowledge graph
* A collection of **Claude Code skills** for synthesis: narrative digests, interactive check-ins, spawning sessions with context
* A **git-backed system** that keeps your group's papers, concepts, projects, and coordination state in one private repository

The name refers to the graph at the core: one side is your world (projects, repos, concepts, active work), the other is the literature (papers, citations, authors), with typed edges connecting them.

## The Problem

Agentic coding has transformed individual contributor (IC) work. A researcher can pair with Claude to write code and debug pipelines. But agents haven't had an equivalent effect for group leaders (PIs, team leads, managers) who need to:

- Track activity across many repositories
- Connect ongoing work to relevant literature
- Identify what needs attention and make decisions

The context for these tasks is scattered across GitHub, Slack, reference managers, and memory. Agents can't help because they can't access it.

Bipartite creates a **context layer** that both agents and humans can query: connecting your internal research world to the external academic world.

## Is This For You?

I wrote this for me. It might be for you if:

- You're happiest spending your working life in the terminal
- Your work concerns text files (e.g. `.tex` and `.py`)
- GitHub is how your group coordinates

This isn't true for everyone, and that's fine. 

## How It Works

The CLI provides fast local operations: searching papers, checking GitHub activity, managing references. When you need synthesis (narrative digests, summarization, connecting ideas), Claude Code skills layer intelligence on top.

The knowledge graph connects your internal world (projects, repos, concepts) to the external academic world (papers, citations, authors). Edges carry meaning: a paper *introduces* a concept, a project *implements* a method. But the graph is one part of the system, not the whole thing.

### Three Audiences, One Tool

| Audience | What they need | How bipartite helps |
|----------|----------------|---------------------|
| **Agents** | Structured context, CLI access | JSON by default, bash interface, no MCP server needed |
| **ICs** | Track work, find papers, manage references | Fast local commands, git-synced library, PDF integration |
| **Group leaders** | Visibility across repos, decision support, team communication | Themed digests, board sync, attention filtering |

### Design Principles

**Agent-native, human-friendly.** CLI-first with JSON output by default. Agents call commands via bash. Humans add `--human` for readable output.

**Private by default, shareable by design.** Everything lives in git-versionable files (JSONL). You control what's committed and shared. Multiple researchers can collaborate through standard git workflows.

**Fast core (Go) + smart synthesis (Claude Skills).** The CLI is a single Go binary. `bip search` returns in milliseconds. When you need synthesis (narrative digests, summarization), Claude Code skills like `/bip.narrative` provide it.

## What It Does

### For Agents: Structured Context

Traditional reference managers are GUI applications designed for humans. Bipartite provides a CLI that agents can use directly:

```bash
# Search the library
bip search "variational inference" --limit 5

# Get paper details as JSON
bip get Smith2024-ab

# Find papers linked to a concept
bip concept papers phylogenetic-inference

# Check what needs attention across repos
bip checkin --since 2d
```

Agents read JSON output and call commands via bash. No MCP server required.

### For ICs: Reference Management + Workflow

Your paper library lives in git, not a proprietary database:

```bash
# Import from Paperpile
bip import --format paperpile export.json

# Add papers via DOI
bip s2 add DOI:10.1038/s41586-024-07487-w

# Open PDFs
bip open Smith2024-ab

# Export citations
bip export --bibtex --append paper.bib Smith2024-ab Jones2023-cd

# Build knowledge graph
bip edge add -s Kingma2014-mo -t variational-autoencoder -r introduces
```

Pull changes from collaborators, run `bip rebuild`, and your index updates. `bip resolve` handles merge conflicts in refs.jsonl using paper metadata.

### For Group Leaders: Coordination + Visibility

See across all your team's repos without checking each one:

```bash
# What needs my attention?
bip checkin

# What happened this week?
bip checkin --since 7d --summarize

# Generate a narrative digest organized by research themes
/bip.narrative dasm2 --verbose

# Post to Slack
bip digest --channel dasm2 --post

# Spawn a Claude session with issue context pre-loaded
bip spawn org/repo#123
```

Filtering shows only items waiting for your action. Digests organize work by research theme rather than by repository.

## A Workflow in Practice

**Alice**, a graduate student, downloads two new papers to her Paperpile folder.

Her **coding agent reads both papers** using the pdf-navigator MCP. It determines Paper X is directly relevant; Paper Y is tangential but cites a foundational Paper Z.

The agent **fetches Paper Z** via Semantic Scholar, adds X and Z to the library with `bip s2 add`, and creates edges linking them to the group's concepts.

`bip open` **opens the PDFs** for Alice to verify. She confirms, the agent commits and pushes.

**Bernadetta**, the PI, pulls the changes and runs `bip rebuild`. Her agent scans the additions against her manuscripts and adds Paper X to the references where appropriate.

**Friday afternoon**, an agent runs `/bip.narrative` to generate a themed digest of the week's work across all repos. Bernadetta reviews it and posts to Slack with `bip digest --post`.

## Installation

```bash
# Build and install
go install ./cmd/bip

# Or build locally
go build -o bip ./cmd/bip
```

Requires Go 1.21+ and `~/go/bin` in your PATH.

## Quick Start

```bash
# Initialize
bip init

# Configure PDF location
bip config pdf-root ~/Google\ Drive/My\ Drive/Paperpile

# Import references
bip import --format paperpile ~/Downloads/export.json

# Build search index
bip rebuild

# Search
bip search "phylogenetics"
```

## Commands

### Reference Management

| Command | Description |
|---------|-------------|
| `bip init` | Initialize repository |
| `bip import --format paperpile <file>` | Import from Paperpile |
| `bip rebuild` | Rebuild search index |
| `bip search <query>` | Full-text search |
| `bip get <id>` | Get paper by ID |
| `bip open <id>` | Open PDF |
| `bip export --bibtex [<id>...]` | Export to BibTeX |
| `bip diff` | Show papers added/removed since last commit |
| `bip resolve` | Smart merge conflict resolution |

### Semantic Scholar Integration

| Command | Description |
|---------|-------------|
| `bip s2 add <paper-id>` | Add paper by DOI, arXiv, or S2 ID |
| `bip s2 lookup <paper-id>` | Look up without adding |
| `bip s2 citations <paper-id>` | Find citing papers |
| `bip s2 gaps` | Discover highly-cited papers you're missing |

### Knowledge Graph

| Command | Description |
|---------|-------------|
| `bip edge add -s <source> -t <target> -r <type>` | Add edge |
| `bip edge list [<paper-id>]` | List edges |
| `bip concept add <id> --name <name>` | Create concept |
| `bip concept papers <id>` | Find linked papers |
| `bip project add <id> --name <name>` | Create project |
| `bip viz` | Generate interactive graph visualization |

### Team Coordination

| Command | Description |
|---------|-------------|
| `bip checkin` | Check GitHub activity across repos |
| `bip checkin --summarize` | With LLM summaries |
| `bip board list` | View project board |
| `bip board sync` | Sync priorities with board |
| `bip spawn org/repo#123` | Launch Claude with issue context |
| `bip digest --channel <name>` | Preview Slack digest |
| `bip digest --channel <name> --post` | Post to Slack |
| `bip tree --open` | View task hierarchy in browser |

### Claude Code Skills

| Skill | Description |
|-------|-------------|
| `/bip.narrative <channel>` | Generate themed prose digest |
| `/bip.checkin` | Interactive activity check-in |
| `/bip.digest` | Generate and post Slack digest |
| `/bip.spawn` | Launch Claude session with context |
| `/bip.board` | Project board operations |
| `/bip.tree` | Task hierarchy visualization |

## Data Storage

```
.bipartite/
├── refs.jsonl      # Papers (source of truth)
├── edges.jsonl     # Knowledge graph edges
├── concepts.jsonl  # Concept nodes
├── projects.jsonl  # Project nodes
├── repos.jsonl     # Repo nodes
├── config.json     # Local config
└── cache/
    └── refs.db     # SQLite index (ephemeral, gitignored)
```

JSONL files are human-readable and git-mergeable. The SQLite cache rebuilds on demand.

## Configuration

| Key | Description |
|-----|-------------|
| `pdf-root` | Path to PDF folder |
| `pdf-reader` | PDF viewer: `system`, `skim`, `zathura`, `evince`, `okular` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `S2_API_KEY` | Semantic Scholar API key (optional, for higher rate limits) |
| `ASTA_API_KEY` | ASTA API key (required for `bip asta` commands) |

## License

MIT
