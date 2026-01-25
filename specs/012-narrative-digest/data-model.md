# Data Model: Narrative Digest

## Overview

This feature uses markdown files for configuration and output. No database changes required.

## Entities

### DigestItem (existing)

Represents a single GitHub PR or issue. Defined in `internal/flow/types.go`.

```go
type DigestItem struct {
    Ref          string   // "org/repo#123"
    Number       int      // Issue/PR number
    Title        string   // Title text
    Author       string   // GitHub username
    IsPR         bool     // PR vs Issue
    State        string   // "open", "closed", "merged"
    Merged       bool     // For PRs
    HTMLURL      string   // GitHub URL
    CreatedAt    string   // RFC3339 timestamp
    UpdatedAt    string   // RFC3339 timestamp
    Contributors []string // All participants
    Body         string   // NEW: Full body text (for --verbose)
    Summary      string   // NEW: LLM-generated summary (for --verbose)
}
```

### Channel Config (markdown)

Location: `nexus/narrative/{channel}.md`

```markdown
# {channel} Narrative Configuration

Inherits from [preferences.md](preferences.md).

## Themes

1. **Theme Name** - Description (repos: repo1, repo2)
2. **Theme Name** - Description

## Project-Specific Preferences

- Preference 1
- Preference 2

## Repo Context

- **repo-name**: What this repo covers
```

**Parsed structure** (conceptual):

```go
type ChannelConfig struct {
    Channel     string
    Themes      []Theme
    Preferences []string
    RepoContext map[string]string
}

type Theme struct {
    Name        string
    Description string
    Order       int
}
```

### Shared Preferences (markdown)

Location: `nexus/narrative/preferences.md`

```markdown
# Narrative Digest Preferences

## Attribution
- Rule 1

## Format
- Rule 1
- Rule 2

## Content
- Rule 1
```

### Narrative Output (markdown)

Location: `nexus/narrative/{channel}/{YYYY-MM-DD}.md`

```markdown
# {channel} Digest: {date range}

## Theme 1

**Subheading:** (if applicable)
- Bullet items or prose

## Theme 2

Content...

## Looking Ahead

- Open issue or PR 1
- Open issue or PR 2
```

## Relationships

```
preferences.md
    ↓ inherits
{channel}.md ──→ themes, repo context
    ↓ uses
bip digest output ──→ raw activity items
    ↓ generates
{channel}/{date}.md ──→ narrative output
```

## State Transitions

### DigestItem.State

- `open`: Active PR/issue
- `closed`: Closed without merge (issues) or rejected (PRs)
- `merged`: PR merged (sets `Merged: true`)

### Output Prefixes

Based on state:
- `merged` → no prefix (default merged section)
- `open` + PR → "In progress:"
- `open` + Issue → "Open:"

## Validation Rules

### Channel Config

- Must have `## Themes` section with numbered list
- Each theme must have `**Name**` format
- `## Repo Context` optional but recommended

### Output

- All theme sections must have content or be omitted
- "Looking Ahead" only includes open items
- GitHub links must use markdown format `[#N](url)`

## No Database Changes

This feature does not modify:
- refs.jsonl
- edges.jsonl
- concepts.jsonl
- SQLite schema
