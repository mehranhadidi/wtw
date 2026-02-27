package cmd

import (
	"github.com/spf13/cobra"

	"wtw/internal/git"
	"wtw/internal/worktree"
)

var doneCmd = &cobra.Command{
	Use:     "done",
	Aliases: []string{"d"},
	Short:   "Remove the current worktree",
	Long:    "Remove the current worktree. Must be run inside a worktree, not the main repo.",
	Args:    cobra.NoArgs,
	RunE:    runDone,
}

func init() {
	rootCmd.AddCommand(doneCmd)
}

func runDone(_ *cobra.Command, _ []string) error {
	worktreeRoot, mainRepoRoot, err := git.RequireWorktree("done")
	if err != nil {
		return err
	}
	return worktree.Remove(worktree.RemoveConfig{
		WorktreeRoot: worktreeRoot,
		MainRepoRoot: mainRepoRoot,
	})
}
