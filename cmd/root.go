// Package cmd defines the cobra commands. Each file owns one subcommand and
// delegates all logic to internal/worktree.
package cmd

import (
	"github.com/spf13/cobra"
)

// rootCmd is the default action: `wtw [branch]` creates a worktree.
// SilenceUsage/SilenceErrors prevent cobra from printing usage on every error.
var rootCmd = &cobra.Command{
	Use:   "wtw [branch-name] [location]",
	Short: "Git worktree helper",
	Long: `wtw — Git Worktree Helper

Creates a new git worktree for the given branch (or prompts for one).
Optionally runs a .wtwrc setup script in the new worktree.`,
	Args:          cobra.MaximumNArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runCreate,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("setup", "c", "", "path to a setup script to run in the new worktree")
}
