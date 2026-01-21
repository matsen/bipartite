# Research: Domain-Aware Conflict Resolution

**Feature Branch**: `009-refs-conflict-resolve`
**Date**: 2026-01-21

## Research Questions

### RQ-1: Git Conflict Marker Parsing

**Question**: What is the exact format of git conflict markers, and what edge cases must be handled?

**Findings**:

Git conflict markers follow this standard format:
```
<<<<<<< HEAD
... ours content ...
=======
... theirs content ...
>>>>>>> branch-name
```

**Key Details**:
- `<<<<<<<` marks start of "ours" section, followed by space and ref name (typically "HEAD")
- `=======` marks boundary between ours and theirs (exactly 7 equals signs)
- `>>>>>>>` marks end, followed by space and branch/ref name
- Each marker is on its own line
- Multiple conflict regions can exist in one file
- Content between markers may be empty (one side deleted content)

**Edge Cases to Handle**:
1. **Nested conflicts**: Not possible with standard git - markers don't nest
2. **Multiple conflicts**: Each region is independent, parse sequentially
3. **Partial lines**: Conflict markers always span complete lines
4. **Empty sides**: One side may have no content (deletion)
5. **Whitespace**: Lines may have trailing whitespace; match exactly 7 characters for markers

**Decision**: Use simple line-by-line state machine parser. Match on prefix `<<<<<<<`, exact `=======`, and prefix `>>>>>>>`.

**Rationale**: State machine is clear, testable, and handles all edge cases. No regex needed.

---

### RQ-2: Paper Field Completeness Scoring

**Question**: How do we determine which paper record is "more complete" for auto-resolution?

**Findings**:

**Reference struct fields** (from `internal/reference/reference.go`):
- `ID` - always present (required)
- `DOI` - primary identifier (may be empty)
- `Title` - string
- `Authors` - slice of Author structs
- `Abstract` - string
- `Venue` - string
- `Published` - struct with Year, Month, Day
- `PDFPath` - string
- `SupplementPaths` - slice of strings
- `Source` - struct with Type, ID
- `Supersedes` - string

**Completeness Scoring Strategy**:

Option A: Count non-empty fields
- Simple count: +1 for each non-empty string, +1 for non-empty slice, +1 for non-zero year
- Problem: Treats all fields equally (Title as important as SupplementPaths)

Option B: Weighted field scoring
- Higher weights for important fields: Title (3), Abstract (3), Authors (2), DOI (2), Venue (1), Published (1)
- Problem: Arbitrary weights, over-engineering

Option C: Priority field comparison (SELECTED)
- Compare fields in priority order: Abstract > Authors > Venue > Published > DOI
- First record with a field the other lacks wins
- If all priority fields equal, fall back to total non-empty count
- Simple, deterministic, meaningful for academic papers

**Decision**: Option C - Priority field comparison.

**Rationale**:
- Abstract and author completeness are most valuable for research use
- Avoids arbitrary scoring weights
- Deterministic and predictable behavior
- Ties are rare after priority comparison

---

### RQ-3: Author List Comparison and Merging

**Question**: How should author lists be compared and merged when both sides have authors?

**Findings**:

**Spec requirement** (from spec.md edge cases):
> "How are author lists merged when both sides have authors but lists differ? -> Prefer the longer/more complete author list; if same length with different content, treat as true conflict"

**Author struct**:
```go
type Author struct {
    First string `json:"first"`
    Last  string `json:"last"`
    ORCID string `json:"orcid,omitempty"`
}
```

**Comparison Strategy**:

1. **Length comparison first**: If one list is longer, prefer it (more complete)
2. **Same length, different content**: This is a "true conflict" requiring interactive resolution
3. **Completeness within authors**: An author with ORCID is more complete than without

**Equality Check**:
- Two Author structs are equal if First and Last match (case-insensitive comparison for robustness)
- ORCID can differ (one may have it, other not)

**Decision**:
- Longer author list wins automatically
- Same length with different names = true conflict
- Same length with one having additional ORCIDs = merge ORCIDs into combined list

**Rationale**: Aligns with spec; longer list usually means S2 enrichment or manual completion.

---

### RQ-4: Complementary Metadata Merging

**Question**: How exactly do we merge complementary metadata (ours has field X, theirs has field Y)?

**Findings**:

**Spec requirement** (FR-006):
> "System MUST merge complementary metadata when one version has a field the other lacks"

**Merge Strategy**:

For each field:
- If only one side has non-empty value → use that value
- If both sides have same value → use that value
- If both sides have different non-empty values → true conflict (needs interactive)

**Field-specific merging**:

| Field | Merge Rule |
|-------|------------|
| ID | Must match (same paper) |
| DOI | Must match or one empty (identifier) |
| Title | Prefer non-empty; different non-empty = conflict |
| Authors | See RQ-3 |
| Abstract | Prefer non-empty; different non-empty = conflict |
| Venue | Prefer non-empty; different non-empty = conflict |
| Published | Merge: take most specific (day > month > year) |
| PDFPath | Prefer non-empty; different non-empty = conflict |
| SupplementPaths | Union of both lists |
| Source | Preserve original (first import) |
| Supersedes | Prefer non-empty; different non-empty = conflict |

**Decision**: Field-by-field merge with union for slice fields, most-specific for dates.

**Rationale**: Maximizes information preserved; slice union never loses data.

---

### RQ-5: Interactive Prompt Design

**Question**: What is the best UX for interactive conflict resolution prompts?

**Findings**:

**Spec requirement** (FR-011a):
> "Interactive prompts MUST use numbered options (e.g., '[1] ours [2] theirs') with user typing the number to select"

**Design Decisions**:

1. **Display format**: Show field name, both values, numbered options
```
Conflict in field 'abstract':
  [1] ours:   "We present a model for..." (142 chars)
  [2] theirs: "This paper introduces a..." (189 chars)
Enter choice [1/2]:
```

2. **Input handling**:
- Accept "1" or "2" only
- Invalid input → repeat prompt (don't exit)
- Ctrl+C → exit with error

3. **Progress indication**:
```
Resolving conflict 2 of 5...
```

**Decision**: Simple numbered prompts with truncated preview of values.

**Rationale**: Matches spec requirement; minimal and clear.

---

### RQ-6: Output JSON Structure

**Question**: What should the resolve command's JSON output look like?

**Findings**:

Following existing patterns in `cmd/bip/types.go`:

```go
type ResolveResult struct {
    Resolved     int               `json:"resolved"`      // Papers successfully resolved
    OursPapers   int               `json:"ours_papers"`   // Papers from ours only
    TheirsPapers int               `json:"theirs_papers"` // Papers from theirs only
    Merged       int               `json:"merged"`        // Papers with merged metadata
    Unresolved   []UnresolvedInfo  `json:"unresolved,omitempty"` // True conflicts (if any)
    Operations   []ResolveOp       `json:"operations,omitempty"` // Detailed operations
}

type UnresolvedInfo struct {
    PaperID string   `json:"paper_id"`
    DOI     string   `json:"doi,omitempty"`
    Fields  []string `json:"fields"` // Fields with true conflicts
}

type ResolveOp struct {
    PaperID   string `json:"paper_id"`
    DOI       string `json:"doi,omitempty"`
    Action    string `json:"action"` // "keep_ours", "keep_theirs", "merge", "add_ours", "add_theirs"
    Reason    string `json:"reason"` // Human-readable explanation
}
```

**Decision**: Above structure provides summary counts plus detailed operations.

**Rationale**: Consistent with existing command outputs (see DiffResult, ExportResult patterns).

---

## Summary

| Topic | Decision | Key Rationale |
|-------|----------|---------------|
| Conflict parsing | Line-by-line state machine | Simple, testable, handles all cases |
| Completeness scoring | Priority field comparison | Meaningful for papers, deterministic |
| Author merging | Longer list wins; same length different = conflict | Aligns with spec |
| Metadata merging | Field-by-field; union for slices; most-specific for dates | Maximizes information |
| Interactive UX | Numbered options [1/2] | Matches spec requirement |
| Output structure | ResolveResult with counts + operations | Consistent with existing patterns |
