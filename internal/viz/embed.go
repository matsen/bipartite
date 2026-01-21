package viz

import (
	_ "embed"
)

//go:embed assets/cytoscape.min.js
var embeddedCytoscapeJS string

func init() {
	cytoscapeJS = embeddedCytoscapeJS
}
