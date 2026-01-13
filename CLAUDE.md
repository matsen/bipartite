# bipartite Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-12

## Active Technologies
- Go 1.21+ (continuing Phase I) + Ollama (local embeddings), pure Go vector storage (002-rag-index)
- GOB-serialized vector index (ephemeral, gitignored) (002-rag-index)
- Go 1.25.5 (continuing Phase I/II) + spf13/cobra (CLI), modernc.org/sqlite (storage) - no new dependencies (003-knowledge-graph)
- JSONL (edges.jsonl) + ephemeral SQLite (edge index rebuilt on `bp rebuild`) (003-knowledge-graph)

- Go 1.21+ (latest stable) (001-core-reference-manager)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.21+ (latest stable)

## Code Style

Go 1.21+ (latest stable): Follow standard conventions

## Recent Changes
- 003-knowledge-graph: Added Go 1.25.5 (continuing Phase I/II) + spf13/cobra (CLI), modernc.org/sqlite (storage) - no new dependencies
- 002-rag-index: Added Go 1.21+ (continuing Phase I) + Ollama (local embeddings), pure Go vector storage

- 001-core-reference-manager: Added Go 1.21+ (latest stable)

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
