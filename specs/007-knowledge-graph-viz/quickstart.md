# Quickstart: Knowledge Graph Visualization

**Feature**: 007-knowledge-graph-viz
**Date**: 2026-01-21

## Prerequisites

1. Go 1.25.5 installed
2. bipartite repository cloned
3. Library initialized with `bip init`
4. Database rebuilt with `bip rebuild`

## Build

```bash
go build -o bip ./cmd/bip
```

## Basic Usage

### Generate and view visualization

```bash
# Generate HTML to stdout, redirect to file
./bip viz > knowledge-graph.html

# Or use --output flag
./bip viz --output knowledge-graph.html

# Open in browser (macOS)
open knowledge-graph.html

# Open in browser (Linux)
xdg-open knowledge-graph.html
```

### Layout options

```bash
# Force-directed layout (default)
./bip viz --layout force --output graph.html

# Circular layout
./bip viz --layout circle --output graph.html

# Grid layout
./bip viz --layout grid --output graph.html
```

### Offline mode

```bash
# Bundle Cytoscape.js inline (no CDN dependency)
./bip viz --offline --output graph-offline.html
```

## Example Workflow

```bash
# 1. Add some concepts to your library
./bip concept add mutation-selection-model "Mutation-Selection Model" \
  --description "A model combining mutation rates with selection pressures"

./bip concept add phylogenetics "Phylogenetics" \
  --description "Study of evolutionary relationships"

# 2. Add edges linking papers to concepts
./bip edge add Halpern1998-yc mutation-selection-model introduces \
  --summary "Introduces the mutation-selection model"

./bip edge add Halpern1998-yc phylogenetics applies \
  --summary "Applies phylogenetic methods"

# 3. Generate visualization
./bip viz --output my-graph.html

# 4. View in browser
open my-graph.html
```

## Testing During Development

```bash
# Run all viz tests
go test ./internal/viz/...

# Run with verbose output
go test -v ./internal/viz/...

# Run specific test
go test -v -run TestGenerateHTML ./internal/viz/...
```

## Test Fixtures

Test fixtures are in `testdata/viz/`:

- `small_graph/`: A few papers, concepts, and edges for basic tests
- `empty_graph/`: Empty JSONL files for edge case testing

## Troubleshooting

### "no such table: concepts"

Run `./bip rebuild` to rebuild the SQLite database from JSONL files.

### Empty visualization

Your library may not have any concept edges yet. Use `bip concept` and `bip edge` commands to add them.

### HTML won't open in browser

Ensure the file has `.html` extension and is a valid HTML file:

```bash
head -5 knowledge-graph.html
# Should show: <!DOCTYPE html>
```
