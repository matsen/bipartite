# Research: Projects and Repos

**Feature**: 011-repo-nodes | **Date**: 2026-01-24

## Research Topics

### 1. GitHub API Integration

**Decision**: Use GitHub REST API v3 with `net/http` (no external client library)

**Rationale**:
- Single endpoint needed: `GET /repos/{owner}/{repo}`
- No OAuth required for public repos (simplicity principle)
- Standard library `net/http` sufficient; avoid dependency bloat
- Support `github_token` env var for private repos and higher rate limits

**Alternatives Considered**:
- `google/go-github` library: Rejected (overkill for one endpoint, adds dependency)
- GraphQL API: Rejected (REST simpler for this use case)

**Implementation Notes**:
- 10-second timeout per request
- Exponential backoff on rate limit (429) responses
- User-Agent header required by GitHub API
- Parse `X-RateLimit-Remaining` header for proactive rate limit awareness

### 2. Type-Prefixed ID Strategy

**Decision**: Use type prefixes (`project:`, `repo:`) for new node types in edges; papers and concepts remain unprefixed for backward compatibility

**Rationale**:
- Spec requires type disambiguation in edges (`source_id`, `target_id`)
- Existing edges reference papers/concepts by unprefixed ID (e.g., `10.1038/...`, `variational-inference`)
- Adding prefixes to existing data would require migration; user prefers manual updates
- New project/repo IDs include prefix from creation time

**Edge ID Format**:
```json
// Existing (unchanged)
{"source_id": "10.1038/...", "target_id": "variational-inference", ...}

// New concept↔project edges (prefixed)
{"source_id": "concept:variational-inference", "target_id": "project:dasm2", ...}
```

**Alternatives Considered**:
- Full migration to prefixed IDs: Rejected (user will handle manually in nexus)
- Separate source_type/target_type fields: Considered, but prefixed IDs cleaner
- Infer type from ID format: Rejected (DOIs contain colons, ambiguous)

**Migration Path**:
- User manually adds `concept:` prefix to existing concept edge targets in nexus
- New edges always use prefixed IDs for projects/concepts
- Papers remain unprefixed (DOI format is self-identifying)

### 3. Edge Validation Strategy

**Decision**: Validate edge endpoints by type prefix, reject invalid combinations

**Rationale**:
- Core graph constraint: no direct paper↔project edges
- Repos have no edges (metadata only)
- Validation at edge creation time (fail-fast)

**Valid Edge Combinations**:
| Source | Target | Allowed | Example |
|--------|--------|---------|---------|
| paper (unprefixed) | paper (unprefixed) | ✅ | `cites` |
| paper (unprefixed) | concept: | ✅ | `introduces` |
| concept: | paper (unprefixed) | ✅ | `critiques` (inverse) |
| concept: | project: | ✅ | `implemented-in` |
| project: | concept: | ✅ | `introduces` |
| paper | project: | ❌ | Must go through concept |
| project: | paper | ❌ | Must go through concept |
| * | repo: | ❌ | Repos have no edges |
| repo: | * | ❌ | Repos have no edges |

**Implementation**:
- Parse type prefix from ID (split on first `:`)
- If no prefix and valid DOI pattern → paper
- If no prefix and matches concept ID pattern → concept (lookup required)
- Explicit `project:` or `repo:` prefix → project/repo

### 4. Transitive Query Implementation

**Decision**: Two-step SQL query with in-memory aggregation

**Rationale**:
- "Papers relevant to project X" = papers linked to concepts linked to project X
- SQLite supports this as JOIN, but clearer as two queries
- Step 1: Find concepts linked to project
- Step 2: Find papers linked to those concepts

**Query Pattern**:
```sql
-- Step 1: Concepts linked to project
SELECT DISTINCT source_id FROM edges
WHERE target_id = 'project:dasm2'
  AND source_id LIKE 'concept:%';

-- Step 2: Papers linked to those concepts (for each concept)
SELECT source_id, relationship_type, summary FROM edges
WHERE target_id = 'concept:X';
```

**Alternatives Considered**:
- Single JOIN query: Possible but less readable
- Graph traversal library: Overkill for two-hop queries

### 5. GitHub URL Parsing

**Decision**: Support both full URLs and `org/repo` shorthand

**Patterns to Accept**:
- `https://github.com/matsen/bipartite` → `matsen`, `bipartite`
- `https://github.com/matsen/bipartite.git` → `matsen`, `bipartite`
- `github.com/matsen/bipartite` → `matsen`, `bipartite`
- `matsen/bipartite` → `matsen`, `bipartite`

**Validation**:
- Must have exactly one `/` in the org/repo part
- Org and repo names follow GitHub naming rules (alphanumeric, hyphen, underscore)

### 6. One-Project-Per-Repo Constraint

**Decision**: Enforce unique GitHub URLs across all repos

**Rationale**:
- A GitHub repository belongs to exactly one project
- Prevents confusion about which project "owns" a repo
- Enforced at repo creation time

**Implementation**:
- Load all repos, check for existing `github_url`
- If found, error with: "repo X already belongs to project Y"

## Open Questions Resolved

| Question | Resolution |
|----------|------------|
| ID collision handling | Fail with error; no auto-suffix |
| sources.json integration | Deferred; use separate JSONL files |
| Migration script | Not needed; user handles manually |
| Bulk import | Out of scope for this PR |
