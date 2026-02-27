package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"wtw/internal/git"
	"wtw/internal/worktree"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"i"},
	Short:   "Create a sample .wtwrc config in the current repo",
	Args:    cobra.NoArgs,
	RunE:    runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}
	return worktree.Init(repoRoot)
}
