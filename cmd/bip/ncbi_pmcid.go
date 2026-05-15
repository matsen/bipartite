package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/ncbi"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var ncbiPMCIDEmail string

func init() {
	ncbiPMCIDCmd.Flags().StringVar(&ncbiPMCIDEmail, "email", "", "Identification email sent to NCBI (recommended)")
	ncbiCmd.AddCommand(ncbiPMCIDCmd)
}

var ncbiPMCIDCmd = &cobra.Command{
	Use:   "pmcid <ref-id | DOI:... | PMID:...>",
	Short: "One-off PMCID lookup for a single identifier",
	Long: `Resolve a single identifier to its PMCID via NCBI's ID Converter.

The argument can be:
  - A bipartite ref ID (resolved against refs.jsonl to get its DOI/PMID)
  - A prefixed identifier: DOI:10.1038/... or PMID:12345678
  - A bare DOI (e.g., 10.1038/...) — treated as DOI:

Examples:
  bip ncbi pmcid DOI:10.1038/s41586-020-2649-2
  bip ncbi pmcid PMID:32939066
  bip ncbi pmcid Smith2024-ab`,
	Args: cobra.ExactArgs(1),
	RunE: runNCBIPMCID,
}

func runNCBIPMCID(cmd *cobra.Command, args []string) error {
	input, err := resolveNCBIArg(args[0])
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}

	client := newNCBIClient(ncbiPMCIDEmail)

	records, err := client.Convert(context.Background(), []ncbi.Input{input})
	if err != nil {
		exitWithError(ExitError, "querying NCBI: %v", err)
	}

	if humanOutput {
		printPMCIDHuman(input, records)
		return nil
	}
	if err := outputJSON(records); err != nil {
		exitWithError(ExitError, "encoding JSON: %v", err)
	}
	return nil
}

// resolveNCBIArg interprets the user's argument as one of:
//   - DOI:... → IDTypeDOI with the trimmed value
//   - PMID:... → IDTypePMID with the trimmed value
//   - bare DOI (contains "/") → IDTypeDOI
//   - otherwise → ref ID; look up in refs.jsonl and prefer DOI over PMID
func resolveNCBIArg(arg string) (ncbi.Input, error) {
	if strings.HasPrefix(arg, "DOI:") {
		return ncbi.Input{Type: ncbi.IDTypeDOI, ID: strings.TrimPrefix(arg, "DOI:")}, nil
	}
	if strings.HasPrefix(arg, "PMID:") {
		return ncbi.Input{Type: ncbi.IDTypePMID, ID: strings.TrimPrefix(arg, "PMID:")}, nil
	}
	if strings.Contains(arg, "/") {
		return ncbi.Input{Type: ncbi.IDTypeDOI, ID: arg}, nil
	}

	// Treat as ref ID. Load refs.jsonl and look up.
	repoRoot := mustFindRepository()
	refs, err := storage.ReadAll(config.RefsPath(repoRoot))
	if err != nil {
		return ncbi.Input{}, fmt.Errorf("reading refs: %w", err)
	}
	idx, ok := storage.FindByID(refs, arg)
	if !ok {
		return ncbi.Input{}, fmt.Errorf("ref not found: %s", arg)
	}
	r := refs[idx]
	if r.DOI != "" {
		return ncbi.Input{Type: ncbi.IDTypeDOI, ID: r.DOI}, nil
	}
	if r.PMID != "" {
		return ncbi.Input{Type: ncbi.IDTypePMID, ID: r.PMID}, nil
	}
	return ncbi.Input{}, fmt.Errorf("ref %s has no DOI or PMID", arg)
}

func printPMCIDHuman(in ncbi.Input, records []ncbi.Record) {
	fmt.Printf("Query: %s:%s\n", in.Type, in.ID)
	if len(records) == 0 {
		fmt.Println("No record returned.")
		return
	}
	for _, r := range records {
		if r.PMCID != "" {
			fmt.Printf("PMCID: %s\n", r.PMCID)
		} else {
			msg := r.ErrMsg
			if msg == "" {
				msg = "no PMCID in response"
			}
			fmt.Printf("No PMCID: %s\n", msg)
		}
		if r.DOI != "" {
			fmt.Printf("DOI:   %s\n", r.DOI)
		}
		if r.PMID != 0 {
			fmt.Printf("PMID:  %d\n", r.PMID)
		}
	}
}
