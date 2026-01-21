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

**Decision**: Use `spf13/cobra` for CLI structure

**Rationale**:
- De facto standard for Go CLIs (Kubernetes, Hugo, GitHub CLI)
- Automatic help generation with consistent formatting
- Clean subcommand handling without boilerplate
- Short/long flag support (`-h` / `--human`) built-in
- Shell completion for free
- Agents generate better Cobra code (more training examples)
- Single focused dependency that solves a real problem

**Alternatives Considered**:
- Standard library `flag`: Requires hand-rolling subcommand dispatch, help formatting, short flags
- `urfave/cli`: Reports of flag handling quirks
- `alecthomas/kong`: Struct-based approach is nice but less common

**Implementation Pattern**:
```go
// cmd/bip/main.go
func main() {
    rootCmd := &cobra.Command{
        Use:   "bp",
        Short: "Agent-first academic reference manager",
    }

    rootCmd.AddCommand(
        newInitCmd(),
        newImportCmd(),
        newSearchCmd(),
        // ...
    )

    rootCmd.Execute()
}

// cmd/bip/import.go
func newImportCmd() *cobra.Command {
    var format string
    var dryRun bool

    cmd := &cobra.Command{
        Use:   "import <file>",
        Short: "Import references from external format",
        RunE: func(cmd *cobra.Command, args []string) error {
            // implementation
        },
    }

    cmd.Flags().StringVar(&format, "format", "", "Import format (paperpile)")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be imported")
    cmd.MarkFlagRequired("format")

    return cmd
}
```

**Note**: Avoid `cobra-cli init` code generator - write commands directly for cleaner code.

Sources:
- [Cobra on GitHub](https://github.com/spf13/cobra)
- [Go CLI comparison](https://github.com/gschauer/go-cli-comparison)

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
| CLI | spf13/cobra | De facto standard, better ergonomics |
| BibTeX | Hand-rolled | Simple format, no parsing needed |
| PDF open | Platform commands | Native integration |
| Data format | JSONL | Git-mergeable, human-readable |
