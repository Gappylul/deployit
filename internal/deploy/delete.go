package deploy

import (
	"context"
	"fmt"

	platformv1 "github.com/gappylul/webapp-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Delete(ctx context.Context, name string) error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, nil,
	).ClientConfig()
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	_ = platformv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	k8s, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("k8s client: %w", err)
	}

	projectLabel := client.MatchingLabels{"project": name}
	namespace := "default"

	fmt.Printf("Deleting project: %s...\n", name)

	webapp := &platformv1.WebApp{}
	err = k8s.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, webapp)
	if err == nil {
		if err := k8s.Delete(ctx, webapp); err != nil {
			fmt.Printf("⚠  Note: Could not delete WebApp CR: %v\n", err)
		} else {
			fmt.Println("[+] Deleted WebApp Custom Resource")
		}
	}

	deployments := &appsv1.Deployment{}
	if err := k8s.DeleteAllOf(ctx, deployments, client.InNamespace(namespace), projectLabel); err != nil {
		return fmt.Errorf("failed to delete deployments: %w", err)
	}
	fmt.Println("[+] Deleted database deployments")

	secrets := &corev1.Secret{}
	if err := k8s.DeleteAllOf(ctx, secrets, client.InNamespace(namespace), projectLabel); err != nil {
		return fmt.Errorf("failed to delete secrets: %w", err)
	}
	fmt.Println("[+] Deleted project secrets")

	pvcs := &corev1.PersistentVolumeClaim{}
	if err := k8s.DeleteAllOf(ctx, pvcs, client.InNamespace(namespace), projectLabel); err != nil {
		return fmt.Errorf("failed to delete storage: %w", err)
	}
	fmt.Println("[+] Deleted persistent storage (PVCs)")

	svcList := &corev1.ServiceList{}
	if err := k8s.List(ctx, svcList, client.InNamespace(namespace), projectLabel); err == nil {
		for _, svc := range svcList.Items {
			_ = k8s.Delete(ctx, &svc)
		}
	}
	fmt.Println("[+] Deleted networking services")

	fmt.Printf("\nProject '%s' has been fully decommissioned.\n", name)
	return nil
}
