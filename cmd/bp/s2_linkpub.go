package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	s2LinkPubAuto bool
)

var s2LinkPubCmd = &cobra.Command{
	Use:   "link-published",
	Short: "Find and link preprints to their published versions",
	Long: `Scan collection for preprints and find their published versions.

Identifies papers from bioRxiv, medRxiv, or arXiv and searches
Semantic Scholar for published versions with matching titles.

Examples:
  bp s2 link-published --human
  bp s2 link-published --auto`,
	Args: cobra.NoArgs,
	RunE: runS2LinkPub,
}

func init() {
	s2Cmd.AddCommand(s2LinkPubCmd)
	s2LinkPubCmd.Flags().BoolVar(&s2LinkPubAuto, "auto", false, "Automatically link without confirmation")
}

// S2LinkPubResult is the JSON output for the link-published command.
type S2LinkPubResult struct {
	Linked           []S2LinkInfo   `json:"linked"`
	NoPublishedFound []string       `json:"no_published_found"`
	AlreadyLinked    []string       `json:"already_linked"`
	TotalPreprints   int            `json:"total_preprints"`
	Error            *S2ErrorResult `json:"error,omitempty"`
}

// S2LinkInfo represents a linked preprint-published pair.
type S2LinkInfo struct {
	PreprintID     string `json:"preprint_id"`
	PreprintDOI    string `json:"preprint_doi"`
	PublishedDOI   string `json:"published_doi"`
	PublishedVenue string `json:"published_venue"`
}

func runS2LinkPub(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Find repository and load refs
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputLinkPubError(ExitS2APIError, "reading refs", err)
	}

	// Create client
	client := s2.NewClient()

	// Find preprints
	preprints := findPreprints(refs)
	if len(preprints) == 0 {
		return outputLinkPubResult(S2LinkPubResult{
			Linked:           []S2LinkInfo{},
			NoPublishedFound: []string{},
			AlreadyLinked:    []string{},
			TotalPreprints:   0,
		})
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Scanning %d preprints for published versions...\n\n", len(preprints))
	}

	result := S2LinkPubResult{
		Linked:           []S2LinkInfo{},
		NoPublishedFound: []string{},
		AlreadyLinked:    []string{},
		TotalPreprints:   len(preprints),
	}

	// Track which refs need updating
	updatedRefs := make(map[int]reference.Reference)

	for idx, ref := range refs {
		if !isPreprint(ref) {
			continue
		}

		// Check if already linked
		if ref.Supersedes != "" {
			result.AlreadyLinked = append(result.AlreadyLinked, ref.ID)
			continue
		}

		// Search for published version
		published, err := findPublishedVersion(ctx, client, ref)
		if err != nil {
			if s2.IsRateLimited(err) {
				return outputS2RateLimited(err)
			}
			// Warn about unexpected errors instead of silently ignoring
			warnAPIError("Failed to find published version", ref.ID, err)
			continue
		}

		if published == nil {
			result.NoPublishedFound = append(result.NoPublishedFound, ref.ID)
			continue
		}

		// Found published version
		if humanOutput {
			fmt.Printf("  %s (%s)\n", ref.ID, ref.DOI)
			fmt.Printf("    -> Published in %s: %s\n", published.Venue, published.ExternalIDs.DOI)
		}

		// Confirm or auto-link
		shouldLink := s2LinkPubAuto
		if !s2LinkPubAuto && humanOutput {
			fmt.Print("    Link? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			shouldLink = strings.ToLower(strings.TrimSpace(input)) == "y"
		}

		if shouldLink {
			// Update the reference
			ref.Supersedes = published.ExternalIDs.DOI
			updatedRefs[idx] = ref

			result.Linked = append(result.Linked, S2LinkInfo{
				PreprintID:     ref.ID,
				PreprintDOI:    ref.DOI,
				PublishedDOI:   published.ExternalIDs.DOI,
				PublishedVenue: published.Venue,
			})

			if humanOutput {
				fmt.Println("    Linked!")
			}
		}

		if humanOutput {
			fmt.Println()
		}
	}

	// Write updated refs if any links were made
	if len(updatedRefs) > 0 {
		for idx, updated := range updatedRefs {
			refs[idx] = updated
		}
		if err := storage.WriteAll(refsPath, refs); err != nil {
			return outputLinkPubError(ExitS2APIError, "saving refs", err)
		}
	}

	return outputLinkPubResult(result)
}

func findPreprints(refs []reference.Reference) []reference.Reference {
	var preprints []reference.Reference
	for _, ref := range refs {
		if isPreprint(ref) {
			preprints = append(preprints, ref)
		}
	}
	return preprints
}

func isPreprint(ref reference.Reference) bool {
	venue := strings.ToLower(ref.Venue)
	return strings.Contains(venue, "biorxiv") ||
		strings.Contains(venue, "medrxiv") ||
		strings.Contains(venue, "arxiv")
}

func findPublishedVersion(ctx context.Context, client *s2.Client, preprint reference.Reference) (*s2.S2Paper, error) {
	// Search by title
	searchResp, err := client.SearchByTitle(ctx, preprint.Title, S2SearchLimit)
	if err != nil {
		return nil, err
	}

	// Find a match that's not a preprint
	for _, paper := range searchResp.Data {
		// Skip preprints
		paperVenue := strings.ToLower(paper.Venue)
		if strings.Contains(paperVenue, "biorxiv") ||
			strings.Contains(paperVenue, "medrxiv") ||
			strings.Contains(paperVenue, "arxiv") {
			continue
		}

		// Check title similarity with strict matching
		if !titlesMatchStrict(preprint.Title, paper.Title) {
			continue
		}

		// Check author overlap with strict matching
		if !authorsOverlapStrict(preprint.Authors, paper.Authors) {
			continue
		}

		// Must have a DOI
		if paper.ExternalIDs.DOI == "" {
			continue
		}

		return &paper, nil
	}

	return nil, nil
}

func outputLinkPubResult(result S2LinkPubResult) error {
	if humanOutput {
		fmt.Printf("Summary: %d linked, %d no published version, %d already linked\n",
			len(result.Linked), len(result.NoPublishedFound), len(result.AlreadyLinked))
	} else {
		outputJSON(result)
	}
	return nil
}

func outputLinkPubError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "api_error", context, err)
}
