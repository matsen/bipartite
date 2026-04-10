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
	Key              string           `json:"key"`
	Version          int              `json:"version"`
	ItemType         string           `json:"itemType"`
	Title            string           `json:"title"`
	Creators         []ZoteroCreator  `json:"creators"`
	AbstractNote     string           `json:"abstractNote"`
	PublicationTitle string           `json:"publicationTitle"` // Journal/conference name
	Volume           string           `json:"volume"`
	Issue            string           `json:"issue"`
	Pages            string           `json:"pages"`
	Date             string           `json:"date"` // Free-form date string
	DOI              string           `json:"DOI"`
	ISSN             string           `json:"ISSN"`
	URL              string           `json:"url"`
	Extra            string           `json:"extra"` // Contains PMID, PMCID, arXiv etc.
	Tags             []ZoteroTag      `json:"tags"`
	Collections      []string         `json:"collections"`
	Relations        map[string]interface{} `json:"relations"`
	DateAdded        string           `json:"dateAdded"`
	DateModified     string           `json:"dateModified"`
}

// ZoteroCreator represents an author or other contributor.
type ZoteroCreator struct {
	CreatorType string `json:"creatorType"` // "author", "editor", etc.
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Name        string `json:"name"` // For single-field names (institutional)
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
