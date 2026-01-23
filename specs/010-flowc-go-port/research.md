# Research: flowc Existing Behavior

This document captures the exact behavior of the Python flowc implementation, derived from the test suite.

## Test Coverage Summary

| Module | Test File | Test Count | Description |
|--------|-----------|------------|-------------|
| activity | test_activity.py | 20 | Ball-in-court filtering logic |
| config | test_config.py | 12 | sources.json parsing |
| digest | test_digest.py | 9 | Duration parsing, date range formatting |
| issue | test_issue.py | 34 | GitHub ref parsing, output formatting |
| llm | test_llm.py | 31 | LLM prompt building and response parsing |
| slack | test_slack.py | 4 | Webhook URL lookup |

Total: 110 tests

## Ball-in-Court Logic (test_activity.py)

### Truth Table

```
| Item Author | Last Commenter | Result | Reason |
|-------------|----------------|--------|--------|
| other_user  | (none)         | SHOW   | Their item, needs review |
| other_user  | me             | HIDE   | Waiting for their reply |
| other_user  | other_user     | SHOW   | They pinged again |
| me          | (none)         | HIDE   | Waiting for feedback |
| me          | other_user     | SHOW   | They replied |
| me          | me             | HIDE   | Waiting for their reply |
```

### Scenarios from Tests

1. **Someone commented on my old issue** → SHOW (they replied)
2. **I added comment to my own issue** → HIDE (waiting for reply)
3. **Someone opened PR, no comments** → SHOW (needs review)
4. **I opened PR, no comments** → HIDE (waiting for feedback)
5. **I reviewed their PR** → HIDE (waiting for them to address)
6. **They replied to my review** → SHOW (they responded)

### Edge Cases

- Multiple comments: uses LAST commenter only
- Comments on other items: ignored (only comments on THIS item count)
- Item number extracted from comment's `issue_url` field

## Config Parsing (test_config.py)

### RepoEntry Formats

```python
# String format
"matsengrp/repo"

# Object format
{"repo": "matsengrp/repo", "channel": "dasm2"}
```

### normalize_repo_entry()

- String → returns as-is
- Object → returns `entry["repo"]`

### load_repos_by_channel()

- Returns repos where `entry.get("channel") == channel`
- String entries (no channel) are NOT returned
- Unknown channel returns empty list

### list_channels()

- Returns sorted, unique channel names from all entries
- Ignores entries without channel

## Duration Parsing (test_digest.py)

### Valid Formats

```
"2d" → timedelta(days=2)
"7d" → timedelta(days=7)
"12h" → timedelta(hours=12)
"24h" → timedelta(hours=24)
"1w" → timedelta(weeks=1)
"2w" → timedelta(weeks=2)
```

### Invalid Formats

- `"5m"` → ValueError("Unknown duration unit")
- `""` → ValueError("Invalid duration format")
- `"d"` → ValueError("Invalid duration format") (too short)
- `"abcd"` → ValueError("Invalid duration format") (non-numeric)

## GitHub Reference Parsing (test_issue.py)

### Hash Format (`org/repo#N`)

```python
"matsengrp/dasm2-experiments#166" → ("matsengrp/dasm2-experiments", 166, None)
"org/repo-v2#42" → ("org/repo-v2", 42, None)
"org/my#repo#123" → ("org/my#repo", 123, None)  # Uses LAST #
```

### Invalid Hash Formats

```python
"org/repo123" → None  # No #
"repo#123" → None     # No org/
"#123" → None         # No org/
"org/repo#abc" → None # Non-numeric
"org/repo#" → None    # Empty number
"org/repo#0" → None   # Zero
"org/repo#-5" → None  # Negative
```

### URL Format

```python
"https://github.com/org/repo/issues/42" → ("org/repo", 42, "issue")
"https://github.com/org/repo/pull/123" → ("org/repo", 123, "pr")
"github.com/org/repo/issues/10" → ("org/repo", 10, "issue")  # No https
"https://www.github.com/org/repo/pull/5" → ("org/repo", 5, "pr")  # www
"https://github.com/org/repo/issues/99/" → ("org/repo", 99, "issue")  # Trailing slash
```

### Invalid URLs

```python
"https://github.com/org/repo" → None  # No issue/pr path
"https://github.com/org/repo/commits/abc" → None  # Wrong path
"https://gitlab.com/org/repo/issues/1" → None  # Wrong domain
```

## Relative Time Formatting (test_issue.py)

```
< 1 minute → "just now"
1 minute → "1 minute ago"
5 minutes → "5 minutes ago"
1 hour → "1 hour ago"
2 hours → "2 hours ago"
1 day → "1 day ago"
3 days → "3 days ago"
45 days → "1 month ago"
90 days → "3 months ago"
400 days → "1 year ago"
800 days → "2 years ago"
```

## Comment Formatting (test_issue.py)

- Empty list → "(No comments)"
- Single comment: "(1 total)" header
- 10 comments: "(10 total)" header, no truncation message
- 15 comments: "(15 total, showing last 10)" header, shows comments 6-15
- Missing author → "@unknown"
- Missing timestamp → handled gracefully

## PR Files Formatting (test_issue.py)

- Empty list → "(No files changed)"
- Format: `"src/main.py (+10/-5)"`
- Header: "(N files)" or "(N total, showing first 20)"
- Truncated to first 20 files

## PR Reviews Formatting (test_issue.py)

- Empty list → "(No reviews)"
- Format: `"@reviewer: STATE\n  body"`
- Header: "(N total)"
- Long bodies truncated to 200 chars + "..."

## LLM Prompt Building (test_llm.py)

### Summary Prompt Structure

```
REF: matsengrp/repo#123
TYPE: Issue|PR
TITLE: ...
AUTHOR: ...
STATUS: needs_action|waiting
BODY: ... (truncated to 300 chars)
COMMENTS:
  @user1: ... (truncated to 200 chars)
  @user2: ...
  (last 5 comments only)
```

### Response Parsing

- Valid JSON object parsed directly
- Markdown code blocks (`\`\`\`json ... \`\`\``) extracted
- Plain code blocks (`\`\`\` ... \`\`\``) extracted
- Whitespace trimmed
- Invalid JSON → empty dict + warning to stderr
- Preamble text → fails (must be pure JSON)

### Digest Postprocessing

- Adds repo name prefix: `"• repo PR: ..."`
- Adds contributor @mentions: `"... — @alice @bob"`
- Preserves non-bullet lines (headers like `*Merged*`)
- Matches items by number AND repo (handles same number in different repos)

## Slack Webhook (test_slack.py)

- Lookup: `os.environ[f"SLACK_WEBHOOK_{channel.upper()}"]`
- Channel name uppercased: "dasm2" → "SLACK_WEBHOOK_DASM2"
- Returns None if not configured

## File Paths

```
ROOT = Path.cwd()
SOURCES_FILE = ROOT / "sources.json"
BEADS_FILE = ROOT / ".beads" / "issues.jsonl"
STATE_FILE = ROOT / ".last-checkin.json"
CACHE_FILE = ROOT / ".flow-cache.json"
```

## Beads Format

```json
{
  "id": "flow-dasm.2.1.2.1",
  "title": "Curate engelhart-2022 binding dataset",
  "description": "AlphaSeq ~105k scFv antibodies. GitHub: matsengrp/data-central#8",
  "status": "open",
  "priority": 0,
  "issue_type": "task",
  "created_at": "2026-01-12T04:19:00.045127-08:00",
  "created_by": "matsen",
  "updated_at": "2026-01-23T05:02:09.386749-08:00"
}
```

Key fields:
- `priority`: INTEGER (0=P0, not "P0" text)
- `description`: Contains `GitHub: org/repo#N` pattern for linked issues
