package deploy

import (
	"context"
	"fmt"

	platformv1 "github.com/gappylul/webapp-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Deploy(ctx context.Context, name, image, host string, replicas int32) error {
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

	webapp := &platformv1.WebApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: platformv1.WebAppSpec{
			Image:    image,
			Replicas: &replicas,
			Host:     host,
		},
	}

	err = k8s.Create(ctx, webapp)
	if err != nil {
		existing := &platformv1.WebApp{}
		if getErr := k8s.Get(ctx, client.ObjectKey{Name: name, Namespace: "default"}, existing); getErr != nil {
			return fmt.Errorf("create webapp: %w", err)
		}
		existing.Spec = webapp.Spec
		if err := k8s.Update(ctx, existing); err != nil {
			return fmt.Errorf("update webapp: %w", err)
		}
		fmt.Printf("-> updated WebApp %s\n", name)
		return nil
	}

	fmt.Printf("-> created WebApp %s\n", name)
	return nil
}

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

func List(ctx context.Context) ([]platformv1.WebApp, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, nil,
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := platformv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("add scheme: %w", err)
	}

	k8s, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("k8s client: %w", err)
	}

	list := &platformv1.WebAppList{}
	if err := k8s.List(ctx, list, client.InNamespace("default")); err != nil {
		return nil, err
	}

	return list.Items, nil
}
