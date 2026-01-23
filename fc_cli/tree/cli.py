"""CLI handler for tree subcommand."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import tempfile
import webbrowser
from datetime import datetime
from pathlib import Path

# Beads directory is .beads/ in the current working directory
BEADS_DIR = Path.cwd() / ".beads"
JSONL_FILE = BEADS_DIR / "issues.jsonl"

HTML_HEADER = """<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Beads Tree View</title>
<style>
  .help { position: fixed; bottom: 1rem; right: 1rem; font-size: 0.85em; color: #666; }
  .help summary { cursor: pointer; list-style: none; }
  .help summary::-webkit-details-marker { display: none; }
  .help-content { background: white; padding: 0.5rem 1rem; border-radius: 4px; box-shadow: 0 2px 8px rgba(0,0,0,0.15); margin-top: 0.5rem; }
  .help kbd { background: #eee; padding: 0.1rem 0.4rem; border-radius: 3px; font-family: monospace; }
  body { font-family: -apple-system, system-ui, sans-serif; margin: 2rem; background: #fafafa; }
  details { margin-left: 1.5rem; }
  details[open] > summary { margin-bottom: 0; }
  summary { cursor: pointer; padding: 0.3rem 0.5rem; border-radius: 4px; list-style: none; }
  summary:hover { background: #e8e8e8; }
  summary::-webkit-details-marker { display: none; }
  summary::before { content: "▶ "; font-size: 0.7em; color: #666; }
  details[open] > summary::before { content: "▼ "; }
  .leaf { margin-left: 1.5rem; padding: 0.3rem 0.5rem; }
  .leaf::before { content: "• "; font-size: 0.7em; color: #666; }
  .id { color: #666; font-size: 0.85em; margin-right: 0.5rem; }
  .title { font-weight: 500; }
  a { color: #0066cc; }
  a:hover { text-decoration: underline; }
  a .title { color: #0066cc; }
  .root { margin-left: 0; }
  .chore { color: #7570b3; }
  .chore::before { content: "↗ "; color: #7570b3; }
  .chore a { color: #7570b3; }
  .unspecd { color: #a6761d; }
  .unspecd::before { content: "○ "; color: #a6761d; }
  .links { font-size: 0.85em; margin-left: 0.5rem; }
  .links a { margin-right: 0.75rem; text-decoration: underline; }
  .new .title { color: #d95f02; }
  .new.chore .title { color: #d95f02; }
</style>
</head>
<body>
"""

HTML_FOOTER = """
<details class="help">
  <summary>?</summary>
  <div class="help-content">
    <div><kbd>c</kbd> collapse all</div>
    <div><kbd>e</kbd> expand all</div>
  </div>
</details>
<script>
function collapseAll() { document.querySelectorAll('details:not(.help)').forEach(d => d.open = false); }
function expandAll() { document.querySelectorAll('details:not(.help)').forEach(d => d.open = true); }
document.addEventListener('keydown', e => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
  if (e.key === 'c') collapseAll();
  if (e.key === 'e') expandAll();
});
</script>
</body>
</html>
"""


def load_issues() -> dict:
    """Load issues from JSONL file."""
    issues = {}
    if not JSONL_FILE.exists():
        return issues
    with open(JSONL_FILE) as f:
        for line in f:
            if line.strip():
                issue = json.loads(line)
                issues[issue["id"]] = issue
    return issues


def build_tree(issues: dict) -> dict:
    """Build tree structure from flat issues dict."""
    tree = {}
    for id_, issue in sorted(issues.items()):
        parts = id_.split(".")
        node = tree
        for i, part in enumerate(parts):
            key = ".".join(parts[: i + 1])
            if key not in node:
                node[key] = {"_children": {}}
            if i == len(parts) - 1:
                node[key]["_issue"] = issue
            node = node[key]["_children"]
    return tree


def parse_github_link(desc: str) -> str | None:
    """Extract GitHub issue link from description like 'GitHub: org/repo#123'."""
    match = re.search(r"GitHub:\s*([^#\s]+)#(\d+)", desc)
    if match:
        repo, num = match.groups()
        return f"https://github.com/{repo}/issues/{num}"
    return None


def parse_reference_links(desc: str) -> list[tuple[str, str]]:
    """Extract Paper/Code links from description."""
    links = []
    paper_match = re.search(r"Paper:\s*(github\.com/\S+)", desc)
    if paper_match:
        links.append(("paper", f"https://{paper_match.group(1)}"))
    code_match = re.search(r"Code:\s*(github\.com/\S+)", desc)
    if code_match:
        links.append(("code", f"https://{code_match.group(1)}"))
    return links


def is_new(issue: dict, since: datetime | None) -> bool:
    """Check if issue was created after the since timestamp."""
    if not since:
        return False
    created = issue.get("created_at", "")
    if not created:
        return False
    try:
        created_dt = datetime.fromisoformat(created)
        return created_dt > since
    except ValueError:
        return False


def render_node(node: dict, is_root: bool = False, since: datetime | None = None) -> str:
    """Render a tree node as HTML."""
    html = []
    for key in sorted(node.keys()):
        if key.startswith("_"):
            continue
        child = node[key]
        issue = child.get("_issue", {})
        children = child.get("_children", {})

        title = issue.get("title", key)
        desc = issue.get("description", "")
        has_children = bool([k for k in children if not k.startswith("_")])
        new = is_new(issue, since)

        desc_attr = f' title="{desc.replace(chr(34), "&quot;")}"' if desc else ""
        gh_link = parse_github_link(desc)

        if has_children:
            css_classes = ["root"] if is_root else []
            if new:
                css_classes.append("new")
            class_attr = f' class="{" ".join(css_classes)}"' if css_classes else ""
            inner = f'<summary{desc_attr}><span class="id">{key}</span><span class="title">{title}</span></summary>'
            inner += render_node(children, since=since)
            html.append(f"<details{class_attr} open>{inner}</details>")
        else:
            itype = issue.get("issue_type", "")
            content = f'<span class="id">{key}</span><span class="title">{title}</span>'

            if itype == "chore":
                ref_links = parse_reference_links(desc)
                if ref_links:
                    links_html = '<span class="links">'
                    for label, url in ref_links:
                        links_html += f'<a href="{url}" target="_blank">[{label}]</a>'
                    links_html += "</span>"
                    content += links_html
                new_class = " new" if new else ""
                html.append(
                    f'<div class="leaf chore{new_class}"{desc_attr}>{content}</div>'
                )
            elif gh_link:
                content = (
                    f'<a href="{gh_link}" target="_blank"{desc_attr}>{content}</a>'
                )
                new_class = " new" if new else ""
                html.append(f'<div class="leaf{new_class}">{content}</div>')
            elif itype == "task":
                new_class = " new" if new else ""
                html.append(
                    f'<div class="leaf unspecd{new_class}"{desc_attr}>{content}</div>'
                )
            else:
                new_class = " new" if new else ""
                html.append(f'<div class="leaf{new_class}"{desc_attr}>{content}</div>')

    return "\n".join(html)


def parse_since(value: str) -> datetime | None:
    """Parse --since value into a timezone-aware datetime object."""
    if not value:
        return None
    try:
        dt = datetime.fromisoformat(value)
        if dt.tzinfo is None:
            dt = dt.astimezone()
        return dt
    except ValueError:
        pass
    try:
        dt = datetime.strptime(value, "%Y-%m-%d")
        return dt.astimezone()
    except ValueError:
        pass
    raise argparse.ArgumentTypeError(
        f"Invalid date format: {value}. Use YYYY-MM-DD or ISO format."
    )


def generate_html(since: datetime | None = None) -> str:
    """Generate the full HTML document."""
    issues = load_issues()
    if not issues:
        return "<html><body><p>No beads issues found.</p></body></html>"

    tree = build_tree(issues)
    content = render_node(tree, is_root=True, since=since)
    return HTML_HEADER + content + HTML_FOOTER


def run_tree(args: argparse.Namespace) -> int:
    """Run the tree command."""
    since = parse_since(args.since) if args.since else None
    html = generate_html(since)

    if args.output:
        output_path = Path(args.output)
        output_path.write_text(html)
        print(f"Written to {output_path}")
        if args.open:
            webbrowser.open(f"file://{output_path.absolute()}")
    elif args.open:
        # Write to temp file and open
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".html", delete=False
        ) as f:
            f.write(html)
            temp_path = f.name
        webbrowser.open(f"file://{temp_path}")
        print(f"Opened {temp_path}")
    else:
        # Print to stdout
        print(html)

    return 0
