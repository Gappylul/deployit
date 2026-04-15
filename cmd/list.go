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
		fmt.Printf("%-20s %-15s %-10s %s\n", "NAME", "STATUS", "READY", "HOST")
		for _, app := range apps {
			ready, total, status := deploy.GetDeploymentStats(ctx, app.Name)

			statusDisplay := status
			if status == "Running" && ready == total {
				statusDisplay = "\033[32m● Running\033[0m"
			} else if status == "Error" {
				statusDisplay = "\033[31m● Error\033[0m"
			} else {
				statusDisplay = fmt.Sprintf("\033[33m● %s\033[0m", status)
			}

			readyStr := fmt.Sprintf("%d/%d", ready, total)
			fmt.Printf("%-20s %-24s %-10s %s\n",
				app.Name,
				statusDisplay,
				readyStr,
				app.Spec.Host,
			)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
