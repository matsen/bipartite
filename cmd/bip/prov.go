package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/prov"
	"github.com/spf13/cobra"
)

var (
	provMain    string
	provNoFetch bool
)

var provCmd = &cobra.Command{
	Use:   "prov",
	Short: "Manuscript provenance ledger commands",
}

var provCheckCmd = &cobra.Command{
	Use:   "check [paper-dir]",
	Short: "Check %PROV tags and provenance.yaml against the code repos",
	Long: `Check every %PROV[id] tag in the main TeX file (and every file it \inputs)
against provenance.yaml, and every ledger entry against git objects of the
repos it names. Each repo is read from a bare clone bip keeps under
$NEXUS_PATH/.bipartite/cache/prov/, created on first use and fetched first;
no other checkout is read or fetched.

Findings are error, review, or info. The exit code is nonzero on any error.
paper-dir holds provenance.yaml and the main TeX file; it defaults to the
current directory and must be a git work tree.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProvCheck,
}

func init() {
	provCheckCmd.Flags().StringVar(&provMain, "main", "main.tex", "main TeX file, relative to paper-dir")
	provCheckCmd.Flags().BoolVar(&provNoFetch, "no-fetch", false, "skip git fetch of each cached repo")
	provCmd.AddCommand(provCheckCmd)
	rootCmd.AddCommand(provCmd)
}

func runProvCheck(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	nexus := config.MustGetNexusPath()
	findings, err := prov.Check(prov.Options{
		PaperDir: dir,
		Main:     provMain,
		Ledger:   "provenance.yaml",
		Fetch:    !provNoFetch,
		Resolve: func(repo string) (string, error) {
			return prov.CacheClone(filepath.Join(config.CachePath(nexus), "prov"), repo)
		},
	})
	if err != nil {
		exitWithError(ExitDataError, "%v", err)
	}
	counts := map[string]int{}
	for _, f := range findings {
		counts[f.Level]++
	}
	if humanOutput {
		for _, f := range findings {
			loc := ""
			if f.File != "" {
				loc = fmt.Sprintf(" (%s:%d)", f.File, f.Line)
			}
			fmt.Printf("%-6s %s: %s%s\n", f.Level, f.ID, f.Message, loc)
		}
		fmt.Printf("%d error, %d review, %d info\n", counts[prov.LevelError], counts[prov.LevelReview], counts[prov.LevelInfo])
	} else {
		if findings == nil {
			findings = []prov.Finding{}
		}
		outputJSON(findings)
	}
	if counts[prov.LevelError] > 0 {
		os.Exit(ExitError)
	}
	return nil
}
