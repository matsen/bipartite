package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/clipboard"
	"github.com/spf13/cobra"
)

// URL format flags
var (
	urlCopyFlag   bool
	urlPubmedFlag bool
	urlPmcFlag    bool
	urlArxivFlag  bool
	urlS2Flag     bool
)

// URLResult is the JSON output for bip url command.
type URLResult struct {
	URL    string `json:"url"`
	Format string `json:"format"` // doi, pubmed, pmc, arxiv, s2
	Copied bool   `json:"copied"` // true if --copy succeeded
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

func runURL(cmd *cobra.Command, args []string) error {
	// Check mutual exclusivity of format flags
	flagCount := 0
	if urlPubmedFlag {
		flagCount++
	}
	if urlPmcFlag {
		flagCount++
	}
	if urlArxivFlag {
		flagCount++
	}
	if urlS2Flag {
		flagCount++
	}
	if flagCount > 1 {
		exitWithError(ExitError, "specify only one URL format flag (--pubmed, --pmc, --arxiv, or --s2)")
	}

	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	id := args[0]
	ref, err := db.GetByID(id)
	if err != nil {
		exitWithError(ExitError, "getting reference: %v", err)
	}
	if ref == nil {
		exitWithError(ExitError, "reference not found: %s", id)
	}

	// Determine URL and format based on flags
	var url, format string

	switch {
	case urlPubmedFlag:
		format = "pubmed"
		if ref.PMID == "" {
			exitWithError(ExitError, "no PubMed ID available for %s", id)
		}
		url = fmt.Sprintf("https://pubmed.ncbi.nlm.nih.gov/%s/", ref.PMID)

	case urlPmcFlag:
		format = "pmc"
		if ref.PMCID == "" {
			exitWithError(ExitError, "no PMC ID available for %s", id)
		}
		url = fmt.Sprintf("https://www.ncbi.nlm.nih.gov/pmc/articles/%s/", ref.PMCID)

	case urlArxivFlag:
		format = "arxiv"
		if ref.ArXivID == "" {
			exitWithError(ExitError, "no arXiv ID available for %s", id)
		}
		url = fmt.Sprintf("https://arxiv.org/abs/%s", ref.ArXivID)

	case urlS2Flag:
		format = "s2"
		if ref.S2ID == "" {
			exitWithError(ExitError, "no Semantic Scholar ID available for %s", id)
		}
		url = fmt.Sprintf("https://www.semanticscholar.org/paper/%s", ref.S2ID)

	default:
		// Default to DOI
		format = "doi"
		if ref.DOI == "" {
			exitWithError(ExitError, "no DOI available for %s", id)
		}
		url = fmt.Sprintf("https://doi.org/%s", ref.DOI)
	}

	// Handle clipboard copy
	copied := false
	var clipboardWarning string
	if urlCopyFlag {
		if !clipboard.IsAvailable() {
			clipboardWarning = "clipboard unavailable (install xclip or xsel on Linux)"
		} else if err := clipboard.Copy(url); err != nil {
			if errors.Is(err, clipboard.ErrClipboardUnavailable) {
				clipboardWarning = "clipboard unavailable (install xclip or xsel on Linux)"
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
			Format: format,
			Copied: copied,
		})
	}

	return nil
}
