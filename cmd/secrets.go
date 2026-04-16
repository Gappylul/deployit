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

func getClientset() (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

var secretsCmd = &cobra.Command{
	Use:   "secrets <app-name> [KEY=VALUE...]",
	Short: "List or set secrets for an application",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		secretName := fmt.Sprintf("%s-secrets", appName)

		clientset, err := getClientset()
		if err != nil {
			return err
		}
		secretClient := clientset.CoreV1().Secrets("default")

		existingSecret, err := secretClient.Get(context.TODO(), secretName, metav1.GetOptions{})

		if len(args) == 1 {
			if err != nil {
				return fmt.Errorf("no secrets found for %s", appName)
			}
			fmt.Printf("Current secrets for '%s':\n", appName)
			for k := range existingSecret.Data {
				fmt.Printf("   - %s\n", k)
			}
			return nil
		}

		data := make(map[string][]byte)
		isNew := false
		if err != nil {
			if errors.IsNotFound(err) {
				isNew = true
			} else {
				return err
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
		} else {
			_, err = secretClient.Update(context.TODO(), newSecret, metav1.UpdateOptions{})
		}

		if err == nil {
			fmt.Printf("✓ Secrets updated for %s\n", appName)
		}
		return err
	},
}

var secretsDeleteCmd = &cobra.Command{
	Use:   "delete <app-name> <key1> [key2...]",
	Short: "Remove one or more keys from an application's secrets",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		keysToDelete := args[1:]
		secretName := fmt.Sprintf("%s-secrets", appName)

		clientset, err := getClientset()
		if err != nil {
			return err
		}
		secretClient := clientset.CoreV1().Secrets("default")

		secret, err := secretClient.Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("no secrets found for app '%s'", appName)
			}
			return err
		}

		for _, key := range keysToDelete {
			if _, exists := secret.Data[key]; exists {
				delete(secret.Data, key)
				fmt.Printf("   - removing %s\n", key)
			} else {
				fmt.Printf("   - %s not found, skipping\n", key)
			}
		}

		_, err = secretClient.Update(context.TODO(), secret, metav1.UpdateOptions{})
		if err == nil {
			fmt.Printf("✓ Updated secrets for %s\n", appName)
		}
		return err
	},
}

func init() {
	secretsCmd.AddCommand(secretsDeleteCmd)
	rootCmd.AddCommand(secretsCmd)
}
