package main

import (
	"github.com/spf13/cobra"
)

var zoteroCmd = &cobra.Command{
	Use:   "zotero",
	Short: "Zotero library integration commands",
	Long: `Commands for syncing with Zotero's Web API.

Sync papers between your bip library and Zotero, or add new papers
to both simultaneously.

Requires zotero_api_key and zotero_user_id in ~/.config/bip/config.yml.

All commands output JSON by default for agent consumption.
Use --human flag for human-readable output.`,
}

func init() {
	rootCmd.AddCommand(zoteroCmd)
}
