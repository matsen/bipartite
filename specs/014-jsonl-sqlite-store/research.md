# Research: Generic JSONL + SQLite Store Abstraction

**Feature**: 014-jsonl-sqlite-store
**Date**: 2026-01-27

## Overview

Minimal research needed - this feature generalizes an existing pattern already implemented in the codebase. Key decisions are informed by existing code rather than external research.

## Decision 1: Schema Format

**Decision**: JSON schema with custom field options (primary, index, fts, enum)

**Rationale**:
- JSON is human-readable and fits the git-versionable architecture principle
- Custom options (primary, index, fts) are domain-specific and simpler than JSON Schema vocabulary
- Similar approach used by beads and other JSONL-first tools

**Alternatives considered**:
- JSON Schema (too complex for our needs, overkill for simple type validation)
- YAML schema (adds dependency, JSON is already used throughout)
- Go struct tags (not portable, couples schema to Go code)

## Decision 2: SQLite DDL Generation

**Decision**: Generate DDL at runtime from parsed schema

**Rationale**:
- Existing `internal/storage/sqlite.go` uses inline DDL strings
- Runtime generation allows schema changes without code changes
- Full rebuild on sync means no migration complexity

**Alternatives considered**:
- sqlc or gorm (adds complexity, we only need basic CRUD)
- Pre-generated DDL files (requires manual sync with schema)

## Decision 3: FTS5 Implementation

**Decision**: Standalone FTS5 tables (not external content)

**Rationale**:
- Existing `refs_fts` table uses standalone FTS5
- Simpler to manage during full rebuilds
- Slight storage overhead acceptable for typical store sizes

**Alternatives considered**:
- External content FTS5 (more complex triggers, brittle during rebuild)
- trigram index (SQLite default FTS5 is sufficient)

## Decision 4: Sync Hash Storage

**Decision**: Store SHA256 hash in `_meta` table within each store's SQLite database

**Rationale**:
- Self-contained (no external state file)
- Simple comparison: compute hash, compare, skip or rebuild
- Existing pattern in similar tools

**Alternatives considered**:
- Separate `.hash` file (extra file to track)
- mtime comparison (unreliable across git operations)

## Decision 5: Atomic JSONL Operations

**Decision**: Write to temp file, then rename (atomic on POSIX)

**Rationale**:
- Existing `WriteAll` doesn't do this but should for delete operations
- Prevents corruption if process crashes mid-write
- Standard pattern for durable writes

**Alternatives considered**:
- Write-ahead log (overkill for CLI tool)
- fsync without rename (not atomic on crash)

## Decision 6: Cross-Store Queries

**Decision**: Use SQLite ATTACH to join databases

**Rationale**:
- SQLite supports up to 10 attached databases by default
- Simple implementation: attach, execute, detach
- No need for data copying

**Alternatives considered**:
- Single shared database (violates co-location principle)
- Application-level join (complex, slow)

## Existing Code Patterns to Follow

From `internal/storage/jsonl.go`:
- Buffer size: `MaxJSONLLineCapacity = 1024 * 1024` (1MB per line)
- Error format: `fmt.Errorf("parsing line %d: %w", lineNum, err)`
- Empty file returns empty slice, not error

From `internal/storage/sqlite.go`:
- `sql.Open("sqlite", path)` with modernc.org/sqlite
- `SetMaxOpenConns(1)` for SQLite
- Nullable fields use `sql.NullString`, `sql.NullInt64`
- FTS5 query escaping in `prepareFTSQuery()`

## Open Questions (Resolved)

**Q: Where to store schema files?**
A: `.bipartite/schemas/<name>.json` - co-located with store data, referenced from `stores.json`

**Q: How to handle enum validation?**
A: Validate in `Append()` before writing to JSONL. Return error with field name and allowed values.

**Q: What output formats for query?**
A: `--human` (default table), `--json`, `--csv`, `--jsonl` - matches existing bip output patterns
