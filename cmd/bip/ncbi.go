package main

import (
	"github.com/spf13/cobra"
)

var ncbiCmd = &cobra.Command{
	Use:   "ncbi",
	Short: "NCBI PMC ID Converter commands",
	Long: `Commands for resolving identifiers via NCBI's PMC ID Converter API.

The converter maps between DOI, PMID, PMCID, and MID identifiers. The primary
use case is backfilling PMCIDs for references that have a DOI or PMID but no
PMCID — needed for NIH RPPR / public access compliance and for surfacing the
PMC full-text link when available.

NCBI only knows PMCIDs for papers actually deposited in PMC, which is a subset
of open-access literature. Absence of a PMCID after backfill is not a signal
that the paper is missing; it likely just isn't in PMC.`,
}

func init() {
	rootCmd.AddCommand(ncbiCmd)
}
