package cmd

import (
	"fmt"

	"github.com/gappylul/deployit/internal/cloudflare"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup --host <hostname>",
	Short: "Remove a hostname from Cloudflare without touching the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cleanupHost == "" {
			return fmt.Errorf("--host is required")
		}
		cf, err := cloudflare.NewClient()
		if err != nil {
			return err
		}
		return cf.RemoveHostname(cleanupHost)
	},
}

var cleanupHost string

func init() {
	cleanupCmd.Flags().StringVar(&cleanupHost, "host", "", "hostname to remove from Cloudflare")
	rootCmd.AddCommand(cleanupCmd)
}
