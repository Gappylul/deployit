package cmd

import (
	"context"
	"fmt"

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

		if err := deploy.Delete(ctx, name); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		fmt.Printf("✓ deleted %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
