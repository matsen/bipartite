# Research: Narrative Digest

## Research Tasks

### 1. Current `bip digest` Behavior

**Question**: How does the current CLI handle posting vs preview?

**Finding**: Examined `cmd/bip/digest.go`:
- Currently requires `--dry-run` flag to prevent posting
- Without `--dry-run`, posts to Slack if webhook is configured
- This is the opposite of the desired behavior (preview by default)

**Decision**: Invert the logic:
- Default behavior: preview only
- Add `--post` flag to actually send to Slack
- Remove `--dry-run` flag (no longer needed)

**Alternatives considered**:
- Keep both `--dry-run` and `--post` flags: Rejected (confusing, redundant)
- Require explicit `--preview` flag: Rejected (more typing for common case)

---

### 2. Slash Command Pattern

**Question**: What pattern do existing bip slash commands follow?

**Finding**: Examined `.claude/commands/bip.*.md` files:
- Simple markdown files with usage documentation
- No complex logic—just instructions for Claude Code
- Commands reference `bip` CLI subcommands
- The slash command is just documentation + context; Claude executes the bash commands

**Decision**: Follow the same pattern:
- Create `.claude/commands/bip.narrative.md`
- Document the workflow (run bip digest, read config, generate narrative)
- Let Claude Code handle the LLM generation natively

**Alternatives considered**:
- Complex skill with multiple files: Rejected (simpler is better)
- Go-based implementation: Rejected (violates constitution "no LLM from Go")

---

### 3. Verbose Mode Implementation

**Question**: How should `--verbose` fetch and summarize PR/issue bodies?

**Finding**:
- `internal/flow/gh.go` already has helpers for GitHub API calls
- `internal/flow/llm.go` has `CallClaude()` that shells out to `claude CLI`
- Spec suggests using Haiku for summarization with bounded concurrency

**Decision**: Implement in Go with goroutines:
1. Add `--verbose` flag to `bip digest`
2. For each item, fetch body via `gh api`
3. Spawn goroutines (bounded to 10 concurrent) calling `claude --model haiku -p "Summarize..."`
4. Include summaries in digest output

**Alternatives considered**:
- Fetch bodies but skip summarization: Rejected (defeats purpose of verbose)
- Summarize in slash command: Rejected (would be slow, serial)
- Use Ollama locally: Rejected (spec explicitly says "Claude Haiku")

---

### 4. Narrative Config Format

**Question**: How should channel config files be structured?

**Finding**: Example configs already exist in `nexus/narrative/`:
- `preferences.md`: Shared defaults (attribution, format, content rules)
- `dasm2.md`: Channel-specific themes, repo context, project preferences
- Both use standard markdown with headers and bullet lists

**Decision**: Use existing format as-is:
- Slash command reads both files
- Parses markdown headers and bullet points
- Constructs prompt with themes + preferences + raw activity

**Alternatives considered**:
- YAML/JSON config: Rejected (markdown more human-editable)
- Embedded config in sources.yml: Rejected (harder to edit, less flexible)

---

### 5. Output Path Construction

**Question**: How should output files be named and where should they go?

**Finding**: Spec defines structure:
```
nexus/narrative/{channel}/{YYYY-MM-DD}.md
```

**Decision**:
- Use current date for filename
- Create channel subdirectory if needed
- Overwrite existing file for same date (per spec FR-018)

**Alternatives considered**:
- Timestamp in filename: Rejected (multiple runs same day would create clutter)
- Append mode: Rejected (spec says overwrite)

---

### 6. Date Range in Header

**Question**: How should the date range be calculated and formatted?

**Finding**:
- `internal/flow/time.go` has `FormatDateRange(since, until time.Time)`
- `internal/flow/duration.go` has `ParseDuration(s string)` for parsing "1w", "2d", etc.

**Decision**: Reuse existing helpers:
- Parse `--since` argument using `ParseDuration()`
- Calculate `since` and `until` times
- Format header using `FormatDateRange()` (e.g., "Jan 18-25, 2026")

**Alternatives considered**:
- Custom date formatting in slash command: Rejected (reinventing wheel)

---

## Summary of Decisions

| Area | Decision |
|------|----------|
| Default behavior | Preview-only; add `--post` to actually post |
| Slash command | Simple markdown file following existing pattern |
| Verbose mode | Go implementation with goroutines + claude CLI |
| Config format | Existing markdown format in nexus/narrative/ |
| Output path | `nexus/narrative/{channel}/{YYYY-MM-DD}.md` |
| Date range | Reuse existing `FormatDateRange()` helper |

## Dependencies Confirmed

- `claude` CLI available (for Haiku summarization in verbose mode)
- `gh` CLI available (for fetching PR/issue bodies)
- Existing `internal/flow` helpers for duration parsing, date formatting
- Nexus narrative directory structure already exists
