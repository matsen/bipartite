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

## Installation

### Full Installation (recommended)

This installs the `bip` CLI plus Claude Code agents and skills:

```bash
git clone https://github.com/matsen/bipartite
cd bipartite
make install
```

Prerequisites:
- Go 1.24+
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code)

Verify with `bip --help`.

### CLI Only

If you just want the `bip` CLI without agents/skills:

```bash
go install github.com/matsen/bipartite/cmd/bip@latest
```

## Quick Start

2. **Create your private [nexus](https://matsen.github.io/bipartite/guides/architecture/)** — the repository that stores your paper library, knowledge graph, and workflow config. Click "Use this template" on [nexus-template](https://github.com/matsen/nexus-template), then clone:

```bash
git clone https://github.com/YOUR_USERNAME/nexus ~/re/nexus
```

3. **Point bip to your nexus** (minimal config to get started):

```bash
mkdir -p ~/.config/bip
echo 'nexus_path: ~/re/nexus' > ~/.config/bip/config.yml
```

4. **Build the index and try it out**:

```bash
bip rebuild
bip search "phylogenetics" --human
bip s2 add DOI:10.1038/s41586-021-03819-2
```

See the [Getting Started guide](https://matsen.github.io/bipartite/guides/getting-started/) for full setup instructions.

## Configuration

For full functionality, add API keys ([Semantic Scholar](https://www.semanticscholar.org/product/api#api-key), [Asta](https://allenai.org/asta/resources/mcp), [GitHub](https://github.com/settings/tokens), [Slack](https://api.slack.com/apps)) to your config:

```yaml
nexus_path: ~/re/nexus
s2_api_key: your-key
asta_api_key: your-key
github_token: ghp_...
slack_bot_token: xoxb-...
```

See the [Configuration Guide](https://matsen.github.io/bipartite/guides/configuration/) for all options.

## License

MIT
