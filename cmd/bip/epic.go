package main

import (
	"github.com/spf13/cobra"
)

var epicCmd = &cobra.Command{
	Use:   "epic",
	Short: "EPIC slot orchestration commands",
	Long: `Commands that support the EPIC multi-slot orchestration workflow.

These commands operate against a repo containing an .epic-config.json file
(see /bip.epic skill docs) and the .epic-status.json files written by each
slot/clone/worktree.`,
}

func init() {
	rootCmd.AddCommand(epicCmd)
}
