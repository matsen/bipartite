# bipartite

**Agentic hacking like a PI.** bipartite runs a research group's agents: Claude Code sessions, each with one job. The agents take the operations, and you keep the science.

The agents implement ideas and report what they find. You turn their results into talks for your group, and the group's questions drive the next work. A manuscript session ties each result to the literature. A change that could touch the science comes to you, and a purely operational change lands without you.

![Talk-driven development: four layers of knowledge, from research up to the field, with a support layer below](images/tier4/talk-driven-development.svg)

**[Driving Science with a Fleet of Agents](guides/driving-science.md)** explains this picture, and three more that zoom in on an issue, a worker, and a fix to the skills.

## Start here

- **[Where to Start](guides/where-to-start.md)** — Adopt bipartite in four tiers, from one Claude session to a fleet that drives the science. Each tier ends with a test for when to move on.
- **[Getting Started](guides/getting-started.md)** — Installation, configuration, and first steps
- **[Agent Roles](guides/roles.md)** — What each long-lived session owns, whom it answers, and which skill starts it.
- **[PI Questions](guides/pi-questions.md)** — The questions a PI asks of a piece of work. The sessions ask them too.

## Guides

bipartite is a Go CLI (`bip`) and a set of Claude Code skills. Its data lives in a git repo, the nexus, as JSONL.

- **[Configuration](guides/configuration.md)** — Every config option, tokens, and environment variables
- **[Issue Lifecycle](guides/issue-lifecycle.md)** — Which skill to run at each step from draft issue to landed PR
- **[Reference Management](guides/reference-management.md)** — Search, import, cite, and collaborate on a git-backed paper library
- **[Projects, Repos, and Stores](guides/projects-and-stores.md)** — Track projects and repositories, and keep custom records in generic stores
- **[Workflow Coordination](guides/workflow-coordination.md)** — Check-ins, digests, and Slack integration across repos
- **[Worktree Layout](guides/layout.md)** — Opt-in per-issue git worktrees for `bip spawn`
- **[Continuation Prompt](guides/continuation-prompt.md)** — Carrying a session across a context reset with `/bip-tuckin` and `/bip-continue`
- **[Server Scout](guides/server-scout.md)** — Monitor remote server resources via SSH
- **[How It Works](guides/architecture.md)** — The nexus, bip CLI, and Claude Code integration explained

## License

MIT
