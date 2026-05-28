package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/gitx"
	"github.com/spf13/cobra"
)

// worktreeCmd is the hidden parent of the worktree plumbing subcommands.
// These commands are not part of bip's user-facing surface — they exist so
// the bip-pr-land skill can call tested Go (in internal/gitx) instead of
// re-implementing primary-clone resolution and `git worktree remove --force`
// as untested shell pipelines in markdown. The command stays Hidden=true so
// it does not appear in `bip --help`, completions, or docs.
var worktreeCmd = &cobra.Command{
	Use:    "worktree",
	Short:  "Plumbing for git worktree operations (used by skills, hidden)",
	Hidden: true,
}

var worktreePrimaryCmd = &cobra.Command{
	Use:   "primary [dir]",
	Short: "Print the primary clone directory if [dir] is a linked worktree",
	Long: `Print the primary clone directory for a linked worktree.

Exit codes:
  0  - [dir] (default: CWD) is a linked worktree; primary path is on stdout.
  3  - [dir] is the primary clone (or not a worktree at all); no stdout.
  1  - [dir] is not a git repository, or git itself failed.

Designed for shell use:
    if PRIMARY=$(bip worktree primary 2>/dev/null); then
        cd "$PRIMARY"
        ...
    fi`,
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	RunE:   runWorktreePrimary,
}

var worktreeRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Remove the worktree at <path> (auto-resolves the primary clone)",
	Long: `Remove a linked worktree.

Equivalent to running 'git worktree remove --force <path>' from the primary
clone, but the primary is auto-resolved from <path> itself, so the caller
does not need to know or cd into it.`,
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   runWorktreeRemove,
}

var (
	worktreeRemoveForce bool
	worktreeRemoveFrom  string
)

func init() {
	worktreeCmd.AddCommand(worktreePrimaryCmd)
	worktreeCmd.AddCommand(worktreeRemoveCmd)
	worktreeRemoveCmd.Flags().BoolVar(&worktreeRemoveForce, "force", true,
		"Pass --force to git worktree remove (default true; required after squash merge)")
	worktreeRemoveCmd.Flags().StringVar(&worktreeRemoveFrom, "from", "",
		"Primary clone directory (default: auto-resolved from <path>)")
	rootCmd.AddCommand(worktreeCmd)
}

func runWorktreePrimary(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if len(args) == 1 {
		dir = args[0]
	}
	inWT, err := gitx.IsInWorktree(dir)
	if err != nil {
		// Not a git repository, or git failed. Surface as a normal error
		// (exit 1) so the caller can distinguish "not in a worktree"
		// (exit 3) from "not a repo at all."
		return err
	}
	if !inWT {
		os.Exit(3)
	}
	primary, err := gitx.PrimaryCloneDir(dir)
	if err != nil {
		return err
	}
	fmt.Println(primary)
	return nil
}

func runWorktreeRemove(cmd *cobra.Command, args []string) error {
	path := args[0]
	primary := worktreeRemoveFrom
	if primary == "" {
		p, err := gitx.PrimaryCloneDir(path)
		if err != nil {
			return fmt.Errorf("could not resolve primary clone from %s: %w (pass --from)", path, err)
		}
		primary = p
	}
	return gitx.RemoveWorktree(primary, path, worktreeRemoveForce)
}
