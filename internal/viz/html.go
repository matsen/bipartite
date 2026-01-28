package viz

import (
	"bytes"
	"fmt"
	"html/template"
)

// compiledTemplate is parsed at init time to fail fast on template errors.
var compiledTemplate *template.Template

func init() {
	compiledTemplate = template.Must(template.New("viz").Parse(htmlTemplate))
}

// HTMLOptions configures HTML generation.
type HTMLOptions struct {
	Layout  string // "force", "circle", or "grid"
	Offline bool   // Whether to embed Cytoscape.js inline
}

// DefaultOptions returns default HTML generation options.
func DefaultOptions() HTMLOptions {
	return HTMLOptions{
		Layout:  "force",
		Offline: false,
	}
}

// ValidLayouts lists the supported layout algorithm names.
var ValidLayouts = []string{"force", "circle", "grid"}

// GenerateHTML generates a self-contained HTML file for the graph visualization.
func GenerateHTML(graph *GraphData, opts HTMLOptions) (string, error) {
	if graph == nil {
		return "", fmt.Errorf("graph cannot be nil")
	}

	// Validate layout option
	if err := validateLayout(opts.Layout); err != nil {
		return "", err
	}

	if graph.IsEmpty() {
		return generateEmptyHTML(), nil
	}

	graphJSON, err := graph.ToCytoscapeJSON()
	if err != nil {
		return "", err
	}

	layout := layoutToCytoscape(opts.Layout)
	scriptTag := buildScriptTag(opts.Offline)

	data := templateData{
		ScriptTag: template.HTML(scriptTag),
		GraphJSON: template.JS(graphJSON),
		Layout:    layout,
	}

	var buf bytes.Buffer
	if err := compiledTemplate.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// validateLayout checks if the layout option is valid.
func validateLayout(layout string) error {
	switch layout {
	case "", "force", "circle", "grid":
		return nil
	default:
		return fmt.Errorf("invalid layout %q: must be force, circle, or grid", layout)
	}
}

// templateData holds data for the HTML template.
type templateData struct {
	ScriptTag template.HTML
	GraphJSON template.JS
	Layout    string
}

// layoutToCytoscape converts user-friendly layout names to Cytoscape.js layout algorithm names.
func layoutToCytoscape(layout string) string {
	switch layout {
	case "circle":
		return "circle"
	case "grid":
		return "grid"
	case "", "force":
		return "cose"
	default:
		return "cose"
	}
}

// buildScriptTag returns either inline script or CDN reference.
func buildScriptTag(offline bool) string {
	if offline {
		return "<script>" + cytoscapeJS + "</script>"
	}
	return `<script src="https://unpkg.com/cytoscape@3/dist/cytoscape.min.js"></script>`
}

// generateEmptyHTML returns HTML for an empty graph state.
func generateEmptyHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Knowledge Graph - Empty</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      display: flex;
      justify-content: center;
      align-items: center;
      height: 100vh;
      margin: 0;
      background: #f5f5f5;
    }
    .empty-state {
      text-align: center;
      color: #666;
    }
    .empty-state h2 {
      margin-bottom: 0.5em;
      color: #333;
    }
    .empty-state p {
      margin: 0.5em 0;
    }
    .empty-state code {
      background: #e0e0e0;
      padding: 2px 6px;
      border-radius: 3px;
    }
  </style>
</head>
<body>
  <div class="empty-state">
    <h2>No graph data</h2>
    <p>Your library doesn't have any concept edges yet.</p>
    <p>Add concepts using <code>bip concept add</code></p>
    <p>Add edges using <code>bip edge add</code></p>
  </div>
</body>
</html>`
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Knowledge Graph Visualization</title>
  {{.ScriptTag}}
  <style>
    * {
      box-sizing: border-box;
    }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      margin: 0;
      padding: 0;
      background: #f5f5f5;
    }
    #cy {
      width: 100%;
      height: 100vh;
      background: white;
    }
    /* Tooltip container */
    #tooltip {
      position: absolute;
      display: none;
      background: white;
      border: 1px solid #ccc;
      border-radius: 4px;
      padding: 8px 12px;
      box-shadow: 0 2px 8px rgba(0,0,0,0.15);
      max-width: 300px;
      font-size: 13px;
      z-index: 1000;
      pointer-events: none;
    }
    #tooltip .type {
      font-size: 10px;
      text-transform: uppercase;
      color: #888;
      margin-bottom: 4px;
    }
    #tooltip .label {
      font-weight: bold;
      margin-bottom: 4px;
    }
    #tooltip .detail {
      color: #555;
      margin: 2px 0;
    }
    #tooltip .summary {
      font-style: italic;
      color: #666;
      margin-top: 4px;
    }
  </style>
</head>
<body>
  <div id="cy"></div>
  <div id="tooltip"></div>
  <script>
    (function() {
      const graphData = {{.GraphJSON}};
      const layout = "{{.Layout}}";

      // Initialize Cytoscape
      const cy = cytoscape({
        container: document.getElementById('cy'),
        elements: graphData,
        style: [
          // Paper nodes - blue circles
          {
            selector: 'node[type="paper"]',
            style: {
              'background-color': '#4A90D9',
              'label': 'data(label)',
              'color': '#333',
              'font-size': '10px',
              'text-valign': 'bottom',
              'text-margin-y': '5px',
              'width': '30px',
              'height': '30px'
            }
          },
          // Concept nodes - orange diamonds
          {
            selector: 'node[type="concept"]',
            style: {
              'background-color': '#E8923A',
              'shape': 'diamond',
              'label': 'data(label)',
              'color': '#333',
              'font-size': '10px',
              'text-valign': 'bottom',
              'text-margin-y': '5px',
              'width': 'mapData(connectionCount, 0, 10, 25, 50)',
              'height': 'mapData(connectionCount, 0, 10, 25, 50)'
            }
          },
          // Project nodes - green hexagons
          {
            selector: 'node[type="project"]',
            style: {
              'background-color': '#27AE60',
              'shape': 'hexagon',
              'label': 'data(label)',
              'color': '#333',
              'font-size': '11px',
              'font-weight': 'bold',
              'text-valign': 'bottom',
              'text-margin-y': '5px',
              'width': 'mapData(connectionCount, 0, 10, 35, 60)',
              'height': 'mapData(connectionCount, 0, 10, 35, 60)'
            }
          },
          // Repo nodes - small gray squares
          {
            selector: 'node[type="repo"]',
            style: {
              'background-color': '#7F8C8D',
              'shape': 'rectangle',
              'label': 'data(label)',
              'color': '#555',
              'font-size': '8px',
              'text-valign': 'bottom',
              'text-margin-y': '3px',
              'width': '20px',
              'height': '20px'
            }
          },
          // Edge styling by relationship type
          {
            selector: 'edge[relationshipType="introduces"]',
            style: {
              'line-color': '#5CB85C',
              'target-arrow-color': '#5CB85C',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'width': 2
            }
          },
          {
            selector: 'edge[relationshipType="applies"]',
            style: {
              'line-color': '#337AB7',
              'target-arrow-color': '#337AB7',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'width': 2
            }
          },
          {
            selector: 'edge[relationshipType="models"]',
            style: {
              'line-color': '#9B59B6',
              'target-arrow-color': '#9B59B6',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'width': 2
            }
          },
          // Concept-project edges - teal
          {
            selector: 'edge[relationshipType="implemented-in"]',
            style: {
              'line-color': '#1ABC9C',
              'target-arrow-color': '#1ABC9C',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'width': 2
            }
          },
          {
            selector: 'edge[relationshipType="applied-in"]',
            style: {
              'line-color': '#16A085',
              'target-arrow-color': '#16A085',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'width': 2
            }
          },
          {
            selector: 'edge[relationshipType="studied-by"]',
            style: {
              'line-color': '#2ECC71',
              'target-arrow-color': '#2ECC71',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'width': 2
            }
          },
          {
            selector: 'edge',
            style: {
              'line-color': '#95A5A6',
              'target-arrow-color': '#95A5A6',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'width': 2
            }
          },
          // Highlighted state
          {
            selector: 'node.highlighted',
            style: {
              'border-width': 3,
              'border-color': '#ff6b6b'
            }
          },
          {
            selector: 'node.dimmed',
            style: {
              'opacity': 0.3
            }
          },
          {
            selector: 'edge.dimmed',
            style: {
              'opacity': 0.2
            }
          }
        ],
        layout: {
          name: layout,
          animate: false,
          // cose-specific options
          nodeRepulsion: 8000,
          idealEdgeLength: 100,
          edgeElasticity: 100
        }
      });

      // Tooltip handling
      const tooltip = document.getElementById('tooltip');

      function showTooltip(evt, content) {
        tooltip.innerHTML = content;
        tooltip.style.display = 'block';
        const pos = evt.renderedPosition || evt.position;
        tooltip.style.left = (pos.x + 15) + 'px';
        tooltip.style.top = (pos.y + 15) + 'px';
      }

      function hideTooltip() {
        tooltip.style.display = 'none';
      }

      // Build tooltip content for nodes
      function getNodeTooltip(node) {
        const data = node.data();
        let html = '<div class="type">' + data.type + '</div>';
        html += '<div class="label">' + escapeHtml(data.label) + '</div>';

        if (data.type === 'paper') {
          if (data.title) html += '<div class="detail">' + escapeHtml(data.title) + '</div>';
          if (data.authors) html += '<div class="detail">Authors: ' + escapeHtml(data.authors) + '</div>';
          if (data.year) html += '<div class="detail">Year: ' + data.year + '</div>';
        } else if (data.type === 'concept') {
          if (data.description) html += '<div class="detail">' + escapeHtml(data.description) + '</div>';
          if (data.aliases && data.aliases.length > 0) {
            html += '<div class="detail">Aliases: ' + data.aliases.map(escapeHtml).join(', ') + '</div>';
          }
          html += '<div class="detail">Connections: ' + data.connectionCount + '</div>';
        } else if (data.type === 'project') {
          if (data.description) html += '<div class="detail">' + escapeHtml(data.description) + '</div>';
          html += '<div class="detail">Connections: ' + data.connectionCount + '</div>';
        } else if (data.type === 'repo') {
          if (data.description) html += '<div class="detail">' + escapeHtml(data.description) + '</div>';
          if (data.projectId) html += '<div class="detail">Project: ' + escapeHtml(data.projectId) + '</div>';
          if (data.language) html += '<div class="detail">Language: ' + escapeHtml(data.language) + '</div>';
          if (data.topics && data.topics.length > 0) {
            html += '<div class="detail">Topics: ' + data.topics.map(escapeHtml).join(', ') + '</div>';
          }
          if (data.githubUrl) html += '<div class="detail"><a href="' + escapeHtml(data.githubUrl) + '" target="_blank">GitHub</a></div>';
        }

        return html;
      }

      // Build tooltip content for edges
      function getEdgeTooltip(edge) {
        const data = edge.data();
        let html = '<div class="type">' + data.relationshipType + '</div>';
        html += '<div class="label">' + escapeHtml(data.source) + ' → ' + escapeHtml(data.target) + '</div>';
        if (data.summary) html += '<div class="summary">' + escapeHtml(data.summary) + '</div>';
        return html;
      }

      function escapeHtml(str) {
        if (!str) return '';
        return str.replace(/&/g, '&amp;')
                  .replace(/</g, '&lt;')
                  .replace(/>/g, '&gt;')
                  .replace(/"/g, '&quot;');
      }

      // Event handlers
      cy.on('mouseover', 'node', function(evt) {
        showTooltip(evt, getNodeTooltip(evt.target));
      });

      cy.on('mouseout', 'node', function() {
        hideTooltip();
      });

      cy.on('mouseover', 'edge', function(evt) {
        showTooltip(evt, getEdgeTooltip(evt.target));
      });

      cy.on('mouseout', 'edge', function() {
        hideTooltip();
      });

      // Click highlighting
      cy.on('tap', 'node', function(evt) {
        const node = evt.target;

        // Reset all
        cy.elements().removeClass('highlighted dimmed');

        // Get connected elements
        const neighborhood = node.neighborhood().add(node);

        // Highlight connected, dim others
        neighborhood.addClass('highlighted');
        cy.elements().not(neighborhood).addClass('dimmed');
      });

      // Click on empty space to reset
      cy.on('tap', function(evt) {
        if (evt.target === cy) {
          cy.elements().removeClass('highlighted dimmed');
        }
      });
    })();
  </script>
</body>
</html>`

// cytoscapeJS will be populated by embed.go for offline mode
var cytoscapeJS string
