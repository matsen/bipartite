package main

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Get or set configuration values",
	Long: `Get or set configuration values.

Usage:
  bp config                          # Show all config
  bp config pdf-root                 # Get specific value
  bp config pdf-root /path/to/pdfs   # Set value
  bp config pdf-reader skim          # Set PDF reader

Keys:
  pdf-root    Path to PDF folder (e.g., ~/Google Drive/Paperpile)
  pdf-reader  PDF reader preference (system, skim, zathura, evince, okular)`,
	Args: cobra.MaximumNArgs(2),
	RunE: runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	cfg := mustLoadConfig(repoRoot)

	// No args: show all config
	if len(args) == 0 {
		if humanOutput {
			fmt.Printf("pdf-root:   %s\n", cfg.PDFRoot)
			fmt.Printf("pdf-reader: %s\n", cfg.PDFReader)
		} else {
			outputJSON(ConfigResponse{
				PDFRoot:   cfg.PDFRoot,
				PDFReader: cfg.PDFReader,
			})
		}
		return nil
	}

	// Convert key format (pdf-root -> PDFRoot)
	key := args[0]
	normalizedKey := normalizeKey(key)

	// One arg: get specific value
	if len(args) == 1 {
		switch normalizedKey {
		case "pdf-root":
			if humanOutput {
				fmt.Println(cfg.PDFRoot)
			} else {
				outputJSON(map[string]string{"pdf_root": cfg.PDFRoot})
			}
		case "pdf-reader":
			if humanOutput {
				fmt.Println(cfg.PDFReader)
			} else {
				outputJSON(map[string]string{"pdf_reader": cfg.PDFReader})
			}
		default:
			exitWithError(ExitError, "unknown configuration key: %s", key)
		}
		return nil
	}

	// Two args: set value
	value := args[1]

	switch normalizedKey {
	case "pdf-root":
		// Expand ~ to home directory and validate
		expandedValue := config.ExpandPath(value)

		if err := config.ValidatePDFRoot(expandedValue); err != nil {
			exitWithError(ExitConfigError, "%v", err)
		}
		cfg.PDFRoot = expandedValue

	case "pdf-reader":
		if err := config.ValidatePDFReader(value); err != nil {
			exitWithError(ExitError, "%v", err)
		}
		cfg.PDFReader = value

	default:
		exitWithError(ExitError, "unknown configuration key: %s", key)
	}

	// Save config
	if err := cfg.Save(repoRoot); err != nil {
		exitWithError(ExitError, "saving config: %v", err)
	}

	// Output success
	if humanOutput {
		fmt.Printf("Updated %s to %s\n", key, value)
	} else {
		outputJSON(UpdateResponse{
			Status: "updated",
			Key:    normalizedKey,
			Value:  value,
		})
	}

	return nil
}

// normalizeKey converts key formats (pdf-root, pdf_root, PDFRoot) to consistent format
func normalizeKey(key string) string {
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", "-")
	return key
}
