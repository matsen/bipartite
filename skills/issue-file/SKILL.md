---
name: issue-file
description: Create or update GitHub issue from markdown file using --body-file
allowed-tools: Bash, Read
---

# /issue-file

Create or update a GitHub issue using a markdown file as the body.

## Usage

```
/issue-file ISSUE-feature-name.md
```

**CRITICAL: Use the `--body-file` flag to ensure the ENTIRE contents of the file become the issue body verbatim.**

## Workflow

### Step 1: Determine the file path

- If `$ARGUMENTS` is provided, use that as the file path
- Otherwise, check if the file path is clearly determinable from the current conversation context
- If unclear, ask the user which file to use

### Step 2: Check for existing issue context

Search the recent conversation history for:
- Issue numbers mentioned in previous `/work-issue` commands
- Issue numbers referenced in discussion (e.g., "#123", "issue 123")
- Any clear indication we're discussing a specific issue

If found, this is an UPDATE operation. Otherwise, CREATE.

### Step 3: Check for assignee context

Look for explicit mentions like:
- "this is an issue for [username]"
- "assign this to [username]"

**Do NOT auto-assign without explicit context.**

### Step 4: Extract title from file

For CREATE operations, extract the title from the file:
1. Read the first line of the file
2. If it starts with `# ` (markdown h1 heading), strip the `# ` prefix and use as title
3. If no markdown heading, use the filename (without extension) as the title

### Step 5: Execute

**Update** (issue number found in context):
```bash
gh issue edit <number> --body-file <file>
```

**Create** (no issue number):
```bash
gh issue create --title "<extracted title>" --body-file <file>
```

**Create with assignee** (if explicitly mentioned):
```bash
gh issue create --title "<extracted title>" --body-file <file> --assignee <username>
```

**IMPORTANT:**
- NEVER use `--label` flags
- NEVER ask the user for the title — extract it from the file or filename
- Use `--body-file` (or `-F`) flag exclusively for the body
- Only add `--assignee` if explicitly mentioned in conversation context

### Step 6: Report results

After running the command, report:
- Success/failure status
- Issue number and URL
- Whether it was a create or update operation
