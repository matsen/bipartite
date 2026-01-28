# bipartite

## The Problem

### Knowledge

In 2025, agentic coding completely revolutionized my role as an individual researcher because it shortened the distance between idea and implementation. However, it didn't really help my role as a research team lead, because agents don't have all of my knowledge of what we are doing and how it fits together.

_Let's tell the agents about everything._

### Tools

Last year I sent 8,077 Slack messages, made 1,607 commits, reviewed 186 PRs, created 250 issues, and evaluated hundreds of papers for relevance to our research. I'd like to get agentic help with this in 2026. In order to do so, an agent is going to need to work with all of these tools.

_Let's connect the agents to everything._

## What Bipartite Does

Bipartite is a tool to give agents that knowledge graph and access to all the tools they need to draw context from our workflow.

- **[Reference Management](https://matsen.github.io/bipartite/guides/reference-management/)** — An agent-first reference manager: JSON output, CLI interface, git-backed storage, Semantic Scholar and Asta search. JSONL means your library is mergeable across collaborators with standard git workflows.
- **[Knowledge Graph](https://matsen.github.io/bipartite/guides/knowledge-graph/)** — This knowledge graph one connects the literature to _your group's projects_ and is designed for agents to traverse.
- **[Workflow Coordination](https://matsen.github.io/bipartite/guides/workflow-coordination/)** — Themed digests, cross-repo check-ins (spawn dedicated `tmux` windows!), and Slack integration for group leaders.

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
