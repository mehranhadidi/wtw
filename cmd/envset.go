package cmd

import (
	"github.com/spf13/cobra"

	"wtw/internal/worktree"
)

var envSetCmd = &cobra.Command{
	Use:   "env-set <file> KEY=VALUE [KEY=VALUE ...]",
	Short: "Set or add key-value pairs in an env file",
	Long: `Set or add key-value pairs in an env file.

Existing keys are updated in-place; keys that do not exist are appended at
the end. Comment lines and blank lines are left unchanged.`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEnvSet,
}

func init() {
	rootCmd.AddCommand(envSetCmd)
}

func runEnvSet(_ *cobra.Command, args []string) error {
	return worktree.EnvSet(worktree.EnvSetConfig{
		File:  args[0],
		Pairs: args[1:],
	})
}
