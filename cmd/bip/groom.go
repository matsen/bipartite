package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

func init() {
	groomCmd.Flags().Bool("fix", false, "Remove orphaned edges after confirmation")
	rootCmd.AddCommand(groomCmd)
}

var groomCmd = &cobra.Command{
	Use:   "groom",
	Short: "Detect and optionally remove orphaned edges",
	Long:  `Scan for edges that reference papers no longer in the repository and optionally remove them.`,
	RunE:  runGroom,
}

// GroomResult is the response for the groom command.
type GroomResult struct {
	Status        string                  `json:"status"`
	OrphanedEdges []edge.OrphanedEdgeInfo `json:"orphaned_edges"`
	Fixed         bool                    `json:"fixed"`
}

func runGroom(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	fix, _ := cmd.Flags().GetBool("fix")

	// Read all references
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading refs: %v", err)
	}

	// Build set of valid paper IDs
	validIDs := make(map[string]bool)
	for _, ref := range refs {
		validIDs[ref.ID] = true
	}

	// Read all edges
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		exitWithError(ExitDataError, "reading edges: %v", err)
	}

	// Find orphaned edges using shared detection function
	orphaned, validEdges := edge.DetectOrphanedEdges(edges, validIDs)

	// Determine status
	status := "clean"
	if len(orphaned) > 0 {
		status = "orphaned"
	}

	// Handle --fix flag
	fixed := false
	if fix && len(orphaned) > 0 {
		// Write back only valid edges
		if err := storage.WriteAllEdges(edgesPath, validEdges); err != nil {
			exitWithError(ExitDataError, "writing edges: %v", err)
		}

		// Rebuild index
		db := mustOpenDatabase(repoRoot)
		defer db.Close()
		if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
			exitWithError(ExitDataError, "rebuilding index: %v", err)
		}

		fixed = true
		status = "fixed"
	}

	// Ensure orphaned is an empty array for JSON
	if orphaned == nil {
		orphaned = []edge.OrphanedEdgeInfo{}
	}

	// Output results
	if humanOutput {
		if len(orphaned) == 0 {
			fmt.Println("No orphaned edges found")
		} else if fixed {
			fmt.Printf("Removed %d orphaned edges\n", len(orphaned))
		} else {
			fmt.Printf("Found %d orphaned edges:\n", len(orphaned))
			for _, o := range orphaned {
				fmt.Printf("  %s --[%s]--> %s (%s)\n", o.SourceID, o.RelationshipType, o.TargetID, o.Reason)
			}
			fmt.Println("\nRun with --fix to remove orphaned edges")
		}
	} else {
		outputJSON(GroomResult{
			Status:        status,
			OrphanedEdges: orphaned,
			Fixed:         fixed,
		})
	}

	return nil
}
