# bipartite

## The Problem

Here is a picture of my world:

![My digital world](docs/readme-prep/diagram1.svg)

In 2025, I sent 8,077 Slack messages, mass commits reviews, and evaluated hundreds of papers for their relevance to our research. Agentic coding completely revolutionized my role as an individual contributor—it shortened the distance between idea and implementation. But it didn't really help my role as a research team lead, because agents don't have access to my knowledge graph.

For agents to help with coordination, they need to:
1. Work with these services (Slack, GitHub, literature)
2. Relate information between within-group activities and the external literature

## The Solution

Bipartite creates a context layer that both agents and humans can query.

![Knowledge graph structure](docs/readme-prep/diagram2.svg)

The name refers to the graph at the core: one side is your world (projects, repos, concepts), the other is the literature (papers, citations, authors), with typed edges connecting them through shared concepts.

## What Bipartite Is

* **Agent-first**: A Go binary wrapped in Claude Code skills—agents call commands via bash, no MCP server needed
* **Secure**: All of your knowledge lives in your private git repository
* **Collaborative**: Data stores use JSONL, designed for merging across team members
* **Comprehensive**: Draws from Slack, GitHub, and Paperpile
* **Fast**: Local SQLite indexes reduce the need for frequent API calls

## Installation

### CLI Binary

```bash
go install ./cmd/bip
```

Requires Go 1.21+. Add your Go bin directory to PATH:

```bash
export PATH="$HOME/go/bin:$PATH"
```

### Claude Code Skills

The skills (`/bip.narrative`, `/bip.checkin`, etc.) require [Claude Code](https://claude.ai/code):

```bash
git clone https://github.com/matsen/bipartite
cd bipartite
ln -s $(pwd)/.claude/skills/* ~/.claude/skills/
```

## Quick Start

```bash
bip init
bip config pdf-root ~/Google\ Drive/My\ Drive/Paperpile
bip import --format paperpile ~/Downloads/export.json
bip rebuild
bip search "phylogenetics"
```

## Commands

### Reference Management

| Command | Description |
|---------|-------------|
| `bip search <query>` | Full-text search |
| `bip get <id>` | Get paper by ID |
| `bip open <id>` | Open PDF |
| `bip s2 add <paper-id>` | Add paper by DOI, arXiv, or S2 ID |
| `bip export --bibtex [<id>...]` | Export to BibTeX |

### Knowledge Graph

| Command | Description |
|---------|-------------|
| `bip edge add -s <source> -t <target> -r <type>` | Add edge |
| `bip concept papers <id>` | Find linked papers |
| `bip viz` | Generate interactive graph visualization |

### Team Coordination

| Command | Description |
|---------|-------------|
| `bip checkin` | Check GitHub activity across repos |
| `bip board list` | View project board |
| `bip spawn org/repo#123` | Launch Claude with issue context |
| `bip digest --channel <name> --post` | Post to Slack |

### Claude Code Skills

| Skill | Description |
|-------|-------------|
| `/bip.narrative <channel>` | Generate themed prose digest |
| `/bip.checkin` | Interactive activity check-in |
| `/bip.spawn` | Launch Claude session with context |

## Data Storage

```
.bipartite/
├── refs.jsonl      # Papers (source of truth)
├── edges.jsonl     # Knowledge graph edges
├── concepts.jsonl  # Concept nodes
├── projects.jsonl  # Project nodes
└── cache/
    └── refs.db     # SQLite index (ephemeral)
```

JSONL files are human-readable and git-mergeable. The SQLite cache rebuilds on demand.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `S2_API_KEY` | Semantic Scholar API key (optional) |
| `SLACK_BOT_TOKEN` | Slack bot token for reading channels |

## License

MIT
