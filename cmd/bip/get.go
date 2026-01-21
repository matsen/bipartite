package main

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single reference by ID",
	Long: `Get a single reference by its ID.

Example:
  bip get Ahn2026-rs`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
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

	if humanOutput {
		printRefDetail(*ref)
	} else {
		outputJSON(ref)
	}

	return nil
}

func printRefDetail(ref reference.Reference) {
	fmt.Println(ref.ID)
	fmt.Println(strings.Repeat("═", DetailTitleMaxLen))
	fmt.Println()

	fmt.Printf("Title:    %s\n", wrapText(ref.Title, TextWrapWidth, "          "))
	fmt.Println()

	// Authors
	if len(ref.Authors) > 0 {
		fmt.Printf("Authors:  %s\n", wrapText(formatAuthorsFull(ref.Authors), TextWrapWidth, "          "))
		fmt.Println()
	}

	if ref.Venue != "" {
		fmt.Printf("Venue:    %s\n", ref.Venue)
	}

	// Date
	date := fmt.Sprintf("%d", ref.Published.Year)
	if ref.Published.Month > 0 {
		date = fmt.Sprintf("%d-%02d", ref.Published.Year, ref.Published.Month)
		if ref.Published.Day > 0 {
			date = fmt.Sprintf("%d-%02d-%02d", ref.Published.Year, ref.Published.Month, ref.Published.Day)
		}
	}
	fmt.Printf("Date:     %s\n", date)

	if ref.DOI != "" {
		fmt.Printf("DOI:      %s\n", ref.DOI)
	}

	// Abstract
	if ref.Abstract != "" {
		fmt.Println()
		fmt.Println("Abstract:")
		fmt.Printf("  %s\n", wrapText(ref.Abstract, DetailTextWrapWidth, "  "))
	}

	// PDF
	if ref.PDFPath != "" {
		fmt.Println()
		fmt.Printf("PDF:      %s\n", ref.PDFPath)
	}

	// Supplements
	if len(ref.SupplementPaths) > 0 {
		fmt.Println()
		fmt.Println("Supplements:")
		for i, p := range ref.SupplementPaths {
			fmt.Printf("  [%d] %s\n", i+1, p)
		}
	}
}
