package tree

import (
	"strings"
	"testing"
	"time"

	"github.com/matsen/bipartite/internal/flow"
)

func TestBuildTree(t *testing.T) {
	beads := []flow.Bead{
		{ID: "flow-dasm.1", Title: "DASM Project"},
		{ID: "flow-dasm.1.1", Title: "Sub-goal 1"},
		{ID: "flow-dasm.1.2", Title: "Sub-goal 2"},
		{ID: "flow-dasm.2", Title: "Another project"},
	}

	tree := BuildTree(beads)

	// Check root has children
	if len(tree.Children) == 0 {
		t.Fatal("expected tree to have children")
	}

	// Check flow-dasm.1 exists
	node1, ok := tree.Children["flow-dasm"]
	if !ok {
		t.Fatal("expected flow-dasm node")
	}

	node11 := node1.Children["flow-dasm.1"]
	if node11 == nil {
		t.Fatal("expected flow-dasm.1 node")
	}
	if node11.Issue == nil || node11.Issue.Title != "DASM Project" {
		t.Errorf("expected flow-dasm.1 to have title 'DASM Project'")
	}

	// Check sub-nodes
	if len(node11.Children) != 2 {
		t.Errorf("expected flow-dasm.1 to have 2 children, got %d", len(node11.Children))
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	tree := BuildTree([]flow.Bead{})
	if len(tree.Children) != 0 {
		t.Errorf("expected empty tree, got %d children", len(tree.Children))
	}
}

func TestIsNew(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	tests := []struct {
		name      string
		bead      *flow.Bead
		since     *time.Time
		wantIsNew bool
	}{
		{
			name:      "nil bead",
			bead:      nil,
			since:     &yesterday,
			wantIsNew: false,
		},
		{
			name:      "nil since",
			bead:      &flow.Bead{CreatedAt: now},
			since:     nil,
			wantIsNew: false,
		},
		{
			name:      "bead created after since",
			bead:      &flow.Bead{CreatedAt: now},
			since:     &yesterday,
			wantIsNew: true,
		},
		{
			name:      "bead created before since",
			bead:      &flow.Bead{CreatedAt: lastWeek},
			since:     &yesterday,
			wantIsNew: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNew(tt.bead, tt.since)
			if got != tt.wantIsNew {
				t.Errorf("IsNew() = %v, want %v", got, tt.wantIsNew)
			}
		})
	}
}

func TestParseGitHubLink(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want string
	}{
		{
			name: "valid github link",
			desc: "GitHub: matsengrp/netam#123",
			want: "https://github.com/matsengrp/netam/issues/123",
		},
		{
			name: "github link with extra text",
			desc: "Some task\nGitHub: org/repo#42\nMore text",
			want: "https://github.com/org/repo/issues/42",
		},
		{
			name: "no github link",
			desc: "Just a regular description",
			want: "",
		},
		{
			name: "empty description",
			desc: "",
			want: "",
		},
		{
			name: "github link with colon space",
			desc: "GitHub:   owner/repo#999",
			want: "https://github.com/owner/repo/issues/999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseGitHubLink(tt.desc)
			if got != tt.want {
				t.Errorf("ParseGitHubLink() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseReferenceLinks(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want int // number of links expected
	}{
		{
			name: "paper and code links",
			desc: "Paper: github.com/org/paper-repo | Code: github.com/org/code-repo",
			want: 2,
		},
		{
			name: "paper only",
			desc: "Paper: github.com/org/paper-tex",
			want: 1,
		},
		{
			name: "code only",
			desc: "Code: github.com/org/code-repo",
			want: 1,
		},
		{
			name: "no links",
			desc: "Just a description",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseReferenceLinks(tt.desc)
			if len(got) != tt.want {
				t.Errorf("ParseReferenceLinks() returned %d links, want %d", len(got), tt.want)
			}
		})
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{"<div>&</div>", "&lt;div&gt;&amp;&lt;/div&gt;"},
		{"", ""},
	}

	for _, tt := range tests {
		got := escapeHTML(tt.input)
		if got != tt.want {
			t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseSince(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "empty string",
			value:   "",
			wantErr: false,
		},
		{
			name:    "valid date",
			value:   "2024-01-15",
			wantErr: false,
		},
		{
			name:    "valid ISO format",
			value:   "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "invalid format",
			value:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "partial date",
			value:   "2024-01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSince(tt.value)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.value == "" && result != nil {
				t.Errorf("expected nil for empty string, got %v", result)
			}

			if tt.value != "" && result == nil {
				t.Errorf("expected non-nil result for %q", tt.value)
			}
		})
	}
}

func TestGenerateHTML(t *testing.T) {
	t.Run("empty beads", func(t *testing.T) {
		html := GenerateHTML([]flow.Bead{}, nil)
		if !strings.Contains(html, "No beads issues found") {
			t.Error("expected 'No beads issues found' message for empty beads")
		}
	})

	t.Run("with beads", func(t *testing.T) {
		beads := []flow.Bead{
			{ID: "test.1", Title: "Test Title", IssueType: "epic"},
		}
		html := GenerateHTML(beads, nil)

		if !strings.Contains(html, "<!DOCTYPE html>") {
			t.Error("expected HTML doctype")
		}
		if !strings.Contains(html, "Test Title") {
			t.Error("expected bead title in HTML")
		}
		if !strings.Contains(html, "test.1") {
			t.Error("expected bead ID in HTML")
		}
	})

	t.Run("with chore type", func(t *testing.T) {
		beads := []flow.Bead{
			{
				ID:          "test.1",
				Title:       "External Work",
				IssueType:   "chore",
				Description: "Paper: github.com/org/paper-tex | Code: github.com/org/code",
			},
		}
		html := GenerateHTML(beads, nil)

		if !strings.Contains(html, "chore") {
			t.Error("expected chore class in HTML")
		}
	})

	t.Run("with github link", func(t *testing.T) {
		beads := []flow.Bead{
			{
				ID:          "test.1",
				Title:       "Fix bug",
				IssueType:   "task",
				Description: "GitHub: org/repo#42",
			},
		}
		html := GenerateHTML(beads, nil)

		if !strings.Contains(html, "https://github.com/org/repo/issues/42") {
			t.Error("expected GitHub link in HTML")
		}
	})

	t.Run("with new highlighting", func(t *testing.T) {
		yesterday := time.Now().Add(-24 * time.Hour)
		beads := []flow.Bead{
			{
				ID:        "test.1",
				Title:     "New Bead",
				CreatedAt: time.Now(),
			},
		}
		html := GenerateHTML(beads, &yesterday)

		if !strings.Contains(html, "new") {
			t.Error("expected 'new' class in HTML for recent bead")
		}
	})
}

func TestRenderNode(t *testing.T) {
	t.Run("leaf node", func(t *testing.T) {
		beads := []flow.Bead{
			{ID: "test.1", Title: "Leaf Node"},
		}
		tree := BuildTree(beads)
		html := RenderNode(tree, true, nil)

		if !strings.Contains(html, "leaf") {
			t.Error("expected 'leaf' class for leaf node")
		}
	})

	t.Run("nested nodes", func(t *testing.T) {
		beads := []flow.Bead{
			{ID: "test.1", Title: "Parent"},
			{ID: "test.1.1", Title: "Child"},
		}
		tree := BuildTree(beads)
		html := RenderNode(tree, true, nil)

		if !strings.Contains(html, "<details") {
			t.Error("expected details element for nested nodes")
		}
		if !strings.Contains(html, "Child") {
			t.Error("expected child content")
		}
	})
}
