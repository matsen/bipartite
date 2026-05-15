package main

import "github.com/matsen/bipartite/internal/ncbi"

// newNCBIClient constructs an NCBI client with the standard `tool` identifier
// and, optionally, the user-supplied `email` for NCBI usage attribution.
func newNCBIClient(email string) *ncbi.Client {
	opts := []ncbi.ClientOption{ncbi.WithTool(ncbi.DefaultTool)}
	if email != "" {
		opts = append(opts, ncbi.WithEmail(email))
	}
	return ncbi.NewClient(opts...)
}
