# bipartite

A context layer for research groups: connecting your internal world (projects, repos, coordination) to the external academic world (papers, citations, authors).

- **Go CLI** for querying literature, GitHub activity, and your group's knowledge graph
- **Claude Code skills** for synthesis: narrative digests, check-ins, spawning sessions with context
- **Git-backed storage** — JSONL source of truth, private by default, shareable by design

## Guides

- **[Reference Management](guides/reference-management.md)** — Search, import, cite, and collaborate on a git-backed paper library
- **[Knowledge Graph](guides/knowledge-graph.md)** — Connect papers, concepts, and projects with typed edges
- **[Workflow Coordination](guides/workflow-coordination.md)** — Check-ins, digests, boards, and Slack integration across repos

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

## Quick Start

```bash
bip init
bip config pdf-root ~/Google\ Drive/My\ Drive/Paperpile
bip import --format paperpile ~/Downloads/export.json
bip rebuild
bip search "phylogenetics"
```

## License

MIT
