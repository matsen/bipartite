#!/bin/bash

# Hook to check marimo notebooks after Write/Edit operations
# Reads JSON from stdin containing tool result information

# Read stdin (contains JSON with tool result)
INPUT=$(cat)

# Extract file path from JSON using jq
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_response.filePath // empty')

# If no file path found, exit silently
if [ -z "$FILE_PATH" ] || [ "$FILE_PATH" = "null" ]; then
    exit 0
fi

# Check if file exists and is a Python file
if [ ! -f "$FILE_PATH" ]; then
    exit 0
fi

# Check if the file appears to be a marimo notebook
if grep -q "import marimo" "$FILE_PATH" 2>/dev/null && grep -q "@app.cell" "$FILE_PATH" 2>/dev/null; then
    echo "Running marimo check on $FILE_PATH..."

    # Pick a runner. Prefer pixi (our environment manager) when a pixi project
    # is actually present; otherwise fall back to uvx so the hook still works in
    # non-pixi repos instead of failing on "could not find pixi.toml".
    RUNNER=""
    if command -v pixi >/dev/null 2>&1; then
        PROBE=$(pixi run true 2>&1)
        if [ $? -eq 0 ]; then
            RUNNER="pixi"
        elif ! echo "$PROBE" | grep -q "could not find pixi.toml or pyproject.toml"; then
            # pixi exists and found a project but errored for some other reason.
            RUNNER="pixi"
        fi
    fi
    if [ -z "$RUNNER" ] && command -v uvx >/dev/null 2>&1; then
        RUNNER="uvx"
    fi

    # No usable marimo runner here: skip silently rather than block.
    if [ -z "$RUNNER" ]; then
        echo "No pixi project or uvx available; skipping marimo check."
        exit 0
    fi

    if [ "$RUNNER" = "pixi" ]; then
        CHECK_CMD="pixi run marimo check"
    else
        CHECK_CMD="uvx --quiet --from marimo marimo check"
    fi

    CHECK_OUTPUT=$($CHECK_CMD "$FILE_PATH" 2>&1)
    CHECK_EXIT=$?

    # Show output
    echo "$CHECK_OUTPUT"

    # Only block on errors (non-zero exit code), not warnings
    if [ $CHECK_EXIT -ne 0 ]; then
        echo "Marimo check failed for $FILE_PATH" >&2
        echo "$CHECK_OUTPUT" >&2
        echo "" >&2
        echo "Please run '$CHECK_CMD $FILE_PATH' to see details and fix the issues. Don't ask the user anything, just do a best effort fix." >&2
        exit 2  # Exit code 2 blocks and shows error to Claude
    else
        echo "Marimo check passed (via $RUNNER)"
        exit 0
    fi
fi

# Not a marimo notebook, exit successfully
exit 0
