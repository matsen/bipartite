// Package ncbi provides a client for the NCBI PMC ID Converter API.
//
// The NCBI ID Converter (https://pmc.ncbi.nlm.nih.gov/tools/id-converter-api/)
// resolves between DOI, PMID, PMCID, and MID identifiers. We use it primarily
// to discover PMCIDs for references that have a DOI or PMID but no PMCID.
//
// API specifics verified against the live endpoint:
//
//   - Batch cap: 200 IDs per request.
//   - The endpoint auto-detects the ID type from the input and rejects batches
//     that mix types ("All values of query param `ids` must be DOIs."). Callers
//     must group IDs by type before batching.
//   - The per-record `requested-id` field is the canonical join key. Response
//     order is not guaranteed to match request order; match by `requested-id`.
//   - The `pmid` field is returned as a JSON number, not a string.
//   - On a per-record lookup failure ("not in PMC"), the record carries a
//     `status: "error"` field and an `errmsg` (e.g., "Identifier not found in
//     PMC") in place of `pmcid`/`pmid`. The top-level response status is still
//     `"ok"`. Treat per-record errors as "no PMCID returned, not a fatal error".
//   - On a request-wide failure (malformed IDs, parse errors), the response is
//     `{"status":"error", "http_status":"400", "errors":[...]}` with an empty
//     records array.
package ncbi

// Record is a single result from the NCBI ID Converter.
//
// PMCID is empty when the converter could not find a PMC entry for the
// requested ID. RequestedID echoes the input verbatim and is the join key
// callers must use to match records back to their source.
type Record struct {
	RequestedID string `json:"requested-id"`
	DOI         string `json:"doi,omitempty"`
	PMID        int    `json:"pmid,omitempty"`
	PMCID       string `json:"pmcid,omitempty"`

	// Status is "error" when this specific record failed (e.g., not in PMC);
	// absent or "ok" otherwise. ErrMsg accompanies an error status.
	Status string `json:"status,omitempty"`
	ErrMsg string `json:"errmsg,omitempty"`
}

// Response is the top-level NCBI ID Converter response envelope.
type Response struct {
	Status       string   `json:"status"`
	ResponseDate string   `json:"response-date,omitempty"`
	Records      []Record `json:"records,omitempty"`

	// Errors is populated on request-wide failures (status="error").
	Errors []APIErrorDetail `json:"errors,omitempty"`
}

// APIErrorDetail is one entry in a request-wide error response.
type APIErrorDetail struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}
