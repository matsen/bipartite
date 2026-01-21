# CLI Contract: Edge Commands

**Feature**: 003-knowledge-graph
**Date**: 2026-01-13

## Overview

All edge commands are subcommands of `bip edge`. Each supports `--json` flag for structured output (agent-first design).

## Commands

### bp edge add

Add a single edge to the knowledge graph.

**Synopsis**:
```bash
bp edge add --source <paper-id> --target <paper-id> --type <relationship> --summary <text> [--json]
```

**Arguments**:
| Flag | Required | Description |
|------|----------|-------------|
| --source, -s | yes | Source paper ID (must exist in refs.jsonl) |
| --target, -t | yes | Target paper ID (must exist in refs.jsonl) |
| --type, -r | yes | Relationship type (e.g., "cites", "extends") |
| --summary, -m | yes | Relational summary text |
| --json | no | Output result as JSON |

**Behavior**:
- Validates source and target papers exist
- If edge with same (source, target, type) exists, updates summary
- Appends to edges.jsonl
- Updates SQLite index

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Source paper not found |
| 2 | Target paper not found |
| 3 | Invalid arguments |

**Output (human)**:
```
Added edge: Smith2024-ab --[extends]--> Jones2023-xy
```

**Output (JSON)**:
```json
{
  "action": "added",
  "edge": {
    "source_id": "Smith2024-ab",
    "target_id": "Jones2023-xy",
    "relationship_type": "extends",
    "summary": "Extends variational framework..."
  }
}
```

---

### bp edge import

Bulk import edges from a JSONL file.

**Synopsis**:
```bash
bp edge import <file> [--json]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| file | yes | Path to JSONL file with edges |
| --json | no | Output results as JSON |

**Input Format**: Each line is a JSON edge object:
```json
{"source_id":"...","target_id":"...","relationship_type":"...","summary":"..."}
```

**Behavior**:
- Validates each edge (source/target exist)
- Skips invalid edges with warning (continues processing)
- Updates existing edges (same source/target/type)
- Reports count of added, updated, skipped

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success (all or some edges imported) |
| 1 | File not found |
| 2 | All edges invalid (none imported) |

**Output (human)**:
```
Imported 45 edges (3 updated, 2 skipped)
Skipped:
  Line 12: source paper "Unknown2024" not found
  Line 34: target paper "Missing2023" not found
```

**Output (JSON)**:
```json
{
  "added": 42,
  "updated": 3,
  "skipped": 2,
  "errors": [
    {"line": 12, "error": "source paper \"Unknown2024\" not found"},
    {"line": 34, "error": "target paper \"Missing2023\" not found"}
  ]
}
```

---

### bp edge list

List edges for a specific paper.

**Synopsis**:
```bash
bp edge list <paper-id> [--incoming] [--all] [--json]
```

**Arguments**:
| Arg | Required | Description |
|-----|----------|-------------|
| paper-id | yes | Paper ID to list edges for |
| --incoming | no | Show edges where paper is target (default: source) |
| --all | no | Show both incoming and outgoing edges |
| --json | no | Output as JSON |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Paper not found |

**Output (human)**:
```
Outgoing edges from Smith2024-ab:
  --[extends]--> Jones2023-xy
    "Extends variational framework to non-Euclidean geometries"
  --[cites]--> Brown2022-cd
    "Cites foundational work on manifold learning"

Incoming edges to Smith2024-ab:
  Lee2025-ef --[builds-on]-->
    "Builds on Smith's geometric extensions"
```

**Output (JSON)**:
```json
{
  "paper_id": "Smith2024-ab",
  "outgoing": [
    {
      "target_id": "Jones2023-xy",
      "relationship_type": "extends",
      "summary": "Extends variational framework..."
    }
  ],
  "incoming": [
    {
      "source_id": "Lee2025-ef",
      "relationship_type": "builds-on",
      "summary": "Builds on Smith's geometric extensions"
    }
  ]
}
```

---

### bp edge search

Search edges by relationship type.

**Synopsis**:
```bash
bp edge search --type <relationship> [--json]
```

**Arguments**:
| Flag | Required | Description |
|------|----------|-------------|
| --type, -r | yes | Relationship type to filter by |
| --json | no | Output as JSON |

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success (may return empty results) |

**Output (human)**:
```
Edges with type "contradicts":
  Smith2024-ab --[contradicts]--> OldTheory2020-zz
    "Presents experimental evidence contradicting..."
  Chen2025-gh --[contradicts]--> Smith2024-ab
    "Challenges Smith's geometric assumptions..."
```

**Output (JSON)**:
```json
{
  "relationship_type": "contradicts",
  "edges": [
    {
      "source_id": "Smith2024-ab",
      "target_id": "OldTheory2020-zz",
      "summary": "Presents experimental evidence..."
    }
  ]
}
```

---

### bp edge export

Export edges to JSONL format.

**Synopsis**:
```bash
bp edge export [--paper <paper-id>]
```

**Arguments**:
| Flag | Required | Description |
|-----|----------|-------------|
| --paper, -p | no | Only export edges involving this paper |

**Behavior**:
- Writes to stdout (pipe to file as needed)
- Each line is a complete JSON edge object
- With --paper, includes both incoming and outgoing edges

**Exit Codes**:
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Paper not found (when --paper specified) |

**Output**:
```jsonl
{"source_id":"Smith2024-ab","target_id":"Jones2023-xy","relationship_type":"extends","summary":"...","created_at":"2026-01-13T10:30:00Z"}
{"source_id":"Smith2024-ab","target_id":"Brown2022-cd","relationship_type":"cites","summary":"...","created_at":"2026-01-13T10:31:00Z"}
```

---

## Integration with Existing Commands

### bp rebuild

Extended to rebuild edge index from edges.jsonl:
- Reads edges.jsonl
- Recreates edges table in SQLite
- Reports edge count in output

### bp groom

Extended to detect orphaned edges:
- Scans edges for references to missing papers
- Reports orphaned edges with option to remove
- `--fix` flag removes orphaned edges after confirmation

### bp check

Extended to verify edge integrity:
- All edge references point to existing papers
- No duplicate edges (same source/target/type)
- Valid JSON in edges.jsonl
