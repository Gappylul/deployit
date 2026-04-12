package cmd

import (
	"context"
	"fmt"

	"github.com/gappylul/deployit/internal/deploy"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed apps",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		apps, err := deploy.List(ctx)
		if err != nil {
			return err
		}
		if len(apps) == 0 {
			fmt.Println("no apps deployed")
			return nil
		}
		fmt.Printf("%-20s %-10s %s\n", "NAME", "REPLICAS", "HOST")
		for _, app := range apps {
			fmt.Printf("%-20s %-10d %s\n", app.Name, *app.Spec.Replicas, app.Spec.Host)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
