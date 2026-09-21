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

	// Collision-check exit codes (bip epic collide). Deliberately outside
	// the 1-6 block above: 1 already means "general error" and 2 means
	// "configuration error", and main.go maps EVERY RunE error to 1 — so a
	// distinct "could not check" verdict cannot arrive through a returned
	// error and cannot reuse 2 without contradicting its current meaning.
	// The command returns nil from RunE and exits directly.
	//
	// 11 dominates 10: a caller acting on "found" believes the pool was
	// fully examined.
	ExitCollisionFound       = 10 // A real overlap was found
	ExitCollisionUncheckable = 11 // Some part of the pool could not be examined

	// ASTA exit codes (from contracts/cli.md)
	ExitASTANotFound  = 1 // Resource not found in ASTA
	ExitASTAAuthError = 2 // Missing or invalid ASTA_API_KEY
	ExitASTAAPIError  = 3 // API error (rate limit, network)
)
