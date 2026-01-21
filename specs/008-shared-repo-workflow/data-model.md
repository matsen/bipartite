# Data Model: Shared Repository Workflow Commands

## Overview

This feature primarily uses existing entities (Reference, BibTeX Entry) with new response types for command outputs. No schema changes to persistent storage (refs.jsonl, SQLite) are required.

---

## Existing Entities (No Changes)

### Reference
Source: `internal/reference/reference.go`

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

Used by: All commands in this feature

### BibTeX Entry (Output Format)
Source: `internal/export/bibtex.go`

Existing `ToBibTeX()` function produces valid BibTeX. No structural changes needed.

---

## New Response Types

### OpenMultipleResult
For `bip open` with multiple papers.

```go
// OpenMultipleResult is the JSON response for opening multiple papers.
type OpenMultipleResult struct {
    Opened []OpenedPaper `json:"opened"`
    Errors []OpenError   `json:"errors,omitempty"`
}

type OpenedPaper struct {
    ID   string `json:"id"`
    Path string `json:"path"`
}

type OpenError struct {
    ID    string `json:"id"`
    Error string `json:"error"`
}
```

**Validation Rules**:
- At least one paper must be opened successfully, or exit with error
- Errors array populated for papers with missing PDFs (FR-004)

### DiffResult
For `bip diff` showing uncommitted changes.

```go
// DiffResult is the JSON response for bip diff.
type DiffResult struct {
    Added   []DiffPaper `json:"added"`
    Removed []DiffPaper `json:"removed"`
}

type DiffPaper struct {
    ID      string `json:"id"`
    Title   string `json:"title"`
    Authors string `json:"authors"` // Formatted as "Last1, Last2, ..."
    Year    int    `json:"year"`
}
```

**Validation Rules**:
- Empty arrays (not null) when no changes
- Papers sorted by ID for deterministic output

### NewPapersResult
For `bip new --since` and `bip new --days`.

```go
// NewPapersResult is the JSON response for bip new.
type NewPapersResult struct {
    Papers     []NewPaper `json:"papers"`
    SinceRef   string     `json:"since_ref,omitempty"`   // Commit SHA or "N days ago"
    TotalCount int        `json:"total_count"`
}

type NewPaper struct {
    ID        string `json:"id"`
    Title     string `json:"title"`
    Authors   string `json:"authors"`
    Year      int    `json:"year"`
    CommitSHA string `json:"commit_sha,omitempty"` // When paper was added
}
```

**Validation Rules**:
- Empty papers array (not null) when no new papers
- CommitSHA populated when available from git history

### ExportResult
Extended response for `bip export --bibtex` with `--append`.

```go
// ExportResult is the JSON response for bip export with --append.
type ExportResult struct {
    Exported   int      `json:"exported"`    // Number of entries written
    Skipped    int      `json:"skipped"`     // Number of duplicates skipped
    SkippedIDs []string `json:"skipped_ids,omitempty"` // IDs that were duplicates
    OutputPath string   `json:"output_path,omitempty"` // When --append used
}
```

**Validation Rules**:
- Without --append: BibTeX written to stdout (no JSON response)
- With --append: JSON response with stats

---

## Internal Types

### BibTeXIndex
For deduplication during append operations.

```go
// BibTeXIndex indexes existing BibTeX entries for deduplication.
type BibTeXIndex struct {
    // Keys maps citation keys to true for existence check
    Keys map[string]bool
    // DOIs maps DOI values to citation keys
    DOIs map[string]string
}

// HasEntry returns true if the entry already exists (by DOI or key).
func (idx *BibTeXIndex) HasEntry(key, doi string) bool

// ParseFile builds an index from an existing .bib file.
func ParseBibTeXFile(path string) (*BibTeXIndex, error)
```

### GitDiff
Internal representation of refs.jsonl changes.

```go
// GitDiff represents changes to refs.jsonl between two git states.
type GitDiff struct {
    Added   []reference.Reference
    Removed []reference.Reference
}

// DiffWorkingTree compares working tree to HEAD.
func DiffWorkingTree(repoRoot string) (*GitDiff, error)

// DiffSince compares current state to a specific commit.
func DiffSince(repoRoot, commitRef string) (*GitDiff, error)
```

---

## State Transitions

### Reference Lifecycle (Existing)
```
[not in library] --import/s2 add--> [in refs.jsonl] --rebuild--> [in SQLite]
```

No changes to this lifecycle.

### BibTeX Append Operation
```
[Paper ID] --lookup--> [Reference] --ToBibTeX--> [BibTeX Entry]
                                                      |
                                                      v
[Existing .bib] --ParseBibTeXFile--> [BibTeXIndex] --HasEntry?--> [skip/append]
```

---

## Entity Relationships

```
┌─────────────────┐
│   Reference     │
│   (refs.jsonl)  │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    v         v
┌───────┐ ┌──────────┐
│ open  │ │ export   │
│ diff  │ │ --bibtex │
│ new   │ └────┬─────┘
└───────┘      │
               v
         ┌───────────┐
         │ BibTeXIndex│
         │ (.bib file)│
         └───────────┘
```

---

## Notes

1. **No persistent storage changes**: All new types are response/internal only
2. **Git is the source of truth for history**: Paper addition times derived from git log, not stored in refs.jsonl
3. **BibTeX parsing is read-only**: We parse existing .bib files but don't modify their structure, only append
