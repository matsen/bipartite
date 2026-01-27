package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/clipboard"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/spf13/cobra"
)

// URL template constants for each external identifier format.
const (
	urlTemplateDOI    = "https://doi.org/%s"
	urlTemplatePubMed = "https://pubmed.ncbi.nlm.nih.gov/%s/"
	urlTemplatePMC    = "https://www.ncbi.nlm.nih.gov/pmc/articles/%s/"
	urlTemplateArXiv  = "https://arxiv.org/abs/%s"
	urlTemplateS2     = "https://www.semanticscholar.org/paper/%s"
)

// clipboardUnavailableMsg is the standard warning when clipboard is not available.
const clipboardUnavailableMsg = "clipboard unavailable (install xclip or xsel on Linux)"

// URL format flags
var (
	urlCopyFlag   bool
	urlPubmedFlag bool
	urlPmcFlag    bool
	urlArxivFlag  bool
	urlS2Flag     bool
)

// urlFormat defines how to generate a URL for a specific external identifier type.
type urlFormat struct {
	name     string                            // Format identifier (doi, pubmed, etc.)
	template string                            // URL template with %s placeholder
	getID    func(*reference.Reference) string // Extracts the ID from a reference
	idName   string                            // Human-readable name for error messages
}

// urlFormats maps format names to their configuration.
var urlFormats = map[string]urlFormat{
	"doi": {
		name:     "doi",
		template: urlTemplateDOI,
		getID:    func(r *reference.Reference) string { return r.DOI },
		idName:   "DOI",
	},
	"pubmed": {
		name:     "pubmed",
		template: urlTemplatePubMed,
		getID:    func(r *reference.Reference) string { return r.PMID },
		idName:   "PubMed ID",
	},
	"pmc": {
		name:     "pmc",
		template: urlTemplatePMC,
		getID:    func(r *reference.Reference) string { return r.PMCID },
		idName:   "PMC ID",
	},
	"arxiv": {
		name:     "arxiv",
		template: urlTemplateArXiv,
		getID:    func(r *reference.Reference) string { return r.ArXivID },
		idName:   "arXiv ID",
	},
	"s2": {
		name:     "s2",
		template: urlTemplateS2,
		getID:    func(r *reference.Reference) string { return r.S2ID },
		idName:   "Semantic Scholar ID",
	},
}

// URLResult is the JSON output for bip url command.
type URLResult struct {
	URL    string `json:"url"`    // The generated URL
	Format string `json:"format"` // Format identifier: doi, pubmed, pmc, arxiv, s2
	Copied bool   `json:"copied"` // true if --copy succeeded; false if --copy not used or failed
}

func init() {
	urlCmd.Flags().BoolVar(&urlCopyFlag, "copy", false, "Copy URL to system clipboard")
	urlCmd.Flags().BoolVar(&urlPubmedFlag, "pubmed", false, "Output PubMed URL instead of DOI")
	urlCmd.Flags().BoolVar(&urlPmcFlag, "pmc", false, "Output PubMed Central URL instead of DOI")
	urlCmd.Flags().BoolVar(&urlArxivFlag, "arxiv", false, "Output arXiv URL instead of DOI")
	urlCmd.Flags().BoolVar(&urlS2Flag, "s2", false, "Output Semantic Scholar URL instead of DOI")
	rootCmd.AddCommand(urlCmd)
}

var urlCmd = &cobra.Command{
	Use:   "url <ref-id>",
	Short: "Get URL for a reference",
	Long: `Get URL for a reference in various formats.

By default outputs the DOI URL. Use flags to get alternative URLs.

Examples:
  bip url Ahn2026-rs              # Get DOI URL
  bip url Ahn2026-rs --copy       # Copy DOI URL to clipboard
  bip url Ahn2026-rs --pubmed     # Get PubMed URL
  bip url Ahn2026-rs --pmc        # Get PubMed Central URL
  bip url Ahn2026-rs --arxiv      # Get arXiv URL
  bip url Ahn2026-rs --s2         # Get Semantic Scholar URL`,
	Args: cobra.ExactArgs(1),
	RunE: runURL,
}

// getSelectedFormat determines which URL format to use based on command flags.
// Returns the format name and an error if multiple format flags are specified.
func getSelectedFormat() (string, error) {
	formatFlags := map[string]bool{
		"pubmed": urlPubmedFlag,
		"pmc":    urlPmcFlag,
		"arxiv":  urlArxivFlag,
		"s2":     urlS2Flag,
	}

	var selected []string
	for name, isSet := range formatFlags {
		if isSet {
			selected = append(selected, "--"+name)
		}
	}

	if len(selected) > 1 {
		return "", fmt.Errorf("specify only one URL format flag, got: %s", strings.Join(selected, ", "))
	}
	if len(selected) == 1 {
		// Extract format name from "--name"
		return strings.TrimPrefix(selected[0], "--"), nil
	}
	return "doi", nil // default
}

// generateURL creates a URL for the given reference using the specified format.
// Returns the URL and an error if the required external ID is not available.
func generateURL(ref *reference.Reference, formatName string) (string, error) {
	format, ok := urlFormats[formatName]
	if !ok {
		return "", fmt.Errorf("unknown URL format: %s", formatName)
	}

	externalID := format.getID(ref)
	if externalID == "" {
		return "", fmt.Errorf("no %s available for %s", format.idName, ref.ID)
	}

	return fmt.Sprintf(format.template, externalID), nil
}

func runURL(cmd *cobra.Command, args []string) error {
	// Determine which format to use
	formatName, err := getSelectedFormat()
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}

	// Open database and look up reference
	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	id := args[0]
	ref, err := db.GetByID(id)
	if err != nil {
		exitWithError(ExitError, "reference %s: failed to retrieve: %v", id, err)
	}
	if ref == nil {
		exitWithError(ExitError, "reference %s: not found", id)
	}

	// Generate URL
	url, err := generateURL(ref, formatName)
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}

	// Handle clipboard copy
	copied := false
	var clipboardWarning string
	if urlCopyFlag {
		if err := clipboard.Copy(url); err != nil {
			if errors.Is(err, clipboard.ErrClipboardUnavailable) {
				clipboardWarning = clipboardUnavailableMsg
			} else {
				clipboardWarning = fmt.Sprintf("clipboard error: %v", err)
			}
		} else {
			copied = true
		}
	}

	// Output result
	if humanOutput {
		// URL to stdout
		fmt.Println(url)
		// Messages to stderr
		if copied {
			fmt.Fprintln(os.Stderr, "Copied to clipboard")
		} else if clipboardWarning != "" {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", clipboardWarning)
		}
	} else {
		outputJSON(URLResult{
			URL:    url,
			Format: formatName,
			Copied: copied,
		})
	}

	return nil
}
