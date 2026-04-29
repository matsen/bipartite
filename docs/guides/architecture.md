# How Bipartite Works

Bipartite has three main pieces: a **nexus** (your data), the **bip CLI** (operations), and **Claude Code skills** (AI integration).

```
┌────────────────────────────────────────────────────────────┐
│                        Your Machine                        │
│                                                            │
│  ┌────────────────┐    ┌───────────────────────────────┐   │
│  │  Claude Code   │    │      Nexus Directory          │   │
│  │                │    │   (configured via nexus_path) │   │
│  │ ┌────────────┐ │    │                               │   │
│  │ │  Skills    │─┼────┼─▶ refs.jsonl  (papers)        │   │
│  │ │            │ │    │   edges.jsonl (graph)         │   │
│  │ │ /bip.lit   │ │    │   concepts.jsonl              │   │
│  │ │ /bip.digest│ │    │   servers.yml (scout)         │   │
│  │ │ /bip.spawn │ │    │   sources.yml (GitHub)        │   │
│  │ └─────┬──────┘ │    │                               │   │
│  └───────┼────────┘    │   .bipartite/ (gitignored)    │   │
│          │             │   └── cache/refs.db           │   │
│          │             │   └── vectors.gob             │   │
│          ▼             └───────────────────────────────┘   │
│  ┌────────────────┐               ▲                        │
│  │   bip CLI      │               │                        │
│  │                │───────────────┘                        │
│  │ bip search     │                                        │
│  │ bip s2 add     │    ┌─────────────────────────────┐     │
│  │ bip digest     │◀──▶│      External APIs          │     │
│  │ bip scout      │    │ • Semantic Scholar          │     │
│  └────────────────┘    │ • GitHub (gh CLI)           │     │
│                        │ • Slack                     │     │
│                        │ • Remote servers (SSH)      │     │
│                        └─────────────────────────────┘     │
└────────────────────────────────────────────────────────────┘
```

## The Nexus

A **nexus** is a git repository containing your research data. It's the central hub that bipartite operates on.

```
my-nexus/
├── refs.jsonl            # Paper library (source of truth)
├── edges.jsonl           # Knowledge graph connections
├── concepts.jsonl        # Topic definitions
│
├── servers.yml           # Remote servers for bip scout
├── sources.yml          # GitHub repos for activity tracking
├── config.yml           # Local settings (PDF paths, etc.)
│
├── context/              # Project context files
├── narrative/            # Generated digest output
│
└── .bipartite/           # Cache directory (gitignored)
    ├── cache/refs.db     # SQLite FTS index
    └── vectors.gob       # Embedding vectors
```

**Key principle:** JSONL files are the source of truth. The `.bipartite/` cache is ephemeral and rebuilt via `bip rebuild`.

The [nexus-template](https://github.com/matsen/nexus-template) provides a ready-to-use starting point.

## The bip CLI

`bip` is a standalone Go binary. With `nexus_path` configured in `~/.config/bip/config.yml`, commands work from any directory:

```bash
bip search "phylogenetics"    # Search papers
bip s2 add DOI:10.1038/...    # Add paper from Semantic Scholar
bip digest --channel dasm     # Generate GitHub activity digest
bip scout                     # Check remote server availability
```

**For humans:** Add `--human` to any command for readable output.

**For agents:** Default JSON output is designed for programmatic consumption.

### Where bip lives

```
~/go/bin/bip                  # Installed binary (via go install)
~/re/bipartite/               # Source repository
```

## Claude Code Skills

Skills are prompt templates that invoke `bip` commands with appropriate context. They live in `~/.claude/skills/` and are invoked with slash commands:

```
~/.claude/skills/
├── bip.lit/                  # /bip.lit - Search, add papers, library guidance
├── bip.checkin/              # /bip.checkin - GitHub activity check
├── bip.digest/               # /bip.digest - Activity digests
├── bip.spawn/                # /bip.spawn - Spawn tmux sessions
├── bip.board/                # /bip.board - Project boards
├── bip.tree/                 # /bip.tree - Beads hierarchy
├── bip.scout/                # /bip.scout - Server availability
└── bip.narrative/            # /bip.narrative - Prose digests
```

Skills are symlinked from the bipartite repo:

```bash
ln -s ~/re/bipartite/skills/* ~/.claude/skills/
```

### Skill vs CLI

| Task | CLI | Skill |
|------|-----|-------|
| Paper search | `bip search "topic"` | `/bip` (local-first policy) |
| Add a paper | `bip s2 add DOI:...` | `/bip` (guides S2 vs ASTA) |
| Daily check-in | `bip checkin` | `/bip.checkin` (adds context) |
| Generate digest | `bip digest --channel x` | `/bip.digest` (interactive) |
| Spawn issue session | `bip spawn org/repo#123` | `/bip.spawn` (sets up context) |

Skills add value when context matters — they read project files, understand your workflow, and guide the interaction.

## Data Flow Examples

### Searching for papers

```
User: bip search "variational inference"
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│    bip CLI      │────▶│  .bipartite/    │
│                 │     │  cache/refs.db  │
│  Runs FTS query │◀────│  (SQLite FTS5)  │
└─────────────────┘     └─────────────────┘
         │
         ▼
    JSON results
```

### Adding a paper from Semantic Scholar

```
User: bip s2 add DOI:10.1038/s41586-021-03819-2
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│    bip CLI      │────▶│ Semantic Scholar│
│                 │     │      API        │
│  Fetches metadata◀────│                 │
└─────────────────┘     └─────────────────┘
         │
         ▼
┌─────────────────┐
│  refs.jsonl     │  ◀── Appends new entry
└─────────────────┘
         │
         ▼
┌─────────────────┐
│  bip rebuild    │  ◀── Updates SQLite index
└─────────────────┘
```

### Generating a GitHub digest

```
User: /bip.digest dasm
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│  Claude Code    │────▶│  sources.yml   │
│  (skill loads   │     │  (repo list)    │
│   context)      │     └─────────────────┘
└─────────────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│    bip digest   │────▶│   GitHub API    │
│                 │     │   (via gh CLI)  │
│  Fetches PRs,   │◀────│                 │
│  issues, reviews│     └─────────────────┘
└─────────────────┘
         │
         ▼
    Formatted digest (or posted to Slack)
```

## Next Steps

- [Getting Started](getting-started.md) — Create your nexus
- [Reference Management](reference-management.md) — Search and import papers
- [Workflow Coordination](workflow-coordination.md) — GitHub activity tracking
