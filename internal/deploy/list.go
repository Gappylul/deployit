package deploy

import (
	"context"
	"fmt"

	platformv1 "github.com/gappylul/webapp-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func GetDeploymentStats(ctx context.Context, name string) (int32, int32, string) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, nil,
	).ClientConfig()

	k8s, err := client.New(config, client.Options{})
	if err != nil {
		return 0, 0, "Unknown"
	}

	deploy := &appsv1.Deployment{}
	err = k8s.Get(ctx, types.NamespacedName{Name: name, Namespace: "default"}, deploy)
	if err != nil {
		return 0, 0, "Missing"
	}

	ready := deploy.Status.ReadyReplicas
	desired := *deploy.Spec.Replicas

	status := "Running"
	if ready < desired {
		status = "Progressing"
	}
	if ready == 0 && desired > 0 {
		status = "Pending"
	}

	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentReplicaFailure && cond.Status == "True" {
			status = "Error"
		}
	}

	return ready, desired, status
}
