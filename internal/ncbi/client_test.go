package ncbi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadFixture reads a JSON fixture file from testdata/.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

// newFixtureServer returns an httptest server that responds to every request
// with the named fixture, and records the most recent request for assertions.
type fixtureServer struct {
	server     *httptest.Server
	lastQuery  map[string][]string
	requests   int
	statusCode int
	body       []byte
}

func newFixtureServer(t *testing.T, statusCode int, body []byte) *fixtureServer {
	t.Helper()
	fs := &fixtureServer{statusCode: statusCode, body: body}
	fs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.requests++
		fs.lastQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(fs.statusCode)
		_, _ = w.Write(fs.body)
	}))
	t.Cleanup(fs.server.Close)
	return fs
}

func TestConvert_JoinByRequestedID(t *testing.T) {
	// Records arrive in [C, A, B] order; caller must match by RequestedID,
	// not positional index.
	fs := newFixtureServer(t, 200, loadFixture(t, "dois_out_of_order.json"))
	client := NewClient(WithBaseURL(fs.server.URL))

	recs, err := client.Convert(context.Background(), []Input{
		{Type: IDTypeDOI, ID: "10.1038/A"},
		{Type: IDTypeDOI, ID: "10.1038/B"},
		{Type: IDTypeDOI, ID: "10.1038/C"},
	})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("expected 3 records, got %d", len(recs))
	}

	got := make(map[string]string)
	for _, r := range recs {
		got[r.RequestedID] = r.PMCID
	}
	want := map[string]string{
		"10.1038/A": "PMC1",
		"10.1038/B": "PMC2",
		"10.1038/C": "PMC3",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("RequestedID=%q: got PMCID=%q, want %q", k, got[k], v)
		}
	}
}

func TestConvert_PerRecordErrorNotFatal(t *testing.T) {
	// One DOI in PMC, one not. Top-level status is "ok"; the missing one
	// comes back as a record with Status="error".
	fs := newFixtureServer(t, 200, loadFixture(t, "dois_mixed_status.json"))
	client := NewClient(WithBaseURL(fs.server.URL))

	recs, err := client.Convert(context.Background(), []Input{
		{Type: IDTypeDOI, ID: "10.1038/s41586-020-2649-2"},
		{Type: IDTypeDOI, ID: "10.99999/nonsense.fake.doi"},
	})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}

	byID := make(map[string]Record)
	for _, r := range recs {
		byID[r.RequestedID] = r
	}
	good := byID["10.1038/s41586-020-2649-2"]
	if good.PMCID != "PMC7759461" {
		t.Errorf("expected PMCID PMC7759461, got %q", good.PMCID)
	}
	if good.Status == "error" {
		t.Errorf("good record should not have error status")
	}

	bad := byID["10.99999/nonsense.fake.doi"]
	if bad.PMCID != "" {
		t.Errorf("expected empty PMCID, got %q", bad.PMCID)
	}
	if bad.Status != "error" {
		t.Errorf("expected status=error, got %q", bad.Status)
	}
	if bad.ErrMsg == "" {
		t.Errorf("expected non-empty errmsg")
	}
}

func TestConvert_PMIDFixtureReturnsInteger(t *testing.T) {
	// Guards the json.Unmarshal mapping: NCBI returns pmid as a JSON number,
	// not a string. If we accidentally typed Record.PMID as string, this fixture
	// would fail to parse.
	fs := newFixtureServer(t, 200, loadFixture(t, "pmid_found.json"))
	client := NewClient(WithBaseURL(fs.server.URL))

	recs, err := client.Convert(context.Background(), []Input{
		{Type: IDTypePMID, ID: "32939066"},
	})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	if recs[0].PMID != 32939066 {
		t.Errorf("expected PMID 32939066, got %d", recs[0].PMID)
	}
	if recs[0].PMCID != "PMC7759461" {
		t.Errorf("expected PMCID PMC7759461, got %q", recs[0].PMCID)
	}
}

func TestConvert_GroupsByIDTypeAndIssuesSeparateBatches(t *testing.T) {
	// Mixing DOIs and PMIDs in a single call must produce two HTTP requests
	// (one per type), each with the correct idtype query param. NCBI rejects
	// mixed-type batches with parse_error.
	requestCount := 0
	seenIDTypes := make(map[string]bool)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		seenIDTypes[r.URL.Query().Get("idtype")] = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","records":[]}`))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	_, err := client.Convert(context.Background(), []Input{
		{Type: IDTypeDOI, ID: "10.1038/A"},
		{Type: IDTypePMID, ID: "12345"},
	})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if requestCount != 2 {
		t.Errorf("expected 2 HTTP requests, got %d", requestCount)
	}
	if !seenIDTypes["doi"] || !seenIDTypes["pmid"] {
		t.Errorf("expected both idtypes seen, got %v", seenIDTypes)
	}
}

func TestConvert_BatchingBoundary(t *testing.T) {
	// 250 IDs → 2 requests (200 + 50), not 1, not 250. The first batch must
	// have ≤ MaxBatchSize entries.
	requestCount := 0
	batchSizes := []int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		ids := r.URL.Query().Get("ids")
		batchSizes = append(batchSizes, strings.Count(ids, ",")+1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","records":[]}`))
	}))
	defer srv.Close()

	inputs := make([]Input, 250)
	for i := range inputs {
		inputs[i] = Input{Type: IDTypeDOI, ID: "10.1038/" + string(rune('A'+(i%26))) + ":" + string(rune('a'+(i/26)))}
	}

	client := NewClient(WithBaseURL(srv.URL))
	if _, err := client.Convert(context.Background(), inputs); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
	if batchSizes[0] != 200 || batchSizes[1] != 50 {
		t.Errorf("expected batch sizes [200, 50], got %v", batchSizes)
	}
}

func TestConvert_HTTP4xxIncludesBatchIDs(t *testing.T) {
	// On an HTTP 4xx, *APIError must carry the failing batch's IDs so the
	// caller can report what failed (don't silently drop). Note: NCBI returns
	// HTTP 400 for parse errors, so this is the path that fires in practice;
	// the response body's JSON envelope is not parsed when status >= 400.
	fs := newFixtureServer(t, 400, []byte(`Bad Request`))
	client := NewClient(WithBaseURL(fs.server.URL))

	_, err := client.Convert(context.Background(), []Input{
		{Type: IDTypeDOI, ID: "10.1038/A"},
		{Type: IDTypeDOI, ID: "10.1038/B"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if len(apiErr.BatchIDs) != 2 {
		t.Errorf("expected 2 batch IDs in error, got %d", len(apiErr.BatchIDs))
	}
	if !strings.Contains(apiErr.Error(), "10.1038/A") {
		t.Errorf("expected error message to include batch ID, got: %s", apiErr.Error())
	}
}

func TestConvert_JSONEnvelopeErrorIncludesBatchIDs(t *testing.T) {
	// Defensive coverage of the parsed-but-status-not-ok path: if NCBI ever
	// serves an error envelope with HTTP 200 (which today they don't, but the
	// shape is documented), the client must still raise an *APIError carrying
	// the batch IDs and the per-error code/message from the envelope.
	fs := newFixtureServer(t, 200, loadFixture(t, "parse_error.json"))
	client := NewClient(WithBaseURL(fs.server.URL))

	_, err := client.Convert(context.Background(), []Input{
		{Type: IDTypeDOI, ID: "10.1038/A"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != "invalid_dois" {
		t.Errorf("expected code from envelope, got %q", apiErr.Code)
	}
	if !strings.Contains(apiErr.Error(), "10.1038/A") {
		t.Errorf("expected error message to include batch ID, got: %s", apiErr.Error())
	}
}

func TestConvert_RateLimit429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`Too Many Requests`))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	_, err := client.Convert(context.Background(), []Input{{Type: IDTypeDOI, ID: "10.1038/A"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsRateLimited(err) {
		t.Errorf("expected IsRateLimited(err) to be true, got: %v", err)
	}
}

func TestConvert_EmptyInputReturnsNil(t *testing.T) {
	// No HTTP request should be made for an empty input list.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to NCBI")
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	recs, err := client.Convert(context.Background(), nil)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if recs != nil {
		t.Errorf("expected nil records, got %v", recs)
	}
}

func TestConvert_RealFixture_BothFound(t *testing.T) {
	// End-to-end smoke against a real captured response.
	fs := newFixtureServer(t, 200, loadFixture(t, "dois_both_found.json"))
	client := NewClient(WithBaseURL(fs.server.URL))

	recs, err := client.Convert(context.Background(), []Input{
		{Type: IDTypeDOI, ID: "10.1038/s41586-020-2649-2"},
		{Type: IDTypeDOI, ID: "10.1038/nature12373"},
	})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
	pmcByDOI := map[string]string{}
	for _, r := range recs {
		pmcByDOI[r.DOI] = r.PMCID
	}
	if pmcByDOI["10.1038/s41586-020-2649-2"] != "PMC7759461" {
		t.Errorf("PMCID mismatch: got %v", pmcByDOI)
	}
}
