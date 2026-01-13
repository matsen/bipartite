# Quickstart: Core Reference Manager Development

**Feature**: 001-core-reference-manager
**Date**: 2026-01-12

## Prerequisites

- Go 1.21+ installed (`go version`)
- Git configured
- Access to `_ignore/paperpile-export-jan-12.json` for test fixtures

## Project Setup

### 1. Initialize Go Module

```bash
cd /Users/matsen/re/bipartite
go mod init github.com/matsen/bipartite
```

### 2. Create Directory Structure

```bash
mkdir -p cmd/bp
mkdir -p internal/{config,importer,reference,storage,query,export,pdf}
mkdir -p testdata
```

### 3. Add Dependencies

```bash
go get github.com/spf13/cobra
go get modernc.org/sqlite
```

## Development Workflow

### Test-Driven Development

Follow the agentic TDD cycle:

1. **Write failing test** with real fixture data
2. **Implement** minimal code to pass
3. **Run tests**: `go test ./...`
4. **Iterate** until green
5. **Refactor** if needed (tests still green)

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/importer/...

# With verbose output
go test -v ./internal/importer/...

# With race detection
go test -race ./...
```

### Building

```bash
# Development build
go build -o bp ./cmd/bp

# Release build (smaller binary)
go build -ldflags="-s -w" -o bp ./cmd/bp

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o bp-linux ./cmd/bp
```

### Running

```bash
# From project root
./bp init
./bp config pdf-root ~/Google\ Drive/Paperpile
./bp import --format paperpile _ignore/paperpile-export-jan-12.json
./bp search "phylogenetics"
```

## Test Fixtures

### Creating Fixtures from Real Data

Extract sanitized entries from `_ignore/paperpile-export-jan-12.json`:

```bash
# Extract first 5 entries for testing
jq '.[0:5]' _ignore/paperpile-export-jan-12.json > testdata/paperpile_sample.json
```

### Required Fixture Types

Create these fixtures in `testdata/`:

| File | Description |
|------|-------------|
| `paperpile_minimal.json` | Single paper with all required fields |
| `paperpile_with_supplements.json` | Paper with main PDF + supplement attachments |
| `paperpile_no_doi.json` | Paper without DOI |
| `paperpile_partial_date.json` | Paper with only year, no month/day |
| `paperpile_no_abstract.json` | Paper with missing abstract |
| `paperpile_collision.json` | Two papers that would have same citekey |

### Sanitization Checklist

When extracting fixtures:
- [ ] Remove `owner` field
- [ ] Replace personal paths with generic paths
- [ ] Remove `gdrive_id` and other cloud-specific fields
- [ ] Keep structure identical to real export

## Implementation Order

### Phase 1: Foundation (Week 1)

1. **Reference types** (`internal/reference/`)
   - `Reference` struct with JSON tags
   - `Author` struct
   - `PublicationDate` struct
   - Validation methods

2. **JSONL storage** (`internal/storage/jsonl.go`)
   - Read all references from file
   - Append reference to file
   - Replace reference in file (for updates)

3. **Configuration** (`internal/config/`)
   - Load/save config.json
   - Validate paths exist

### Phase 2: Import (Week 2)

4. **Paperpile importer** (`internal/importer/paperpile.go`)
   - Parse Paperpile JSON format
   - Map to Reference type
   - Handle attachments (main vs supplement)

5. **Deduplication** (`internal/storage/`)
   - DOI-based matching
   - ID collision handling with suffix

### Phase 3: Query Layer (Week 3)

6. **SQLite schema** (`internal/storage/sqlite.go`)
   - Create tables from schema
   - Rebuild from JSONL
   - Basic CRUD operations

7. **Search** (`internal/query/`)
   - FTS5 setup
   - Keyword search
   - Field-specific search (author:, title:)

### Phase 4: CLI & Export (Week 4)

8. **CLI framework** (`cmd/bp/`)
   - Command dispatcher
   - Flag parsing per command
   - JSON/human output formatting

9. **BibTeX export** (`internal/export/bibtex.go`)
   - Entry type mapping
   - Field formatting
   - LaTeX character escaping

10. **PDF opener** (`internal/pdf/`)
    - Path resolution
    - Platform-specific open commands

## Code Quality

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Formatting

```bash
# Format all code
gofmt -w .

# Or use goimports (also organizes imports)
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

## Common Patterns

### Error Handling

```go
// Always return explicit errors, never panic
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config file %s: %w", path, err)
    }
    // ...
}
```

### JSON Output

```go
// All commands use this pattern
func outputJSON(v interface{}) error {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(v)
}
```

### Testing with Fixtures

```go
func TestPaperpileImport(t *testing.T) {
    // Load real fixture
    data, err := os.ReadFile("testdata/paperpile_sample.json")
    if err != nil {
        t.Fatalf("loading fixture: %v", err)
    }

    refs, err := importer.ParsePaperpile(data)
    if err != nil {
        t.Fatalf("parsing: %v", err)
    }

    // Assert on real data
    if len(refs) != 5 {
        t.Errorf("expected 5 references, got %d", len(refs))
    }
}
```

## Troubleshooting

### "database is locked"

SQLite doesn't support concurrent writes. Use `SetMaxOpenConns(1)`:

```go
db.SetMaxOpenConns(1)
```

### Cross-compilation fails

Ensure you're using `modernc.org/sqlite` (pure Go), not `mattn/go-sqlite3` (CGO):

```bash
# Should not require CGO
CGO_ENABLED=0 go build ./cmd/bp
```

### FTS5 not working

FTS5 is built into modernc.org/sqlite. If queries return no results:

1. Check triggers are creating FTS entries
2. Verify `authors_text` is populated correctly
3. Test with simple query first: `SELECT * FROM references_fts WHERE references_fts MATCH 'test'`
