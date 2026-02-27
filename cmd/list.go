package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"wtw/internal/git"
	"wtw/internal/worktree"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all worktrees and their paths",
	Args:    cobra.NoArgs,
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(_ *cobra.Command, _ []string) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}
	return worktree.List(repoRoot)
}
