# Implementation Plan: Knowledge Graph Visualization

**Branch**: `007-knowledge-graph-viz` | **Date**: 2026-01-21 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/007-knowledge-graph-viz/spec.md`

## Summary

Add a `bip viz` command that generates self-contained HTML files for interactive visualization of the concept-paper knowledge graph using Cytoscape.js. The visualization embeds graph data directly in HTML, loads JS from CDN (or bundles inline with `--offline`), and supports multiple layout algorithms.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: spf13/cobra (CLI), modernc.org/sqlite (storage), html/template (HTML generation)
**Storage**: SQLite (read from existing refs, concepts, edges tables rebuilt from JSONL)
**Testing**: go test with table-driven tests and real JSONL fixtures
**Target Platform**: macOS, Linux (cross-platform CLI)
**Project Type**: single (CLI tool)
**Performance Goals**: 100-node graph renders in <3 seconds, HTML generation instant
**Constraints**: HTML file <1MB (excluding offline JS), user data never leaves machine
**Scale/Scope**: Typical graphs under 500 nodes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | PASS | CLI command with JSON-embeddable HTML output |
| II. Git-Versionable Architecture | PASS | Reads from JSONL via SQLite; HTML output is ephemeral |
| III. Fail-Fast Philosophy | PASS | Will error on missing database, invalid paths |
| IV. Real Testing (Agentic TDD) | PASS | Tests use real JSONL fixtures from testdata/ |
| V. Clean Architecture | PASS | Separate viz package for HTML generation |
| VI. Simplicity | PASS | Single command, no server, static HTML output |

**Technology Constraints Check:**
- CLI Responsiveness: PASS (Go compiled, no startup overhead)
- Embeddable Over Client-Server: PASS (SQLite reads, no server)
- Data Portability: PASS (HTML is human-readable, graph data embedded as JSON)
- Platform Support: PASS (macOS/Linux supported)

## Project Structure

### Documentation (this feature)

```text
specs/007-knowledge-graph-viz/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no API contracts)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/bip/
├── viz.go               # New: bip viz command implementation

internal/
├── viz/                 # New: visualization package
│   ├── graph.go         # Graph data extraction from SQLite
│   ├── html.go          # HTML template and generation
│   ├── cytoscape.go     # Cytoscape.js configuration
│   └── viz_test.go      # Tests

testdata/
├── viz/                 # New: test fixtures for viz
│   ├── small_graph/     # refs.jsonl, concepts.jsonl, edges.jsonl
│   └── empty_graph/     # Empty JSONL files
```

**Structure Decision**: Follows existing project structure (cmd/bip/ for commands, internal/ for packages). New `internal/viz/` package encapsulates all visualization logic, keeping it separate from storage concerns.

## Complexity Tracking

No violations to justify - design follows all constitution principles.

