package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// outputJSON writes a value as formatted JSON to stdout.
func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// outputJSONCompact writes a value as compact JSON to stdout.
func outputJSONCompact(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(v)
}

// outputHuman writes a human-readable string to stdout.
func outputHuman(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// outputError writes an error message to stderr and returns the exit code.
func outputError(code int, format string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	return code
}

// StatusResponse is a generic response for commands that return status.
type StatusResponse struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}

// ConfigResponse is the response for config get commands.
type ConfigResponse struct {
	PDFRoot   string `json:"pdf_root,omitempty"`
	PDFReader string `json:"pdf_reader,omitempty"`
}

// UpdateResponse is the response for config set commands.
type UpdateResponse struct {
	Status string `json:"status"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

// ErrorResponse is a JSON error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// outputErrorJSON writes an error as JSON and returns the exit code.
func outputErrorJSON(code int, message string) int {
	outputJSON(ErrorResponse{Error: message})
	return code
}
