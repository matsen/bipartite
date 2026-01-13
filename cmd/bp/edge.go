package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Exit codes specific to edge commands (per CLI contract)
const (
	ExitEdgeSourceNotFound = 1 // Source paper not found
	ExitEdgeTargetNotFound = 2 // Target paper not found
	ExitEdgeInvalidArgs    = 3 // Invalid arguments
)

func init() {
	rootCmd.AddCommand(edgeCmd)

	// bp edge add flags
	edgeAddCmd.Flags().StringP("source", "s", "", "Source paper ID (required)")
	edgeAddCmd.Flags().StringP("target", "t", "", "Target paper ID (required)")
	edgeAddCmd.Flags().StringP("type", "r", "", "Relationship type (required)")
	edgeAddCmd.Flags().StringP("summary", "m", "", "Relational summary text (required)")
	edgeAddCmd.MarkFlagRequired("source")
	edgeAddCmd.MarkFlagRequired("target")
	edgeAddCmd.MarkFlagRequired("type")
	edgeAddCmd.MarkFlagRequired("summary")
	edgeCmd.AddCommand(edgeAddCmd)

	// bp edge import
	edgeCmd.AddCommand(edgeImportCmd)

	// bp edge list flags
	edgeListCmd.Flags().Bool("incoming", false, "Show edges where paper is target (default: source)")
	edgeListCmd.Flags().Bool("all", false, "Show both incoming and outgoing edges")
	edgeCmd.AddCommand(edgeListCmd)

	// bp edge search flags
	edgeSearchCmd.Flags().StringP("type", "r", "", "Relationship type to filter by (required)")
	edgeSearchCmd.MarkFlagRequired("type")
	edgeCmd.AddCommand(edgeSearchCmd)

	// bp edge export flags
	edgeExportCmd.Flags().StringP("paper", "p", "", "Only export edges involving this paper")
	edgeCmd.AddCommand(edgeExportCmd)
}

var edgeCmd = &cobra.Command{
	Use:   "edge",
	Short: "Manage knowledge graph edges",
	Long:  `Commands for managing directed edges between papers in the knowledge graph.`,
}

// loadPaperIDSet loads all paper IDs from refs and returns them as a set for O(1) lookup.
func loadPaperIDSet(repoRoot string) map[string]bool {
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading refs: %v", err)
	}

	idSet := make(map[string]bool, len(refs))
	for _, ref := range refs {
		idSet[ref.ID] = true
	}
	return idSet
}

// validateEdgePapers checks that source and target papers exist.
// Returns an error if validation fails, nil otherwise.
func validateEdgePapers(e edge.Edge, paperIDs map[string]bool) error {
	if !paperIDs[e.SourceID] {
		return fmt.Errorf("source paper %q not found", e.SourceID)
	}
	if !paperIDs[e.TargetID] {
		return fmt.Errorf("target paper %q not found", e.TargetID)
	}
	return nil
}

// EdgeAddResult is the response for the edge add command.
type EdgeAddResult struct {
	Action string    `json:"action"` // "added" or "updated"
	Edge   edge.Edge `json:"edge"`
}

var edgeAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an edge to the knowledge graph",
	Long:  `Add a directed relationship between two papers.`,
	RunE:  runEdgeAdd,
}

func runEdgeAdd(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	sourceID, _ := cmd.Flags().GetString("source")
	targetID, _ := cmd.Flags().GetString("target")
	relType, _ := cmd.Flags().GetString("type")
	summary, _ := cmd.Flags().GetString("summary")

	// Create edge
	e := edge.Edge{
		SourceID:         sourceID,
		TargetID:         targetID,
		RelationshipType: relType,
		Summary:          summary,
	}

	// Validate edge structure
	if err := e.ValidateForCreate(); err != nil {
		exitWithError(ExitEdgeInvalidArgs, "invalid edge: %v", err)
	}

	// Load paper IDs and validate endpoints using shared helper
	paperIDs := loadPaperIDSet(repoRoot)
	if err := validateEdgePapers(e, paperIDs); err != nil {
		if strings.Contains(err.Error(), "source paper") {
			exitWithError(ExitEdgeSourceNotFound, "%v", err)
		} else {
			exitWithError(ExitEdgeTargetNotFound, "%v", err)
		}
	}

	// Load existing edges
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		exitWithError(ExitDataError, "reading edges: %v", err)
	}

	// Upsert edge
	edges, updated := storage.UpsertEdge(edges, e)
	e = edges[len(edges)-1] // Get the edge with CreatedAt set

	if updated {
		// Find the updated edge
		idx, _ := storage.FindEdgeByKey(edges, e.Key())
		e = edges[idx]
	}

	// Write back to JSONL
	if err := storage.WriteAllEdges(edgesPath, edges); err != nil {
		exitWithError(ExitDataError, "writing edges: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if err := db.InsertEdge(e); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	// Output results
	action := "added"
	if updated {
		action = "updated"
	}

	if humanOutput {
		if action == "added" {
			fmt.Printf("Added edge: %s --[%s]--> %s\n", sourceID, relType, targetID)
		} else {
			fmt.Printf("Updated edge: %s --[%s]--> %s\n", sourceID, relType, targetID)
		}
	} else {
		outputJSON(EdgeAddResult{
			Action: action,
			Edge:   e,
		})
	}

	return nil
}

// EdgeImportResult is the response for the edge import command.
type EdgeImportResult struct {
	Added   int               `json:"added"`
	Updated int               `json:"updated"`
	Skipped int               `json:"skipped"`
	Errors  []EdgeImportError `json:"errors"`
}

// EdgeImportError represents an error during import.
type EdgeImportError struct {
	Line  int    `json:"line"`
	Error string `json:"error"`
}

var edgeImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import edges from a JSONL file",
	Long:  `Bulk import edges from a JSONL file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEdgeImport,
}

func runEdgeImport(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	importPath := args[0]

	// Check file exists
	f, err := os.Open(importPath)
	if err != nil {
		exitWithError(ExitEdgeInvalidArgs, "file not found: %s", importPath)
	}
	defer f.Close()

	// Load paper IDs for validation
	paperIDs := loadPaperIDSet(repoRoot)

	// Load existing edges
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		exitWithError(ExitDataError, "reading edges: %v", err)
	}

	// Process import file
	result := EdgeImportResult{Errors: []EdgeImportError{}}
	edges, err = processImportFile(f, edges, paperIDs, &result)
	if err != nil {
		exitWithError(ExitDataError, "%v", err)
	}

	// Check if all edges were invalid
	if result.Added == 0 && result.Updated == 0 && result.Skipped > 0 {
		outputImportFailure(result)
		os.Exit(ExitEdgeInvalidArgs)
	}

	// Write back to JSONL
	if err := storage.WriteAllEdges(edgesPath, edges); err != nil {
		exitWithError(ExitDataError, "writing edges: %v", err)
	}

	// Rebuild SQLite index for imported edges
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	// Output results
	if humanOutput {
		total := result.Added + result.Updated
		fmt.Printf("Imported %d edges (%d updated, %d skipped)\n", total, result.Updated, result.Skipped)
		if len(result.Errors) > 0 {
			fmt.Println("Skipped:")
			for _, e := range result.Errors {
				fmt.Printf("  Line %d: %s\n", e.Line, e.Error)
			}
		}
	} else {
		outputJSON(result)
	}

	return nil
}

// processImportFile reads edges from a file and validates/upserts them.
// Returns the updated edges slice and any file reading error.
func processImportFile(f *os.File, edges []edge.Edge, paperIDs map[string]bool, result *EdgeImportResult) ([]edge.Edge, error) {
	scanner := bufio.NewScanner(f)
	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var e edge.Edge
		if err := json.Unmarshal(line, &e); err != nil {
			result.Errors = append(result.Errors, EdgeImportError{
				Line:  lineNum,
				Error: fmt.Sprintf("invalid JSON: %v", err),
			})
			result.Skipped++
			continue
		}

		// Validate edge structure
		if err := e.ValidateForCreate(); err != nil {
			result.Errors = append(result.Errors, EdgeImportError{
				Line:  lineNum,
				Error: err.Error(),
			})
			result.Skipped++
			continue
		}

		// Validate paper endpoints using shared helper
		if err := validateEdgePapers(e, paperIDs); err != nil {
			result.Errors = append(result.Errors, EdgeImportError{
				Line:  lineNum,
				Error: err.Error(),
			})
			result.Skipped++
			continue
		}

		// Upsert edge
		var updated bool
		edges, updated = storage.UpsertEdge(edges, e)
		if updated {
			result.Updated++
		} else {
			result.Added++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading import file: %w", err)
	}

	return edges, nil
}

// outputImportFailure outputs error information when all edges fail validation.
func outputImportFailure(result EdgeImportResult) {
	if humanOutput {
		fmt.Printf("Import failed: all %d edges were invalid\n", result.Skipped)
		for _, e := range result.Errors {
			fmt.Printf("  Line %d: %s\n", e.Line, e.Error)
		}
	} else {
		outputJSON(result)
	}
}

// EdgeListResult is the response for the edge list command.
type EdgeListResult struct {
	PaperID  string      `json:"paper_id"`
	Outgoing []edge.Edge `json:"outgoing,omitempty"`
	Incoming []edge.Edge `json:"incoming,omitempty"`
}

var edgeListCmd = &cobra.Command{
	Use:   "list <paper-id>",
	Short: "List edges for a paper",
	Long:  `List all edges connected to a specific paper.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEdgeList,
}

func runEdgeList(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	paperID := args[0]

	incoming, _ := cmd.Flags().GetBool("incoming")
	all, _ := cmd.Flags().GetBool("all")

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	var result EdgeListResult
	result.PaperID = paperID

	// Get outgoing edges (paper is source)
	if !incoming || all {
		outgoing, err := db.GetEdgesBySource(paperID)
		if err != nil {
			exitWithError(ExitDataError, "querying outgoing edges: %v", err)
		}
		result.Outgoing = outgoing
	}

	// Get incoming edges (paper is target)
	if incoming || all {
		incomingEdges, err := db.GetEdgesByTarget(paperID)
		if err != nil {
			exitWithError(ExitDataError, "querying incoming edges: %v", err)
		}
		result.Incoming = incomingEdges
	}

	// Output results
	if humanOutput {
		if len(result.Outgoing) == 0 && len(result.Incoming) == 0 {
			fmt.Printf("No edges found for %s\n", paperID)
			return nil
		}

		if len(result.Outgoing) > 0 {
			fmt.Printf("Outgoing edges from %s:\n", paperID)
			for _, e := range result.Outgoing {
				fmt.Printf("  --[%s]--> %s\n", e.RelationshipType, e.TargetID)
				fmt.Printf("    %q\n", e.Summary)
			}
		}

		if len(result.Incoming) > 0 {
			if len(result.Outgoing) > 0 {
				fmt.Println()
			}
			fmt.Printf("Incoming edges to %s:\n", paperID)
			for _, e := range result.Incoming {
				fmt.Printf("  %s --[%s]-->\n", e.SourceID, e.RelationshipType)
				fmt.Printf("    %q\n", e.Summary)
			}
		}
	} else {
		// Ensure arrays are not null
		if result.Outgoing == nil {
			result.Outgoing = []edge.Edge{}
		}
		if result.Incoming == nil {
			result.Incoming = []edge.Edge{}
		}
		outputJSON(result)
	}

	return nil
}

// EdgeSearchResult is the response for the edge search command.
type EdgeSearchResult struct {
	RelationshipType string      `json:"relationship_type"`
	Edges            []edge.Edge `json:"edges"`
}

var edgeSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search edges by relationship type",
	Long:  `Search for edges with a specific relationship type.`,
	RunE:  runEdgeSearch,
}

func runEdgeSearch(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	relType, _ := cmd.Flags().GetString("type")

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	edges, err := db.GetEdgesByType(relType)
	if err != nil {
		exitWithError(ExitDataError, "searching edges: %v", err)
	}

	// Output results
	if humanOutput {
		if len(edges) == 0 {
			fmt.Printf("No edges found with type %q\n", relType)
			return nil
		}

		fmt.Printf("Edges with type %q:\n", relType)
		for _, e := range edges {
			fmt.Printf("  %s --[%s]--> %s\n", e.SourceID, e.RelationshipType, e.TargetID)
			fmt.Printf("    %q\n", e.Summary)
		}
	} else {
		if edges == nil {
			edges = []edge.Edge{}
		}
		outputJSON(EdgeSearchResult{
			RelationshipType: relType,
			Edges:            edges,
		})
	}

	return nil
}

var edgeExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export edges to JSONL format",
	Long:  `Export edges to JSONL format, writing to stdout.`,
	RunE:  runEdgeExport,
}

func runEdgeExport(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	paperID, _ := cmd.Flags().GetString("paper")

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	var edges []edge.Edge
	var err error

	if paperID != "" {
		edges, err = db.GetEdgesByPaper(paperID)
		if err != nil {
			exitWithError(ExitDataError, "querying edges: %v", err)
		}
	} else {
		edges, err = db.GetAllEdges()
		if err != nil {
			exitWithError(ExitDataError, "querying edges: %v", err)
		}
	}

	// Output as JSONL (one JSON object per line)
	for _, e := range edges {
		outputJSONCompact(e)
	}

	return nil
}
