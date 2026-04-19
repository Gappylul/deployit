package cmd

import (
	"github.com/gappylul/deployit/internal/kube"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var topCmd = &cobra.Command{
	Use:   "top <app-name>",
	Short: "View CPU and Memory usage for an application",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil).ClientConfig()
		if err != nil {
			return err
		}

		return kube.PrintTop(config, appName)
	},
}

func init() {
	rootCmd.AddCommand(topCmd)
}
