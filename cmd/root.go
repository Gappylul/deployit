package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/gappylul/deployit/internal/version"
	"github.com/spf13/cobra"
)

var (
	newVersion     string
	versionCheckWG sync.WaitGroup
)

var rootCmd = &cobra.Command{
	Use:   "deployit",
	Short: "Deploy any project to your homelab with one command",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		versionCheckWG.Add(1)
		go func() {
			defer versionCheckWG.Done()
			newVersion = version.CheckForUpdate()
		}()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		versionCheckWG.Wait()
		if newVersion != "" && newVersion != version.CurrentVersion {
			fmt.Printf("\n! A newer version of deployit is available: %s (Current: %s)\n", newVersion, version.CurrentVersion)
			fmt.Println("! Run 'go install github.com/gappylul/deployit@latest' to update.")
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
