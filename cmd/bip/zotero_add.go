package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	Long: `Add a paper by fetching metadata and creating it in both your bip
library and your Zotero library.

Metadata is fetched from Semantic Scholar first. If S2 is rate-limited
and the ID is a DOI, falls back to CrossRef (free, no key needed).

Supported paper ID formats:
  DOI:10.1038/nature12373      DOI
  ARXIV:2106.15928             arXiv ID
  PMID:19872477                PubMed ID

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
	Action    string             `json:"action"`              // added, skipped
	Source    string             `json:"source,omitempty"`    // s2, crossref
	BipPaper  *S2AddPaperSummary `json:"bip_paper,omitempty"`
	ZoteroKey string             `json:"zotero_key,omitempty"`
	Error     *S2ErrorResult     `json:"error,omitempty"`
}

func runZoteroAdd(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Create Zotero client
	zotClient, err := zotero.NewClient()
	if err != nil {
		return outputZoteroError(ExitZoteroNotConfigured, "Zotero not configured", err)
	}

	// Find repository
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Load existing refs
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputZoteroError(ExitZoteroAPIError, "reading refs", err)
	}

	// Resolve metadata: try S2 first, fall back to CrossRef for DOIs
	ref, metadataSource, err := resolveMetadata(ctx, paperID)
	if err != nil {
		return outputZoteroError(ExitZoteroAPIError, "resolving paper metadata", err)
	}

	// Check for duplicates in bip
	if ref.DOI != "" {
		if _, found := storage.FindByDOI(refs, ref.DOI); found {
			return outputZoteroDuplicate(ref.ID, ref.DOI)
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
		Source:    metadataSource,
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
		fmt.Printf("Added to both bip and Zotero (via %s):\n", metadataSource)
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

// resolveMetadata tries S2 first, then CrossRef for DOIs.
func resolveMetadata(ctx context.Context, paperID string) (reference.Reference, string, error) {
	parsed := s2.ParsePaperID(paperID)
	s2ID := parsed.String()

	// Try Semantic Scholar first
	s2Client := s2.NewClient()
	paper, err := s2Client.GetPaper(ctx, s2ID)
	if err == nil {
		ref := s2.MapS2ToReference(*paper)
		return ref, "s2", nil
	}

	// If S2 failed and we have a DOI, try CrossRef
	if parsed.Type == "DOI" || strings.HasPrefix(paperID, "DOI:") {
		doi := parsed.Value
		if humanOutput {
			fmt.Fprintf(os.Stderr, "S2 unavailable (%v), trying CrossRef...\n", err)
		}

		ref, crErr := zotero.LookupDOI(ctx, doi)
		if crErr == nil {
			return ref, "crossref", nil
		}
		// Both failed
		return reference.Reference{}, "", fmt.Errorf("S2: %v; CrossRef: %v", err, crErr)
	}

	// Non-DOI identifier and S2 failed
	if s2.IsNotFound(err) {
		return reference.Reference{}, "", fmt.Errorf("paper not found in Semantic Scholar: %s", paperID)
	}
	return reference.Reference{}, "", fmt.Errorf("Semantic Scholar error: %v", err)
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
