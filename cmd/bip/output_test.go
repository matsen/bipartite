package main

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestFormatAuthorShortNonASCIIInitial(t *testing.T) {
	got := formatAuthorShort(reference.Author{First: "Álvaro", Last: "Ciudad"})
	if got != "Ciudad Á" {
		t.Errorf("formatAuthorShort = %q, want %q", got, "Ciudad Á")
	}
}

func TestAbbreviateAuthorNameNonASCII(t *testing.T) {
	if got := abbreviateAuthorName("Łukasz Nowak"); got != "Nowak Ł" {
		t.Errorf("abbreviateAuthorName = %q, want %q", got, "Nowak Ł")
	}
}
