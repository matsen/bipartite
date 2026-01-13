package main

// Exit codes as defined in contracts/cli.md
const (
	ExitSuccess     = 0 // Success
	ExitError       = 1 // General error (invalid arguments, runtime failure)
	ExitConfigError = 2 // Configuration error (missing config, invalid paths)
	ExitDataError   = 3 // Data error (malformed input, validation failure)
)
