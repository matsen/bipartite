# bipartite Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-12

## Active Technologies

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

- 001-core-reference-manager: Added Go 1.21+ (latest stable)

<!-- MANUAL ADDITIONS START -->

## Session Management

- **Continuation prompts**: Save to `_ignore/CONTINUE.md`, never commit
- The `_ignore/` directory is gitignored for local-only files

## Ralph Loop

- Use `/ralph-loop:ralph-loop` (full qualified name) to start the autonomous task loop
- Example: `/ralph-loop:ralph-loop "Your task prompt here" --max-iterations 30 --completion-promise "DONE"`

<!-- MANUAL ADDITIONS END -->
