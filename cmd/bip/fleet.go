package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/matsen/bipartite/internal/config"
	"github.com/spf13/cobra"
)

// exitCouldNotCheck is what the fleet check scripts exit when they cannot
// give an answer. The wrappers use it for their own failures too, so a
// missing config never reads as "found something" (1) or "clear" (0).
const exitCouldNotCheck = 2

var fleetCmd = &cobra.Command{
	Use:   "fleet",
	Short: "Commands for a conductor's pool of slots",
	Long: `Commands that act on a conductor's pool of slots.

Run them from the conductor clone: each reads .epic-config.json in the
current directory for clone_root and clone_names.`,
}

func init() {
	rootCmd.AddCommand(fleetCmd)
	fleetCmd.AddCommand(newFleetScriptCmd("currency", "clone-currency.sh",
		"Report whether each idle slot is current with main",
		func(cwd, root string) []string { return []string{cwd, root} }))
	fleetCmd.AddCommand(newFleetScriptCmd("collisions", "fleet-collisions.sh",
		"Report live branches that touch the same files, and stale or missing status files",
		func(_, root string) []string { return []string{root} }))
}

// newFleetScriptCmd wraps skills/lib/<script>, run with bash from the
// skills symlink so it is the current copy, not one frozen into this binary.
func newFleetScriptCmd(use, script, short string, args func(cwd, root string) []string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long: short + `.

Runs ~/.claude/skills/lib/` + script + ` (the symlink "make symlink-skills"
creates) against the clone_root in ./.epic-config.json, and exits with the
script's code, where 2 means could not check. Its own failures (no config,
a worker's frozen copy of the config, no script) also exit 2.`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			home, _ := os.UserHomeDir()
			os.Exit(runFleetScript(cwdOrEmpty(), home, script, args, os.Stdout, os.Stderr))
		},
	}
}

func cwdOrEmpty() string {
	cwd, _ := os.Getwd()
	return cwd
}

// runFleetScript runs home/.claude/skills/lib/script from cwd and returns
// the exit code to use.
func runFleetScript(cwd, home, script string, args func(cwd, root string) []string, stdout, stderr io.Writer) int {
	fail := func(format string, a ...interface{}) int {
		fmt.Fprintf(stderr, "bip fleet: "+format+" -- NOT a clean check\n", a...)
		return exitCouldNotCheck
	}
	if cwd == "" {
		return fail("cannot read the current directory")
	}
	cfg, err := loadEpicConfig(cwd)
	if err != nil {
		return fail("%v", err)
	}
	// Every worker clone carries a copy of the config taken at spawn time.
	conductor := cwd
	if cfg.MainCheckout != "" {
		if !samePath(cfg.MainCheckout, cwd) {
			return fail("%s is a worker's frozen copy (main_checkout=%s); run from %s", epicConfigName, cfg.MainCheckout, cfg.MainCheckout)
		}
		// clone-currency.sh compares its conductor argument to
		// main_checkout literally, so hand it main_checkout's spelling.
		conductor = cfg.MainCheckout
	}
	path := filepath.Join(home, ".claude", "skills", "lib", script)
	if _, err := os.Stat(path); err != nil {
		return fail("%s not found; run `make symlink-skills` in the bipartite repo", path)
	}
	cmd := exec.Command("bash", append([]string{path}, args(conductor, resolveCloneRoot(cwd, cfg))...)...)
	cmd.Dir = cwd
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exitErr) && exitErr.ExitCode() >= 0:
		return exitErr.ExitCode()
	default:
		return fail("running %s: %v", path, err)
	}
}

// samePath reports whether a and b name the same directory, following
// symlinks and a leading tilde.
func samePath(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(config.ExpandTilde(a))
	rb, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && ra == rb
}
