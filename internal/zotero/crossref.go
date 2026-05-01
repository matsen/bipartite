package zotero

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/reference"
)

const (
	crossrefBaseURL = "https://api.crossref.org"
	crossrefTimeout = 15 * time.Second
)

// crossrefResponse represents the CrossRef API response.
type crossrefResponse struct {
	Message crossrefWork `json:"message"`
}

type crossrefWork struct {
	DOI            string           `json:"DOI"`
	Title          []string         `json:"title"`
	ContainerTitle []string         `json:"container-title"`
	Abstract       string           `json:"abstract"`
	Author         []crossrefAuthor `json:"author"`
	Published      crossrefDate     `json:"published"`
	ISSN           []string         `json:"ISSN"`
}

type crossrefAuthor struct {
	Given  string `json:"given"`
	Family string `json:"family"`
	Name   string `json:"name"` // For institutional authors
}

type crossrefDate struct {
	DateParts [][]int `json:"date-parts"`
}

// LookupDOI fetches paper metadata from CrossRef by DOI.
// This is a free API that doesn't require authentication.
func LookupDOI(ctx context.Context, doi string) (reference.Reference, error) {
	client := &http.Client{Timeout: crossrefTimeout}

	endpoint := fmt.Sprintf("%s/works/%s", crossrefBaseURL, url.PathEscape(doi))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return reference.Reference{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return reference.Reference{}, fmt.Errorf("CrossRef lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return reference.Reference{}, fmt.Errorf("DOI not found in CrossRef: %s", doi)
	}
	if resp.StatusCode != http.StatusOK {
		return reference.Reference{}, fmt.Errorf("CrossRef API error (status %d)", resp.StatusCode)
	}

	var result crossrefResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return reference.Reference{}, fmt.Errorf("decoding CrossRef response: %w", err)
	}

	return mapCrossRefToReference(result.Message)
}

func mapCrossRefToReference(work crossrefWork) (reference.Reference, error) {
	title := ""
	if len(work.Title) > 0 {
		title = work.Title[0]
	}
	if title == "" {
		return reference.Reference{}, fmt.Errorf("no title in CrossRef record")
	}

	authors := make([]reference.Author, 0, len(work.Author))
	for _, a := range work.Author {
		if a.Name != "" {
			authors = append(authors, reference.Author{Last: a.Name})
		} else {
			authors = append(authors, reference.Author{First: a.Given, Last: a.Family})
		}
	}

	var pubDate reference.PublicationDate
	if len(work.Published.DateParts) > 0 && len(work.Published.DateParts[0]) > 0 {
		parts := work.Published.DateParts[0]
		pubDate.Year = parts[0]
		if len(parts) >= 2 {
			pubDate.Month = parts[1]
		}
		if len(parts) >= 3 {
			pubDate.Day = parts[2]
		}
	}

	venue := ""
	if len(work.ContainerTitle) > 0 {
		venue = work.ContainerTitle[0]
	}

	// Strip HTML tags from abstract (CrossRef often includes JATS XML)
	abstract := stripHTMLTags(work.Abstract)

	id := generateCiteKey(authors, pubDate.Year, title)

	ref := reference.Reference{
		ID:        id,
		DOI:       work.DOI,
		Title:     title,
		Authors:   authors,
		Abstract:  abstract,
		Venue:     venue,
		Published: pubDate,
	}

	return ref, nil
}

// stripHTMLTags removes HTML/XML tags and unescapes HTML entities.
// CrossRef abstracts commonly contain JATS XML markup and entities like &amp;.
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(html.UnescapeString(result.String()))
}
