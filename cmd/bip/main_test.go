package main

import (
	"io"
	"testing"
)

// TestUnknownSubcommandFails checks that every command group rejects an
// unknown subcommand, and still prints help with no arguments.
func TestUnknownSubcommandFails(t *testing.T) {
	rejectUnknownSubcommands(rootCmd)
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	defer rootCmd.SetArgs(nil)
	groups := 0
	for _, c := range rootCmd.Commands() {
		if !c.HasSubCommands() {
			continue
		}
		groups++
		rootCmd.SetArgs([]string{c.Name(), "nosuchcmd"})
		if err := rootCmd.Execute(); err == nil {
			t.Errorf("bip %s nosuchcmd: no error", c.Name())
		}
		rootCmd.SetArgs([]string{c.Name()})
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("bip %s: %v", c.Name(), err)
		}
	}
	if groups == 0 {
		t.Fatal("no command groups found")
	}
}
