package cmd

import (
	"fmt"

	"github.com/gappylul/deployit/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current version of deployit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("deployit %s", version.CurrentVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
