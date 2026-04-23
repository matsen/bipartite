# bipartite

**Agentic hacking like a PI.**

Computational PIs have always worked at a high level: directing teams of researchers, framing problems, choosing which experiments are worth running, and weaving the results into papers. Bipartite is a platform for bringing that workflow into the agentic age — a `bip` CLI and a library of Claude Code skills that coordinate teams of agents across GitHub issues, code repositories, manuscripts, and the literature.

## What Bipartite Does

### Agent Orchestration (EPIC workflow)

The core of bipartite is the **EPIC orchestration system** — a conductor/worker pattern for managing multiple Claude Code sessions across clones and worktrees. The conductor session stays on `main`, scans GitHub for open issues, and spawns workers in dedicated `tmux` windows. Workers implement, test, and create PRs autonomously; an issue-lead subagent evaluates progress and escalates only when human judgment is needed.

Key skills: `/bip.epic`, `/bip.epic.spawn`, `/bip.epic.poll`, `/bip.epic.handoff`, `/bip.epic.tuckin`

### Manuscript Coordination

For a PI, the paper has traditionally been the unit that ties a team's work together. Manuscript sessions (`/bip.ms`) operate at that level: they track EPIC issues across code repositories and react when new results arrive — pulling data, importing figures, and keeping the manuscript in sync with the science.

Key skills: `/bip.ms`, `/bip.ms.poll`, `/bip.ms.tuckin`

### Reference Management

Bipartite includes an agent-first reference manager with JSON output, a CLI interface, git-backed JSONL storage, and search via Semantic Scholar and Asta. Because the storage format is JSONL, your library is mergeable across collaborators using standard git workflows.

Guide: [Reference Management](https://matsen.github.io/bipartite/guides/reference-management/)

### Workflow Coordination

Beyond orchestration, bipartite provides themed narrative digests, cross-repo check-ins that spawn dedicated `tmux` windows, Slack integration, and server resource scouting via SSH.

Key skills: `/bip.checkin`, `/bip.digest`, `/bip.narrative`, `/bip.spawn`, `/bip.scout`

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

**Note:** This installs to `$GOBIN` if set, otherwise `$HOME/go/bin`. Ensure the appropriate directory is in your PATH.

## Quick Start

1. **Create your private [nexus](https://matsen.github.io/bipartite/guides/architecture/)** — the repository that stores your paper library, workflow config, and project context. Click "Use this template" on [nexus-template](https://github.com/matsen/nexus-template), then clone:

```bash
git clone https://github.com/YOUR_USERNAME/nexus ~/re/nexus
```

2. **Point bip to your nexus** (minimal config to get started):

```bash
mkdir -p ~/.config/bip
echo 'nexus_path: ~/re/nexus' > ~/.config/bip/config.yml
```

3. **Build the index and try it out**:

```bash
bip rebuild
bip search "phylogenetics" --human
bip s2 add DOI:10.1038/s41586-021-03819-2
```

See the [Getting Started guide](https://matsen.github.io/bipartite/guides/getting-started/) for full setup instructions.

## Configuration

For full functionality, add API keys ([Semantic Scholar](https://www.semanticscholar.org/product/api#api-key), [Asta](https://allenai.org/asta/resources/mcp), [GitHub](https://matsen.github.io/bipartite/guides/configuration/#github-authentication), [Slack](https://api.slack.com/apps)) to your config:

```yaml
nexus_path: ~/re/nexus
s2_api_key: your-key
asta_api_key: your-key
github_token: ghp_...
slack_bot_token: xoxb-...
```

See the [Configuration Guide](https://matsen.github.io/bipartite/guides/configuration/) for all options.

## Who Is This For?

Bipartite isn't just for people who hold the official PI title. It's for anyone who wants to work with a team of agents the way a PI works with a team of researchers — directing the science at a high level while detailed work runs across many parallel sessions.

## License

MIT
