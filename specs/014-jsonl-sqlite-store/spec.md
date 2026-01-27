# Feature Specification: Generic JSONL + SQLite Store Abstraction

**Feature Branch**: `014-jsonl-sqlite-store`
**Created**: 2026-01-27
**Status**: Draft
**Input**: User description: "Add a general-purpose store abstraction where JSONL files are the source of truth and SQLite databases serve as queryable indexes (gitignored, rebuilt on demand)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Define and Initialize a Custom Store (Priority: P1)

A user wants to create a new store for tracking GitHub activity data. They define a JSON schema specifying fields, types, indexes, and full-text search capabilities, then initialize the store which creates the JSONL file and SQLite database.

**Why this priority**: Without the ability to define and create stores, no other functionality is possible. This is the foundation of the entire feature.

**Independent Test**: Can be fully tested by creating a schema file and running `bip store init`. Delivers immediate value by establishing the data storage infrastructure.

**Acceptance Scenarios**:

1. **Given** a valid JSON schema file defining fields with types and options, **When** running `bip store init gh_activity --schema schema.json`, **Then** the system creates an empty `.bipartite/gh_activity.jsonl` file and `.bipartite/gh_activity.db` SQLite database with the correct schema
2. **Given** a schema with `"fts": true` on specific fields, **When** initializing the store, **Then** the SQLite database includes an FTS5 virtual table for full-text search
3. **Given** a schema with `"index": true` on fields, **When** initializing the store, **Then** the SQLite database includes indexes on those columns
4. **Given** an invalid schema (missing primary key, unknown type), **When** running `bip store init`, **Then** the system reports a clear validation error and does not create files

---

### User Story 2 - Append and Query Records (Priority: P1)

A user wants to add records to their store and query them. They append JSON records via CLI and run SQL queries to retrieve data.

**Why this priority**: Core CRUD operations are essential for any useful store. Without read/write capabilities, the store has no value.

**Independent Test**: Can be tested by appending a few records and running queries. Delivers value by enabling data storage and retrieval.

**Acceptance Scenarios**:

1. **Given** an initialized store, **When** running `bip store append gh_activity '{"id":"123","type":"pr",...}'`, **Then** the record is appended to the JSONL file
2. **Given** a schema with enum constraints, **When** appending a record with an invalid enum value, **Then** the system rejects the record with a validation error
3. **Given** records in the JSONL file and a synced database, **When** running `bip store query gh_activity "SELECT * FROM gh_activity WHERE type = 'pr'"`, **Then** matching records are returned in the requested format
4. **Given** fields marked with `fts: true`, **When** running a full-text search query, **Then** matching records are found via the FTS5 index

---

### User Story 3 - Sync JSONL to SQLite (Priority: P1)

A user has appended multiple records to the JSONL file and wants to rebuild the SQLite database to reflect the current state.

**Why this priority**: The sync mechanism is core to the JSONL-as-source-of-truth design. Without it, queries cannot reflect the current data.

**Independent Test**: Can be tested by appending records, running sync, and verifying queries return the new data.

**Acceptance Scenarios**:

1. **Given** new records in the JSONL file not yet in SQLite, **When** running `bip store sync gh_activity`, **Then** the SQLite database is rebuilt to include all JSONL records
2. **Given** an unchanged JSONL file (same SHA256 hash), **When** running sync, **Then** the system skips the rebuild and reports "already in sync"
3. **Given** multiple stores registered, **When** running `bip store sync --all`, **Then** all stores are synced

---

### User Story 4 - Delete Records (Priority: P2)

A user wants to remove records from their store, either by ID or by a WHERE clause condition.

**Why this priority**: Delete capability is important but less frequently used than append/query. Can be deferred to a later iteration if needed.

**Independent Test**: Can be tested by appending records, deleting some, and verifying they're removed from both JSONL and queries.

**Acceptance Scenarios**:

1. **Given** a record with id "123" in the store, **When** running `bip store delete gh_activity 123`, **Then** the JSONL file is rewritten excluding that record and sync is triggered
2. **Given** records matching a condition, **When** running `bip store delete gh_activity --where "date < '2020-01-01'"`, **Then** all matching records are removed
3. **Given** a delete operation, **When** the JSONL is rewritten, **Then** the operation is atomic (no partial writes on failure)

---

### User Story 5 - List and Inspect Stores (Priority: P2)

A user wants to see all registered stores and inspect details of a specific store.

**Why this priority**: Management and introspection commands are useful but not critical for core functionality.

**Independent Test**: Can be tested by creating stores and running list/info commands.

**Acceptance Scenarios**:

1. **Given** multiple registered stores, **When** running `bip store list`, **Then** a table shows store names, record counts, and file paths
2. **Given** a registered store, **When** running `bip store info gh_activity`, **Then** the system displays schema, record count, last sync time, and file paths

---

### User Story 6 - Cross-Store Queries (Priority: P3)

A user wants to join data across multiple stores in a single SQL query.

**Why this priority**: Advanced feature that builds on single-store functionality. Valuable but not required for initial release.

**Independent Test**: Can be tested by creating two stores and running a JOIN query across them.

**Acceptance Scenarios**:

1. **Given** two stores (refs and gh_activity), **When** running `bip store query --cross "SELECT * FROM refs r JOIN gh_activity g ON ..."`, **Then** the query executes across both databases via SQLite ATTACH
2. **Given** a cross-store query, **When** store names conflict with SQL keywords, **Then** the system handles aliasing appropriately

---

### User Story 7 - Migrate Existing Stores (Priority: P3)

The existing bipartite stores (refs, concepts, edges) should be migrated to use the generic store abstraction.

**Why this priority**: Migration provides consistency but existing stores already work. Can be done incrementally after core features are stable.

**Independent Test**: Can be tested by defining schemas for existing stores and verifying `bip rebuild` uses the new sync mechanism.

**Acceptance Scenarios**:

1. **Given** existing refs.jsonl, concepts.jsonl, edges.jsonl files, **When** schemas are defined and registered, **Then** the generic store API can manage them
2. **Given** the migration is complete, **When** running `bip rebuild`, **Then** it calls `bip store sync --all` for all registered stores

---

### Edge Cases

- What happens when appending a record with a duplicate primary key? System should reject with a clear error.
- What happens when the JSONL file is corrupted (invalid JSON on a line)? Sync should report the specific line and error.
- What happens when disk space runs out during JSONL rewrite? Operation should fail atomically without corrupting the original file.
- What happens when querying a store that hasn't been synced? System should warn and suggest running sync first.
- What happens when a schema field type changes? User must delete and rebuild the database (documented behavior).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support JSON schema definitions with field types: string, integer, float, boolean, date, datetime, json
- **FR-002**: System MUST support schema options: primary (exactly one required), index, fts, enum
- **FR-003**: System MUST create co-located files: `<name>.jsonl` and `<name>.db` in the same directory
- **FR-004**: System MUST validate records against schema on append (type checking, enum validation, required primary key)
- **FR-005**: System MUST append records atomically to JSONL files
- **FR-006**: System MUST implement hard deletes by rewriting JSONL files (no tombstones)
- **FR-007**: System MUST perform full SQLite rebuild on sync (drop tables, recreate, insert all)
- **FR-008**: System MUST skip sync when JSONL SHA256 hash matches stored hash in SQLite metadata
- **FR-009**: System MUST generate SQLite indexes for fields marked with `index: true`
- **FR-010**: System MUST generate FTS5 virtual tables for stores with fields marked `fts: true`
- **FR-011**: System MUST register stores in `.bipartite/stores.json` configuration file
- **FR-012**: System MUST support multiple output formats for queries: human-readable table, JSON, CSV, JSONL
- **FR-013**: System MUST support cross-store queries via SQLite ATTACH for all registered stores

### Key Entities

- **Store**: A named data collection with a schema, JSONL source file, and SQLite index database
- **Schema**: JSON definition of fields, types, and options (indexes, FTS, enums) for a store
- **Record**: A single JSON object conforming to a store's schema, identified by primary key

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can define a new store schema and initialize it in under 1 minute
- **SC-002**: Appending 1000 records completes in under 5 seconds
- **SC-003**: Syncing a store with 10,000 records completes in under 10 seconds
- **SC-004**: Full-text search queries return results in under 1 second for stores with 10,000 records
- **SC-005**: Cross-store joins execute successfully across at least 5 attached databases
- **SC-006**: All existing bipartite functionality (refs, concepts, edges) continues to work after migration
- **SC-007**: Schema validation catches 100% of type mismatches and enum violations before data is written

## Assumptions

- Store sizes will typically be in the thousands to tens of thousands of records (not millions)
- Full rebuild on sync is acceptable; incremental sync is not needed
- Users are comfortable with SQL for queries
- Schema migrations are handled by deleting and rebuilding the database (no ALTER TABLE support needed)
- All stores are local filesystem; no remote or distributed storage
