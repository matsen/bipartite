# bipartite Development Guidelines

## Task Management with Beads (CRITICAL)

**ALWAYS use `bd` (beads) for task tracking. NEVER use TodoWrite or internal todo lists.**

Beads is a Git-backed task tracker designed for AI agents. It provides persistent memory across sessions and proper dependency tracking.

### Essential Commands
```bash
bd ready                    # Show tasks ready to work on (no blockers)
bd create "Title" -p 0      # Create a priority-0 (highest) task
bd create "Title" -p 1      # Create a priority-1 task
bd show <id>                # View task details
bd update <id> -p 1         # Change priority (use update, not edit)
bd update <id> -s in_progress  # Update status
bd close <id>               # Mark task complete
bd dep add <child> <parent> # Child blocked by parent
bd list                     # List all tasks
```

### Workflow
1. **Start of work**: Run `bd ready` to see actionable tasks
2. **Planning**: Create tasks with `bd create`, set dependencies with `bd dep add`
3. **Working**: Update task status as you progress
4. **Completion**: Mark done with `bd close <id>`

### Why Beads Over TodoWrite
- Persists across sessions (stored in `.beads/`, gitignored for local-only use)
- Tracks dependencies between tasks
- Prevents merge conflicts with hash-based IDs
- Designed specifically for AI agent workflows

## Active Technologies
- Go 1.21+ (matches existing go.mod) + spf13/cobra (CLI), modernc.org/sqlite (pure Go SQLite) (014-jsonl-sqlite-store)
- JSONL (source of truth) + SQLite (ephemeral index) (014-jsonl-sqlite-store)
- Go 1.24.1 (from go.mod) + spf13/cobra (CLI), modernc.org/sqlite (pure Go SQLite) (015-url-clipboard)
- JSONL (source of truth) + ephemeral SQLite (rebuilt via `bip rebuild`) (015-url-clipboard)
- Go 1.24.1 (from go.mod) + spf13/cobra (CLI), golang.org/x/crypto/ssh (native SSH), gopkg.in/yaml.v3 (config parsing) (016-bip-scout)
- N/A — stateless command, no persistence (016-bip-scout)

**Go version**: See `go.mod` for minimum version (no cutting-edge features required)

- **CLI**: spf13/cobra
- **Storage**: modernc.org/sqlite (pure Go, no CGO)
- **Embeddings**: Ollama for local embeddings, pure Go vector storage
- **External APIs**: Semantic Scholar (internal/s2 package)
- **Data model**: JSONL (source of truth) + ephemeral SQLite (rebuilt on `bip rebuild`)
- **Vector index**: GOB-serialized (ephemeral, gitignored)

## Project Structure

```text
cmd/           # Go CLI command implementations (bip)
internal/      # Go internal packages (s2, store, index, flow, etc.)
specs/         # Feature specifications
testdata/      # Test fixtures
tests/         # Go integration tests
```

## Building

```bash
go build -o bip ./cmd/bip && ./bip --help
```

See README.md for full command reference.

## Code Style

Follow standard Go conventions (`go fmt`, `go vet`)

<!-- MANUAL ADDITIONS START -->

## Session Management

- **Continuation prompts**: Save to `_ignore/CONTINUE.md`, never commit
- The `_ignore/` directory is gitignored for local-only files

## Git Workflow

- **Repository owner**: This repo is `matsen/bipartite`, NOT `matsengrp/bipartite`. Use `matsen` when constructing GitHub URLs or API calls.
- **PR merge strategy**: Always use squash and merge (`gh pr merge --squash`)

## SQLite Schema Changes

When modifying SQLite schema (e.g., adding columns to FTS5 tables):

1. **Rebuild the binary**: `go build -o bip ./cmd/bip`
2. **Delete the old database**: `rm .bipartite/cache/refs.db`
3. **Rebuild the index**: `./bip rebuild`

Note: `CREATE ... IF NOT EXISTS` does not update existing table schemas - you must delete the database file for schema changes to take effect.

## Bip Skill

Use `/bip` for unified CLI guidance including paper search, library management, and S2 vs ASTA command selection. The skill is defined in `.claude/skills/bip/` and symlinked to `~/.claude/skills/` for global availability.

## Paper Lookups (nexus)

When looking for papers or adding edges to the knowledge graph:

1. **Search locally first** in nexus using grep on `.bipartite/refs.jsonl`:
   ```bash
   grep -i "author_name\|keyword" ~/re/nexus/.bipartite/refs.jsonl | jq -r '.id + " - " + .title'
   ```

2. **Ask before using ASTA MCP** - Always ask the user before making ASTA API calls. ASTA should only be used for:
   - Papers confirmed not in the local database
   - Discovering new papers via citation/reference graphs
   - Searching for papers by topic when local search yields no results

3. **Add papers via S2** when rate limits allow: `./bip s2 add DOI:...`

The nexus library has ~6000 papers already imported - most relevant immunology/antibody papers are likely already there. **Always search locally first before proposing ASTA queries.**

## GitHub Activity Commands (bip)

bip includes built-in GitHub activity tracking and project board management. These commands must be run from the nexus directory:

```bash
cd ~/re/nexus
bip checkin              # Check recent GitHub activity
bip board list           # View project boards
bip spawn org/repo#123   # Spawn tmux window for issue review
bip spawn --prompt "Explore the clamping question"  # Adhoc session without issue
bip digest --channel foo # Preview Slack digest (safe default)
bip digest --channel foo --post  # Actually post to Slack
bip digest --channel foo --verbose  # Include PR/issue body summaries
bip tree --open          # View beads hierarchy in browser
```

**Claude Code slash commands:** `/bip.checkin`, `/bip.spawn`, `/bip.board`, `/bip.digest`, `/bip.tree`, `/bip.narrative`, `/bip.scout`

### Narrative Digests

Use `/bip.narrative` to generate thematic, prose-style digests:

```bash
/bip.narrative dasm2                 # Generate narrative for channel
/bip.narrative dasm2 --since 2w      # Custom date range
/bip.narrative dasm2 --verbose       # Include body summaries
```

Output is written to `narrative/{channel}/{YYYY-MM-DD}.md`. Requires config files:
- `narrative/preferences.md` - Shared formatting rules
- `narrative/{channel}.md` - Channel themes and repo context

These commands read configuration from:
- `sources.json` - Repository list and board mappings
- `config.json` - Local path configuration
- `context/` - Project context files

## Pre-PR Quality Checklist

Before any pull request, ensure the following workflow is completed:

### Requirement Verification (Do This First!)
1. **Spec Compliance**: Review the feature's `spec.md` and `tasks.md` to verify 100% completion of all specified requirements. If any requirement cannot be met, engage with the user to resolve blockers before proceeding

### Code Quality Foundation
2. **Format Code**: Run `go fmt ./...` to apply consistent formatting
3. **Documentation**: Ensure all exported functions and types have doc comments

### Architecture and Implementation Review
4. **Clean Code Review**: Run `@clean-code-reviewer` agent on all new/modified code for architectural review

### Test Quality Validation
5. **Test Implementation Audit**: Scan all test files for partially implemented tests or placeholder implementations. All tests must provide real validation
6. **Run Tests**: Ensure all tests pass: `go test ./...`

### Final Static Analysis
7. **Vet and Lint**: Run static analysis to verify code quality: `go vet ./...`

### Documentation Sync
8. **Documentation Update**: If the feature adds new commands or changes user-facing behavior:
   - `README.md` — Keep short (overview, installation, environment variables only)
   - `docs/guides/` — Detailed guides with examples, config reference, troubleshooting
   - `.claude/skills/bip/` — Update skill docs for Claude Code integration

<!-- MANUAL ADDITIONS END -->

## Recent Changes
- 016-bip-scout: Added Go 1.24.1 (from go.mod) + spf13/cobra (CLI), golang.org/x/crypto/ssh (native SSH), gopkg.in/yaml.v3 (config parsing)
- 015-url-clipboard: Added Go 1.24.1 (from go.mod) + spf13/cobra (CLI), modernc.org/sqlite (pure Go SQLite)
- 014-jsonl-sqlite-store: Added Go 1.21+ (matches existing go.mod) + spf13/cobra (CLI), modernc.org/sqlite (pure Go SQLite)
