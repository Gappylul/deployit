package deploy

import (
	"context"
	"fmt"

	platformv1 "github.com/gappylul/webapp-operator/api/v1"
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
	if err := platformv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add scheme: %w", err)
	}

	k8s, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("k8s client: %w", err)
	}

	webapp := &platformv1.WebApp{}
	if err := k8s.Get(ctx, client.ObjectKey{Name: name, Namespace: "default"}, webapp); err != nil {
		return fmt.Errorf("not found: %s", name)
	}

	return k8s.Delete(ctx, webapp)
}
