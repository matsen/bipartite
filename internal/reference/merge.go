package reference

// MergeUpdate produces an updated Reference from an incoming import and the
// existing on-disk version. For every field other than ID and Source, an
// incoming zero value falls back to the existing value. This preserves
// post-import edits (e.g., PMCIDs added by `bip ncbi backfill`, or PDF paths
// linked via `bip s2 linkpub`) when the import format doesn't carry that
// field at all.
//
// ID and Source are always taken from incoming because they describe this
// import event itself: ID is the citekey the importer chose, and Source.Type
// / Source.ID identify which external system this came from.
//
// Trade-off: if a user clears a field in their import source (e.g., removes
// a tag in Paperpile), the import sends a zero value, and the merge keeps
// the old value. That's the price of preserving externally-resolved fields.
// Users who want a true wipe should edit refs.jsonl directly.
func MergeUpdate(existing, incoming Reference) Reference {
	out := incoming

	if out.DOI == "" {
		out.DOI = existing.DOI
	}
	if out.Title == "" {
		out.Title = existing.Title
	}
	if len(out.Authors) == 0 {
		out.Authors = existing.Authors
	}
	if out.Abstract == "" {
		out.Abstract = existing.Abstract
	}
	if out.Venue == "" {
		out.Venue = existing.Venue
	}
	if out.Note == "" {
		out.Note = existing.Note
	}
	if out.Published.Year == 0 {
		out.Published = existing.Published
	}
	if out.PDFPath == "" {
		out.PDFPath = existing.PDFPath
	}
	if len(out.SupplementPaths) == 0 {
		out.SupplementPaths = existing.SupplementPaths
	}
	if len(out.Tags) == 0 {
		out.Tags = existing.Tags
	}
	if out.Supersedes == "" {
		out.Supersedes = existing.Supersedes
	}
	if out.PMID == "" {
		out.PMID = existing.PMID
	}
	if out.PMCID == "" {
		out.PMCID = existing.PMCID
	}
	if out.ArXivID == "" {
		out.ArXivID = existing.ArXivID
	}
	if out.S2ID == "" {
		out.S2ID = existing.S2ID
	}

	return out
}
