# Feature Specification: Slack Channel Reading

**Feature Branch**: `013-slack-read`
**Created**: 2026-01-27
**Status**: Draft
**Input**: User description: "Add Slack channel reading capabilities to bip CLI. Two commands: `bip slack history <channel>` to fetch messages and `bip slack channels` to list configured channels."

## Clarifications

### Session 2026-01-27

- Q: How should user name lookups be cached? → A: Persistent .gitignored file cache (team membership is stable)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fetch Channel History (Priority: P1)

As a user (human or AI agent), I want to fetch recent messages from a Slack channel so I can analyze team activity, track goal progress, or review retrospectives.

**Why this priority**: This is the core capability that enables all downstream use cases—goal tracking, team coordination analysis, and agent-based workflow automation. Without this, the feature has no value.

**Independent Test**: Can be fully tested by running `bip slack history <channel>` against a configured channel and verifying messages are returned with user attribution and timestamps.

**Acceptance Scenarios**:

1. **Given** a configured channel the bot has access to, **When** user runs `bip slack history fortnight-goals`, **Then** messages from the last 14 days are returned in JSON format with channel name, period, and message list.

2. **Given** a configured channel, **When** user runs `bip slack history fortnight-goals --days 7`, **Then** only messages from the last 7 days are returned.

3. **Given** a configured channel, **When** user runs `bip slack history fortnight-goals --since 2025-01-13`, **Then** messages from that date forward are returned.

4. **Given** a configured channel, **When** user runs `bip slack history fortnight-goals --human`, **Then** output is formatted as human-readable markdown with headers per user and date.

5. **Given** a channel the bot has NOT been added to, **When** user runs `bip slack history <channel>`, **Then** a clear error message explains the bot membership requirement.

---

### User Story 2 - List Available Channels (Priority: P2)

As a user, I want to see which Slack channels are configured and accessible so I know what data I can query.

**Why this priority**: Supporting capability that helps users discover available channels. Less critical than actual history fetching but important for discoverability.

**Independent Test**: Can be fully tested by running `bip slack channels` and verifying the configured channels are listed with their IDs and purposes.

**Acceptance Scenarios**:

1. **Given** channels configured in sources.json, **When** user runs `bip slack channels`, **Then** all configured channels are listed in JSON format with name, ID, and purpose.

2. **Given** channels configured, **When** user runs `bip slack channels --human`, **Then** channels are displayed in a human-readable table format.

---

### User Story 3 - Ingest Messages into Store (Priority: P2)

As a user, I want to ingest Slack messages directly into a queryable store so I can archive and search them later without re-fetching from the API.

**Why this priority**: This bridges the slack reading and store subsystems, enabling persistent archival of messages for later querying (e.g., "what did Kevin commit to in Q3?"). Important for the fortnight-goals/feats use case.

**Independent Test**: Can be tested by running `bip slack ingest <channel> --store <name>` and verifying records appear in the store's JSONL file and are queryable via `bip store query`.

**Acceptance Scenarios**:

1. **Given** an existing store with compatible schema, **When** user runs `bip slack ingest fortnight-goals --store slack_msgs --days 30`, **Then** messages from the last 30 days are appended to the store as records.

2. **Given** no existing store, **When** user runs `bip slack ingest fortnight-goals --store slack_msgs`, **Then** the command fails with a clear error suggesting `--create-store`.

3. **Given** no existing store, **When** user runs `bip slack ingest fortnight-goals --store slack_msgs --create-store`, **Then** the store is created with the predefined slack messages schema, and messages are ingested.

4. **Given** a store with some messages already ingested, **When** user runs the ingest command again, **Then** duplicate messages (by id) are skipped and the command reports "Ingested X messages (Y duplicates skipped)".

5. **Given** a successful ingest, **When** user runs `bip store query slack_msgs "SELECT * FROM slack_msgs WHERE user = 'ksung25'"`, **Then** messages from that user are returned.

---

### User Story 4 - Agent-Driven Goal Analysis (Priority: P3)

As an AI agent running a goal-tracking skill, I want to programmatically fetch goals and retrospectives so I can analyze goal quality and match outcomes to stated goals.

**Why this priority**: Key use case that motivates the feature, but depends on P1 being complete. The agent workflow builds on top of the basic history fetching.

**Independent Test**: Can be tested by having an agent script call `bip slack history` for goals and retrospectives channels, then verify the JSON output can be parsed for analysis.

**Acceptance Scenarios**:

1. **Given** fortnight-goals and fortnight-feats channels are configured, **When** an agent fetches history from both, **Then** the JSON output includes all required fields (timestamp, user, date, text) for programmatic analysis.

---

### Edge Cases

- What happens when the channel name doesn't exist in configuration? → Clear error with list of valid channels.
- What happens when slack_bot_token is not configured? → Clear error explaining the required config setting.
- What happens when the token lacks required permissions? → Error explaining which scopes are needed.
- What happens when --days and --since are both specified? → --since takes precedence (document this behavior).
- What happens when --limit is reached? → Messages are truncated with indication that more exist.
- What happens when a channel has no messages in the requested period? → Empty message list returned (not an error).
- What happens when ingesting to a store that doesn't exist? → Clear error suggesting `--create-store` flag.
- What happens when ingesting duplicate messages? → Duplicates are skipped (idempotent); count reported in output.
- What happens when ingesting messages from multiple channels to the same store? → Works correctly; id includes channel name to prevent collisions (`channel:timestamp`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a command to fetch message history from a configured Slack channel.
- **FR-002**: System MUST support filtering messages by time period (number of days or specific start date).
- **FR-003**: System MUST support limiting the number of returned messages (default: 100).
- **FR-004**: System MUST resolve user IDs to display names in the output, using a persistent .gitignored file cache (team membership is stable; no TTL required).
- **FR-005**: System MUST support both JSON (default) and human-readable output formats.
- **FR-006**: System MUST provide a command to list all configured channels.
- **FR-007**: System MUST read channel configuration from the existing sources.json file.
- **FR-008**: System MUST authenticate using the slack_bot_token from global config.
- **FR-009**: System MUST provide clear error messages when channels are inaccessible due to bot membership.
- **FR-010**: System MUST include message timestamp, user name, date, and text in output.
- **FR-011**: System MUST provide a command to ingest messages directly into a store.
- **FR-012**: System MUST support `--create-store` flag to create the target store with a predefined schema if it doesn't exist.
- **FR-013**: System MUST skip duplicate messages during ingest (idempotent operation) based on record id.
- **FR-014**: System MUST use composite id format `channel:timestamp` to ensure uniqueness across channels.
- **FR-015**: System MUST report the number of messages ingested and duplicates skipped.

### Key Entities

- **Channel**: A configured Slack channel with name, ID, and purpose. Channels must have the bot as a member to be readable.
- **Message**: A Slack message with timestamp, author (user ID and resolved name), date, and text content.
- **Period**: The time range for history queries, defined by start date and end date.
- **SlackRecord**: A store record representing a Slack message with fields: id (channel:timestamp), channel, user, date, text.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can fetch channel history and receive results within 5 seconds for typical queries (100 messages or fewer).
- **SC-002**: 100% of error conditions produce actionable error messages that explain how to resolve the issue.
- **SC-003**: JSON output is valid and parseable by standard JSON tools without transformation.
- **SC-004**: Human-readable output can be read and understood without referring to documentation.
- **SC-005**: Agent workflows can successfully fetch and parse goal/retrospective data for automated analysis.
- **SC-006**: Users can ingest messages and query them via `bip store query` within 10 seconds for typical volumes (100 messages).
- **SC-007**: Repeated ingest operations are idempotent—running twice produces the same result as running once.

## Scope Boundaries

### In Scope

- Reading message history from public channels where bot is a member
- Listing configured channels
- Time-based filtering (days, since date)
- Message limit support
- User name resolution
- JSON and human-readable output formats
- Ingesting messages into a store for persistent archival
- Auto-creating stores with predefined slack message schema

### Out of Scope

- Posting messages (already exists via webhooks)
- Reading thread replies
- Real-time message subscriptions
- Private channels or direct messages
- Reactions or message metadata beyond basic fields
- Custom schema for slack message stores (uses predefined schema)

## Assumptions

- The slack_bot_token in global config is already set up by users (same as existing posting functionality).
- Channel configuration already exists in sources.json for posting; this feature reuses that configuration.
- Bot has already been invited to channels of interest (this is a one-time setup step users must perform).
- The Slack API rate limits are sufficient for typical usage patterns (occasional queries, not continuous polling).
