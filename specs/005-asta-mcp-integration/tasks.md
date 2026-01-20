# Tasks: ASTA MCP Integration

**Input**: Design documents from `/specs/005-asta-mcp-integration/`
**Prerequisites**: spec.md

## Phase 1: Setup

- [ ] T001 Create `internal/asta/` package directory
- [ ] T002 Create `internal/asta/client.go` with MCP HTTP client
- [ ] T003 Create `internal/asta/types.go` with request/response types
- [ ] T004 Create `internal/asta/errors.go` with error types
- [ ] T005 Update `.env` loading to include `ASTA_API_KEY`

## Phase 2: Parent Command

- [ ] T006 Create `cmd/bp/asta.go` with parent command and `--human` flag
- [ ] T007 Add exit codes for ASTA commands

## Phase 3: Search Commands (P1)

- [ ] T008 Create `cmd/bp/asta_search.go` - keyword search
- [ ] T009 Implement `--limit`, `--year`, `--venue` flags for search
- [ ] T010 Create `cmd/bp/asta_snippet.go` - snippet search
- [ ] T011 Implement snippet output formatting (show paper context)

## Phase 4: Paper Commands (P2)

- [ ] T012 Create `cmd/bp/asta_paper.go` - get paper details
- [ ] T013 Create `cmd/bp/asta_citations.go` - get citing papers
- [ ] T014 Create `cmd/bp/asta_references.go` - get referenced papers
- [ ] T015 Implement paper ID parsing (DOI:, ARXIV:, etc.)

## Phase 5: Author Commands (P3)

- [ ] T016 Create `cmd/bp/asta_author.go` - search authors
- [ ] T017 Create `cmd/bp/asta_author_papers.go` - get author's papers

## Phase 6: Polish

- [ ] T018 Add rate limiting (10 req/sec)
- [ ] T019 Update README.md with ASTA commands
- [ ] T020 Update CLAUDE.md with ASTA notes
- [ ] T021 Test all commands manually
- [ ] T022 Run `go vet` and `go fmt`

## Checkpoint Tests

After each phase, verify:
- `go build -o bp ./cmd/bp` succeeds
- `go test ./...` passes
- New commands show in `./bp asta --help`
