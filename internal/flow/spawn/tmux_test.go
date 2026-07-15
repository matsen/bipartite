package spawn

import "testing"

func TestBuildWindowName(t *testing.T) {
	tests := []struct {
		name     string
		repoPath string
		number   int
		want     string
	}{
		{
			name:     "simple repo path",
			repoPath: "/Users/matsen/re/netam",
			number:   123,
			want:     "netam#123",
		},
		{
			name:     "nested repo path",
			repoPath: "/home/user/projects/work/my-repo",
			number:   42,
			want:     "my-repo#42",
		},
		{
			name:     "single directory",
			repoPath: "repo",
			number:   1,
			want:     "repo#1",
		},
		{
			name:     "path with trailing slash",
			repoPath: "/Users/matsen/re/netam/",
			number:   456,
			want:     "netam#456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildWindowName(tt.repoPath, tt.number)
			if got != tt.want {
				t.Errorf("BuildWindowName(%q, %d) = %q, want %q", tt.repoPath, tt.number, got, tt.want)
			}
		})
	}
}

func TestIsInTmux(t *testing.T) {
	// This test just verifies the function runs without panic.
	// The actual result depends on the test environment.
	_ = IsInTmux()
}

func TestBuildClaudeInvocation(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "empty model is unchanged from pre-existing behavior",
			model: "",
			want:  `claude --dangerously-skip-permissions "$prompt"`,
		},
		{
			name:  "model is passed through as --model",
			model: "opus",
			want:  `claude --dangerously-skip-permissions --model 'opus' "$prompt"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildClaudeInvocation(tt.model)
			if got != tt.want {
				t.Errorf("buildClaudeInvocation(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}
