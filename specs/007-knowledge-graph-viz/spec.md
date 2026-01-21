# Feature Specification: Knowledge Graph Visualization

**Feature Branch**: `007-knowledge-graph-viz`
**Created**: 2026-01-21
**Status**: Draft
**Input**: User description: "Add bip viz command for interactive knowledge graph visualization using Cytoscape.js"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Generate Basic Visualization (Priority: P1)

A researcher wants to see their concept-paper knowledge graph as an interactive visualization to understand the structure of their literature collection and how concepts connect papers.

**Why this priority**: This is the core value proposition - without basic visualization generation, no other features matter.

**Independent Test**: Can be fully tested by running `bip viz` with a populated library and opening the resulting HTML in a browser to see nodes and edges rendered.

**Acceptance Scenarios**:

1. **Given** a library with papers, concepts, and edges, **When** user runs `bip viz`, **Then** valid HTML is output to stdout containing the graph data
2. **Given** a library with papers and concepts, **When** user runs `bip viz --output graph.html`, **Then** HTML file is created at the specified path
3. **Given** the generated HTML file, **When** user opens it in a browser, **Then** papers and concepts display as visually distinct nodes with connecting edges

---

### User Story 2 - Explore Graph via Hover Tooltips (Priority: P2)

A researcher hovering over nodes and edges wants to see detailed information without cluttering the main visualization, enabling quick exploration of the graph structure.

**Why this priority**: Tooltips transform a static picture into an explorable interface, adding significant utility without changing core functionality.

**Independent Test**: Can be tested by generating HTML, opening in browser, and hovering over various node and edge types to verify tooltip content appears.

**Acceptance Scenarios**:

1. **Given** the visualization is open in browser, **When** user hovers over a paper node, **Then** tooltip shows paper title, authors, and year
2. **Given** the visualization is open in browser, **When** user hovers over a concept node, **Then** tooltip shows concept name, description, and aliases
3. **Given** the visualization is open in browser, **When** user hovers over an edge, **Then** tooltip shows relationship type and summary

---

### User Story 3 - Click to Highlight Connections (Priority: P3)

A researcher clicks on a node to highlight all directly connected nodes, making it easy to see which papers relate to a specific concept or which concepts a paper touches.

**Why this priority**: Click interactions enhance exploration but are not essential for basic visualization utility.

**Independent Test**: Can be tested by generating HTML, clicking nodes, and verifying connected nodes become visually highlighted.

**Acceptance Scenarios**:

1. **Given** the visualization is open in browser, **When** user clicks a concept node, **Then** all connected paper nodes are visually highlighted
2. **Given** the visualization is open in browser, **When** user clicks a paper node, **Then** all connected concept nodes are visually highlighted
3. **Given** a node is currently highlighted, **When** user clicks elsewhere in the graph, **Then** highlighting is cleared

---

### User Story 4 - Offline Mode (Priority: P4)

A researcher working without internet connectivity needs the visualization to work completely offline by bundling all JavaScript dependencies inline.

**Why this priority**: Important for portability and offline use but not core functionality since CDN loading works for most users.

**Independent Test**: Can be tested by generating HTML with `--offline` flag, disconnecting from internet, and verifying the visualization still renders correctly.

**Acceptance Scenarios**:

1. **Given** user runs `bip viz --offline`, **When** HTML is generated, **Then** all JavaScript is bundled inline with no external CDN references
2. **Given** an offline-generated HTML file, **When** opened in a browser with no internet, **Then** visualization renders and functions correctly

---

### User Story 5 - Layout Options (Priority: P5)

A researcher wants to choose different layout algorithms to better visualize their graph based on its structure and size.

**Why this priority**: Layout options are a nice-to-have customization; the default force-directed layout works well for most graphs.

**Independent Test**: Can be tested by generating HTML with different `--layout` options and verifying different visual arrangements.

**Acceptance Scenarios**:

1. **Given** user runs `bip viz --layout force`, **When** HTML is opened, **Then** nodes are arranged using force-directed layout
2. **Given** user runs `bip viz --layout circle`, **When** HTML is opened, **Then** nodes are arranged in a circular layout
3. **Given** user runs `bip viz` without layout flag, **When** HTML is opened, **Then** force-directed layout is used as default

---

### Edge Cases

- What happens when the library has no concepts or edges? Visualization should render with only paper nodes (or empty state message if no papers either)
- What happens when a concept has no linked papers? Concept node should still appear as an orphan node
- What happens when paper/concept data has missing fields? System should gracefully handle missing optional fields (show "Unknown" or omit from tooltip)
- What happens with very large graphs (hundreds of nodes)? Visualization should still be responsive; performance may degrade but should not crash

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST generate valid, self-contained HTML that opens correctly in modern browsers (Chrome, Firefox, Safari, Edge)
- **FR-002**: System MUST embed all graph data (nodes and edges) directly in the HTML file
- **FR-003**: System MUST output HTML to stdout by default
- **FR-004**: System MUST support `--output` flag to write HTML to a specified file path
- **FR-005**: System MUST display paper nodes with distinct visual styling from concept nodes
- **FR-006**: System MUST size concept nodes proportionally to their number of connected papers
- **FR-007**: System MUST color edges by relationship type using distinct colors
- **FR-008**: System MUST show tooltips on hover for papers containing title, authors, and year
- **FR-009**: System MUST show tooltips on hover for concepts containing name, description, and aliases
- **FR-010**: System MUST show tooltips on hover for edges containing relationship type and summary
- **FR-011**: System MUST highlight connected nodes when a node is clicked
- **FR-012**: System MUST support `--offline` flag to bundle JavaScript inline instead of loading from CDN
- **FR-013**: System MUST support `--layout` flag with options: force (default), circle, grid
- **FR-014**: System MUST handle empty graphs gracefully (no concepts or no edges)
- **FR-015**: System MUST ensure user data never leaves the local machine (all processing and rendering is local)
- **FR-016**: System MUST generate HTML files under 1MB for typical graphs (under 500 nodes)

### Key Entities

- **Paper Node**: Represents a reference from refs.jsonl; displayed as a circle with paper metadata (ID, title, authors, year)
- **Concept Node**: Represents a concept from concepts.jsonl; displayed as a diamond with concept metadata (ID, name, description, aliases)
- **Edge**: Represents a paper-concept relationship from edges.jsonl; connects a paper to a concept with type (introduces, applies, models, etc.) and summary

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Generated HTML opens and renders correctly in all major browsers (Chrome, Firefox, Safari, Edge)
- **SC-002**: Users can identify all papers connected to any concept within 3 clicks/interactions
- **SC-003**: Users can understand what a paper or concept is about by hovering for under 1 second (tooltip appears promptly)
- **SC-004**: Visualization with 100 nodes renders initial layout in under 3 seconds
- **SC-005**: HTML file size remains under 1MB for graphs with up to 500 nodes (excluding offline JS bundle)
- **SC-006**: 100% of paper-concept relationships in the library are accurately represented in the visualization

## Assumptions

- Users have access to a modern web browser capable of running JavaScript
- The existing SQLite database contains all necessary data from refs.jsonl, concepts.jsonl, and edges.jsonl
- Cytoscape.js CDN (unpkg.com) is generally available; offline mode provides fallback
- Force-directed layout is appropriate as default for most academic knowledge graphs
- Graph sizes in typical use will be under 500 nodes; larger graphs may have degraded performance
