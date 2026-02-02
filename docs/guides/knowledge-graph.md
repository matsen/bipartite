# Knowledge Graph

The knowledge graph connects your internal world (projects, repos, concepts) to the external academic world (papers, citations, authors). Edges carry meaning: a paper *introduces* a concept, a project *implements* a method.

![Knowledge graph structure](../images/knowledge-graph.svg)

## Structure

The graph has three node types and directed edges between them:

- **Papers** — imported from Paperpile or Semantic Scholar
- **Concepts** — ideas, methods, or topics (e.g. `variational-autoencoder`, `phylogenetic-inference`)
- **Projects** — research efforts that group repos and connect to concepts

Edges link nodes with typed relationships and optional summary text.

## Data Storage

```
.bipartite/
├── refs.jsonl       # Papers (source of truth)
├── edges.jsonl      # Knowledge graph edges
├── concepts.jsonl   # Concept nodes
├── projects.jsonl   # Project nodes
├── repos.jsonl      # Repository nodes
├── config.yml      # Local config
└── cache/
    └── refs.db      # SQLite index (ephemeral, gitignored)
```

Everything is JSONL — human-readable, git-mergeable, diff-friendly. The SQLite cache is ephemeral and rebuilds on `bip rebuild`.

## Concepts

Concepts are the bridge between papers and projects:

```bash
bip concept add variational-autoencoder --name "Variational Autoencoder"
bip concept list
bip concept get variational-autoencoder
bip concept papers variational-autoencoder    # Papers linked to this concept
bip concept merge old-concept new-concept     # Merge, updating all edges
bip concept delete unused-concept
```

## Projects

Projects group repos and connect to the literature through concepts:

```bash
bip project add dasm2 --name "Deep Amino-acid Selection Models"
bip project list
bip project concepts dasm2      # Concepts linked to this project
bip project papers dasm2        # Papers relevant (via linked concepts)
bip project repos dasm2         # Repos belonging to this project
bip project import config.yml  # Bulk import from config file
```

`bip project papers` traverses the graph: project → concepts → papers. This lets an agent find all literature relevant to a project without manual curation of paper lists.

## Edges

Edges are directed relationships between any two nodes:

```bash
bip edge add -s Kingma2014-mo -t variational-autoencoder -r introduces -m "Introduced the VAE framework"
bip edge list                           # All edges
bip edge list Kingma2014-mo             # Edges involving a specific paper
bip edge search --type introduces       # Filter by relationship type
bip paper concepts Smith2024-ab         # Concepts linked to a paper
```

### Relationship Types

The `--type` flag accepts any string, but the visualization uses color coding for these common types:

| Type | Meaning | Viz color |
|------|---------|-----------|
| `introduces` | Paper introduces a concept or method | Green |
| `applies` | Paper applies an existing method | Blue |
| `models` | Paper models a phenomenon | Purple |
| Other | Any custom relationship | Gray |

## Visualization

```bash
bip viz > graph.html                     # Interactive HTML to stdout
bip viz --output graph.html              # Write to file
bip viz --layout circle --output g.html  # Circular layout
bip viz --offline --output g.html        # Bundle Cytoscape.js for offline use
```

The visualization renders papers as blue circles and concepts as orange diamonds, with colored edges showing relationship types.

## Edge Maintenance

```bash
bip groom              # Find edges referencing removed papers
bip groom --fix        # Remove orphaned edges after confirmation
bip edge export > edges-backup.jsonl
bip edge import edges.jsonl
```

## Generic Stores

For data beyond the built-in node types, bipartite provides generic JSONL-backed stores with SQLite query indexes:

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

## Agent Usage

Agents can traverse the graph programmatically:

```bash
# Find literature relevant to a project
bip project papers dasm2

# What concepts does a paper contribute to?
bip paper concepts Smith2024-ab

# Build context for a research question
bip concept papers phylogenetic-inference | jq '.[].id'
```

All output is JSON by default. Add `--human` for readable output.
