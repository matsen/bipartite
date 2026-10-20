# bipartite

**Agentic hacking like a PI.**

> For motivation and a walkthrough of the workflow, see the blog post: [Bipartite: manuscript-driven development with a team of agents](https://matsen.group/general/2026/06/06/bipartite.html).

Computational PIs have always worked at a high level: directing teams of researchers, framing problems, choosing which experiments are worth running, and weaving the results into papers. Bipartite is a platform for bringing that workflow into the agentic age — a `bip` CLI and a library of Claude Code skills that coordinate teams of agents across GitHub issues, code repositories, manuscripts, and the literature.

The workflow runs as two coupled loops, **ideas** and **experiments**, with GitHub as the shared transport layer. On the ideas side, manuscript sessions surface new results, situate them in the literature, and turn them into well-scoped issues. On the experiments side, those issues are picked up by autonomous workers in dedicated clones, implemented, reviewed, and landed — surfacing fresh results back for discussion. Two human touchpoints anchor the otherwise-autonomous flow: high-level discussion of new findings on the ideas side, and pre-merge review on the experiments side.

## What Bipartite Does

### Ideas Coordination

For a PI, the paper is the unit that ties a team's work together. Manuscript sessions (`/bip-ms`) operate at that level: they track EPIC issues across code repositories and react when new results arrive. High-level discussion of findings, grounded in the literature via `/bip-lit`, becomes new issues, which are validated against project conventions before being filed.

Key skills: `/bip-ms`, `/bip-ms-poll`, `/bip-lit`, `/bip-issue-file`, `/bip-issue-check`, `/bip-issue-next`, `/bip-issue-iterate`, `/bip-ms-sweep` (pre-submission polish), `/bip-ms-math` (revise a manuscript's mathematics in reviewable commits), `/bip-lit-import` (Paperpile import)

### Agent Orchestration (the experiments side, EPIC workflow)

The experiments side is the **EPIC orchestration system** — split across two roles. A topic-scoped `/bip-epic` session tracks program strategy, triages issues/PRs, and flags dependency/collision conflicts between them; a topic-agnostic `/bip-conductor` session owns the clone pool, spawns workers in dedicated `tmux` windows, and runs mechanical checks (staleness, occupancy) neither role could catch alone. The two coordinate over `SendMessage` and a shared `.spawn-prompts/` directory. Workers implement, test, and create PRs autonomously. Two subagents keep the loop honest: an `issue-lead` evaluates progress from file-based state and escalates only when human judgment is needed, and a `surprising-conclusion-skeptic` interrogates strong or negative claims before they propagate. Quality gates and PR landing close the loop, with follow-up issues flowing back to the ideas side.

Key skills: `/bip-epic`, `/bip-conductor`, `/bip-conductor-spawn`, `/bip-conductor-handoff`, `/bip-pr-review`, `/bip-pr-land`, `/bip-epic-check`, `/bip-conductor-prepare-reboot` and `/bip-conductor-recover` (host reboots)

The [Agent Roles](https://matsen.github.io/bipartite/guides/roles/) guide names the session roles (conductor, epic, manuscript, staff, worker) and what each owns.
The [Issue Lifecycle](https://matsen.github.io/bipartite/guides/issue-lifecycle/) guide gives the order in which to run the issue and PR skills, from draft to landed PR.

### Workflow Coordination

Cross-cutting tools that span both sides of the workflow: themed narrative digests, cross-repo check-ins that spawn dedicated `tmux` windows for review, Slack integration, and server resource scouting via SSH.

Key skills: `/bip-checkin`, `/bip-digest`, `/bip-narrative`, `/bip-scout`

Standalone: `/bip-helper` (a long-lived background session for side work), `/bip-decay-audit` (whole-repo decay check), `/bip-marimo` (marimo notebooks), `/bip-tuckin` and `/bip-continue` (carry a session across a context reset)

### Reference Management

The library backing `/bip-lit` is an agent-first reference manager with JSON output, a CLI interface, git-backed JSONL storage, and search via Semantic Scholar and Asta. Because the storage format is JSONL, your library is mergeable across collaborators using standard git workflows.

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

## Documentation

- [Getting Started](https://matsen.github.io/bipartite/guides/getting-started/): full setup
- [Configuration](https://matsen.github.io/bipartite/guides/configuration/): every config option and token
- [Agent Roles](https://matsen.github.io/bipartite/guides/roles/): the session roles and what each owns
- [Issue Lifecycle](https://matsen.github.io/bipartite/guides/issue-lifecycle/): which skill to run at each step, from draft issue to landed PR
- [Workflow Coordination](https://matsen.github.io/bipartite/guides/workflow-coordination/): check-ins, digests, boards, Slack, `bip spawn`
- [Reference Management](https://matsen.github.io/bipartite/guides/reference-management/): the paper library behind `/bip-lit`
- [Projects, Repos, and Stores](https://matsen.github.io/bipartite/guides/projects-and-stores/)
- [Worktree Layout](https://matsen.github.io/bipartite/guides/layout/): opt-in per-issue git worktrees
- [Continuation Prompt](https://matsen.github.io/bipartite/guides/continuation-prompt/): `/bip-tuckin` and `/bip-continue` across context resets
- [Server Scout](https://matsen.github.io/bipartite/guides/server-scout/)
- [How It Works](https://matsen.github.io/bipartite/guides/architecture/): the nexus, the CLI, and Claude Code

Every skill lives in [`skills/`](skills/); the `description:` line at the top of each `SKILL.md` says what it does.

## Configuration

Set `nexus_path` in `~/.config/bip/config.yml`:

```yaml
nexus_path: ~/re/nexus
```

For full functionality, put API keys ([ASTA/Semantic Scholar](https://allenai.org/asta/resources/mcp), [GitHub](https://matsen.github.io/bipartite/guides/configuration/#github-authentication), [Slack](https://api.slack.com/apps)) in `~/.config/bip/secrets.env` (mode 600), not in `config.yml`, which agents read routinely:

```bash
BIP_ASTA_API_KEY=your-key
BIP_GITHUB_TOKEN=ghp_...
BIP_SLACK_TOKEN=xoxb-...
```

`asta_api_key` covers both the ASTA MCP API and the Semantic Scholar Graph API — AI2 issues a single key for both.

bip looks each token up in the environment, then under the same names in `secrets.env`, then in the matching `config.yml` field (`asta_api_key`, `github_token`, `slack_bot_token`):

| Token  | Env vars consulted (in order)                             |
|--------|-----------------------------------------------------------|
| ASTA   | `BIP_ASTA_API_KEY`, `ASTA_API_KEY`                        |
| GitHub | `BIP_GITHUB_TOKEN`, `GITHUB_TOKEN`, `GH_TOKEN`            |
| Slack  | `BIP_SLACK_TOKEN`, `SLACK_BOT_TOKEN`                      |

The `BIP_`-prefixed names are recommended when a globally-exported
`GITHUB_TOKEN` (e.g. for the `gh` CLI) might have different scopes than
what you want `bip` to use.

See the [Configuration Guide](https://matsen.github.io/bipartite/guides/configuration/) for all options.

To opt into per-issue git worktrees for `bip spawn` (instead of one
shared clone per repo), see [Worktree Layout](https://matsen.github.io/bipartite/guides/layout/).

## Who Is This For?

Bipartite isn't just for people who hold the official PI title. It's for anyone who wants to work with a team of agents the way a PI works with a team of researchers — directing the science at a high level while detailed work runs across many parallel sessions.

## License

MIT
