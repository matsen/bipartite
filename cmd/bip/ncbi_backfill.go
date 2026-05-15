package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/ncbi"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	ncbiBackfillDryRun bool
	ncbiBackfillLimit  int
	ncbiBackfillTag    string
	ncbiBackfillEmail  string
)

func init() {
	ncbiBackfillCmd.Flags().BoolVar(&ncbiBackfillDryRun, "dry-run", false, "Query and report but do not write refs.jsonl")
	ncbiBackfillCmd.Flags().IntVar(&ncbiBackfillLimit, "limit", 0, "Cap total NCBI queries (0 = no limit)")
	ncbiBackfillCmd.Flags().StringVar(&ncbiBackfillTag, "tag", "", "Restrict to refs whose tags partial-match this substring")
	ncbiBackfillCmd.Flags().StringVar(&ncbiBackfillEmail, "email", "", "Identification email sent to NCBI (recommended)")
	ncbiCmd.AddCommand(ncbiBackfillCmd)
}

var ncbiBackfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "Backfill missing PMCIDs from NCBI for refs with a DOI or PMID",
	Long: `Sweep refs.jsonl for entries that have a DOI or PMID but no PMCID, query
the NCBI ID Converter in batches of 200, and write returned PMCIDs back into
the JSONL file.

Refs without a DOI or PMID are silently skipped. Refs that already have a
PMCID are not re-queried — running this command twice in a row makes zero
queries on the second run.

Examples:
  bip ncbi backfill --dry-run
  bip ncbi backfill --tag immunology
  bip ncbi backfill --limit 50 --email you@example.com`,
	RunE: runNCBIBackfill,
}

// BackfillSummary is the JSON output shape for ncbi backfill.
type BackfillSummary struct {
	DryRun        bool     `json:"dry_run"`
	Scanned       int      `json:"scanned"`
	NoConvertible int      `json:"no_convertible_id"`
	Queried       int      `json:"queried"`
	Found         int      `json:"found"`
	UpdatedIDs    []string `json:"updated_ids,omitempty"`
}

// Converter is the subset of *ncbi.Client used by the backfill flow. Defined
// here so tests can substitute a fake without spinning up a real HTTP server.
type Converter interface {
	Convert(ctx context.Context, inputs []ncbi.Input) ([]ncbi.Record, error)
}

// BackfillOptions controls a single backfill run.
type BackfillOptions struct {
	DryRun bool
	Limit  int
	Tag    string
}

func runNCBIBackfill(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	client := newNCBIClient(ncbiBackfillEmail)

	summary, err := backfillPMCIDs(context.Background(), refsPath, client, BackfillOptions{
		DryRun: ncbiBackfillDryRun,
		Limit:  ncbiBackfillLimit,
		Tag:    ncbiBackfillTag,
	})
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}

	emitBackfillSummary(summary)
	return nil
}

// backfillPMCIDs is the testable core of `bip ncbi backfill`. It loads refs
// from refsPath, queries the converter for candidates, and (unless DryRun)
// writes updated refs back to refsPath. The summary reflects what was scanned,
// queried, and found regardless of whether writes happened.
func backfillPMCIDs(ctx context.Context, refsPath string, client Converter, opts BackfillOptions) (BackfillSummary, error) {
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return BackfillSummary{}, fmt.Errorf("reading refs: %w", err)
	}

	candidates, summary := selectBackfillCandidates(refs, opts.Tag, opts.Limit)
	summary.DryRun = opts.DryRun

	if len(candidates) == 0 {
		return summary, nil
	}

	inputs := make([]ncbi.Input, 0, len(candidates))
	// Map requested-id (the exact string we send) to the refs[] index of its
	// source ref, so we can join results back without relying on positional order.
	candidateByRequestedID := make(map[string]int, len(candidates))
	for _, c := range candidates {
		in := candidateToInput(refs[c])
		inputs = append(inputs, in)
		candidateByRequestedID[in.ID] = c
	}
	summary.Queried = len(inputs)

	records, err := client.Convert(ctx, inputs)
	if err != nil {
		return summary, fmt.Errorf("querying NCBI: %w", err)
	}

	for _, rec := range records {
		if rec.PMCID == "" {
			continue
		}
		idx, ok := candidateByRequestedID[rec.RequestedID]
		if !ok {
			continue
		}
		refs[idx].PMCID = rec.PMCID
		summary.Found++
		summary.UpdatedIDs = append(summary.UpdatedIDs, refs[idx].ID)
	}

	if !opts.DryRun && summary.Found > 0 {
		if err := storage.WriteAll(refsPath, refs); err != nil {
			return summary, fmt.Errorf("writing refs: %w", err)
		}
	}

	return summary, nil
}

// selectBackfillCandidates returns indices into refs of entries eligible for
// backfill: have a valid DOI or PMID, no existing PMCID, and (if tag is
// non-empty) matching the tag substring. limit caps the count when > 0.
//
// Validity of DOI is checked via isLikelyDOI because NCBI rejects an entire
// batch with HTTP 400 if any single ID doesn't match its DOI pattern — one
// Paperpile sentinel (`XXXXXXX.XXXXXXX`) is enough to poison 6500 refs.
func selectBackfillCandidates(refs []reference.Reference, tag string, limit int) ([]int, BackfillSummary) {
	var summary BackfillSummary
	var candidates []int
	for i, r := range refs {
		summary.Scanned++
		if !hasConvertibleID(r) {
			summary.NoConvertible++
			continue
		}
		if r.PMCID != "" {
			continue
		}
		if tag != "" && !refMatchesTag(r, tag) {
			continue
		}
		candidates = append(candidates, i)
		if limit > 0 && len(candidates) >= limit {
			break
		}
	}
	return candidates, summary
}

// hasConvertibleID reports whether a ref carries something NCBI's converter
// will accept: either a syntactically valid DOI or a non-empty PMID.
func hasConvertibleID(r reference.Reference) bool {
	return isLikelyDOI(r.DOI) || r.PMID != ""
}

// isLikelyDOI reports whether s looks enough like a DOI for NCBI to accept.
// Real DOIs are `10.<registrant>/<suffix>` with a non-empty suffix; anything
// else (empty, sentinels like "XXXXXXX.XXXXXXX", URL fragments, prefix-only
// strings like "10.1042/") is rejected up-front to keep one garbage entry
// from poisoning the whole batch.
func isLikelyDOI(s string) bool {
	if !strings.HasPrefix(s, "10.") {
		return false
	}
	slash := strings.Index(s, "/")
	if slash < 0 {
		return false
	}
	// Must have at least one character before AND after the slash.
	return slash > len("10.") && slash < len(s)-1
}

// refMatchesTag reports whether any of ref.Tags contains the substring
// (case-insensitive), matching the `bip search --tag` partial-match semantics
// in cmd/bip/search.go.
func refMatchesTag(r reference.Reference, tag string) bool {
	needle := strings.ToLower(tag)
	for _, t := range r.Tags {
		if strings.Contains(strings.ToLower(t), needle) {
			return true
		}
	}
	return false
}

// candidateToInput maps a reference to the NCBI Input we'll query for it. DOI
// is preferred because NCBI's DOI lookup is more reliable; PMID is the
// fallback. A syntactically invalid DOI is treated as absent so the PMID
// fallback fires — selectBackfillCandidates would have rejected the ref
// entirely if neither was usable.
func candidateToInput(r reference.Reference) ncbi.Input {
	if isLikelyDOI(r.DOI) {
		return ncbi.Input{Type: ncbi.IDTypeDOI, ID: r.DOI}
	}
	return ncbi.Input{Type: ncbi.IDTypePMID, ID: r.PMID}
}

func emitBackfillSummary(s BackfillSummary) {
	if humanOutput {
		mode := "would update"
		if !s.DryRun {
			mode = "updated"
		}
		fmt.Printf("%d refs scanned\n", s.Scanned)
		fmt.Printf("%d had no DOI or PMID (skipped)\n", s.NoConvertible)
		fmt.Printf("%d queried\n", s.Queried)
		fmt.Printf("%d PMCIDs found (%s %d refs)\n", s.Found, mode, len(s.UpdatedIDs))
		if s.DryRun && len(s.UpdatedIDs) > 0 {
			fmt.Println()
			fmt.Println("Refs that would be updated:")
			for _, id := range s.UpdatedIDs {
				fmt.Printf("  %s\n", id)
			}
		}
		return
	}
	if err := outputJSON(s); err != nil {
		exitWithError(ExitError, "encoding JSON: %v", err)
	}
}
