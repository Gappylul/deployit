package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets <app-name> [KEY=VALUE KEY2=VALUE2...]",
	Short: "Manage secrets for an application",
	Long:  "Lists, sets or updates secrets. Existing keys are preserved unless explicitly overwritten.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		secretName := fmt.Sprintf("%s-secrets", appName)

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		config, err := kubeConfig.ClientConfig()
		if err != nil {
			return fmt.Errorf("failed to load kubeconfig: %w", err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("failed to connect to cluster: %w", err)
		}

		secretClient := clientset.CoreV1().Secrets("default")

		data := make(map[string][]byte)
		existingSecret, err := secretClient.Get(context.TODO(), secretName, metav1.GetOptions{})

		if len(args) == 1 {
			if err != nil {
				return fmt.Errorf("no secrets found for %s", appName)
			}
			fmt.Printf("Current secrets for '%s':\n", appName)
			for k := range existingSecret.Data {
				fmt.Printf("	- %s\n", k)
			}
			return nil
		}

		isNew := false
		if err != nil {
			if errors.IsNotFound(err) {
				isNew = true
			} else {
				return fmt.Errorf("failed to fetch existing secrets: %w", err)
			}
		} else {
			data = existingSecret.Data
		}

		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format '%s', must be KEY=VALUE", arg)
			}
			data[parts[0]] = []byte(parts[1])
		}

		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "deployit",
					"app":                          appName,
				},
			},
			Data: data,
		}

		if isNew {
			_, err = secretClient.Create(context.TODO(), newSecret, metav1.CreateOptions{})
			fmt.Printf("✓ Created new secret store for %s\n", appName)
		} else {
			_, err = secretClient.Update(context.TODO(), newSecret, metav1.UpdateOptions{})
			fmt.Printf("✓ Updated secrets for %s (merged %d keys)\n", appName, len(args)-1)
		}

		if err != nil {
			return fmt.Errorf("failed to save secret: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(secretsCmd)
}
