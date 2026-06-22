package pdf

import "testing"

func TestFindDOI(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "single doi",
			text: "Published online. doi:10.1038/nature12373 in Nature.",
			want: "10.1038/nature12373",
		},
		{
			name: "trailing period stripped",
			text: "See 10.1234/foo.",
			want: "10.1234/foo",
		},
		{
			name: "trailing punctuation stripped",
			text: "(10.1234/foo);",
			want: "10.1234/foo",
		},
		{
			name: "no doi",
			text: "This text has no identifier at all.",
			want: "",
		},
		{
			name: "multiple returns first",
			text: "10.1111/aaa and later 10.2222/bbb",
			want: "10.1111/aaa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findDOI(tt.text); got != tt.want {
				t.Errorf("findDOI(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsValidDOI(t *testing.T) {
	tests := []struct {
		name string
		doi  string
		want bool
	}{
		{"valid", "10.1038/nature12373", true},
		{"too short", "10.1/x", false},
		{"missing 10. prefix", "11.1234/foobar", false},
		{"missing slash", "10.1234foobar", false},
		{"trailing slash only", "10.1234567/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidDOI(tt.doi); got != tt.want {
				t.Errorf("isValidDOI(%q) = %v, want %v", tt.doi, got, tt.want)
			}
		})
	}
}

func TestIsHeaderLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"journal", "Journal of Molecular Biology", true},
		{"volume and issue", "Volume 42, Issue 3", true},
		{"copyright", "Copyright 2024 The Authors", true},
		{"article published", "Article published online 2024", true},
		{"ordinary title", "A Bayesian Approach to Phylogenetic Inference", false},
		{"volume without issue", "Volume 42 of the proceedings", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHeaderLine(tt.line); got != tt.want {
				t.Errorf("isHeaderLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
