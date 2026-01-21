package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Exit codes for paper commands
const (
	ExitPaperNotFound = 2 // Paper not found
)

func init() {
	rootCmd.AddCommand(paperCmd)

	// paper concepts flags
	paperConceptsCmd.Flags().StringP("type", "t", "", "Filter by relationship type")
	paperCmd.AddCommand(paperConceptsCmd)
}

var paperCmd = &cobra.Command{
	Use:   "paper",
	Short: "Query paper relationships",
	Long:  `Commands for querying paper relationships in the knowledge graph.`,
}

// PaperConceptsResult is the response for the paper concepts command.
type PaperConceptsResult struct {
	PaperID  string                     `json:"paper_id"`
	Concepts []storage.PaperConceptEdge `json:"concepts"`
	Count    int                        `json:"count"`
}

var paperConceptsCmd = &cobra.Command{
	Use:   "concepts <paper-id>",
	Short: "List concepts linked to a paper",
	Long:  `Query all concepts linked to a specific paper, optionally filtered by relationship type.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPaperConcepts,
}

func runPaperConcepts(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	paperID := args[0]
	relType, _ := cmd.Flags().GetString("type")

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Verify paper exists
	ref, err := db.GetByID(paperID)
	if err != nil {
		exitWithError(ExitDataError, "querying paper: %v", err)
	}
	if ref == nil {
		exitWithError(ExitPaperNotFound, "paper %q not found", paperID)
	}

	// Get concepts
	concepts, err := db.GetConceptsByPaper(paperID, relType)
	if err != nil {
		exitWithError(ExitDataError, "querying concepts: %v", err)
	}

	if humanOutput {
		fmt.Printf("Concepts for paper: %s\n", paperID)
		if len(concepts) == 0 {
			fmt.Println("\n(no concepts)")
		} else {
			// Group by relationship type
			byType := make(map[string][]storage.PaperConceptEdge)
			for _, c := range concepts {
				byType[c.RelationshipType] = append(byType[c.RelationshipType], c)
			}

			for relType, typeConcepts := range byType {
				fmt.Printf("\n[%s]\n", relType)
				for _, c := range typeConcepts {
					fmt.Printf("  %s: %s\n", c.ConceptID, c.Summary)
				}
			}
		}
		fmt.Printf("\nTotal: %d concepts\n", len(concepts))
	} else {
		if concepts == nil {
			concepts = []storage.PaperConceptEdge{}
		}
		outputJSON(PaperConceptsResult{
			PaperID:  paperID,
			Concepts: concepts,
			Count:    len(concepts),
		})
	}

	return nil
}
