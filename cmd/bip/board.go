package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/matsen/bipartite/internal/flow/board"
	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Manage GitHub project boards",
	Long: `Manage GitHub project boards.

Boards are configured in sources.json under the "boards" key.
Requires sources.json in the current directory (run from nexus directory).`,
}

// Shared flags
var boardKey string

func init() {
	rootCmd.AddCommand(boardCmd)

	// Add subcommands
	boardCmd.AddCommand(boardListCmd)
	boardCmd.AddCommand(boardAddCmd)
	boardCmd.AddCommand(boardMoveCmd)
	boardCmd.AddCommand(boardRemoveCmd)
	boardCmd.AddCommand(boardSyncCmd)
	boardCmd.AddCommand(boardRefreshCmd)

	// Shared flags
	boardCmd.PersistentFlags().StringVar(&boardKey, "board", "", "Board to use (owner/number). Defaults to first in sources.json")
}

// board list
var (
	boardListStatus string
	boardListLabel  string
	boardListJSON   bool
)

var boardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List board items by status",
	Run:   runBoardList,
}

func init() {
	boardListCmd.Flags().StringVar(&boardListStatus, "status", "", "Filter by status")
	boardListCmd.Flags().StringVar(&boardListLabel, "label", "", "Filter by label")
	boardListCmd.Flags().BoolVar(&boardListJSON, "json", false, "Output as JSON")
}

func runBoardList(cmd *cobra.Command, args []string) {
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	key := resolveBoardKey()

	items, err := board.ListBoardItems(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Filter by status
	if boardListStatus != "" {
		var filtered []flow.BoardItem
		for _, item := range items {
			if item.Status == boardListStatus {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	// Output
	if boardListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(items)
		return
	}

	// Group by status
	byStatus := make(map[string][]flow.BoardItem)
	for _, item := range items {
		byStatus[item.Status] = append(byStatus[item.Status], item)
	}

	for status, statusItems := range byStatus {
		fmt.Printf("## %s (%d)\n", status, len(statusItems))
		for _, item := range statusItems {
			fmt.Printf("  - %s#%d: %s\n", item.Content.Repository, item.Content.Number, item.Title)
		}
		fmt.Println()
	}

	if len(items) == 0 {
		fmt.Println("No items on board.")
	}
}

// board add
var (
	boardAddStatus string
	boardAddLabel  string
	boardAddRepo   string
)

var boardAddCmd = &cobra.Command{
	Use:   "add <issue-number>",
	Short: "Add issue to board",
	Args:  cobra.ExactArgs(1),
	Run:   runBoardAdd,
}

func init() {
	boardAddCmd.Flags().StringVar(&boardAddStatus, "status", "", "Initial status")
	boardAddCmd.Flags().StringVar(&boardAddLabel, "label", "", "Label to apply")
	boardAddCmd.Flags().StringVar(&boardAddRepo, "repo", "", "Repository (org/repo format, required)")
	boardAddCmd.MarkFlagRequired("repo")
}

func runBoardAdd(cmd *cobra.Command, args []string) {
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	issueNum := parseIssueNumber(args[0])
	key := resolveBoardKey()

	err := board.AddIssueToBoard(key, issueNum, boardAddRepo, boardAddStatus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	msg := fmt.Sprintf("Added #%d to board", issueNum)
	if boardAddStatus != "" {
		msg += fmt.Sprintf(" with status '%s'", boardAddStatus)
	}
	fmt.Println(msg)
}

// board move
var (
	boardMoveStatus string
	boardMoveRepo   string
)

var boardMoveCmd = &cobra.Command{
	Use:   "move <issue-number>",
	Short: "Move item to different status",
	Args:  cobra.ExactArgs(1),
	Run:   runBoardMove,
}

func init() {
	boardMoveCmd.Flags().StringVar(&boardMoveStatus, "status", "", "New status (required)")
	boardMoveCmd.Flags().StringVar(&boardMoveRepo, "repo", "", "Repository (org/repo format, required)")
	boardMoveCmd.MarkFlagRequired("status")
	boardMoveCmd.MarkFlagRequired("repo")
}

func runBoardMove(cmd *cobra.Command, args []string) {
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	issueNum := parseIssueNumber(args[0])
	key := resolveBoardKey()

	err := board.MoveItem(key, issueNum, boardMoveStatus, boardMoveRepo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Moved #%d to '%s'\n", issueNum, boardMoveStatus)
}

// board remove
var boardRemoveRepo string

var boardRemoveCmd = &cobra.Command{
	Use:   "remove <issue-number>",
	Short: "Remove issue from board",
	Args:  cobra.ExactArgs(1),
	Run:   runBoardRemove,
}

func init() {
	boardRemoveCmd.Flags().StringVar(&boardRemoveRepo, "repo", "", "Repository (org/repo format, required)")
	boardRemoveCmd.MarkFlagRequired("repo")
}

func runBoardRemove(cmd *cobra.Command, args []string) {
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	issueNum := parseIssueNumber(args[0])
	key := resolveBoardKey()

	err := board.RemoveIssueFromBoard(key, issueNum, boardRemoveRepo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed #%d from board\n", issueNum)
}

// board sync
var boardSyncFix bool

var boardSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync board with beads (report mismatches)",
	Run:   runBoardSync,
}

func init() {
	boardSyncCmd.Flags().BoolVar(&boardSyncFix, "fix", false, "Auto-fix mismatches")
}

func runBoardSync(cmd *cobra.Command, args []string) {
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	key := resolveBoardKey()

	result, err := board.SyncBoardWithBeads(key, boardSyncFix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	board.PrintSyncReport(result, key)
}

// board refresh-cache
var boardRefreshCmd = &cobra.Command{
	Use:   "refresh-cache",
	Short: "Refresh cached board metadata",
	Run:   runBoardRefresh,
}

func runBoardRefresh(cmd *cobra.Command, args []string) {
	if err := flow.ValidateNexusDirectory(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	key := resolveBoardKey()
	owner, projectNum, err := board.ParseBoardKey(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	projectID, err := board.FetchProjectID(owner, projectNum)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	meta, err := board.FetchProjectFields(projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Cache refreshed. Found %d status options.\n", len(meta.StatusOptions))
}

// Helper functions

func resolveBoardKey() string {
	if boardKey != "" {
		return boardKey
	}

	defaultBoard, err := flow.GetDefaultBoard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return defaultBoard
}

func parseIssueNumber(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	if n == 0 {
		fmt.Fprintf(os.Stderr, "Error: invalid issue number: %s\n", s)
		os.Exit(1)
	}
	return n
}
