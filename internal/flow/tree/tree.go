// Package tree provides HTML tree generation from beads hierarchy.
package tree

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/flow"
)

// Patterns for extracting links from descriptions.
var (
	gitHubLinkPattern = regexp.MustCompile(`GitHub:\s*([^#\s]+)#(\d+)`)
	paperLinkPattern  = regexp.MustCompile(`Paper:\s*(github\.com/\S+)`)
	codeLinkPattern   = regexp.MustCompile(`Code:\s*(github\.com/\S+)`)
)

// TreeNode represents a node in the beads tree.
type TreeNode struct {
	Key      string
	Issue    *flow.Bead
	Children map[string]*TreeNode
}

// BuildTree builds a tree structure from flat beads list.
func BuildTree(beads []flow.Bead) *TreeNode {
	root := &TreeNode{Children: make(map[string]*TreeNode)}

	// Sort beads by ID for consistent ordering
	sort.Slice(beads, func(i, j int) bool {
		return beads[i].ID < beads[j].ID
	})

	for _, bead := range beads {
		parts := strings.Split(bead.ID, ".")
		node := root

		for i := range parts {
			key := strings.Join(parts[:i+1], ".")
			if node.Children[key] == nil {
				node.Children[key] = &TreeNode{
					Key:      key,
					Children: make(map[string]*TreeNode),
				}
			}

			// Set issue on leaf node
			if i == len(parts)-1 {
				beadCopy := bead // Make a copy to avoid pointer issues
				node.Children[key].Issue = &beadCopy
			}

			node = node.Children[key]
		}
	}

	return root
}

// IsNew checks if a bead was created after the given time.
func IsNew(bead *flow.Bead, since *time.Time) bool {
	if since == nil || bead == nil {
		return false
	}
	return bead.CreatedAt.After(*since)
}

// ParseGitHubLink extracts a GitHub issue link from description.
func ParseGitHubLink(desc string) string {
	match := gitHubLinkPattern.FindStringSubmatch(desc)
	if match != nil {
		return fmt.Sprintf("https://github.com/%s/issues/%s", match[1], match[2])
	}
	return ""
}

// ParseReferenceLinks extracts Paper and Code links from description.
func ParseReferenceLinks(desc string) [][2]string {
	var links [][2]string

	if match := paperLinkPattern.FindStringSubmatch(desc); match != nil {
		links = append(links, [2]string{"paper", "https://" + match[1]})
	}
	if match := codeLinkPattern.FindStringSubmatch(desc); match != nil {
		links = append(links, [2]string{"code", "https://" + match[1]})
	}

	return links
}

// HTML template parts
const HTMLHeader = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Beads Tree View</title>
<style>
  .help { position: fixed; bottom: 1rem; right: 1rem; font-size: 0.85em; color: #666; }
  .help summary { cursor: pointer; list-style: none; }
  .help summary::-webkit-details-marker { display: none; }
  .help-content { background: white; padding: 0.5rem 1rem; border-radius: 4px; }
  .help-content { box-shadow: 0 2px 8px rgba(0,0,0,0.15); margin-top: 0.5rem; }
  .help kbd { background: #eee; padding: 0.1rem 0.4rem; border-radius: 3px; }
  .help kbd { font-family: monospace; }
  body { font-family: -apple-system, system-ui, sans-serif; margin: 2rem; }
  body { background: #fafafa; }
  details { margin-left: 1.5rem; }
  details[open] > summary { margin-bottom: 0; }
  summary { cursor: pointer; padding: 0.3rem 0.5rem; border-radius: 4px; }
  summary { list-style: none; }
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
`

const HTMLFooter = `
<details class="help">
  <summary>?</summary>
  <div class="help-content">
    <div><kbd>c</kbd> collapse all</div>
    <div><kbd>e</kbd> expand all</div>
  </div>
</details>
<script>
const sel = 'details:not(.help)';
function collapseAll() { document.querySelectorAll(sel).forEach(d => d.open = false); }
function expandAll() { document.querySelectorAll(sel).forEach(d => d.open = true); }
document.addEventListener('keydown', e => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
  if (e.key === 'c') collapseAll();
  if (e.key === 'e') expandAll();
});
</script>
</body>
</html>
`

// RenderNode renders a tree node as HTML.
func RenderNode(node *TreeNode, isRoot bool, since *time.Time) string {
	var sb strings.Builder

	// Sort children by key
	var keys []string
	for k := range node.Children {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		child := node.Children[key]
		issue := child.Issue

		title := key
		desc := ""
		if issue != nil {
			title = issue.Title
			desc = issue.Description
		}

		hasChildren := len(child.Children) > 0
		isNewBead := IsNew(issue, since)

		descAttr := ""
		if desc != "" {
			escapedDesc := strings.ReplaceAll(desc, `"`, "&quot;")
			descAttr = fmt.Sprintf(` title="%s"`, escapedDesc)
		}

		ghLink := ParseGitHubLink(desc)

		if hasChildren {
			// Expandable node
			var cssClasses []string
			if isRoot {
				cssClasses = append(cssClasses, "root")
			}
			if isNewBead {
				cssClasses = append(cssClasses, "new")
			}
			classAttr := ""
			if len(cssClasses) > 0 {
				classAttr = fmt.Sprintf(` class="%s"`, strings.Join(cssClasses, " "))
			}

			idSpan := fmt.Sprintf(`<span class="id">%s</span>`, key)
			titleSpan := fmt.Sprintf(`<span class="title">%s</span>`, escapeHTML(title))
			inner := fmt.Sprintf("<summary%s>%s%s</summary>", descAttr, idSpan, titleSpan)
			inner += RenderNode(child, false, since)

			sb.WriteString(fmt.Sprintf("<details%s open>%s</details>", classAttr, inner))
		} else {
			// Leaf node
			issueType := ""
			if issue != nil {
				issueType = issue.IssueType
			}

			content := fmt.Sprintf(`<span class="id">%s</span><span class="title">%s</span>`, key, escapeHTML(title))

			newClass := ""
			if isNewBead {
				newClass = " new"
			}

			if issueType == "chore" {
				refLinks := ParseReferenceLinks(desc)
				if len(refLinks) > 0 {
					linksHTML := `<span class="links">`
					for _, link := range refLinks {
						linksHTML += fmt.Sprintf(`<a href="%s" target="_blank">[%s]</a>`, link[1], link[0])
					}
					linksHTML += "</span>"
					content += linksHTML
				}
				sb.WriteString(fmt.Sprintf(`<div class="leaf chore%s"%s>%s</div>`, newClass, descAttr, content))
			} else if ghLink != "" {
				content = fmt.Sprintf(`<a href="%s" target="_blank"%s>%s</a>`, ghLink, descAttr, content)
				sb.WriteString(fmt.Sprintf(`<div class="leaf%s">%s</div>`, newClass, content))
			} else if issueType == "task" {
				sb.WriteString(fmt.Sprintf(`<div class="leaf unspecd%s"%s>%s</div>`, newClass, descAttr, content))
			} else {
				sb.WriteString(fmt.Sprintf(`<div class="leaf%s"%s>%s</div>`, newClass, descAttr, content))
			}
		}
	}

	return sb.String()
}

// escapeHTML escapes HTML special characters.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// GenerateHTML generates the full HTML document.
func GenerateHTML(beads []flow.Bead, since *time.Time) string {
	if len(beads) == 0 {
		return "<html><body><p>No beads issues found.</p></body></html>"
	}

	tree := BuildTree(beads)
	content := RenderNode(tree, true, since)
	return HTMLHeader + content + HTMLFooter
}

// ParseSince parses a --since value into a time.
func ParseSince(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	// Try ISO format first
	t, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return &t, nil
	}

	// Try date-only format
	t, err = time.ParseInLocation("2006-01-02", value, time.Local)
	if err == nil {
		return &t, nil
	}

	return nil, fmt.Errorf("invalid date format: %s (use YYYY-MM-DD or ISO format)", value)
}
