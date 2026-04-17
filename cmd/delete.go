package cmd

import (
	"context"
	"fmt"

	"github.com/gappylul/deployit/internal/cloudflare"
	"github.com/gappylul/deployit/internal/deploy"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a deployed app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		name := args[0]

		var hostToDelete string
		if deleteHost != "" {
			hostToDelete = deleteHost
		}

		if err := deploy.Delete(ctx, name); err != nil {
			return fmt.Errorf("delete: %w", err)
		}
		fmt.Printf("✓ deleted %s\n", name)

		if hostToDelete != "" {
			cf, err := cloudflare.NewClient()
			if err == nil {
				fmt.Printf("   Removing Cloudflare record for %s...\n", hostToDelete)
				err := cf.RemoveHostname(hostToDelete)
				if err != nil {
					return err
				}
			}
		}
		return nil
	},
}

var deleteHost string

func init() {
	deleteCmd.Flags().StringVar(&deleteHost, "host", "", "hostname to remove from Cloudflare tunnel")
	rootCmd.AddCommand(deleteCmd)
}
