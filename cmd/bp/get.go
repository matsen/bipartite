package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/storage"
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
  bp get Ahn2026-rs`,
	Args: cobra.ExactArgs(1),
	RunE: runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	root, exitCode := getRepoRoot()
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	// Find repository
	repoRoot, err := config.FindRepository(root)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: err.Error()})
		}
		os.Exit(ExitConfigError)
	}

	// Open database
	dbPath := config.DBPath(repoRoot)
	db, err := storage.OpenDB(dbPath)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: opening database: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("opening database: %v", err)})
		}
		os.Exit(ExitError)
	}
	defer db.Close()

	id := args[0]
	ref, err := db.GetByID(id)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: getting reference: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("getting reference: %v", err)})
		}
		os.Exit(ExitError)
	}

	if ref == nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: reference not found: %s\n", id)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("reference not found: %s", id)})
		}
		os.Exit(ExitError)
	}

	if humanOutput {
		printRefDetail(*ref)
	} else {
		outputJSON(ref)
	}

	return nil
}

func printRefDetail(ref storage.Reference) {
	fmt.Println(ref.ID)
	fmt.Println(strings.Repeat("═", 70))
	fmt.Println()

	fmt.Printf("Title:    %s\n", wrapText(ref.Title, 60, "          "))
	fmt.Println()

	// Authors
	if len(ref.Authors) > 0 {
		var authorNames []string
		for _, a := range ref.Authors {
			if a.First != "" {
				authorNames = append(authorNames, a.First+" "+a.Last)
			} else {
				authorNames = append(authorNames, a.Last)
			}
		}
		fmt.Printf("Authors:  %s\n", wrapText(strings.Join(authorNames, ", "), 60, "          "))
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
		fmt.Printf("  %s\n", wrapText(ref.Abstract, 68, "  "))
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

func wrapText(text string, width int, indent string) string {
	if len(text) <= width {
		return text
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n"+indent)
}
