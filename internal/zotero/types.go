// Package zotero provides a client for the Zotero Web API v3.
package zotero

// ZoteroItem represents an item from the Zotero API.
type ZoteroItem struct {
	Key     string         `json:"key"`
	Version int            `json:"version"`
	Library ZoteroLibrary  `json:"library"`
	Data    ZoteroItemData `json:"data"`
}

// ZoteroLibrary identifies which library an item belongs to.
type ZoteroLibrary struct {
	Type string `json:"type"` // "user" or "group"
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ZoteroItemData holds the metadata for a Zotero item.
type ZoteroItemData struct {
	Key              string           `json:"key,omitempty"`
	Version          int              `json:"version,omitempty"`
	ItemType         string           `json:"itemType"`
	Title            string           `json:"title"`
	Creators         []ZoteroCreator  `json:"creators"`
	AbstractNote     string           `json:"abstractNote,omitempty"`
	PublicationTitle string           `json:"publicationTitle,omitempty"` // Journal/conference name
	Volume           string           `json:"volume,omitempty"`
	Issue            string           `json:"issue,omitempty"`
	Pages            string           `json:"pages,omitempty"`
	Date             string           `json:"date,omitempty"` // Free-form date string
	DOI              string           `json:"DOI,omitempty"`
	ISSN             string           `json:"ISSN,omitempty"`
	URL              string           `json:"url,omitempty"`
	Extra            string           `json:"extra,omitempty"` // Contains PMID, PMCID, arXiv etc.
	Tags             []ZoteroTag      `json:"tags,omitempty"`
	Collections      []string         `json:"collections,omitempty"`
	Relations        map[string]interface{} `json:"relations,omitempty"`
	DateAdded        string           `json:"dateAdded,omitempty"`
	DateModified     string           `json:"dateModified,omitempty"`
}

// ZoteroCreator represents an author or other contributor.
type ZoteroCreator struct {
	CreatorType string `json:"creatorType"` // "author", "editor", etc.
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Name        string `json:"name,omitempty"` // For single-field names (institutional)
}

// ZoteroTag represents a tag on an item.
type ZoteroTag struct {
	Tag  string `json:"tag"`
	Type int    `json:"type,omitempty"`
}

// ZoteroItemTemplate is the template returned by the /items/new endpoint.
type ZoteroItemTemplate struct {
	ItemType string `json:"itemType"`
	// All other fields are dynamic
}

// CreateItemRequest wraps items for the POST /items endpoint.
// The Zotero API expects an array of items at the top level.
type CreateItemRequest []ZoteroItemData

// CreateItemResponse is the response from POST /items.
type CreateItemResponse struct {
	Successful   map[string]ZoteroItem `json:"successful"`
	Success      map[string]string     `json:"success"` // index -> key
	Unchanged    map[string]string     `json:"unchanged"`
	Failed       map[string]FailedItem `json:"failed"`
}

// FailedItem describes why an item creation failed.
type FailedItem struct {
	Key     string `json:"key"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// VersionsResponse maps item keys to their version numbers.
type VersionsResponse map[string]int
