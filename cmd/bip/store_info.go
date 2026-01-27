package main

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	storeCmd.AddCommand(storeInfoCmd)
}

var storeInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a store",
	Long: `Display detailed information about a store including schema, file sizes,
record count, and sync status.

Example:
  bip store info gh_activity`,
	Args: cobra.ExactArgs(1),
	RunE: runStoreInfo,
}

func runStoreInfo(cmd *cobra.Command, args []string) error {
	storeName := args[0]
	repoRoot := mustFindRepository()

	s, err := store.OpenStore(repoRoot, storeName)
	if err != nil {
		exitWithError(ExitError, "store %q not found", storeName)
	}

	info, err := s.Info()
	if err != nil {
		exitWithError(ExitError, "getting store info: %v", err)
	}

	if humanOutput {
		fmt.Printf("Store: %s\n\n", info.Name)

		fmt.Println("Files:")
		fmt.Printf("  JSONL:  %s (%s)\n", info.JSONLPath, formatBytes(info.JSONLSize))
		fmt.Printf("  DB:     %s (%s)\n", info.DBPath, formatBytes(info.DBSize))
		fmt.Printf("  Schema: %s\n", info.SchemaPath)

		fmt.Printf("\nRecords: %d\n", info.Records)

		if !info.LastSync.IsZero() {
			fmt.Printf("Last Sync: %s\n", info.LastSync.Format("2006-01-02T15:04:05Z"))
		}

		if info.InSync {
			fmt.Println("Sync Status: In sync")
		} else {
			fmt.Println("Sync Status: Out of sync (run 'bip store sync')")
		}

		if info.Schema != nil {
			fmt.Println("\nSchema:")
			for name, field := range info.Schema.Fields {
				var flags []string
				if field.Primary {
					flags = append(flags, "primary")
				}
				if field.Index {
					flags = append(flags, "index")
				}
				if field.FTS {
					flags = append(flags, "fts")
				}
				if len(field.Enum) > 0 {
					flags = append(flags, fmt.Sprintf("enum: %s", strings.Join(field.Enum, "|")))
				}

				flagStr := ""
				if len(flags) > 0 {
					flagStr = fmt.Sprintf(" (%s)", strings.Join(flags, ", "))
				}

				fmt.Printf("  %-12s %s%s\n", name, field.Type, flagStr)
			}
		}
	} else {
		outputJSON(info)
	}

	return nil
}
