# CLI Command Contracts: Generic Store

**Feature**: 014-jsonl-sqlite-store
**Date**: 2026-01-27

## Command Group

```
bip store <subcommand> [flags]
```

All store commands support `--json` flag for JSON output (agent-first design).

---

## bip store init

Initialize a new store.

### Usage

```
bip store init <name> --schema <path> [--dir <path>] [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Yes | Store name (alphanumeric + underscore) |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--schema` | `-s` | (required) | Path to JSON schema file |
| `--dir` | `-d` | `.bipartite/` | Directory for store files |
| `--json` | | false | Output JSON |

### Output (Human)

```
Created store 'gh_activity':
  JSONL: .bipartite/gh_activity.jsonl
  DB:    .bipartite/gh_activity.db
  Schema: .bipartite/schemas/gh_activity.json
```

### Output (JSON)

```json
{
  "name": "gh_activity",
  "jsonl_path": ".bipartite/gh_activity.jsonl",
  "db_path": ".bipartite/gh_activity.db",
  "schema_path": ".bipartite/schemas/gh_activity.json"
}
```

### Errors

| Condition | Exit Code | Message |
|-----------|-----------|---------|
| Store exists | 1 | `store 'gh_activity' already exists` |
| Invalid schema | 1 | `invalid schema: <reason>` |
| Schema file not found | 1 | `schema file not found: <path>` |

---

## bip store append

Append records to a store.

### Usage

```
bip store append <name> [json] [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Yes | Store name |
| `json` | No | JSON record (if not using --file or --stdin) |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | | Path to JSON/JSONL file |
| `--stdin` | | false | Read JSONL from stdin |
| `--json` | | false | Output JSON |

### Output (Human)

```
Appended 1 record to 'gh_activity'
```

### Output (JSON)

```json
{
  "store": "gh_activity",
  "appended": 1
}
```

### Errors

| Condition | Exit Code | Message |
|-----------|-----------|---------|
| Store not found | 1 | `store 'foo' not found` |
| Validation error | 1 | `validation error: field 'type' must be one of [commit, pr, issue]` |
| Duplicate primary key | 1 | `duplicate primary key: 'abc123' already exists` |
| Invalid JSON | 1 | `invalid JSON: <parse error>` |

---

## bip store delete

Delete records from a store.

### Usage

```
bip store delete <name> [id] [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Yes | Store name |
| `id` | No | Primary key of record to delete (if not using --where) |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--where` | `-w` | | SQL WHERE clause for batch delete |
| `--json` | | false | Output JSON |

### Output (Human)

```
Deleted 3 records from 'gh_activity'
```

### Output (JSON)

```json
{
  "store": "gh_activity",
  "deleted": 3
}
```

### Errors

| Condition | Exit Code | Message |
|-----------|-----------|---------|
| Store not found | 1 | `store 'foo' not found` |
| Record not found | 1 | `record 'abc123' not found` |
| Invalid WHERE | 1 | `invalid WHERE clause: <sql error>` |

---

## bip store sync

Sync JSONL to SQLite index.

### Usage

```
bip store sync [name] [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | No | Store name (omit for --all) |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--all` | `-a` | false | Sync all registered stores |
| `--json` | | false | Output JSON |

### Output (Human)

```
Synced 'gh_activity': 1,234 records (rebuilt)
```

or

```
'gh_activity' already in sync (skipped)
```

### Output (JSON)

```json
{
  "store": "gh_activity",
  "records": 1234,
  "action": "rebuilt"
}
```

or

```json
{
  "store": "gh_activity",
  "records": 1234,
  "action": "skipped"
}
```

### Errors

| Condition | Exit Code | Message |
|-----------|-----------|---------|
| Store not found | 1 | `store 'foo' not found` |
| JSONL corrupted | 1 | `parse error at line 42: <json error>` |

---

## bip store query

Query a store using SQL.

### Usage

```
bip store query <name> <sql> [flags]
```

or for cross-store:

```
bip store query --cross <sql> [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Yes* | Store name (*not required with --cross) |
| `sql` | Yes | SQL SELECT statement |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--cross` | `-x` | false | Enable cross-store query (ATTACHes all stores) |
| `--json` | | false | Output JSON array |
| `--csv` | | false | Output CSV |
| `--jsonl` | | false | Output JSONL |
| `--human` | | true | Output formatted table (default) |

### Output (Human)

```
ID       TYPE  REPO               AUTHOR  DATE
pr-123   pr    matsen/bipartite   erick   2026-01-27
pr-124   pr    matsen/bipartite   erick   2026-01-26
(2 rows)
```

### Output (JSON)

```json
[
  {"id": "pr-123", "type": "pr", "repo": "matsen/bipartite", "author": "erick", "date": "2026-01-27"},
  {"id": "pr-124", "type": "pr", "repo": "matsen/bipartite", "author": "erick", "date": "2026-01-26"}
]
```

### Errors

| Condition | Exit Code | Message |
|-----------|-----------|---------|
| Store not found | 1 | `store 'foo' not found` |
| Store not synced | 1 | `store 'foo' not synced, run 'bip store sync foo' first` |
| SQL error | 1 | `SQL error: <sqlite error>` |

---

## bip store list

List all registered stores.

### Usage

```
bip store list [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | | false | Output JSON |

### Output (Human)

```
NAME          RECORDS  PATH
refs          1,247    .bipartite/refs.jsonl
concepts      89       .bipartite/concepts.jsonl
gh_activity   2,341    .bipartite/gh_activity.jsonl
```

### Output (JSON)

```json
[
  {"name": "refs", "records": 1247, "path": ".bipartite/refs.jsonl"},
  {"name": "concepts", "records": 89, "path": ".bipartite/concepts.jsonl"},
  {"name": "gh_activity", "records": 2341, "path": ".bipartite/gh_activity.jsonl"}
]
```

---

## bip store info

Show detailed information about a store.

### Usage

```
bip store info <name> [flags]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Yes | Store name |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--json` | | false | Output JSON |

### Output (Human)

```
Store: gh_activity

Files:
  JSONL:  .bipartite/gh_activity.jsonl (234 KB)
  DB:     .bipartite/gh_activity.db (512 KB)
  Schema: .bipartite/schemas/gh_activity.json

Records: 2,341
Last Sync: 2026-01-27T10:30:00Z
Sync Status: In sync

Schema:
  id       string   (primary)
  type     string   (index, enum: commit|pr|issue|review)
  repo     string   (index)
  author   string   (index)
  date     datetime (index)
  title    string   (fts)
  body     string   (fts)
  url      string
```

### Output (JSON)

```json
{
  "name": "gh_activity",
  "jsonl_path": ".bipartite/gh_activity.jsonl",
  "db_path": ".bipartite/gh_activity.db",
  "schema_path": ".bipartite/schemas/gh_activity.json",
  "records": 2341,
  "last_sync": "2026-01-27T10:30:00Z",
  "in_sync": true,
  "schema": {
    "name": "gh_activity",
    "fields": {
      "id": {"type": "string", "primary": true},
      "type": {"type": "string", "index": true, "enum": ["commit", "pr", "issue", "review"]},
      ...
    }
  }
}
```

### Errors

| Condition | Exit Code | Message |
|-----------|-----------|---------|
| Store not found | 1 | `store 'foo' not found` |
