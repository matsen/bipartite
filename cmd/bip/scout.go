package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/scout"
	"github.com/spf13/cobra"
)

var scoutServerFlag string

var scoutCmd = &cobra.Command{
	Use:   "scout",
	Short: "Check remote server availability via SSH",
	Long: `Check CPU, memory, load, and GPU availability on remote servers.

Reads server definitions from servers.yml in the nexus directory,
connects via SSH in parallel, and outputs JSON (default) or a
human-readable table (--human).`,
	RunE: runScout,
}

func init() {
	scoutCmd.Flags().StringVar(&scoutServerFlag, "server", "", "Check a specific server by name")
	rootCmd.AddCommand(scoutCmd)
}

func runScout(cmd *cobra.Command, args []string) error {
	// Find servers.yml in nexus directory
	nexusDir := config.GetNexusPath()
	if nexusDir == "" {
		fmt.Fprintln(os.Stderr, config.HelpfulConfigMessage())
		os.Exit(ExitConfigError)
	}

	configPath := filepath.Join(nexusDir, "servers.yml")
	cfg, err := scout.LoadConfig(configPath)
	if err != nil {
		exitWithError(ExitConfigError, "%v", err)
	}

	// Expand server patterns
	servers, err := scout.ExpandServers(cfg)
	if err != nil {
		exitWithError(ExitConfigError, "%v", err)
	}

	// Filter to single server if --server flag is set
	if scoutServerFlag != "" {
		var filtered []scout.Server
		for _, s := range servers {
			if s.Name == scoutServerFlag {
				filtered = append(filtered, s)
				break
			}
		}
		if len(filtered) == 0 {
			// List available server names for helpful error
			names := make([]string, len(servers))
			for i, s := range servers {
				names[i] = s.Name
			}
			exitWithError(ExitConfigError, "unknown server %q. Available servers: %v", scoutServerFlag, names)
		}
		servers = filtered
	}

	// Create SSH client
	sshClient, err := scout.NewSSHClient(cfg.SSH)
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}
	defer sshClient.Close()

	// Check all servers
	result := scout.CheckAllServers(sshClient, servers)

	// Output result
	if humanOutput {
		fmt.Print(scout.FormatTable(result))
	} else {
		if err := outputJSON(result); err != nil {
			exitWithError(ExitError, "encoding JSON: %v", err)
		}
	}

	return nil
}
