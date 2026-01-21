package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/concept"
	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Exit codes for concept commands (per CLI contract)
const (
	ExitConceptNotFound   = 2 // Concept not found
	ExitConceptValidation = 3 // Validation error (invalid ID, duplicate, has edges)
)

func init() {
	rootCmd.AddCommand(conceptCmd)

	// concept add flags
	conceptAddCmd.Flags().StringP("name", "n", "", "Display name (required)")
	conceptAddCmd.Flags().StringP("aliases", "a", "", "Comma-separated aliases")
	conceptAddCmd.Flags().StringP("description", "d", "", "Description text")
	conceptAddCmd.MarkFlagRequired("name")
	conceptCmd.AddCommand(conceptAddCmd)

	// concept get - no extra flags
	conceptCmd.AddCommand(conceptGetCmd)

	// concept list - no extra flags
	conceptCmd.AddCommand(conceptListCmd)

	// concept update flags
	conceptUpdateCmd.Flags().StringP("name", "n", "", "New display name")
	conceptUpdateCmd.Flags().StringP("aliases", "a", "", "New comma-separated aliases (replaces existing)")
	conceptUpdateCmd.Flags().StringP("description", "d", "", "New description")
	conceptCmd.AddCommand(conceptUpdateCmd)

	// concept delete flags
	conceptDeleteCmd.Flags().BoolP("force", "f", false, "Delete even if edges exist")
	conceptCmd.AddCommand(conceptDeleteCmd)

	// concept papers flags
	conceptPapersCmd.Flags().StringP("type", "t", "", "Filter by relationship type")
	conceptCmd.AddCommand(conceptPapersCmd)

	// concept merge - no extra flags
	conceptCmd.AddCommand(conceptMergeCmd)
}

var conceptCmd = &cobra.Command{
	Use:   "concept",
	Short: "Manage concept nodes",
	Long:  `Commands for managing concept nodes in the knowledge graph.`,
}

// ConceptAddResult is the response for the concept add command.
type ConceptAddResult struct {
	Status  string          `json:"status"`
	Concept concept.Concept `json:"concept"`
}

var conceptAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a new concept",
	Long:  `Add a new concept node to the knowledge graph.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConceptAdd,
}

func runConceptAdd(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	conceptID := args[0]

	name, _ := cmd.Flags().GetString("name")
	aliasesStr, _ := cmd.Flags().GetString("aliases")
	description, _ := cmd.Flags().GetString("description")

	// Parse aliases
	var aliases []string
	if aliasesStr != "" {
		aliases = strings.Split(aliasesStr, ",")
		for i := range aliases {
			aliases[i] = strings.TrimSpace(aliases[i])
		}
	}

	// Create concept
	c := concept.Concept{
		ID:          conceptID,
		Name:        name,
		Aliases:     aliases,
		Description: description,
	}

	// Validate
	if err := c.ValidateForCreate(); err != nil {
		exitWithError(ExitConceptValidation, "invalid concept: %v", err)
	}

	// Load existing concepts
	conceptsPath := config.ConceptsPath(repoRoot)
	concepts, err := storage.ReadAllConcepts(conceptsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading concepts: %v", err)
	}

	// Check for duplicate
	if _, found := storage.FindConceptByID(concepts, conceptID); found {
		exitWithError(ExitConceptValidation, "concept with id %q already exists", conceptID)
	}

	// Append to JSONL
	if err := storage.AppendConcept(conceptsPath, c); err != nil {
		exitWithError(ExitDataError, "writing concept: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Created concept: %s\n", conceptID)
		fmt.Printf("  Name: %s\n", name)
		if len(aliases) > 0 {
			fmt.Printf("  Aliases: %s\n", strings.Join(aliases, ", "))
		}
		if description != "" {
			fmt.Printf("  Description: %s\n", description)
		}
	} else {
		outputJSON(ConceptAddResult{
			Status:  "created",
			Concept: c,
		})
	}

	return nil
}

var conceptGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a concept by ID",
	Long:  `Retrieve a concept node by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConceptGet,
}

func runConceptGet(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	conceptID := args[0]

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	c, err := db.GetConceptByID(conceptID)
	if err != nil {
		exitWithError(ExitDataError, "querying concept: %v", err)
	}
	if c == nil {
		exitWithError(ExitConceptNotFound, "concept %q not found", conceptID)
	}

	if humanOutput {
		fmt.Println(c.ID)
		fmt.Printf("  Name: %s\n", c.Name)
		if len(c.Aliases) > 0 {
			fmt.Printf("  Aliases: %s\n", strings.Join(c.Aliases, ", "))
		}
		if c.Description != "" {
			fmt.Printf("  Description: %s\n", c.Description)
		}
	} else {
		outputJSON(c)
	}

	return nil
}

// ConceptListResult is the response for the concept list command.
type ConceptListResult struct {
	Concepts []concept.Concept `json:"concepts"`
	Count    int               `json:"count"`
}

var conceptListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all concepts",
	Long:  `List all concept nodes in the knowledge graph.`,
	RunE:  runConceptList,
}

func runConceptList(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	concepts, err := db.GetAllConcepts()
	if err != nil {
		exitWithError(ExitDataError, "querying concepts: %v", err)
	}

	if humanOutput {
		if len(concepts) == 0 {
			fmt.Println("No concepts found")
			return nil
		}
		for i, c := range concepts {
			if i > 0 {
				fmt.Println()
			}
			fmt.Println(c.ID)
			fmt.Printf("  Name: %s\n", c.Name)
			if len(c.Aliases) > 0 {
				fmt.Printf("  Aliases: %s\n", strings.Join(c.Aliases, ", "))
			}
		}
		fmt.Printf("\nTotal: %d concepts\n", len(concepts))
	} else {
		if concepts == nil {
			concepts = []concept.Concept{}
		}
		outputJSON(ConceptListResult{
			Concepts: concepts,
			Count:    len(concepts),
		})
	}

	return nil
}

// ConceptUpdateResult is the response for the concept update command.
type ConceptUpdateResult struct {
	Status  string          `json:"status"`
	Concept concept.Concept `json:"concept"`
}

var conceptUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a concept",
	Long:  `Update an existing concept node.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConceptUpdate,
}

func runConceptUpdate(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	conceptID := args[0]

	nameFlag := cmd.Flags().Changed("name")
	aliasesFlag := cmd.Flags().Changed("aliases")
	descFlag := cmd.Flags().Changed("description")

	if !nameFlag && !aliasesFlag && !descFlag {
		exitWithError(ExitConceptValidation, "no update flags provided (use --name, --aliases, or --description)")
	}

	// Load existing concepts
	conceptsPath := config.ConceptsPath(repoRoot)
	concepts, err := storage.ReadAllConcepts(conceptsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading concepts: %v", err)
	}

	// Find concept
	idx, found := storage.FindConceptByID(concepts, conceptID)
	if !found {
		exitWithError(ExitConceptNotFound, "concept %q not found", conceptID)
	}

	// Apply updates
	c := concepts[idx]
	if nameFlag {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			exitWithError(ExitConceptValidation, "name cannot be empty")
		}
		c.Name = name
	}
	if aliasesFlag {
		aliasesStr, _ := cmd.Flags().GetString("aliases")
		if aliasesStr == "" {
			c.Aliases = nil
		} else {
			aliases := strings.Split(aliasesStr, ",")
			for i := range aliases {
				aliases[i] = strings.TrimSpace(aliases[i])
			}
			c.Aliases = aliases
		}
	}
	if descFlag {
		description, _ := cmd.Flags().GetString("description")
		c.Description = description
	}

	concepts[idx] = c

	// Write back
	if err := storage.WriteAllConcepts(conceptsPath, concepts); err != nil {
		exitWithError(ExitDataError, "writing concepts: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Updated concept: %s\n", conceptID)
		fmt.Printf("  Name: %s\n", c.Name)
		if len(c.Aliases) > 0 {
			fmt.Printf("  Aliases: %s\n", strings.Join(c.Aliases, ", "))
		}
		if c.Description != "" {
			fmt.Printf("  Description: %s\n", c.Description)
		}
	} else {
		outputJSON(ConceptUpdateResult{
			Status:  "updated",
			Concept: c,
		})
	}

	return nil
}

// ConceptDeleteResult is the response for the concept delete command.
type ConceptDeleteResult struct {
	Status       string `json:"status"`
	ID           string `json:"id"`
	EdgesRemoved int    `json:"edges_removed"`
}

// ConceptDeleteBlockedResult is the response when delete is blocked by edges.
type ConceptDeleteBlockedResult struct {
	Error     string `json:"error"`
	EdgeCount int    `json:"edge_count"`
}

var conceptDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a concept",
	Long:  `Delete a concept node from the knowledge graph.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConceptDelete,
}

func runConceptDelete(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	conceptID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Load existing concepts
	conceptsPath := config.ConceptsPath(repoRoot)
	concepts, err := storage.ReadAllConcepts(conceptsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading concepts: %v", err)
	}

	// Find concept
	_, found := storage.FindConceptByID(concepts, conceptID)
	if !found {
		exitWithError(ExitConceptNotFound, "concept %q not found", conceptID)
	}

	// Check for linked edges
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	edgeCount, err := db.CountEdgesByTarget(conceptID)
	if err != nil {
		exitWithError(ExitDataError, "counting edges: %v", err)
	}

	if edgeCount > 0 && !force {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: concept %q has %d linked edges; use --force to delete anyway\n", conceptID, edgeCount)
		} else {
			outputJSON(ConceptDeleteBlockedResult{
				Error:     fmt.Sprintf("concept %q has %d linked edges; use --force to delete anyway", conceptID, edgeCount),
				EdgeCount: edgeCount,
			})
		}
		os.Exit(ExitConceptValidation)
	}

	// Delete concept
	concepts, _ = storage.DeleteConceptFromSlice(concepts, conceptID)
	if err := storage.WriteAllConcepts(conceptsPath, concepts); err != nil {
		exitWithError(ExitDataError, "writing concepts: %v", err)
	}

	// Delete linked edges if force
	edgesRemoved := 0
	if force && edgeCount > 0 {
		edgesPath := config.EdgesPath(repoRoot)
		edges, err := storage.ReadAllEdges(edgesPath)
		if err != nil {
			exitWithError(ExitDataError, "reading edges: %v", err)
		}

		var remaining []edge.Edge
		for _, e := range edges {
			if e.TargetID != conceptID {
				remaining = append(remaining, e)
			} else {
				edgesRemoved++
			}
		}

		if err := storage.WriteAllEdges(edgesPath, remaining); err != nil {
			exitWithError(ExitDataError, "writing edges: %v", err)
		}

		// Rebuild edges index
		if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
			exitWithError(ExitDataError, "rebuilding edges index: %v", err)
		}
	}

	// Rebuild concepts index
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		if edgesRemoved > 0 {
			fmt.Printf("Deleted concept %q and %d linked edges\n", conceptID, edgesRemoved)
		} else {
			fmt.Printf("Deleted concept %q\n", conceptID)
		}
	} else {
		outputJSON(ConceptDeleteResult{
			Status:       "deleted",
			ID:           conceptID,
			EdgesRemoved: edgesRemoved,
		})
	}

	return nil
}

// ConceptPapersResult is the response for the concept papers command.
type ConceptPapersResult struct {
	ConceptID string                     `json:"concept_id"`
	Papers    []storage.PaperConceptEdge `json:"papers"`
	Count     int                        `json:"count"`
}

var conceptPapersCmd = &cobra.Command{
	Use:   "papers <concept-id>",
	Short: "List papers linked to a concept",
	Long:  `Query all papers linked to a specific concept, optionally filtered by relationship type.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConceptPapers,
}

func runConceptPapers(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	conceptID := args[0]
	relType, _ := cmd.Flags().GetString("type")

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Verify concept exists
	c, err := db.GetConceptByID(conceptID)
	if err != nil {
		exitWithError(ExitDataError, "querying concept: %v", err)
	}
	if c == nil {
		exitWithError(ExitConceptNotFound, "concept %q not found", conceptID)
	}

	// Get papers
	papers, err := db.GetPapersByConcept(conceptID, relType)
	if err != nil {
		exitWithError(ExitDataError, "querying papers: %v", err)
	}

	if humanOutput {
		fmt.Printf("Papers linked to: %s\n", conceptID)
		if len(papers) == 0 {
			fmt.Println("\n(no papers)")
		} else {
			// Group by relationship type
			byType := make(map[string][]storage.PaperConceptEdge)
			for _, p := range papers {
				byType[p.RelationshipType] = append(byType[p.RelationshipType], p)
			}

			for relType, typePapers := range byType {
				fmt.Printf("\n[%s]\n", relType)
				for _, p := range typePapers {
					fmt.Printf("  %s: %s\n", p.PaperID, p.Summary)
				}
			}
		}
		fmt.Printf("\nTotal: %d papers\n", len(papers))
	} else {
		if papers == nil {
			papers = []storage.PaperConceptEdge{}
		}
		outputJSON(ConceptPapersResult{
			ConceptID: conceptID,
			Papers:    papers,
			Count:     len(papers),
		})
	}

	return nil
}

// ConceptMergeResult is the response for the concept merge command.
type ConceptMergeResult struct {
	Status            string   `json:"status"`
	SourceID          string   `json:"source_id"`
	TargetID          string   `json:"target_id"`
	EdgesUpdated      int      `json:"edges_updated"`
	AliasesAdded      []string `json:"aliases_added"`
	DuplicatesRemoved int      `json:"duplicates_removed"`
}

var conceptMergeCmd = &cobra.Command{
	Use:   "merge <source-id> <target-id>",
	Short: "Merge one concept into another",
	Long:  `Merge source concept into target concept, transferring all edges.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runConceptMerge,
}

func runConceptMerge(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	sourceID := args[0]
	targetID := args[1]

	// Validate not same
	if sourceID == targetID {
		exitWithError(ExitConceptValidation, "source and target concepts cannot be the same")
	}

	// Load concepts
	conceptsPath := config.ConceptsPath(repoRoot)
	concepts, err := storage.ReadAllConcepts(conceptsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading concepts: %v", err)
	}

	// Find source
	sourceIdx, found := storage.FindConceptByID(concepts, sourceID)
	if !found {
		exitWithError(ExitConceptNotFound, "source concept %q not found", sourceID)
	}
	sourceConcept := concepts[sourceIdx]

	// Find target
	targetIdx, found := storage.FindConceptByID(concepts, targetID)
	if !found {
		exitWithError(ExitConceptNotFound, "target concept %q not found", targetID)
	}
	targetConcept := &concepts[targetIdx]

	// Merge aliases
	aliasesAdded := targetConcept.MergeAliases(&sourceConcept)

	// Delete source concept
	concepts, _ = storage.DeleteConceptFromSlice(concepts, sourceID)

	// Write concepts
	if err := storage.WriteAllConcepts(conceptsPath, concepts); err != nil {
		exitWithError(ExitDataError, "writing concepts: %v", err)
	}

	// Load edges and update target_id
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		exitWithError(ExitDataError, "reading edges: %v", err)
	}

	edgesUpdated := 0
	for i := range edges {
		if edges[i].TargetID == sourceID {
			edges[i].TargetID = targetID
			edgesUpdated++
		}
	}

	// Deduplicate edges (same source_id + target_id + relationship_type)
	// Keep the one with earlier created_at
	duplicatesRemoved := 0
	seen := make(map[edge.EdgeKey]int) // key -> index in result
	var deduped []edge.Edge

	for _, e := range edges {
		key := e.Key()
		if existingIdx, exists := seen[key]; exists {
			// Duplicate found - keep the one with earlier created_at
			if e.CreatedAt < deduped[existingIdx].CreatedAt {
				deduped[existingIdx] = e
			}
			duplicatesRemoved++
		} else {
			seen[key] = len(deduped)
			deduped = append(deduped, e)
		}
	}

	// Write edges
	if err := storage.WriteAllEdges(edgesPath, deduped); err != nil {
		exitWithError(ExitDataError, "writing edges: %v", err)
	}

	// Update indexes
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		exitWithError(ExitDataError, "updating concepts index: %v", err)
	}
	if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
		exitWithError(ExitDataError, "updating edges index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Merged %q into %q\n", sourceID, targetID)
		fmt.Printf("  Edges updated: %d\n", edgesUpdated)
		if len(aliasesAdded) > 0 {
			fmt.Printf("  Aliases added: %s\n", strings.Join(aliasesAdded, ", "))
		}
		if duplicatesRemoved > 0 {
			fmt.Printf("  Duplicate edges removed: %d\n", duplicatesRemoved)
		}
	} else {
		if aliasesAdded == nil {
			aliasesAdded = []string{}
		}
		outputJSON(ConceptMergeResult{
			Status:            "merged",
			SourceID:          sourceID,
			TargetID:          targetID,
			EdgesUpdated:      edgesUpdated,
			AliasesAdded:      aliasesAdded,
			DuplicatesRemoved: duplicatesRemoved,
		})
	}

	return nil
}
