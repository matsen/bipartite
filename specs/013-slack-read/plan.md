# Implementation Plan: Slack Channel Reading

**Branch**: `013-slack-read` | **Date**: 2026-01-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/013-slack-read/spec.md`
**Implementation Reference**: `/specs/slack-read.md` (detailed design draft)

## Summary

Add read capabilities to the existing Slack integration. Two new commands: `bip slack history <channel>` fetches messages from configured channels where the bot is a member, and `bip slack channels` lists configured channels. Extends the existing `internal/flow/slack.go` which handles posting. Uses SLACK_BOT_TOKEN (different from webhooks) for API access with `channels:history`, `channels:read`, and `users:read` scopes.

## Technical Context

**Language/Version**: Go 1.25.5
**Primary Dependencies**: spf13/cobra (CLI), net/http (Slack API)
**Storage**: N/A (no local storage; user cache in .gitignored file per FR-004)
**Testing**: go test (integration tests with real Slack API, mocked for unit tests)
**Target Platform**: macOS, Linux
**Project Type**: Single CLI project
**Performance Goals**: <5 seconds for typical queries (100 messages or fewer) per SC-001
**Constraints**: Bot must be member of channels to read them; rate limits apply
**Scale/Scope**: Typical usage: occasional queries, not continuous polling

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | PASS | CLI-primary, JSON default output, --human flag for readable format |
| II. Git-Versionable Architecture | PASS | No persistent data; user cache is .gitignored |
| III. Fail-Fast Philosophy | PASS | Clear errors for missing token, inaccessible channels, permission issues |
| IV. Real Testing (Agentic TDD) | PASS | Integration tests against real Slack workspace |
| V. Clean Architecture | PASS | Extends existing slack.go; SlackClient abstraction per design |
| VI. Simplicity | PASS | Minimal implementation; no premature abstraction; no backwards compat needed |

**Pre-design verdict**: All gates pass. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/013-slack-read/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── checklists/
│   └── requirements.md  # Requirements validation checklist
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/bip/
├── slack.go             # NEW: Parent command + history subcommand
└── slack_channels.go    # NEW: Channels subcommand

internal/flow/
├── slack.go             # EXTEND: Add SlackClient, reading functions
└── slack_test.go        # NEW: Unit tests for Slack reading

tests/
└── slack_integration_test.go  # NEW: Integration tests against real Slack
```

**Structure Decision**: Follows existing pattern (see cmd/bip/s2.go, s2_add.go, etc. for parent+subcommand structure). Extends internal/flow/slack.go which already has PostToSlack/SendDigest.

## Complexity Tracking

> No violations - feature follows all Constitution principles.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none) | - | - |

---

## Phase 0: Research

### R1: Slack API Authentication

**Decision**: Use SLACK_BOT_TOKEN environment variable (xoxb-... format)

**Rationale**:
- Bot tokens are the standard for app-to-Slack communication
- Already used for webhooks in the existing codebase (same bot, different auth method)
- Supports required scopes: `channels:history`, `channels:read`, `users:read`

**Alternatives considered**:
- User tokens (xoxp-...): More permissions but security concerns, not appropriate for bots
- Webhooks: Write-only, cannot read messages

### R2: Slack API Methods

**Decision**: Use `conversations.history` for messages, `users.list` for user mapping

**Rationale**:
- `conversations.history` is the current Slack API (replaces deprecated `channels.history`)
- Works for public channels where bot is a member
- Returns messages with timestamps, user IDs, and text

**API Endpoints**:
- GET `https://slack.com/api/conversations.history?channel=CHANNEL_ID&oldest=TIMESTAMP&limit=N`
- GET `https://slack.com/api/users.list` (for user ID → name mapping)

### R3: Channel Configuration

**Decision**: Add `slack.channels` section to sources.json

**Rationale**:
- Follows existing pattern of configuring repos, boards, context in sources.json
- Allows specifying channel name, ID, and purpose
- IDs required because Slack API uses IDs, not names

**Configuration format** (add to sources.json):
```json
{
  "slack": {
    "channels": {
      "fortnight-goals": {"id": "C02GYV1GH2M", "purpose": "goals"},
      "fortnight-feats": {"id": "C02A75JUA", "purpose": "retrospectives"}
    }
  }
}
```

### R4: User Name Caching

**Decision**: Persistent .gitignored file cache in `.bipartite/cache/slack_users.json`

**Rationale**:
- Team membership is stable; users don't change frequently
- Avoids repeated API calls for user lookups
- File-based cache is simple and git-versionable (but gitignored for privacy)
- No TTL needed - refresh manually if stale or on cache miss

**Cache format**:
```json
{
  "U12345": "psathyrella",
  "U67890": "ksung25"
}
```

### R5: Bot Membership Requirement

**Decision**: Fail fast with clear error when channel is inaccessible

**Rationale**:
- Slack API limitation: bot tokens can only read channels where bot is a member
- Cannot detect membership without attempting to read (no separate permission check API)
- Clear error message explaining how to fix (invite bot to channel)

**Error message**: "Cannot read channel 'X': bot is not a member. Invite the bot with /invite @bot-name"

---

## Phase 1: Data Model

### Entities

#### Message

Represents a Slack message returned from history queries.

```go
type Message struct {
    Timestamp string `json:"ts"`        // Slack message timestamp (e.g., "1737990123.000100")
    UserID    string `json:"user_id"`   // Slack user ID (e.g., "U12345")
    UserName  string `json:"user_name"` // Resolved display name
    Date      string `json:"date"`      // Human-readable date (YYYY-MM-DD)
    Text      string `json:"text"`      // Message content
}
```

#### HistoryResponse

Response structure for `bip slack history` JSON output.

```go
type HistoryResponse struct {
    Channel   string    `json:"channel"`    // Channel name
    ChannelID string    `json:"channel_id"` // Channel ID
    Period    Period    `json:"period"`     // Query time range
    Messages  []Message `json:"messages"`   // Retrieved messages
}

type Period struct {
    Start string `json:"start"` // YYYY-MM-DD
    End   string `json:"end"`   // YYYY-MM-DD
}
```

#### ChannelConfig

Configuration entry for a Slack channel.

```go
type ChannelConfig struct {
    ID      string `json:"id"`      // Slack channel ID
    Purpose string `json:"purpose"` // Short description
}
```

#### ChannelsResponse

Response structure for `bip slack channels` JSON output.

```go
type ChannelsResponse struct {
    Channels []ChannelInfo `json:"channels"`
}

type ChannelInfo struct {
    Name    string `json:"name"`
    ID      string `json:"id"`
    Purpose string `json:"purpose"`
}
```

### SlackClient

Client abstraction for reading from Slack.

```go
type SlackClient struct {
    token      string
    httpClient *http.Client
    userCache  map[string]string // Loaded from/saved to cache file
}

func NewSlackClient() (*SlackClient, error)
func (c *SlackClient) GetChannelHistory(channelID string, oldest time.Time, limit int) ([]Message, error)
func (c *SlackClient) GetUsers() (map[string]string, error)
func (c *SlackClient) loadUserCache() error
func (c *SlackClient) saveUserCache() error
```

---

## CLI Commands

### `bip slack`

Parent command grouping Slack operations.

```
Usage: bip slack <command>

Commands:
  history   Fetch message history from a channel
  channels  List configured channels
```

### `bip slack history <channel>`

Fetch recent messages from a Slack channel.

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --days | int | 14 | Number of days to fetch |
| --since | string | - | Start date (YYYY-MM-DD), overrides --days |
| --limit | int | 100 | Maximum messages to return |
| --human | bool | false | Human-readable output |

**Exit codes**:
- 0: Success
- 1: General error (missing config, API failure)
- 2: Channel not found in configuration
- 3: Bot not member of channel

### `bip slack channels`

List configured Slack channels.

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| --human | bool | false | Human-readable output |

---

## Implementation Notes

1. **Extend existing file**: Add to `internal/flow/slack.go` rather than creating new file
2. **Reuse patterns**: Follow `cmd/bip/s2.go` + `s2_add.go` pattern for parent+subcommand
3. **Error messages**: Include remediation steps per FR-009
4. **No external dependencies**: Use standard library net/http for API calls
5. **Date precedence**: When both --days and --since are specified, --since wins

---

## Post-Design Constitution Re-Check

| Principle | Status | Verification |
|-----------|--------|--------------|
| I. Agent-First Design | PASS | JSON default, --human flag, composable with pipes |
| II. Git-Versionable Architecture | PASS | User cache is gitignored; no persistent state |
| III. Fail-Fast Philosophy | PASS | Explicit errors for all failure modes |
| IV. Real Testing (Agentic TDD) | PASS | Tests use real Slack workspace |
| V. Clean Architecture | PASS | Single responsibility; clean abstractions |
| VI. Simplicity | PASS | Minimal code; no unnecessary abstraction |

**Post-design verdict**: All gates pass. Ready for `/speckit.tasks`.
