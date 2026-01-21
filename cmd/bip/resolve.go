package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/conflict"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	resolveDryRun      bool
	resolveInteractive bool
)

func init() {
	rootCmd.AddCommand(resolveCmd)
	resolveCmd.Flags().BoolVar(&resolveDryRun, "dry-run", false, "Show proposed resolution without modifying files")
	resolveCmd.Flags().BoolVar(&resolveInteractive, "interactive", false, "Prompt for true conflicts that cannot be auto-resolved")
}

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Domain-aware conflict resolution for refs.jsonl",
	Long: `Resolve git merge conflicts in refs.jsonl using domain knowledge about papers.

Git sees JSON as opaque blobs, but bip understands:
- DOI is a unique identifier for matching papers
- One version might have metadata the other lacks
- A paper with more complete metadata is probably better
- Author lists can be merged if one is longer

Examples:
  bip resolve              # Auto-resolve and write result
  bip resolve --dry-run    # Preview what would happen
  bip resolve --interactive # Prompt for true conflicts
  bip resolve --dry-run --human  # Human-readable preview`,
	RunE: runResolve,
}

func runResolve(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Read the refs file
	content, err := os.ReadFile(refsPath)
	if err != nil {
		if os.IsNotExist(err) {
			exitWithError(ExitDataError, "refs.jsonl not found at %s", refsPath)
		}
		exitWithError(ExitError, "reading refs.jsonl: %v", err)
	}

	// Check if file is empty
	if len(content) == 0 {
		exitWithError(ExitDataError, "refs.jsonl is empty")
	}

	// Parse the file
	parseResult, err := conflict.ParseString(string(content))
	if err != nil {
		if parseErr, ok := err.(conflict.ParseError); ok {
			exitWithError(ExitDataError, "parsing refs.jsonl: %s", parseErr.Error())
		}
		exitWithError(ExitError, "parsing refs.jsonl: %v", err)
	}

	// Check if there are any conflicts
	if !parseResult.HasConflicts() {
		result := ResolveResult{
			Resolved: countCleanRefs(parseResult.CleanLines),
		}
		if humanOutput {
			fmt.Println("No conflicts detected in refs.jsonl.")
		} else {
			outputJSON(result)
		}
		return nil
	}

	// Process conflicts
	result, resolvedRefs, hasUnresolved := resolveConflicts(parseResult, refsPath)

	if resolveDryRun {
		// Dry run - just show what would happen
		if humanOutput {
			printResolveResultHuman(result, true)
		} else {
			outputJSON(result)
		}
		return nil
	}

	// Check for unresolved conflicts
	if hasUnresolved && !resolveInteractive {
		if humanOutput {
			printResolveResultHuman(result, false)
			fmt.Fprintln(os.Stderr, "\nerror: unresolvable conflicts require --interactive flag")
		} else {
			outputJSON(result)
		}
		os.Exit(ExitError)
	}

	// Write resolved file
	if err := writeResolvedFile(refsPath, parseResult, resolvedRefs); err != nil {
		exitWithError(ExitError, "writing resolved file: %v", err)
	}

	if humanOutput {
		printResolveResultHuman(result, false)
		fmt.Printf("\nResolved refs.jsonl written to %s\n", refsPath)
	} else {
		outputJSON(result)
	}

	return nil
}

// resolveConflicts processes all conflict regions and returns the result.
func resolveConflicts(parseResult *conflict.ParseResult, refsPath string) (ResolveResult, []reference.Reference, bool) {
	var result ResolveResult
	var resolvedRefs []reference.Reference
	hasUnresolved := false

	for _, region := range parseResult.Conflicts {
		matchResult := conflict.MatchPapers(region)

		// Process matched papers
		for _, match := range matchResult.Matches {
			plan := conflict.Resolve(match)
			op := ResolveOp{
				PaperID: plan.PaperID,
				DOI:     plan.DOI,
				Action:  string(plan.Action),
				Reason:  plan.Reason,
			}
			result.Operations = append(result.Operations, op)

			if plan.Action == conflict.ActionConflict {
				// True conflict - needs interactive resolution
				if resolveInteractive {
					// Prompt for each conflict
					resolved := promptForConflict(match, plan)
					resolvedRefs = append(resolvedRefs, resolved)
					result.Merged++
				} else {
					hasUnresolved = true
					fields := make([]string, len(plan.Conflicts))
					for i, c := range plan.Conflicts {
						fields[i] = c.FieldName
					}
					result.Unresolved = append(result.Unresolved, UnresolvedInfo{
						PaperID: plan.PaperID,
						DOI:     plan.DOI,
						Fields:  fields,
					})
				}
			} else {
				resolved := conflict.ApplyResolution(match, plan)
				resolvedRefs = append(resolvedRefs, resolved)
				if plan.Action == conflict.ActionMerge {
					result.Merged++
				}
			}
		}

		// Add papers only on ours side
		for _, ref := range matchResult.OursOnly {
			resolvedRefs = append(resolvedRefs, ref)
			result.OursPapers++
			result.Operations = append(result.Operations, ResolveOp{
				PaperID: ref.ID,
				DOI:     ref.DOI,
				Action:  string(conflict.ActionAddOurs),
				Reason:  "paper only in ours",
			})
		}

		// Add papers only on theirs side
		for _, ref := range matchResult.TheirsOnly {
			resolvedRefs = append(resolvedRefs, ref)
			result.TheirsPapers++
			result.Operations = append(result.Operations, ResolveOp{
				PaperID: ref.ID,
				DOI:     ref.DOI,
				Action:  string(conflict.ActionAddTheirs),
				Reason:  "paper only in theirs",
			})
		}
	}

	// Count clean refs
	result.Resolved = countCleanRefs(parseResult.CleanLines) + len(resolvedRefs)

	return result, resolvedRefs, hasUnresolved
}

// countCleanRefs counts the number of valid reference lines in clean lines.
func countCleanRefs(cleanLines []conflict.CleanLine) int {
	count := 0
	for _, line := range cleanLines {
		content := strings.TrimSpace(line.Content)
		if content != "" {
			var ref reference.Reference
			if json.Unmarshal([]byte(content), &ref) == nil {
				count++
			}
		}
	}
	return count
}

// promptForConflict prompts the user for each conflicting field.
func promptForConflict(match conflict.PaperMatch, plan conflict.ResolutionPlan) reference.Reference {
	// Start with merged base (non-conflicting fields)
	resolved, _ := conflict.MergeReferences(match.Ours, match.Theirs)
	reader := bufio.NewReader(os.Stdin)

	totalConflicts := len(plan.Conflicts)
	for i, fc := range plan.Conflicts {
		fmt.Printf("\nResolving conflict %d of %d for paper %s...\n", i+1, totalConflicts, plan.PaperID)
		fmt.Printf("Conflict in field '%s':\n", fc.FieldName)
		fmt.Printf("  [1] ours:   %q\n", fc.OursValue)
		fmt.Printf("  [2] theirs: %q\n", fc.TheirsValue)

		for {
			fmt.Print("Enter choice [1/2]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "1" {
				applyFieldChoice(&resolved, fc.FieldName, match.Ours)
				break
			} else if input == "2" {
				applyFieldChoice(&resolved, fc.FieldName, match.Theirs)
				break
			} else {
				fmt.Println("Invalid choice. Please enter 1 or 2.")
			}
		}
	}

	return resolved
}

// applyFieldChoice applies the chosen field value from source to target.
func applyFieldChoice(target *reference.Reference, fieldName string, source reference.Reference) {
	switch fieldName {
	case "title":
		target.Title = source.Title
	case "abstract":
		target.Abstract = source.Abstract
	case "venue":
		target.Venue = source.Venue
	case "pdf_path":
		target.PDFPath = source.PDFPath
	case "supersedes":
		target.Supersedes = source.Supersedes
	case "authors":
		target.Authors = source.Authors
	}
}

// writeResolvedFile writes the resolved refs.jsonl file.
func writeResolvedFile(path string, parseResult *conflict.ParseResult, resolvedRefs []reference.Reference) error {
	// Build the final list of all references
	var allRefs []reference.Reference

	// Track which conflict regions we've output
	resolvedIdx := 0

	// Process line by line, inserting resolved refs at conflict positions
	prevConflictEnd := 0
	for _, region := range parseResult.Conflicts {
		// Output clean lines before this conflict
		for _, cl := range parseResult.CleanLines {
			if cl.LineNum > prevConflictEnd && cl.LineNum < region.StartLine {
				content := strings.TrimSpace(cl.Content)
				if content != "" {
					var ref reference.Reference
					if err := json.Unmarshal([]byte(content), &ref); err == nil {
						allRefs = append(allRefs, ref)
					}
				}
			}
		}

		// Output resolved refs for this conflict region
		matchResult := conflict.MatchPapers(region)
		numRefsInRegion := len(matchResult.Matches) + len(matchResult.OursOnly) + len(matchResult.TheirsOnly)
		for i := 0; i < numRefsInRegion && resolvedIdx < len(resolvedRefs); i++ {
			allRefs = append(allRefs, resolvedRefs[resolvedIdx])
			resolvedIdx++
		}

		prevConflictEnd = region.EndLine
	}

	// Output remaining clean lines after last conflict
	for _, cl := range parseResult.CleanLines {
		if cl.LineNum > prevConflictEnd {
			content := strings.TrimSpace(cl.Content)
			if content != "" {
				var ref reference.Reference
				if err := json.Unmarshal([]byte(content), &ref); err == nil {
					allRefs = append(allRefs, ref)
				}
			}
		}
	}

	return storage.WriteAll(path, allRefs)
}

// printResolveResultHuman prints the resolve result in human-readable format.
func printResolveResultHuman(result ResolveResult, isDryRun bool) {
	if isDryRun {
		fmt.Println("Dry run - no changes made")
		fmt.Println()
	}

	fmt.Printf("Resolution summary:\n")
	fmt.Printf("  Total papers resolved: %d\n", result.Resolved)
	if result.Merged > 0 {
		fmt.Printf("  Merged (complementary): %d\n", result.Merged)
	}
	if result.OursPapers > 0 {
		fmt.Printf("  Added from ours: %d\n", result.OursPapers)
	}
	if result.TheirsPapers > 0 {
		fmt.Printf("  Added from theirs: %d\n", result.TheirsPapers)
	}

	if len(result.Operations) > 0 {
		fmt.Println()
		fmt.Println("Operations:")
		for _, op := range result.Operations {
			fmt.Printf("  %s: %s (%s)\n", op.PaperID, op.Action, op.Reason)
		}
	}

	if len(result.Unresolved) > 0 {
		fmt.Println()
		fmt.Printf("Unresolved conflicts (%d):\n", len(result.Unresolved))
		for _, u := range result.Unresolved {
			fmt.Printf("  %s: conflicts on %s\n", u.PaperID, strings.Join(u.Fields, ", "))
		}
		fmt.Println()
		fmt.Println("Use --interactive to resolve these conflicts manually.")
	}
}
