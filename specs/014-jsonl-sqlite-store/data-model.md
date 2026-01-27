# Data Model: Generic JSONL + SQLite Store Abstraction

**Feature**: 014-jsonl-sqlite-store
**Date**: 2026-01-27

## Entities

### Schema

Defines the structure of a store's records.

```go
// Schema defines the structure of a store.
type Schema struct {
    Name   string            `json:"name"`
    Fields map[string]*Field `json:"fields"`
}

// Field defines a single field in a schema.
type Field struct {
    Type    FieldType `json:"type"`
    Primary bool      `json:"primary,omitempty"`
    Index   bool      `json:"index,omitempty"`
    FTS     bool      `json:"fts,omitempty"`
    Enum    []string  `json:"enum,omitempty"`
}

// FieldType represents the data type of a field.
type FieldType string

const (
    FieldTypeString   FieldType = "string"
    FieldTypeInteger  FieldType = "integer"
    FieldTypeFloat    FieldType = "float"
    FieldTypeBoolean  FieldType = "boolean"
    FieldTypeDate     FieldType = "date"     // ISO 8601: YYYY-MM-DD
    FieldTypeDatetime FieldType = "datetime" // ISO 8601: YYYY-MM-DDTHH:MM:SSZ
    FieldTypeJSON     FieldType = "json"     // Stored as TEXT, queryable via JSON functions
)
```

**Validation rules**:
- Exactly one field must have `primary: true`
- Field names must be valid SQLite identifiers (alphanumeric + underscore)
- Enum values must be non-empty strings if specified
- FTS only valid for string fields

### Store

Represents a registered store with its schema and file paths.

```go
// Store represents a registered data store.
type Store struct {
    Name       string
    Schema     *Schema
    Dir        string // Directory containing JSONL and DB files
    jsonlPath  string // Derived: Dir/<name>.jsonl
    dbPath     string // Derived: Dir/<name>.db
}
```

**Derived paths**:
- JSONL: `<dir>/<name>.jsonl`
- SQLite: `<dir>/<name>.db`

### StoreRegistry

Configuration file listing all registered stores.

```go
// StoreRegistry is the configuration file format for stores.json.
type StoreRegistry struct {
    Stores map[string]*StoreConfig `json:"stores"`
}

// StoreConfig defines a single store's configuration.
type StoreConfig struct {
    SchemaPath string `json:"schema"` // Relative path to schema file
}
```

**File location**: `.bipartite/stores.json`

### Record

A single JSON object conforming to a store's schema.

```go
// Record represents a single record in a store.
// Stored as map[string]any since schema is dynamic.
type Record map[string]any
```

**Identity**: Primary key field value
**Uniqueness**: No duplicate primary keys allowed within a store

## SQLite Schema Generation

### Main Table

Generated from schema fields:

```sql
CREATE TABLE <store_name> (
    <primary_field> <sql_type> PRIMARY KEY,
    <field_2> <sql_type>,
    ...
);
```

**Type mapping**:

| Schema Type | SQLite Type |
|-------------|-------------|
| string      | TEXT        |
| integer     | INTEGER     |
| float       | REAL        |
| boolean     | INTEGER     |
| date        | TEXT        |
| datetime    | TEXT        |
| json        | TEXT        |

### Indexes

For each field with `index: true`:

```sql
CREATE INDEX idx_<store>_<field> ON <store>(<field>);
```

### FTS5 Table

If any field has `fts: true`:

```sql
CREATE VIRTUAL TABLE <store>_fts USING fts5(
    <primary_field>,
    <fts_field_1>,
    <fts_field_2>,
    ...
);
```

### Metadata Table

For sync hash tracking:

```sql
CREATE TABLE _meta (
    key TEXT PRIMARY KEY,
    value TEXT
);
```

Keys:
- `jsonl_hash`: SHA256 of JSONL file content
- `last_sync`: ISO 8601 timestamp of last sync

## State Transitions

### Store Lifecycle

```
[Not Exists] --init--> [Empty] --append--> [Has Records]
                          |                      |
                          v                      v
                       [Synced]  <--sync--  [Out of Sync]
                          |                      |
                          +-------delete---------+
```

### Sync States

```
[Unknown] --compute hash--> [Hash Known]
    |                            |
    v                            v
[First Sync]              [Compare Hash]
                               /    \
                              /      \
                           Match    Mismatch
                             |          |
                             v          v
                         [Skip]    [Rebuild]
```

## File Formats

### Schema File (JSON)

```json
{
  "name": "gh_activity",
  "fields": {
    "id": {"type": "string", "primary": true},
    "type": {"type": "string", "index": true, "enum": ["commit", "pr", "issue"]},
    "repo": {"type": "string", "index": true},
    "date": {"type": "datetime", "index": true},
    "title": {"type": "string", "fts": true},
    "body": {"type": "string", "fts": true},
    "data": {"type": "json"}
  }
}
```

### Store Registry (JSON)

```json
{
  "stores": {
    "gh_activity": {
      "schema": ".bipartite/schemas/gh_activity.json"
    },
    "refs": {
      "schema": ".bipartite/schemas/refs.json"
    }
  }
}
```

### JSONL Record

```json
{"id":"abc123","type":"pr","repo":"matsen/bipartite","date":"2026-01-27T10:30:00Z","title":"Add store feature","body":"...","data":{}}
```

## Relationships

```
StoreRegistry 1--* StoreConfig
StoreConfig 1--1 Schema
Store 1--1 Schema
Store 1--* Record
```

## Constraints

1. **Primary key uniqueness**: Enforced at append time by scanning existing records
2. **Enum validation**: Enforced at append time before writing to JSONL
3. **Type validation**: Enforced at append time with type coercion where safe (string→string, int→float)
4. **Required fields**: Primary key field is always required
5. **File co-location**: JSONL and DB files must be in same directory with same base name
