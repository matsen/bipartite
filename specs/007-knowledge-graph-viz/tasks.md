# Tasks: Knowledge Graph Visualization

**Input**: Design documents from `/specs/007-knowledge-graph-viz/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: Not explicitly requested in feature specification. Tests are NOT included.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Project structure**: `cmd/bip/` for CLI commands, `internal/viz/` for visualization package
- **Test fixtures**: `testdata/viz/` for test data

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the viz package structure and test fixtures

- [X] T001 Create internal/viz/ directory structure
- [X] T002 [P] Create testdata/viz/small_graph/ with refs.jsonl, concepts.jsonl, edges.jsonl fixtures
- [X] T003 [P] Create testdata/viz/empty_graph/ with empty JSONL fixtures for edge case testing

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core data extraction and types that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Define GraphData, Node, and Edge types in internal/viz/types.go
- [X] T005 Implement ExtractGraphData function to query SQLite and build GraphData in internal/viz/graph.go
- [X] T006 Add connection count computation for concept nodes in internal/viz/graph.go
- [X] T007 Handle empty graph case (no nodes) with appropriate sentinel or empty state in internal/viz/graph.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Generate Basic Visualization (Priority: P1) 🎯 MVP

**Goal**: Generate self-contained HTML with Cytoscape.js that displays paper and concept nodes with connecting edges

**Independent Test**: Run `bip viz` with a populated library and open the resulting HTML in a browser to see nodes and edges rendered

### Implementation for User Story 1

- [X] T008 [US1] Create HTML template constant with Cytoscape.js CDN script tag in internal/viz/html.go
- [X] T009 [US1] Implement ToCytoscapeJSON method to convert GraphData to Cytoscape.js format in internal/viz/cytoscape.go
- [X] T010 [US1] Add CSS styles for paper nodes (blue circles) and concept nodes (orange diamonds) in internal/viz/html.go
- [X] T011 [US1] Add edge styling with colors by relationship type (introduces=green, applies=blue, models=purple, other=gray) in internal/viz/html.go
- [X] T012 [US1] Implement concept node sizing proportional to connection count in internal/viz/cytoscape.go
- [X] T013 [US1] Implement GenerateHTML function that combines template, CSS, and graph JSON in internal/viz/html.go
- [X] T014 [US1] Create bip viz command with --output flag in cmd/bip/viz.go
- [X] T015 [US1] Wire up viz command: open database, extract graph, generate HTML, output to stdout or file in cmd/bip/viz.go
- [X] T016 [US1] Handle empty graph by rendering informative "No graph data" message in HTML in internal/viz/html.go

**Checkpoint**: User Story 1 complete - `bip viz` generates viewable HTML with styled graph

---

## Phase 4: User Story 2 - Explore Graph via Hover Tooltips (Priority: P2)

**Goal**: Show detailed information in tooltips when hovering over nodes and edges

**Independent Test**: Generate HTML, open in browser, hover over paper/concept nodes and edges to verify tooltip content appears

### Implementation for User Story 2

- [X] T017 [US2] Add tooltip data attributes to paper nodes (title, authors, year) in internal/viz/cytoscape.go
- [X] T018 [US2] Add tooltip data attributes to concept nodes (name, description, aliases) in internal/viz/cytoscape.go
- [X] T019 [US2] Add tooltip data attributes to edges (relationship type, summary) in internal/viz/cytoscape.go
- [X] T020 [US2] Implement CSS-only tooltip styling with :hover pseudo-element in internal/viz/html.go
- [X] T021 [US2] Add Cytoscape.js event handlers for mouseover/mouseout to show/hide tooltip div in internal/viz/html.go
- [X] T022 [US2] Handle missing optional fields gracefully (show "Unknown" or omit) in internal/viz/cytoscape.go

**Checkpoint**: User Story 2 complete - hovering shows detailed information for all element types

---

## Phase 5: User Story 3 - Click to Highlight Connections (Priority: P3)

**Goal**: Clicking a node highlights all directly connected nodes

**Independent Test**: Generate HTML, click nodes, verify connected nodes become visually highlighted

### Implementation for User Story 3

- [X] T023 [US3] Add Cytoscape.js tap event handler for nodes in internal/viz/html.go
- [X] T024 [US3] Implement getConnectedNodes helper function to find neighbors in internal/viz/html.go
- [X] T025 [US3] Add CSS classes for highlighted nodes and dimmed non-connected nodes in internal/viz/html.go
- [X] T026 [US3] Apply highlight styling to connected nodes on tap in internal/viz/html.go
- [X] T027 [US3] Clear highlighting when clicking on empty canvas area in internal/viz/html.go

**Checkpoint**: User Story 3 complete - clicking nodes highlights their connections

---

## Phase 6: User Story 4 - Offline Mode (Priority: P4)

**Goal**: Bundle Cytoscape.js inline for offline use with `--offline` flag

**Independent Test**: Generate HTML with `--offline` flag, disconnect from internet, verify visualization still renders

### Implementation for User Story 4

- [X] T028 [US4] Download cytoscape.min.js (v3.x) and add to internal/viz/assets/ directory
- [X] T029 [US4] Add go:embed directive for cytoscape.min.js in internal/viz/embed.go
- [X] T030 [US4] Modify GenerateHTML to accept offline bool parameter in internal/viz/html.go
- [X] T031 [US4] Conditionally embed inline script or CDN reference based on offline flag in internal/viz/html.go
- [X] T032 [US4] Add --offline flag to bip viz command in cmd/bip/viz.go

**Checkpoint**: User Story 4 complete - `bip viz --offline` generates fully self-contained HTML

---

## Phase 7: User Story 5 - Layout Options (Priority: P5)

**Goal**: Allow users to choose different graph layout algorithms

**Independent Test**: Generate HTML with different `--layout` options and verify different visual arrangements

### Implementation for User Story 5

- [X] T033 [US5] Define layout name mapping (force→cose, circle→circle, grid→grid) in internal/viz/cytoscape.go
- [X] T034 [US5] Add layout configuration to Cytoscape initialization in internal/viz/html.go
- [X] T035 [US5] Modify GenerateHTML to accept layout string parameter in internal/viz/html.go
- [X] T036 [US5] Add --layout flag with choices (force, circle, grid) and default=force to cmd/bip/viz.go
- [X] T037 [US5] Validate layout flag value and error on invalid choice in cmd/bip/viz.go

**Checkpoint**: User Story 5 complete - `bip viz --layout <type>` arranges graph using specified algorithm

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and documentation

- [X] T038 Verify all edge cases from spec.md: orphan concepts, missing fields, very large graphs
- [X] T039 Validate HTML file size under 1MB for typical graphs (under 500 nodes)
- [X] T040 Run quickstart.md validation manually
- [X] T041 Update CLAUDE.md with bip viz command documentation if needed

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can proceed sequentially in priority order (P1 → P2 → P3 → P4 → P5)
  - US2-US5 build incrementally on US1 functionality
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories - **MVP**
- **User Story 2 (P2)**: Depends on US1 HTML structure being complete
- **User Story 3 (P3)**: Depends on US1 HTML structure being complete; independent of US2
- **User Story 4 (P4)**: Depends on US1 GenerateHTML function; independent of US2/US3
- **User Story 5 (P5)**: Depends on US1 Cytoscape initialization; independent of US2/US3/US4

### Within Each User Story

- Types and data extraction before HTML generation
- HTML template before JavaScript behaviors
- Core implementation before CLI integration
- Story complete before moving to next priority

### Parallel Opportunities

- T002 and T003 can run in parallel (different fixture directories)
- Within US1: T008-T012 are mostly parallel (different aspects of template/styling)
- Within US2: T017-T019 are parallel (different element types)
- US3, US4, US5 could theoretically run in parallel after US1 completes (if team capacity allows)

---

## Parallel Example: User Story 1

```bash
# After foundational tasks complete, these US1 tasks can run in parallel:
Task: "Create HTML template constant with Cytoscape.js CDN script tag in internal/viz/html.go"
Task: "Implement ToCytoscapeJSON method to convert GraphData to Cytoscape.js format in internal/viz/cytoscape.go"

# Then styling tasks in parallel:
Task: "Add CSS styles for paper nodes (blue circles) and concept nodes (orange diamonds) in internal/viz/html.go"
Task: "Add edge styling with colors by relationship type in internal/viz/html.go"
Task: "Implement concept node sizing proportional to connection count in internal/viz/cytoscape.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test US1 by running `bip viz` and opening HTML in browser
5. Deploy/demo if ready - basic visualization is useful standalone

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → **MVP: viewable graph**
3. Add User Story 2 → Test independently → Tooltips add exploration
4. Add User Story 3 → Test independently → Click highlighting improves UX
5. Add User Story 4 → Test independently → Offline capability
6. Add User Story 5 → Test independently → Layout customization
7. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable after US1
- No tests included (not explicitly requested in spec)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- HTML generation is pure functions - easy to unit test if needed later
