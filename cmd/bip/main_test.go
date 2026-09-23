package main

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestUnknownSubcommandFails checks that every command group, at any depth,
// rejects an unknown subcommand and still prints help with no arguments.
func TestUnknownSubcommandFails(t *testing.T) {
	rejectUnknownSubcommands(rootCmd)
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	defer rootCmd.SetArgs(nil)
	groups := 0
	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		for _, sub := range c.Commands() {
			p := append(append([]string{}, path...), sub.Name())
			if !sub.HasSubCommands() {
				continue
			}
			groups++
			name := "bip " + strings.Join(p, " ")
			rootCmd.SetArgs(append(p, "nosuchcmd"))
			if err := rootCmd.Execute(); err == nil {
				t.Errorf("%s nosuchcmd: no error", name)
			}
			rootCmd.SetArgs(p)
			if err := rootCmd.Execute(); err != nil {
				t.Errorf("%s: %v", name, err)
			}
			walk(sub, p)
		}
	}
	walk(rootCmd, nil)
	if groups == 0 {
		t.Fatal("no command groups found")
	}
	t.Logf("%d command groups checked", groups)
}
