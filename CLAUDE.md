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
- **Go 1.25.5** with spf13/cobra (CLI) and modernc.org/sqlite (storage)
- **Ollama** for local embeddings, pure Go vector storage
- **Semantic Scholar API** for paper metadata (internal/s2 package)
- **Data storage**: JSONL (refs.jsonl, edges.jsonl, concepts.jsonl) + ephemeral SQLite (rebuilt on `bip rebuild`)
- **Vector index**: GOB-serialized (ephemeral, gitignored)

## Project Structure

```text
cmd/           # Go CLI command implementations (bip)
internal/      # Go internal packages (s2, store, index, etc.)
fc_cli/        # Python CLI implementation (flowc)
tests_fc/      # Python tests
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

## flowc (Python CLI)

flowc manages GitHub activity and project boards. It must be run from the nexus directory:

```bash
cd ~/re/nexus
flowc checkin              # Check recent GitHub activity
flowc board list           # View project boards
flowc spawn org/repo#123   # Spawn tmux window for issue review
```

**Claude Code slash commands:** `/flowc.checkin`, `/flowc.spawn`, `/flowc.board`, `/flowc.digest`, `/flowc.tree`

flowc reads configuration from:
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
6. **Run Tests**: Ensure all tests pass:
   - Go: `go test ./...`
   - Python: `pytest tests_fc/`

### Final Static Analysis
7. **Vet and Lint**: Run `go vet ./...` and any configured linters to verify code quality

### Documentation Sync
8. **README Update**: If the feature adds new commands or changes user-facing behavior, update `README.md` to document the changes

<!-- MANUAL ADDITIONS END -->
