# bipartite Development Guidelines

## Active Technologies
- **Go 1.25.5** with spf13/cobra (CLI) and modernc.org/sqlite (storage)
- **Ollama** for local embeddings, pure Go vector storage
- **Semantic Scholar API** for paper metadata (internal/s2 package)
- **Data storage**: JSONL (refs.jsonl, edges.jsonl) + ephemeral SQLite (rebuilt on `bip rebuild`)
- **Vector index**: GOB-serialized (ephemeral, gitignored)
- Go 1.25.5 (matches existing codebase) + spf13/cobra (CLI), golang.org/x/time/rate (rate limiting), joho/godotenv (env loading) (005-asta-mcp-integration)
- N/A (ASTA is read-only external API, no local persistence) (005-asta-mcp-integration)
- Go 1.25.5 + spf13/cobra (CLI), modernc.org/sqlite (storage) (006-concept-nodes)
- JSONL (concepts.jsonl) + ephemeral SQLite (rebuilt on `bip rebuild`) (006-concept-nodes)
- Go 1.25.5 + spf13/cobra (CLI), modernc.org/sqlite (storage), html/template (HTML generation) (007-knowledge-graph-viz)
- SQLite (read from existing refs, concepts, edges tables rebuilt from JSONL) (007-knowledge-graph-viz)
- Go 1.25.5 (matches existing codebase) + spf13/cobra (CLI), modernc.org/sqlite (storage), os/exec (git integration) (008-shared-repo-workflow)
- JSONL (refs.jsonl) + ephemeral SQLite (rebuilt on `bip rebuild`) - no schema changes needed (008-shared-repo-workflow)
- Go 1.25.5 (matches existing codebase) + spf13/cobra (CLI), os/exec (git integration), bufio (user prompts) (009-refs-conflict-resolve)
- JSONL (refs.jsonl) - reads conflicted file, writes resolved version (009-refs-conflict-resolve)

## Project Structure

```text
cmd/           # CLI command implementations
internal/      # Internal packages (s2, store, index, etc.)
specs/         # Feature specifications
testdata/      # Test fixtures
tests/         # Integration tests
```

## Commands

```bash
# Build and run
go build -o bip ./cmd/bip && ./bip --help

# Common operations
./bip import --format paperpile export.json   # Import references
./bip rebuild                                  # Rebuild SQLite index after import
./bip search <query>                          # Search by keyword
./bip semantic <query>                        # Semantic similarity search
./bip get <id>                                # Get reference by ID
./bip list                                    # List all references
./bip export --bibtex                          # Export all to BibTeX
./bip export --bibtex <id>...                  # Export specific papers
./bip export --bibtex --append refs.bib <id>   # Append with deduplication

# Open commands (with git integration)
./bip open <id>                               # Open single paper PDF
./bip open <id> <id> ...                      # Open multiple papers
./bip open --recent 5                         # Open 5 most recently added
./bip open --since HEAD~3                     # Open papers added since commit

# Diff and tracking commands (git integration)
./bip diff                                    # Show uncommitted changes to library
./bip diff --human                            # Human-readable diff output
./bip new --since <commit>                    # Papers added since commit
./bip new --days 7                            # Papers added in last N days

# Semantic Scholar (S2) commands
./bip s2 add DOI:10.1234/example              # Add paper by DOI
./bip s2 lookup DOI:10.1234/example           # Look up paper info
./bip s2 citations <paper-id>                 # Find citing papers
./bip s2 references <paper-id>                # Find referenced papers
./bip s2 gaps                                 # Find literature gaps

# ASTA (Academic Search Tool API) commands - read-only exploration
./bip asta search "phylogenetics"             # Search papers by keyword
./bip asta snippet "variational inference"    # Search text snippets in papers
./bip asta paper DOI:10.1093/sysbio/syy032    # Get paper details
./bip asta citations DOI:10.1093/sysbio/syy032 # Get citing papers
./bip asta references DOI:10.1093/sysbio/syy032 # Get referenced papers
./bip asta author "Frederick Matsen"          # Search for authors
./bip asta author-papers 145666442            # Get papers by author ID

# Visualization commands
./bip viz                                      # Generate HTML graph visualization to stdout
./bip viz --output graph.html                  # Generate to file
./bip viz --layout circle                      # Use circular layout (force, circle, grid)
./bip viz --offline                            # Bundle Cytoscape.js for offline use

# Add --human flag for readable output (default is JSON)
```

## Code Style

Follow standard Go conventions (`go fmt`, `go vet`)

<!-- MANUAL ADDITIONS START -->

## Session Management

- **Continuation prompts**: Save to `_ignore/CONTINUE.md`, never commit
- The `_ignore/` directory is gitignored for local-only files

## Git Workflow

- **PR merge strategy**: Always use squash and merge (`gh pr merge --squash`)

## SQLite Schema Changes

When modifying SQLite schema (e.g., adding columns to FTS5 tables):

1. **Rebuild the binary**: `go build -o bip ./cmd/bip`
2. **Delete the old database**: `rm .bipartite/cache/refs.db`
3. **Rebuild the index**: `./bip rebuild`

Note: `CREATE ... IF NOT EXISTS` does not update existing table schemas - you must delete the database file for schema changes to take effect.

## Bip Skill

Use `/bip` for unified CLI guidance including paper search, library management, and S2 vs ASTA command selection. The skill is defined in `.claude/skills/bip/` and symlinked to `~/.claude/skills/` for global availability.

## Ralph Loop

- Use `/ralph-loop:ralph-loop` (full qualified name) to start the autonomous task loop
- Example: `/ralph-loop:ralph-loop "Your task prompt here" --max-iterations 30 --completion-promise "DONE"`

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
6. **Run Tests**: Ensure all tests pass with `go test ./...`

### Final Static Analysis
7. **Vet and Lint**: Run `go vet ./...` and any configured linters to verify code quality

### Documentation Sync
8. **README Update**: If the feature adds new commands or changes user-facing behavior, update `README.md` to document the changes

<!-- MANUAL ADDITIONS END -->

## Recent Changes
- 009-refs-conflict-resolve: Added Go 1.25.5 (matches existing codebase) + spf13/cobra (CLI), os/exec (git integration), bufio (user prompts)
- 008-shared-repo-workflow: Added Go 1.25.5 (matches existing codebase) + spf13/cobra (CLI), modernc.org/sqlite (storage), os/exec (git integration)
- 007-knowledge-graph-viz: Added Go 1.25.5 + spf13/cobra (CLI), modernc.org/sqlite (storage), html/template (HTML generation)
