package cmd

import (
	"github.com/spf13/cobra"

	"wtw/internal/update"
)

var updateYes bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for a new version and install it with approval",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return update.ManualUpdate(appVersion, updateYes)
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&updateYes, "yes", "y", false, "install update without prompting")
	rootCmd.AddCommand(updateCmd)
}
