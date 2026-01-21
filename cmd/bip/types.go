package main

// OpenMultipleResult is the JSON response for opening multiple papers.
type OpenMultipleResult struct {
	Opened []OpenedPaper `json:"opened"`
	Errors []OpenError   `json:"errors,omitempty"`
}

// OpenedPaper represents a successfully opened paper.
type OpenedPaper struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// OpenError represents an error opening a specific paper.
type OpenError struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// DiffResult is the JSON response for bip diff.
type DiffResult struct {
	Added   []DiffPaper `json:"added"`
	Removed []DiffPaper `json:"removed"`
}

// DiffPaper represents a paper in diff output.
type DiffPaper struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Authors string `json:"authors"` // Formatted as "Last1, Last2, ..."
	Year    int    `json:"year"`
}

// NewPapersResult is the JSON response for bip new.
type NewPapersResult struct {
	Papers     []NewPaper `json:"papers"`
	SinceRef   string     `json:"since_ref,omitempty"` // Commit SHA or "N days ago"
	TotalCount int        `json:"total_count"`
}

// NewPaper represents a paper in the new papers output.
type NewPaper struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Authors   string `json:"authors"`
	Year      int    `json:"year"`
	CommitSHA string `json:"commit_sha,omitempty"` // When paper was added
}

// ExportResult is the JSON response for bip export with --append.
type ExportResult struct {
	Exported   int      `json:"exported"`              // Number of entries written
	Skipped    int      `json:"skipped"`               // Number of duplicates skipped
	SkippedIDs []string `json:"skipped_ids,omitempty"` // IDs that were duplicates
	OutputPath string   `json:"output_path,omitempty"` // When --append used
}
