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

1. [Install bipartite](guides/getting-started.md#installation)

2. Create your [nexus](guides/architecture.md) from the [nexus-template](https://github.com/matsen/nexus-template) (click "Use this template"), then clone it:

```bash
git clone https://github.com/YOUR_USERNAME/nexus ~/path/to/nexus
```

3. Point bip to your nexus:

```bash
mkdir -p ~/.config/bip
echo 'nexus_path: ~/path/to/nexus' > ~/.config/bip/config.yml
```

4. Build the index and try it out:

```bash
bip rebuild
bip search "phylogenetics"
bip s2 add DOI:10.1038/s41586-021-03819-2
```

See [Getting Started](guides/getting-started.md) for full setup instructions.

## License

MIT
