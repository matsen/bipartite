# Projects, Repos, and Stores

Beyond the paper library, a nexus tracks the internal side of your research world: the projects you work on, the repositories that implement them, and any custom record types you want to keep alongside.

## Data Storage

```
my-nexus/
├── .bipartite/
│   ├── refs.jsonl        # Paper library (source of truth)
│   ├── projects.jsonl    # Projects
│   ├── repos.jsonl       # Tracked repositories
│   ├── config.yml        # Local settings (PDF paths, etc.)
│   └── cache/
│       └── refs.db       # SQLite index (ephemeral, gitignored)
│
├── servers.yml           # (optional) Remote servers for bip scout
├── sources.yml           # (optional) GitHub repos for activity tracking
│
├── context/              # (optional) Project context files
└── narrative/            # (optional) Generated digest output
```

Everything is JSONL — human-readable, git-mergeable, diff-friendly. The SQLite cache is ephemeral and rebuilds on `bip rebuild`.

## Projects

Projects group related repositories:

```bash
bip project add dasm2 --name "Deep Amino-acid Selection Models"
bip project list
bip project get dasm2
bip project repos dasm2         # Repos belonging to this project
bip project update dasm2 --description "New description"
bip project import config.yml   # Bulk import from a config file
bip project delete dasm2        # Use --force if it still has repos
```

Project IDs must not collide with paper IDs — `bip project add` rejects an ID that already exists in `refs.jsonl`.

### Bulk Import

`bip project import` reads a YAML file with project IDs as keys:

```yaml
dasm:
  name: DASM
  repos:
    - matsengrp/netam
    - matsengrp/dasm2-experiments
  context: context/dasm.md
```

Use `--dry-run` to preview, and `--no-fetch` to skip the GitHub metadata lookup.

## Repos

```bash
bip repo add --project dasm2 matsengrp/netam
bip repo list
bip repo get netam
bip repo refresh netam          # Re-fetch GitHub metadata
```

Repos added from GitHub carry metadata (description, language, topics) fetched via the API; see [Configuration](configuration.md) for the token setup and rate limits. Repos created with `--manual` have no upstream to refresh.

## Generic Stores

For data beyond the built-in types, bipartite provides generic JSONL-backed stores with SQLite query indexes:

```bash
bip store init my_store --schema schema.json
bip store append my_store '{"id": "foo", "title": "Example"}'
bip store sync my_store        # Rebuild SQLite from JSONL
bip store query my_store "SELECT * FROM my_store WHERE title LIKE '%example%'"
bip store query --cross "SELECT * FROM refs JOIN my_store ON ..."
bip store list
bip store info my_store
bip store delete my_store foo
```

Schemas define field types, indexes, enums, and full-text search:

```json
{
  "name": "my_store",
  "fields": {
    "id": {"type": "string", "primary": true},
    "title": {"type": "string", "fts": true},
    "status": {"type": "string", "index": true, "enum": ["active", "archived"]}
  }
}
```

`--cross` joins a store against the built-in tables, which is how an agent combines custom records with the paper library in a single query.

## Agent Usage

All output is JSON by default; add `--human` for readable output.

```bash
# What repos does a project own?
bip project repos dasm2

# Pull custom records into the same query as the paper library
bip store query --cross "SELECT r.title FROM refs r JOIN my_store m ON m.id = r.id"
```
