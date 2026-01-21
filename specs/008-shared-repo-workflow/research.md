# Research: Shared Repository Workflow Commands

## Research Tasks

Based on Technical Context unknowns and dependency best practices:

1. Git integration patterns for JSONL diff detection
2. BibTeX parsing for append deduplication
3. Concurrent PDF opening across platforms
4. Commit-based timestamp filtering

---

## 1. Git Integration for JSONL Diff Detection

### Decision: Use `git diff` and `git show` via os/exec

### Rationale
The codebase already uses `os/exec` for external tools (see `internal/pdf/opener.go`). Go has no standard git library, and CGO-based libgit2 bindings would violate constitution principle VI (Simplicity) by adding heavyweight dependencies.

### Approach
- **For `bip diff`**: Compare working tree vs HEAD using `git diff --name-status`
- **For `bip new --since <commit>`**: Use `git log --oneline --since-commit` combined with `git show <commit>:.bipartite/refs.jsonl` to get historical state
- **For `--recent N`**: Parse git log to find commits that touched refs.jsonl, extract paper IDs from diffs

### Key Commands
```bash
# Check if commit exists
git rev-parse --verify <commit>^{commit}

# Get refs.jsonl at specific commit
git show <commit>:.bipartite/refs.jsonl

# Get commits touching refs.jsonl since a commit
git log --oneline <commit>..HEAD -- .bipartite/refs.jsonl

# Get unified diff of refs.jsonl
git diff HEAD -- .bipartite/refs.jsonl
```

### Alternatives Considered
- **go-git (pure Go)**: Large dependency, overkill for simple diff operations
- **libgit2/git2go**: CGO dependency, complicates builds
- **Direct JSONL comparison**: Would miss uncommitted changes for `bip diff`

---

## 2. BibTeX Parsing for Append Deduplication

### Decision: Implement minimal regex-based parser for citation key and DOI extraction

### Rationale
Full BibTeX parsing is complex (nested braces, string concatenation, etc.). For deduplication, we only need:
1. Extract citation keys (`@article{key123,`)
2. Extract DOI field values (`doi = {10.1234/example},`)

### Approach
- Parse existing .bib file line-by-line with regex
- Build a set of existing citation keys and DOIs
- Skip entries where DOI matches (primary) or key matches (fallback)

### Key Patterns
```go
// Match entry start: @type{key,
entryStartRegex := regexp.MustCompile(`@\w+\{([^,]+),`)

// Match DOI field: doi = {value} or doi = "value"
doiFieldRegex := regexp.MustCompile(`(?i)doi\s*=\s*[\{"]([^\}"]+)[\}"]`)
```

### Alternatives Considered
- **Full BibTeX parser library**: Overkill for deduplication, adds dependency
- **String contains search**: Unreliable for field extraction
- **Parse entire BibTeX AST**: Constitution VI violation (YAGNI)

---

## 3. Concurrent PDF Opening

### Decision: Sequential open calls with small delay, reuse existing pdf.Opener

### Rationale
Opening PDFs in parallel can overwhelm the PDF viewer. The existing `pdf.Opener.Open()` method uses `cmd.Start()` which doesn't block. Sequential calls with the existing implementation will open multiple PDFs.

### Approach
- Loop through paper IDs, call `opener.Open()` for each
- Collect errors for papers with missing PDFs (FR-004: continue with available)
- No artificial delay needed - `cmd.Start()` returns immediately

### Platform Behavior
- **macOS**: `open` command queues files for the same app
- **Linux**: Each `xdg-open` or reader call spawns independently

### Alternatives Considered
- **Single command with multiple files**: Not portable across readers
- **Parallel goroutines**: Unnecessary complexity, no benefit
- **Batching**: Platform-dependent, unpredictable behavior

---

## 4. Commit-Based Timestamp Filtering

### Decision: Compare JSONL snapshots at different commits

### Rationale
The `--since <commit>` flag needs to identify papers added after a commit. Since refs.jsonl is the source of truth, comparing the file at two points in git history provides accurate results.

### Approach for `bip new --since <commit>`
1. Validate commit exists: `git rev-parse --verify <commit>`
2. Get refs.jsonl at commit: `git show <commit>:.bipartite/refs.jsonl`
3. Parse both historical and current JSONL into maps keyed by ID
4. Papers in current but not historical are "new"

### Approach for `--recent N`
1. Query git log for recent commits touching refs.jsonl
2. For each commit, diff against parent to find added IDs
3. Return up to N most recent additions

### Edge Cases
- Non-existent commit: Exit with error per spec
- Merge commits: Use first-parent history for simplicity
- Rebased history: Works correctly (compares snapshots, not commit parents)

### Alternatives Considered
- **Track add timestamps in JSONL**: Requires schema change, breaks existing data
- **Use commit timestamps as proxy**: Doesn't account for force-push/rebase
- **SQLite triggers**: Would require persistent tracking, violates constitution II

---

## Summary of Decisions

| Topic | Decision | Key Rationale |
|-------|----------|---------------|
| Git integration | os/exec with git CLI | Simplicity, no CGO |
| BibTeX parsing | Regex for key/DOI only | Minimal for deduplication |
| PDF opening | Sequential, existing Opener | cmd.Start() is non-blocking |
| Commit filtering | JSONL snapshot comparison | Accurate, works with rebase |

All decisions align with constitution principles, particularly VI (Simplicity) and II (Git-Versionable Architecture).
