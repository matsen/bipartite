package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var astaPaperFields string

var astaPaperCmd = &cobra.Command{
	Use:   "paper <paper-id>",
	Short: "Get paper details by ID",
	Long: `Get detailed information about a paper by its identifier.

Supported ID formats:
  DOI:10.1093/sysbio/syy032
  ARXIV:2106.15928
  PMID:19872477
  PMCID:2323736
  CorpusId:215416146
  <S2 paper ID>

Examples:
  bp asta paper DOI:10.1093/sysbio/syy032
  bp asta paper ARXIV:2106.15928 --human
  bp asta paper DOI:10.1038/nature12373 --fields title,authors,citationCount`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaPaper,
}

func init() {
	astaPaperCmd.Flags().StringVar(&astaPaperFields, "fields", "", "Comma-separated fields to return")
	astaCmd.AddCommand(astaPaperCmd)
}

func runAstaPaper(cmd *cobra.Command, args []string) {
	paperID := args[0]
	client := asta.NewClient()

	paper, err := client.GetPaper(context.Background(), paperID, astaPaperFields)
	if err != nil {
		os.Exit(astaOutputError(err, paperID))
	}

	if astaHuman {
		fmt.Println(paper.Title)
		fmt.Printf("Authors: %s\n", formatASTAAuthors(paper.Authors))
		if paper.Year > 0 {
			fmt.Printf("Year: %d\n", paper.Year)
		}
		if paper.Venue != "" {
			fmt.Printf("Venue: %s\n", paper.Venue)
		}
		if paper.PublicationDate != "" {
			fmt.Printf("Published: %s\n", paper.PublicationDate)
		}
		if paper.Abstract != "" {
			fmt.Printf("\nAbstract:\n%s\n", paper.Abstract)
		}
		fmt.Println()
		fmt.Printf("Citations: %d | References: %d", paper.CitationCount, paper.ReferenceCount)
		if paper.IsOpenAccess {
			fmt.Print(" | Open Access")
		}
		fmt.Println()
		if len(paper.FieldsOfStudy) > 0 {
			fmt.Printf("Fields: %s\n", strings.Join(paper.FieldsOfStudy, ", "))
		}
		if paper.URL != "" {
			fmt.Printf("URL: %s\n", paper.URL)
		}
	} else {
		if err := astaOutputJSON(paper); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(ExitError)
		}
	}
}
