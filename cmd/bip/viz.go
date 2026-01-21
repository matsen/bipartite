package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/viz"
	"github.com/spf13/cobra"
)

var vizOutput string
var vizLayout string
var vizOffline bool

func init() {
	vizCmd.Flags().StringVarP(&vizOutput, "output", "o", "", "Output file path (default: stdout)")
	vizCmd.Flags().StringVar(&vizLayout, "layout", "force", "Layout algorithm: force, circle, or grid")
	vizCmd.Flags().BoolVar(&vizOffline, "offline", false, "Bundle Cytoscape.js inline for offline use")
	rootCmd.AddCommand(vizCmd)
}

var vizCmd = &cobra.Command{
	Use:   "viz",
	Short: "Generate knowledge graph visualization",
	Long: `Generate an interactive HTML visualization of the knowledge graph.

The visualization shows papers (blue circles) and concepts (orange diamonds)
connected by relationship edges. Edge colors indicate relationship types:
  - green: introduces
  - blue: applies
  - purple: models
  - gray: other

Examples:
  # Generate HTML to stdout
  bip viz > graph.html

  # Generate to file
  bip viz --output graph.html

  # Use circular layout
  bip viz --layout circle --output graph.html

  # Generate offline-capable HTML
  bip viz --offline --output graph.html`,
	RunE: runViz,
}

func runViz(cmd *cobra.Command, args []string) error {
	// Validate layout
	if err := validateLayout(vizLayout); err != nil {
		return err
	}

	// Find repository and open database
	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Extract graph data
	graph, err := viz.ExtractGraphData(db)
	if err != nil {
		return fmt.Errorf("extracting graph data: %w", err)
	}

	// Generate HTML
	opts := viz.HTMLOptions{
		Layout:  vizLayout,
		Offline: vizOffline,
	}
	html, err := viz.GenerateHTML(graph, opts)
	if err != nil {
		return fmt.Errorf("generating HTML: %w", err)
	}

	// Output
	if vizOutput == "" {
		fmt.Print(html)
	} else {
		if err := os.WriteFile(vizOutput, []byte(html), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		if !humanOutput {
			fmt.Printf("{\"output\":\"%s\"}\n", vizOutput)
		} else {
			fmt.Printf("Visualization written to %s\n", vizOutput)
		}
	}

	return nil
}

func validateLayout(layout string) error {
	switch layout {
	case "force", "circle", "grid":
		return nil
	default:
		return fmt.Errorf("invalid layout %q: must be force, circle, or grid", layout)
	}
}
