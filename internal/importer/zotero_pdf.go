package importer

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/matsen/bipartite/internal/reference"
	_ "modernc.org/sqlite"
)

// ResolveZoteroPDFs enriches references with PDF paths from Zotero's local database.
// It matches references by their Source.ID (Zotero item key) against Zotero's
// itemAttachments table and sets the PDFPath field to the relative path within
// Zotero's storage directory.
//
// dbPath is the path to Zotero's SQLite database (typically ~/Zotero/zotero.sqlite).
// Only references with Source.Type == "zotero" and empty PDFPath are processed.
//
// The database is opened read-only. If Zotero is running and holds a lock,
// this function returns an error advising the user to close Zotero.
func ResolveZoteroPDFs(refs []reference.Reference, dbPath string) (int, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return 0, fmt.Errorf("opening Zotero database: %w", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	// Test the connection — catches SQLITE_BUSY from Zotero holding a lock
	if err := db.Ping(); err != nil {
		return 0, fmt.Errorf("cannot read Zotero database (is Zotero running? close it and retry): %w", err)
	}

	// Query: for each item key, find the PDF attachment path.
	// Zotero stores attachments as child items linked via itemAttachments.parentItemID.
	// The attachment path is in itemAttachments.path, prefixed with "storage:" for
	// locally stored files.
	//
	// We join items (to get the item key) with itemAttachments (to get the file path).
	// We filter for PDF content type.
	const query = `
		SELECT parent.key, ia.path
		FROM itemAttachments ia
		JOIN items child ON child.itemID = ia.itemID
		JOIN items parent ON parent.itemID = ia.parentItemID
		WHERE ia.contentType = 'application/pdf'
		AND ia.path IS NOT NULL
		AND ia.path != ''
	`

	rows, err := db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("querying Zotero attachments: %w", err)
	}
	defer rows.Close()

	// Build a map of Zotero item key → relative PDF path
	pdfMap := make(map[string]string)
	for rows.Next() {
		var key, path string
		if err := rows.Scan(&key, &path); err != nil {
			continue
		}
		// Zotero stores paths as "storage:filename.pdf" for managed files
		relPath := stripStoragePrefix(path)
		// Full relative path within Zotero storage: <key>/<filename>
		pdfMap[key] = filepath.Join(key, relPath)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("reading Zotero attachments: %w", err)
	}

	// Enrich references
	resolved := 0
	for i := range refs {
		if refs[i].Source.Type != "zotero" || refs[i].PDFPath != "" {
			continue
		}
		if path, ok := pdfMap[refs[i].Source.ID]; ok {
			refs[i].PDFPath = path
			resolved++
		}
	}

	return resolved, nil
}

// stripStoragePrefix removes the "storage:" prefix from Zotero attachment paths.
func stripStoragePrefix(path string) string {
	const prefix = "storage:"
	if len(path) > len(prefix) && path[:len(prefix)] == prefix {
		return path[len(prefix):]
	}
	return path
}
