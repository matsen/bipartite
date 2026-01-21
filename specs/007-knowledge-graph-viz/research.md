# Research: Knowledge Graph Visualization

**Feature**: 007-knowledge-graph-viz
**Date**: 2026-01-21

## 1. Visualization Library Selection

### Decision: Cytoscape.js

### Rationale
- Purpose-built for network/graph visualization
- Excellent documentation and active maintenance
- Multiple layout algorithms built-in (force-directed, circle, grid)
- Native tooltip support via popper extension
- ~500KB minified, fast CDN loading
- MIT licensed (compatible with project)
- Well-suited for academic knowledge graphs

### Alternatives Considered

| Library | Pros | Cons | Why Rejected |
|---------|------|------|--------------|
| D3.js force graph | Maximum flexibility, industry standard | Much more code to write, no built-in graph abstractions | Overkill for this use case; Cytoscape provides what we need out of box |
| vis.js | Similar feature set to Cytoscape | Slightly larger bundle, less active maintenance | Cytoscape has better documentation and community |
| sigma.js | Optimized for very large graphs | More complex setup, overkill for <500 nodes | Our scale doesn't need the extra complexity |

## 2. HTML Template Approach

### Decision: Go html/template with embedded template string

### Rationale
- Standard library, no external dependencies
- Proper HTML escaping for security
- Can embed template as string constant (no external files needed)
- Straightforward JSON embedding via template variables

### Implementation Pattern
```go
const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
  <script src="{{.CytoscapeURL}}"></script>
  <style>{{.CSS}}</style>
</head>
<body>
  <div id="cy"></div>
  <script>
    const data = {{.GraphJSON}};
    // Cytoscape initialization
  </script>
</body>
</html>`
```

### Alternatives Considered
- External template files: Requires file distribution with binary
- Text templates: Less safe for HTML content

## 3. Offline Mode Implementation

### Decision: Fetch and embed Cytoscape.js at build time via go:embed

### Rationale
- Cytoscape.js minified is ~500KB - acceptable for --offline flag
- go:embed allows bundling at compile time
- No runtime HTTP requests needed
- User explicitly opts in with --offline flag

### Implementation Pattern
```go
//go:embed cytoscape.min.js
var cytoscapeJS string

func getScriptTag(offline bool) string {
    if offline {
        return "<script>" + cytoscapeJS + "</script>"
    }
    return `<script src="https://unpkg.com/cytoscape@3/dist/cytoscape.min.js"></script>`
}
```

### Alternatives Considered
- Runtime fetch and cache: Adds complexity, requires network for first use
- Always bundle: Wastes bandwidth for online users

## 4. Graph Data Structure

### Decision: Cytoscape.js native format

### Rationale
- Direct compatibility with cy.add()
- No client-side transformation needed
- Well-documented structure

### Format
```json
{
  "nodes": [
    { "data": { "id": "paper-1", "type": "paper", "label": "Title...", ... } },
    { "data": { "id": "concept-1", "type": "concept", "label": "Name", ... } }
  ],
  "edges": [
    { "data": { "source": "paper-1", "target": "concept-1", "type": "introduces", ... } }
  ]
}
```

## 5. Tooltip Implementation

### Decision: CSS-only tooltips with data attributes

### Rationale
- No additional JS library needed (popper.js would add ~20KB)
- Sufficient for our simple hover tooltips
- Works in all modern browsers

### Implementation Pattern
```css
node:hover::after {
  content: attr(data-tooltip);
  /* positioning styles */
}
```

Note: If CSS tooltips prove insufficient (e.g., for rich HTML content), we can add popper extension later.

## 6. Layout Algorithms

### Decision: Support force (cose), circle, grid layouts

### Rationale
- cose (Compound Spring Embedder): Good default for knowledge graphs, shows clusters
- circle: Useful for small graphs, shows all nodes clearly
- grid: Simple, predictable arrangement

### Cytoscape Layout Names
- `--layout force` → Cytoscape `cose` layout
- `--layout circle` → Cytoscape `circle` layout
- `--layout grid` → Cytoscape `grid` layout

## 7. Color Scheme

### Decision: Semantic colors by node/edge type

### Rationale
- Consistent with spec requirements
- Colorblind-friendly palette

### Color Mapping
| Element | Color | Hex |
|---------|-------|-----|
| Paper nodes | Blue | #4A90D9 |
| Concept nodes | Orange | #E8923A |
| Edge: introduces | Green | #5CB85C |
| Edge: applies | Blue | #337AB7 |
| Edge: models | Purple | #9B59B6 |
| Edge: other | Gray | #95A5A6 |

## 8. Empty Graph Handling

### Decision: Show informative message instead of empty visualization

### Rationale
- Spec requires graceful handling of empty graphs
- Better UX than blank screen

### Implementation
When graph has 0 nodes:
```html
<div class="empty-state">
  <h2>No graph data</h2>
  <p>Add concepts and edges using bip concept and bip edge commands.</p>
</div>
```

## Summary

All technical decisions are resolved. No NEEDS CLARIFICATION items remain. Ready for Phase 1 design artifacts.
