# /tree

Generate interactive HTML tree view of beads issues.

## Instructions

```bash
flowc tree --open                    # Generate and open in browser
flowc tree -o tree.html              # Save to file
flowc tree --since 2024-01-15        # Highlight items created after date
flowc tree --since 2024-01-15 --open # Both
```

## Options

- `--open` — Open in browser after generating
- `-o FILE`, `--output FILE` — Save to file (default: stdout)
- `--since DATE` — Highlight beads created after this date (YYYY-MM-DD or ISO)

## Keyboard shortcuts (in browser)

- `c` — Collapse all nodes
- `e` — Expand all nodes

## Output

Generates an interactive HTML document with:
- Hierarchical tree based on bead IDs (e.g., `proj.feat.task`)
- Expandable/collapsible nodes
- Links to GitHub issues where available
- Visual distinction for chores, unspec'd tasks, and new items
