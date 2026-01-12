# Research: Core Reference Manager

**Feature**: 001-core-reference-manager
**Date**: 2026-01-12

## Technology Decisions

### 1. SQLite Library

**Decision**: Use `modernc.org/sqlite` (pure Go SQLite)

**Rationale**:
- CGO-free: Enables easy cross-compilation for macOS and Linux without a C toolchain
- Well-maintained: Active development, upgraded to SQLite 3.49.0 in February 2025
- Proven adoption: Used by Gogs for 2+ years without known issues
- Standard `database/sql` interface: Familiar API for Go developers

**Alternatives Considered**:
- `mattn/go-sqlite3`: Requires CGO, prevents cross-compilation
- `zombiezen/go-sqlite`: Lower-level API, uses modernc under the hood anyway
- DuckDB: Overkill for simple queries, larger binary

**Performance Note**: Pure Go SQLite is ~6x slower than CGO version in benchmarks, but this is acceptable for our scale (10k papers, <500ms query target). The cross-compilation benefit outweighs the performance cost.

**Concurrency**: Use `DB.SetMaxOpenConns(1)` to prevent "database is locked" errors, as SQLite doesn't support concurrent writes.

Sources:
- [modernc.org/sqlite on pkg.go.dev](https://pkg.go.dev/modernc.org/sqlite)
- [go-sqlite-bench benchmarks](https://github.com/cvilsmeier/go-sqlite-bench)

---

### 2. CLI Framework

**Decision**: Use Go standard library `flag` package with a simple command dispatcher

**Rationale**:
- Minimal dependencies: Aligns with constitution principle VI (Simplicity)
- Fast startup: No framework initialization overhead
- Sufficient for our needs: We have ~10 commands with simple flag structures
- Easy to understand: No magic, explicit command routing

**Alternatives Considered**:
- `spf13/cobra`: Powerful but "bloated", adds complexity we don't need
- `urfave/cli`: Reports of flag handling issues, still heavier than needed
- `alecthomas/kong`: Struct-based approach is nice but adds a dependency
- `peterbourgon/ff`: Good middle ground but still external dependency

**Implementation Pattern**:
```go
// Simple dispatcher pattern
func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    cmd := os.Args[1]
    args := os.Args[2:]

    switch cmd {
    case "init":
        runInit(args)
    case "import":
        runImport(args)
    // ...
    }
}
```

Each command defines its own flag set, keeping concerns separated.

Sources:
- [Go CLI comparison](https://github.com/gschauer/go-cli-comparison)
- [ffcli - simpler building block](https://mfridman.medium.com/a-simpler-building-block-for-go-clis-4c3f7f0f6e03)

---

### 3. BibTeX Generation

**Decision**: Hand-roll BibTeX output (no external library)

**Rationale**:
- We only need **generation**, not parsing (import is from Paperpile JSON)
- BibTeX format is simple and well-documented
- Avoids adding dependency for ~50 lines of code
- Full control over output formatting

**BibTeX Entry Format**:
```bibtex
@article{Ahn2026-rs,
  author = {Ahn, Jenny J and Yu, Timothy C and Dadonaite, Bernadeta},
  title = {Influenza hemagglutinin subtypes...},
  journal = {bioRxiv},
  year = {2026},
  doi = {10.64898/2026.01.05.697808},
}
```

**Entry Type Mapping**:
- bioRxiv/medRxiv/preprints → `@article` with note field
- Journal articles → `@article`
- Conference papers → `@inproceedings`
- Books → `@book`

**Field Escaping**: Escape special LaTeX characters: `& % $ # _ { } ~ ^`

Sources:
- [nickng/bibtex](https://github.com/nickng/bibtex) - referenced for format understanding
- [jschaf/bibtex](https://github.com/jschaf/bibtex) - referenced for parsing patterns

---

### 4. PDF Opening

**Decision**: Platform-specific commands with configurable reader

**macOS**:
- Default: `open <path>` (uses system default)
- Skim: `open -a Skim <path>` (supports page targeting via URL scheme)

**Linux**:
- Default: `xdg-open <path>` (uses system default)
- Zathura: `zathura <path>` (supports `--page=N`)
- Evince: `evince <path>` (supports `--page-index=N`)
- Okular: `okular <path>` (supports `--page N`)

**Configuration Storage**:
```json
{
  "pdf_root": "/Users/name/Google Drive/Paperpile",
  "pdf_reader": "system"  // or "skim", "zathura", etc.
}
```

**Implementation**: Use `os/exec` to run the appropriate command based on platform (`runtime.GOOS`) and configuration.

---

### 5. JSONL Format Design

**Decision**: One JSON object per line, append-only for imports

**Format**:
```jsonl
{"id":"Ahn2026-rs","doi":"10.64898/2026.01.05.697808","title":"...","authors":[...],...}
{"id":"Smith2025-ab","doi":"10.1000/xyz","title":"...","authors":[...],...}
```

**Git Merge Strategy**:
- New imports append to file
- Re-imports with same DOI: Remove old line, append updated entry
- Merge conflicts: Each line is independent, conflicts are rare
- Worst case: Both sides added same paper → deduplicate on rebuild

**Why Not One File Per Paper**:
- Thousands of small files slow git operations
- Single file easier to backup, grep, edit
- JSONL is a well-understood format

---

### 6. Paperpile JSON Import Mapping

**Paperpile Field** → **Bipartite Field**:

| Paperpile | Bipartite | Notes |
|-----------|-----------|-------|
| `_id` | `source.id` | Track original ID for re-import |
| `citekey` | `id` | Internal stable identifier |
| `doi` | `doi` | Primary deduplication key |
| `title` | `title` | Direct mapping |
| `author[].first` | `authors[].first` | Direct mapping |
| `author[].last` | `authors[].last` | Direct mapping |
| `author[].orcid` | `authors[].orcid` | Optional |
| `abstract` | `abstract` | Direct mapping |
| `journal` | `venue` | Normalized name |
| `published.year/month/day` | `published.year/month/day` | Structured date |
| `attachments[article_pdf=1].filename` | `pdf_path` | Main PDF |
| `attachments[article_pdf=0].filename` | `supplement_paths[]` | Supplements |

**Edge Cases**:
- Missing DOI: Store with citekey as ID, flag for groom command
- Missing abstract: Store as empty string, don't omit field
- Partial dates: Store available components (year always present)

---

### 7. ID Generation and Collision Handling

**Decision**: Use Paperpile citekey as ID, suffix on collision

**Algorithm**:
1. Use `citekey` from Paperpile (e.g., `Ahn2026-rs`)
2. If ID exists and DOIs match → update existing entry
3. If ID exists and DOIs differ → append suffix (`Ahn2026-rs-2`)
4. If ID exists and neither has DOI → compare titles, merge or suffix

**Why Citekey**:
- Human-readable and memorable
- Stable across re-imports (same paper → same citekey)
- Works in BibTeX export directly

---

### 8. Test Fixture Strategy

**Decision**: Extract sanitized fixtures from real Paperpile export

**Fixture Types**:
1. `minimal.json` - Single paper with all required fields
2. `with_supplements.json` - Paper with main PDF + supplements
3. `no_doi.json` - Paper without DOI (common for preprints)
4. `partial_date.json` - Paper with only year, no month/day
5. `no_abstract.json` - Paper with missing abstract
6. `collision.json` - Two papers with same citekey pattern

**Sanitization**:
- Replace personal folder paths with generic paths
- Remove `owner` and sensitive metadata
- Keep structure identical to real export

**Storage**: `testdata/` in repo (committed), sourced from `_ignore/` (not committed)

---

## Summary

| Decision | Choice | Key Reason |
|----------|--------|------------|
| SQLite | modernc.org/sqlite | CGO-free cross-compilation |
| CLI | Standard library `flag` | Minimal dependencies |
| BibTeX | Hand-rolled | Simple format, no parsing needed |
| PDF open | Platform commands | Native integration |
| Data format | JSONL | Git-mergeable, human-readable |
