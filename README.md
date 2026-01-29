# bipartite

## The Problem

### Knowledge

In 2025, agentic coding completely revolutionized my role as an individual researcher because it shortened the distance between idea and implementation. However, it didn't really help my role as a research team lead, because agents don't have all of my knowledge of what we are doing and how it fits together.

Let's tell the agents about everything!

### Tools

Last year I sent 8,077 Slack messages, reviewed 186 PRs, created 250 issues, and evaluated hundreds of papers for relevance to our research. I'd like to get agentic help with this in 2026. In order to do so, an agent is going to need to work with all of these tools.

Let's connect the agents to everything!

## What Bipartite Does

Bipartite is a tool to give agents knowledge and access to all the tools they need to draw context from our workflow.

- **[Reference Management](https://matsen.github.io/bipartite/guides/reference-management/)** — An agent-first reference manager: JSON output, CLI interface, git-backed storage, Semantic Scholar and Asta search. JSONL means your library is mergeable across collaborators with standard git workflows.
- **[Knowledge Graph](https://matsen.github.io/bipartite/guides/knowledge-graph/)** — This knowledge graph connects the literature to _your group's projects_ and is designed for agents to traverse.
- **[Workflow Coordination](https://matsen.github.io/bipartite/guides/workflow-coordination/)** — Themed digests, cross-repo check-ins (spawn dedicated `tmux` windows!), and Slack integration for group leaders.
- **[Server Scout](https://matsen.github.io/bipartite/guides/server-scout/)** — Check remote server CPU, memory, load, and GPU availability via native SSH.

## Quick Start

The fastest way to get started is with the [nexus-template](https://github.com/matsen/nexus-template):

1. Click **[Use this template](https://github.com/matsen/nexus-template/generate)** to create your nexus repo
2. Clone it and run:

```bash
bip rebuild
bip search "phylogenetics"
bip s2 add DOI:10.1038/s41586-021-03819-2
```

See the [Getting Started guide](https://matsen.github.io/bipartite/guides/getting-started/) for full setup instructions.

## Installation

```bash
go install ./cmd/bip
export PATH="$HOME/go/bin:$PATH"
```

Requires Go 1.24+. For Claude Code skills:

```bash
git clone https://github.com/matsen/bipartite
cd bipartite
ln -s $(pwd)/.claude/skills/* ~/.claude/skills/
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `S2_API_KEY` | Optional | Semantic Scholar API key (higher rate limits for `bip s2` commands) |
| `ASTA_API_KEY` | For `bip asta` | ASTA API key for academic search (`bip asta search`, `bip asta snippets`) |
| `SLACK_BOT_TOKEN` | For Slack features | Slack bot token (`channels:history`, `channels:read`, `users:read` scopes) |

## License

MIT
