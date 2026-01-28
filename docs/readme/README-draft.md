# bipartite

## The Problem

Here is a picture of my world:

![My digital world](diagram1.svg)

In 2025, I sent 8,077 Slack messages and evaluated hundreds of papers for relevance to our research.

| Activity | Count |
|----------|-------|
| Commits | 1,607 |
| PRs reviewed | 186 |
| Issues created | 250 |

I'd like to get agentic help with this in 2026. In order to do so, an agent is going to need to:
1. Work with these services
2. Relate information between within-group activities and the literature

Here is a more detailed picture of my world:

![Knowledge graph structure](diagram2.svg)

In 2025, agentic coding completely revolutionized my role as an individual researcher because it shortened the distance between idea and implementation. However, it didn't really help my role as a research team lead, because agents don't have all of my knowledge graph.

Bipartite is a tool to give agents that knowledge graph.

## What Bipartite Is

* **Agent-first**: A Go binary wrapped in Claude Code skills
* **Secure**: All of your knowledge lives in your private repository
* **Collaborative**: Data stores are designed for merging
* **Comprehensive**: Draws from Slack, GitHub, and Paperpile
* **Fast**: Local databases reduce the need for frequent API calls

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
