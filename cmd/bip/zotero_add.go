package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/matsen/bipartite/internal/zotero"
	"github.com/spf13/cobra"
)

var zoteroAddCmd = &cobra.Command{
	Use:   "add <paper-id>",
	Short: "Add a paper to both bip and Zotero",
	Long: `Add a paper by fetching metadata from Semantic Scholar and creating
it in both your bip library and your Zotero library.

Supported paper ID formats:
  DOI:10.1038/nature12373      DOI
  ARXIV:2106.15928             arXiv ID
  PMID:19872477                PubMed ID

The paper metadata is fetched from Semantic Scholar (free, no Zotero
lookup needed), then pushed to Zotero and saved to bip simultaneously.

Examples:
  bip zotero add DOI:10.1038/nature12373
  bip zotero add DOI:10.1093/ve/vead055 --human`,
	Args: cobra.ExactArgs(1),
	RunE: runZoteroAdd,
}

func init() {
	zoteroCmd.AddCommand(zoteroAddCmd)
}

// ZoteroAddResult is the JSON output for the add command.
type ZoteroAddResult struct {
	Action    string             `json:"action"` // added, skipped
	BipPaper  *S2AddPaperSummary `json:"bip_paper,omitempty"`
	ZoteroKey string             `json:"zotero_key,omitempty"`
	Error     *S2ErrorResult     `json:"error,omitempty"`
}

func runZoteroAdd(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Create clients
	zotClient, err := zotero.NewClient()
	if err != nil {
		return outputZoteroError(ExitZoteroNotConfigured, "Zotero not configured", err)
	}
	s2Client := s2.NewClient()

	// Find repository
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Load existing refs
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputZoteroError(ExitZoteroAPIError, "reading refs", err)
	}

	// Parse and fetch from S2
	parsed := s2.ParsePaperID(paperID)
	s2ID := parsed.String()

	paper, err := s2Client.GetPaper(ctx, s2ID)
	if err != nil {
		if s2.IsNotFound(err) {
			return outputGenericNotFound(paperID, "Paper not found in Semantic Scholar")
		}
		return outputZoteroError(ExitZoteroAPIError, "fetching from Semantic Scholar", err)
	}

	// Map to reference
	ref := s2.MapS2ToReference(*paper)

	// Check for duplicates in bip
	if paper.ExternalIDs.DOI != "" {
		if _, found := storage.FindByDOI(refs, paper.ExternalIDs.DOI); found {
			return outputZoteroDuplicate(ref.ID, paper.ExternalIDs.DOI)
		}
	}

	// Add to Zotero
	zotItem := zotero.MapReferenceToZotero(ref)
	createdItem, err := zotClient.CreateItem(ctx, zotItem)
	if err != nil {
		return outputZoteroError(ExitZoteroAPIError, "creating item in Zotero", err)
	}

	// Update the ref with the Zotero key
	ref.Source = reference.ImportSource{
		Type: "zotero",
		ID:   createdItem.Key,
	}

	// Add to bip
	ref.ID = storage.GenerateUniqueID(refs, ref.ID)
	if err := storage.Append(refsPath, ref); err != nil {
		return outputZoteroError(ExitZoteroAPIError, "saving reference", err)
	}

	// Output
	authors := formatAuthors(ref.Authors)
	result := ZoteroAddResult{
		Action:    "added",
		ZoteroKey: createdItem.Key,
		BipPaper: &S2AddPaperSummary{
			ID:      ref.ID,
			DOI:     ref.DOI,
			Title:   ref.Title,
			Authors: authors,
			Year:    ref.Published.Year,
			Venue:   ref.Venue,
		},
	}

	if humanOutput {
		fmt.Printf("Added to both bip and Zotero:\n")
		fmt.Printf("  bip ID:     %s\n", ref.ID)
		fmt.Printf("  Zotero key: %s\n", createdItem.Key)
		fmt.Printf("  Title:      %s\n", ref.Title)
		fmt.Printf("  Authors:    %s\n", joinAuthorsDisplay(authors))
		fmt.Printf("  Year:       %d\n", ref.Published.Year)
		if ref.Venue != "" {
			fmt.Printf("  Venue:      %s\n", ref.Venue)
		}
	} else {
		outputJSON(result)
	}

	return nil
}

func outputZoteroDuplicate(existingID, doi string) error {
	result := ZoteroAddResult{
		Action: "skipped",
		Error: &S2ErrorResult{
			Code:    "duplicate",
			Message: "Paper already exists in bip collection",
			PaperID: existingID,
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Paper already exists: %s\n", existingID)
		if doi != "" {
			fmt.Fprintf(os.Stderr, "  DOI: %s\n", doi)
		}
	} else {
		outputJSON(result)
	}
	os.Exit(ExitZoteroDuplicate)
	return nil
}
