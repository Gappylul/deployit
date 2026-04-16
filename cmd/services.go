package cmd

import (
	"context"
	"fmt"

	"github.com/gappylul/deployit/internal/deploy"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var servicesCmd = &cobra.Command{
	Use:   "services <app-name>",
	Short: "Show attached services (like redis) for an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		clientset, _ := deploy.GetClientset()

		labelSelector := fmt.Sprintf("project=%s", appName)
		pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Attached services for '%s':\n", appName)
		if len(pods.Items) == 0 {
			fmt.Println("  (none)")
			return nil
		}

		for _, pod := range pods.Items {
			status := pod.Status.Phase
			fmt.Printf("  ● %-15s [%s]\n", pod.Name, status)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(servicesCmd)
}
