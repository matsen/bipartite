package board

import (
	"fmt"

	"github.com/matsen/bipartite/internal/flow"
)

// SyncResult contains the result of a board sync operation.
type SyncResult struct {
	MissingFromBoard []flow.BeadGitHubRef // P0 beads not on board
	NotInBeads       []flow.BoardItem     // Board issues without P0 bead
	FixesApplied     []string             // Messages about fixes applied
}

// SyncBoardWithBeads compares P0 beads with board items and optionally fixes mismatches.
func SyncBoardWithBeads(boardKey string, fix bool) (*SyncResult, error) {
	p0Beads, err := flow.GetP0BeadsWithGitHubRefs()
	if err != nil {
		return nil, fmt.Errorf("loading P0 beads: %w", err)
	}

	boardIssues, err := ListBoardItems(boardKey)
	if err != nil {
		return nil, fmt.Errorf("listing board items: %w", err)
	}

	// Build sets for comparison
	p0Refs := make(map[string]flow.BeadGitHubRef)
	for _, b := range p0Beads {
		key := fmt.Sprintf("%s#%d", b.Repo, b.IssueNumber)
		p0Refs[key] = b
	}

	boardRefs := make(map[string]flow.BoardItem)
	for _, i := range boardIssues {
		if i.Content.Type != "Issue" {
			continue
		}
		key := fmt.Sprintf("%s#%d", i.Content.Repository, i.Content.Number)
		boardRefs[key] = i
	}

	result := &SyncResult{}

	// Find P0 beads not on board
	for key, bead := range p0Refs {
		if _, ok := boardRefs[key]; !ok {
			result.MissingFromBoard = append(result.MissingFromBoard, bead)
		}
	}

	// Find board issues not in P0 beads
	for key, issue := range boardRefs {
		if _, ok := p0Refs[key]; !ok {
			result.NotInBeads = append(result.NotInBeads, issue)
		}
	}

	// Apply fixes if requested
	if fix {
		for _, bead := range result.MissingFromBoard {
			err := AddIssueToBoard(boardKey, bead.IssueNumber, bead.Repo, "next")
			if err != nil {
				result.FixesApplied = append(result.FixesApplied,
					fmt.Sprintf("Failed to add #%d: %v", bead.IssueNumber, err))
			} else {
				result.FixesApplied = append(result.FixesApplied,
					fmt.Sprintf("Added #%d to board", bead.IssueNumber))
			}
		}
	}

	return result, nil
}

// PrintSyncReport prints a sync report to stdout.
func PrintSyncReport(result *SyncResult, boardKey string) {
	fmt.Printf("## Board Sync: %s\n\n", boardKey)

	if len(result.MissingFromBoard) > 0 {
		fmt.Printf("**P0 beads not on board** (%d):\n", len(result.MissingFromBoard))
		for _, bead := range result.MissingFromBoard {
			title := bead.Title
			if len(title) > 50 {
				title = title[:50] + "..."
			}
			fmt.Printf("  - %s#%d: %s\n", bead.Repo, bead.IssueNumber, title)
			fmt.Printf("    Bead: %s\n", bead.BeadID)
		}
		fmt.Println()
	}

	if len(result.NotInBeads) > 0 {
		fmt.Printf("**Board issues without P0 bead** (%d):\n", len(result.NotInBeads))
		for _, issue := range result.NotInBeads {
			title := issue.Title
			if len(title) > 50 {
				title = title[:50] + "..."
			}
			fmt.Printf("  - %s#%d: %s\n", issue.Content.Repository, issue.Content.Number, title)
			fmt.Printf("    Status: %s\n", issue.Status)
		}
		fmt.Println()
	}

	if len(result.FixesApplied) > 0 {
		fmt.Println("**Fixes applied:**")
		for _, fix := range result.FixesApplied {
			fmt.Printf("  - %s\n", fix)
		}
		fmt.Println()
	}

	if len(result.MissingFromBoard) == 0 && len(result.NotInBeads) == 0 {
		fmt.Println("Board and P0 beads are in sync!")
	}
}
