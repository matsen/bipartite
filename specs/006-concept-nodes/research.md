# Research: Concept Nodes

**Feature Branch**: `006-concept-nodes`
**Date**: 2026-01-21

## Research Questions

### 1. Concept ID Validation Pattern

**Question**: What regex/validation pattern should concept IDs use?

**Decision**: `^[a-z0-9][a-z0-9_-]*$` (lowercase alphanumeric, hyphens, underscores; must start with alphanumeric)

**Rationale**:
- Matches FR-007 requirement: "lowercase alphanumeric characters, hyphens, and underscores"
- Leading alphanumeric prevents edge cases like `--help` being interpreted as a flag
- Examples: `somatic-hypermutation`, `variational_inference`, `bcr123`

**Alternatives Considered**:
- Allow uppercase → rejected (creates case-sensitivity confusion, e.g., `SHM` vs `shm`)
- Allow leading hyphen → rejected (conflicts with CLI flag parsing)
- Allow dots → rejected (could conflict with file extensions, version numbers)

---

### 2. Edge Type Discrimination (Paper-Paper vs Paper-Concept)

**Question**: How do we distinguish paper-paper edges from paper-concept edges in the shared edges.jsonl?

**Decision**: Runtime lookup — when validating an edge, check if `target_id` exists in refs vs concepts. No field-level discrimination needed.

**Rationale**:
- Paper IDs (e.g., `Matsen2025-oj`) and concept IDs (e.g., `somatic-hypermutation`) have distinct patterns
- Assumption in spec: "Paper IDs from Paperpile (e.g., Halpern1998-yc) and concept IDs (e.g., somatic-hypermutation) are unlikely to collide"
- Adding a `target_type` field would require schema migration and add complexity
- Existing edge validation already loads ref IDs; extend to also load concept IDs

**Alternatives Considered**:
- Add `target_type: "paper"|"concept"` field → rejected (YAGNI, schema change, migration burden)
- Prefix concept IDs with `concept:` → rejected (ugly, requires all edge commands to strip prefix)
- Separate paper-concept-edges.jsonl file → rejected (fragments edges, complicates queries like "all edges for paper X")

---

### 3. Concept-Aware Edge Validation

**Question**: How should `bip edge add` know to validate concept targets vs paper targets?

**Decision**: Load both ref IDs and concept IDs into validation map. Check target against both sets. Accept if found in either.

**Rationale**:
- Follows existing pattern in `cmd/bip/edge.go` lines 89-99 (load all ref IDs, validate source/target)
- Extend `loadRefIDs()` to `loadAllValidIDs()` that returns both ref and concept ID sets
- Paper-paper relationship types (cites, extends) only make sense with paper targets
- Paper-concept relationship types (introduces, applies) only make sense with concept targets
- But: enforcing relationship-type/target-type constraints is a "warning" per FR-012, not an error

**Alternatives Considered**:
- Strict type checking (error if wrong target type for relationship) → rejected (FR-012 says warn, not error)
- No validation (let users create invalid edges) → rejected (violates fail-fast principle)

---

### 4. Concept FTS Schema

**Question**: Should concepts have full-text search? What fields?

**Decision**: Yes, create `concepts_fts` table with `id`, `name`, `aliases_text`, `description` columns.

**Rationale**:
- Users will want to find concepts by partial name match ("somatic" → "somatic-hypermutation")
- Aliases are important for discovery ("SHM" → "somatic-hypermutation")
- Description provides additional searchable context
- Matches existing pattern for `refs_fts`

**Alternatives Considered**:
- No FTS for concepts → rejected (limits discoverability as concept vocabulary grows)
- Only search by exact ID → rejected (users won't always remember exact IDs)

---

### 5. Merge Operation Semantics

**Question**: What happens to duplicate edges when merging concepts?

**Decision**: Update edge target_id, then deduplicate. If paper P has edges to both old concept C1 and surviving concept C2 with same relationship type, keep only one (preserve the one with earlier `created_at`).

**Rationale**:
- After merge, all edges to C1 become edges to C2
- If paper already had edge to C2 with same relationship type, we'd have a duplicate
- Deduplication by keeping earlier timestamp preserves original intent
- This matches upsert behavior in existing `UpsertEdgeInSlice`

**Alternatives Considered**:
- Error on duplicate detection → rejected (blocks valid merge operations)
- Keep all edges (allow duplicates) → rejected (violates edge uniqueness constraint)
- Merge summaries → rejected (over-engineering, summaries might conflict)

---

### 6. Delete with Linked Papers

**Question**: How should `bip concept delete` handle concepts with existing edges?

**Decision**: Require `--force` flag if edges exist. Without flag, error with count of affected edges.

**Rationale**:
- Matches edge case in spec: "What happens when user tries to delete a concept that has linked papers? System should warn and require confirmation or a force flag."
- Fail-fast by default (protect user from accidental data loss)
- `--force` for intentional deletion (common CLI pattern)

**Alternatives Considered**:
- Cascade delete edges automatically → rejected (too destructive, no recovery)
- Interactive confirmation prompt → rejected (breaks agent workflows, violates agent-first design)
- Soft delete (mark as deleted) → rejected (YAGNI, complicates queries)

---

### 7. SQLite Rebuild Transaction Handling

**Question**: Should concept rebuild use transactions?

**Decision**: No explicit transaction needed — follow existing pattern in `RebuildFromJSONL` and `RebuildEdgesFromJSONL`.

**Rationale**:
- Existing rebuild doesn't use explicit transactions (relies on SQLite autocommit)
- Rebuild is idempotent (drops and recreates table contents)
- Failed rebuild leaves empty tables, user re-runs
- Keeping consistency with existing pattern avoids special-casing concepts

**Alternatives Considered**:
- Add transaction wrapper → rejected (would need to retrofit all rebuild operations for consistency)

---

## Technology Best Practices Applied

### Go CLI Patterns (existing in codebase)

1. **Output helpers**: Use `outputJSON()` / `outputHuman()` pattern from existing commands
2. **Exit codes**: Use defined constants (`ExitError`, `ExitDataError`, `ExitValidationError`)
3. **Flag conventions**: `--human` for human-readable output (default JSON)
4. **Error messages**: Include "expected X, got Y" format per fail-fast principle

### JSONL Storage Patterns

1. **Read all with validation**: Fail-fast on first invalid line
2. **Append single**: For add operations
3. **Write all**: For update/delete operations (read-modify-write)
4. **Find by ID**: Linear scan acceptable for hundreds of concepts

### SQLite Index Patterns

1. **Schema creation in `ensureSchema()`**: Idempotent `CREATE TABLE IF NOT EXISTS`
2. **Rebuild from JSONL**: Clear table, bulk insert
3. **Indexes on query columns**: `source_id`, `target_id`, `relationship_type`

---

## Summary of Decisions

| Question | Decision | Key Reason |
|----------|----------|------------|
| Concept ID format | `^[a-z0-9][a-z0-9_-]*$` | Matches spec, avoids flag conflicts |
| Edge type discrimination | Runtime lookup | No schema change, IDs naturally distinct |
| Edge validation | Load refs + concepts | Extend existing pattern |
| Concept FTS | Yes, 4 columns | Matches refs pattern, enables discovery |
| Merge duplicates | Dedupe, keep earlier | Preserves original edge intent |
| Delete with edges | Require `--force` | Fail-fast, protect data |
| Rebuild transactions | None (follow existing) | Consistency with codebase |

All NEEDS CLARIFICATION items resolved. Ready for Phase 1.
