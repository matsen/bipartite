package main

import "testing"

func TestColorOutput_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if colorOutput() {
		t.Error("expected no color with NO_COLOR set")
	}
}
