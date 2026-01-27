package main

import (
	"github.com/spf13/cobra"
)

var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Generic JSONL + SQLite store commands",
	Long: `Manage generic data stores backed by JSONL files with SQLite query indexes.

Each store is defined by a JSON schema that specifies field types, indexes,
and full-text search capabilities. JSONL is the source of truth, SQLite
provides fast queries.

All commands support --json flag for agent consumption.`,
}

func init() {
	rootCmd.AddCommand(storeCmd)
}
