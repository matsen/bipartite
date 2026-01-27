# Quickstart: Generic JSONL + SQLite Store

**Feature**: 014-jsonl-sqlite-store
**Date**: 2026-01-27

## Overview

The generic store provides a schema-driven way to create JSONL-backed data stores with SQLite query indexes.

## 1. Define a Schema

Create a schema file defining your data structure:

```json
// .bipartite/schemas/gh_activity.json
{
  "name": "gh_activity",
  "fields": {
    "id": {"type": "string", "primary": true},
    "type": {"type": "string", "index": true, "enum": ["commit", "pr", "issue", "review"]},
    "repo": {"type": "string", "index": true},
    "author": {"type": "string", "index": true},
    "date": {"type": "datetime", "index": true},
    "title": {"type": "string", "fts": true},
    "body": {"type": "string", "fts": true},
    "url": {"type": "string"}
  }
}
```

## 2. Initialize the Store

```bash
bip store init gh_activity --schema .bipartite/schemas/gh_activity.json
```

This creates:
- `.bipartite/gh_activity.jsonl` (empty)
- `.bipartite/gh_activity.db` (with schema)
- Entry in `.bipartite/stores.json`

## 3. Append Records

```bash
# Single record from argument
bip store append gh_activity '{"id":"pr-123","type":"pr","repo":"matsen/bipartite","author":"erick","date":"2026-01-27T10:00:00Z","title":"Add stores","body":"Generic store implementation","url":"https://github.com/matsen/bipartite/pull/123"}'

# From a file
bip store append gh_activity --file new_activity.json

# From stdin (piped JSONL)
cat activity_batch.jsonl | bip store append gh_activity --stdin
```

## 4. Sync to SQLite

After appending records, sync to make them queryable:

```bash
bip store sync gh_activity

# Or sync all stores at once
bip store sync --all
```

## 5. Query Records

```bash
# Basic SQL query
bip store query gh_activity "SELECT * FROM gh_activity WHERE type = 'pr'"

# Full-text search
bip store query gh_activity "SELECT * FROM gh_activity WHERE id IN (SELECT id FROM gh_activity_fts WHERE gh_activity_fts MATCH 'store implementation')"

# Output formats
bip store query gh_activity "SELECT id, title FROM gh_activity" --json
bip store query gh_activity "SELECT id, title FROM gh_activity" --csv
bip store query gh_activity "SELECT id, title FROM gh_activity" --jsonl
```

## 6. Delete Records

```bash
# By ID
bip store delete gh_activity pr-123

# By condition
bip store delete gh_activity --where "date < '2025-01-01'"
```

## 7. Manage Stores

```bash
# List all stores
bip store list

# Get store details
bip store info gh_activity
```

## Example: GitHub Activity Tracking

```bash
# 1. Create schema (above)
# 2. Initialize
bip store init gh_activity --schema .bipartite/schemas/gh_activity.json

# 3. Fetch and append (via script or agent)
gh api repos/matsen/bipartite/pulls --jq '.[] | {id: ("pr-" + (.number|tostring)), type: "pr", repo: "matsen/bipartite", author: .user.login, date: .created_at, title: .title, body: .body, url: .html_url}' | bip store append gh_activity --stdin

# 4. Sync
bip store sync gh_activity

# 5. Query
bip store query gh_activity "SELECT author, COUNT(*) as prs FROM gh_activity WHERE type = 'pr' GROUP BY author ORDER BY prs DESC"
```

## Cross-Store Queries

Join data across multiple stores:

```bash
bip store query --cross "SELECT r.title, g.author FROM refs r JOIN gh_activity g ON r.id = g.id WHERE g.type = 'pr'"
```

## Field Types Reference

| Type | SQLite | Example |
|------|--------|---------|
| string | TEXT | `"hello"` |
| integer | INTEGER | `42` |
| float | REAL | `3.14` |
| boolean | INTEGER | `true` → 1 |
| date | TEXT | `"2026-01-27"` |
| datetime | TEXT | `"2026-01-27T10:30:00Z"` |
| json | TEXT | `{"nested": "data"}` |

## Schema Options Reference

| Option | Description |
|--------|-------------|
| `primary` | Primary key (exactly one required) |
| `index` | Create SQLite index for fast lookups |
| `fts` | Include in FTS5 full-text search |
| `enum` | Restrict to allowed values |
