package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"wtw/internal/git"
	"wtw/internal/worktree"
)

func runCreate(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	customSetup, _ := cmd.Flags().GetString("setup")
	if customSetup != "" {
		abs, err := filepath.Abs(customSetup)
		if err != nil {
			return fmt.Errorf("invalid path: %s", customSetup)
		}
		customSetup = abs
	}

	setupScript, err := worktree.ResolveSetupScript(customSetup, repoRoot)
	if err != nil {
		return err
	}

	var branchName, baseDir string
	if len(args) > 0 {
		branchName = args[0]
	}
	if len(args) > 1 {
		baseDir = args[1]
	}

	originalDir, _ := os.Getwd()

	return worktree.Create(worktree.CreateConfig{
		BranchName:  branchName,
		BaseDir:     baseDir,
		SetupScript: setupScript,
		RepoRoot:    repoRoot,
		RepoName:    filepath.Base(repoRoot),
		OriginalDir: originalDir,
	})
}
