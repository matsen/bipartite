package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseRepoEntries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []RepoEntry
	}{
		{
			name:  "string entries",
			input: `["matsengrp/repo1", "matsengrp/repo2"]`,
			expected: []RepoEntry{
				{Repo: "matsengrp/repo1"},
				{Repo: "matsengrp/repo2"},
			},
		},
		{
			name:  "object entries",
			input: `[{"repo": "matsengrp/repo1", "channel": "dasm2"}]`,
			expected: []RepoEntry{
				{Repo: "matsengrp/repo1", Channel: "dasm2"},
			},
		},
		{
			name:  "mixed entries",
			input: `["matsengrp/repo1", {"repo": "matsengrp/repo2", "channel": "test"}]`,
			expected: []RepoEntry{
				{Repo: "matsengrp/repo1"},
				{Repo: "matsengrp/repo2", Channel: "test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRepoEntries(json.RawMessage(tt.input))
			if err != nil {
				t.Fatalf("parseRepoEntries() error: %v", err)
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("parseRepoEntries() got %d entries, want %d", len(got), len(tt.expected))
			}

			for i, entry := range got {
				if entry.Repo != tt.expected[i].Repo {
					t.Errorf("entry[%d].Repo = %q, want %q", i, entry.Repo, tt.expected[i].Repo)
				}
				if entry.Channel != tt.expected[i].Channel {
					t.Errorf("entry[%d].Channel = %q, want %q", i, entry.Channel, tt.expected[i].Channel)
				}
			}
		})
	}
}

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"matsengrp/dasm2-experiments", "dasm2-experiments"},
		{"org/repo-v2", "repo-v2"},
		{"repo", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractRepoName(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractRepoName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoadSourcesIntegration(t *testing.T) {
	// Create a temp directory with a test sources.json
	tmpDir := t.TempDir()
	sourcesPath := filepath.Join(tmpDir, "sources.json")

	sourcesContent := `{
		"boards": {"matsengrp/30": "test-bead"},
		"context": {"matsengrp/repo": "context/test.md"},
		"code": [
			"matsengrp/repo1",
			{"repo": "matsengrp/repo2", "channel": "dasm2"}
		],
		"writing": ["matsengrp/paper1"]
	}`

	if err := os.WriteFile(sourcesPath, []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("Failed to write test sources.json: %v", err)
	}

	// Change to temp dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Test LoadSources
	sources, err := LoadSources()
	if err != nil {
		t.Fatalf("LoadSources() error: %v", err)
	}

	// Verify boards
	if sources.Boards["matsengrp/30"] != "test-bead" {
		t.Errorf("Boards['matsengrp/30'] = %q, want 'test-bead'", sources.Boards["matsengrp/30"])
	}

	// Verify context
	if sources.Context["matsengrp/repo"] != "context/test.md" {
		t.Errorf("Context['matsengrp/repo'] = %q, want 'context/test.md'", sources.Context["matsengrp/repo"])
	}

	// Verify code repos
	if len(sources.Code) != 2 {
		t.Errorf("Code has %d entries, want 2", len(sources.Code))
	}
	if sources.Code[0].Repo != "matsengrp/repo1" {
		t.Errorf("Code[0].Repo = %q, want 'matsengrp/repo1'", sources.Code[0].Repo)
	}
	if sources.Code[1].Channel != "dasm2" {
		t.Errorf("Code[1].Channel = %q, want 'dasm2'", sources.Code[1].Channel)
	}

	// Verify writing repos
	if len(sources.Writing) != 1 {
		t.Errorf("Writing has %d entries, want 1", len(sources.Writing))
	}
}

func TestLoadReposByChannel(t *testing.T) {
	// Create a temp directory with a test sources.json
	tmpDir := t.TempDir()
	sourcesPath := filepath.Join(tmpDir, "sources.json")

	sourcesContent := `{
		"code": [
			"matsengrp/repo1",
			{"repo": "matsengrp/repo2", "channel": "dasm2"},
			{"repo": "matsengrp/repo3", "channel": "test"},
			{"repo": "matsengrp/repo4", "channel": "dasm2"}
		]
	}`

	if err := os.WriteFile(sourcesPath, []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("Failed to write test sources.json: %v", err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Test loading dasm2 channel
	repos, err := LoadReposByChannel("dasm2")
	if err != nil {
		t.Fatalf("LoadReposByChannel() error: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("Got %d repos, want 2", len(repos))
	}

	// Test unknown channel
	repos, err = LoadReposByChannel("unknown")
	if err != nil {
		t.Fatalf("LoadReposByChannel() error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("Got %d repos for unknown channel, want 0", len(repos))
	}
}

func TestListChannels(t *testing.T) {
	tmpDir := t.TempDir()
	sourcesPath := filepath.Join(tmpDir, "sources.json")

	sourcesContent := `{
		"code": [
			"matsengrp/repo1",
			{"repo": "matsengrp/repo2", "channel": "dasm2"},
			{"repo": "matsengrp/repo3", "channel": "test"},
			{"repo": "matsengrp/repo4", "channel": "dasm2"}
		],
		"writing": [
			{"repo": "matsengrp/paper1", "channel": "test"}
		]
	}`

	if err := os.WriteFile(sourcesPath, []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("Failed to write test sources.json: %v", err)
	}

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	channels, err := ListChannels()
	if err != nil {
		t.Fatalf("ListChannels() error: %v", err)
	}

	if len(channels) != 2 {
		t.Errorf("Got %d channels, want 2", len(channels))
	}

	// Should be sorted
	if channels[0] != "dasm2" || channels[1] != "test" {
		t.Errorf("Channels = %v, want [dasm2, test]", channels)
	}
}
