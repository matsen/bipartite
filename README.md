# bipartite

**Agentic hacking like a PI.**

Computational PIs have always worked at a high level: directing teams of researchers, framing problems, choosing which experiments are worth running, and weaving the results into papers. Bipartite is a platform for bringing that workflow into the agentic age — a `bip` CLI and a library of Claude Code skills that coordinate teams of agents across GitHub issues, code repositories, manuscripts, and the literature.

The workflow runs as two coupled loops, **ideas** and **experiments**, with GitHub as the shared transport layer. On the ideas side, manuscript sessions surface new results, situate them in the literature, and turn them into well-scoped issues. On the experiments side, those issues are picked up by autonomous workers in dedicated clones, implemented, reviewed, and landed — surfacing fresh results back for discussion. Two human touchpoints anchor the otherwise-autonomous flow: high-level discussion of new findings on the ideas side, and pre-merge review on the experiments side.

## What Bipartite Does

### Ideas Coordination

For a PI, the paper is the unit that ties a team's work together. Manuscript sessions (`/bip.ms`) operate at that level: they track EPIC issues across code repositories and react when new results arrive. High-level discussion of findings, grounded in the literature via `/bip.lit`, becomes new issues, which are validated against project conventions before being filed.

Key skills: `/bip.ms`, `/bip.ms.poll`, `/bip.lit`, `/bip.issue.file`, `/bip.issue.check`, `/bip.issue.next`

### Agent Orchestration (the experiments side, EPIC workflow)

The experiments side is the **EPIC orchestration system** — a conductor/worker pattern for managing multiple Claude Code sessions across clones and worktrees. A conductor session stays on `main`, scans GitHub for open issues, and spawns workers in dedicated `tmux` windows. Workers implement, test, and create PRs autonomously. Two subagents keep the loop honest: an `issue-lead` evaluates progress from file-based state and escalates only when human judgment is needed, and a `surprising-conclusion-skeptic` interrogates strong or negative claims before they propagate. Quality gates and PR landing close the loop, with follow-up issues flowing back to the ideas side.

Key skills: `/bip.epic`, `/bip.epic.spawn`, `/bip.epic.poll`, `/bip.epic.handoff`, `/bip.pr.review`, `/bip.land`

### Workflow Coordination

Cross-cutting tools that span both sides of the workflow: themed narrative digests, cross-repo check-ins that spawn dedicated `tmux` windows for review, Slack integration, and server resource scouting via SSH.

Key skills: `/bip.checkin`, `/bip.digest`, `/bip.narrative`, `/bip.spawn`, `/bip.scout`

### Reference Management

The library backing `/bip.lit` is an agent-first reference manager with JSON output, a CLI interface, git-backed JSONL storage, and search via Semantic Scholar and Asta. Because the storage format is JSONL, your library is mergeable across collaborators using standard git workflows.

Guide: [Reference Management](https://matsen.github.io/bipartite/guides/reference-management/)

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
