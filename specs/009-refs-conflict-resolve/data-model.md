# Data Model: Domain-Aware Conflict Resolution

**Feature Branch**: `009-refs-conflict-resolve`
**Date**: 2026-01-21

## Entities

### Existing Entities (No Changes)

#### Reference
Location: `internal/reference/reference.go`

The existing Reference struct is read-only for this feature. We parse refs.jsonl (potentially conflicted) and write back resolved refs.jsonl.

```go
type Reference struct {
    ID              string          `json:"id"`
    DOI             string          `json:"doi"`
    Title           string          `json:"title"`
    Authors         []Author        `json:"authors"`
    Abstract        string          `json:"abstract"`
    Venue           string          `json:"venue"`
    Published       PublicationDate `json:"published"`
    PDFPath         string          `json:"pdf_path"`
    SupplementPaths []string        `json:"supplement_paths,omitempty"`
    Source          ImportSource    `json:"source"`
    Supersedes      string          `json:"supersedes,omitempty"`
}
```

---

### New Entities

#### ConflictRegion
Location: `internal/conflict/types.go`

Represents a single conflict region from the file, bounded by git markers.

```go
// ConflictRegion represents a single git conflict region in a JSONL file.
type ConflictRegion struct {
    // Line numbers in original file (1-indexed)
    StartLine int // Line of <<<<<<< marker
    EndLine   int // Line of >>>>>>> marker

    // Parsed content from each side
    OursRefs   []reference.Reference // Papers from "ours" (HEAD) side
    TheirsRefs []reference.Reference // Papers from "theirs" side

    // Raw content (for error messages)
    OursRaw   string // Raw JSONL content from ours
    TheirsRaw string // Raw JSONL content from theirs
}
```

**Validation Rules**:
- StartLine < EndLine
- OursRefs and TheirsRefs may be empty (deletion case)
- Raw content preserved for debugging parse errors

---

#### PaperMatch
Location: `internal/conflict/types.go`

Represents a matched pair of papers (same paper on both sides).

```go
// PaperMatch represents a paper that appears on both sides of a conflict.
type PaperMatch struct {
    Ours      reference.Reference // Version from ours side
    Theirs    reference.Reference // Version from theirs side
    MatchedBy string              // "doi" or "id" - how they were matched
}
```

**Invariants**:
- If MatchedBy == "doi", then Ours.DOI == Theirs.DOI and both non-empty
- If MatchedBy == "id", then Ours.ID == Theirs.ID

---

#### FieldConflict
Location: `internal/conflict/types.go`

Represents a true conflict where a field has different non-empty values.

```go
// FieldConflict represents a true conflict for a specific field.
type FieldConflict struct {
    FieldName  string // e.g., "abstract", "title", "venue"
    OursValue  string // Value from ours (may be truncated for display)
    TheirsValue string // Value from theirs (may be truncated for display)
}
```

---

#### ResolutionPlan
Location: `internal/conflict/types.go`

The plan for resolving a single paper match (before execution).

```go
// ResolutionPlan describes how a matched paper pair will be resolved.
type ResolutionPlan struct {
    PaperID string // ID of the paper
    DOI     string // DOI if available

    // Resolution action
    Action ResolutionAction // See enum below
    Reason string           // Human-readable explanation

    // For merge actions, which fields come from where
    FieldSources map[string]string // field name -> "ours" | "theirs" | "merged"

    // True conflicts requiring interactive resolution
    Conflicts []FieldConflict // Empty if auto-resolvable
}

// ResolutionAction indicates the type of resolution applied.
type ResolutionAction string

const (
    ActionKeepOurs   ResolutionAction = "keep_ours"   // Ours is more complete
    ActionKeepTheirs ResolutionAction = "keep_theirs" // Theirs is more complete
    ActionMerge      ResolutionAction = "merge"       // Complementary metadata merged
    ActionAddOurs    ResolutionAction = "add_ours"    // Paper only in ours
    ActionAddTheirs  ResolutionAction = "add_theirs"  // Paper only in theirs
    ActionConflict   ResolutionAction = "conflict"    // True conflict, needs interactive
)
```

---

#### ParseError
Location: `internal/conflict/types.go`

Structured error for parse failures.

```go
// ParseError represents an error while parsing conflict markers or JSONL.
type ParseError struct {
    Line    int    // Line number where error occurred (1-indexed)
    Message string // Description of the error
    Context string // Surrounding content for debugging
}

func (e ParseError) Error() string {
    return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}
```

---

### Output Types

#### ResolveResult
Location: `cmd/bip/types.go`

JSON output structure for the resolve command.

```go
// ResolveResult is the JSON output of the bip resolve command.
type ResolveResult struct {
    // Summary counts
    Resolved     int `json:"resolved"`      // Total papers in resolved output
    OursPapers   int `json:"ours_papers"`   // Papers unique to ours side
    TheirsPapers int `json:"theirs_papers"` // Papers unique to theirs side
    Merged       int `json:"merged"`        // Papers with metadata merged

    // Unresolved conflicts (only present if --interactive not used)
    Unresolved []UnresolvedInfo `json:"unresolved,omitempty"`

    // Detailed operations (for debugging/audit)
    Operations []ResolveOp `json:"operations,omitempty"`
}

// UnresolvedInfo describes a paper with true conflicts.
type UnresolvedInfo struct {
    PaperID string   `json:"paper_id"`
    DOI     string   `json:"doi,omitempty"`
    Fields  []string `json:"fields"` // Field names with conflicts
}

// ResolveOp describes a single resolution operation.
type ResolveOp struct {
    PaperID string `json:"paper_id"`
    DOI     string `json:"doi,omitempty"`
    Action  string `json:"action"` // keep_ours, keep_theirs, merge, add_ours, add_theirs
    Reason  string `json:"reason"` // Human-readable explanation
}
```

---

## State Transitions

### Conflict Parser States

```
                    ┌──────────────────────┐
                    │      NORMAL          │
                    │ (outside conflict)   │
                    └──────────┬───────────┘
                               │
                    <<<<<<< marker detected
                               │
                               ▼
                    ┌──────────────────────┐
                    │    IN_OURS           │
                    │ (collecting ours)    │
                    └──────────┬───────────┘
                               │
                    ======= marker detected
                               │
                               ▼
                    ┌──────────────────────┐
                    │    IN_THEIRS         │
                    │ (collecting theirs)  │
                    └──────────┬───────────┘
                               │
                    >>>>>>> marker detected
                               │
                               ▼
                    ┌──────────────────────┐
                    │      NORMAL          │
                    │ (emit ConflictRegion)│
                    └──────────────────────┘
```

**Error Conditions**:
- ======= in NORMAL state → error "unexpected separator outside conflict"
- >>>>>>> in NORMAL state → error "unexpected end marker outside conflict"
- <<<<<<< in IN_OURS or IN_THEIRS → error "nested conflict markers"
- EOF in IN_OURS or IN_THEIRS → error "unterminated conflict region"

---

## Relationships

```
ConflictRegion
    │
    ├── 1:N ── OursRefs (Reference[])
    │
    └── 1:N ── TheirsRefs (Reference[])
                    │
                    ▼
              PaperMatch (when DOI/ID matches)
                    │
                    ├── Has Conflicts? ─Yes──▶ FieldConflict[]
                    │
                    └── No ──▶ ResolutionPlan (auto-resolved)
```

---

## File Formats

### Input: Conflicted refs.jsonl

```jsonl
{"id":"paper1","doi":"10.1234/a","title":"Paper One","authors":[]}
<<<<<<< HEAD
{"id":"paper2","doi":"10.1234/b","title":"Paper Two","abstract":"From ours"}
=======
{"id":"paper2","doi":"10.1234/b","title":"Paper Two","abstract":"From theirs","venue":"Nature"}
>>>>>>> feature-branch
{"id":"paper3","doi":"10.1234/c","title":"Paper Three","authors":[]}
```

### Output: Resolved refs.jsonl

```jsonl
{"id":"paper1","doi":"10.1234/a","title":"Paper One","authors":[]}
{"id":"paper2","doi":"10.1234/b","title":"Paper Two","abstract":"From theirs","venue":"Nature"}
{"id":"paper3","doi":"10.1234/c","title":"Paper Three","authors":[]}
```

Note: The resolved version keeps paper2 from "theirs" because it has more fields (abstract + venue vs just abstract).
