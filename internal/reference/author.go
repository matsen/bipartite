package reference

// Author represents a paper author with optional ORCID identifier.
type Author struct {
	First string `json:"first"`           // First/given name(s)
	Last  string `json:"last"`            // Last/family name
	ORCID string `json:"orcid,omitempty"` // ORCID identifier (without URL prefix)
}
