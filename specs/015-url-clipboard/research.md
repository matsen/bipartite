# Research: URL Output and Clipboard Support

**Feature**: 015-url-clipboard
**Date**: 2026-01-27

## Research Tasks

### 1. Clipboard Implementation Strategy

**Decision**: Shell-out to platform-native tools (pbcopy/xclip/xsel)

**Rationale**:
- Constitution VI (Simplicity) favors minimal dependencies
- `golang.design/x/clipboard` requires CGO or X11 libraries on Linux
- Shell-out approach requires zero additional dependencies
- `pbcopy` is built-in on macOS
- `xclip`/`xsel` are standard on Linux desktops

**Alternatives Considered**:
| Approach | Pros | Cons |
|----------|------|------|
| `golang.design/x/clipboard` | Pure Go API, cross-platform | Requires X11 dev packages on Linux, CGO on some platforms |
| Shell-out to pbcopy/xclip | Zero dependencies, simple | Requires external tools on Linux |
| No clipboard support | Simplest | Defeats core feature value |

**Implementation Pattern**:
```go
func copyToClipboard(text string) error {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("pbcopy")
    case "linux":
        // Try xclip first, fall back to xsel
        if _, err := exec.LookPath("xclip"); err == nil {
            cmd = exec.Command("xclip", "-selection", "clipboard")
        } else if _, err := exec.LookPath("xsel"); err == nil {
            cmd = exec.Command("xsel", "--clipboard", "--input")
        } else {
            return ErrClipboardUnavailable
        }
    default:
        return ErrClipboardUnavailable
    }
    cmd.Stdin = strings.NewReader(text)
    return cmd.Run()
}
```

### 2. External ID Field Storage

**Decision**: Flat fields directly on Reference struct

**Rationale**:
- Spec explicitly states: "Flat fields directly on reference object (pmid, pmcid, arxiv_id, s2_id)"
- Simpler than nested ExternalIDs struct
- Each field optional (omitempty)
- Clean JSON representation in JSONL

**Implementation**:
```go
type Reference struct {
    // ... existing fields ...

    // External identifiers (populated from S2 API)
    PMID    string `json:"pmid,omitempty"`
    PMCID   string `json:"pmcid,omitempty"`
    ArXivID string `json:"arxiv_id,omitempty"`
    S2ID    string `json:"s2_id,omitempty"`
}
```

### 3. URL Format Patterns

**Decision**: Use standard URL patterns for each provider

| Provider | URL Pattern | Example |
|----------|-------------|---------|
| DOI | `https://doi.org/{doi}` | `https://doi.org/10.1234/example` |
| PubMed | `https://pubmed.ncbi.nlm.nih.gov/{pmid}/` | `https://pubmed.ncbi.nlm.nih.gov/12345678/` |
| PMC | `https://www.ncbi.nlm.nih.gov/pmc/articles/{pmcid}/` | `https://www.ncbi.nlm.nih.gov/pmc/articles/PMC1234567/` |
| arXiv | `https://arxiv.org/abs/{arxiv_id}` | `https://arxiv.org/abs/2106.15928` |
| S2 | `https://www.semanticscholar.org/paper/{s2_id}` | `https://www.semanticscholar.org/paper/649def34...` |

### 4. SQLite Schema Changes

**Decision**: Add columns, require rebuild

**Rationale**:
- Constitution II: SQLite is ephemeral, rebuilt from JSONL
- CLAUDE.md notes: `CREATE ... IF NOT EXISTS` doesn't update existing schemas
- Standard workflow: delete .bipartite/cache/refs.db, run `bip rebuild`

**Schema Addition**:
```sql
-- Add to refs table
pmid TEXT,
pmcid TEXT,
arxiv_id TEXT,
s2_id TEXT
```

### 5. Flag Mutual Exclusivity

**Decision**: Error when multiple format flags specified

**Rationale**:
- Spec clarification: "Error with clear message asking user to specify only one format flag"
- Consistent with existing patterns (see `open.go` mutual exclusivity handling)

**Implementation**:
```go
flagCount := 0
if pubmedFlag { flagCount++ }
if pmcFlag { flagCount++ }
if arxivFlag { flagCount++ }
if s2Flag { flagCount++ }
if flagCount > 1 {
    exitWithError(ExitError, "specify only one URL format flag (--pubmed, --pmc, --arxiv, or --s2)")
}
```

### 6. Output Composability

**Decision**: URL to stdout, confirmation to stderr

**Rationale**:
- Spec clarification: "URL to stdout and confirmation to stderr (composable for piping)"
- Allows `bip url Smith2024-ab --copy | other-command` patterns
- JSON mode outputs structured data to stdout only

**Human Output Pattern**:
```
https://doi.org/10.1234/example    # stdout
Copied to clipboard                # stderr (only with --copy)
```

**JSON Output Pattern**:
```json
{
  "url": "https://doi.org/10.1234/example",
  "format": "doi",
  "copied": true
}
```

## Sources

- [golang.design/x/clipboard](https://github.com/golang-design/clipboard) - Evaluated but rejected for simplicity
- Existing bipartite codebase patterns (open.go, get.go)
- Spec clarifications from session 2026-01-27
