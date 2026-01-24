package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/matsen/bipartite/internal/flow/tree"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Generate interactive HTML tree view of beads issues",
	Long: `Generate an interactive HTML tree view of beads issues.

The tree is built from .beads/issues.jsonl in the current directory.
Use --since to highlight recently created beads.`,
	Run: runTree,
}

var (
	treeSince  string
	treeOutput string
	treeOpen   bool
)

func init() {
	rootCmd.AddCommand(treeCmd)

	treeCmd.Flags().StringVar(&treeSince, "since", "", "Highlight beads created after this date (YYYY-MM-DD or ISO format)")
	treeCmd.Flags().StringVarP(&treeOutput, "output", "o", "", "Output file path (default: stdout)")
	treeCmd.Flags().BoolVar(&treeOpen, "open", false, "Open in browser after generating")
}

func runTree(cmd *cobra.Command, args []string) {
	// Load beads
	beads, err := flow.LoadBeads()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse since
	since, err := tree.ParseSince(treeSince)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Generate HTML
	html := tree.GenerateHTML(beads, since)

	// Output
	if treeOutput != "" {
		// Write to file
		if err := os.WriteFile(treeOutput, []byte(html), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Written to %s\n", treeOutput)

		if treeOpen {
			absPath, _ := filepath.Abs(treeOutput)
			openBrowser("file://" + absPath)
		}
	} else if treeOpen {
		// Write to temp file and open
		tmpFile, err := os.CreateTemp("", "beads-tree-*.html")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
			os.Exit(1)
		}

		if _, err := tmpFile.WriteString(html); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			fmt.Fprintf(os.Stderr, "Error writing temp file: %v\n", err)
			os.Exit(1)
		}
		tmpFile.Close()

		fmt.Printf("Opened %s\n", tmpFile.Name())
		openBrowser("file://" + tmpFile.Name())
	} else {
		// Print to stdout
		fmt.Println(html)
	}
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Fprintf(os.Stderr, "Warning: don't know how to open browser on %s\n", runtime.GOOS)
		return
	}

	cmd.Start()
}
