# bipartite

A context layer for research groups: connecting your internal world (projects, repos, coordination) to the external academic world (papers, citations, authors).

- **Go CLI** for querying literature, GitHub activity, and your group's knowledge graph
- **Claude Code skills** for synthesis: narrative digests, check-ins, spawning sessions with context
- **Git-backed storage** — JSONL source of truth, private by default, shareable by design

## Guides

- **[Getting Started](guides/getting-started.md)** — Installation, configuration, and first steps
- **[Reference Management](guides/reference-management.md)** — Search, import, cite, and collaborate on a git-backed paper library
- **[Knowledge Graph](guides/knowledge-graph.md)** — Connect papers, concepts, and projects with typed edges
- **[Workflow Coordination](guides/workflow-coordination.md)** — Check-ins, digests, boards, and Slack integration across repos
- **[Server Scout](guides/server-scout.md)** — Monitor remote server resources via SSH
- **[How It Works](guides/architecture.md)** — The nexus, bip CLI, and Claude Code integration explained

## Quick Start

```bash
# Install
git clone https://github.com/matsen/bipartite && cd bipartite
make install

# Create nexus from template, then configure
echo 'nexus_path: ~/re/nexus' > ~/.config/bip/config.yml

# Build index and search
bip rebuild
bip search "phylogenetics"
```

See [Getting Started](guides/getting-started.md) for full setup instructions.

## License

MIT
