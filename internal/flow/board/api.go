// Package board provides GitHub project board management functionality.
package board

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/matsen/bipartite/internal/flow"
)

// ParseBoardKey parses a board key like "matsengrp/30" into (owner, number).
func ParseBoardKey(boardKey string) (owner, number string, err error) {
	parts := strings.Split(boardKey, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid board key: %s (expected owner/number)", boardKey)
	}
	return parts[0], parts[1], nil
}

// FetchProjectID fetches the GraphQL node ID for a project.
func FetchProjectID(owner, projectNum string) (string, error) {
	// Try org first
	query := `
	query($owner: String!, $number: Int!) {
	  organization(login: $owner) {
	    projectV2(number: $number) {
	      id
	    }
	  }
	}`

	data, err := flow.GHGraphQL(query, map[string]string{
		"owner":  owner,
		"number": projectNum,
	})
	if err == nil {
		var result struct {
			Data struct {
				Organization struct {
					ProjectV2 struct {
						ID string `json:"id"`
					} `json:"projectV2"`
				} `json:"organization"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &result) == nil && result.Data.Organization.ProjectV2.ID != "" {
			return result.Data.Organization.ProjectV2.ID, nil
		}
	}

	// Try user
	query = `
	query($owner: String!, $number: Int!) {
	  user(login: $owner) {
	    projectV2(number: $number) {
	      id
	    }
	  }
	}`

	data, err = flow.GHGraphQL(query, map[string]string{
		"owner":  owner,
		"number": projectNum,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			User struct {
				ProjectV2 struct {
					ID string `json:"id"`
				} `json:"projectV2"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if result.Data.User.ProjectV2.ID == "" {
		return "", fmt.Errorf("project not found: %s/%s", owner, projectNum)
	}

	return result.Data.User.ProjectV2.ID, nil
}

// BoardMetadata contains cached board information.
type BoardMetadata struct {
	ProjectID     string            `json:"project_id"`
	Owner         string            `json:"owner"`
	ProjectNum    string            `json:"project_num"`
	FieldIDs      map[string]string `json:"field_ids"`
	StatusOptions map[string]string `json:"status_options"` // lowercase status -> option ID
	ItemIDs       map[string]string `json:"item_ids"`       // "repo#number" -> item ID
}

// FetchProjectFields fetches field IDs and status options for a project.
func FetchProjectFields(projectID string) (*BoardMetadata, error) {
	query := `
	query($projectId: ID!) {
	  node(id: $projectId) {
	    ... on ProjectV2 {
	      fields(first: 20) {
	        nodes {
	          ... on ProjectV2Field {
	            id
	            name
	          }
	          ... on ProjectV2SingleSelectField {
	            id
	            name
	            options {
	              id
	              name
	            }
	          }
	        }
	      }
	    }
	  }
	}`

	data, err := flow.GHGraphQL(query, map[string]string{"projectId": projectID})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Node struct {
				Fields struct {
					Nodes []struct {
						ID      string `json:"id"`
						Name    string `json:"name"`
						Options []struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"options,omitempty"`
					} `json:"nodes"`
				} `json:"fields"`
			} `json:"node"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	meta := &BoardMetadata{
		ProjectID:     projectID,
		FieldIDs:      make(map[string]string),
		StatusOptions: make(map[string]string),
		ItemIDs:       make(map[string]string),
	}

	for _, field := range result.Data.Node.Fields.Nodes {
		if field.Name != "" && field.ID != "" {
			meta.FieldIDs[field.Name] = field.ID
		}

		if field.Name == "Status" && len(field.Options) > 0 {
			for _, opt := range field.Options {
				meta.StatusOptions[strings.ToLower(opt.Name)] = opt.ID
			}
		}
	}

	return meta, nil
}

// ListBoardItems lists all items on a board using gh CLI.
func ListBoardItems(boardKey string) ([]flow.BoardItem, error) {
	owner, projectNum, err := ParseBoardKey(boardKey)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("gh", "project", "item-list", projectNum, "--owner", owner, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("listing board items: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var result struct {
		Items []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Status  string `json:"status"`
			Content struct {
				Type       string `json:"type"`
				Repository string `json:"repository"`
				Number     int    `json:"number"`
			} `json:"content"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	var items []flow.BoardItem
	for _, item := range result.Items {
		items = append(items, flow.BoardItem{
			ID:     item.ID,
			Title:  item.Title,
			Status: item.Status,
			Content: flow.BoardContent{
				Type:       item.Content.Type,
				Repository: item.Content.Repository,
				Number:     item.Content.Number,
			},
		})
	}

	return items, nil
}

// AddIssueToBoard adds an issue to a board.
func AddIssueToBoard(boardKey string, issueNumber int, repo string, status string) error {
	owner, projectNum, err := ParseBoardKey(boardKey)
	if err != nil {
		return err
	}

	issueURL := fmt.Sprintf("https://github.com/%s/issues/%d", repo, issueNumber)

	// Try gh project item-add first
	cmd := exec.Command("gh", "project", "item-add", projectNum, "--owner", owner, "--url", issueURL)
	output, err := cmd.Output()

	var itemID string
	if err == nil && len(output) > 0 {
		itemID = strings.TrimSpace(string(output))
	} else {
		// Fall back to GraphQL
		itemID, err = addIssueViaGraphQL(boardKey, issueNumber, repo)
		if err != nil {
			return err
		}
	}

	// Apply status if specified
	if status != "" && itemID != "" {
		return SetItemStatus(boardKey, itemID, status)
	}

	return nil
}

// addIssueViaGraphQL adds an issue to a board using GraphQL.
func addIssueViaGraphQL(boardKey string, issueNumber int, repo string) (string, error) {
	owner, projectNum, err := ParseBoardKey(boardKey)
	if err != nil {
		return "", err
	}

	// Get project ID
	projectID, err := FetchProjectID(owner, projectNum)
	if err != nil {
		return "", err
	}

	// Get issue node ID
	issueNodeID, err := flow.GetIssueNodeID(repo, issueNumber)
	if err != nil {
		return "", err
	}

	mutation := `
	mutation($projectId: ID!, $contentId: ID!) {
	  addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
	    item {
	      id
	    }
	  }
	}`

	data, err := flow.GHGraphQL(mutation, map[string]string{
		"projectId": projectID,
		"contentId": issueNodeID,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			AddProjectV2ItemById struct {
				Item struct {
					ID string `json:"id"`
				} `json:"item"`
			} `json:"addProjectV2ItemById"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	return result.Data.AddProjectV2ItemById.Item.ID, nil
}

// SetItemStatus sets the status of a board item.
func SetItemStatus(boardKey, itemID, status string) error {
	owner, projectNum, err := ParseBoardKey(boardKey)
	if err != nil {
		return err
	}

	projectID, err := FetchProjectID(owner, projectNum)
	if err != nil {
		return err
	}

	meta, err := FetchProjectFields(projectID)
	if err != nil {
		return err
	}

	statusFieldID := meta.FieldIDs["Status"]
	if statusFieldID == "" {
		return fmt.Errorf("Status field not found on board")
	}

	optionID := meta.StatusOptions[strings.ToLower(status)]
	if optionID == "" {
		var available []string
		for k := range meta.StatusOptions {
			available = append(available, k)
		}
		return fmt.Errorf("unknown status '%s'; available: %s", status, strings.Join(available, ", "))
	}

	mutation := `
	mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
	  updateProjectV2ItemFieldValue(input: {
	    projectId: $projectId
	    itemId: $itemId
	    fieldId: $fieldId
	    value: {singleSelectOptionId: $optionId}
	  }) {
	    projectV2Item {
	      id
	    }
	  }
	}`

	_, err = flow.GHGraphQL(mutation, map[string]string{
		"projectId": projectID,
		"itemId":    itemID,
		"fieldId":   statusFieldID,
		"optionId":  optionID,
	})

	return err
}

// GetItemID finds the board item ID for an issue.
func GetItemID(boardKey string, issueNumber int, repo string) (string, error) {
	items, err := ListBoardItems(boardKey)
	if err != nil {
		return "", err
	}

	for _, item := range items {
		if item.Content.Repository == repo && item.Content.Number == issueNumber {
			return item.ID, nil
		}
	}

	return "", fmt.Errorf("issue #%d not found on board", issueNumber)
}

// MoveItem moves a board item to a new status.
func MoveItem(boardKey string, issueNumber int, status, repo string) error {
	itemID, err := GetItemID(boardKey, issueNumber, repo)
	if err != nil {
		return err
	}

	return SetItemStatus(boardKey, itemID, status)
}

// RemoveIssueFromBoard removes an issue from a board.
func RemoveIssueFromBoard(boardKey string, issueNumber int, repo string) error {
	owner, projectNum, err := ParseBoardKey(boardKey)
	if err != nil {
		return err
	}

	itemID, err := GetItemID(boardKey, issueNumber, repo)
	if err != nil {
		return err
	}

	projectID, err := FetchProjectID(owner, projectNum)
	if err != nil {
		return err
	}

	mutation := `
	mutation($projectId: ID!, $itemId: ID!) {
	  deleteProjectV2Item(input: {projectId: $projectId, itemId: $itemId}) {
	    deletedItemId
	  }
	}`

	_, err = flow.GHGraphQL(mutation, map[string]string{
		"projectId": projectID,
		"itemId":    itemID,
	})

	return err
}
