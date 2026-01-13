# bipartite Development Guidelines

## Active Technologies
- **Go 1.25.5** with spf13/cobra (CLI) and modernc.org/sqlite (storage)
- **Ollama** for local embeddings, pure Go vector storage
- **Data storage**: JSONL (refs.jsonl, edges.jsonl) + ephemeral SQLite (rebuilt on `bp rebuild`)
- **Vector index**: GOB-serialized (ephemeral, gitignored)

## Project Structure

```text
cmd/           # CLI command implementations
internal/      # Internal packages (store, index, etc.)
specs/         # Feature specifications
testdata/      # Test fixtures
tests/         # Integration tests
```

## Commands

```bash
# Build and run
go build -o bp . && ./bp --help

# Common operations
./bp import --format paperpile export.json   # Import references
./bp rebuild                                  # Rebuild SQLite index after import
./bp search <query>                          # Search by keyword
./bp semantic <query>                        # Semantic similarity search
./bp get <id>                                # Get reference by ID
./bp list                                    # List all references
./bp export                                  # Export to BibTeX

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

<!-- MANUAL ADDITIONS END -->
