package main

// Exit codes as defined in contracts/cli.md
const (
	ExitSuccess       = 0 // Success
	ExitError         = 1 // General error (invalid arguments, runtime failure)
	ExitConfigError   = 2 // Configuration error (missing config, invalid paths) / Index not found (Phase II)
	ExitDataError     = 3 // Data error (malformed input, validation failure) / Ollama not available (Phase II)
	ExitNoAbstract    = 4 // Paper has no abstract (Phase II)
	ExitModelNotFound = 5 // Embedding model not found (Phase II)
	ExitIndexStale    = 6 // Semantic index is stale (Phase II)

	// ASTA exit codes (from contracts/cli.md)
	ExitASTANotFound  = 1 // Resource not found in ASTA
	ExitASTAAuthError = 2 // Missing or invalid ASTA_API_KEY
	ExitASTAAPIError  = 3 // API error (rate limit, network)
)
