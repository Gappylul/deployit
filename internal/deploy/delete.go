package deploy

import (
	"context"
	"fmt"

	platformv1 "github.com/gappylul/webapp-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Delete(ctx context.Context, name string) error {
	k8s, err := getControllerClient()
	if err != nil {
		return err
	}

	namespace := "default"
	fmt.Printf("Deleting project: %s...\n", name)

	deleteWebAppCR(ctx, k8s, name, namespace)

	cleanupKnownExtensions(ctx, k8s, name, namespace)

	cleanupByLabel(ctx, k8s, name, namespace)

	fmt.Printf("\nProject '%s' has been fully decommissioned.\n", name)
	return nil
}

func getControllerClient() (client.Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, nil,
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	_ = platformv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	return client.New(config, client.Options{Scheme: scheme})
}

func deleteWebAppCR(ctx context.Context, k8s client.Client, name, namespace string) {
	webapp := &platformv1.WebApp{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	if err := k8s.Delete(ctx, webapp); err != nil {
		fmt.Printf("⚠  WebApp CR not found (already deleted?)\n")
	} else {
		fmt.Println("[+] Deleted WebApp Custom Resource")
	}
}

func cleanupKnownExtensions(ctx context.Context, k8s client.Client, name, namespace string) {
	prefixes := []string{"redis-", "postgres-"}
	for _, prefix := range prefixes {
		extName := prefix + name

		dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: extName, Namespace: namespace}}
		if err := k8s.Delete(ctx, dep); err == nil {
			fmt.Printf("[+] Deleted extension deployment: %s\n", extName)
		}

		svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: extName, Namespace: namespace}}
		if err := k8s.Delete(ctx, svc); err == nil {
			fmt.Printf("[+] Deleted extension service: %s\n", extName)
		}
	}
}

func cleanupByLabel(ctx context.Context, k8s client.Client, name, namespace string) {
	projectLabel := client.MatchingLabels{"project": name}
	inNs := client.InNamespace(namespace)

	_ = k8s.DeleteAllOf(ctx, &appsv1.Deployment{}, inNs, projectLabel)
	fmt.Println("[+] Deleted database deployments")

	_ = k8s.DeleteAllOf(ctx, &corev1.Secret{}, inNs, projectLabel)
	fmt.Println("[+] Deleted project secrets")

	_ = k8s.DeleteAllOf(ctx, &corev1.PersistentVolumeClaim{}, inNs, projectLabel)
	fmt.Println("[+] Deleted persistent storage (PVCs)")

	svcList := &corev1.ServiceList{}
	if err := k8s.List(ctx, svcList, inNs, projectLabel); err == nil {
		for _, svc := range svcList.Items {
			_ = k8s.Delete(ctx, &svc)
		}
	}
	fmt.Println("[+] Deleted networking services")
}
