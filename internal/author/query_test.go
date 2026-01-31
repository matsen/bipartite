package author

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Query
	}{
		{
			name:  "single word is last name",
			input: "Yu",
			want:  Query{Last: "Yu"},
		},
		{
			name:  "two words is First Last",
			input: "Timothy Yu",
			want:  Query{First: "Timothy", Last: "Yu"},
		},
		{
			name:  "three words: first two are first name",
			input: "Timothy C Yu",
			want:  Query{First: "Timothy C", Last: "Yu"},
		},
		{
			name:  "comma format: Last, First",
			input: "Yu, Timothy",
			want:  Query{First: "Timothy", Last: "Yu"},
		},
		{
			name:  "comma format with spaces",
			input: "Yu,  Timothy C",
			want:  Query{First: "Timothy C", Last: "Yu"},
		},
		{
			name:  "leading/trailing whitespace",
			input: "  Bloom  ",
			want:  Query{Last: "Bloom"},
		},
		{
			name:  "empty string",
			input: "",
			want:  Query{},
		},
		{
			name:  "whitespace only",
			input: "   ",
			want:  Query{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseQuery(tt.input)
			if got != tt.want {
				t.Errorf("ParseQuery(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestQueryMatches(t *testing.T) {
	tests := []struct {
		name   string
		query  Query
		author reference.Author
		want   bool
	}{
		{
			name:   "exact last name match",
			query:  Query{Last: "Yu"},
			author: reference.Author{First: "Timothy C", Last: "Yu"},
			want:   true,
		},
		{
			name:   "last name case insensitive",
			query:  Query{Last: "yu"},
			author: reference.Author{First: "Timothy", Last: "Yu"},
			want:   true,
		},
		{
			name:   "last name no partial match",
			query:  Query{Last: "Yu"},
			author: reference.Author{First: "Yujia Alina", Last: "Chan"},
			want:   false,
		},
		{
			name:   "first and last match",
			query:  Query{First: "Timothy", Last: "Yu"},
			author: reference.Author{First: "Timothy C", Last: "Yu"},
			want:   true,
		},
		{
			name:   "first name prefix match",
			query:  Query{First: "Tim", Last: "Yu"},
			author: reference.Author{First: "Timothy C", Last: "Yu"},
			want:   true,
		},
		{
			name:   "first name case insensitive",
			query:  Query{First: "timothy", Last: "Yu"},
			author: reference.Author{First: "Timothy", Last: "Yu"},
			want:   true,
		},
		{
			name:   "first name mismatch",
			query:  Query{First: "John", Last: "Yu"},
			author: reference.Author{First: "Timothy", Last: "Yu"},
			want:   false,
		},
		{
			name:   "last name mismatch",
			query:  Query{First: "Timothy", Last: "Chan"},
			author: reference.Author{First: "Timothy", Last: "Yu"},
			want:   false,
		},
		{
			name:   "full first name with middle initial",
			query:  Query{First: "Timothy C", Last: "Yu"},
			author: reference.Author{First: "Timothy C", Last: "Yu"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.query.Matches(tt.author)
			if got != tt.want {
				t.Errorf("Query%+v.Matches(%+v) = %v, want %v", tt.query, tt.author, got, tt.want)
			}
		})
	}
}

func TestQueryMatchesAny(t *testing.T) {
	authors := []reference.Author{
		{First: "Jesse D", Last: "Bloom"},
		{First: "Yujia Alina", Last: "Chan"},
		{First: "Timothy C", Last: "Yu"},
	}

	tests := []struct {
		name  string
		query Query
		want  bool
	}{
		{
			name:  "matches first author",
			query: Query{Last: "Bloom"},
			want:  true,
		},
		{
			name:  "matches last author",
			query: Query{Last: "Yu"},
			want:  true,
		},
		{
			name:  "no match - Yu is not a last name here",
			query: Query{First: "Yu", Last: "Something"},
			want:  false,
		},
		{
			name:  "matches with first name",
			query: Query{First: "Jesse", Last: "Bloom"},
			want:  true,
		},
		{
			name:  "no match for nonexistent author",
			query: Query{Last: "Smith"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.query.MatchesAny(authors)
			if got != tt.want {
				t.Errorf("Query%+v.MatchesAny() = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestAllMatch(t *testing.T) {
	authors := []reference.Author{
		{First: "Jesse D", Last: "Bloom"},
		{First: "Timothy C", Last: "Yu"},
	}

	tests := []struct {
		name    string
		queries []Query
		want    bool
	}{
		{
			name:    "both authors match",
			queries: []Query{{Last: "Bloom"}, {Last: "Yu"}},
			want:    true,
		},
		{
			name:    "one author missing",
			queries: []Query{{Last: "Bloom"}, {Last: "Chan"}},
			want:    false,
		},
		{
			name:    "empty queries matches all",
			queries: []Query{},
			want:    true,
		},
		{
			name:    "single query matches",
			queries: []Query{{Last: "Bloom"}},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllMatch(tt.queries, authors)
			if got != tt.want {
				t.Errorf("AllMatch(%+v, authors) = %v, want %v", tt.queries, got, tt.want)
			}
		})
	}
}

// TestIssue81 tests the specific cases from GitHub issue #81.
func TestIssue81(t *testing.T) {
	// Paper: "Investigate the origins of COVID-19"
	// Authors: Jesse D Bloom, Yujia Alina Chan, Ralph S Baric, ...
	bloom2021Authors := []reference.Author{
		{First: "Jesse D", Last: "Bloom"},
		{First: "Yujia Alina", Last: "Chan"},
		{First: "Ralph S", Last: "Baric"},
	}

	t.Run("Yu should not match Yujia", func(t *testing.T) {
		query := ParseQuery("Yu")
		if query.MatchesAny(bloom2021Authors) {
			t.Error("Query 'Yu' should NOT match paper with Yujia Alina Chan")
		}
	})

	t.Run("Bloom should match Jesse D Bloom", func(t *testing.T) {
		query := ParseQuery("Bloom")
		if !query.MatchesAny(bloom2021Authors) {
			t.Error("Query 'Bloom' should match paper with Jesse D Bloom")
		}
	})

	// Paper with actual Timothy Yu
	yuPaperAuthors := []reference.Author{
		{First: "Timothy C", Last: "Yu"},
		{First: "Jesse D", Last: "Bloom"},
	}

	t.Run("Yu should match Timothy C Yu", func(t *testing.T) {
		query := ParseQuery("Yu")
		if !query.MatchesAny(yuPaperAuthors) {
			t.Error("Query 'Yu' should match paper with Timothy C Yu")
		}
	})

	t.Run("Timothy Yu should match Timothy C Yu", func(t *testing.T) {
		query := ParseQuery("Timothy Yu")
		if !query.MatchesAny(yuPaperAuthors) {
			t.Error("Query 'Timothy Yu' should match paper with Timothy C Yu")
		}
	})

	t.Run("Yu, Timothy should match Timothy C Yu", func(t *testing.T) {
		query := ParseQuery("Yu, Timothy")
		if !query.MatchesAny(yuPaperAuthors) {
			t.Error("Query 'Yu, Timothy' should match paper with Timothy C Yu")
		}
	})

	// Hodcroft2021-xj case: Timothy G Vaughan and Jesse D Bloom
	hodcroftAuthors := []reference.Author{
		{First: "Timothy G", Last: "Vaughan"},
		{First: "Jesse D", Last: "Bloom"},
	}

	t.Run("Timothy Yu should not match Timothy Vaughan", func(t *testing.T) {
		queries := []Query{ParseQuery("Timothy Yu"), ParseQuery("Bloom")}
		if AllMatch(queries, hodcroftAuthors) {
			t.Error("Queries 'Timothy Yu' + 'Bloom' should NOT match Hodcroft paper")
		}
	})
}
