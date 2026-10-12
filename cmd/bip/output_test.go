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
