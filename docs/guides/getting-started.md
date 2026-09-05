# Getting Started

Bipartite operates on a **[nexus](architecture.md)** — a directory that serves as the central hub for your paper library, server configurations, and workflow coordination. See [How It Works](architecture.md) for the full picture of how bipartite's pieces fit together.

## Installation

### Prerequisites

- Go 1.24 or later ([install Go](https://go.dev/doc/install))
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (for agents/skills)
- Git

### Full Installation (Recommended)

This method installs the `bip` CLI, Claude Code agents, and skills in one command:

```bash
git clone https://github.com/matsen/bipartite
cd bipartite
make install
```

This will:

1. Install `bip` via `go install` (to `$GOBIN` if set, otherwise `$HOME/go/bin`)
2. Symlink agents to `~/.claude/agents/`
3. Symlink skills to `~/.claude/skills/`
4. Symlink the statusline script to `~/.claude/statusline/`

**Ensure the Go bin directory is in your PATH.** If you haven't configured `$GOBIN`, add to `~/.bashrc` or `~/.zshrc`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Then reload your shell (`source ~/.bashrc` or `source ~/.zshrc`).

**Enable the Claude Code status line.** Add the following to `~/.claude/settings.json` to display model name, working directory, git branch, and context window usage in the Claude Code status bar:

```json
"statusLine": {
  "type": "command",
  "command": "node ~/.claude/statusline/ctx_monitor.js"
}
```

**Verify installation:**

```bash
bip --help
ls ~/.claude/agents/
ls ~/.claude/skills/
```

### CLI-Only Installation

If you only want the `bip` command-line tool without Claude Code integration:

```bash
go install github.com/matsen/bipartite/cmd/bip@latest
```

This installs to `$GOBIN` if set, otherwise `$HOME/go/bin`.

**Add to PATH** (if you haven't configured `$GOBIN`):

```bash
# Add to ~/.bashrc or ~/.zshrc:
export PATH="$HOME/go/bin:$PATH"

# Then reload:
source ~/.bashrc  # or source ~/.zshrc
```

**Verify installation:**

```bash
bip --help
```

### Manual Agent/Skill Installation

If you want to manually symlink agents and skills (instead of using `make install`):

```bash
# Clone repository
git clone https://github.com/matsen/bipartite
cd bipartite

# Create directories
mkdir -p ~/.claude/agents
mkdir -p ~/.claude/skills

# Symlink agents
for f in agents/*.md; do
  ln -sf "$(pwd)/$f" ~/.claude/agents/$(basename "$f")
done

# Symlink skills (note: skills are in ./skills/, not ./.claude/skills/)
for d in skills/*/; do
  ln -sf "$(pwd)/$d" ~/.claude/skills/$(basename "$d")
done
```

**Verify:**

```bash
ls -l ~/.claude/agents/
ls -l ~/.claude/skills/
```

## Creating a Nexus

### Option 1: Use the Template (Recommended)

The [nexus-template](https://github.com/matsen/nexus-template) repository provides a ready-to-use starting point with example papers and configuration templates.

1. Go to [github.com/matsen/nexus-template](https://github.com/matsen/nexus-template)
2. Click **Use this template** → **Create a new repository**
3. Clone your new repository:

```bash
git clone https://github.com/yourusername/my-nexus
cd my-nexus
```

4. Build the search index:

```bash
bip rebuild
```

5. Verify it works:

```bash
bip search "phylogenetics"
```

### Option 2: Create Manually

For an empty nexus without using the template:

```bash
mkdir my-nexus && cd my-nexus
git init
mkdir -p .bipartite
touch .bipartite/refs.jsonl
echo ".bipartite/cache/" >> .gitignore
bip rebuild
```

This creates the minimal structure needed to start adding papers.

## What's in a Nexus?

```
my-nexus/
├── .bipartite/               # Tracked in git, except cache/
│   ├── refs.jsonl            # Paper references (source of truth)
│   ├── projects.jsonl        # Projects
│   ├── repos.jsonl           # Tracked repositories
│   ├── config.yml            # (optional) Local paths, API keys
│   └── cache/                # (gitignored, ephemeral)
│       ├── refs.db           # SQLite FTS index
│       └── semantic.gob      # Embedding vectors
│
├── servers.yml               # (optional) Remote servers for bip scout
├── sources.yml               # (optional) GitHub repos for activity tracking
│
├── context/                  # (optional) Project context files
└── narrative/                # (optional) Generated digest output
```

### Core Files (Source of Truth)

- **refs.jsonl** — Your paper library. Each line is a JSON object with paper metadata.
- **projects.jsonl** — Project definitions (see [Projects, Repos, and Stores](projects-and-stores.md)).
- **repos.jsonl** — Tracked repositories.

These files are the source of truth. They're plain text, git-friendly, and designed for collaboration.

### Cache Directory (Ephemeral)

The `.bipartite/cache/` subdirectory contains:

- **refs.db** — SQLite full-text search index
- **semantic.gob** — Embedding vectors for semantic search

Only `cache/` is gitignored — the JSONL files above it are tracked. The cache is rebuilt from those source files via `bip rebuild`, so you can delete it and rebuild anytime.

### Configuration Files (Optional)

- **.bipartite/config.yml** — Local paths (PDF root), Ollama settings
- **servers.yml** — Remote servers for `bip scout`
- **sources.yml** — GitHub repos and boards for `bip checkin`, `bip digest`

## Adding Your First Paper

```bash
bip s2 add DOI:10.1038/s41586-021-03819-2
```

This fetches metadata from Semantic Scholar and appends to `.bipartite/refs.jsonl`.

## Searching

```bash
bip search "machine learning"        # Keyword search
bip search "author:Felsenstein"      # Author search
bip search "title:phylogenetics"     # Title search
```

## Next Steps

- [Reference Management](reference-management.md) — Search, import, and organize papers
- [Projects, Repos, and Stores](projects-and-stores.md) — Track projects, repos, and custom record stores
- [Workflow Coordination](workflow-coordination.md) — GitHub activity tracking and Slack integration
- [Server Scout](server-scout.md) — Monitor remote compute resources
