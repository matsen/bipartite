package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"github.com/matsen/bipartite/internal/flow/board"
	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Manage GitHub project boards",
	Long: `Manage GitHub project boards.

Boards are configured in sources.json under the "boards" key.
Requires nexus_path configured in ~/.config/bip/config.json.`,
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
	Use:   "list [board]",
	Short: "List board items by status",
	Long: `List items on project boards grouped by status.

By default, shows all configured boards. Provide a board key to show only that board.

Examples:
  bip board list                     # Show all boards
  bip board list matsengrp/30        # Show only board 30
  bip board list --status "In Progress"  # Filter by status`,
	Args: cobra.MaximumNArgs(1),
	Run:  runBoardList,
}

func init() {
	boardListCmd.Flags().StringVar(&boardListStatus, "status", "", "Filter by status")
	boardListCmd.Flags().StringVar(&boardListLabel, "label", "", "Filter by label")
	boardListCmd.Flags().BoolVar(&boardListJSON, "json", false, "Output as JSON")
}

func runBoardList(cmd *cobra.Command, args []string) {
	nexusPath := config.MustGetNexusPath()

	var boards []string
	if len(args) > 0 {
		// Single board specified
		boards = []string{args[0]}
	} else {
		// All boards
		var err error
		boards, err = flow.GetAllBoards(nexusPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Get channel names for boards (for display)
	boardsMapping, _ := flow.GetBoardsMapping(nexusPath)
	channelForBoard := make(map[string]string)
	for channel, boardKey := range boardsMapping {
		channelForBoard[boardKey] = channel
	}

	allItems := make(map[string][]flow.BoardItem)
	for _, key := range boards {
		items, err := board.ListBoardItems(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list board %s: %v\n", key, err)
			continue
		}

		// Filter by status if specified
		if boardListStatus != "" {
			var filtered []flow.BoardItem
			for _, item := range items {
				if item.Status == boardListStatus {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}

		allItems[key] = items
	}

	// Output
	if boardListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(allItems)
		return
	}

	// Print each board
	for _, key := range boards {
		items := allItems[key]

		// Board header with channel name if available
		header := key
		if channel, ok := channelForBoard[key]; ok {
			header = fmt.Sprintf("%s (%s)", key, channel)
		}
		fmt.Printf("## %s\n\n", header)

		if len(items) == 0 {
			fmt.Println("No items on board.")
			fmt.Println()
			continue
		}

		// Group by status
		byStatus := make(map[string][]flow.BoardItem)
		statusOrder := []string{}
		for _, item := range items {
			if _, seen := byStatus[item.Status]; !seen {
				statusOrder = append(statusOrder, item.Status)
			}
			byStatus[item.Status] = append(byStatus[item.Status], item)
		}

		for _, status := range statusOrder {
			statusItems := byStatus[status]
			fmt.Printf("### %s\n", status)
			for _, item := range statusItems {
				fmt.Printf("- %s#%d: %s\n", item.Content.Repository, item.Content.Number, item.Title)
			}
			fmt.Println()
		}
	}
}

// board add
var (
	boardAddStatus string
	boardAddLabel  string
	boardAddTo     string // explicit board override
)

var boardAddCmd = &cobra.Command{
	Use:   "add <repo#number>",
	Short: "Add issue/PR to board",
	Long: `Add an issue or pull request to a project board.

The board is automatically resolved from the repo's channel mapping in sources.json.
Use --to to specify a board explicitly.

Examples:
  bip board add dasm2-experiments#207    # Resolves board from dasm2 channel
  bip board add netam#171                # Same board (both in dasm2 channel)
  bip board add loris-experiments#15     # Different board (loris channel)
  bip board add myrepo#42 --to matsengrp/30  # Explicit board`,
	Args: cobra.ExactArgs(1),
	Run:  runBoardAdd,
}

func init() {
	boardAddCmd.Flags().StringVar(&boardAddStatus, "status", "", "Initial status")
	boardAddCmd.Flags().StringVar(&boardAddLabel, "label", "", "Label to apply")
	boardAddCmd.Flags().StringVar(&boardAddTo, "to", "", "Explicit board (owner/number), overrides channel resolution")
}

func runBoardAdd(cmd *cobra.Command, args []string) {
	nexusPath := config.MustGetNexusPath()

	// Parse repo#number format
	repo, issueNum, err := parseRepoIssue(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Resolve board key
	var key string
	if boardAddTo != "" {
		key = boardAddTo
	} else {
		key, err = flow.GetBoardForRepo(nexusPath, repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Hint: Use --to to specify a board explicitly\n")
			os.Exit(1)
		}
	}

	err = board.AddIssueToBoard(key, issueNum, repo, boardAddStatus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	msg := fmt.Sprintf("Added %s#%d to board %s", repo, issueNum, key)
	if boardAddStatus != "" {
		msg += fmt.Sprintf(" with status '%s'", boardAddStatus)
	}
	fmt.Println(msg)
}

// board move
var (
	boardMoveStatus string
	boardMoveTo     string
)

var boardMoveCmd = &cobra.Command{
	Use:   "move <repo#number> --status <status>",
	Short: "Move item to different status",
	Long: `Move a board item to a different status.

Examples:
  bip board move dasm2-experiments#207 --status done
  bip board move netam#171 --status "in progress"`,
	Args: cobra.ExactArgs(1),
	Run:  runBoardMove,
}

func init() {
	boardMoveCmd.Flags().StringVar(&boardMoveStatus, "status", "", "New status (required)")
	boardMoveCmd.Flags().StringVar(&boardMoveTo, "to", "", "Explicit board (owner/number), overrides channel resolution")
	boardMoveCmd.MarkFlagRequired("status")
}

func runBoardMove(cmd *cobra.Command, args []string) {
	nexusPath := config.MustGetNexusPath()

	repo, issueNum, err := parseRepoIssue(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var key string
	if boardMoveTo != "" {
		key = boardMoveTo
	} else {
		key, err = flow.GetBoardForRepo(nexusPath, repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Hint: Use --to to specify a board explicitly\n")
			os.Exit(1)
		}
	}

	err = board.MoveItem(key, issueNum, boardMoveStatus, repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Moved %s#%d to '%s'\n", repo, issueNum, boardMoveStatus)
}

// board remove
var boardRemoveTo string

var boardRemoveCmd = &cobra.Command{
	Use:   "remove <repo#number>",
	Short: "Remove issue/PR from board",
	Long: `Remove an issue or pull request from its project board.

Examples:
  bip board remove dasm2-experiments#207
  bip board remove netam#171`,
	Args: cobra.ExactArgs(1),
	Run:  runBoardRemove,
}

func init() {
	boardRemoveCmd.Flags().StringVar(&boardRemoveTo, "to", "", "Explicit board (owner/number), overrides channel resolution")
}

func runBoardRemove(cmd *cobra.Command, args []string) {
	nexusPath := config.MustGetNexusPath()

	repo, issueNum, err := parseRepoIssue(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var key string
	if boardRemoveTo != "" {
		key = boardRemoveTo
	} else {
		key, err = flow.GetBoardForRepo(nexusPath, repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "Hint: Use --to to specify a board explicitly\n")
			os.Exit(1)
		}
	}

	err = board.RemoveIssueFromBoard(key, issueNum, repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed %s#%d from board %s\n", repo, issueNum, key)
}

// board refresh-cache
var boardRefreshCmd = &cobra.Command{
	Use:   "refresh-cache",
	Short: "Refresh cached board metadata",
	Run:   runBoardRefresh,
}

func runBoardRefresh(cmd *cobra.Command, args []string) {
	nexusPath := config.MustGetNexusPath()
	key := resolveBoardKey(nexusPath)
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

func resolveBoardKey(nexusPath string) string {
	if boardKey != "" {
		return boardKey
	}

	defaultBoard, err := flow.GetDefaultBoard(nexusPath)
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

// parseRepoIssue parses "repo#N" or "org/repo#N" format.
// For short names like "dasm2-experiments#207", it expands to "matsengrp/dasm2-experiments".
func parseRepoIssue(s string) (repo string, number int, err error) {
	// Find the # separator
	idx := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '#' {
			idx = i
			break
		}
	}
	if idx == -1 || idx == 0 || idx == len(s)-1 {
		return "", 0, fmt.Errorf("invalid format %q: expected repo#number (e.g., dasm2-experiments#207)", s)
	}

	repo = s[:idx]
	numStr := s[idx+1:]

	// Parse number
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return "", 0, fmt.Errorf("invalid issue number in %q", s)
		}
		number = number*10 + int(c-'0')
	}
	if number == 0 {
		return "", 0, fmt.Errorf("invalid issue number in %q", s)
	}

	// If repo doesn't contain '/', assume matsengrp
	if !containsSlash(repo) {
		repo = "matsengrp/" + repo
	}

	return repo, number, nil
}

func containsSlash(s string) bool {
	for _, c := range s {
		if c == '/' {
			return true
		}
	}
	return false
}
