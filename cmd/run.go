package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"wtw/internal/git"
	"wtw/internal/worktree"
)

var runCmd = &cobra.Command{
	Use:     "run-wtwrc",
	Aliases: []string{"rrc"},
	Short:   "Re-run the setup script in the current worktree",
	Args:    cobra.NoArgs,
	RunE:    runSetup,
}

func init() {
	runCmd.Flags().StringP("setup", "c", "", "path to a setup script to run")
	rootCmd.AddCommand(runCmd)
}

func runSetup(cmd *cobra.Command, _ []string) error {
	worktreeRoot, mainRepoRoot, err := git.RequireWorktree("run-wtwrc")
	if err != nil {
		return err
	}

	customSetup, _ := cmd.Flags().GetString("setup")
	if customSetup != "" {
		abs, err := filepath.Abs(customSetup)
		if err != nil {
			return err
		}
		customSetup = abs
	}

	originalDir, _ := os.Getwd()

	return worktree.RunSetup(worktree.RunSetupConfig{
		SetupScript:  customSetup,
		WorktreeRoot: worktreeRoot,
		MainRepoRoot: mainRepoRoot,
		OriginalDir:  originalDir,
	})
}
