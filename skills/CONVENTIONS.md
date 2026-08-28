# Formatting conventions for skill files

This governs `skills/*/SKILL.md` only. It does not apply to GitHub issue, PR, or comment bodies — those follow `PROSE-DISCIPLINE.md` instead, which hard-wraps prose into one line per paragraph so GitHub's renderer controls wrapping.

## One sentence per line

Every prose paragraph, list item, and blockquote in a skill file uses semantic line breaks: each sentence starts a new line. Blank lines still separate paragraphs, list items, and headings as usual. Code fences and table rows are exempt — leave their internal line breaks exactly as needed for the code or table to render and function correctly.

**Why**: skill files are source, read as prompt text, edited with line-based diffs, and checked with `grep`. A sentence hard-wrapped across a fixed column defeats all three — a `grep` for a phrase spanning a line break silently misses it, editing one sentence inside a wrapped paragraph touches every line in that paragraph and buries the real diff, and Markdown rendering is unaffected either way (a human or an agent reading the file doesn't care about source line breaks within a paragraph). See PR #191 and issue #192 for the incident that prompted this.

**How to apply**: when writing or editing a skill file, break after each sentence-ending `.`, `!`, or `?`, not at a fixed column. A list item with multiple sentences gets one sentence per line, with continuation lines indented to align under the marker (`- ` = 2 spaces, `1. ` = 3 spaces, etc.) — the same continuation indent already used for wrapped list items today.
