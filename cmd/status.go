package cmd

import (
	"fmt"

	"github.com/gappylul/deployit/internal/cloudflare"
	"github.com/gappylul/deployit/internal/kube"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var statusCmd = &cobra.Command{
	Use:   "status <app-name>",
	Short: "Get the live status and event history of an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]

		fmt.Printf("Checking Cloudflare Tunnel for %s...\n", appName)
		cfClient, err := cloudflare.NewClient()
		if err != nil {
			fmt.Printf("⚠  Cloudflare check skipped: %v\n", err)
		} else {
			ingress, err := cfClient.GetTunnelConfig()
			if err != nil {
				fmt.Printf("!  Cloudflare API Error: %v\n", err)
			} else {
				found := false

				for _, rule := range ingress {
					if rule.Hostname != "" && (rule.Hostname == appName || rule.Hostname == fmt.Sprintf("%s.gappy.hu", appName)) {
						found = true
						break
					}
				}
				if found {
					fmt.Println("-> Cloudflare: Traffic is correctly routed to the tunnel.")
				} else {
					fmt.Println("! Cloudflare: No tunnel route found for this app.")
				}
			}
		}

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil).ClientConfig()
		if err != nil {
			return err
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}

		return kube.PrintAppStatus(clientset, appName)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
