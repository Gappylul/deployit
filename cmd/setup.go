package cmd

import (
	"context"
	"fmt"

	"github.com/gappylul/deployit/internal/bootstrap"
	"github.com/gappylul/deployit/internal/version"
	"github.com/spf13/cobra"
)

var setupDomain string

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Prepare the cluster for the Cloud",
	Long:  `Installs the WebApp Custom Resource Definitions, RBAC permissions, and the webapp-operator manager.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if setupDomain == "" {
			return fmt.Errorf("domain is required. Example: --domain yourdomain.com")
		}

		fmt.Println("Checking for latest operator version...")
		latestOp := version.GetLatestOperatorVersion()
		fmt.Printf("Using webapp-operator %s\n", latestOp)

		config := bootstrap.SetupConfig{
			Domain:          setupDomain,
			OperatorVersion: latestOp,
		}

		ctx := context.Background()
		if err := bootstrap.RunSetup(ctx, config); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}

		fmt.Println("\nCluster is ready! You can now use 'deployit deploy'.")
		return nil
	},
}

func init() {
	setupCmd.Flags().StringVar(&setupDomain, "domain", "", "The base domain for your apps")
	rootCmd.AddCommand(setupCmd)
}
